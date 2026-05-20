//go:build windows

package windows

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/sys/windows"
)

const (
	dbPageSize    = 4096
	dbKeySize     = 32
	dbSaltSize    = 16
	dbIVSize      = 16
	dbHMACSize    = 64
	dbReserveSize = dbIVSize + dbHMACSize
	maxRegionSize = 500 * 1024 * 1024
)

var cachedHexKeyPattern = regexp.MustCompile(`x'([0-9a-fA-F]{64,192})'`)

type dbKeyCandidate struct {
	relPath string
	path    string
	saltHex string
	page1   []byte
	size    int64
}

type dbKeyJSONEntry struct {
	EncKey string `json:"enc_key"`
	Salt   string `json:"salt"`
}

type GetDbKeyResult struct {
	Success bool
	Key     string
	Error   string
	Message string
}

func GetDbKeyFromMemory(ctx context.Context, dataDir string, preferredPID uint32, onProgress func(string)) GetDbKeyResult {
	progressMessages := make([]string, 0, 16)
	reportProgress := func(message string) {
		message = strings.TrimSpace(message)
		if message == "" {
			return
		}
		progressMessages = append(progressMessages, message)
		if onProgress != nil {
			onProgress(message)
		}
	}
	fail := func(format string, args ...any) GetDbKeyResult {
		return GetDbKeyResult{
			Success: false,
			Error:   fmt.Sprintf(format, args...),
			Message: strings.Join(progressMessages, "\n"),
		}
	}

	if strings.TrimSpace(dataDir) == "" {
		return fail("微信数据目录为空，无法定位数据库文件")
	}

	dbDir := filepath.Join(dataDir, "db_storage")
	reportProgress("正在收集数据库 salt...")
	candidates, err := collectDbKeyCandidates(dbDir)
	if err != nil {
		return fail("收集数据库文件失败: %v", err)
	}
	if len(candidates) == 0 {
		return fail("未找到可用于校验的微信数据库文件: %s", dbDir)
	}
	sortDbKeyCandidates(candidates)
	saltCount := countDbKeySalts(candidates)
	reportProgress(fmt.Sprintf("找到 %d 个可校验数据库、%d 个 salt，正在扫描微信进程内存...", len(candidates), saltCount))

	pids := []uint32{}
	if preferredPID != 0 {
		pids = append(pids, preferredPID)
	}
	if len(pids) == 0 {
		for _, name := range []string{"Weixin.exe", "WeChat.exe"} {
			found, findErr := FindProcessIdsByName(name)
			if findErr == nil {
				pids = append(pids, found...)
			}
		}
	}
	if len(pids) == 0 {
		return fail("微信进程未运行，无法扫描内存")
	}

	keyBySalt := make(map[string]string)
	for _, pid := range pids {
		select {
		case <-ctx.Done():
			return fail("数据库密钥扫描已取消: %v", ctx.Err())
		default:
		}

		reportProgress(fmt.Sprintf("正在扫描微信进程 PID=%d...", pid))
		matched, err := scanProcessForDbKeys(ctx, pid, candidates, keyBySalt, reportProgress)
		crossVerifyDbKeys(candidates, keyBySalt)
		if matched > 0 {
			reportProgress(fmt.Sprintf("PID=%d 命中 %d 个数据库 salt，累计 %d/%d", pid, matched, len(keyBySalt), saltCount))
		}
		if err != nil {
			reportProgress(fmt.Sprintf("PID=%d 扫描结束: %v", pid, err))
		}
		if len(keyBySalt) >= saltCount {
			break
		}
	}

	if len(keyBySalt) == 0 {
		return fail("未能从微信进程内存中提取到数据库密钥")
	}

	keyJSON, err := buildDbKeyJSON(candidates, keyBySalt)
	if err != nil {
		return fail("生成数据库密钥映射失败: %v", err)
	}
	reportProgress(fmt.Sprintf("数据库密钥获取成功，已匹配 %d/%d 个 salt", len(keyBySalt), saltCount))
	return GetDbKeyResult{Success: true, Key: keyJSON, Message: strings.Join(progressMessages, "\n")}
}

func collectDbKeyCandidates(dbDir string) ([]dbKeyCandidate, error) {
	candidates := make([]dbKeyCandidate, 0, 16)
	err := filepath.WalkDir(dbDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".db") || strings.HasSuffix(name, "-wal") || strings.HasSuffix(name, "-shm") {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil || info.Size() < dbPageSize {
			return nil
		}
		page1 := make([]byte, dbPageSize)
		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		_, readErr := f.Read(page1)
		_ = f.Close()
		if readErr != nil {
			return nil
		}
		if bytes.Equal(page1[:len("SQLite format 3")], []byte("SQLite format 3")) {
			return nil
		}
		rel, relErr := filepath.Rel(dbDir, path)
		if relErr != nil {
			rel = path
		}
		candidates = append(candidates, dbKeyCandidate{
			relPath: rel,
			path:    path,
			saltHex: hex.EncodeToString(page1[:dbSaltSize]),
			page1:   page1,
			size:    info.Size(),
		})
		return nil
	})
	return candidates, err
}

func sortDbKeyCandidates(candidates []dbKeyCandidate) {
	preferred := []string{
		filepath.Join("message", "message_0.db"),
		filepath.Join("session", "session.db"),
	}
	for i := len(preferred) - 1; i >= 0; i-- {
		needle := strings.ToLower(preferred[i])
		for idx, candidate := range candidates {
			if strings.ToLower(candidate.relPath) == needle {
				picked := candidate
				copy(candidates[1:idx+1], candidates[0:idx])
				candidates[0] = picked
				break
			}
		}
	}
}

func countDbKeySalts(candidates []dbKeyCandidate) int {
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		seen[candidate.saltHex] = struct{}{}
	}
	return len(seen)
}

func scanProcessForDbKeys(ctx context.Context, pid uint32, candidates []dbKeyCandidate, keyBySalt map[string]string, onProgress func(string)) (int, error) {
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(hProcess)

	regions := GetMemoryRegions(hProcess)
	if len(regions) == 0 {
		return 0, fmt.Errorf("未找到可读内存区域")
	}

	startedAt := time.Now()
	scannedRegions := 0
	matched := 0
	for _, region := range regions {
		select {
		case <-ctx.Done():
			return matched, ctx.Err()
		default:
		}
		if !isDbKeyReadableRegion(region) {
			continue
		}
		data, err := ReadMemoryChunk(hProcess, region.BaseAddress, int(region.RegionSize))
		if err != nil || len(data) == 0 {
			continue
		}
		scannedRegions++
		matched += scanMemoryChunkForDbKeys(data, candidates, keyBySalt)
		if onProgress != nil && scannedRegions%200 == 0 {
			onProgress(fmt.Sprintf("已扫描 %d 个内存区域，耗时 %.1fs", scannedRegions, time.Since(startedAt).Seconds()))
		}
		if len(keyBySalt) >= countDbKeySalts(candidates) {
			return matched, nil
		}
	}

	if matched == 0 {
		return 0, fmt.Errorf("未匹配到可验证的 key")
	}
	return matched, nil
}

func isDbKeyReadableRegion(region MemoryRegion) bool {
	if region.RegionSize == 0 || region.RegionSize > maxRegionSize {
		return false
	}
	if region.State != windows.MEM_COMMIT {
		return false
	}
	if region.Protect == windows.PAGE_NOACCESS || (region.Protect&windows.PAGE_GUARD) != 0 {
		return false
	}
	return true
}

func scanMemoryChunkForDbKeys(data []byte, candidates []dbKeyCandidate, keyBySalt map[string]string) int {
	matched := 0
	matches := cachedHexKeyPattern.FindAllSubmatch(data, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		hexPayload := string(match[1])
		hexLen := len(hexPayload)

		if hexLen == 96 {
			if setDbKeyForSalt(hexPayload[:64], hexPayload[64:], candidates, keyBySalt) {
				matched++
			}
			continue
		}

		if hexLen == 64 {
			if setDbKeyByValidation(hexPayload, candidates, keyBySalt) {
				matched++
			}
			continue
		}

		if hexLen > 96 && hexLen%2 == 0 {
			if setDbKeyForSalt(hexPayload[:64], hexPayload[hexLen-32:], candidates, keyBySalt) {
				matched++
			}
		}
	}
	return matched
}

func setDbKeyForSalt(keyHex string, saltHex string, candidates []dbKeyCandidate, keyBySalt map[string]string) bool {
	if _, ok := keyBySalt[saltHex]; ok {
		return false
	}
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil || len(keyBytes) != dbKeySize {
		return false
	}
	for _, candidate := range candidates {
		if candidate.saltHex == saltHex && verifyDbEncKey(keyBytes, candidate.page1) {
			keyBySalt[saltHex] = keyHex
			return true
		}
	}
	return false
}

func setDbKeyByValidation(keyHex string, candidates []dbKeyCandidate, keyBySalt map[string]string) bool {
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil || len(keyBytes) != dbKeySize {
		return false
	}
	for _, candidate := range candidates {
		if _, ok := keyBySalt[candidate.saltHex]; ok {
			continue
		}
		if verifyDbEncKey(keyBytes, candidate.page1) {
			keyBySalt[candidate.saltHex] = keyHex
			return true
		}
	}
	return false
}

func crossVerifyDbKeys(candidates []dbKeyCandidate, keyBySalt map[string]string) {
	if len(keyBySalt) == 0 {
		return
	}
	knownKeys := make([]string, 0, len(keyBySalt))
	for _, keyHex := range keyBySalt {
		knownKeys = append(knownKeys, keyHex)
	}
	for _, candidate := range candidates {
		if _, ok := keyBySalt[candidate.saltHex]; ok {
			continue
		}
		for _, keyHex := range knownKeys {
			keyBytes, err := hex.DecodeString(keyHex)
			if err != nil || len(keyBytes) != dbKeySize {
				continue
			}
			if verifyDbEncKey(keyBytes, candidate.page1) {
				keyBySalt[candidate.saltHex] = keyHex
				break
			}
		}
	}
}

func buildDbKeyJSON(candidates []dbKeyCandidate, keyBySalt map[string]string) (string, error) {
	result := make(map[string]any, len(candidates)+1)
	for _, candidate := range candidates {
		keyHex, ok := keyBySalt[candidate.saltHex]
		if !ok {
			continue
		}
		result[candidate.relPath] = dbKeyJSONEntry{
			EncKey: keyHex,
			Salt:   candidate.saltHex,
		}
	}
	if len(result) == 0 {
		return "", fmt.Errorf("没有可保存的数据库密钥")
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// ScanDataKey extracts the database encryption key from a running WeChat process memory.
// pid must be non-zero. Returns the key JSON string or an error.
func ScanDataKey(ctx context.Context, dataDir string, pid uint32, onProgress func(string)) (string, error) {
	result := GetDbKeyFromMemory(ctx, dataDir, pid, onProgress)
	if result.Success {
		return result.Key, nil
	}
	msg := result.Error
	if msg == "" {
		msg = result.Message
	}
	if msg == "" {
		msg = "未能从微信进程内存中提取到数据库密钥"
	}
	return "", fmt.Errorf("%s", msg)
}

func verifyDbEncKey(encKey []byte, page1 []byte) bool {
	if len(encKey) != dbKeySize || len(page1) < dbPageSize {
		return false
	}
	salt := page1[:dbSaltSize]
	macSalt := make([]byte, len(salt))
	for i, b := range salt {
		macSalt[i] = b ^ 0x3a
	}
	macKey := pbkdf2.Key(encKey, macSalt, 2, dbKeySize, sha512.New)
	hmacData := page1[dbSaltSize : dbPageSize-dbReserveSize+dbIVSize]
	storedHMAC := page1[dbPageSize-dbHMACSize : dbPageSize]
	mac := hmac.New(sha512.New, macKey)
	_, _ = mac.Write(hmacData)
	pageNoBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(pageNoBytes, 1)
	_, _ = mac.Write(pageNoBytes)
	return hmac.Equal(mac.Sum(nil), storedHMAC)
}

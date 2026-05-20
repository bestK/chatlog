package darwin

import (
	"crypto/aes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// V2 .dat file magic bytes
var v2Magic = []byte{0x07, 0x08, 0x56, 0x32, 0x08, 0x07}

// Image magic bytes for AES decryption verification
var imageMagics = [][]byte{
	{0xFF, 0xD8, 0xFF},             // JPEG
	{0x89, 0x50, 0x4E, 0x47},       // PNG
	{'G', 'I', 'F'},                // GIF
	{'R', 'I', 'F', 'F'},           // WebP container
	{'w', 'x', 'g', 'f'},           // WeChat HEVC GIF
}

var kvcommFileRe = regexp.MustCompile(`^key_(\d+)_.+\.statistic$`)

// DeriveImgKey attempts to derive the image AES key from kvcomm cache on macOS.
// Returns (aesKey, error). aesKey is a 16-char ASCII hex string.
func DeriveImgKey(dataDir string, onProgress func(string)) (string, error) {
	if dataDir == "" {
		return "", fmt.Errorf("dataDir is empty")
	}

	// dataDir is typically .../xwechat_files/<wxid>/db_storage
	// baseDir = .../xwechat_files/<wxid>
	baseDir := filepath.Dir(dataDir)
	if filepath.Base(dataDir) != "db_storage" {
		baseDir = dataDir
	}
	attachDir := filepath.Join(baseDir, "msg", "attach")

	// Collect V2 template ciphertexts for verification
	templates := collectV2Templates(attachDir, 3)
	if len(templates) == 0 {
		return "", fmt.Errorf("未找到 V2 模板文件，请先在微信中查看几张图片")
	}
	progress(onProgress, fmt.Sprintf("找到 %d 个 V2 模板用于交叉验证", len(templates)))

	// Try kvcomm approach first
	wxidCandidates := collectWxidCandidates(dataDir)
	if len(wxidCandidates) == 0 {
		return "", fmt.Errorf("无法从数据目录提取 wxid")
	}

	kvcommDir := findKvcommDir(dataDir)
	if kvcommDir != "" {
		progress(onProgress, fmt.Sprintf("使用 kvcomm 目录: %s", kvcommDir))
		codes := collectKvcommCodes(kvcommDir)
		if len(codes) > 0 {
			progress(onProgress, fmt.Sprintf("找到 %d 个 uin 候选", len(codes)))
			for _, wxid := range wxidCandidates {
				for _, code := range codes {
					aesKey := deriveAesKey(code, wxid)
					if verifyAesKeyAll(aesKey, templates) {
						progress(onProgress, fmt.Sprintf("kvcomm 验证成功: uin=%d, wxid=%s", code, wxid))
						return aesKey, nil
					}
				}
			}
			progress(onProgress, "kvcomm 所有组合未通过验证")
		} else {
			progress(onProgress, "kvcomm 目录无 statistic 文件")
		}
	} else {
		progress(onProgress, "未找到 kvcomm 缓存目录")
	}

	// Fallback: brute-force via wxid suffix
	progress(onProgress, "尝试 wxid 后缀候选搜索...")
	result, err := bruteforceViaWxidSuffix(dataDir, attachDir, templates, onProgress)
	if err != nil {
		return "", err
	}
	if result != "" {
		return result, nil
	}

	return "", fmt.Errorf("无法派生图片密钥，请确保微信中有 V2 格式的图片缓存")
}

// deriveAesKey computes aes_key = MD5(str(uin) + wxid)[:16]
func deriveAesKey(uin uint32, wxid string) string {
	data := fmt.Sprintf("%d%s", uin, wxid)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

// verifyAesKey decrypts a 16-byte ciphertext with AES-128-ECB and checks image magic.
func verifyAesKey(aesKeyASCII string, ciphertext []byte) bool {
	if len(aesKeyASCII) < 16 || len(ciphertext) != 16 {
		return false
	}
	keyBytes := []byte(aesKeyASCII[:16])
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return false
	}
	decrypted := make([]byte, 16)
	block.Decrypt(decrypted, ciphertext)
	for _, magic := range imageMagics {
		if len(decrypted) >= len(magic) && string(decrypted[:len(magic)]) == string(magic) {
			return true
		}
	}
	return false
}

// verifyAesKeyAll checks that aesKey passes verification against ALL templates.
func verifyAesKeyAll(aesKey string, templates [][]byte) bool {
	if len(templates) == 0 {
		return false
	}
	for _, ct := range templates {
		if !verifyAesKey(aesKey, ct) {
			return false
		}
	}
	return true
}

// collectV2Templates finds V2 .dat template ciphertexts ([0xF:0x1F]) for verification.
func collectV2Templates(attachDir string, maxTemplates int) [][]byte {
	if attachDir == "" {
		return nil
	}
	var templates [][]byte
	seen := make(map[string]struct{})

	scan := func(suffix string) [][]byte {
		_ = filepath.WalkDir(attachDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(d.Name(), suffix) {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			buf := make([]byte, 0x20)
			n, _ := f.Read(buf)
			f.Close()
			if n < 0x1F {
				return nil
			}
			if string(buf[:6]) != string(v2Magic) {
				return nil
			}
			ct := buf[0x0F:0x1F]
			key := string(ct)
			if _, ok := seen[key]; ok {
				return nil
			}
			seen[key] = struct{}{}
			templates = append(templates, append([]byte(nil), ct...))
			if len(templates) >= maxTemplates {
				return filepath.SkipAll
			}
			return nil
		})
		return templates
	}

	// Prefer _t.dat (thumbnails, small and fast)
	if result := scan("_t.dat"); len(result) >= maxTemplates {
		return result
	}
	// Fallback to any .dat
	return scan(".dat")
}

// collectWxidCandidates extracts wxid candidates from the data directory path.
func collectWxidCandidates(dataDir string) []string {
	parts := strings.Split(filepath.ToSlash(dataDir), "/")
	idx := -1
	for i, p := range parts {
		if p == "xwechat_files" {
			idx = i
			break
		}
	}
	if idx < 0 || idx+1 >= len(parts) {
		return nil
	}
	raw := parts[idx+1]
	candidates := []string{raw}
	normalized := normalizeWxid(raw)
	if normalized != "" && normalized != raw {
		candidates = append(candidates, normalized)
	}
	return candidates
}

// normalizeWxid strips the macOS path suffix (_<4 alnum>) from wxid.
func normalizeWxid(aid string) string {
	if strings.HasPrefix(strings.ToLower(aid), "wxid_") {
		re := regexp.MustCompile(`^(wxid_[^_]+)`)
		m := re.FindString(aid)
		if m != "" {
			return m
		}
		return aid
	}
	re := regexp.MustCompile(`^(.+)_([a-zA-Z0-9]{4})$`)
	m := re.FindStringSubmatch(aid)
	if m != nil {
		return m[1]
	}
	return aid
}

// findKvcommDir locates the kvcomm cache directory from dataDir.
func findKvcommDir(dataDir string) string {
	for _, candidate := range deriveKvcommCandidates(dataDir) {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

// deriveKvcommCandidates returns possible kvcomm directory paths ordered by priority.
func deriveKvcommCandidates(dataDir string) []string {
	parts := strings.Split(filepath.ToSlash(dataDir), "/")
	var candidates []string

	idx := -1
	for i, p := range parts {
		if p == "xwechat_files" {
			idx = i
			break
		}
	}

	if idx >= 0 {
		documentsRoot := strings.Join(parts[:idx], "/")
		candidates = append(candidates,
			filepath.FromSlash(documentsRoot+"/app_data/net/kvcomm"),
			filepath.FromSlash(documentsRoot+"/xwechat/net/kvcomm"),
		)
		if idx >= 1 {
			containerRoot := strings.Join(parts[:idx-1], "/")
			candidates = append(candidates,
				filepath.FromSlash(containerRoot+"/Library/Application Support/com.tencent.xinWeChat/xwechat/net/kvcomm"),
				filepath.FromSlash(containerRoot+"/Library/Application Support/com.tencent.xinWeChat/net/kvcomm"),
			)
		}
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, "Library", "Containers", "com.tencent.xinWeChat", "Data", "Documents", "app_data", "net", "kvcomm"),
		)
	}

	// Deduplicate
	seen := make(map[string]struct{})
	deduped := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if _, ok := seen[c]; !ok {
			seen[c] = struct{}{}
			deduped = append(deduped, c)
		}
	}
	return deduped
}

// collectKvcommCodes scans the kvcomm directory for uin codes from statistic filenames.
func collectKvcommCodes(kvcommDir string) []uint32 {
	entries, err := os.ReadDir(kvcommDir)
	if err != nil {
		return nil
	}
	seen := make(map[uint32]struct{})
	for _, entry := range entries {
		m := kvcommFileRe.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		code, err := strconv.ParseUint(m[1], 10, 32)
		if err != nil || code == 0 {
			continue
		}
		seen[uint32(code)] = struct{}{}
	}
	codes := make([]uint32, 0, len(seen))
	for c := range seen {
		codes = append(codes, c)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}

// bruteforceViaWxidSuffix uses the wxid directory's 4-hex suffix to narrow uin candidates.
// The suffix equals md5(str(uin))[:4]. We enumerate all uin where (uin & 0xFF == xorKey)
// and md5 prefix matches, then verify with AES templates.
func bruteforceViaWxidSuffix(dataDir, attachDir string, templates [][]byte, onProgress func(string)) (string, error) {
	wxidCandidates := collectWxidCandidates(dataDir)
	if len(wxidCandidates) == 0 {
		return "", nil
	}

	// Extract 4-hex suffix from wxid directory name
	raw := wxidCandidates[0]
	hexSuffixRe := regexp.MustCompile(`^(.+)_([0-9a-fA-F]{4})$`)
	m := hexSuffixRe.FindStringSubmatch(raw)
	if m == nil {
		progress(onProgress, "wxid 路径不含 _<4 hex> 后缀，无法应用候选搜索")
		return "", nil
	}
	wxidNorm := m[1]
	suffix := strings.ToLower(m[2])

	// Derive xor_key from V2 .dat tail bytes (assuming JPEG EOI = 0xD9)
	xorKey, ok := deriveXorKeyFromV2Dat(attachDir)
	if !ok {
		progress(onProgress, "V2 .dat 样本不足，无法推导 xor_key")
		return "", nil
	}
	progress(onProgress, fmt.Sprintf("xor_key=0x%02x, suffix=%s, 开始枚举...", xorKey, suffix))

	// Enumerate: uin & 0xFF == xorKey, md5(str(uin))[:4] == suffix
	suffixBytes, _ := hex.DecodeString(suffix)
	wxidBytes := []byte(wxidNorm)

	for i := 0; i < (1 << 24); i++ {
		uin := uint32(i<<8) | uint32(xorKey)
		uinStr := strconv.FormatUint(uint64(uin), 10)
		hash := md5.Sum([]byte(uinStr))
		if hash[0] != suffixBytes[0] || hash[1] != suffixBytes[1] {
			continue
		}
		// md5 prefix matches, try AES verification
		fullHash := md5.Sum(append([]byte(uinStr), wxidBytes...))
		aesKey := hex.EncodeToString(fullHash[:])[:16]
		if verifyAesKeyAll(aesKey, templates) {
			progress(onProgress, fmt.Sprintf("候选搜索成功: uin=%d, wxid=%s", uin, wxidNorm))
			return aesKey, nil
		}
	}

	return "", nil
}

// deriveXorKeyFromV2Dat votes on xor_key from V2 .dat tail bytes (JPEG EOI = 0xD9).
func deriveXorKeyFromV2Dat(attachDir string) (byte, bool) {
	if attachDir == "" {
		return 0, false
	}
	type vote struct {
		key   byte
		count int
	}
	counts := make(map[byte]int)
	total := 0
	const maxSamples = 10

	_ = filepath.WalkDir(attachDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || total >= maxSamples {
			if total >= maxSamples {
				return filepath.SkipAll
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".dat") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		head := make([]byte, 6)
		if n, _ := f.Read(head); n < 6 || string(head) != string(v2Magic) {
			return nil
		}
		info, _ := f.Stat()
		if info == nil || info.Size() < 0x20 {
			return nil
		}
		f.Seek(-1, 2)
		lastByte := make([]byte, 1)
		if _, err := f.Read(lastByte); err != nil {
			return nil
		}
		counts[lastByte[0]^0xD9]++
		total++
		return nil
	})

	if total < 3 {
		return 0, false
	}
	var best byte
	var bestCount int
	for k, c := range counts {
		if c > bestCount {
			best = k
			bestCount = c
		}
	}
	return best, true
}

func progress(fn func(string), msg string) {
	if fn != nil {
		fn(msg)
	}
	log.Debug().Msg(msg)
}
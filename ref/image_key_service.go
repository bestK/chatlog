package main

import (
	"bytes"
	"crypto/aes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/sys/windows"
)

// ImageKeyResult 密钥获取结果
type ImageKeyResult struct {
	Success             bool
	XorKey              int
	AesKey              string
	Error               string
	NeedManualSelection bool
}

// SuccessResult 创建成功结果
func SuccessResult(xorKey int, aesKey string) ImageKeyResult {
	return ImageKeyResult{
		Success: true,
		XorKey:  xorKey,
		AesKey:  aesKey,
	}
}

// FailureResult 创建失败结果
func FailureResult(err string, needManual bool) ImageKeyResult {
	return ImageKeyResult{
		Success:             false,
		Error:               err,
		NeedManualSelection: needManual,
	}
}

// GetWeChatCacheDirectory 获取微信缓存目录
func GetWeChatCacheDirectory() (string, error) {
	dirs, err := FindWeChatCacheDirectories()
	if err != nil {
		return "", err
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("未找到微信缓存目录")
	}
	return dirs[0], nil
}

// FindWeChatCacheDirectories 查找所有微信缓存目录
func FindWeChatCacheDirectories() ([]string, error) {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return nil, fmt.Errorf("无法获取 USERPROFILE 环境变量")
	}

	wechatFilesPath := filepath.Join(userProfile, "Documents", "xwechat_files")
	if _, err := os.Stat(wechatFilesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("微信缓存目录不存在: %s", wechatFilesPath)
	}

	var highConfidence, lowConfidence []string

	entries, err := os.ReadDir(wechatFilesPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		if !isPotentialAccountDirectory(dirName) {
			continue
		}

		fullPath := filepath.Join(wechatFilesPath, dirName)

		hasDbStorage := directoryHasDbStorage(fullPath)
		hasImageCache := directoryHasImageCache(fullPath)

		if hasDbStorage || hasImageCache {
			highConfidence = append(highConfidence, fullPath)
		} else {
			lowConfidence = append(lowConfidence, fullPath)
		}
	}

	if len(highConfidence) > 0 {
		sort.Strings(highConfidence)
		return highConfidence, nil
	}

	sort.Strings(lowConfidence)
	return lowConfidence, nil
}

func isPotentialAccountDirectory(dirName string) bool {
	lower := strings.ToLower(dirName)
	if strings.HasPrefix(lower, "all") ||
		strings.HasPrefix(lower, "applet") ||
		strings.HasPrefix(lower, "backup") ||
		strings.HasPrefix(lower, "wmpf") {
		return false
	}
	return strings.HasPrefix(dirName, "wxid_") || len(dirName) > 5
}

func directoryHasDbStorage(dir string) bool {
	dbStoragePath := filepath.Join(dir, "db_storage")
	_, err := os.Stat(dbStoragePath)
	return err == nil
}

func directoryHasImageCache(dir string) bool {
	imagePath := filepath.Join(dir, "FileStorage", "Image")
	_, err := os.Stat(imagePath)
	return err == nil
}

// FindTemplateDatFiles 查找所有 *_t.dat 模板文件
func FindTemplateDatFiles(userDir string) ([]string, error) {
	var files []string
	const maxFiles = 32

	err := filepath.WalkDir(userDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 忽略错误，继续遍历
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_t.dat") {
			files = append(files, path)
			if len(files) >= maxFiles {
				return filepath.SkipAll
			}
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}

	if len(files) == 0 {
		return nil, nil
	}

	// 按日期排序（降序）
	dateRegex := regexp.MustCompile(`(\d{4}-\d{2})`)
	sort.Slice(files, func(i, j int) bool {
		matchI := dateRegex.FindString(files[i])
		matchJ := dateRegex.FindString(files[j])
		return matchI > matchJ
	})

	if len(files) > 16 {
		files = files[:16]
	}

	return files, nil
}

// GetXorKey 从模板文件获取 XOR 密钥
func GetXorKey(templateFiles []string) (int, error) {
	lastBytesMap := make(map[string]int)

	for _, filePath := range templateFiles {
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		if len(data) >= 2 {
			lastTwo := data[len(data)-2:]
			key := fmt.Sprintf("%d_%d", lastTwo[0], lastTwo[1])
			lastBytesMap[key]++
		}
	}

	if len(lastBytesMap) == 0 {
		return 0, fmt.Errorf("无法从模板文件提取尾字节")
	}

	// 找出出现次数最多的字节对
	var mostCommon string
	maxCount := 0
	for key, count := range lastBytesMap {
		if count > maxCount {
			maxCount = count
			mostCommon = key
		}
	}

	if mostCommon == "" {
		return 0, fmt.Errorf("未找到有效的字节对")
	}

	var x, y int
	fmt.Sscanf(mostCommon, "%d_%d", &x, &y)

	// XOR 密钥计算：利用 JPEG 尾部固定为 0xFF 0xD9
	xorKey := x ^ 0xFF
	check := y ^ 0xD9

	if xorKey != check {
		return 0, fmt.Errorf("XOR 密钥验证失败")
	}

	return xorKey, nil
}

// GetCiphertextFromTemplate 从模板文件读取加密的 AES 数据
func GetCiphertextFromTemplate(templateFiles []string) ([]byte, error) {
	// 特殊格式文件头部
	header := []byte{0x07, 0x08, 0x56, 0x32, 0x08, 0x07}

	for _, filePath := range templateFiles {
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		if len(data) < 8 {
			continue
		}

		// 检查头部是否匹配
		if bytes.Equal(data[:6], header) {
			if len(data) >= 0x1F {
				ciphertext := data[0x0F:0x1F]
				return ciphertext, nil
			}
		}
	}

	return nil, fmt.Errorf("未找到包含加密数据的模板文件")
}

// VerifyKey 验证 AES 密钥是否正确
func VerifyKey(encrypted, aesKey []byte) bool {
	if len(aesKey) < 16 {
		return false
	}

	key := aesKey[:16]

	block, err := aes.NewCipher(key)
	if err != nil {
		return false
	}

	// AES-ECB 解密
	decrypted := make([]byte, len(encrypted))
	for i := 0; i < len(encrypted); i += 16 {
		block.Decrypt(decrypted[i:i+16], encrypted[i:i+16])
	}

	// 检查 JPEG 文件头 0xFF 0xD8 0xFF
	if len(decrypted) >= 3 &&
		decrypted[0] == 0xFF &&
		decrypted[1] == 0xD8 &&
		decrypted[2] == 0xFF {
		return true
	}

	return false
}

// isAlphaNumAscii 检查字节是否是字母或数字
func isAlphaNumAscii(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

// isUtf16AsciiKey 检查是否是 UTF-16 编码的 ASCII 密钥
func isUtf16AsciiKey(data []byte, start int) bool {
	if start+64 > len(data) {
		return false
	}

	for j := 0; j < 32; j++ {
		charByte := data[start+(j*2)]
		nullByte := data[start+(j*2)+1]
		if nullByte != 0x00 || !isAlphaNumAscii(charByte) {
			return false
		}
	}

	return true
}

// GetAesKeyFromMemory 从微信进程内存中搜索 AES 密钥
func GetAesKeyFromMemory(pid uint32, ciphertext []byte, onProgress func(string)) (string, error) {
	if onProgress != nil {
		onProgress(fmt.Sprintf("开始内存搜索，目标进程: %d", pid))
	}

	hProcess, err := OpenProcessByPID(pid)
	if err != nil {
		return "", fmt.Errorf("无法打开进程: %v", err)
	}
	defer CloseHandle(hProcess)

	regions := GetMemoryRegions(hProcess)
	if onProgress != nil {
		onProgress(fmt.Sprintf("找到 %d 个内存区域", len(regions)))
	}

	const chunkSize = 4 * 1024 * 1024 // 4MB
	const overlap = 65

	scannedCount := 0

	for _, region := range regions {
		// 跳过过大的内存区域
		if region.RegionSize > 100*1024*1024 {
			continue
		}

		scannedCount++
		if scannedCount%10 == 0 && onProgress != nil {
			onProgress(fmt.Sprintf("正在扫描微信内存... (%d/%d)", scannedCount, len(regions)))
		}

		var offset uintptr = 0
		var trailing []byte

		for offset < region.RegionSize {
			remaining := region.RegionSize - offset
			currentChunkSize := int(remaining)
			if currentChunkSize > chunkSize {
				currentChunkSize = chunkSize
			}

			chunk, err := ReadMemoryChunk(hProcess, region.BaseAddress+offset, currentChunkSize)
			if err != nil || len(chunk) == 0 {
				offset += uintptr(currentChunkSize)
				trailing = nil
				continue
			}

			// 与上一块尾部拼接
			var dataToScan []byte
			if len(trailing) > 0 {
				dataToScan = append(trailing, chunk...)
			} else {
				dataToScan = chunk
			}

			// 搜索 32 字节 ASCII 密钥
			key := searchAsciiKey(dataToScan, ciphertext)
			if key != "" {
				if onProgress != nil {
					onProgress("已找到 AES 密钥！")
				}
				return key, nil
			}

			// 搜索 UTF-16 编码的密钥
			key = searchUtf16Key(dataToScan, ciphertext)
			if key != "" {
				if onProgress != nil {
					onProgress("已找到 AES 密钥 (UTF-16)！")
				}
				return key, nil
			}

			// 保存末尾用于跨块检测
			if len(dataToScan) > overlap {
				trailing = dataToScan[len(dataToScan)-overlap:]
			} else {
				trailing = nil
			}

			offset += uintptr(currentChunkSize)
		}
	}

	return "", fmt.Errorf("未在内存中找到 AES 密钥")
}

// searchAsciiKey 搜索 ASCII 编码的 32 字节密钥
func searchAsciiKey(data, ciphertext []byte) string {
	for i := 0; i < len(data)-34; i++ {
		// 前导字符不是字母或数字
		if isAlphaNumAscii(data[i]) {
			continue
		}

		// 检查接下来的 32 个字节
		valid := true
		for j := 1; j <= 32; j++ {
			if i+j >= len(data) || !isAlphaNumAscii(data[i+j]) {
				valid = false
				break
			}
		}

		if !valid {
			continue
		}

		// 尾部字符不是字母或数字
		if i+33 < len(data) && isAlphaNumAscii(data[i+33]) {
			continue
		}

		keyBytes := data[i+1 : i+33]
		if VerifyKey(ciphertext, keyBytes) {
			return string(keyBytes)
		}
	}

	return ""
}

// searchUtf16Key 搜索 UTF-16 编码的 32 字节密钥
func searchUtf16Key(data, ciphertext []byte) string {
	for i := 0; i < len(data)-65; i++ {
		if !isUtf16AsciiKey(data, i) {
			continue
		}

		keyBytes := make([]byte, 32)
		for j := 0; j < 32; j++ {
			keyBytes[j] = data[i+(j*2)]
		}

		if VerifyKey(ciphertext, keyBytes) {
			return string(keyBytes)
		}
	}

	return ""
}

// GetImageKeys 获取图片密钥（主入口）
func GetImageKeys(manualDirectory string, onProgress func(string)) ImageKeyResult {
	if onProgress != nil {
		onProgress("正在定位微信缓存目录...")
	}

	var cacheDir string
	var err error

	if manualDirectory != "" {
		cacheDir = manualDirectory
	} else {
		cacheDir, err = GetWeChatCacheDirectory()
		if err != nil {
			return FailureResult("未找到微信缓存目录，请手动选择目录", true)
		}
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("找到缓存目录: %s", cacheDir))
	}

	// 查找模板文件
	if onProgress != nil {
		onProgress("正在收集模板文件...")
	}

	templateFiles, err := FindTemplateDatFiles(cacheDir)
	if err != nil || len(templateFiles) == 0 {
		return FailureResult("未找到模板文件，可能该微信账号没有图片缓存", false)
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("找到 %d 个模板文件，正在计算 XOR 密钥...", len(templateFiles)))
	}

	// 获取 XOR 密钥
	xorKey, err := GetXorKey(templateFiles)
	if err != nil {
		return FailureResult("无法获取 XOR 密钥", false)
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("XOR 密钥获取成功: 0x%02X，正在读取加密数据...", xorKey))
	}

	// 获取加密数据
	ciphertext, err := GetCiphertextFromTemplate(templateFiles)
	if err != nil {
		return FailureResult("无法读取加密数据", false)
	}

	if onProgress != nil {
		onProgress("成功读取加密数据，正在检查微信进程...")
	}

	// 查找微信进程（支持多种进程名）
	processNames := []string{"Weixin.exe", "WeChat.exe"}
	var pids []uint32
	var foundProcessName string
	
	for _, name := range processNames {
		pids, err = FindProcessIdsByName(name)
		if err == nil && len(pids) > 0 {
			foundProcessName = name
			break
		}
	}
	
	if len(pids) == 0 {
		// 尝试使用 Windows 命令查找进程
		return FailureResult("微信进程未运行（请确保以管理员权限运行）", false)
	}
	
	_ = foundProcessName // 用于日志

	if onProgress != nil {
		onProgress(fmt.Sprintf("已定位微信进程 (PID: %d)，正在扫描内存获取 AES 密钥...", pids[0]))
	}

	// 从内存获取 AES 密钥
	aesKey, err := GetAesKeyFromMemory(pids[0], ciphertext, onProgress)
	if err != nil {
		return FailureResult(
			"无法从内存中获取 AES 密钥。\n"+
				"建议操作步骤：\n"+
				"1. 彻底关闭当前登录的微信。\n"+
				"2. 重新启动微信并登录。\n"+
				"3. 打开朋友圈，寻找带图片的动态。\n"+
				"4. 点击图片，再点击右上角打开大图。\n"+
				"5. 重复步骤3和4，大概2-3次即可。\n"+
				"6. 迅速回到工具内获取图片密钥。",
			false)
	}

	// 截取前 16 字符
	if len(aesKey) > 16 {
		aesKey = aesKey[:16]
	}

	return SuccessResult(xorKey, aesKey)
}

// OpenProcessByPID 在此文件中定义以避免循环引用
func openProcessForKey(pid uint32) (windows.Handle, error) {
	return windows.OpenProcess(PROCESS_ALL_ACCESS, false, pid)
}

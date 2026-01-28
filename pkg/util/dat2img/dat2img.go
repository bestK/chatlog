package dat2img

// Implementation based on:
// - https://github.com/tujiaw/wechat_dat_to_image
// - https://github.com/LC044/WeChatMsg/blob/6535ed0/wxManager/decrypt/decrypackage dat2img

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// Format defines the header and extension for different image types
type Format struct {
	Header []byte
	AesKey []byte
	Ext    string
}

var (
	// Common image format definitions
	JPG     = Format{Header: []byte{0xFF, 0xD8, 0xFF}, Ext: "jpg"}
	PNG     = Format{Header: []byte{0x89, 0x50, 0x4E, 0x47}, Ext: "png"}
	GIF     = Format{Header: []byte{0x47, 0x49, 0x46, 0x38}, Ext: "gif"}
	TIFF    = Format{Header: []byte{0x49, 0x49, 0x2A, 0x00}, Ext: "tiff"}
	BMP     = Format{Header: []byte{0x42, 0x4D}, Ext: "bmp"}
	WXGF    = Format{Header: []byte{0x77, 0x78, 0x67, 0x66}, Ext: "wxgf"}
	Formats = []Format{JPG, PNG, GIF, TIFF, BMP, WXGF}

	// Updated V4 definitions to match Dart implementation (6 bytes signature)
	// V4 Type 1: 0x07 0x08 0x56 0x31 0x08 0x07
	V4Format1 = Format{Header: []byte{0x07, 0x08, 0x56, 0x31, 0x08, 0x07}, AesKey: []byte("cfcd208495d565ef")}
	// V4 Type 2: 0x07 0x08 0x56 0x32 0x08 0x07
	V4Format2 = Format{Header: []byte{0x07, 0x08, 0x56, 0x32, 0x08, 0x07}, AesKey: []byte("0000000000000000")} // User needs to provide key
	V4Formats = []*Format{&V4Format1, &V4Format2}

	// WeChat v4 related constants
	V4XorKey byte = 0x37               // Default XOR key for WeChat v4 dat files
	JpgTail       = []byte{0xFF, 0xD9} // JPG file tail marker
)

// Dat2Image converts WeChat dat file data to image data
func Dat2Image(data []byte) ([]byte, string, error) {
	if len(data) < 4 {
		return nil, "", fmt.Errorf("data length is too short: %d", len(data))
	}

	// Check if this is a WeChat v4 dat file (Check first 4 or 6 bytes)
	if len(data) >= 6 {
		for _, format := range V4Formats {
			// 优先尝试 6 字节精确匹配，失败则尝试 4 字节前缀匹配
			if bytes.Equal(data[:6], format.Header) || bytes.Equal(data[:4], format.Header[:4]) {
				return Dat2ImageV4(data, format.AesKey)
			}
		}
	} else if len(data) >= 4 {
		// 只有 4 字节数据的情况
		for _, format := range V4Formats {
			if bytes.Equal(data[:4], format.Header[:4]) {
				return Dat2ImageV4(data, format.AesKey)
			}
		}
	}

	// For older WeChat versions (V3), use XOR decryption
	findFormat := func(data []byte, header []byte) bool {
		xorBit := data[0] ^ header[0]
		for i := 0; i < len(header); i++ {
			if data[i]^header[i] != xorBit {
				return false
			}
		}
		return true
	}

	var xorBit byte
	var found bool
	var ext string
	for _, format := range Formats {
		if found = findFormat(data, format.Header); found {
			xorBit = data[0] ^ format.Header[0]
			ext = format.Ext
			break
		}
	}

	if !found {
		return nil, "", fmt.Errorf("unknown image type: %x %x", data[0], data[1])
	}

	// Apply XOR decryption (V3)
	out := make([]byte, len(data))
	for i := range data {
		out[i] = data[i] ^ xorBit
	}

	return out, ext, nil
}

// calculateXorKeyV4 calculates the XOR key for WeChat v4 dat files
func calculateXorKeyV4(data []byte) (byte, error) {
	if len(data) < 2 {
		return 0, fmt.Errorf("data too short to calculate XOR key")
	}
	fileTail := data[len(data)-2:]
	xorKeys := make([]byte, 2)
	for i := 0; i < 2; i++ {
		xorKeys[i] = fileTail[i] ^ JpgTail[i]
	}
	if xorKeys[0] == xorKeys[1] {
		return xorKeys[0], nil
	}
	return xorKeys[0], fmt.Errorf("inconsistent XOR key, using first byte: 0x%x", xorKeys[0])
}

// ScanAndSetXorKey scans a directory to calculate and set the global XOR key
func ScanAndSetXorKey(dirPath string) (byte, error) {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), "_t.dat") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Check header (looser check for scan is acceptable, or use exact)
		isV4 := false
		if len(data) >= 4 {
			if bytes.Equal(data[:4], V4Format1.Header[:4]) || bytes.Equal(data[:4], V4Format2.Header[:4]) {
				isV4 = true
			}
		}
		if !isV4 {
			return nil
		}

		if len(data) < 15 {
			return nil
		}

		xorEncryptLen := binary.LittleEndian.Uint32(data[10:14])
		fileData := data[15:]

		if xorEncryptLen == 0 || uint32(len(fileData)) <= uint32(len(fileData))-xorEncryptLen {
			return nil
		}

		xorData := fileData[uint32(len(fileData))-xorEncryptLen:]
		key, err := calculateXorKeyV4(xorData)
		if err != nil {
			return nil
		}

		V4XorKey = key
		return filepath.SkipAll
	})

	if err != nil && err != filepath.SkipAll {
		return V4XorKey, fmt.Errorf("error scanning directory: %v", err)
	}
	return V4XorKey, nil
}

func SetAesKey(key string) {
	if key == "" {
		return
	}

	var aesKey []byte
	// 优先判断：如果长度是 16，通常是原始 ASCII 密钥，直接使用
	if len(key) == 16 {
		aesKey = []byte(key)
	} else {
		// 否则尝试十六进制解码（适用于 32 字符的十六进制密钥）
		decoded, err := hex.DecodeString(key)
		if err != nil {
			log.Error().Err(err).Msg("invalid aes key")
			return
		}
		aesKey = decoded
	}

	// 统一更新所有加密格式的密钥。由于微信一个账号 session 通常只用一个密钥，
	// 且我们目前只提取一个，因此同时更新 Format1 和 Format2 是最稳妥的。
	V4Format1.AesKey = aesKey
	V4Format2.AesKey = aesKey
	log.Debug().Str("key", key).Int("len", len(aesKey)).Msg("AES key updated for V4")
}

// Dat2ImageV4 processes WeChat v4 dat image files
// Refactored to match Dart implementation logic
func Dat2ImageV4(data []byte, aesKey []byte) ([]byte, string, error) {
	if len(data) < 15 {
		return nil, "", fmt.Errorf("data length is too short for WeChat v4 format")
	}

	// 1. Parse Headers (Little Endian)
	// Offset 6-10: AES Encryption Length
	aesSize := binary.LittleEndian.Uint32(data[6:10])
	// Offset 10-14: XOR Encryption Length
	xorSize := binary.LittleEndian.Uint32(data[10:14])

	// Skip header (15 bytes)
	fileData := data[15:]

	// 2. AES Decryption Logic
	// Calculate aligned size: size + (16 - size % 16)
	// This ensures we read the full PKCS7 padded block
	alignedAesSize := aesSize + (16 - (aesSize % 16))

	if uint32(len(fileData)) < alignedAesSize {
		return nil, "", fmt.Errorf("file data too short for declared AES length")
	}

	// Split data: [AES Part] [Middle Raw Part] [XOR Part]
	aesPart := fileData[:alignedAesSize]
	remainingPart := fileData[alignedAesSize:]

	log.Debug().Uint32("aesSize", aesSize).Uint32("xorSize", xorSize).Int("alignedAesSize", int(alignedAesSize)).Msg("V4 decrypt starts")

	var unpaddedAesData []byte
	var err error

	// Decrypt AES part
	if len(aesPart) > 0 {
		unpaddedAesData, err = decryptAESECBStrict(aesPart, aesKey)
		if err != nil {
			log.Warn().Err(err).Hex("header", data[:15]).Msg("V4 AES decryption failed")
			return nil, "", fmt.Errorf("AES decryption failed: %v", err)
		}
	}

	// 3. Handle Middle and XOR Parts
	// XOR size validation
	if uint32(len(remainingPart)) < xorSize {
		return nil, "", fmt.Errorf("file data too short for declared XOR length")
	}

	rawLen := uint32(len(remainingPart)) - xorSize
	rawMiddleData := remainingPart[:rawLen]
	xorTailData := remainingPart[rawLen:]

	// Decrypt XOR part
	decryptedXorData := make([]byte, len(xorTailData))
	for i := range xorTailData {
		decryptedXorData[i] = xorTailData[i] ^ V4XorKey
	}

	// 4. Reassemble: [Unpadded AES] + [Raw Middle] + [Decrypted XOR]
	// Pre-allocate exact size
	totalLen := len(unpaddedAesData) + len(rawMiddleData) + len(decryptedXorData)
	result := make([]byte, 0, totalLen)

	result = append(result, unpaddedAesData...)
	result = append(result, rawMiddleData...)
	result = append(result, decryptedXorData...)

	// Identify image type
	imgType := ""
	for _, format := range Formats {
		// Only check headers for image types
		if format.Ext == "wxgf" || format.Ext == "jpg" || format.Ext == "png" || format.Ext == "gif" || format.Ext == "tiff" || format.Ext == "bmp" {
			if len(result) >= len(format.Header) && bytes.Equal(result[:len(format.Header)], format.Header) {
				imgType = format.Ext
				break
			}
		}
	}

	if imgType == "wxgf" {
		return Wxam2pic(result)
	}

	if imgType == "" {
		if len(result) > 2 {
			log.Warn().Hex("header", result[:16]).Msg("V4 decrypted failed to match image header")
			return nil, "", fmt.Errorf("unknown image type after decryption: %x %x", result[0], result[1])
		}
		return nil, "", errors.New("unknown image type")
	}

	return result, imgType, nil
}

// decryptAESECBStrict decrypts data using AES in ECB mode and strictly removes PKCS7 padding
func decryptAESECBStrict(data, key []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("data length %d is not a multiple of block size", len(data))
	}

	decrypted := make([]byte, len(data))
	for bs, be := 0, aes.BlockSize; bs < len(data); bs, be = bs+aes.BlockSize, be+aes.BlockSize {
		cipher.Decrypt(decrypted[bs:be], data[bs:be])
	}

	// Strict PKCS7 Unpadding
	length := len(decrypted)
	if length == 0 {
		return nil, errors.New("decrypted data is empty")
	}

	paddingLen := int(decrypted[length-1])
	if paddingLen == 0 || paddingLen > aes.BlockSize || paddingLen > length {
		return nil, fmt.Errorf("invalid PKCS7 padding length: %d", paddingLen)
	}

	// Verify all padding bytes
	for i := length - paddingLen; i < length; i++ {
		if decrypted[i] != byte(paddingLen) {
			return nil, errors.New("invalid PKCS7 padding content")
		}
	}

	return decrypted[:length-paddingLen], nil
}

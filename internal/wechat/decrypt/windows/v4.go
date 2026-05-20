package windows

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"io"
	"os"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/decrypt/common"

	"golang.org/x/crypto/pbkdf2"
)

// V4 版本特定常量
const (
	V4IterCount    = 256000
	HmacSHA512Size = 64
	PageSize       = 4096
)

// V4Decryptor 实现Windows V4版本的解密器
type V4Decryptor struct {
	// V4 特定参数
	iterCount int
	hmacSize  int
	hashFunc  func() hash.Hash
	reserve   int
	pageSize  int
	version   string
}

// NewV4Decryptor 创建Windows V4解密器
func NewV4Decryptor() *V4Decryptor {
	hashFunc := sha512.New
	hmacSize := HmacSHA512Size
	reserve := common.IVSize + hmacSize
	if reserve%common.AESBlockSize != 0 {
		reserve = ((reserve / common.AESBlockSize) + 1) * common.AESBlockSize
	}

	return &V4Decryptor{
		iterCount: V4IterCount,
		hmacSize:  hmacSize,
		hashFunc:  hashFunc,
		reserve:   reserve,
		pageSize:  PageSize,
		version:   "Windows v4",
	}
}

// deriveKeys 派生加密密钥和MAC密钥
func (d *V4Decryptor) deriveKeys(key []byte, salt []byte) ([]byte, []byte) {
	// 生成加密密钥
	encKey := pbkdf2.Key(key, salt, d.iterCount, common.KeySize, d.hashFunc)

	// 生成MAC密钥
	macSalt := common.XorBytes(salt, 0x3a)
	macKey := pbkdf2.Key(encKey, macSalt, 2, common.KeySize, d.hashFunc)

	return encKey, macKey
}

// Validate 验证密钥是否有效
func (d *V4Decryptor) Validate(page1 []byte, key []byte) bool {
	_, _, ok := d.resolveKeys(page1, key)
	return ok
}

func (d *V4Decryptor) resolveKeys(page1 []byte, key []byte) ([]byte, []byte, bool) {
	if len(page1) < d.pageSize || len(key) != common.KeySize {
		return nil, nil, false
	}

	if d.validateEncKey(page1, key) {
		return key, d.deriveMacKey(key, page1[:common.SaltSize]), true
	}

	salt := page1[:common.SaltSize]
	if common.ValidateKey(page1, key, salt, d.hashFunc, d.hmacSize, d.reserve, d.pageSize, d.deriveKeys) {
		encKey, macKey := d.deriveKeys(key, salt)
		return encKey, macKey, true
	}

	return nil, nil, false
}

func (d *V4Decryptor) deriveMacKey(encKey []byte, salt []byte) []byte {
	macSalt := common.XorBytes(salt, 0x3a)
	return pbkdf2.Key(encKey, macSalt, 2, common.KeySize, d.hashFunc)
}

func (d *V4Decryptor) validateEncKey(page1 []byte, encKey []byte) bool {
	macKey := d.deriveMacKey(encKey, page1[:common.SaltSize])
	dataEnd := d.pageSize - d.reserve + common.IVSize
	storedMAC := page1[d.pageSize-d.hmacSize : d.pageSize]

	mac := hmac.New(d.hashFunc, macKey)
	mac.Write(page1[common.SaltSize:dataEnd])

	pageNoBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(pageNoBytes, 1)
	mac.Write(pageNoBytes)

	return hmac.Equal(mac.Sum(nil), storedMAC)
}

// Decrypt 解密数据库
func (d *V4Decryptor) Decrypt(ctx context.Context, dbfile string, hexKey string, output io.Writer) error {
	// 解码密钥
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return errors.DecodeKeyFailed(err)
	}

	// 打开数据库文件并读取基本信息
	dbInfo, err := common.OpenDBFile(dbfile, d.pageSize)
	if err != nil {
		return err
	}

	// 验证并解析密钥。Windows 内存扫描拿到的是 SQLCipher 页加密 enc_key；旧路径可能传入 raw key。
	encKey, macKey, ok := d.resolveKeys(dbInfo.FirstPage, key)
	if !ok {
		return errors.ErrDecryptIncorrectKey
	}

	// 打开数据库文件
	dbFile, err := os.Open(dbfile)
	if err != nil {
		return errors.OpenFileFailed(dbfile, err)
	}
	defer dbFile.Close()

	// 写入SQLite头
	_, err = output.Write([]byte(common.SQLiteHeader))
	if err != nil {
		return errors.WriteOutputFailed(err)
	}

	// 处理每一页
	pageBuf := make([]byte, d.pageSize)

	for curPage := int64(0); curPage < dbInfo.TotalPages; curPage++ {
		// 检查是否取消
		select {
		case <-ctx.Done():
			return errors.ErrDecryptOperationCanceled
		default:
			// 继续处理
		}

		// 读取一页
		n, err := io.ReadFull(dbFile, pageBuf)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// 处理最后一部分页面
				if n > 0 {
					break
				}
			}
			return errors.ReadFileFailed(dbfile, err)
		}

		// 检查页面是否全为零
		allZeros := true
		for _, b := range pageBuf {
			if b != 0 {
				allZeros = false
				break
			}
		}

		if allZeros {
			// 写入零页面
			_, err = output.Write(pageBuf)
			if err != nil {
				return errors.WriteOutputFailed(err)
			}
			continue
		}

		// 解密页面
		decryptedData, err := common.DecryptPage(pageBuf, encKey, macKey, curPage, d.hashFunc, d.hmacSize, d.reserve, d.pageSize)
		if err != nil {
			return err
		}

		// 写入解密后的页面
		_, err = output.Write(decryptedData)
		if err != nil {
			return errors.WriteOutputFailed(err)
		}
	}

	return nil
}

// GetPageSize 返回页面大小
func (d *V4Decryptor) GetPageSize() int {
	return d.pageSize
}

// GetReserve 返回保留字节数
func (d *V4Decryptor) GetReserve() int {
	return d.reserve
}

// GetHMACSize 返回HMAC大小
func (d *V4Decryptor) GetHMACSize() int {
	return d.hmacSize
}

// GetVersion 返回解密器版本
func (d *V4Decryptor) GetVersion() string {
	return d.version
}

// GetIterCount 返回迭代次数（Windows特有）
func (d *V4Decryptor) GetIterCount() int {
	return d.iterCount
}

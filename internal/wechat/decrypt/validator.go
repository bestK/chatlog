package decrypt

import (
	"path/filepath"

	"github.com/sjzar/chatlog/internal/wechat/decrypt/common"
	"github.com/sjzar/chatlog/pkg/util/dat2img"
)

type Validator struct {
	platform        string
	dbPath          string
	decryptor       Decryptor
	dbFile          *common.DBFile
	imgKeyValidator *dat2img.AesKeyValidator
}

// NewValidator 创建一个仅用于验证的验证器
func NewValidator(platform string, dataDir string) (*Validator, error) {
	return NewValidatorWithFile(platform, dataDir)
}

func NewValidatorWithFile(platform string, dataDir string) (*Validator, error) {
	dbFile := GetSimpleDBFile(platform)
	dbPath := filepath.Join(dataDir, dbFile)
	decryptor, err := NewDecryptor(platform)
	if err != nil {
		return nil, err
	}
	d, err := common.OpenDBFile(dbPath, decryptor.GetPageSize())
	if err != nil {
		return nil, err
	}

	validator := &Validator{
		platform:  platform,
		dbPath:    dbPath,
		decryptor: decryptor,
		dbFile:    d,
	}

	validator.imgKeyValidator = dat2img.NewImgKeyValidator(dataDir)

	return validator, nil
}

func (v *Validator) Validate(key []byte) bool {
	return v.decryptor.Validate(v.dbFile.FirstPage, key)
}

func (v *Validator) ValidateImgKey(key []byte) bool {
	if v.imgKeyValidator == nil {
		return false
	}
	return v.imgKeyValidator.Validate(key)
}

func GetSimpleDBFile(platform string) string {
	switch platform {
	case "windows":
		return "db_storage\\message\\message_0.db"
	case "darwin":
		return "db_storage/message/message_0.db"
	}
	return ""

}

package windows

import (
	"github.com/sjzar/chatlog/internal/wechat/decrypt"
)

type V4Extractor struct {
	validator *decrypt.Validator
}

func NewV4Extractor() *V4Extractor {
	return &V4Extractor{}
}

func (e *V4Extractor) SetValidate(validator *decrypt.Validator) {
	e.validator = validator
}

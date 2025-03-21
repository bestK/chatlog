package windows

import (
	"errors"

	"github.com/sjzar/chatlog/internal/wechat/decrypt"
)

// Common error definitions
var (
	ErrWeChatOffline    = errors.New("wechat is not logged in")
	ErrOpenProcess      = errors.New("failed to open process")
	ErrCheckProcessBits = errors.New("failed to check process architecture")
	ErrFindWeChatDLL    = errors.New("WeChatWin.dll module not found")
	ErrNoValidKey       = errors.New("no valid key found")
)

type V3Extractor struct {
	validator *decrypt.Validator
}

func NewV3Extractor() *V3Extractor {
	return &V3Extractor{}
}

func (e *V3Extractor) SetValidate(validator *decrypt.Validator) {
	e.validator = validator
}

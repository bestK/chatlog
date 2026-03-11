package key

import (
	"context"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/decrypt"
	"github.com/sjzar/chatlog/internal/wechat/key/darwin"
	"github.com/sjzar/chatlog/internal/wechat/key/windows"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

// Extractor 定义密钥提取器接口
type Extractor interface {
	// Extract 从进程中提取密钥（兼容旧接口）
	// dataKey, imgKey, error
	Extract(ctx context.Context, proc *model.Process) (string, string, error)

	// ExtractDataKey 仅提取数据库密钥
	ExtractDataKey(ctx context.Context, proc *model.Process) (string, error)

	// ExtractImgKey 仅提取图片密钥
	ExtractImgKey(ctx context.Context, proc *model.Process) (string, error)

	// SearchKey 在内存中搜索密钥
	SearchKey(ctx context.Context, memory []byte) (string, bool)

	SetValidate(validator *decrypt.Validator)
	SetProgress(progress func(string))
}

// NewExtractor 创建适合当前平台的密钥提取器
func NewExtractor(platform string) (Extractor, error) {
	switch platform {
	case "windows":
		return windows.NewV4Extractor(), nil
	case "darwin":
		return darwin.NewV4Extractor(), nil
	default:
		return nil, errors.PlatformUnsupported(platform)
	}
}

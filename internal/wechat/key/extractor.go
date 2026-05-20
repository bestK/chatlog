package key

import (
	"context"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/decrypt"
	"github.com/sjzar/chatlog/internal/wechat/key/darwin"
	"github.com/sjzar/chatlog/internal/wechat/key/windows"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

// Keys holds extracted encryption keys.
type Keys struct {
	DataKey string // db encryption key (hex string or JSON map for multi-salt)
	ImgKey  string // image AES key
}

// ExtractOpts controls which keys to extract and how to report progress.
type ExtractOpts struct {
	NeedDataKey bool
	NeedImgKey  bool
	OnProgress  func(string)
}

// Extractor extracts encryption keys from a running WeChat process.
type Extractor interface {
	Extract(ctx context.Context, proc *model.Process, opts ExtractOpts) (Keys, error)
}

// ExtractorOption configures an Extractor at construction time.
type ExtractorOption func(*extractorConfig)

type extractorConfig struct {
	validator *decrypt.Validator
}

// WithValidator injects a key validator (required for darwin).
func WithValidator(v *decrypt.Validator) ExtractorOption {
	return func(c *extractorConfig) { c.validator = v }
}

// NewExtractor creates a platform-appropriate key extractor.
func NewExtractor(platform string, opts ...ExtractorOption) (Extractor, error) {
	cfg := &extractorConfig{}
	for _, o := range opts {
		o(cfg)
	}
	switch platform {
	case "windows":
		return &windowsAdapter{impl: windows.NewV4Extractor()}, nil
	case "darwin":
		return &darwinAdapter{impl: darwin.NewV4Extractor(cfg.validator)}, nil
	default:
		return nil, errors.PlatformUnsupported(platform)
	}
}

// windowsAdapter wraps windows.V4Extractor to satisfy the Extractor interface.
type windowsAdapter struct {
	impl *windows.V4Extractor
}

func (a *windowsAdapter) Extract(ctx context.Context, proc *model.Process, opts ExtractOpts) (Keys, error) {
	dataKey, imgKey, err := a.impl.Extract(ctx, proc, opts.NeedDataKey, opts.NeedImgKey, opts.OnProgress)
	return Keys{DataKey: dataKey, ImgKey: imgKey}, err
}

// darwinAdapter wraps darwin.V4Extractor to satisfy the Extractor interface.
type darwinAdapter struct {
	impl *darwin.V4Extractor
}

func (a *darwinAdapter) Extract(ctx context.Context, proc *model.Process, opts ExtractOpts) (Keys, error) {
	dataKey, imgKey, err := a.impl.Extract(ctx, proc, opts.NeedDataKey, opts.NeedImgKey, opts.OnProgress)
	return Keys{DataKey: dataKey, ImgKey: imgKey}, err
}

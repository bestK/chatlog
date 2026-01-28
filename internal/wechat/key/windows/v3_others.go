//go:build !windows

package windows

import (
	"context"

	"github.com/sjzar/chatlog/internal/wechat/model"
)

func (e *V3Extractor) Extract(ctx context.Context, proc *model.Process) (string, string, error) {
	return "", "", nil
}

func (e *V3Extractor) ExtractDataKey(ctx context.Context, proc *model.Process) (string, error) {
	return "", nil
}

func (e *V3Extractor) ExtractImgKey(ctx context.Context, proc *model.Process) (string, error) {
	return "", nil
}

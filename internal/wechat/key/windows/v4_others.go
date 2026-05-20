//go:build !windows

package windows

import (
	"context"

	"github.com/sjzar/chatlog/internal/wechat/model"
)

func (e *V4Extractor) Extract(ctx context.Context, proc *model.Process, needDataKey, needImgKey bool, onProgress func(string)) (string, string, error) {
	return "", "", nil
}

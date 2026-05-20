package windows

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

// Extract extracts keys from a running WeChat process.
// Returns (dataKey, imgKey, error). Either key may be empty if not requested or extraction fails partially.
func (e *V4Extractor) Extract(ctx context.Context, proc *model.Process, needDataKey, needImgKey bool, onProgress func(string)) (string, string, error) {
	if proc.Status == model.StatusOffline {
		return "", "", errors.ErrWeChatOffline
	}

	var dataKey, imgKey string
	var dataErr, imgErr error

	if needDataKey {
		log.Info().Uint32("pid", proc.PID).Str("account", proc.AccountName).Msg("windows data key extraction started")
		dataKey, dataErr = ScanDataKey(ctx, proc.DataDir, proc.PID, onProgress)
		if dataErr != nil {
			log.Error().Err(dataErr).Uint32("pid", proc.PID).Msg("data key extraction failed")
		} else {
			log.Info().Uint32("pid", proc.PID).Msg("data key extraction succeeded")
		}
	}

	if needImgKey {
		log.Info().Uint32("pid", proc.PID).Msg("windows img key extraction started")
		imgKey, imgErr = ScanImgKey(ctx, proc.DataDir, proc.PID, onProgress)
		if imgErr != nil {
			log.Error().Err(imgErr).Uint32("pid", proc.PID).Msg("img key extraction failed")
		} else {
			log.Info().Uint32("pid", proc.PID).Msg("img key extraction succeeded")
		}
	}

	if dataKey == "" && imgKey == "" {
		if dataErr != nil {
			return "", "", dataErr
		}
		if imgErr != nil {
			return "", "", imgErr
		}
		return "", "", errors.ErrNoValidKey
	}

	return dataKey, imgKey, nil
}
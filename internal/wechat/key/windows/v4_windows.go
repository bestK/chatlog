package windows

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

const (
	// 定义 DLL 路径，假设在当前目录下
	DllPath = "wx_key.dll"
)

func (e *V4Extractor) Extract(ctx context.Context, proc *model.Process) (string, string, error) {
	dataKey, err := e.ExtractDataKey(ctx, proc)
	if err != nil && dataKey == "" {
		// 如果 dataKey 获取失败，仍然尝试获取 imgKey
		log.Error().Err(err).Msg("Failed to get data key, trying image key")
	}

	imgKey, imgErr := e.ExtractImgKey(ctx, proc)
	if imgErr != nil && imgKey == "" {
		log.Error().Err(imgErr).Msg("Failed to get image key")
	}

	if dataKey == "" && imgKey == "" {
		return "", "", errors.ErrNoValidKey
	}

	return dataKey, imgKey, nil
}

// ExtractDataKey 仅提取数据库密钥
func (e *V4Extractor) ExtractDataKey(ctx context.Context, proc *model.Process) (string, error) {
	if proc.Status == model.StatusOffline {
		return "", errors.ErrWeChatOffline
	}

	resultChan := make(chan struct {
		dataKey string
		err     error
	})

	go func() {
		// 获取数据库密钥 - 使用完整流程（自动重启微信）
		dbKeyResult := GetDbKeyFull(DllPath, 60, func(msg string) {
			log.Debug().Msg(msg)
		})

		if dbKeyResult.Success {
			resultChan <- struct {
				dataKey string
				err     error
			}{dbKeyResult.Key, nil}
		} else {
			resultChan <- struct {
				dataKey string
				err     error
			}{"", errors.ErrNoValidKey}
		}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-resultChan:
		return res.dataKey, res.err
	}
}

// ExtractImgKey 仅提取图片密钥
func (e *V4Extractor) ExtractImgKey(ctx context.Context, proc *model.Process) (string, error) {
	if proc.Status == model.StatusOffline {
		return "", errors.ErrWeChatOffline
	}

	resultChan := make(chan struct {
		imgKey string
		err    error
	})

	go func() {
		// 获取图片密钥 - 如果 DataDir 为空，让 GetImageKeys 自动查找缓存目录
		dataDir := proc.DataDir
		if dataDir == "" {
			log.Debug().Msg("DataDir is empty, will auto-detect cache directory")
		}
		imgKeyResult := GetImageKeys(dataDir, func(msg string) {
			log.Debug().Msg(msg)
		})

		if imgKeyResult.Success {
			resultChan <- struct {
				imgKey string
				err    error
			}{imgKeyResult.AesKey, nil}
		} else {
			resultChan <- struct {
				imgKey string
				err    error
			}{"", fmt.Errorf("%s", imgKeyResult.Error)}
		}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-resultChan:
		return res.imgKey, res.err
	}
}

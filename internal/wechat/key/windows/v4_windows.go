package windows

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

const (
	// 定义 DLL 路径，假设在当前目录下
	DllPath = "wx_key.dll"
)

func (e *V4Extractor) Extract(ctx context.Context, proc *model.Process) (string, string, error) {
	if proc.Status == model.StatusOffline {
		return "", "", errors.ErrWeChatOffline
	}

	resultChan := make(chan struct {
		dataKey string
		imgKey  string
		err     error
	})

	go func() {
		// 获取数据库密钥
		dbKeyResult := GetDbKey(DllPath, proc.PID, 15, func(msg string) {
			log.Debug().Msg(msg)
		})

		var dataKey string
		if dbKeyResult.Success {
			dataKey = dbKeyResult.Key
		} else {
			log.Error().Msgf("Failed to get DB key: %s", dbKeyResult.Error)
		}

		// 获取图片密钥
		imgKeyResult := GetImageKeys("", func(msg string) {
			log.Debug().Msg(msg)
		})

		var imgKey string
		if imgKeyResult.Success {
			imgKey = imgKeyResult.AesKey
		} else {
			log.Error().Msgf("Failed to get Image key: %s", imgKeyResult.Error)
		}

		if dataKey == "" && imgKey == "" {
			resultChan <- struct {
				dataKey string
				imgKey  string
				err     error
			}{"", "", errors.ErrNoValidKey}
			return
		}

		resultChan <- struct {
			dataKey string
			imgKey  string
			err     error
		}{dataKey, imgKey, nil}
	}()

	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case res := <-resultChan:
		return res.dataKey, res.imgKey, res.err
	}
}

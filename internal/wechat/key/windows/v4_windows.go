package windows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

const (
	// DLL 文件名
	DllName = "wx_key.dll"
)

// getDllPath 返回 wx_key.dll 的绝对路径（基于 exe 所在目录）
func getDllPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return DllName // fallback: 使用相对路径
	}
	return filepath.Join(filepath.Dir(exePath), DllName)
}

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
		// 优先返回 dataKey 的错误，其次返回 imgKey 的错误
		if err != nil {
			return "", "", err
		}
		if imgErr != nil {
			return "", "", imgErr
		}
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
		dbKeyResult := GetDbKeyFull(getDllPath(), 60, func(msg string) {
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
			}{"", fmt.Errorf("%s", dbKeyResult.Error)}
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
		imgKeyResult := GetImageKeys(dataDir, proc.PID, func(msg string) {
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

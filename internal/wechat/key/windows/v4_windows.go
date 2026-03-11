package windows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

const (
	DllName               = "wx_key.dll"
	DataKeyTimeoutSeconds = 20
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
	log.Info().
		Uint32("pid", proc.PID).
		Str("account", proc.AccountName).
		Int("timeout_seconds", DataKeyTimeoutSeconds).
		Msg("windows data key extraction started")

	resultChan := make(chan struct {
		dataKey string
		err     error
	})

	go func() {
		progressMessages := make([]string, 0, 24)

		dbKeyResult := GetDbKeyFull(getDllPath(), DataKeyTimeoutSeconds, func(msg string) {
			if msg != "" {
				progressMessages = append(progressMessages, msg)
			}
			if e.progress != nil {
				e.progress(msg)
			}
			log.Info().
				Uint32("pid", proc.PID).
				Str("progress", msg).
				Msg("windows data key extraction progress")
		})

		if dbKeyResult.Success {
			log.Info().
				Uint32("pid", proc.PID).
				Msg("windows data key extraction succeeded")
			resultChan <- struct {
				dataKey string
				err     error
			}{dbKeyResult.Key, nil}
		} else {
			errorMessage := strings.TrimSpace(dbKeyResult.Error)
			progressMessage := strings.TrimSpace(dbKeyResult.Message)
			if progressMessage == "" && len(progressMessages) > 0 {
				progressMessage = strings.Join(progressMessages, "\n")
			}
			if progressMessage != "" {
				if errorMessage != "" {
					errorMessage = fmt.Sprintf("%s\n\n过程信息：\n%s", errorMessage, progressMessage)
				} else {
					errorMessage = progressMessage
				}
			}
			log.Info().
				Uint32("pid", proc.PID).
				Str("error", errorMessage).
				Msg("windows data key extraction failed")
			resultChan <- struct {
				dataKey string
				err     error
			}{"", fmt.Errorf("%s", errorMessage)}
		}
	}()

	select {
	case <-ctx.Done():
		log.Info().
			Uint32("pid", proc.PID).
			Err(ctx.Err()).
			Msg("windows data key extraction canceled or timed out")
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

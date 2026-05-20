package wechat

import (
	"context"
	"os"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/decrypt"
	"github.com/sjzar/chatlog/internal/wechat/key"
	"github.com/sjzar/chatlog/internal/wechat/model"
	"github.com/sjzar/chatlog/pkg/util"
)

// Account 表示一个微信账号
type Account struct {
	Name     string
	Platform string

	FullVersion string
	DataDir     string
	Key         string
	ImgKey      string
	PID         uint32
	ExePath     string
	Status      string
}

func (a *Account) resolveProcess() (*model.Process, error) {
	Load()

	if a.Name != "" {
		if process, err := GetProcess(a.Name); err == nil {
			return process, nil
		}
	}

	if a.PID != 0 {
		if process, err := GetProcessByPID(a.PID); err == nil {
			return process, nil
		}
	}

	normalizedDataDir := util.NormalizeDataDirPath(a.DataDir)
	if normalizedDataDir != "" {
		for _, process := range GetProcesses() {
			if util.NormalizeDataDirPath(process.DataDir) == normalizedDataDir {
				return process, nil
			}
		}
	}

	return nil, errors.WeChatAccountNotFound(a.Name)
}

// NewAccount 创建新的账号对象
func NewAccount(proc *model.Process) *Account {
	return &Account{
		Name:     proc.AccountName,
		Platform: proc.Platform,

		FullVersion: proc.FullVersion,
		DataDir:     util.NormalizeDataDirPath(proc.DataDir),
		PID:         proc.PID,
		ExePath:     proc.ExePath,
		Status:      proc.Status,
	}
}

// RefreshStatus 刷新账号的进程状态
func (a *Account) RefreshStatus() error {
	process, err := a.resolveProcess()
	if err != nil {
		a.Status = model.StatusOffline
		return nil
	}

	if process.AccountName != "" {
		a.Name = process.AccountName
	}

	// 更新进程信息
	a.PID = process.PID
	a.ExePath = process.ExePath
	a.Platform = process.Platform

	a.FullVersion = process.FullVersion
	a.Status = process.Status
	a.DataDir = util.NormalizeDataDirPath(process.DataDir)

	return nil
}

// GetKeys extracts encryption keys from the running WeChat process.
// It returns cached keys when available and only extracts what's needed.
func (a *Account) GetKeys(ctx context.Context, opts key.ExtractOpts) (key.Keys, error) {
	result := key.Keys{}

	// Return cached keys if available
	if opts.NeedDataKey && a.Key != "" {
		result.DataKey = a.Key
		opts.NeedDataKey = false
	}
	if opts.NeedImgKey && a.ImgKey != "" {
		result.ImgKey = a.ImgKey
		opts.NeedImgKey = false
	}
	if !opts.NeedDataKey && !opts.NeedImgKey {
		return result, nil
	}

	// Refresh process status
	if err := a.RefreshStatus(); err != nil {
		return result, errors.RefreshProcessStatusFailed(err)
	}
	if a.Status != model.StatusOnline {
		return result, errors.WeChatAccountNotOnline(a.Name)
	}

	// Resolve process and create extractor
	process, err := a.resolveProcess()
	if err != nil {
		return result, err
	}

	var extractorOpts []key.ExtractorOption
	if process.Platform != "windows" {
		validator, err := decrypt.NewValidator(process.Platform, process.DataDir)
		if err != nil {
			return result, err
		}
		extractorOpts = append(extractorOpts, key.WithValidator(validator))
	}

	extractor, err := key.NewExtractor(a.Platform, extractorOpts...)
	if err != nil {
		return result, err
	}

	keys, err := extractor.Extract(ctx, process, opts)
	if err != nil {
		return result, err
	}

	// Cache extracted keys
	if keys.DataKey != "" {
		a.Key = keys.DataKey
		result.DataKey = keys.DataKey
	}
	if keys.ImgKey != "" {
		a.ImgKey = keys.ImgKey
		result.ImgKey = keys.ImgKey
	}

	return result, nil
}

// GetDataKeyWithProgress is a convenience wrapper for GetKeys (data key only).
func (a *Account) GetDataKeyWithProgress(ctx context.Context, onProgress func(string)) (string, error) {
	keys, err := a.GetKeys(ctx, key.ExtractOpts{NeedDataKey: true, OnProgress: onProgress})
	return keys.DataKey, err
}

// GetKeyWithProgress extracts both data key and image key.
func (a *Account) GetKeyWithProgress(ctx context.Context, onProgress func(string)) (string, string, error) {
	keys, err := a.GetKeys(ctx, key.ExtractOpts{NeedDataKey: true, NeedImgKey: true, OnProgress: onProgress})
	return keys.DataKey, keys.ImgKey, err
}

// GetImgKey extracts only the image key.
func (a *Account) GetImgKey(ctx context.Context) (string, error) {
	keys, err := a.GetKeys(ctx, key.ExtractOpts{NeedImgKey: true})
	return keys.ImgKey, err
}

// DecryptDatabase 解密数据库
func (a *Account) DecryptDatabase(ctx context.Context, dbPath, outputPath string) error {
	keys, err := a.GetKeys(ctx, key.ExtractOpts{NeedDataKey: true})
	if err != nil {
		return err
	}

	// 创建解密器 - 传入平台信息和版本
	decryptor, err := decrypt.NewDecryptor(a.Platform)
	if err != nil {
		return err
	}

	// 创建输出文件
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	// 解密数据库
	return decryptor.Decrypt(ctx, dbPath, keys.DataKey, output)
}

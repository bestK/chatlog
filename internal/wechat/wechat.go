package wechat

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog/log"
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

// GetDataKey 获取数据库密钥
func (a *Account) GetDataKey(ctx context.Context) (string, error) {
	return a.getDataKey(ctx, nil)
}

func (a *Account) GetDataKeyWithProgress(ctx context.Context, onProgress func(string)) (string, error) {
	return a.getDataKey(ctx, onProgress)
}

func (a *Account) getDataKey(ctx context.Context, onProgress func(string)) (string, error) {
	startedAt := time.Now()
	log.Info().
		Str("account", a.Name).
		Uint32("pid", a.PID).
		Str("platform", a.Platform).
		Msg("account get data key entered")
	// 如果已经有密钥，直接返回
	if a.Key != "" {
		log.Info().
			Str("account", a.Name).
			Uint32("pid", a.PID).
			Msg("account get data key hit cached key")
		return a.Key, nil
	}

	// 刷新进程状态
	if err := a.RefreshStatus(); err != nil {
		log.Info().Err(err).Str("account", a.Name).Uint32("pid", a.PID).Msg("account refresh status failed before get data key")
		return "", errors.RefreshProcessStatusFailed(err)
	}
	log.Info().
		Str("account", a.Name).
		Uint32("pid", a.PID).
		Str("status", a.Status).
		Str("data_dir", a.DataDir).
		Dur("elapsed", time.Since(startedAt)).
		Msg("account refresh status completed before get data key")

	// 检查账号状态
	if a.Status != model.StatusOnline {
		log.Info().Str("account", a.Name).Uint32("pid", a.PID).Str("status", a.Status).Msg("account is not online for data key extraction")
		return "", errors.WeChatAccountNotOnline(a.Name)
	}

	// 创建密钥提取器
	extractor, err := key.NewExtractor(a.Platform)
	if err != nil {
		log.Info().Err(err).Str("account", a.Name).Uint32("pid", a.PID).Str("platform", a.Platform).Msg("create key extractor failed")
		return "", err
	}
	log.Info().Str("account", a.Name).Uint32("pid", a.PID).Str("platform", a.Platform).Msg("key extractor created")

	process, err := a.resolveProcess()
	if err != nil {
		log.Info().Err(err).Str("account", a.Name).Uint32("pid", a.PID).Msg("resolve process failed before data key extraction")
		return "", err
	}
	log.Info().
		Str("account", a.Name).
		Uint32("pid", process.PID).
		Str("process_data_dir", process.DataDir).
		Dur("elapsed", time.Since(startedAt)).
		Msg("resolve process completed before data key extraction")

	extractor.SetProgress(onProgress)
	if process.Platform != "windows" {
		validator, err := decrypt.NewValidator(process.Platform, process.DataDir)
		if err != nil {
			log.Info().Err(err).Str("account", a.Name).Uint32("pid", process.PID).Str("process_data_dir", process.DataDir).Msg("create validator failed before data key extraction")
			return "", err
		}
		log.Info().
			Str("account", a.Name).
			Uint32("pid", process.PID).
			Dur("elapsed", time.Since(startedAt)).
			Msg("validator created before data key extraction")
		extractor.SetValidate(validator)
	} else {
		log.Info().
			Str("account", a.Name).
			Uint32("pid", process.PID).
			Dur("elapsed", time.Since(startedAt)).
			Msg("skip validator creation for windows data key extraction")
	}
	log.Info().
		Str("account", a.Name).
		Uint32("pid", process.PID).
		Dur("elapsed", time.Since(startedAt)).
		Msg("calling extractor extract data key")

	// 提取数据库密钥
	dataKey, err := extractor.ExtractDataKey(ctx, process)
	if err != nil {
		log.Info().
			Err(err).
			Str("account", a.Name).
			Uint32("pid", process.PID).
			Dur("elapsed", time.Since(startedAt)).
			Msg("extract data key failed")
		return "", err
	}

	if dataKey != "" {
		a.Key = dataKey
	}
	log.Info().
		Str("account", a.Name).
		Uint32("pid", process.PID).
		Dur("elapsed", time.Since(startedAt)).
		Msg("account get data key finished")

	return dataKey, nil
}

// GetImgKey 获取图片密钥
func (a *Account) GetImgKey(ctx context.Context) (string, error) {
	// 如果已经有密钥，直接返回
	if a.ImgKey != "" {
		return a.ImgKey, nil
	}

	// 刷新进程状态
	if err := a.RefreshStatus(); err != nil {
		return "", errors.RefreshProcessStatusFailed(err)
	}

	// 创建密钥提取器
	extractor, err := key.NewExtractor(a.Platform)
	if err != nil {
		return "", err
	}

	process, err := a.resolveProcess()
	if err != nil {
		return "", err
	}

	// 提取图片密钥
	imgKey, err := extractor.ExtractImgKey(ctx, process)
	if err != nil {
		return "", err
	}

	if imgKey != "" {
		a.ImgKey = imgKey
	}

	return imgKey, nil
}

// GetKey 获取账号的密钥（兼容旧接口）
func (a *Account) GetKey(ctx context.Context) (string, string, error) {
	dataKey, err := a.GetDataKey(ctx)
	if err != nil {
		return "", "", err
	}

	imgKey, err := a.GetImgKey(ctx)
	if err != nil {
		return dataKey, "", err
	}

	return dataKey, imgKey, nil
}

func (a *Account) GetKeyWithProgress(ctx context.Context, onProgress func(string)) (string, string, error) {
	dataKey, err := a.GetDataKeyWithProgress(ctx, onProgress)
	if err != nil {
		return "", "", err
	}

	imgKey, err := a.GetImgKey(ctx)
	if err != nil {
		return dataKey, "", err
	}

	return dataKey, imgKey, nil
}

// DecryptDatabase 解密数据库
func (a *Account) DecryptDatabase(ctx context.Context, dbPath, outputPath string) error {
	// 获取密钥
	hexKey, _, err := a.GetKey(ctx)
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
	return decryptor.Decrypt(ctx, dbPath, hexKey, output)
}

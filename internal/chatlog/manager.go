package chatlog

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/chatlog/ai"
	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/chatlog/database"
	"github.com/sjzar/chatlog/internal/chatlog/http"
	"github.com/sjzar/chatlog/internal/chatlog/wechat"
	iwechat "github.com/sjzar/chatlog/internal/wechat"
	"github.com/sjzar/chatlog/internal/wechatdb"
	"github.com/sjzar/chatlog/pkg/config"
	"github.com/sjzar/chatlog/pkg/filemonitor"
	"github.com/sjzar/chatlog/pkg/util"
	"github.com/sjzar/chatlog/pkg/util/dat2img"
)

// Manager 管理聊天日志应用
type Manager struct {
	ctx *ctx.Context
	sc  *conf.ServerConfig
	scm *config.Manager

	onSync func()

	// Services
	db     *database.Service
	http   *http.Service
	wechat *wechat.Service
	ai     *ai.Service

	imgKeyOnce sync.Once
}

func (m *Manager) resetImgKeyOnce() {
	m.imgKeyOnce = sync.Once{}
}

func New() *Manager {
	return &Manager{}
}

func (m *Manager) Init(configPath string) error {
	var err error
	m.ctx, err = ctx.New(configPath)
	if err != nil {
		return err
	}

	if m.ctx.GetDebug() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	m.wechat = wechat.NewService(m.ctx)
	m.db = database.NewService(m.ctx)
	m.ai = ai.New()
	m.http = http.NewService(m.ctx, m.db, m.ai, m.ctx)

	m.ctx.WeChatInstances = m.wechat.GetWeChatInstances()
	if len(m.ctx.WeChatInstances) >= 1 {
		m.ctx.SwitchCurrent(m.ctx.WeChatInstances[0])
	}
	_ = m.RefreshSession()

	if m.ctx.AutoDecrypt {
		go func() {
			if err := m.StartAutoDecrypt(); err != nil {
				log.Error().Err(err).Msg("启动时自动开启自动解密失败")
			} else {
				log.Info().Msg("启动时已自动开启自动解密服务")
			}
		}()
	}

	if m.ctx.HTTPEnabled {
		if err := m.StartService(); err != nil {
			_ = m.StopService()
		}
	}

	go m.CheckAndSyncData()
	return nil
}

func (m *Manager) Close() {
	if m == nil || m.ctx == nil {
		return
	}
	if m.ctx.HTTPEnabled {
		_ = m.StopService()
	}
	if m.ctx.AutoDecrypt {
		_ = m.StopAutoDecrypt()
	}
}

func (m *Manager) OnSync(fn func()) {
	m.onSync = fn
}

func (m *Manager) Context() *ctx.Context {
	return m.ctx
}

func (m *Manager) ReloadWebhook() error {
	if m == nil || m.db == nil {
		return nil
	}
	return m.db.ReloadWebhook()
}

func (m *Manager) SetWebhook(hook *conf.Webhook) error {
	if m == nil || m.ctx == nil {
		return fmt.Errorf("未初始化")
	}
	if err := m.ctx.SetWebhook(hook); err != nil {
		return err
	}
	return m.ReloadWebhook()
}

func (m *Manager) GetAIProviders() []*conf.AIProvider {
	if m == nil || m.ctx == nil {
		return nil
	}
	return m.ctx.GetAIProviders()
}

func (m *Manager) SetAIProviders(providers []*conf.AIProvider) error {
	if m == nil || m.ctx == nil {
		return fmt.Errorf("未初始化")
	}
	return m.ctx.SetAIProviders(providers)
}

func (m *Manager) ReloadWeChatInstances() {
	if m == nil || m.ctx == nil || m.wechat == nil {
		return
	}
	instances := m.wechat.GetWeChatInstances()
	for _, inst := range instances {
		if inst == nil {
			continue
		}
		_ = inst.RefreshStatus()
	}
	m.ctx.WeChatInstances = instances
}

func (m *Manager) Switch(info *iwechat.Account, history string) error {
	if m.ctx.AutoDecrypt {
		if err := m.StopAutoDecrypt(); err != nil {
			return err
		}
	}
	if m.ctx.HTTPEnabled {
		if err := m.stopService(); err != nil {
			return err
		}
	}
	if info != nil {
		m.ctx.SwitchCurrent(info)
	} else {
		m.ctx.SwitchHistory(history)
	}
	_ = m.RefreshSession()

	// 切换账号后自动恢复自动解密
	if m.ctx.AutoDecrypt {
		go func() {
			if err := m.StartAutoDecrypt(); err != nil {
				log.Error().Err(err).Msg("切换账号后自动开启自动解密失败")
			} else {
				log.Info().Msg("切换账号后已自动开启自动解密服务")
			}
		}()
	}

	if m.ctx.HTTPEnabled {
		// 启动HTTP服务
		if err := m.StartService(); err != nil {
			log.Info().Err(err).Msg("启动服务失败")
			m.StopService()
		}
	}
	return nil
}

func (m *Manager) StartService() error {
	if !m.ctx.AutoDecrypt {
		if err := m.StartAutoDecrypt(); err != nil {
			return fmt.Errorf("启动 HTTP 失败，自动解密开启失败: %w", err)
		}
	}

	// 按依赖顺序启动服务
	if err := m.db.Start(); err != nil {
		return err
	}
	m.syncCurrentProfile()

	if err := m.http.Start(); err != nil {
		m.db.Stop()
		return err
	}

	// 更新 xorkey
	dat2img.SetAesKey(m.ctx.ImgKey)
	dat2img.ScanAndSetXorKey(m.ctx.DataDir)

	// 更新状态
	m.ctx.SetHTTPEnabled(true)

	return nil
}

func (m *Manager) StopService() error {
	if err := m.stopService(); err != nil {
		return err
	}

	// 更新状态
	m.ctx.SetHTTPEnabled(false)

	return nil
}

func (m *Manager) stopService() error {
	// 按依赖的反序停止服务
	var errs []error

	if err := m.http.Stop(); err != nil {
		errs = append(errs, err)
	}

	if err := m.db.Stop(); err != nil {
		errs = append(errs, err)
	}

	// 如果有错误，返回第一个错误
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (m *Manager) SetHTTPAddr(text string) error {
	var addr string
	if util.IsNumeric(text) {
		addr = fmt.Sprintf("127.0.0.1:%s", text)
	} else if strings.HasPrefix(text, "http://") {
		addr = strings.TrimPrefix(text, "http://")
	} else if strings.HasPrefix(text, "https://") {
		addr = strings.TrimPrefix(text, "https://")
	} else {
		addr = text
	}
	m.ctx.SetHTTPAddr(addr)
	return nil
}

func (m *Manager) GetDataKey() (string, error) {
	return m.GetDataKeyWithProgress(nil)
}

func (m *Manager) currentAccountOrErr() (*iwechat.Account, error) {
	if m == nil || m.ctx == nil || m.ctx.Current == nil {
		return nil, fmt.Errorf("未选择任何账号，请先在[切换账号]菜单中选择一个运行中的微信进程")
	}
	return m.ctx.Current, nil
}

func (m *Manager) GetDataKeyWithProgress(onProgress func(string)) (string, error) {
	startedAt := time.Now()
	if err := m.stopAutoDecryptBeforeGetKey(onProgress); err != nil {
		return "", err
	}
	current, err := m.currentAccountOrErr()
	if err != nil {
		log.Info().Err(err).Msg("get data key aborted before extraction")
		return "", err
	}
	log.Info().
		Str("account", current.Name).
		Uint32("pid", current.PID).
		Str("platform", current.Platform).
		Msg("start getting data key for current account")
	if onProgress != nil {
		onProgress("正在读取当前账号的数据库密钥...")
	}

	dataKey, err := m.wechat.GetDataKeyWithProgress(current, onProgress)
	if err != nil {
		log.Info().
			Err(err).
			Str("account", current.Name).
			Uint32("pid", current.PID).
			Dur("elapsed", time.Since(startedAt)).
			Msg("get data key failed")
		return "", err
	}
	m.ctx.Refresh()
	m.ctx.UpdateConfig()
	log.Info().
		Str("account", current.Name).
		Uint32("pid", current.PID).
		Dur("elapsed", time.Since(startedAt)).
		Msg("get data key succeeded")

	return dataKey, nil
}

func (m *Manager) GetKeys() (string, string, error) {
	return m.GetKeysWithProgress(nil)
}

func (m *Manager) GetKeysWithProgress(onProgress func(string)) (string, string, error) {
	if err := m.stopAutoDecryptBeforeGetKey(onProgress); err != nil {
		return "", "", err
	}
	current, err := m.currentAccountOrErr()
	if err != nil {
		return "", "", err
	}
	if onProgress != nil {
		onProgress("正在读取当前账号的密钥信息...")
	}

	dataKey, imgKey, err := m.wechat.GetKeysWithProgress(current, onProgress)
	if err != nil {
		return "", "", err
	}
	m.ctx.Refresh()
	m.ctx.UpdateConfig()

	if imgKey != "" {
		dat2img.SetAesKey(imgKey)
		dat2img.ScanAndSetXorKey(m.ctx.DataDir)
		m.resetImgKeyOnce()
	}

	return dataKey, imgKey, nil
}

// GetImgKey 仅获取图片密钥（不会重启微信）
func (m *Manager) GetImgKey() (string, error) {
	if err := m.stopAutoDecryptBeforeGetKey(nil); err != nil {
		return "", err
	}
	current, err := m.currentAccountOrErr()
	if err != nil {
		return "", err
	}

	imgKey, err := m.wechat.GetImgKey(current)
	if err != nil {
		return "", err
	}
	m.ctx.Refresh()
	m.ctx.UpdateConfig()

	// 更新图片解密密钥
	if imgKey != "" {
		dat2img.SetAesKey(imgKey)
		dat2img.ScanAndSetXorKey(m.ctx.DataDir)
		m.resetImgKeyOnce()
	}

	return imgKey, nil
}

func (m *Manager) stopAutoDecryptBeforeGetKey(onProgress func(string)) error {
	if m == nil || m.ctx == nil || !m.ctx.AutoDecrypt {
		return nil
	}

	if onProgress != nil {
		onProgress("正在停止自动解密...")
	}
	log.Info().Msg("获取 key 前先停止自动解密")
	return m.StopAutoDecrypt()
}

func (m *Manager) DecryptDBFiles() error {
	if m.ctx.DataKey == "" {
		if m.ctx.Current == nil {
			return fmt.Errorf("未选择任何账号")
		}
		if _, err := m.GetDataKey(); err != nil {
			return err
		}
	}
	if m.ctx.WorkDir == "" {
		m.ctx.WorkDir = util.DefaultWorkDir(m.ctx.Account)
	}

	if err := m.wechat.DecryptDBFiles(); err != nil {
		return err
	}
	m.ctx.Refresh()
	m.ctx.UpdateConfig()
	return nil
}

func (m *Manager) StartAutoDecrypt() error {
	if m.ctx.DataKey == "" || m.ctx.DataDir == "" {
		return fmt.Errorf("请先获取密钥")
	}
	if m.ctx.WorkDir == "" {
		return fmt.Errorf("请先执行解密数据")
	}

	if err := m.wechat.StartAutoDecrypt(); err != nil {
		return err
	}

	m.ctx.SetAutoDecrypt(true)
	return nil
}

func (m *Manager) StopAutoDecrypt() error {
	if err := m.wechat.StopAutoDecrypt(); err != nil {
		return err
	}

	m.ctx.SetAutoDecrypt(false)
	return nil
}

func (m *Manager) RefreshSession() error {
	if m.db.GetDB() == nil {
		if err := m.db.Start(); err != nil {
			return err
		}
	}
	m.syncCurrentProfile()
	resp, err := m.db.GetSessions("", 1, 0)
	if err != nil {
		return err
	}
	if len(resp.Items) == 0 {
		return nil
	}
	m.ctx.LastSession = resp.Items[0].NTime.Time()
	return nil
}

func (m *Manager) GetContacts(keyword string, isInChatRoom, limit, offset int) (*wechatdb.GetContactsResp, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("未初始化")
	}
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	if m.db.GetDB() == nil {
		if err := m.db.Start(); err != nil {
			return nil, err
		}
		m.syncCurrentProfile()
	}
	return m.db.GetContacts(keyword, isInChatRoom, limit, offset)
}

func (m *Manager) GetMessages(timeStr, talker, sender, keyword string, limit, offset int, order string) (*wechatdb.GetMessagesResp, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("未初始化")
	}
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	if m.db.GetDB() == nil {
		if err := m.db.Start(); err != nil {
			return nil, err
		}
		m.syncCurrentProfile()
	}
	start, end, ok := util.TimeRangeOf(timeStr)
	if !ok {
		return nil, fmt.Errorf("invalid time range: %s", timeStr)
	}
	return m.db.GetMessages(start, end, talker, sender, keyword, limit, offset, order)
}

func (m *Manager) GetSelectedAIProvider() string {
	if m == nil || m.ctx == nil {
		return ""
	}
	return m.ctx.GetSelectedAIProvider()
}

func (m *Manager) SetSelectedAIProvider(providerID string) {
	if m == nil || m.ctx == nil {
		return
	}
	m.ctx.SetSelectedAIProvider(providerID)
}

func (m *Manager) GetSummaryPrompt() string {
	if m == nil || m.ctx == nil {
		return ""
	}
	return m.ctx.GetSummaryPrompt()
}

func (m *Manager) SetSummaryPrompt(prompt string) {
	if m == nil || m.ctx == nil {
		return
	}
	m.ctx.SetSummaryPrompt(prompt)
}

type MediaDataResult struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

func (m *Manager) GetMediaData(mediaType string, key string) (*MediaDataResult, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("未初始化")
	}
	if m.db.GetDB() == nil {
		if err := m.db.Start(); err != nil {
			return nil, err
		}
	}

	keys := util.Str2List(key, ",")
	if len(keys) == 0 {
		return nil, fmt.Errorf("key 为空")
	}

	for _, k := range keys {
		if strings.Contains(k, "/") {
			absPath := filepath.Join(m.ctx.GetDataDir(), k)
			return m.readMediaFile(absPath)
		}
		media, err := m.db.GetMedia(mediaType, k)
		if err != nil {
			continue
		}
		if media.Data != nil && len(media.Data) > 0 {
			return &MediaDataResult{
				Data:     "data:audio/mp3;base64," + encodeBase64(media.Data),
				MimeType: "audio/mp3",
			}, nil
		}
		absPath := filepath.Join(m.ctx.GetDataDir(), media.Path)
		return m.readMediaFile(absPath)
	}
	return nil, fmt.Errorf("媒体文件未找到")
}

func (m *Manager) readMediaFile(absPath string) (*MediaDataResult, error) {
	b, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	if ext == ".dat" {
		m.imgKeyOnce.Do(func() {
			if m.ctx.ImgKey != "" {
				dat2img.SetAesKey(m.ctx.ImgKey)
			}
			if m.ctx.DataDir != "" {
				dat2img.ScanAndSetXorKey(m.ctx.DataDir)
			}
		})
		out, imgExt, err := dat2img.Dat2Image(b)
		if err != nil {
			return nil, fmt.Errorf("图片解密失败: %v", err)
		}
		mime := "image/jpeg"
		switch imgExt {
		case "png":
			mime = "image/png"
		case "gif":
			mime = "image/gif"
		case "bmp":
			mime = "image/bmp"
		}
		return &MediaDataResult{
			Data:     "data:" + mime + ";base64," + encodeBase64(out),
			MimeType: mime,
		}, nil
	}

	mime := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".png":
		mime = "image/png"
	case ".gif":
		mime = "image/gif"
	case ".mp4":
		mime = "video/mp4"
	case ".mp3":
		mime = "audio/mp3"
	}
	return &MediaDataResult{
		Data:     "data:" + mime + ";base64," + encodeBase64(b),
		MimeType: mime,
	}, nil
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func (m *Manager) syncCurrentProfile() {
	if m == nil || m.ctx == nil || m.db == nil || m.db.GetDB() == nil {
		return
	}
	m.ctx.SmallHeadImgUrl = m.db.GetSelfSmallHeadImgUrl()
	m.ctx.Nickname = m.db.GetSelfName()
}

func (m *Manager) CommandKey(configPath string, pid int, force bool, showXorKey bool) (string, error) {

	var err error
	m.ctx, err = ctx.New(configPath)
	if err != nil {
		return "", err
	}

	m.wechat = wechat.NewService(m.ctx)

	m.ctx.WeChatInstances = m.wechat.GetWeChatInstances()
	if len(m.ctx.WeChatInstances) == 0 {
		return "", fmt.Errorf("wechat process not found")
	}

	if len(m.ctx.WeChatInstances) == 1 {
		current := m.ctx.WeChatInstances[0]
		m.ctx.SwitchCurrent(current)
		key, imgKey := m.ctx.DataKey, m.ctx.ImgKey
		if len(key) == 0 || len(imgKey) == 0 || force {
			key, imgKey, err = current.GetKeyWithProgress(context.Background(), nil)
			if err != nil {
				return "", err
			}
			m.ctx.Refresh()
			m.ctx.UpdateConfig()
		}

		result := fmt.Sprintf("Data Key: [%s]\nImage Key: [%s]", key, imgKey)
		if showXorKey {
			if b, err := dat2img.ScanAndSetXorKey(m.ctx.DataDir); err == nil {
				result += fmt.Sprintf("\nXor Key: [0x%X]", b)
			}
		}

		return result, nil
	}
	if pid == 0 {
		str := "Select a process:\n"
		for _, ins := range m.ctx.WeChatInstances {
			str += fmt.Sprintf("PID: %d. %s[Version: %s Data Dir: %s ]\n", ins.PID, ins.Name, ins.FullVersion, ins.DataDir)
		}
		return str, nil
	}
	for _, ins := range m.ctx.WeChatInstances {
		if ins.PID == uint32(pid) {
			m.ctx.SwitchCurrent(ins)
			key, imgKey := m.ctx.DataKey, m.ctx.ImgKey
			if len(key) == 0 || len(imgKey) == 0 || force {
				key, imgKey, err = ins.GetKeyWithProgress(context.Background(), nil)
				if err != nil {
					return "", err
				}
				m.ctx.Refresh()
				m.ctx.UpdateConfig()
			}
			result := fmt.Sprintf("Data Key: [%s]\nImage Key: [%s]", key, imgKey)
			if showXorKey {
				if b, err := dat2img.ScanAndSetXorKey(m.ctx.DataDir); err == nil {
					result += fmt.Sprintf("\nXor Key: [0x%X]", b)
				}
			}
			return result, nil
		}
	}
	return "", fmt.Errorf("wechat process not found")
}

func (m *Manager) ensureServiceDataKey(configPath string) error {
	if m.sc == nil || strings.TrimSpace(m.sc.GetDataKey()) != "" {
		return nil
	}

	appCtx, err := ctx.New(configPath)
	if err != nil {
		return err
	}
	keyService := wechat.NewService(appCtx)
	instances := keyService.GetWeChatInstances()
	if len(instances) == 0 {
		return fmt.Errorf("wechat process not found")
	}

	var selected *iwechat.Account
	dataDir := util.NormalizeDataDirPath(m.sc.GetDataDir())
	if dataDir != "" {
		for _, instance := range instances {
			if instance != nil && util.NormalizeDataDirPath(instance.DataDir) == dataDir {
				selected = instance
				break
			}
		}
	}
	if selected == nil {
		for _, account := range []string{m.sc.CurrentAccount, m.sc.LastAccount} {
			account = strings.TrimSpace(account)
			if account == "" {
				continue
			}
			for _, instance := range instances {
				if instance != nil && instance.Name == account {
					selected = instance
					break
				}
			}
			if selected != nil {
				break
			}
		}
	}
	if selected == nil && len(instances) == 1 {
		selected = instances[0]
	}
	if selected == nil {
		return fmt.Errorf("wechat process not found for data dir: %s", dataDir)
	}

	appCtx.WeChatInstances = instances
	appCtx.SwitchCurrent(selected)
	dataKey, imgKey, err := keyService.GetKeysWithProgress(selected, nil)
	if err != nil {
		return err
	}
	appCtx.Refresh()
	appCtx.UpdateConfig()

	m.ctx = appCtx
	m.sc.DataKey = dataKey
	if imgKey != "" {
		m.sc.ImgKey = imgKey
	}
	if m.sc.DataDir == "" {
		m.sc.DataDir = appCtx.DataDir
	}
	if m.sc.WorkDir == "" {
		m.sc.WorkDir = appCtx.WorkDir
	}
	return nil
}

func (m *Manager) CommandDecrypt(configPath string, overrides conf.ServerOverrides) error {

	var err error
	m.sc, m.scm, err = conf.LoadServiceConfig(configPath, overrides)
	if err != nil {
		return err
	}

	dataDir := m.sc.GetDataDir()
	if len(dataDir) == 0 {
		return fmt.Errorf("dataDir is required")
	}

	if err := m.ensureServiceDataKey(configPath); err != nil {
		return err
	}

	m.wechat = wechat.NewService(m.sc)

	if err := m.wechat.DecryptDBFiles(); err != nil {
		return err
	}

	return nil
}

func (m *Manager) CommandHTTPServer(configPath string, overrides conf.ServerOverrides) error {

	var err error
	m.sc, m.scm, err = conf.LoadServiceConfig(configPath, overrides)
	if err != nil {
		return err
	}

	// 根据配置设置日志级别
	log.Info().Msgf("debug config: %v", m.sc.GetDebug())
	if m.sc.GetDebug() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Info().Msg("debug mode enabled")
	}

	dataDir := m.sc.GetDataDir()
	workDir := m.sc.GetWorkDir()
	if len(dataDir) == 0 && len(workDir) == 0 {
		return fmt.Errorf("dataDir or workDir is required")
	}

	if err := m.ensureServiceDataKey(configPath); err != nil {
		return err
	}
	dataDir = m.sc.GetDataDir()
	workDir = m.sc.GetWorkDir()

	// 处理图片密钥
	if len(dataDir) != 0 {
		dat2img.SetAesKey(m.sc.GetImgKey())
		dat2img.ScanAndSetXorKey(dataDir)
	}

	log.Info().Msgf("server config: %+v", m.sc)

	m.wechat = wechat.NewService(m.sc)

	m.db = database.NewService(m.sc)

	// 注入 DBController，用于在解密替换文件时控制连接（锁定、关闭、解锁）
	m.wechat.SetDBController(m.db)

	if m.ai == nil {
		m.ai = ai.New()
	}
	m.http = http.NewService(m.sc, m.db, m.ai, m.ctx)

	if m.sc.GetAutoDecrypt() {
		if err := m.wechat.StartAutoDecrypt(); err != nil {
			return err
		}
		log.Info().Msg("auto decrypt is enabled")
	}

	// 启动时异步检查数据更新
	go m.CheckAndSyncData()

	// init db
	go func() {
		// 如果工作目录为空，则解密数据
		if entries, err := os.ReadDir(workDir); err == nil && len(entries) == 0 {
			log.Info().Msgf("work dir is empty, decrypt data.")
			m.db.SetDecrypting()
			if err := m.wechat.DecryptDBFiles(); err != nil {
				log.Info().Msgf("decrypt data failed: %v", err)
				return
			}
			log.Info().Msg("decrypt data success")
		}

		// 按依赖顺序启动服务
		if err := m.db.Start(); err != nil {
			log.Info().Msgf("start db failed, try to decrypt data.")
			m.db.SetDecrypting()
			if err := m.wechat.DecryptDBFiles(); err != nil {
				log.Info().Msgf("decrypt data failed: %v", err)
				return
			}
			log.Info().Msg("decrypt data success")
			if err := m.db.Start(); err != nil {
				log.Info().Msgf("start db failed: %v", err)
				m.db.SetError(err.Error())
				return
			}
		}
	}()

	return m.http.ListenAndServe()
}

func (m *Manager) CheckAndSyncData() {
	var dataKey, dataDir, workDir string

	if m.ctx != nil {
		dataKey = m.ctx.DataKey
		dataDir = m.ctx.DataDir
		workDir = m.ctx.WorkDir
	} else if m.sc != nil {
		dataKey = m.sc.DataKey
		dataDir = m.sc.DataDir
		workDir = m.sc.WorkDir
	}

	if dataKey == "" || dataDir == "" || workDir == "" {
		return
	}

	log.Info().Msgf("开始启动后异步数据检查... (DataDir: %s, WorkDir: %s)", dataDir, workDir)

	// 使用与原本逻辑一致的 FileGroup 进行文件扫描
	dbGroup, err := filemonitor.NewFileGroup("check_sync", dataDir, `.*\.db$`, []string{"fts"})
	if err != nil {
		log.Error().Err(err).Msg("异步检查创建文件组失败")
		return
	}

	dbFiles, err := dbGroup.List()
	if err != nil {
		log.Error().Err(err).Msg("异步检查扫描文件失败")
		return
	}

	updatedCount := 0
	for _, srcPath := range dbFiles {
		rel, err := filepath.Rel(dataDir, srcPath)
		if err != nil {
			continue
		}
		dstPath := filepath.Join(workDir, rel)

		srcInfo, err := os.Stat(srcPath)
		if err != nil {
			continue
		}

		needsUpdate := false
		dstInfo, err := os.Stat(dstPath)
		if err != nil {
			// 如果目标文件不存在，需要更新
			if os.IsNotExist(err) {
				needsUpdate = true
			} else {
				continue
			}
		} else {
			// 如果源文件比目标文件新，则需要更新
			if srcInfo.ModTime().After(dstInfo.ModTime()) {
				needsUpdate = true
			}
		}

		if needsUpdate {
			log.Debug().Msgf("检测到文件需要同步: %s", rel)
			if err := m.wechat.DecryptDBFile(srcPath); err != nil {
				log.Error().Err(err).Msgf("异步解密文件失败: %s", rel)
			} else {
				updatedCount++
			}
		}
	}

	if updatedCount > 0 {
		log.Info().Msgf("异步数据检查完成，共更新 %d 个文件", updatedCount)
		if m.ctx != nil {
			m.ctx.Refresh()
			m.ctx.UpdateConfig()
		}
		if m.onSync != nil {
			m.onSync()
		}
	} else {
		log.Info().Msg("异步数据检查完成，未发现更新")
	}
}

package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/chatlog"
	"github.com/sjzar/chatlog/internal/chatlog/ai"
	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/wechatdb"
	"github.com/sjzar/chatlog/pkg/util"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

type App struct {
	ctx         context.Context
	mgr         *chatlog.Manager
	closed      chan struct{}
	logPath     string
	logFile     *os.File
	logWatcher  *fsnotify.Watcher
	logDebounce *time.Timer
	stateEvents bool
	logEvents   bool
	logMu       sync.Mutex
}

type notifyingWriter struct {
	app *App
}

func (w *notifyingWriter) Write(p []byte) (int, error) {
	if w.app != nil {
		w.app.debounceLogChanged()
	}
	return len(p), nil
}

type State struct {
	Account         string `json:"account"`
	Platform        string `json:"platform"`
	FullVersion     string `json:"fullVersion"`
	DataDir         string `json:"dataDir"`
	DataKey         string `json:"dataKey"`
	ImgKey          string `json:"imgKey"`
	WorkDir         string `json:"workDir"`
	HTTPEnabled     bool   `json:"httpEnabled"`
	HTTPAddr        string `json:"httpAddr"`
	AutoDecrypt     bool   `json:"autoDecrypt"`
	LastSession     string `json:"lastSession"`
	PID             int    `json:"pid"`
	ExePath         string `json:"exePath"`
	Status          string `json:"status"`
	Nickname        string `json:"nickname"`
	SmallHeadImgUrl string `json:"smallHeadImgUrl"`
}

type Instance struct {
	Name        string `json:"name"`
	PID         int    `json:"pid"`
	Platform    string `json:"platform"`
	FullVersion string `json:"fullVersion"`
	DataDir     string `json:"dataDir"`
	ExePath     string `json:"exePath"`
	Status      string `json:"status"`
}

type WebhookItem struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Talker      string `json:"talker"`
	Sender      string `json:"sender"`
	Keyword     string `json:"keyword"`
	Disabled    bool   `json:"disabled"`
}

type WebhookConfig struct {
	Host    string        `json:"host"`
	DelayMs int64         `json:"delayMs"`
	Items   []WebhookItem `json:"items"`
}

type KeyProgressEvent struct {
	Operation string `json:"operation"`
	Message   string `json:"message"`
}

func newApp() *App {
	return &App{mgr: chatlog.New(), closed: make(chan struct{})}
}

func runGUI() error {
	app := newApp()
	return wails.Run(&options.App{
		Title:       "Chatlog",
		Width:       1100,
		Height:      760,
		MinWidth:    900,
		MinHeight:   640,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup:   app.startup,
		OnShutdown:  app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.initLogger()
	a.startLogWatcher()
	a.ensureFileLogging()
	a.mgr.OnSync(func() {
		_ = a.emitState()
	})
	if err := a.mgr.Init(""); err != nil {
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "启动失败",
			Message: err.Error(),
		})
		return
	}
	_ = a.emitState()
}

func (a *App) initLogger() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	_, _ = a.ensureLogFile()
	a.resetLogger()
	log.Info().Str("path", a.logPath).Msg("gui logger initialized")
}

func (a *App) resetLogger() {
	writers := make([]io.Writer, 0, 3)
	if util.HasUsableConsole(os.Stderr) {
		writers = append(writers, util.NewPlainLogWriter(os.Stderr, false))
	}
	if a.logFile != nil {
		writers = append(writers, util.NewPlainLogWriter(a.logFile, true))
	}
	writers = append(writers, &notifyingWriter{app: a})
	log.Logger = log.Output(io.MultiWriter(writers...))
}

func (a *App) ensureFileLogging() {
	if a.logFile != nil {
		return
	}
	if _, err := a.ensureLogFile(); err != nil {
		log.Error().Err(err).Msg("enable file logging failed")
		return
	}
	log.Info().Str("path", a.logPath).Msg("file logging enabled")
}

func (a *App) startLogWatcher() {
	if a.ctx == nil {
		return
	}
	path, err := a.GetLogPath()
	if err != nil {
		return
	}
	if a.logWatcher != nil {
		return
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	a.logWatcher = w
	dir := filepath.Dir(path)
	if err := w.Add(dir); err != nil {
		_ = w.Close()
		a.logWatcher = nil
		return
	}

	go func() {
		for {
			select {
			case <-a.closed:
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if !strings.EqualFold(filepath.Clean(ev.Name), filepath.Clean(path)) {
					continue
				}
				if !(ev.Op.Has(fsnotify.Write) || ev.Op.Has(fsnotify.Create) || ev.Op.Has(fsnotify.Rename)) {
					continue
				}
				a.debounceLogChanged()
			case <-w.Errors:
			}
		}
	}()
}

func (a *App) debounceLogChanged() {
	if a.ctx == nil {
		return
	}
	if !a.logEvents {
		return
	}
	a.logMu.Lock()
	defer a.logMu.Unlock()
	if a.logDebounce != nil {
		a.logDebounce.Stop()
	}
	a.logDebounce = time.AfterFunc(180*time.Millisecond, func() {
		runtime.EventsEmit(a.ctx, "log:changed", map[string]string{"path": a.logPath})
	})
}

func (a *App) ensureLogFile() (string, error) {
	if a.logFile != nil && a.logPath != "" {
		return a.logPath, nil
	}

	f, path, err := util.OpenLogFile()
	if err != nil {
		return "", err
	}

	if a.logFile != nil {
		_ = a.logFile.Close()
	}
	a.logFile = f
	a.logPath = path
	a.resetLogger()
	return path, nil
}

func (a *App) shutdown(ctx context.Context) {
	select {
	case <-a.closed:
	default:
		close(a.closed)
	}
	a.mgr.Close()
	if a.logWatcher != nil {
		_ = a.logWatcher.Close()
		a.logWatcher = nil
	}
	a.logMu.Lock()
	if a.logDebounce != nil {
		a.logDebounce.Stop()
		a.logDebounce = nil
	}
	a.logMu.Unlock()
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
}

func (a *App) emitState() error {
	if a.ctx == nil {
		return nil
	}
	if !a.stateEvents {
		return nil
	}
	st, err := a.GetState()
	if err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "state", st)
	return nil
}

func (a *App) emitKeyProgress(operation string, message string) {
	if a.ctx == nil {
		return
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	runtime.EventsEmit(a.ctx, "key:progress", KeyProgressEvent{
		Operation: operation,
		Message:   message,
	})
}

func (a *App) EnableStateEvents(enabled bool) {
	a.stateEvents = enabled
	if enabled {
		_ = a.emitState()
	}
}

func (a *App) EnableLogEvents(enabled bool) {
	a.logEvents = enabled
}

func (a *App) GetWebhookConfig() (WebhookConfig, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return WebhookConfig{}, errors.New("未初始化")
	}
	hook := ctx.GetWebhook()
	if hook == nil {
		return WebhookConfig{Items: []WebhookItem{}}, nil
	}
	items := make([]WebhookItem, 0, len(hook.Items))
	for _, it := range hook.Items {
		if it == nil {
			continue
		}
		items = append(items, WebhookItem{
			Description: it.Description,
			Type:        it.Type,
			URL:         it.URL,
			Talker:      it.Talker,
			Sender:      it.Sender,
			Keyword:     it.Keyword,
			Disabled:    it.Disabled,
		})
	}
	return WebhookConfig{Host: hook.Host, DelayMs: hook.DelayMs, Items: items}, nil
}

func (a *App) SetWebhookConfig(cfg WebhookConfig) error {
	ctx := a.mgr.Context()
	if ctx == nil {
		return errors.New("未初始化")
	}
	items := make([]*conf.WebhookItem, 0, len(cfg.Items))
	for _, it := range cfg.Items {
		url := strings.TrimSpace(it.URL)
		if url == "" {
			continue
		}
		t := strings.TrimSpace(it.Type)
		if t == "" {
			t = "message"
		}
		items = append(items, &conf.WebhookItem{
			Description: strings.TrimSpace(it.Description),
			Type:        t,
			URL:         url,
			Talker:      strings.TrimSpace(it.Talker),
			Sender:      strings.TrimSpace(it.Sender),
			Keyword:     strings.TrimSpace(it.Keyword),
			Disabled:    it.Disabled,
		})
	}

	host := strings.TrimSpace(cfg.Host)
	var hook *conf.Webhook
	if host == "" && cfg.DelayMs == 0 && len(items) == 0 {
		hook = nil
	} else {
		hook = &conf.Webhook{Host: host, DelayMs: cfg.DelayMs, Items: items}
	}

	if err := a.mgr.SetWebhook(hook); err != nil {
		return err
	}
	return nil
}

// AI Providers ----------------------------------------------------------------

type AIProvider struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	BaseURL   string `json:"baseUrl"`
	APIKey    string `json:"apiKey"`
	Model     string `json:"model"`
	Disabled  bool   `json:"disabled"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type AITestResult struct {
	OK       bool   `json:"ok"`
	Latency  int64  `json:"latencyMs"`
	Endpoint string `json:"endpoint"`
	Status   int    `json:"status"`
	Message  string `json:"message"`
}

func toAIProvider(p *conf.AIProvider) AIProvider {
	if p == nil {
		return AIProvider{}
	}
	return AIProvider{
		ID:        p.ID,
		Name:      p.Name,
		Type:      p.Type,
		BaseURL:   p.BaseURL,
		APIKey:    p.APIKey,
		Model:     p.Model,
		Disabled:  p.Disabled,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func fromAIProvider(p AIProvider) *conf.AIProvider {
	return &conf.AIProvider{
		ID:        strings.TrimSpace(p.ID),
		Name:      strings.TrimSpace(p.Name),
		Type:      strings.TrimSpace(p.Type),
		BaseURL:   strings.TrimSpace(p.BaseURL),
		APIKey:    strings.TrimSpace(p.APIKey),
		Model:     strings.TrimSpace(p.Model),
		Disabled:  p.Disabled,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func (a *App) ListAIProviders() ([]AIProvider, error) {
	if a.mgr.Context() == nil {
		return []AIProvider{}, errors.New("未初始化")
	}
	src := a.mgr.GetAIProviders()
	out := make([]AIProvider, 0, len(src))
	for _, p := range src {
		if p == nil {
			continue
		}
		out = append(out, toAIProvider(p))
	}
	return out, nil
}

func (a *App) SaveAIProvider(p AIProvider) (AIProvider, error) {
	if a.mgr.Context() == nil {
		return AIProvider{}, errors.New("未初始化")
	}
	if strings.TrimSpace(p.Name) == "" {
		return AIProvider{}, errors.New("名称不能为空")
	}
	if strings.TrimSpace(p.Type) == "" {
		return AIProvider{}, errors.New("提供商类型不能为空")
	}
	now := time.Now().UnixMilli()

	current := a.mgr.GetAIProviders()
	target := fromAIProvider(p)

	if target.ID == "" {
		target.ID = generateAIID()
		target.CreatedAt = now
		target.UpdatedAt = now
		current = append(current, target)
	} else {
		found := false
		for i, item := range current {
			if item != nil && item.ID == target.ID {
				if target.CreatedAt == 0 {
					target.CreatedAt = item.CreatedAt
				}
				target.UpdatedAt = now
				current[i] = target
				found = true
				break
			}
		}
		if !found {
			target.CreatedAt = now
			target.UpdatedAt = now
			current = append(current, target)
		}
	}
	if err := a.mgr.SetAIProviders(current); err != nil {
		return AIProvider{}, err
	}
	return toAIProvider(target), nil
}

func (a *App) DeleteAIProvider(id string) error {
	if a.mgr.Context() == nil {
		return errors.New("未初始化")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("id 不能为空")
	}
	current := a.mgr.GetAIProviders()
	next := make([]*conf.AIProvider, 0, len(current))
	for _, p := range current {
		if p == nil || p.ID == id {
			continue
		}
		next = append(next, p)
	}
	return a.mgr.SetAIProviders(next)
}

func (a *App) TestAIProvider(p AIProvider) AITestResult {
	provider := fromAIProvider(p)
	svc := ai.New()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r := svc.TestProvider(ctx, provider)
	return AITestResult{
		OK:       r.OK,
		Latency:  r.Latency,
		Endpoint: r.Endpoint,
		Status:   r.Status,
		Message:  r.Message,
	}
}

func (a *App) ListAIModels(p AIProvider) ([]string, error) {
	provider := fromAIProvider(p)
	svc := ai.New()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return svc.ListModels(ctx, provider)
}

func generateAIID() string {
	return fmt.Sprintf("ai-%d", time.Now().UnixNano())
}

func (a *App) GetState() (State, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return State{}, errors.New("未初始化")
	}
	snap := ctx.Snapshot()
	last := ""
	if snap.LastSessionUnix > 1000000000 {
		last = time.Unix(snap.LastSessionUnix, 0).Format("2006-01-02 15:04:05")
	}
	return State{
		Account:         snap.Account,
		Platform:        snap.Platform,
		FullVersion:     snap.FullVersion,
		DataDir:         snap.DataDir,
		DataKey:         snap.DataKey,
		ImgKey:          snap.ImgKey,
		WorkDir:         snap.WorkDir,
		HTTPEnabled:     snap.HTTPEnabled,
		HTTPAddr:        snap.HTTPAddr,
		AutoDecrypt:     snap.AutoDecrypt,
		LastSession:     last,
		PID:             snap.PID,
		ExePath:         snap.ExePath,
		Status:          snap.Status,
		Nickname:        snap.Nickname,
		SmallHeadImgUrl: snap.SmallHeadImgUrl,
	}, nil
}

func (a *App) ListInstances() ([]Instance, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return nil, errors.New("未初始化")
	}
	instances := make([]Instance, 0)
	for _, inst := range ctx.Snapshot().WeChatInstances {
		instances = append(instances, Instance{
			Name:        inst.Name,
			PID:         inst.PID,
			Platform:    inst.Platform,
			FullVersion: inst.FullVersion,
			DataDir:     inst.DataDir,
			ExePath:     inst.ExePath,
			Status:      inst.Status,
		})
	}
	return instances, nil
}

func (a *App) Refresh() (State, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return State{}, errors.New("未初始化")
	}
	a.mgr.ReloadWeChatInstances()
	_ = a.mgr.RefreshSession()
	return a.GetState()
}

func (a *App) GetContacts(keyword string, isInChatRoom, limit, offset int) (*wechatdb.GetContactsResp, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return nil, errors.New("未初始化")
	}
	return a.mgr.GetContacts(strings.TrimSpace(keyword), isInChatRoom, limit, offset)
}

func (a *App) SwitchToPID(pid int) (State, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return State{}, errors.New("未初始化")
	}
	for _, inst := range ctx.Snapshot().WeChatInstances {
		if inst.PID == pid {
			if err := a.mgr.Switch(inst.Raw, ""); err != nil {
				return State{}, err
			}
			_ = a.emitState()
			return a.GetState()
		}
	}
	return State{}, fmt.Errorf("未找到 PID=%d 的微信进程", pid)
}

func (a *App) SwitchToHistory(account string) (State, error) {
	if err := a.mgr.Switch(nil, account); err != nil {
		return State{}, err
	}
	_ = a.emitState()
	return a.GetState()
}

func (a *App) GetDataKey() (string, error) {
	key, err := a.mgr.GetDataKeyWithProgress(func(message string) {
		a.emitKeyProgress("dataKey", message)
	})
	if err != nil {
		return "", err
	}
	_ = a.emitState()
	return key, nil
}

func (a *App) GetImgKey() (string, error) {
	key, err := a.mgr.GetImgKey()
	if err != nil {
		return "", err
	}
	_ = a.emitState()
	return key, nil
}

func (a *App) GetKeys() (map[string]string, error) {
	dataKey, imgKey, err := a.mgr.GetKeysWithProgress(func(message string) {
		a.emitKeyProgress("dataKey", message)
	})
	if err != nil {
		return nil, err
	}
	_ = a.emitState()
	return map[string]string{"dataKey": dataKey, "imgKey": imgKey}, nil
}

func (a *App) Decrypt() error {
	if err := a.mgr.DecryptDBFiles(); err != nil {
		return err
	}
	_ = a.emitState()
	return nil
}

// ListListenIPs 返回当前可用于监听的 IP 候选列表（不含端口）。
// 默认包含 127.0.0.1 与 0.0.0.0，并追加所有处于 UP 且非环回的 IPv4 地址。
func (a *App) ListListenIPs() ([]string, error) {
	ips := []string{"127.0.0.1", "0.0.0.0"}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips, nil
	}
	seen := map[string]struct{}{ips[0]: {}, ips[1]: {}}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		ipAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, raw := range ipAddrs {
			var ip net.IP
			switch v := raw.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			s := ip4.String()
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			ips = append(ips, s)
		}
	}
	return ips, nil
}

func (a *App) SetHTTPAddr(addr string) (State, error) {
	if err := a.mgr.SetHTTPAddr(addr); err != nil {
		return State{}, err
	}
	_ = a.emitState()
	return a.GetState()
}

func (a *App) SetWorkDir(dir string) (State, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return State{}, errors.New("未初始化")
	}
	ctx.SetWorkDir(dir)
	_ = a.emitState()
	return a.GetState()
}

func (a *App) SetDataDir(dir string) (State, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return State{}, errors.New("未初始化")
	}
	ctx.SetDataDir(dir)
	_ = a.emitState()
	return a.GetState()
}

func (a *App) SetDataKey(key string) (State, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return State{}, errors.New("未初始化")
	}
	ctx.SetDataKey(key)
	_ = a.emitState()
	return a.GetState()
}

func (a *App) SetImgKey(key string) (State, error) {
	ctx := a.mgr.Context()
	if ctx == nil {
		return State{}, errors.New("未初始化")
	}
	ctx.SetImgKey(key)
	_ = a.emitState()
	return a.GetState()
}

func (a *App) StartHTTP() error {
	if err := a.mgr.StartService(); err != nil {
		return err
	}
	_ = a.emitState()
	return nil
}

func (a *App) StopHTTP() error {
	if err := a.mgr.StopService(); err != nil {
		return err
	}
	_ = a.emitState()
	return nil
}

func (a *App) SetAutoDecrypt(enabled bool) error {
	if enabled {
		if err := a.mgr.StartAutoDecrypt(); err != nil {
			return err
		}
	} else {
		if err := a.mgr.StopAutoDecrypt(); err != nil {
			return err
		}
	}
	_ = a.emitState()
	return nil
}

func (a *App) GetLogPath() (string, error) {
	a.ensureFileLogging()
	if a.logPath != "" {
		return a.logPath, nil
	}
	path, err := a.ensureLogFile()
	if err != nil {
		return "", err
	}
	a.logPath = path
	return a.logPath, nil
}

func (a *App) ReadLogTail(maxLines int) (string, error) {
	path, err := a.GetLogPath()
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()

	if maxLines <= 0 {
		maxLines = 300
	}

	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	const maxRead = int64(512 * 1024)
	offset := st.Size() - maxRead
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, 0); err != nil {
		return "", err
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	text := string(b)
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return "", nil
	}
	start := len(lines) - maxLines
	if start < 0 {
		start = 0
	}
	out := make([]string, 0, len(lines)-start)
	for _, line := range lines[start:] {
		out = append(out, strings.TrimRight(line, "\r"))
	}
	return strings.Join(out, "\n"), nil
}

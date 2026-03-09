package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/chatlog"
	"github.com/sjzar/chatlog/internal/chatlog/conf"
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
}

func (a *App) resetLogger() {
	writers := []io.Writer{util.NewPlainLogWriter(os.Stderr, false)}
	if a.logFile != nil {
		writers = append(writers, util.NewPlainLogWriter(a.logFile, true))
	}
	writers = append(writers, &notifyingWriter{app: a})
	log.Logger = log.Output(io.MultiWriter(writers...))
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
	key, err := a.mgr.GetDataKey()
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
	dataKey, imgKey, err := a.mgr.GetKeys()
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

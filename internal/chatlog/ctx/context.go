package ctx

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/wechat"
	"github.com/sjzar/chatlog/pkg/config"
	"github.com/sjzar/chatlog/pkg/util"
)

const (
	DefalutHTTPAddr = "127.0.0.1:5030"
)

// Context is a context for a chatlog.
// It is used to store information about the chatlog.
type Context struct {
	conf *conf.AppConfig
	cm   *config.Manager
	mu   sync.RWMutex

	History map[string]conf.ProcessConfig

	// 微信账号相关状态
	Account  string
	Platform string

	FullVersion string
	DataDir     string
	DataKey     string
	DataUsage   string
	ImgKey      string

	// 工作目录相关状态
	WorkDir   string
	WorkUsage string

	// HTTP服务相关状态
	HTTPEnabled bool
	HTTPAddr    string

	// 自动解密
	AutoDecrypt bool
	LastSession time.Time

	// 当前选中的微信实例
	Current         *wechat.Account
	PID             int
	ExePath         string
	Status          string
	Nickname        string // 当前用户昵称/备注
	SmallHeadImgUrl string // 当前用户小头像URL

	// 所有可用的微信实例
	WeChatInstances []*wechat.Account
}

type InstanceSnapshot struct {
	Name        string
	PID         int
	Platform    string
	FullVersion string
	DataDir     string
	ExePath     string
	Status      string

	Raw *wechat.Account
}

type Snapshot struct {
	Account  string
	Platform string

	FullVersion string
	DataDir     string
	DataKey     string
	ImgKey      string

	WorkDir string

	HTTPEnabled bool
	HTTPAddr    string

	AutoDecrypt     bool
	LastSessionUnix int64

	PID             int
	ExePath         string
	Status          string
	Nickname        string // 当前用户昵称/备注
	SmallHeadImgUrl string // 当前用户小头像URL

	WeChatInstances []InstanceSnapshot
}

func New(configPath string) (*Context, error) {

	conf, tcm, err := conf.LoadAppConfig(configPath)
	if err != nil {
		return nil, err
	}

	ctx := &Context{
		conf: conf,
		cm:   tcm,
	}

	ctx.loadConfig()

	return ctx, nil
}

func (c *Context) loadConfig() {
	c.History = c.conf.ParseHistory()
	selected := c.conf.SelectedAccount()
	c.conf.CurrentAccount = selected
	if c.conf.LastAccount == "" {
		c.conf.LastAccount = selected
	}
	c.SwitchHistory(selected)
	c.Refresh()
}

func (c *Context) SwitchHistory(account string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Current = nil
	c.PID = 0
	c.ExePath = ""
	c.Status = ""
	history, ok := c.History[account]
	if ok {
		c.Account = history.Account
		c.Platform = history.Platform

		c.FullVersion = history.FullVersion
		c.DataKey = history.DataKey
		c.ImgKey = history.ImgKey
		c.DataDir = util.NormalizeDataDirPath(history.DataDir)
		c.WorkDir = history.WorkDir
		c.HTTPEnabled = history.HTTPEnabled
		c.HTTPAddr = history.HTTPAddr
		c.AutoDecrypt = history.AutoDecrypt
	} else {
		c.Account = ""
		c.Platform = ""

		c.FullVersion = ""
		c.DataKey = ""
		c.ImgKey = ""
		c.DataDir = ""
		c.WorkDir = ""
		c.HTTPEnabled = false
		c.HTTPAddr = ""
		c.AutoDecrypt = true
	}
}

func (c *Context) SwitchCurrent(info *wechat.Account) {
	c.SwitchHistory(info.Name)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Current = info
	c.Refresh()

}
func (c *Context) Refresh() {
	if c.Current != nil {
		c.Account = c.Current.Name
		c.Platform = c.Current.Platform

		c.FullVersion = c.Current.FullVersion
		c.PID = int(c.Current.PID)
		c.ExePath = c.Current.ExePath
		c.Status = c.Current.Status
		if c.Current.Key != "" && c.Current.Key != c.DataKey {
			c.DataKey = c.Current.Key
		}
		if c.Current.ImgKey != "" && c.Current.ImgKey != c.ImgKey {
			c.ImgKey = c.Current.ImgKey
		}
		if c.Current.DataDir != "" && c.Current.DataDir != c.DataDir {
			c.DataDir = util.NormalizeDataDirPath(c.Current.DataDir)
		}
	}
	if c.DataUsage == "" && c.DataDir != "" {
		go func() {
			c.DataUsage = util.GetDirSize(c.DataDir)
		}()
	}
	if c.WorkUsage == "" && c.WorkDir != "" {
		go func() {
			c.WorkUsage = util.GetDirSize(c.WorkDir)
		}()
	}
}

func (c *Context) Snapshot() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	instances := make([]InstanceSnapshot, 0, len(c.WeChatInstances))
	for _, inst := range c.WeChatInstances {
		if inst == nil {
			continue
		}
		instances = append(instances, InstanceSnapshot{
			Name:        inst.Name,
			PID:         int(inst.PID),
			Platform:    inst.Platform,
			FullVersion: inst.FullVersion,
			DataDir:     inst.DataDir,
			ExePath:     inst.ExePath,
			Status:      inst.Status,
			Raw:         inst,
		})
	}

	last := int64(0)
	if c.LastSession.Unix() > 0 {
		last = c.LastSession.Unix()
	}

	return Snapshot{
		Account:         c.Account,
		Platform:        c.Platform,
		FullVersion:     c.FullVersion,
		DataDir:         c.DataDir,
		DataKey:         c.DataKey,
		ImgKey:          c.ImgKey,
		WorkDir:         c.WorkDir,
		HTTPEnabled:     c.HTTPEnabled,
		HTTPAddr:        c.HTTPAddr,
		AutoDecrypt:     c.AutoDecrypt,
		LastSessionUnix: last,
		PID:             c.PID,
		ExePath:         c.ExePath,
		Status:          c.Status,
		Nickname:        c.Nickname,
		SmallHeadImgUrl: c.SmallHeadImgUrl,
		WeChatInstances: instances,
	}
}

func (c *Context) GetDataDir() string {
	return c.DataDir
}

func (c *Context) GetWorkDir() string {
	return c.WorkDir
}

func (c *Context) GetPlatform() string {
	return c.Platform
}

func (c *Context) GetDataKey() string {
	return c.DataKey
}

func (c *Context) GetHTTPAddr() string {
	if c.HTTPAddr == "" {
		c.HTTPAddr = DefalutHTTPAddr
	}
	return c.HTTPAddr
}

func (c *Context) GetWebhook() *conf.Webhook {
	return c.conf.Webhook
}

func (c *Context) SetWebhook(hook *conf.Webhook) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conf.Webhook = hook
	if err := c.cm.SetConfig("webhook", hook); err != nil {
		log.Error().Err(err).Msg("set webhook failed")
		return err
	}
	return nil
}

func (c *Context) GetDebug() bool {
	return c.conf.Debug
}

func (c *Context) SetHTTPEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.HTTPEnabled == enabled {
		return
	}
	c.HTTPEnabled = enabled
	c.UpdateConfig()
}

func (c *Context) SetHTTPAddr(addr string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.HTTPAddr == addr {
		return
	}
	c.HTTPAddr = addr
	c.UpdateConfig()
}

func (c *Context) SetWorkDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.WorkDir == dir {
		return
	}
	c.WorkDir = dir
	c.UpdateConfig()
	c.Refresh()
}

func (c *Context) SetDataDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.DataDir == dir {
		return
	}
	c.DataDir = util.NormalizeDataDirPath(dir)
	c.UpdateConfig()
	c.Refresh()
}

func (c *Context) SetImgKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ImgKey == key {
		return
	}
	c.ImgKey = key
	c.UpdateConfig()
}

func (c *Context) SetDataKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.DataKey == key {
		return
	}
	c.DataKey = key
	c.UpdateConfig()
}

func (c *Context) SetAutoDecrypt(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.AutoDecrypt == enabled {
		return
	}
	c.AutoDecrypt = enabled
	c.UpdateConfig()
}

// 更新配置
func (c *Context) UpdateConfig() {

	pconf := conf.ProcessConfig{
		Type:     "wechat",
		Account:  c.Account,
		Platform: c.Platform,

		FullVersion: c.FullVersion,
		DataDir:     c.DataDir,
		DataKey:     c.DataKey,
		ImgKey:      c.ImgKey,
		WorkDir:     c.WorkDir,
		HTTPEnabled: c.HTTPEnabled,
		HTTPAddr:    c.HTTPAddr,
		AutoDecrypt: c.AutoDecrypt,
	}

	if c.conf.History == nil {
		c.conf.History = make([]conf.ProcessConfig, 0)
	}
	if len(c.conf.History) == 0 {
		c.conf.History = append(c.conf.History, pconf)
	} else {
		isFind := false
		for i, v := range c.conf.History {
			if v.Account == c.Account {
				isFind = true
				c.conf.History[i] = pconf
				break
			}
		}
		if !isFind {
			c.conf.History = append(c.conf.History, pconf)
		}
	}

	c.conf.CurrentAccount = c.Account
	c.conf.LastAccount = c.Account

	if err := c.cm.SetConfig("current_account", c.Account); err != nil {
		log.Error().Err(err).Msg("set current_account failed")
		return
	}

	if err := c.cm.SetConfig("last_account", c.Account); err != nil {
		log.Error().Err(err).Msg("set last_account failed")
		return
	}

	if err := c.cm.SetConfig("history", c.conf.History); err != nil {
		log.Error().Err(err).Msg("set history failed")
		return
	}
}

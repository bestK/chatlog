package ctx

import (
	"sync"

	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/wechat"
	"github.com/sjzar/chatlog/pkg/util"
)

// Context is a context for a chatlog.
// It is used to store information about the chatlog.
type Context struct {
	conf *conf.Service
	mu   sync.RWMutex

	History map[string]conf.ProcessConfig

	// 微信账号相关状态
	Account      string
	Version      string
	MajorVersion int
	DataKey      string
	DataUsage    string
	DataDir      string

	// 工作目录相关状态
	WorkUsage string
	WorkDir   string

	// HTTP服务相关状态
	HTTPEnabled bool
	HTTPAddr    string

	// 当前选中的微信实例
	Current *wechat.Info
	PID     int
	ExePath string
	Status  string

	// 所有可用的微信实例
	WeChatInstances []*wechat.Info
}

func New(conf *conf.Service) *Context {
	ctx := &Context{
		conf: conf,
	}

	ctx.loadConfig()

	return ctx
}

func (c *Context) loadConfig() {
	conf := c.conf.GetConfig()
	c.History = conf.ParseHistory()
	c.SwitchHistory(conf.LastAccount)
	c.Refresh()
}

func (c *Context) SwitchHistory(account string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	history, ok := c.History[account]
	if ok {
		c.Account = history.Account
		c.Version = history.Version
		c.MajorVersion = history.MajorVersion
		c.DataKey = history.DataKey
		c.DataDir = history.DataDir
		c.WorkDir = history.WorkDir
		c.HTTPEnabled = history.HTTPEnabled
		c.HTTPAddr = history.HTTPAddr
	}
}

func (c *Context) SwitchCurrent(info *wechat.Info) {
	c.SwitchHistory(info.AccountName)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Current = info
	c.Refresh()

}
func (c *Context) Refresh() {
	if c.Current != nil {
		c.Account = c.Current.AccountName
		c.Version = c.Current.Version.FileVersion
		c.MajorVersion = c.Current.Version.FileMajorVersion
		c.PID = int(c.Current.PID)
		c.ExePath = c.Current.ExePath
		c.Status = c.Current.Status
		if c.Current.Key != "" && c.Current.Key != c.DataKey {
			c.DataKey = c.Current.Key
		}
		if c.Current.DataDir != "" && c.Current.DataDir != c.DataDir {
			c.DataDir = c.Current.DataDir
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

func (c *Context) SetHTTPEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.HTTPEnabled = enabled
	c.UpdateConfig()
}

func (c *Context) SetHTTPAddr(addr string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.HTTPAddr = addr
	c.UpdateConfig()
}

func (c *Context) SetWorkDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.WorkDir = dir
	c.UpdateConfig()
	c.Refresh()
}

func (c *Context) SetDataDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DataDir = dir
	c.UpdateConfig()
	c.Refresh()
}

// 更新配置
func (c *Context) UpdateConfig() {
	pconf := conf.ProcessConfig{
		Type:         "wechat",
		Version:      c.Version,
		MajorVersion: c.MajorVersion,
		Account:      c.Account,
		DataKey:      c.DataKey,
		DataDir:      c.DataDir,
		WorkDir:      c.WorkDir,
		HTTPEnabled:  c.HTTPEnabled,
		HTTPAddr:     c.HTTPAddr,
	}
	conf := c.conf.GetConfig()
	conf.UpdateHistory(c.Account, pconf)
}

package conf

import (
	"strings"

	"github.com/sjzar/chatlog/pkg/util"
)

const (
	DefalutHTTPAddr = "0.0.0.0:5030"
)

type ServerConfig struct {
	CurrentAccount string          `mapstructure:"current_account"`
	LastAccount    string          `mapstructure:"last_account"`
	History        []ProcessConfig `mapstructure:"history"`

	Type     string `mapstructure:"type"`
	Platform string `mapstructure:"platform"`

	FullVersion string   `mapstructure:"full_version"`
	DataDir     string   `mapstructure:"data_dir"`
	DataKey     string   `mapstructure:"data_key"`
	ImgKey      string   `mapstructure:"img_key"`
	WorkDir     string   `mapstructure:"work_dir"`
	HTTPAddr    string   `mapstructure:"http_addr"`
	AutoDecrypt bool     `mapstructure:"auto_decrypt"`
	Debug       bool     `mapstructure:"debug"`
	Webhook     *Webhook `mapstructure:"webhook"`
}

type ServerOverrides struct {
	Type        *string
	Platform    *string
	FullVersion *string
	DataDir     *string
	DataKey     *string
	ImgKey      *string
	WorkDir     *string
	HTTPAddr    *string
	AutoDecrypt *bool
	Debug       *bool
}

func StringOverride(value string) *string {
	return &value
}

func BoolOverride(value bool) *bool {
	return &value
}

func (o ServerOverrides) ApplyTo(c *ServerConfig) {
	if o.Type != nil {
		c.Type = *o.Type
	}
	if o.Platform != nil {
		c.Platform = *o.Platform
	}
	if o.FullVersion != nil {
		c.FullVersion = *o.FullVersion
	}
	if o.DataDir != nil {
		c.DataDir = *o.DataDir
	}
	if o.DataKey != nil {
		c.DataKey = *o.DataKey
	}
	if o.ImgKey != nil {
		c.ImgKey = *o.ImgKey
	}
	if o.WorkDir != nil {
		c.WorkDir = *o.WorkDir
	}
	if o.HTTPAddr != nil {
		c.HTTPAddr = *o.HTTPAddr
	}
	if o.AutoDecrypt != nil {
		c.AutoDecrypt = *o.AutoDecrypt
	}
	if o.Debug != nil {
		c.Debug = *o.Debug
	}
}

func (o ServerOverrides) IsTypeSet() bool        { return o.Type != nil }
func (o ServerOverrides) IsPlatformSet() bool    { return o.Platform != nil }
func (o ServerOverrides) IsFullVersionSet() bool { return o.FullVersion != nil }
func (o ServerOverrides) IsDataDirSet() bool     { return o.DataDir != nil }
func (o ServerOverrides) IsDataKeySet() bool     { return o.DataKey != nil }
func (o ServerOverrides) IsImgKeySet() bool      { return o.ImgKey != nil }
func (o ServerOverrides) IsWorkDirSet() bool     { return o.WorkDir != nil }
func (o ServerOverrides) IsHTTPAddrSet() bool    { return o.HTTPAddr != nil }

var ServerDefaults = map[string]any{
	"auto_decrypt": true,
	"webhook": &Webhook{
		Host: "localhost:5030",
		Items: []*WebhookItem{
			{
				Keyword: "",
				Sender:  "momo",
				Talker:  "测试群",
				URL:     "http://127.0.0.1:3000/api/v1/webhook",
			},
		},
	},
}

func (c *ServerConfig) ApplyHistoryDefaults(overrides ServerOverrides) {
	history := c.matchHistory()
	if history == nil {
		return
	}
	if c.Type == "" && !overrides.IsTypeSet() {
		c.Type = history.Type
	}
	if c.Platform == "" && !overrides.IsPlatformSet() {
		c.Platform = history.Platform
	}
	if c.FullVersion == "" && !overrides.IsFullVersionSet() {
		c.FullVersion = history.FullVersion
	}
	if c.DataDir == "" && !overrides.IsDataDirSet() {
		c.DataDir = history.DataDir
	}
	if c.DataKey == "" && !overrides.IsDataKeySet() {
		c.DataKey = history.DataKey
	}
	if c.ImgKey == "" && !overrides.IsImgKeySet() {
		c.ImgKey = history.ImgKey
	}
	if c.WorkDir == "" && !overrides.IsWorkDirSet() {
		c.WorkDir = history.WorkDir
	}
	if c.HTTPAddr == "" && !overrides.IsHTTPAddrSet() {
		c.HTTPAddr = history.HTTPAddr
	}
}

func (c *ServerConfig) matchHistory() *ProcessConfig {
	if len(c.History) == 0 {
		return nil
	}
	if c.DataDir != "" {
		dataDir := util.NormalizeDataDirPath(c.DataDir)
		for i := range c.History {
			if util.NormalizeDataDirPath(c.History[i].DataDir) == dataDir {
				return &c.History[i]
			}
		}
	}
	for _, account := range []string{c.CurrentAccount, c.LastAccount} {
		account = strings.TrimSpace(account)
		if account == "" {
			continue
		}
		for i := range c.History {
			if c.History[i].Account == account {
				return &c.History[i]
			}
		}
	}
	return nil
}

func (c *ServerConfig) GetDataDir() string {
	c.DataDir = util.NormalizeDataDirPath(c.DataDir)
	return c.DataDir
}

func (c *ServerConfig) GetWorkDir() string {
	return c.WorkDir
}

func (c *ServerConfig) GetPlatform() string {
	return c.Platform
}

func (c *ServerConfig) GetDataKey() string {
	return c.DataKey
}

func (c *ServerConfig) GetImgKey() string {
	return c.ImgKey
}

func (c *ServerConfig) GetAutoDecrypt() bool {
	return c.AutoDecrypt
}

func (c *ServerConfig) GetHTTPAddr() string {
	if c.HTTPAddr == "" {
		c.HTTPAddr = DefalutHTTPAddr
	}
	return c.HTTPAddr
}

func (c *ServerConfig) GetWebhook() *Webhook {
	return c.Webhook
}

func (c *ServerConfig) GetDebug() bool {
	return c.Debug
}

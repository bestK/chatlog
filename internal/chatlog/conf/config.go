package conf

import "github.com/sjzar/chatlog/pkg/config"

type Config struct {
	ConfigDir   string          `mapstructure:"-"`
	LastAccount string          `mapstructure:"last_account" json:"last_account"`
	History     []ProcessConfig `mapstructure:"history" json:"history"`
}

type ProcessConfig struct {
	Type         string `mapstructure:"type" json:"type"`
	Version      string `mapstructure:"version" json:"version"`
	MajorVersion int    `mapstructure:"major_version" json:"major_version"`
	Account      string `mapstructure:"account" json:"account"`
	DataKey      string `mapstructure:"data_key" json:"data_key"`
	DataDir      string `mapstructure:"data_dir" json:"data_dir"`
	WorkDir      string `mapstructure:"work_dir" json:"work_dir"`
	HTTPEnabled  bool   `mapstructure:"http_enabled" json:"http_enabled"`
	HTTPAddr     string `mapstructure:"http_addr" json:"http_addr"`
	LastTime     int64  `mapstructure:"last_time" json:"last_time"`
	Files        []File `mapstructure:"files" json:"files"`
}

type File struct {
	Path         string `mapstructure:"path" json:"path"`
	ModifiedTime int64  `mapstructure:"modified_time" json:"modified_time"`
	Size         int64  `mapstructure:"size" json:"size"`
}

func (c *Config) ParseHistory() map[string]ProcessConfig {
	m := make(map[string]ProcessConfig)
	for _, v := range c.History {
		m[v.Account] = v
	}
	return m
}

func (c *Config) UpdateHistory(account string, conf ProcessConfig) error {
	if c.History == nil {
		c.History = make([]ProcessConfig, 0)
	}
	if len(c.History) == 0 {
		c.History = append(c.History, conf)
	} else {
		isFind := false
		for i, v := range c.History {
			if v.Account == account {
				isFind = true
				c.History[i] = conf
				break
			}
		}
		if !isFind {
			c.History = append(c.History, conf)
		}
	}
	config.SetConfig("last_account", account)
	return config.SetConfig("history", c.History)
}

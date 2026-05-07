package conf

// AIProvider 描述一个 AI 服务提供商配置。
type AIProvider struct {
	ID        string `mapstructure:"id" json:"id"`
	Name      string `mapstructure:"name" json:"name"`
	Type      string `mapstructure:"type" json:"type"`
	BaseURL   string `mapstructure:"base_url" json:"base_url,omitempty"`
	APIKey    string `mapstructure:"api_key" json:"api_key"`
	Model     string `mapstructure:"model" json:"model,omitempty"`
	Disabled  bool   `mapstructure:"disabled" json:"disabled,omitempty"`
	CreatedAt int64  `mapstructure:"created_at" json:"created_at,omitempty"`
	UpdatedAt int64  `mapstructure:"updated_at" json:"updated_at,omitempty"`
}

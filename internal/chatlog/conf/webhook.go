package conf

type Webhook struct {
	Host    string         `mapstructure:"host" json:"host"`
	DelayMs int64          `mapstructure:"delay_ms" json:"delay_ms"`
	Items   []*WebhookItem `mapstructure:"items" json:"items"`
}

type WebhookItem struct {
	Description string `mapstructure:"description" json:"description"`
	Type        string `mapstructure:"type" json:"type"`
	URL         string `mapstructure:"url" json:"url"`
	Talker      string `mapstructure:"talker" json:"talker"`
	Sender      string `mapstructure:"sender" json:"sender"`
	Keyword     string `mapstructure:"keyword" json:"keyword"`
	Disabled    bool   `mapstructure:"disabled" json:"disabled"`
}

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb"
)

type Config interface {
	GetWebhook() *conf.Webhook
}

type Webhook interface {
	Do(event fsnotify.Event)
}

type Service struct {
	config *conf.Webhook
	hooks  map[string][]*conf.WebhookItem
}

var processedMessages = newProcessedMessageStore()

func New(config Config) *Service {
	s := &Service{
		config: config.GetWebhook(),
	}

	if s.config == nil {
		return s
	}

	hooks := make(map[string][]*conf.WebhookItem)
	seen := make(map[string]struct{})
	for _, item := range s.config.Items {
		if item.Disabled {
			continue
		}
		if item.Type == "" {
			item.Type = "message"
		}
		signature := webhookItemSignature(item)
		if _, ok := seen[signature]; ok {
			log.Warn().Msgf("skip duplicated webhook item: %s", signature)
			continue
		}
		seen[signature] = struct{}{}
		switch item.Type {
		case "message":
			if hooks["message"] == nil {
				hooks["message"] = make([]*conf.WebhookItem, 0)
			}
			hooks["message"] = append(hooks["message"], item)
		default:
			log.Error().Msgf("unknown webhook type: %s", item.Type)
		}
	}
	s.hooks = hooks

	return s
}

func (s *Service) GetConfig() *conf.Webhook {
	return s.config
}

func (s *Service) GetHooks(ctx context.Context, db *wechatdb.DB) []*Group {

	if len(s.hooks) == 0 {
		return nil
	}

	groups := make([]*Group, 0)
	for group, items := range s.hooks {
		hooks := make([]Webhook, 0)
		for _, item := range items {
			hooks = append(hooks, NewMessageWebhook(item, db, s.config.Host))
		}
		groups = append(groups, NewGroup(ctx, group, hooks, s.config.DelayMs))
	}

	return groups
}

type Group struct {
	ctx     context.Context
	group   string
	hooks   []Webhook
	delayMs int64
	ch      chan fsnotify.Event
	mu      sync.Mutex
	lastHit map[string]time.Time
}

const eventDebounceWindow = 500 * time.Millisecond

func NewGroup(ctx context.Context, group string, hooks []Webhook, delayMs int64) *Group {
	g := &Group{
		group:   group,
		hooks:   hooks,
		delayMs: delayMs,
		ctx:     ctx,
		ch:      make(chan fsnotify.Event, 1),
		lastHit: make(map[string]time.Time),
	}
	go g.loop()
	return g
}

func (g *Group) Callback(event fsnotify.Event) error {
	// skip remove event
	if !(event.Op.Has(fsnotify.Create) || event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Rename)) {
		return nil
	}
	if g.shouldSkipEvent(event) {
		return nil
	}

	select {
	case g.ch <- event:
	default:
	}
	return nil
}

func (g *Group) shouldSkipEvent(event fsnotify.Event) bool {
	if event.Name == "" {
		return false
	}
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	for path, ts := range g.lastHit {
		if now.Sub(ts) > eventDebounceWindow {
			delete(g.lastHit, path)
		}
	}
	last, ok := g.lastHit[event.Name]
	g.lastHit[event.Name] = now
	return ok && now.Sub(last) <= eventDebounceWindow
}

func (g *Group) Group() string {
	return g.group
}

func (g *Group) loop() {
	for {
		select {
		case event, ok := <-g.ch:
			if !ok {
				return
			}
			if g.delayMs > 0 {
				time.Sleep(time.Duration(g.delayMs) * time.Millisecond)
				// 延迟后清空 channel 中的其他事件，避免重复触发
				for {
					select {
					case <-g.ch:
						// 丢弃延迟期间积累的事件
					default:
						goto done
					}
				}
			done:
			}
			g.do(event)
		case <-g.ctx.Done():
			return
		}
	}
}

func (g *Group) do(event fsnotify.Event) {
	for _, hook := range g.hooks {
		hook.Do(event)
	}
}

type MessageWebhook struct {
	host     string
	conf     *conf.WebhookItem
	client   *http.Client
	db       *wechatdb.DB
	lastTime time.Time
	lastSeq  int64
	mu       sync.Mutex
}

func NewMessageWebhook(conf *conf.WebhookItem, db *wechatdb.DB, host string) *MessageWebhook {
	m := &MessageWebhook{
		host:     host,
		conf:     conf,
		client:   &http.Client{Timeout: time.Second * 10},
		db:       db,
		lastTime: time.Now(),
	}
	return m
}

func (m *MessageWebhook) Do(event fsnotify.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	messages, err := m.db.GetMessages(m.lastTime, time.Now().Add(time.Minute*10), m.conf.Talker, m.conf.Sender, m.conf.Keyword, 0, 0)
	if err != nil {
		log.Error().Err(err).Msgf("webhook get messages failed")
		return
	}

	if len(messages.Items) == 0 {
		return
	}

	filtered := make([]*model.Message, 0, len(messages.Items))
	for _, message := range messages.Items {
		if message == nil {
			continue
		}
		if m.lastSeq != 0 && message.Seq <= m.lastSeq {
			continue
		}
		if processedMessages.Seen(m.conf, message.Seq) {
			continue
		}
		filtered = append(filtered, message)
	}

	if len(filtered) == 0 {
		return
	}

	m.lastTime = filtered[len(filtered)-1].Time.Add(time.Second).Time()
	m.lastSeq = filtered[len(filtered)-1].Seq

	for _, message := range filtered {
		message.SetContent("host", m.host)
		message.Content = message.PlainTextContent()
		log.Info().Msgf(
			"📨 receive message: talker=%s sender=%s content=%s",
			displayName(message.TalkerName, message.Talker),
			displayName(message.SenderName, message.Sender),
			message.Content,
		)
	}

	actualTalker := uniqueJoined(filtered, func(message *model.Message) string {
		if message.TalkerName != "" {
			return message.TalkerName
		}
		return message.Talker
	})
	actualSender := uniqueJoined(filtered, func(message *model.Message) string {
		if message.SenderName != "" {
			return message.SenderName
		}
		return message.Sender
	})

	ret := map[string]any{
		"talker":         actualTalker,
		"sender":         actualSender,
		"keyword":        m.conf.Keyword,
		"filter_talker":  m.conf.Talker,
		"filter_sender":  m.conf.Sender,
		"filter_keyword": m.conf.Keyword,
		"lastTime":       m.lastTime.Format(time.DateTime),
		"length":         len(filtered),
		"messages":       filtered,
	}
	body, _ := json.Marshal(ret)
	req, _ := http.NewRequest("POST", m.conf.URL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	log.Info().Msgf("🚀 post messages to %s, body: %s", m.conf.URL, string(body))
	resp, err := m.client.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("post messages failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("post messages failed, status code: %d", resp.StatusCode)
	}
}

func uniqueJoined(messages []*model.Message, selector func(*model.Message) string) string {
	seen := make(map[string]struct{})
	ordered := make([]string, 0)
	for _, message := range messages {
		value := selector(message)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ordered = append(ordered, value)
	}
	if len(ordered) == 0 {
		return ""
	}
	if len(ordered) == 1 {
		return ordered[0]
	}
	result := ordered[0]
	for i := 1; i < len(ordered); i++ {
		result += "," + ordered[i]
	}
	return result
}

func displayName(preferred string, fallback string) string {
	if preferred != "" {
		return preferred
	}
	return fallback
}

func webhookItemSignature(item *conf.WebhookItem) string {
	if item == nil {
		return ""
	}
	return fmt.Sprintf(
		"%s|%s|%s|%s|%s",
		item.Type,
		item.URL,
		item.Talker,
		item.Sender,
		item.Keyword,
	)
}

type processedMessageStore struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func newProcessedMessageStore() *processedMessageStore {
	return &processedMessageStore{items: make(map[string]time.Time)}
}

func (s *processedMessageStore) Seen(item *conf.WebhookItem, seq int64) bool {
	if item == nil || seq == 0 {
		return false
	}
	now := time.Now()
	key := fmt.Sprintf("%s#%d", webhookItemSignature(item), seq)

	s.mu.Lock()
	defer s.mu.Unlock()
	for existingKey, expiry := range s.items {
		if now.After(expiry) {
			delete(s.items, existingKey)
		}
	}
	if expiry, ok := s.items[key]; ok && now.Before(expiry) {
		return true
	}
	s.items[key] = now.Add(5 * time.Minute)
	return false
}

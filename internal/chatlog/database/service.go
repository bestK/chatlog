package database

import (
	"context"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/chatlog/messageview"
	"github.com/sjzar/chatlog/internal/chatlog/webhook"
	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb"
)

const (
	StateInit = iota
	StateDecrypting
	StateReady
	StateError
)

type Service struct {
	State         int
	StateMsg      string
	conf          Config
	db            *wechatdb.DB
	webhook       *webhook.Service
	webhookCancel context.CancelFunc
	webhookRegs   []webhookRegistration
	messageLogCb  func(event fsnotify.Event) error
	sessionLogMu  sync.Mutex
	sessionLogSeq map[string]int
}

type webhookRegistration struct {
	group    string
	callback func(event fsnotify.Event) error
}

type Config interface {
	GetWorkDir() string
	GetPlatform() string
	GetWebhook() *conf.Webhook
}

func NewService(conf Config) *Service {
	return &Service{
		conf:          conf,
		webhook:       webhook.New(conf),
		sessionLogSeq: make(map[string]int),
	}
}

func (s *Service) Start() error {
	db, err := wechatdb.New(s.conf.GetWorkDir(), s.conf.GetPlatform())
	if err != nil {
		return err
	}
	s.SetReady()
	s.db = db
	s.initMessageObserver()
	s.initWebhook()
	return nil
}

func (s *Service) Stop() error {
	s.clearMessageObserver()
	s.clearWebhookCallbacks()
	if s.db != nil {
		s.db.Close()
	}
	s.SetInit()
	s.db = nil
	if s.webhookCancel != nil {
		s.webhookCancel()
		s.webhookCancel = nil
	}
	return nil
}

func (s *Service) SetInit() {
	s.State = StateInit
}

func (s *Service) SetDecrypting() {
	s.State = StateDecrypting
}

func (s *Service) SetReady() {
	s.State = StateReady
}

func (s *Service) SetError(msg string) {
	s.State = StateError
	s.StateMsg = msg
}

func (s *Service) GetDB() *wechatdb.DB {
	return s.db
}

func (s *Service) GetMessages(start, end time.Time, talker string, sender string, keyword string, limit, offset int) (*wechatdb.GetMessagesResp, error) {
	return s.db.GetMessages(start, end, talker, sender, keyword, limit, offset)
}

func (s *Service) GetContacts(key string, limit, offset int) (*wechatdb.GetContactsResp, error) {
	return s.db.GetContacts(key, limit, offset)
}

func (s *Service) GetChatRooms(key string, limit, offset int) (*wechatdb.GetChatRoomsResp, error) {
	return s.db.GetChatRooms(key, limit, offset)
}

// GetSession retrieves session information
func (s *Service) GetSessions(key string, limit, offset int) (*wechatdb.GetSessionsResp, error) {
	return s.db.GetSessions(key, limit, offset)
}

func (s *Service) GetMedia(_type string, key string) (*model.Media, error) {
	return s.db.GetMedia(_type, key)
}

func (s *Service) initWebhook() error {
	if s.webhook == nil {
		return nil
	}
	s.clearWebhookCallbacks()

	// 输出详细的 webhook 配置
	config := s.webhook.GetConfig()
	if config != nil {
		log.Info().Msgf("webhook config: host=%s, delay_ms=%d, items=%d",
			config.Host, config.DelayMs, len(config.Items))
		for i, item := range config.Items {
			log.Info().Msgf("  item[%d]: type=%s, url=%s, talker=%s, sender=%s, keyword=%s, disabled=%v",
				i, item.Type, item.URL, item.Talker, item.Sender, item.Keyword, item.Disabled)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.webhookCancel = cancel
	hooks := s.webhook.GetHooks(ctx, s.db)
	log.Info().Msgf("webhook: %d hooks registered", len(hooks))
	s.webhookRegs = make([]webhookRegistration, 0, len(hooks))
	for _, hook := range hooks {
		log.Info().Msgf("set callback for group: %v", hook.Group())
		cb := hook.Callback
		if err := s.db.SetCallback(hook.Group(), cb); err != nil {
			log.Error().Err(err).Msgf("set callback %#v failed", hook)
			return err
		}
		s.webhookRegs = append(s.webhookRegs, webhookRegistration{group: hook.Group(), callback: cb})
	}
	return nil
}

func (s *Service) ReloadWebhook() error {
	s.clearWebhookCallbacks()
	if s.webhookCancel != nil {
		s.webhookCancel()
		s.webhookCancel = nil
	}
	s.webhook = webhook.New(s.conf)
	if s.db == nil {
		return nil
	}
	return s.initWebhook()
}

// Close closes the database connection
func (s *Service) Close() {
	// Add cleanup code if needed
	s.clearMessageObserver()
	s.clearWebhookCallbacks()
	if s.db != nil {
		s.db.Close()
	}
	if s.webhookCancel != nil {
		s.webhookCancel()
		s.webhookCancel = nil
	}
}

// CloseDB closes a specific database file connection
func (s *Service) CloseDB(path string) error {
	if s.db != nil {
		return s.db.CloseDB(path)
	}
	return nil
}

// LockDB locks a specific database file, preventing new connections
func (s *Service) LockDB(path string) error {
	if s.db != nil {
		return s.db.LockDB(path)
	}
	return nil
}

// UnlockDB unlocks a specific database file
func (s *Service) UnlockDB(path string) error {
	if s.db != nil {
		return s.db.UnlockDB(path)
	}
	return nil
}
func (s *Service) GetSelfSmallHeadImgUrl() string {
	if s.db != nil {
		return s.db.GetSelfSmallHeadImgUrl()
	}
	return ""
}

func (s *Service) GetSelfName() string {
	if s.db != nil {
		return s.db.GetSelfName()
	}
	return ""
}

func (s *Service) clearWebhookCallbacks() {
	if s.db == nil || len(s.webhookRegs) == 0 {
		s.webhookRegs = nil
		return
	}
	for _, reg := range s.webhookRegs {
		s.db.RemoveCallback(reg.group, reg.callback)
	}
	s.webhookRegs = nil
}

func (s *Service) initMessageObserver() {
	if s.db == nil {
		return
	}
	s.seedSessionLogState()
	if s.messageLogCb != nil {
		s.db.RemoveCallback("message", s.messageLogCb)
	}
	s.messageLogCb = s.onMessageEvent
	if err := s.db.SetCallback("message", s.messageLogCb); err != nil {
		log.Error().Err(err).Msg("set message observer callback failed")
	}
}

func (s *Service) clearMessageObserver() {
	if s.db != nil && s.messageLogCb != nil {
		s.db.RemoveCallback("message", s.messageLogCb)
	}
	s.messageLogCb = nil
	s.sessionLogMu.Lock()
	s.sessionLogSeq = make(map[string]int)
	s.sessionLogMu.Unlock()
}

func (s *Service) seedSessionLogState() {
	resp, err := s.db.GetSessions("", 50, 0)
	if err != nil {
		return
	}
	state := make(map[string]int, len(resp.Items))
	for _, session := range resp.Items {
		if session == nil {
			continue
		}
		state[session.TopicID] = sessionKey(session)
	}
	s.sessionLogMu.Lock()
	s.sessionLogSeq = state
	s.sessionLogMu.Unlock()
}

func (s *Service) onMessageEvent(event fsnotify.Event) error {
	if !(event.Op.Has(fsnotify.Create) || event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Rename)) {
		return nil
	}
	resp, err := s.db.GetSessions("", 20, 0)
	if err != nil {
		return nil
	}
	s.sessionLogMu.Lock()
	defer s.sessionLogMu.Unlock()
	for _, session := range resp.Items {
		if session == nil || session.TopicID == "" {
			continue
		}
		current := sessionKey(session)
		previous := s.sessionLogSeq[session.TopicID]
		if current <= previous {
			continue
		}
		s.sessionLogSeq[session.TopicID] = current
		log.Info().Msgf(
			"📨 receive message: talker=%s sender=%s content=%s",
			messageview.SessionTalkerName(session),
			messageview.SessionSenderName(session),
			session.Content,
		)
	}
	return nil
}

func sessionKey(session *model.Session) int {
	if session == nil {
		return 0
	}
	if session.LastMsgLocalID > 0 {
		return session.LastMsgLocalID
	}
	return session.NOrder
}

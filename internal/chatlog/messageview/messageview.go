package messageview

import (
	"sync"
	"time"

	"github.com/sjzar/chatlog/internal/model"
)

func PreferName(preferred string, fallback string) string {
	if preferred != "" {
		return preferred
	}
	return fallback
}

func TalkerName(message *model.Message) string {
	if message == nil {
		return ""
	}
	if message.TalkerName != "" {
		return message.TalkerName
	}
	if !message.IsChatRoom {
		return PreferName(message.SenderName, message.Sender)
	}
	return message.Talker
}

func SenderName(message *model.Message) string {
	if message == nil {
		return ""
	}
	return PreferName(message.SenderName, message.Sender)
}

func SessionTalkerName(session *model.Session) string {
	if session == nil {
		return ""
	}
	return PreferName(session.TopicName, session.TopicID)
}

func SessionSenderName(session *model.Session) string {
	if session == nil {
		return ""
	}
	return PreferName(session.PersonName, session.PersonID)
}

type DedupStore struct {
	mu    sync.Mutex
	items map[string]time.Time
	ttl   time.Duration
}

func NewDedupStore(ttl time.Duration) *DedupStore {
	return &DedupStore{items: make(map[string]time.Time), ttl: ttl}
}

func (s *DedupStore) Seen(key string) bool {
	if key == "" {
		return false
	}
	now := time.Now()
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
	s.items[key] = now.Add(s.ttl)
	return false
}

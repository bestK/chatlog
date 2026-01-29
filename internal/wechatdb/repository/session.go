package repository

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/model"
)

func (r *Repository) GetSessions(ctx context.Context, key string, limit, offset int) ([]*model.Session, error) {
	sessions, err := r.ds.GetSessions(ctx, key, limit, offset)
	if err != nil {
		return nil, err
	}

	if err := r.EnrichSessions(ctx, sessions); err != nil {
		log.Debug().Msgf("EnrichSessions failed: %v", err)
	}

	return sessions, nil
}

// EnrichSessions 补充会话的额外信息
func (r *Repository) EnrichSessions(ctx context.Context, sessions []*model.Session) error {
	for _, s := range sessions {
		if s.GroupID != "" {
			// 群聊
			// 获取更准确的群名（如果有）
			if chatRoom, ok := r.chatRoomCache[s.GroupID]; ok {
				s.GroupName = chatRoom.DisplayName()
			} else {
				contact := r.getFullContact(s.GroupID)
				if contact != nil {
					s.GroupName = contact.DisplayName()
				}
			}

			// 解析最近发言人 ID
			if idx := strings.Index(s.Content, ":"); idx != -1 {
				senderID := s.Content[:idx]
				// 排除一些明显不是 ID 的情况（如包含空格或长度过长）
				if len(senderID) > 0 && len(senderID) < 64 && !strings.ContainsAny(senderID, " \t\n\r") {
					s.PersonID = senderID
					// 补充发言人名称
					if chatRoom, ok := r.chatRoomCache[s.GroupID]; ok {
						if displayName, ok := chatRoom.User2DisplayName[senderID]; ok {
							s.PersonName = displayName
						}
					}
					if s.PersonName == "" {
						contact := r.getFullContact(senderID)
						if contact != nil {
							s.PersonName = contact.DisplayName()
						}
					}
				}
			}
		}

		// 统一处理 PersonName 补全逻辑（针对单聊或群聊解析出的 ID）
		if s.PersonID != "" && s.PersonName == "" {
			contact := r.getFullContact(s.PersonID)
			if contact != nil {
				s.PersonName = contact.DisplayName()
			}
		}
	}
	return nil
}

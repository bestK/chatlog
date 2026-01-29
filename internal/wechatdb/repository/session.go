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
		// 强制要求：如果发送人 ID 为空，默认为本人
		if s.PersonID == "" && r.SelfID != "" {
			s.PersonID = r.SelfID
			s.IsSelf = true
		}

		// 通过 SelfID 二次核对 IsSelf 标识
		if r.SelfID != "" && s.PersonID != "" {
			if strings.Contains(r.SelfID, s.PersonID) {
				s.IsSelf = true
			}
		}

		// 通过 SelfName 核对 IsMentionMe 标识
		if r.SelfName != "" && strings.Contains(s.Content, "@"+r.SelfName) {
			s.IsMentionMe = true
		}

		if s.IsChatroom {
			// 群聊
			// 获取更准确的群名（作为会话名）
			if chatRoom, ok := r.chatRoomCache[s.TopicID]; ok {
				s.TopicName = chatRoom.DisplayName()
			} else {
				contact := r.getFullContact(s.TopicID)
				if contact != nil {
					s.TopicName = contact.DisplayName()
				}
			}

			// 如果 PersonID 还是空的（或者 Wrap 没给），尝试从内容解析
			if s.PersonID == "" {
				if idx := strings.Index(s.Content, ":"); idx != -1 {
					senderID := s.Content[:idx]
					if len(senderID) > 0 && len(senderID) < 64 && !strings.ContainsAny(senderID, " \t\n\r") {
						s.PersonID = senderID
					}
				}
			}

			// 补全群聊发言人信息
			if s.PersonID != "" {
				if chatRoom, ok := r.chatRoomCache[s.TopicID]; ok {
					if displayName, ok := chatRoom.User2DisplayName[s.PersonID]; ok {
						s.PersonName = displayName
					}
				}
				if s.PersonName == "" {
					contact := r.getFullContact(s.PersonID)
					if contact != nil {
						s.PersonName = contact.DisplayName()
						if contact.Alias != "" {
							s.PersonID = contact.Alias
						}
					}
				}
			}
		} else {
			// 单聊
			contact := r.getFullContact(s.TopicID)
			if contact != nil {
				s.TopicName = contact.DisplayName()
			} else {
				s.TopicName = s.TopicID
			}

			if s.IsSelf {
				s.PersonID = r.SelfID
				s.PersonName = r.SelfName
				if s.PersonName == "" {
					s.PersonName = "Self"
				}
				// 尝试补充本人 Alias
				selfContact := r.findContact(r.SelfID)
				if selfContact != nil && selfContact.Alias != "" {
					s.PersonID = selfContact.Alias
				}
			} else {
				// 单聊对方，如果 PersonID 还是空的（通常不应该），则使用会话 ID
				if s.PersonID == "" {
					s.PersonID = s.TopicID
				}

				if contact != nil {
					// 针对单聊发送人，优先使用昵称
					if contact.NickName != "" {
						s.PersonName = contact.NickName
					} else {
						s.PersonName = s.TopicName
					}
					// 尝试使用 Alias
					if contact.Alias != "" {
						s.PersonID = contact.Alias
					}
				}
			}
		}

		// 统一保底：如果 PersonName 依然为空且有 ID，则由 ID 决定
		if s.PersonName == "" && s.PersonID != "" {
			s.PersonName = s.PersonID
		}
	}
	return nil
}

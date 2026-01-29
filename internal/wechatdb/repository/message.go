package repository

import (
	"context"
	"strings"
	"time"

	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/pkg/util"

	"github.com/rs/zerolog/log"
)

// GetMessages 实现 Repository 接口的 GetMessages 方法
func (r *Repository) GetMessages(ctx context.Context, startTime, endTime time.Time, talker string, sender string, keyword string, limit, offset int) ([]*model.Message, error) {

	talker, sender = r.parseTalkerAndSender(ctx, talker, sender)
	messages, err := r.ds.GetMessages(ctx, startTime, endTime, talker, sender, keyword, limit, offset)
	if err != nil {
		return nil, err
	}

	// 补充消息信息
	if err := r.EnrichMessages(ctx, messages); err != nil {
		log.Debug().Msgf("EnrichMessages failed: %v", err)
	}

	return messages, nil
}

// EnrichMessages 补充消息的额外信息
func (r *Repository) EnrichMessages(ctx context.Context, messages []*model.Message) error {
	for _, msg := range messages {
		r.enrichMessage(msg)
	}
	return nil
}

// enrichMessage 补充单条消息的额外信息
func (r *Repository) enrichMessage(msg *model.Message) {
	// 通过 SelfID 二次核对 IsSelf 标识
	if r.SelfID != "" && msg.Sender != "" {
		if strings.Contains(r.SelfID, msg.Sender) {
			msg.IsSelf = true
		}
	}

	// 检测提到了我 (IsMentionMe)
	if !msg.IsSelf {
		if r.SelfName != "" && strings.Contains(msg.Content, "@"+r.SelfName) {
			msg.IsMentionMe = true
		}
		// 特殊情况补全：如果昵称有特殊字符导致匹配失败，尝试使用微信号或备注进行二次核对
		if !msg.IsMentionMe && r.SelfID != "" {
			selfContact := r.findContact(r.SelfID)
			if selfContact != nil {
				if selfContact.Alias != "" && strings.Contains(msg.Content, "@"+selfContact.Alias) {
					msg.IsMentionMe = true
				}
				if selfContact.Remark != "" && strings.Contains(msg.Content, "@"+selfContact.Remark) {
					msg.IsMentionMe = true
				}
			}
		}
	}

	// 处理群聊消息
	if msg.IsChatRoom {
		// 补充群聊名称
		if chatRoom, ok := r.chatRoomCache[msg.Talker]; ok {
			msg.TalkerName = chatRoom.DisplayName()

			// 补充发送者在群里的显示名称
			if displayName, ok := chatRoom.User2DisplayName[msg.Sender]; ok {
				msg.SenderName = displayName
			}
		}
	}

	// 如果是自己发送的消息，补充名称（优先使用实名，其次为“我”）
	if msg.IsSelf && msg.SenderName == "" {
		if r.SelfName != "" {
			msg.SenderName = r.SelfName
		} else {
			msg.SenderName = "我"
		}
	}

	// 如果不是自己发送的消息且还没有显示名称，尝试补充发送者信息
	if msg.SenderName == "" && !msg.IsSelf {
		contact := r.getFullContact(msg.Sender)
		if contact != nil {
			msg.SenderName = contact.DisplayName()
		}
	}
}

func (r *Repository) parseTalkerAndSender(ctx context.Context, talker, sender string) (string, string) {
	displayName2User := make(map[string]string)
	users := make(map[string]bool)

	talkers := util.Str2List(talker, ",")
	if len(talkers) > 0 {
		for i := 0; i < len(talkers); i++ {
			if contact, _ := r.GetContact(ctx, talkers[i]); contact != nil {
				talkers[i] = contact.UserName
			} else if chatRoom, _ := r.GetChatRoom(ctx, talkers[i]); chatRoom != nil {
				talkers[i] = chatRoom.Name
			}
		}
		// 获取群聊的用户列表
		for i := 0; i < len(talkers); i++ {
			if chatRoom, _ := r.GetChatRoom(ctx, talkers[i]); chatRoom != nil {
				for user, displayName := range chatRoom.User2DisplayName {
					displayName2User[displayName] = user
				}
				for _, user := range chatRoom.Users {
					users[user.UserName] = true
				}
			}
		}
		talker = strings.Join(talkers, ",")
	}

	senders := util.Str2List(sender, ",")
	if len(senders) > 0 {
		for i := 0; i < len(senders); i++ {
			if user, ok := displayName2User[senders[i]]; ok {
				senders[i] = user
			} else {
				// 尝试直接获取联系人
				if contact, _ := r.GetContact(ctx, senders[i]); contact != nil {
					senders[i] = contact.UserName
					continue
				}

				// FIXME 大量群聊用户名称重复，无法直接通过 GetContact 获取 ID，后续再优化
				for user := range users {
					if contact := r.getFullContact(user); contact != nil {
						if contact.DisplayName() == senders[i] {
							senders[i] = user
							break
						}
					}
				}
			}
		}
		sender = strings.Join(senders, ",")
	}

	return talker, sender
}

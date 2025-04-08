package repository

import (
	"context"
	"time"

	"github.com/sjzar/chatlog/internal/model"

	"github.com/rs/zerolog/log"
)

// GetMessages 实现 Repository 接口的 GetMessages 方法
func (r *Repository) GetMessages(ctx context.Context, startTime, endTime time.Time, talker string, limit, offset int) ([]*model.Message, error) {

	if contact, _ := r.GetContact(ctx, talker); contact != nil {
		talker = contact.UserName
	} else if chatRoom, _ := r.GetChatRoom(ctx, talker); chatRoom != nil {
		talker = chatRoom.Name
	}

	messages, err := r.ds.GetMessages(ctx, startTime, endTime, talker, limit, offset)
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

	// 如果不是自己发送的消息且还没有显示名称，尝试补充发送者信息
	if msg.SenderName == "" && !msg.IsSelf {
		contact := r.getFullContact(msg.Sender)
		if contact != nil {
			msg.SenderName = contact.DisplayName()
		}
	}
}

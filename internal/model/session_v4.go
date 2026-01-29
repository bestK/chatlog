package model

import (
	"strings"
	"time"
)

// 注意，v4 session 是独立数据库文件
// CREATE TABLE SessionTable(
// username TEXT PRIMARY KEY,
// type INTEGER,
// unread_count INTEGER,
// unread_first_msg_srv_id INTEGER,
// is_hidden INTEGER,
// summary TEXT,
// draft TEXT,
// status INTEGER,
// last_timestamp INTEGER,
// sort_timestamp INTEGER,
// last_clear_unread_timestamp INTEGER,
// last_msg_locald_id INTEGER,
// last_msg_type INTEGER,
// last_msg_sub_type INTEGER,
// last_msg_sender TEXT,
// last_sender_display_name TEXT,
// last_msg_ext_type INTEGER
// )
type SessionV4 struct {
	Username              string `json:"username"`
	Summary               string `json:"summary"`
	LastTimestamp         int    `json:"last_timestamp"`
	LastMsgSender         string `json:"last_msg_sender"`
	LastSenderDisplayName string `json:"last_sender_display_name"`
	LastMsgType           int    `json:"last_msg_type"`
	LastMsgSubType        int    `json:"last_msg_sub_type"`
	Status                int    `json:"status"`

	// Type                     int    `json:"type"`
	// UnreadCount              int    `json:"unread_count"`
	// UnreadFirstMsgSrvID      int    `json:"unread_first_msg_srv_id"`
	// IsHidden                 int    `json:"is_hidden"`
	// Draft                    string `json:"draft"`
	// SortTimestamp            int    `json:"sort_timestamp"`
	// LastClearUnreadTimestamp int    `json:"last_clear_unread_timestamp"`
	// LastMsgLocaldID          int    `json:"last_msg_locald_id"`
	// LastMsgExtType           int    `json:"last_msg_ext_type"`
}

func (s *SessionV4) Wrap() *Session {
	content := s.Summary
	if content == "" {
		switch s.LastMsgType {
		case MessageTypeImage:
			content = "[图片]"
		case MessageTypeVoice:
			content = "[语音]"
		case MessageTypeVideo:
			content = "[视频]"
		case MessageTypeLocation:
			content = "[位置]"
		case MessageTypeAnimation:
			content = "[表情]"
		case MessageTypeVOIP:
			content = "[语音通话]"
		case MessageTypeCard:
			content = "[名片]"
		case MessageTypeShare:
			switch s.LastMsgSubType {
			case MessageSubTypeFile:
				content = "[文件]"
			case MessageSubTypeLink, MessageSubTypeLink2:
				content = "[链接]"
			case MessageSubTypeMiniProgram, MessageSubTypeMiniProgram2:
				content = "[小程序]"
			case MessageSubTypeChannel:
				content = "[视频号]"
			case MessageSubTypeMusic:
				content = "[音乐]"
			default:
				content = "[分享]"
			}
		case MessageTypeSystem:
			if s.LastMsgSubType == MessageSubTypePat {
				content = "[拍一拍]"
			} else {
				content = "[系统消息]"
			}
		}
	}
	isChatroom := strings.HasSuffix(s.Username, "@chatroom")
	res := &Session{
		TopicName:  s.LastSenderDisplayName,
		TopicID:    s.Username,
		NOrder:     s.LastTimestamp,
		Content:    content,
		NTime:      time.Unix(int64(s.LastTimestamp), 0),
		IsSelf:     s.Status == 2,
		IsChatroom: isChatroom,
		PersonID:   s.LastMsgSender,
		PersonName: s.LastSenderDisplayName,
	}
	if res.TopicName == "" {
		res.TopicName = s.Username
	}
	return res
}

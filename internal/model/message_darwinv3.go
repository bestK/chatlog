package model

import (
	"strings"
	"time"
)

// CREATE TABLE Chat_md5(talker)(
// mesLocalID INTEGER PRIMARY KEY AUTOINCREMENT,
// mesSvrID INTEGER,msgCreateTime INTEGER,
// msgContent TEXT,msgStatus INTEGER,
// msgImgStatus INTEGER,
// messageType INTEGER,
// mesDes INTEGER,
// msgSource TEXT,
// IntRes1 INTEGER,
// IntRes2 INTEGER,
// StrRes1 TEXT,
// StrRes2 TEXT,
// msgVoiceText TEXT,
// msgSeq INTEGER,
// CompressContent BLOB,
// ConBlob BLOB
// )
type MessageDarwinV3 struct {
	MsgCreateTime int64  `json:"msgCreateTime"`
	MsgContent    string `json:"msgContent"`
	MessageType   int64  `json:"messageType"`
	MesDes        int    `json:"mesDes"` // 0: 发送, 1: 接收

	// MesLocalID      int64  `json:"mesLocalID"`
	// MesSvrID        int64  `json:"mesSvrID"`
	// MesStatus       int    `json:"mesStatus"`
	// MesImgStatus    int    `json:"mesImgStatus"`
	// MsgSource       string `json:"msgSource"`
	// IntRes1         int    `json:"IntRes1"`
	// IntRes2         int    `json:"IntRes2"`
	// StrRes1         string `json:"StrRes1"`
	// StrRes2         string `json:"StrRes2"`
	// MesVoiceText    string `json:"mesVoiceText"`
	// MesSeq          int    `json:"mesSeq"`
	// CompressContent []byte `json:"CompressContent"`
	// ConBlob         []byte `json:"ConBlob"`
}

func (m *MessageDarwinV3) Wrap(talker string) *Message {

	_m := &Message{
		CreateTime: time.Unix(m.MsgCreateTime, 0),
		Type:       m.MessageType,
		IsSender:   (m.MesDes + 1) % 2,
		Version:    WeChatDarwinV3,
	}

	_m.IsChatRoom = strings.HasSuffix(talker, "@chatroom")

	_m.Content = m.MsgContent
	if _m.IsChatRoom {
		split := strings.SplitN(m.MsgContent, ":\n", 2)
		if len(split) == 2 {
			_m.ChatRoomSender = split[0]
			_m.Content = split[1]
		}
	}

	if _m.Type != 1 {
		mediaMessage, err := NewMediaMessage(_m.Type, _m.Content)
		if err == nil {
			_m.MediaMessage = mediaMessage
		}
	}

	return _m
}

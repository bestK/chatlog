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
	MesCreateTime int64  `json:"mesCreateTime"`
	MesContent    string `json:"mesContent"`
	MesType       int    `json:"mesType"`
	MesDes        int    `json:"mesDes"` // 0: 发送, 1: 接收
	MesSource     string `json:"mesSource"`

	// MesLocalID      int64  `json:"mesLocalID"`
	// MesSvrID        int64  `json:"mesSvrID"`
	// MesStatus       int    `json:"mesStatus"`
	// MesImgStatus    int    `json:"mesImgStatus"`
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
	isChatRoom := strings.HasSuffix(talker, "@chatroom")

	var chatRoomSender string
	content := m.MesContent
	if isChatRoom {
		split := strings.SplitN(m.MesContent, ":\n", 2)
		if len(split) == 2 {
			chatRoomSender = split[0]
			content = split[1]
		}
	}

	return &Message{
		CreateTime:     time.Unix(m.MesCreateTime, 0),
		Content:        content,
		Talker:         talker,
		Type:           m.MesType,
		IsSender:       (m.MesDes + 1) % 2,
		IsChatRoom:     isChatRoom,
		ChatRoomSender: chatRoomSender,
		Version:        WeChatDarwinV3,
	}
}

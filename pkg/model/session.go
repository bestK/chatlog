package model

import (
	"strings"
	"time"
)

type Session struct {
	UserName string    `json:"userName"`
	NOrder   int       `json:"nOrder"`
	NickName string    `json:"nickName"`
	Content  string    `json:"content"`
	NTime    time.Time `json:"nTime"`
}

// CREATE TABLE Session(
// strUsrName TEXT  PRIMARY KEY,
// nOrder INT DEFAULT 0,
// nUnReadCount INTEGER DEFAULT 0,
// parentRef TEXT,
// Reserved0 INTEGER DEFAULT 0,
// Reserved1 TEXT,
// strNickName TEXT,
// nStatus INTEGER,
// nIsSend INTEGER,
// strContent TEXT,
// nMsgType	INTEGER,
// nMsgLocalID INTEGER,
// nMsgStatus INTEGER,
// nTime INTEGER,
// editContent TEXT,
// othersAtMe INT,
// Reserved2 INTEGER DEFAULT 0,
// Reserved3 TEXT,
// Reserved4 INTEGER DEFAULT 0,
// Reserved5 TEXT,
// bytesXml BLOB
// )
type SessionV3 struct {
	StrUsrName  string `json:"strUsrName"`
	NOrder      int    `json:"nOrder"`
	StrNickName string `json:"strNickName"`
	StrContent  string `json:"strContent"`
	NTime       int64  `json:"nTime"`

	// NUnReadCount int    `json:"nUnReadCount"`
	// ParentRef    string `json:"parentRef"`
	// Reserved0    int    `json:"Reserved0"`
	// Reserved1    string `json:"Reserved1"`
	// NStatus      int    `json:"nStatus"`
	// NIsSend      int    `json:"nIsSend"`
	// NMsgType     int    `json:"nMsgType"`
	// NMsgLocalID  int    `json:"nMsgLocalID"`
	// NMsgStatus   int    `json:"nMsgStatus"`
	// EditContent  string `json:"editContent"`
	// OthersAtMe   int    `json:"othersAtMe"`
	// Reserved2    int    `json:"Reserved2"`
	// Reserved3    string `json:"Reserved3"`
	// Reserved4    int    `json:"Reserved4"`
	// Reserved5    string `json:"Reserved5"`
	// BytesXml     string `json:"bytesXml"`
}

func (s *SessionV3) Wrap() *Session {
	return &Session{
		UserName: s.StrUsrName,
		NOrder:   s.NOrder,
		NickName: s.StrNickName,
		Content:  s.StrContent,
		NTime:    time.Unix(int64(s.NTime), 0),
	}
}

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

	// Type                     int    `json:"type"`
	// UnreadCount              int    `json:"unread_count"`
	// UnreadFirstMsgSrvID      int    `json:"unread_first_msg_srv_id"`
	// IsHidden                 int    `json:"is_hidden"`
	// Draft                    string `json:"draft"`
	// Status                   int    `json:"status"`
	// SortTimestamp            int    `json:"sort_timestamp"`
	// LastClearUnreadTimestamp int    `json:"last_clear_unread_timestamp"`
	// LastMsgLocaldID          int    `json:"last_msg_locald_id"`
	// LastMsgType              int    `json:"last_msg_type"`
	// LastMsgSubType           int    `json:"last_msg_sub_type"`
	// LastMsgExtType           int    `json:"last_msg_ext_type"`
}

func (s *Session) PlainText(limit int) string {
	buf := strings.Builder{}
	buf.WriteString(s.NickName)
	buf.WriteString("(")
	buf.WriteString(s.UserName)
	buf.WriteString(") ")
	buf.WriteString(s.NTime.Format("2006-01-02 15:04:05"))
	buf.WriteString("\n")
	if limit > 0 {
		if len(s.Content) > limit {
			buf.WriteString(s.Content[:limit])
			buf.WriteString(" <...>")
		} else {
			buf.WriteString(s.Content)
		}
	}
	buf.WriteString("\n")
	return buf.String()
}

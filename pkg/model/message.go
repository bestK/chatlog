package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/sjzar/chatlog/pkg/model/wxproto"
	"github.com/sjzar/chatlog/pkg/util"

	"google.golang.org/protobuf/proto"
)

const (
	// Source
	WeChatV3 = "wechatv3"
	WeChatV4 = "wechatv4"
)

type Message struct {
	Sequence        int64     `json:"sequence"`        // 消息序号，10位时间戳 + 3位序号
	CreateTime      time.Time `json:"createTime"`      // 消息创建时间，10位时间戳
	TalkerID        int       `json:"talkerID"`        // 聊天对象，Name2ID 表序号，索引值
	Talker          string    `json:"talker"`          // 聊天对象，微信 ID or 群 ID
	IsSender        int       `json:"isSender"`        // 是否为发送消息，0 接收消息，1 发送消息
	Type            int       `json:"type"`            // 消息类型
	SubType         int       `json:"subType"`         // 消息子类型
	Content         string    `json:"content"`         // 消息内容，文字聊天内容 或 XML
	CompressContent []byte    `json:"compressContent"` // 非文字聊天内容，如图片、语音、视频等
	IsChatRoom      bool      `json:"isChatRoom"`      // 是否为群聊消息
	ChatRoomSender  string    `json:"chatRoomSender"`  // 群聊消息发送人

	// Fill Info
	// 从联系人等信息中填充
	DisplayName  string `json:"-"` // 显示名称
	CharRoomName string `json:"-"` // 群聊名称

	Version string `json:"-"` // 消息版本，内部判断
}

// CREATE TABLE MSG (
// localId INTEGER PRIMARY KEY AUTOINCREMENT,
// TalkerId INT DEFAULT 0,
// MsgSvrID INT,
// Type INT,
// SubType INT,
// IsSender INT,
// CreateTime INT,
// Sequence INT DEFAULT 0,
// StatusEx INT DEFAULT 0,
// FlagEx INT,
// Status INT,
// MsgServerSeq INT,
// MsgSequence INT,
// StrTalker TEXT,
// StrContent TEXT,
// DisplayContent TEXT,
// Reserved0 INT DEFAULT 0,
// Reserved1 INT DEFAULT 0,
// Reserved2 INT DEFAULT 0,
// Reserved3 INT DEFAULT 0,
// Reserved4 TEXT,
// Reserved5 TEXT,
// Reserved6 TEXT,
// CompressContent BLOB,
// BytesExtra BLOB,
// BytesTrans BLOB
// )
type MessageV3 struct {
	Sequence        int64  `json:"Sequence"`        // 消息序号，10位时间戳 + 3位序号
	CreateTime      int64  `json:"CreateTime"`      // 消息创建时间，10位时间戳
	TalkerID        int    `json:"TalkerId"`        // 聊天对象，Name2ID 表序号，索引值
	StrTalker       string `json:"StrTalker"`       // 聊天对象，微信 ID or 群 ID
	IsSender        int    `json:"IsSender"`        // 是否为发送消息，0 接收消息，1 发送消息
	Type            int    `json:"Type"`            // 消息类型
	SubType         int    `json:"SubType"`         // 消息子类型
	StrContent      string `json:"StrContent"`      // 消息内容，文字聊天内容 或 XML
	CompressContent []byte `json:"CompressContent"` // 非文字聊天内容，如图片、语音、视频等
	BytesExtra      []byte `json:"BytesExtra"`      // protobuf 额外数据，记录群聊发送人等信息

	// 非关键信息，后续有需要再加入
	// LocalID        int64  `json:"localId"`
	// MsgSvrID       int64  `json:"MsgSvrID"`
	// StatusEx       int    `json:"StatusEx"`
	// FlagEx         int    `json:"FlagEx"`
	// Status         int    `json:"Status"`
	// MsgServerSeq   int64  `json:"MsgServerSeq"`
	// MsgSequence    int64  `json:"MsgSequence"`
	// DisplayContent string `json:"DisplayContent"`
	// Reserved0      int    `json:"Reserved0"`
	// Reserved1      int    `json:"Reserved1"`
	// Reserved2      int    `json:"Reserved2"`
	// Reserved3      int    `json:"Reserved3"`
	// Reserved4      string `json:"Reserved4"`
	// Reserved5      string `json:"Reserved5"`
	// Reserved6      string `json:"Reserved6"`
	// BytesTrans     []byte `json:"BytesTrans"`
}

func (m *MessageV3) Wrap() *Message {

	isChatRoom := strings.HasSuffix(m.StrTalker, "@chatroom")

	var chatRoomSender string
	if len(m.BytesExtra) != 0 && isChatRoom {
		chatRoomSender = ParseBytesExtra(m.BytesExtra)
	}

	return &Message{
		Sequence:        m.Sequence,
		CreateTime:      time.Unix(m.CreateTime, 0),
		TalkerID:        m.TalkerID,
		Talker:          m.StrTalker,
		IsSender:        m.IsSender,
		Type:            m.Type,
		SubType:         m.SubType,
		Content:         m.StrContent,
		CompressContent: m.CompressContent,
		IsChatRoom:      isChatRoom,
		ChatRoomSender:  chatRoomSender,
		Version:         WeChatV3,
	}
}

// CREATE TABLE Msg_xxxxxxxxxxxx(
// local_id INTEGER PRIMARY KEY AUTOINCREMENT,
// server_id INTEGER,
// local_type INTEGER,
// sort_seq INTEGER,
// real_sender_id INTEGER,
// create_time INTEGER,
// status INTEGER,
// upload_status INTEGER,
// download_status INTEGER,
// server_seq INTEGER,
// origin_source INTEGER,
// source TEXT,
// message_content TEXT,
// compress_content TEXT,
// packed_info_data BLOB,
// WCDB_CT_message_content INTEGER DEFAULT NULL,
// WCDB_CT_source INTEGER DEFAULT NULL
// )
type MessageV4 struct {
	SortSeq        int64  `json:"sort_seq"`         // 消息序号，10位时间戳 + 3位序号
	LocalType      int    `json:"local_type"`       // 消息类型
	RealSenderID   int    `json:"real_sender_id"`   // 发送人 ID，对应 Name2Id 表序号
	CreateTime     int64  `json:"create_time"`      // 消息创建时间，10位时间戳
	MessageContent []byte `json:"message_content"`  // 消息内容，文字聊天内容 或 zstd 压缩内容
	PackedInfoData []byte `json:"packed_info_data"` // 额外数据，类似 proto，格式与 v3 有差异
	Status         int    `json:"status"`           // 消息状态，2 是已发送，4 是已接收，可以用于判断 IsSender（猜测）

	// 非关键信息，后续有需要再加入
	// LocalID         int    `json:"local_id"`
	// ServerID        int64  `json:"server_id"`

	// UploadStatus    int    `json:"upload_status"`
	// DownloadStatus  int    `json:"download_status"`
	// ServerSeq       int    `json:"server_seq"`
	// OriginSource    int    `json:"origin_source"`
	// Source          string `json:"source"`
	// CompressContent string `json:"compress_content"`
}

func (m *MessageV4) Wrap(id2Name map[int]string, isChatRoom bool) *Message {

	_m := &Message{
		Sequence:        m.SortSeq,
		CreateTime:      time.Unix(m.CreateTime, 0),
		TalkerID:        m.RealSenderID, // 依赖 Name2Id 表进行转换为 StrTalker
		CompressContent: m.PackedInfoData,
		Type:            m.LocalType,
		Version:         WeChatV4,
	}

	if name, ok := id2Name[m.RealSenderID]; ok {
		_m.Talker = name
	}

	if m.Status == 2 {
		_m.IsSender = 1
	}

	if util.IsNormalString(m.MessageContent) {
		_m.Content = string(m.MessageContent)
	} else {
		_m.CompressContent = m.MessageContent
	}

	if isChatRoom {
		_m.IsChatRoom = true
		split := strings.Split(_m.Content, "\n")
		if len(split) > 1 {
			_m.Content = split[1]
			_m.ChatRoomSender = strings.TrimSuffix(split[0], ":")
		}
	}

	return _m
}

// ParseBytesExtra 解析额外数据
// 按需解析
func ParseBytesExtra(b []byte) (chatRoomSender string) {
	var pbMsg wxproto.BytesExtra
	if err := proto.Unmarshal(b, &pbMsg); err != nil {
		return
	}
	if pbMsg.Items == nil {
		return
	}

	for _, item := range pbMsg.Items {
		if item.Type == 1 {
			return item.Value
		}
	}

	return
}

func (m *Message) PlainText(showChatRoom bool) string {
	buf := strings.Builder{}

	talker := m.Talker
	if m.IsSender == 1 {
		talker = "我"
	} else if m.IsChatRoom {
		talker = m.ChatRoomSender
	}
	if m.DisplayName != "" {
		buf.WriteString(m.DisplayName)
		buf.WriteString("(")
		buf.WriteString(talker)
		buf.WriteString(")")
	} else {
		buf.WriteString(talker)
	}
	buf.WriteString(" ")

	if m.IsChatRoom && showChatRoom {
		buf.WriteString("[")
		if m.CharRoomName != "" {
			buf.WriteString(m.CharRoomName)
			buf.WriteString("(")
			buf.WriteString(m.Talker)
			buf.WriteString(")")
		} else {
			buf.WriteString(m.Talker)
		}
		buf.WriteString("] ")
	}

	buf.WriteString(m.CreateTime.Format("2006-01-02 15:04:05"))
	buf.WriteString("\n")

	switch m.Type {
	case 1:
		buf.WriteString(m.Content)
	case 3:
		buf.WriteString("[图片]")
	case 34:
		buf.WriteString("[语音]")
	case 43:
		buf.WriteString("[视频]")
	case 47:
		buf.WriteString("[动画表情]")
	case 49:
		switch m.SubType {
		case 6:
			buf.WriteString("[文件]")
		case 8:
			buf.WriteString("[GIF表情]")
		case 19:
			buf.WriteString("[合并转发]")
		case 33, 36:
			buf.WriteString("[小程序]")
		case 57:
			buf.WriteString("[引用]")
		case 63:
			buf.WriteString("[视频号]")
		case 87:
			buf.WriteString("[群公告]")
		case 2000:
			buf.WriteString("[转账]")
		case 2003:
			buf.WriteString("[红包封面]")
		default:
			buf.WriteString("[分享]")
		}
	case 50:
		buf.WriteString("[语音通话]")
	case 10000:
		buf.WriteString("[系统消息]")
	default:
		buf.WriteString(fmt.Sprintf("Type: %d Content: %s", m.Type, m.Content))
	}
	buf.WriteString("\n")

	return buf.String()
}

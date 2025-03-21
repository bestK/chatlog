package model

import (
	"bytes"
	"strings"
	"time"

	"github.com/sjzar/chatlog/pkg/util/zstd"
)

// CREATE TABLE Msg_md5(talker)(
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

	if _m.Type == 1 {
		_m.Content = string(m.MessageContent)
	} else {
		if bytes.HasPrefix(m.MessageContent, []byte{0x28, 0xb5, 0x2f, 0xfd}) {
			if b, err := zstd.Decompress(m.MessageContent); err == nil {
				_m.Content = string(b)
			}
		} else {
			_m.CompressContent = m.MessageContent
		}
	}

	if isChatRoom {
		_m.IsChatRoom = true
		split := strings.SplitN(_m.Content, ":\n", 2)
		if len(split) == 2 {
			_m.ChatRoomSender = split[0]
			_m.Content = split[1]
		}
	}

	return _m
}

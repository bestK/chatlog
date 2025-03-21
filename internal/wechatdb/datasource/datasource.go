package datasource

import (
	"context"
	"fmt"
	"time"

	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb/datasource/darwinv3"
	v4 "github.com/sjzar/chatlog/internal/wechatdb/datasource/v4"
	"github.com/sjzar/chatlog/internal/wechatdb/datasource/windowsv3"
)

// 错误定义
var (
	ErrUnsupportedPlatform = fmt.Errorf("unsupported platform")
)

type DataSource interface {

	// 消息
	GetMessages(ctx context.Context, startTime, endTime time.Time, talker string, limit, offset int) ([]*model.Message, error)

	// 联系人
	GetContacts(ctx context.Context, key string, limit, offset int) ([]*model.Contact, error)

	// 群聊
	GetChatRooms(ctx context.Context, key string, limit, offset int) ([]*model.ChatRoom, error)

	// 最近会话
	GetSessions(ctx context.Context, key string, limit, offset int) ([]*model.Session, error)

	Close() error
}

func NewDataSource(path string, platform string, version int) (DataSource, error) {
	switch {
	case platform == "windows" && version == 3:
		return windowsv3.New(path)
	case platform == "windows" && version == 4:
		return v4.New(path)
	case platform == "darwin" && version == 3:
		return darwinv3.New(path)
	case platform == "darwin" && version == 4:
		return v4.New(path)
	default:
		return nil, fmt.Errorf("%w: %s v%d", ErrUnsupportedPlatform, platform, version)
	}
}

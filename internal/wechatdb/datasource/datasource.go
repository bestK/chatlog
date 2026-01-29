package datasource

import (
	"context"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb/datasource/darwinv3"
	v4 "github.com/sjzar/chatlog/internal/wechatdb/datasource/v4"
	"github.com/sjzar/chatlog/internal/wechatdb/datasource/windowsv3"
)

type DataSource interface {

	// 消息
	GetMessages(ctx context.Context, startTime, endTime time.Time, speakerto string, talker string, sender string, keyword string, limit, offset int) ([]*model.Message, error)

	// 联系人
	GetContacts(ctx context.Context, key string, limit, offset int) ([]*model.Contact, error)

	// 群聊
	GetChatRooms(ctx context.Context, key string, limit, offset int) ([]*model.ChatRoom, error)

	// 最近会话
	GetSessions(ctx context.Context, key string, limit, offset int) ([]*model.Session, error)

	// 媒体
	GetMedia(ctx context.Context, _type string, key string) (*model.Media, error)

	// 设置回调函数
	SetCallback(group string, callback func(event fsnotify.Event) error) error

	// 关闭指定路径的数据库连接
	CloseDB(path string) error

	// 锁定/解锁指定路径（用于文件替换时阻止新的 OpenDB）
	LockDB(path string) error
	UnlockDB(path string) error

	Close() error
}

func New(path string, platform string, version int) (DataSource, error) {
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
		return nil, errors.PlatformUnsupported(platform, version)
	}
}

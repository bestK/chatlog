package database

import (
	"time"

	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/wechatdb"
	"github.com/sjzar/chatlog/pkg/model"
)

type Service struct {
	ctx *ctx.Context
	db  *wechatdb.DB
}

func NewService(ctx *ctx.Context) *Service {
	return &Service{
		ctx: ctx,
	}
}

func (s *Service) Start() error {
	db, err := wechatdb.New(s.ctx.WorkDir, s.ctx.MajorVersion)
	if err != nil {
		return err
	}
	s.db = db
	return nil
}

func (s *Service) Stop() error {
	if s.db != nil {
		s.db.Close()
	}
	s.db = nil
	return nil
}

// GetDB returns the underlying database
func (s *Service) GetDB() *wechatdb.DB {
	return s.db
}

// GetMessages retrieves messages based on criteria
func (s *Service) GetMessages(start, end time.Time, talker string, limit, offset int) ([]*model.Message, error) {
	return s.db.GetMessages(start, end, talker, limit, offset)
}

// GetContact retrieves contact information
func (s *Service) GetContact(userName string) *model.Contact {
	return s.db.GetContact(userName)
}

// ListContact retrieves all contacts
func (s *Service) ListContact() (*wechatdb.ListContactResp, error) {
	return s.db.ListContact()
}

// GetChatRoom retrieves chat room information
func (s *Service) GetChatRoom(name string) *model.ChatRoom {
	return s.db.GetChatRoom(name)
}

// ListChatRoom retrieves all chat rooms
func (s *Service) ListChatRoom() (*wechatdb.ListChatRoomResp, error) {
	return s.db.ListChatRoom()
}

// GetSession retrieves session information
func (s *Service) GetSession(limit int) (*wechatdb.GetSessionResp, error) {
	return s.db.GetSession(limit)
}

// Close closes the database connection
func (s *Service) Close() {
	// Add cleanup code if needed
}

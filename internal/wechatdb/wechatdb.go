package wechatdb

import (
	"time"

	"github.com/sjzar/chatlog/pkg/model"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	BasePath string
	Version  int

	contact *Contact
	message *Message
}

func New(path string, version int) (*DB, error) {
	w := &DB{
		BasePath: path,
		Version:  version,
	}

	// 初始化，加载数据库文件信息
	if err := w.Initialize(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *DB) Close() error {
	return nil
}

func (w *DB) Initialize() error {

	var err error
	w.message, err = NewMessage(w.BasePath, w.Version)
	if err != nil {
		return err
	}

	w.contact, err = NewContact(w.BasePath, w.Version)
	if err != nil {
		return err
	}

	return nil
}

func (w *DB) GetMessages(start, end time.Time, talker string, limit, offset int) ([]*model.Message, error) {

	if talker != "" {
		if contact := w.contact.GetContact(talker); contact != nil {
			talker = contact.UserName
		}
	}

	messages, err := w.message.GetMessages(start, end, talker, limit, offset)
	if err != nil {
		return nil, err
	}
	for i := range messages {
		w.contact.MessageFillInfo(messages[i])
	}

	return messages, nil
}

type ListContactResp struct {
	Items []*model.Contact `json:"items"`
}

func (w *DB) ListContact() (*ListContactResp, error) {
	list, err := w.contact.ListContact()
	if err != nil {
		return nil, err
	}
	return &ListContactResp{
		Items: list,
	}, nil
}

func (w *DB) GetContact(userName string) *model.Contact {
	return w.contact.GetContact(userName)
}

type ListChatRoomResp struct {
	Items []*model.ChatRoom `json:"items"`
}

func (w *DB) ListChatRoom() (*ListChatRoomResp, error) {
	list, err := w.contact.ListChatRoom()
	if err != nil {
		return nil, err
	}
	return &ListChatRoomResp{
		Items: list,
	}, nil
}

func (w *DB) GetChatRoom(userName string) *model.ChatRoom {
	return w.contact.GetChatRoom(userName)
}

type GetSessionResp struct {
	Items []*model.Session `json:"items"`
}

func (w *DB) GetSession(limit int) (*GetSessionResp, error) {
	sessions := w.contact.GetSession(limit)
	return &GetSessionResp{
		Items: sessions,
	}, nil
}

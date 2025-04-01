package repository

import (
	"context"
	"sort"
	"strings"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
)

// initChatRoomCache 初始化群聊缓存
func (r *Repository) initChatRoomCache(ctx context.Context) error {
	// 加载所有群聊到缓存
	chatRooms, err := r.ds.GetChatRooms(ctx, "", 0, 0)
	if err != nil {
		return err
	}

	chatRoomMap := make(map[string]*model.ChatRoom)
	remarkToChatRoom := make(map[string]*model.ChatRoom)
	nickNameToChatRoom := make(map[string]*model.ChatRoom)
	chatRoomList := make([]string, 0)
	chatRoomRemark := make([]string, 0)
	chatRoomNickName := make([]string, 0)

	for _, chatRoom := range chatRooms {
		// 补充群聊信息（从联系人中获取 Remark 和 NickName）
		r.enrichChatRoom(chatRoom)
		chatRoomMap[chatRoom.Name] = chatRoom
		chatRoomList = append(chatRoomList, chatRoom.Name)
		if chatRoom.Remark != "" {
			remarkToChatRoom[chatRoom.Remark] = chatRoom
			chatRoomRemark = append(chatRoomRemark, chatRoom.Remark)
		}
		if chatRoom.NickName != "" {
			nickNameToChatRoom[chatRoom.NickName] = chatRoom
			chatRoomNickName = append(chatRoomNickName, chatRoom.NickName)
		}
	}

	for _, contact := range r.chatRoomInContact {
		if _, ok := chatRoomMap[contact.UserName]; !ok {
			chatRoom := &model.ChatRoom{
				Name:     contact.UserName,
				Remark:   contact.Remark,
				NickName: contact.NickName,
			}
			chatRoomMap[contact.UserName] = chatRoom
			chatRoomList = append(chatRoomList, contact.UserName)
			if contact.Remark != "" {
				remarkToChatRoom[contact.Remark] = chatRoom
				chatRoomRemark = append(chatRoomRemark, contact.Remark)
			}
			if contact.NickName != "" {
				nickNameToChatRoom[contact.NickName] = chatRoom
				chatRoomNickName = append(chatRoomNickName, contact.NickName)
			}
		}
	}
	sort.Strings(chatRoomList)
	sort.Strings(chatRoomRemark)
	sort.Strings(chatRoomNickName)

	r.chatRoomCache = chatRoomMap
	r.chatRoomList = chatRoomList
	r.remarkToChatRoom = remarkToChatRoom
	r.nickNameToChatRoom = nickNameToChatRoom
	return nil
}

func (r *Repository) GetChatRooms(ctx context.Context, key string, limit, offset int) ([]*model.ChatRoom, error) {

	ret := make([]*model.ChatRoom, 0)
	if key != "" {
		ret = r.findChatRooms(key)
		if len(ret) == 0 {
			return nil, errors.ChatRoomNotFound(key)
		}

		if limit > 0 {
			end := offset + limit
			if end > len(ret) {
				end = len(ret)
			}
			if offset >= len(ret) {
				return []*model.ChatRoom{}, nil
			}
			return ret[offset:end], nil
		}
	} else {
		list := r.chatRoomList
		if limit > 0 {
			end := offset + limit
			if end > len(list) {
				end = len(list)
			}
			if offset >= len(list) {
				return []*model.ChatRoom{}, nil
			}
			list = list[offset:end]
		}
		for _, name := range list {
			ret = append(ret, r.chatRoomCache[name])
		}
	}

	return ret, nil
}

func (r *Repository) GetChatRoom(ctx context.Context, key string) (*model.ChatRoom, error) {
	chatRoom := r.findChatRoom(key)
	if chatRoom == nil {
		return nil, errors.ChatRoomNotFound(key)
	}
	return chatRoom, nil
}

// enrichChatRoom 从联系人信息中补充群聊信息
func (r *Repository) enrichChatRoom(chatRoom *model.ChatRoom) {
	if contact, ok := r.contactCache[chatRoom.Name]; ok {
		chatRoom.Remark = contact.Remark
		chatRoom.NickName = contact.NickName
	}
}

func (r *Repository) findChatRoom(key string) *model.ChatRoom {
	if chatRoom, ok := r.chatRoomCache[key]; ok {
		return chatRoom
	}
	if chatRoom, ok := r.remarkToChatRoom[key]; ok {
		return chatRoom
	}
	if chatRoom, ok := r.nickNameToChatRoom[key]; ok {
		return chatRoom
	}

	// Contain
	for _, remark := range r.chatRoomRemark {
		if strings.Contains(remark, key) {
			return r.remarkToChatRoom[remark]
		}
	}
	for _, nickName := range r.chatRoomNickName {
		if strings.Contains(nickName, key) {
			return r.nickNameToChatRoom[nickName]
		}
	}

	return nil
}

func (r *Repository) findChatRooms(key string) []*model.ChatRoom {
	ret := make([]*model.ChatRoom, 0)
	distinct := make(map[string]bool)
	if chatRoom, ok := r.chatRoomCache[key]; ok {
		ret = append(ret, chatRoom)
		distinct[chatRoom.Name] = true
	}
	if chatRoom, ok := r.remarkToChatRoom[key]; ok && !distinct[chatRoom.Name] {
		ret = append(ret, chatRoom)
		distinct[chatRoom.Name] = true
	}
	if chatRoom, ok := r.nickNameToChatRoom[key]; ok && !distinct[chatRoom.Name] {
		ret = append(ret, chatRoom)
		distinct[chatRoom.Name] = true
	}

	// Contain
	for _, remark := range r.chatRoomRemark {
		if strings.Contains(remark, key) && !distinct[r.remarkToChatRoom[remark].Name] {
			ret = append(ret, r.remarkToChatRoom[remark])
			distinct[r.remarkToChatRoom[remark].Name] = true
		}
	}
	for _, nickName := range r.chatRoomNickName {
		if strings.Contains(nickName, key) && !distinct[r.nickNameToChatRoom[nickName].Name] {
			ret = append(ret, r.nickNameToChatRoom[nickName])
			distinct[r.nickNameToChatRoom[nickName].Name] = true
		}
	}

	return ret
}

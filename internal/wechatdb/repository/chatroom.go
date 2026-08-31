package repository

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
)

// initChatRoomCache 初始化群聊缓存
func (r *Repository) initChatRoomCache(ctx context.Context) error {

	chatRoomMap := make(map[string]*model.ChatRoom)
	remarkToChatRoom := make(map[string][]*model.ChatRoom)
	nickNameToChatRoom := make(map[string][]*model.ChatRoom)
	chatRoomList := make([]string, 0)
	chatRoomRemark := make([]string, 0)
	chatRoomNickName := make([]string, 0)

	// 加载所有群聊到缓存
	// 暂时忽略获取不到群聊的错误
	chatRooms, err := r.ds.GetChatRooms(ctx, "", 0, 0)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load chat rooms")
	}

	for _, chatRoom := range chatRooms {
		// 补充群聊信息（从联系人中获取 Remark 和 NickName）
		r.enrichChatRoom(chatRoom)
		chatRoomMap[chatRoom.Name] = chatRoom
		chatRoomList = append(chatRoomList, chatRoom.Name)
		if chatRoom.Remark != "" {
			remark, ok := remarkToChatRoom[chatRoom.Remark]
			if !ok {
				remark = make([]*model.ChatRoom, 0)
			}
			remark = append(remark, chatRoom)
			remarkToChatRoom[chatRoom.Remark] = remark
			chatRoomRemark = append(chatRoomRemark, chatRoom.Remark)
		}
		if chatRoom.NickName != "" {
			nickName, ok := nickNameToChatRoom[chatRoom.NickName]
			if !ok {
				nickName = make([]*model.ChatRoom, 0)
			}
			nickName = append(nickName, chatRoom)
			nickNameToChatRoom[chatRoom.NickName] = nickName
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
				remark, ok := remarkToChatRoom[chatRoom.Remark]
				if !ok {
					remark = make([]*model.ChatRoom, 0)
				}
				remark = append(remark, chatRoom)
				remarkToChatRoom[chatRoom.Remark] = remark
				chatRoomRemark = append(chatRoomRemark, contact.Remark)
			}
			if contact.NickName != "" {
				nickName, ok := nickNameToChatRoom[chatRoom.NickName]
				if !ok {
					nickName = make([]*model.ChatRoom, 0)
				}
				nickName = append(nickName, chatRoom)
				nickNameToChatRoom[chatRoom.NickName] = nickName
				chatRoomNickName = append(chatRoomNickName, contact.NickName)
			}
		}
	}
	sort.Strings(chatRoomList)
	sort.Strings(chatRoomRemark)
	sort.Strings(chatRoomNickName)

	r.chatRoomCache = chatRoomMap
	r.remarkToChatRoom = remarkToChatRoom
	r.nickNameToChatRoom = nickNameToChatRoom
	r.chatRoomList = chatRoomList
	r.chatRoomRemark = chatRoomRemark
	r.chatRoomNickName = chatRoomNickName

	return nil
}

func (r *Repository) GetChatRooms(ctx context.Context, key string, limit, offset int) (int, []*model.ChatRoom, error) {

	ret := make([]*model.ChatRoom, 0)
	if key != "" {
		ret = r.findChatRooms(key)
		if len(ret) == 0 {
			return 0, []*model.ChatRoom{}, nil
		}

		total := len(ret)
		if limit > 0 {
			end := offset + limit
			if end > len(ret) {
				end = len(ret)
			}
			if offset >= len(ret) {
				return total, []*model.ChatRoom{}, nil
			}
			ret = ret[offset:end]
		}
		r.enrichChatRoomJoinTimes(ctx, ret)
		return total, ret, nil
	} else {
		list := r.chatRoomList
		total := len(list)
		if limit > 0 {
			end := offset + limit
			if end > len(list) {
				end = len(list)
			}
			if offset >= len(list) {
				return total, []*model.ChatRoom{}, nil
			}
			list = list[offset:end]
		}
		for _, name := range list {
			ret = append(ret, r.chatRoomCache[name])
		}
		r.enrichChatRoomJoinTimes(ctx, ret)
		return total, ret, nil
	}
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

	for i := range chatRoom.Users {
		user := &chatRoom.Users[i]
		if contact, ok := r.contactCache[user.UserName]; ok {
			if user.DisplayName == "" {
				user.DisplayName = contact.DisplayName()
			}
			user.AvatarURL = contactAvatarURL(contact)
		}

		if user.Inviter != "" {
			if inviter, ok := r.contactCache[user.Inviter]; ok {
				user.InviterDisplayName = inviter.DisplayName()
				user.InviterAvatarURL = contactAvatarURL(inviter)
			}
		}
	}
}

func (r *Repository) enrichChatRoomJoinTimes(ctx context.Context, chatRooms []*model.ChatRoom) {
	for _, chatRoom := range chatRooms {
		if chatRoom == nil || chatRoom.JoinTimesLoaded {
			continue
		}
		r.enrichChatRoomJoinTime(ctx, chatRoom)
	}
}

func (r *Repository) enrichChatRoomJoinTime(ctx context.Context, chatRoom *model.ChatRoom) {
	chatRoom.JoinTimesLoaded = true
	if len(chatRoom.Users) == 0 {
		return
	}

	startTime := time.Unix(0, 0)
	endTime := time.Date(9999, time.December, 31, 23, 59, 59, 0, time.Local)
	messages, err := r.ds.GetMessages(ctx, startTime, endTime, r.SelfID, chatRoom.Name, "", "", 0, 0, "asc")
	if err != nil {
		log.Debug().Err(err).Str("chatroom", chatRoom.Name).Msg("Failed to load chat room join times")
		return
	}

	userNameToIndexes := make(map[string][]int)
	for i, user := range chatRoom.Users {
		addChatRoomUserName(userNameToIndexes, user.UserName, i)
		addChatRoomUserName(userNameToIndexes, user.DisplayName, i)
		if contact, ok := r.contactCache[user.UserName]; ok {
			addChatRoomUserName(userNameToIndexes, contact.NickName, i)
			addChatRoomUserName(userNameToIndexes, contact.Remark, i)
			addChatRoomUserName(userNameToIndexes, contact.Alias, i)
		}
	}

	for _, message := range messages {
		if message.Type != model.MessageTypeSystem || !strings.Contains(message.Content, "加入") || !strings.Contains(message.Content, "群聊") {
			continue
		}

		quotedNames := quotedChatRoomNames(message.Content)
		if strings.Contains(message.Content, "邀请你") {
			setChatRoomUserJoinTime(chatRoom, userNameToIndexes, r.SelfID, message.Time)
		}

		var memberNames []string
		switch {
		case strings.Contains(message.Content, "你邀请"):
			memberNames = quotedNames
		case strings.Contains(message.Content, "邀请"):
			if len(quotedNames) > 1 {
				memberNames = quotedNames[1:]
			}
		case strings.Contains(message.Content, "通过扫描"):
			if len(quotedNames) > 0 {
				memberNames = quotedNames[:1]
			}
		default:
			if len(quotedNames) > 0 {
				memberNames = quotedNames[:1]
			}
		}

		for _, name := range memberNames {
			setChatRoomUserJoinTime(chatRoom, userNameToIndexes, name, message.Time)
		}
	}
}

func addChatRoomUserName(userNameToIndexes map[string][]int, name string, index int) {
	if name == "" {
		return
	}
	for _, existingIndex := range userNameToIndexes[name] {
		if existingIndex == index {
			return
		}
	}
	userNameToIndexes[name] = append(userNameToIndexes[name], index)
}

func setChatRoomUserJoinTime(chatRoom *model.ChatRoom, userNameToIndexes map[string][]int, name string, joinTime model.JSONTime) {
	indexes := userNameToIndexes[name]
	if len(indexes) != 1 {
		return
	}

	user := &chatRoom.Users[indexes[0]]
	if user.JoinTime == nil || joinTime.Before(*user.JoinTime) {
		joinTimeCopy := joinTime
		user.JoinTime = &joinTimeCopy
	}
}

func quotedChatRoomNames(content string) []string {
	names := make([]string, 0)
	for {
		start := strings.IndexByte(content, '"')
		if start < 0 {
			break
		}
		content = content[start+1:]
		end := strings.IndexByte(content, '"')
		if end < 0 {
			break
		}
		names = append(names, content[:end])
		content = content[end+1:]
	}
	return names
}

func contactAvatarURL(contact *model.Contact) string {
	if contact.BigHeadImgUrl != "" {
		return contact.BigHeadImgUrl
	}
	return contact.SmallHeadImgUrl
}

func (r *Repository) findChatRoom(key string) *model.ChatRoom {
	if chatRoom, ok := r.chatRoomCache[key]; ok {
		return chatRoom
	}
	if chatRoom, ok := r.remarkToChatRoom[key]; ok {
		return chatRoom[0]
	}
	if chatRoom, ok := r.nickNameToChatRoom[key]; ok {
		return chatRoom[0]
	}

	// Contain
	for _, remark := range r.chatRoomRemark {
		if strings.Contains(remark, key) {
			return r.remarkToChatRoom[remark][0]
		}
	}
	for _, nickName := range r.chatRoomNickName {
		if strings.Contains(nickName, key) {
			return r.nickNameToChatRoom[nickName][0]
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
	if chatRooms, ok := r.remarkToChatRoom[key]; ok {
		for _, chatRoom := range chatRooms {
			if !distinct[chatRoom.Name] {
				ret = append(ret, chatRoom)
				distinct[chatRoom.Name] = true
			}
		}
	}
	if chatRooms, ok := r.nickNameToChatRoom[key]; ok {
		for _, chatRoom := range chatRooms {
			if !distinct[chatRoom.Name] {
				ret = append(ret, chatRoom)
				distinct[chatRoom.Name] = true
			}
		}
	}

	// Contain
	for _, remark := range r.chatRoomRemark {
		if strings.Contains(remark, key) {
			for _, chatRoom := range r.remarkToChatRoom[remark] {
				if !distinct[chatRoom.Name] {
					ret = append(ret, chatRoom)
					distinct[chatRoom.Name] = true
				}
			}
		}
	}
	for _, nickName := range r.chatRoomNickName {
		if strings.Contains(nickName, key) {
			for _, chatRoom := range r.nickNameToChatRoom[nickName] {
				if !distinct[chatRoom.Name] {
					ret = append(ret, chatRoom)
					distinct[chatRoom.Name] = true
				}
			}
		}
	}

	return ret
}

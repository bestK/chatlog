package repository

import (
	"context"
	"sort"
	"strings"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
)

// initContactCache 初始化联系人缓存
func (r *Repository) initContactCache(ctx context.Context) error {
	// 加载所有联系人到缓存
	contacts, err := r.ds.GetContacts(ctx, "", 0, 0)
	if err != nil {
		return err
	}

	contactMap := make(map[string]*model.Contact)
	aliasMap := make(map[string]*model.Contact)
	remarkMap := make(map[string]*model.Contact)
	nickNameMap := make(map[string]*model.Contact)
	chatRoomUserMap := make(map[string]*model.Contact)
	chatRoomInContactMap := make(map[string]*model.Contact)
	contactList := make([]string, 0)
	aliasList := make([]string, 0)
	remarkList := make([]string, 0)
	nickNameList := make([]string, 0)

	for _, contact := range contacts {
		contactMap[contact.UserName] = contact
		contactList = append(contactList, contact.UserName)

		// 建立快速查找索引
		if contact.Alias != "" {
			aliasMap[contact.Alias] = contact
			aliasList = append(aliasList, contact.Alias)
		}
		if contact.Remark != "" {
			remarkMap[contact.Remark] = contact
			remarkList = append(remarkList, contact.Remark)
		}
		if contact.NickName != "" {
			nickNameMap[contact.NickName] = contact
			nickNameList = append(nickNameList, contact.NickName)
		}

		// 如果是群聊成员（非好友），添加到群聊成员索引
		if !contact.IsFriend {
			chatRoomUserMap[contact.UserName] = contact
		}

		if strings.HasSuffix(contact.UserName, "@chatroom") {
			chatRoomInContactMap[contact.UserName] = contact
		}
	}

	sort.Strings(contactList)
	sort.Strings(aliasList)
	sort.Strings(remarkList)
	sort.Strings(nickNameList)

	r.contactCache = contactMap
	r.aliasToContact = aliasMap
	r.remarkToContact = remarkMap
	r.nickNameToContact = nickNameMap
	r.chatRoomUserToInfo = chatRoomUserMap
	r.chatRoomInContact = chatRoomInContactMap
	r.contactList = contactList
	r.aliasList = aliasList
	r.remarkList = remarkList
	r.nickNameList = nickNameList
	return nil
}

func (r *Repository) GetContact(ctx context.Context, key string) (*model.Contact, error) {
	// 先尝试从缓存中获取
	contact := r.findContact(key)
	if contact == nil {
		return nil, errors.ContactNotFound(key)
	}
	return contact, nil
}

func (r *Repository) GetContacts(ctx context.Context, key string, limit, offset int) ([]*model.Contact, error) {
	ret := make([]*model.Contact, 0)
	if key != "" {
		ret = r.findContacts(key)
		if len(ret) == 0 {
			return nil, errors.ContactNotFound(key)
		}
		if limit > 0 {
			end := offset + limit
			if end > len(ret) {
				end = len(ret)
			}
			if offset >= len(ret) {
				return []*model.Contact{}, nil
			}
			return ret[offset:end], nil
		}
	} else {
		list := r.contactList
		if limit > 0 {
			end := offset + limit
			if end > len(list) {
				end = len(list)
			}
			if offset >= len(list) {
				return []*model.Contact{}, nil
			}
			list = list[offset:end]
		}
		for _, name := range list {
			ret = append(ret, r.contactCache[name])
		}
	}
	return ret, nil
}

func (r *Repository) findContact(key string) *model.Contact {
	if contact, ok := r.contactCache[key]; ok {
		return contact
	}
	if contact, ok := r.aliasToContact[key]; ok {
		return contact
	}
	if contact, ok := r.remarkToContact[key]; ok {
		return contact
	}
	if contact, ok := r.nickNameToContact[key]; ok {
		return contact
	}

	// Contain
	for _, alias := range r.aliasList {
		if strings.Contains(alias, key) {
			return r.aliasToContact[alias]
		}
	}
	for _, remark := range r.remarkList {
		if strings.Contains(remark, key) {
			return r.remarkToContact[remark]
		}
	}
	for _, nickName := range r.nickNameList {
		if strings.Contains(nickName, key) {
			return r.nickNameToContact[nickName]
		}
	}
	return nil
}

func (r *Repository) findContacts(key string) []*model.Contact {
	ret := make([]*model.Contact, 0)
	distinct := make(map[string]bool)
	if contact, ok := r.contactCache[key]; ok {
		ret = append(ret, contact)
		distinct[contact.UserName] = true
	}
	if contact, ok := r.aliasToContact[key]; ok && !distinct[contact.UserName] {
		ret = append(ret, contact)
		distinct[contact.UserName] = true
	}
	if contact, ok := r.remarkToContact[key]; ok && !distinct[contact.UserName] {
		ret = append(ret, contact)
		distinct[contact.UserName] = true
	}
	if contact, ok := r.nickNameToContact[key]; ok && !distinct[contact.UserName] {
		ret = append(ret, contact)
		distinct[contact.UserName] = true
	}
	// Contain
	for _, alias := range r.aliasList {
		if strings.Contains(alias, key) && !distinct[r.aliasToContact[alias].UserName] {
			ret = append(ret, r.aliasToContact[alias])
			distinct[r.aliasToContact[alias].UserName] = true
		}
	}
	for _, remark := range r.remarkList {
		if strings.Contains(remark, key) && !distinct[r.remarkToContact[remark].UserName] {
			ret = append(ret, r.remarkToContact[remark])
			distinct[r.remarkToContact[remark].UserName] = true
		}
	}
	for _, nickName := range r.nickNameList {
		if strings.Contains(nickName, key) && !distinct[r.nickNameToContact[nickName].UserName] {
			ret = append(ret, r.nickNameToContact[nickName])
			distinct[r.nickNameToContact[nickName].UserName] = true
		}
	}
	return ret
}

// getFullContact 获取联系人信息，包括群聊成员
func (r *Repository) getFullContact(userName string) *model.Contact {
	// 先查找联系人缓存
	if contact, ok := r.contactCache[userName]; ok {
		return contact
	}

	// 再查找群聊成员缓存
	contact, ok := r.chatRoomUserToInfo[userName]

	if ok {
		return contact
	}

	return nil
}

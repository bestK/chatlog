package wechatdb

import (
	"database/sql"
	"fmt"

	"github.com/sjzar/chatlog/pkg/model"
	"github.com/sjzar/chatlog/pkg/util"

	log "github.com/sirupsen/logrus"
)

var (
	ContactFileV3 = "^MicroMsg.db$"
	ContactFileV4 = "contact.db$"
)

type Contact struct {
	version int
	dbFile  string
	db      *sql.DB

	Contact  map[string]*model.Contact  // 好友和群聊信息，Key UserName
	ChatRoom map[string]*model.ChatRoom // 群聊信息，Key UserName
	Sessions []*model.Session           // 历史会话，按时间倒序

	// Quick Search
	ChatRoomUsers    map[string]*model.Contact // 群聊成员信息，Key UserName
	Alias2Contack    map[string]*model.Contact // 别名到联系人的映射
	Remark2Contack   map[string]*model.Contact // 备注名到联系人的映射
	NickName2Contack map[string]*model.Contact // 昵称到联系人的映射
}

func NewContact(path string, version int) (*Contact, error) {
	c := &Contact{
		version: version,
	}

	files, err := util.FindFilesWithPatterns(path, ContactFileV3, true)
	if err != nil {
		return nil, fmt.Errorf("查找数据库文件失败: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("未找到任何数据库文件: %s", path)
	}

	c.dbFile = files[0]

	c.db, err = sql.Open("sqlite3", c.dbFile)
	if err != nil {
		log.Printf("警告: 连接数据库 %s 失败: %v", c.dbFile, err)
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	c.loadContact()
	c.loadChatRoom()
	c.loadSession()
	c.fillChatRoomInfo()

	return c, nil
}

func (c *Contact) loadContact() {
	contactMap := make(map[string]*model.Contact)
	chatRoomUserMap := make(map[string]*model.Contact)
	aliasMap := make(map[string]*model.Contact)
	remarkMap := make(map[string]*model.Contact)
	nickNameMap := make(map[string]*model.Contact)
	rows, err := c.db.Query("SELECT UserName, Alias, Remark, NickName, Reserved1 FROM Contact")
	if err != nil {
		log.Errorf("查询联系人失败: %v", err)
		return
	}

	for rows.Next() {
		var contactv3 model.ContactV3

		if err := rows.Scan(
			&contactv3.UserName,
			&contactv3.Alias,
			&contactv3.Remark,
			&contactv3.NickName,
			&contactv3.Reserved1,
		); err != nil {
			log.Printf("警告: 扫描联系人行失败: %v", err)
			continue
		}
		contact := contactv3.Wrap()

		if contact.IsFriend {
			contactMap[contact.UserName] = contact
			if contact.Alias != "" {
				aliasMap[contact.Alias] = contact
			}
			if contact.Remark != "" {
				remarkMap[contact.Remark] = contact
			}
			if contact.NickName != "" {
				nickNameMap[contact.NickName] = contact
			}
		} else {
			chatRoomUserMap[contact.UserName] = contact
		}

	}
	rows.Close()

	c.Contact = contactMap
	c.ChatRoomUsers = chatRoomUserMap
	c.Alias2Contack = aliasMap
	c.Remark2Contack = remarkMap
	c.NickName2Contack = nickNameMap
}

func (c *Contact) loadChatRoom() {

	chatRoomMap := make(map[string]*model.ChatRoom)
	rows, err := c.db.Query("SELECT ChatRoomName, Reserved2, RoomData FROM ChatRoom")
	if err != nil {
		log.Errorf("查询群聊失败: %v", err)
		return
	}
	for rows.Next() {
		var chatRoom model.ChatRoomV3
		if err := rows.Scan(
			&chatRoom.ChatRoomName,
			&chatRoom.Reserved2,
			&chatRoom.RoomData,
		); err != nil {
			log.Printf("警告: 扫描群聊行失败: %v", err)
			continue
		}
		chatRoomMap[chatRoom.ChatRoomName] = chatRoom.Wrap()
	}
	rows.Close()
	c.ChatRoom = chatRoomMap
}

func (c *Contact) loadSession() {

	sessions := make([]*model.Session, 0)
	rows, err := c.db.Query("SELECT strUsrName, nOrder, strNickName, strContent, nTime FROM Session ORDER BY nOrder DESC")
	if err != nil {
		log.Errorf("查询群聊失败: %v", err)
		return
	}
	for rows.Next() {
		var sessionV3 model.SessionV3
		if err := rows.Scan(
			&sessionV3.StrUsrName,
			&sessionV3.NOrder,
			&sessionV3.StrNickName,
			&sessionV3.StrContent,
			&sessionV3.NTime,
		); err != nil {
			log.Printf("警告: 扫描历史会话失败: %v", err)
			continue
		}
		session := sessionV3.Wrap()
		sessions = append(sessions, session)

	}
	rows.Close()
	c.Sessions = sessions
}

func (c *Contact) ListContact() ([]*model.Contact, error) {
	contacts := make([]*model.Contact, 0, len(c.Contact))
	for _, contact := range c.Contact {
		contacts = append(contacts, contact)
	}
	return contacts, nil
}

func (c *Contact) ListChatRoom() ([]*model.ChatRoom, error) {
	chatRooms := make([]*model.ChatRoom, 0, len(c.ChatRoom))
	for _, chatRoom := range c.ChatRoom {
		chatRooms = append(chatRooms, chatRoom)
	}
	return chatRooms, nil
}

func (c *Contact) GetContact(key string) *model.Contact {
	if contact, ok := c.Contact[key]; ok {
		return contact
	}
	if contact, ok := c.Alias2Contack[key]; ok {
		return contact
	}
	if contact, ok := c.Remark2Contack[key]; ok {
		return contact
	}
	if contact, ok := c.NickName2Contack[key]; ok {
		return contact
	}
	return nil
}

func (c *Contact) GetChatRoom(name string) *model.ChatRoom {
	if chatRoom, ok := c.ChatRoom[name]; ok {
		return chatRoom
	}

	if contact := c.GetContact(name); contact != nil {
		if chatRoom, ok := c.ChatRoom[contact.UserName]; ok {
			return chatRoom
		} else {
			// 被删除的群聊，在 ChatRoom 记录中没有了，但是能找到 Contact，做下 Mock
			return &model.ChatRoom{
				Name:             contact.UserName,
				Remark:           contact.Remark,
				NickName:         contact.NickName,
				Users:            make([]model.ChatRoomUser, 0),
				User2DisplayName: make(map[string]string),
			}
		}
	}

	return nil
}

func (c *Contact) GetSession(limit int) []*model.Session {
	if limit <= 0 {
		limit = len(c.Sessions)
	}

	if len(c.Sessions) < limit {
		limit = len(c.Sessions)
	}
	return c.Sessions[:limit]
}

func (c *Contact) getFullContact(userName string) *model.Contact {
	if contact := c.GetContact(userName); contact != nil {
		return contact
	}
	if contact, ok := c.ChatRoomUsers[userName]; ok {
		return contact
	}
	return nil
}

func (c *Contact) fillChatRoomInfo() {
	for i := range c.ChatRoom {
		if contact := c.GetContact(c.ChatRoom[i].Name); contact != nil {
			c.ChatRoom[i].Remark = contact.Remark
			c.ChatRoom[i].NickName = contact.NickName
		}
	}
}

func (c *Contact) MessageFillInfo(msg *model.Message) {
	talker := msg.Talker
	if msg.IsChatRoom {
		talker = msg.ChatRoomSender
		if chatRoom := c.GetChatRoom(msg.Talker); chatRoom != nil {
			msg.CharRoomName = chatRoom.DisplayName()
			if displayName, ok := chatRoom.User2DisplayName[talker]; ok {
				msg.DisplayName = displayName
			}
		}
	}
	if msg.DisplayName == "" && msg.IsSender != 1 {
		if contact := c.getFullContact(talker); contact != nil {
			msg.DisplayName = contact.DisplayName()
		}
	}
}

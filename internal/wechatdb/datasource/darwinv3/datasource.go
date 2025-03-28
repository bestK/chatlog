package darwinv3

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/pkg/util"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

const (
	MessageFilePattern  = "^msg_([0-9]?[0-9])?\\.db$"
	ContactFilePattern  = "^wccontact_new2\\.db$"
	ChatRoomFilePattern = "^group_new\\.db$"
	SessionFilePattern  = "^session_new\\.db$"
	MediaFilePattern    = "^hldata\\.db$"
)

type DataSource struct {
	path       string
	messageDbs []*sql.DB
	contactDb  *sql.DB
	chatRoomDb *sql.DB
	sessionDb  *sql.DB
	mediaDb    *sql.DB

	talkerDBMap      map[string]*sql.DB
	user2DisplayName map[string]string
}

func New(path string) (*DataSource, error) {
	ds := &DataSource{
		path:             path,
		messageDbs:       make([]*sql.DB, 0),
		talkerDBMap:      make(map[string]*sql.DB),
		user2DisplayName: make(map[string]string),
	}

	if err := ds.initMessageDbs(path); err != nil {
		return nil, fmt.Errorf("初始化消息数据库失败: %w", err)
	}
	if err := ds.initContactDb(path); err != nil {
		return nil, fmt.Errorf("初始化联系人数据库失败: %w", err)
	}
	if err := ds.initChatRoomDb(path); err != nil {
		return nil, fmt.Errorf("初始化群聊数据库失败: %w", err)
	}
	if err := ds.initSessionDb(path); err != nil {
		return nil, fmt.Errorf("初始化会话数据库失败: %w", err)
	}
	if err := ds.initMediaDb(path); err != nil {
		return nil, fmt.Errorf("初始化会话数据库失败: %w", err)
	}

	return ds, nil
}

func (ds *DataSource) initMessageDbs(path string) error {

	files, err := util.FindFilesWithPatterns(path, MessageFilePattern, true)
	if err != nil {
		return fmt.Errorf("查找消息数据库文件失败: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("未找到任何消息数据库文件: %s", path)
	}

	// 处理每个数据库文件
	for _, filePath := range files {
		// 连接数据库
		db, err := sql.Open("sqlite3", filePath)
		if err != nil {
			log.Printf("警告: 连接数据库 %s 失败: %v", filePath, err)
			continue
		}
		ds.messageDbs = append(ds.messageDbs, db)

		// 获取所有表名
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Chat_%'")
		if err != nil {
			log.Printf("警告: 获取表名失败: %v", err)
			continue
		}

		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				log.Printf("警告: 扫描表名失败: %v", err)
				continue
			}

			// 从表名中提取可能的talker信息
			talkerMd5 := extractTalkerFromTableName(tableName)
			if talkerMd5 == "" {
				continue
			}
			ds.talkerDBMap[talkerMd5] = db
		}
		rows.Close()

	}
	return nil
}

func (ds *DataSource) initContactDb(path string) error {

	files, err := util.FindFilesWithPatterns(path, ContactFilePattern, true)
	if err != nil {
		return fmt.Errorf("查找联系人数据库文件失败: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("未找到联系人数据库文件: %s", path)
	}

	ds.contactDb, err = sql.Open("sqlite3", files[0])
	if err != nil {
		return fmt.Errorf("连接联系人数据库失败: %w", err)
	}

	return nil
}

func (ds *DataSource) initChatRoomDb(path string) error {
	files, err := util.FindFilesWithPatterns(path, ChatRoomFilePattern, true)
	if err != nil {
		return fmt.Errorf("查找群聊数据库文件失败: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("未找到群聊数据库文件: %s", path)
	}
	ds.chatRoomDb, err = sql.Open("sqlite3", files[0])
	if err != nil {
		return fmt.Errorf("连接群聊数据库失败: %w", err)
	}

	rows, err := ds.chatRoomDb.Query("SELECT m_nsUsrName, IFNULL(nickname,\"\") FROM GroupMember")
	if err != nil {
		log.Printf("警告: 获取群聊成员失败: %v", err)
		return nil
	}

	for rows.Next() {
		var user string
		var nickName string
		if err := rows.Scan(&user, &nickName); err != nil {
			log.Printf("警告: 扫描表名失败: %v", err)
			continue
		}
		ds.user2DisplayName[user] = nickName
	}
	rows.Close()

	return nil
}

func (ds *DataSource) initSessionDb(path string) error {
	files, err := util.FindFilesWithPatterns(path, SessionFilePattern, true)
	if err != nil {
		return fmt.Errorf("查找最近会话数据库文件失败: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("未找到最近会话数据库文件: %s", path)
	}
	ds.sessionDb, err = sql.Open("sqlite3", files[0])
	if err != nil {
		return fmt.Errorf("连接最近会话数据库失败: %w", err)
	}
	return nil
}

func (ds *DataSource) initMediaDb(path string) error {
	files, err := util.FindFilesWithPatterns(path, MediaFilePattern, true)
	if err != nil {
		return fmt.Errorf("查找媒体数据库文件失败: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("未找到媒体数据库文件: %s", path)
	}
	ds.mediaDb, err = sql.Open("sqlite3", files[0])
	if err != nil {
		return fmt.Errorf("连接媒体数据库失败: %w", err)
	}
	return nil
}

// GetMessages 实现获取消息的方法
func (ds *DataSource) GetMessages(ctx context.Context, startTime, endTime time.Time, talker string, limit, offset int) ([]*model.Message, error) {
	// 在 darwinv3 中，每个联系人/群聊的消息存储在单独的表中，表名为 Chat_md5(talker)
	// 首先需要找到对应的表名
	if talker == "" {
		return nil, fmt.Errorf("talker 不能为空")
	}

	_talkerMd5Bytes := md5.Sum([]byte(talker))
	talkerMd5 := hex.EncodeToString(_talkerMd5Bytes[:])
	db, ok := ds.talkerDBMap[talkerMd5]
	if !ok {
		return nil, fmt.Errorf("未找到 talker %s 的消息数据库", talker)
	}
	tableName := fmt.Sprintf("Chat_%s", talkerMd5)

	// 构建查询条件
	query := fmt.Sprintf(`
		SELECT msgCreateTime, msgContent, messageType, mesDes
		FROM %s 
		WHERE msgCreateTime >= ? AND msgCreateTime <= ? 
		ORDER BY msgCreateTime ASC
	`, tableName)

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)

		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	rows, err := db.QueryContext(ctx, query, startTime.Unix(), endTime.Unix())
	if err != nil {
		return nil, fmt.Errorf("查询表 %s 失败: %w", tableName, err)
	}
	defer rows.Close()

	// 处理查询结果
	messages := []*model.Message{}
	for rows.Next() {
		var msg model.MessageDarwinV3
		err := rows.Scan(
			&msg.MsgCreateTime,
			&msg.MsgContent,
			&msg.MessageType,
			&msg.MesDes,
		)
		if err != nil {
			log.Printf("警告: 扫描消息行失败: %v", err)
			continue
		}

		// 将消息包装为通用模型
		message := msg.Wrap(talker)
		messages = append(messages, message)
	}

	return messages, nil
}

// 从表名中提取 talker
func extractTalkerFromTableName(tableName string) string {

	if !strings.HasPrefix(tableName, "Chat_") {
		return ""
	}

	if strings.HasSuffix(tableName, "_dels") {
		return ""
	}

	return strings.TrimPrefix(tableName, "Chat_")
}

// GetContacts 实现获取联系人信息的方法
func (ds *DataSource) GetContacts(ctx context.Context, key string, limit, offset int) ([]*model.Contact, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), m_uiSex, IFNULL(m_nsAliasName,"") 
				FROM WCContact 
				WHERE m_nsUsrName = ? OR nickname = ? OR m_nsRemark = ? OR m_nsAliasName = ?`
		args = []interface{}{key, key, key, key}
	} else {
		// 查询所有联系人
		query = `SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), m_uiSex, IFNULL(m_nsAliasName,"") 
				FROM WCContact`
	}

	// 添加排序、分页
	query += ` ORDER BY m_nsUsrName`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	rows, err := ds.contactDb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询联系人失败: %w", err)
	}
	defer rows.Close()

	contacts := []*model.Contact{}
	for rows.Next() {
		var contactDarwinV3 model.ContactDarwinV3
		err := rows.Scan(
			&contactDarwinV3.M_nsUsrName,
			&contactDarwinV3.Nickname,
			&contactDarwinV3.M_nsRemark,
			&contactDarwinV3.M_uiSex,
			&contactDarwinV3.M_nsAliasName,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描联系人行失败: %w", err)
		}

		contacts = append(contacts, contactDarwinV3.Wrap())
	}

	return contacts, nil
}

// GetChatRooms 实现获取群聊信息的方法
func (ds *DataSource) GetChatRooms(ctx context.Context, key string, limit, offset int) ([]*model.ChatRoom, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), IFNULL(m_nsChatRoomMemList,""), IFNULL(m_nsChatRoomAdminList,"") 
				FROM GroupContact 
				WHERE m_nsUsrName = ? OR nickname = ? OR m_nsRemark = ?`
		args = []interface{}{key, key, key}
	} else {
		// 查询所有群聊
		query = `SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), IFNULL(m_nsChatRoomMemList,""), IFNULL(m_nsChatRoomAdminList,"") 
				FROM GroupContact`
	}

	// 添加排序、分页
	query += ` ORDER BY m_nsUsrName`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	rows, err := ds.chatRoomDb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询群聊失败: %w", err)
	}
	defer rows.Close()

	chatRooms := []*model.ChatRoom{}
	for rows.Next() {
		var chatRoomDarwinV3 model.ChatRoomDarwinV3
		err := rows.Scan(
			&chatRoomDarwinV3.M_nsUsrName,
			&chatRoomDarwinV3.Nickname,
			&chatRoomDarwinV3.M_nsRemark,
			&chatRoomDarwinV3.M_nsChatRoomMemList,
			&chatRoomDarwinV3.M_nsChatRoomAdminList,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描群聊行失败: %w", err)
		}

		chatRooms = append(chatRooms, chatRoomDarwinV3.Wrap(ds.user2DisplayName))
	}

	// 如果没有找到群聊，尝试通过联系人查找
	if len(chatRooms) == 0 && key != "" {
		contacts, err := ds.GetContacts(ctx, key, 1, 0)
		if err == nil && len(contacts) > 0 && strings.HasSuffix(contacts[0].UserName, "@chatroom") {
			// 再次尝试通过用户名查找群聊
			rows, err := ds.chatRoomDb.QueryContext(ctx,
				`SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), IFNULL(m_nsChatRoomMemList,""), IFNULL(m_nsChatRoomAdminList,"") 
				FROM GroupContact 
				WHERE m_nsUsrName = ?`,
				contacts[0].UserName)

			if err != nil {
				return nil, fmt.Errorf("查询群聊失败: %w", err)
			}
			defer rows.Close()

			for rows.Next() {
				var chatRoomDarwinV3 model.ChatRoomDarwinV3
				err := rows.Scan(
					&chatRoomDarwinV3.M_nsUsrName,
					&chatRoomDarwinV3.Nickname,
					&chatRoomDarwinV3.M_nsRemark,
					&chatRoomDarwinV3.M_nsChatRoomMemList,
					&chatRoomDarwinV3.M_nsChatRoomAdminList,
				)

				if err != nil {
					return nil, fmt.Errorf("扫描群聊行失败: %w", err)
				}

				chatRooms = append(chatRooms, chatRoomDarwinV3.Wrap(ds.user2DisplayName))
			}

			// 如果群聊记录不存在，但联系人记录存在，创建一个模拟的群聊对象
			if len(chatRooms) == 0 {
				chatRooms = append(chatRooms, &model.ChatRoom{
					Name:  contacts[0].UserName,
					Users: make([]model.ChatRoomUser, 0),
				})
			}
		}
	}

	return chatRooms, nil
}

// GetSessions 实现获取会话信息的方法
func (ds *DataSource) GetSessions(ctx context.Context, key string, limit, offset int) ([]*model.Session, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT m_nsUserName, m_uLastTime 
				FROM SessionAbstract 
				WHERE m_nsUserName = ?`
		args = []interface{}{key}
	} else {
		// 查询所有会话
		query = `SELECT m_nsUserName, m_uLastTime 
				FROM SessionAbstract`
	}

	// 添加排序、分页
	query += ` ORDER BY m_uLastTime DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	rows, err := ds.sessionDb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}
	defer rows.Close()

	sessions := []*model.Session{}
	for rows.Next() {
		var sessionDarwinV3 model.SessionDarwinV3
		err := rows.Scan(
			&sessionDarwinV3.M_nsUserName,
			&sessionDarwinV3.M_uLastTime,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描会话行失败: %w", err)
		}

		// 包装成通用模型
		session := sessionDarwinV3.Wrap()

		// 尝试获取联系人信息以补充会话信息
		contacts, err := ds.GetContacts(ctx, session.UserName, 1, 0)
		if err == nil && len(contacts) > 0 {
			session.NickName = contacts[0].DisplayName()
		} else {
			// 尝试获取群聊信息
			chatRooms, err := ds.GetChatRooms(ctx, session.UserName, 1, 0)
			if err == nil && len(chatRooms) > 0 {
				session.NickName = chatRooms[0].DisplayName()
			}
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (ds *DataSource) GetMedia(ctx context.Context, _type string, key string) (*model.Media, error) {
	if key == "" {
		return nil, fmt.Errorf("key 不能为空")
	}
	query := `SELECT 
    r.mediaMd5,
    r.mediaSize,
    r.inodeNumber,
    r.modifyTime,
    d.relativePath,
    d.fileName
FROM 
    HlinkMediaRecord r
JOIN 
    HlinkMediaDetail d ON r.inodeNumber = d.inodeNumber
WHERE 
    r.mediaMd5 = ?`
	args := []interface{}{key}
	// 执行查询
	rows, err := ds.mediaDb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询媒体失败: %w", err)
	}
	defer rows.Close()

	var media *model.Media
	for rows.Next() {
		var mediaDarwinV3 model.MediaDarwinV3
		err := rows.Scan(
			&mediaDarwinV3.MediaMd5,
			&mediaDarwinV3.MediaSize,
			&mediaDarwinV3.InodeNumber,
			&mediaDarwinV3.ModifyTime,
			&mediaDarwinV3.RelativePath,
			&mediaDarwinV3.FileName,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描会话行失败: %w", err)
		}

		// 包装成通用模型
		media = mediaDarwinV3.Wrap()
	}

	if media == nil {
		return nil, fmt.Errorf("未找到媒体 %s", key)
	}

	return media, nil
}

// Close 实现关闭数据库连接的方法
func (ds *DataSource) Close() error {
	var errs []error

	// 关闭消息数据库连接
	for i, db := range ds.messageDbs {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭消息数据库 %d 失败: %w", i, err))
		}
	}

	// 关闭联系人数据库连接
	if ds.contactDb != nil {
		if err := ds.contactDb.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭联系人数据库失败: %w", err))
		}
	}

	// 关闭群聊数据库连接
	if ds.chatRoomDb != nil {
		if err := ds.chatRoomDb.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭群聊数据库失败: %w", err))
		}
	}

	// 关闭会话数据库连接
	if ds.sessionDb != nil {
		if err := ds.sessionDb.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭会话数据库失败: %w", err))
		}
	}

	// 关闭媒体数据库连接
	if ds.mediaDb != nil {
		if err := ds.mediaDb.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭媒体数据库失败: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭数据库连接时发生错误: %v", errs)
	}

	return nil
}

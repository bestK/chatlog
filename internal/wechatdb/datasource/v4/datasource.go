package v4

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "modernc.org/sqlite"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb/datasource/dbm"
	"github.com/sjzar/chatlog/pkg/util"
)

const (
	Message = "message"
	Contact = "contact"
	Session = "session"
	Media   = "media"
	Voice   = "voice"
)

var Groups = []*dbm.Group{
	{
		Name:      Message,
		Pattern:   `^message_([0-9]?[0-9])?\.db$`,
		BlackList: []string{},
	},
	{
		Name:      Contact,
		Pattern:   `^contact\.db$`,
		BlackList: []string{},
	},
	{
		Name:      Session,
		Pattern:   `session\.db$`,
		BlackList: []string{},
	},
	{
		Name:      Media,
		Pattern:   `^hardlink\.db$`,
		BlackList: []string{},
	},
	{
		Name:      Voice,
		Pattern:   `^media_([0-9]?[0-9])?\.db$`,
		BlackList: []string{},
	},
}

// MessageDBInfo 存储消息数据库的信息
type MessageDBInfo struct {
	FilePath  string
	StartTime time.Time
	EndTime   time.Time
}

type DataSource struct {
	path string
	dbm  *dbm.DBManager

	// 消息数据库信息
	messageInfos []MessageDBInfo

	// 联系人缓存
	contactCache map[string]string
}

func New(path string) (*DataSource, error) {

	ds := &DataSource{
		path:         path,
		dbm:          dbm.NewDBManager(path),
		messageInfos: make([]MessageDBInfo, 0),
		contactCache: make(map[string]string),
	}

	for _, g := range Groups {
		ds.dbm.AddGroup(g)
	}

	if err := ds.dbm.Start(); err != nil {
		return nil, err
	}

	if err := ds.initMessageDbs(); err != nil {
		return nil, errors.DBInitFailed(err)
	}

	if err := ds.initContactCache(); err != nil {
		log.Err(err).Msg("Failed to initialize contact cache")
	}

	ds.dbm.AddCallback(Message, func(event fsnotify.Event) error {
		if !(event.Op.Has(fsnotify.Create) || event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Rename)) {
			return nil
		}
		if err := ds.initMessageDbs(); err != nil {
			log.Err(err).Msgf("Failed to reinitialize message DBs: %s", event.Name)
		}
		return nil
	})

	ds.dbm.AddCallback(Contact, func(event fsnotify.Event) error {
		if !(event.Op.Has(fsnotify.Create) || event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Rename)) {
			return nil
		}
		if err := ds.initContactCache(); err != nil {
			log.Err(err).Msgf("Failed to reinitialize contact cache: %s", event.Name)
		}
		return nil
	})

	return ds, nil
}

func (ds *DataSource) SetCallback(group string, callback func(event fsnotify.Event) error) error {
	if group == "chatroom" {
		group = Contact
	}
	return ds.dbm.AddCallback(group, callback)
}

func (ds *DataSource) RemoveCallback(group string, callback func(event fsnotify.Event) error) bool {
	if group == "chatroom" {
		group = Contact
	}
	return ds.dbm.RemoveCallback(group, callback)
}

func (ds *DataSource) initMessageDbs() error {
	dbPaths, err := ds.dbm.GetDBPath(Message)
	if err != nil {
		if strings.Contains(err.Error(), "db file not found") {
			ds.messageInfos = make([]MessageDBInfo, 0)
			return nil
		}
		return err
	}

	// 处理每个数据库文件
	infos := make([]MessageDBInfo, 0)
	for _, filePath := range dbPaths {
		db, err := ds.dbm.OpenDB(filePath)
		if err != nil {
			log.Err(err).Msgf("获取数据库 %s 失败", filePath)
			continue
		}
		// 不需要 defer，直接在循环末尾关闭，或者使用 func 闭包。
		// 在这里直接使用 defer 可能会导致直到 initMessageDbs 结束才关闭。
		// 更好的做法是显式调用 Close。

		// 获取 Timestamp 表中的开始时间
		var startTime time.Time
		var timestamp int64

		row := db.QueryRow("SELECT timestamp FROM Timestamp LIMIT 1")
		if err := row.Scan(&timestamp); err != nil {
			log.Err(err).Msgf("获取数据库 %s 的时间戳失败", filePath)
			continue
		}
		startTime = time.Unix(timestamp, 0)

		// 保存数据库信息
		infos = append(infos, MessageDBInfo{
			FilePath:  filePath,
			StartTime: startTime,
		})

		db.Close()
	}

	// 按照 StartTime 排序数据库文件
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].StartTime.Before(infos[j].StartTime)
	})

	// 设置结束时间
	for i := range infos {
		if i == len(infos)-1 {
			infos[i].EndTime = time.Now().Add(time.Hour)
		} else {
			infos[i].EndTime = infos[i+1].StartTime
		}
	}
	if len(ds.messageInfos) > 0 && len(infos) < len(ds.messageInfos) {
		log.Warn().Msgf("message db count decreased from %d to %d, skip init", len(ds.messageInfos), len(infos))
		return nil
	}
	ds.messageInfos = infos
	return nil
}

func (ds *DataSource) initContactCache() error {
	db, err := ds.dbm.GetDB(Contact)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query("SELECT username, IFNULL(nick_name, IFNULL(remark, username)) FROM contact")
	if err != nil {
		return err
	}
	defer rows.Close()

	cache := make(map[string]string)
	for rows.Next() {
		var username, displayName string
		if err := rows.Scan(&username, &displayName); err != nil {
			continue
		}
		cache[username] = displayName
	}
	ds.contactCache = cache
	return nil
}

// getDBInfosForTimeRange 获取时间范围内的数据库信息
func (ds *DataSource) getDBInfosForTimeRange(startTime, endTime time.Time) []MessageDBInfo {
	var dbs []MessageDBInfo
	for _, info := range ds.messageInfos {
		if info.StartTime.Before(endTime) && info.EndTime.After(startTime) {
			dbs = append(dbs, info)
		}
	}
	return dbs
}

func (ds *DataSource) GetMessages(ctx context.Context, startTime, endTime time.Time, selfID string, talker string, sender string, keyword string, limit, offset int, order string) ([]*model.Message, error) {
	if talker == "" {
		return nil, errors.ErrTalkerEmpty
	}

	// 解析talker参数，支持多个talker（以英文逗号分隔）
	talkers := util.Str2List(talker, ",")
	if len(talkers) == 0 {
		return nil, errors.ErrTalkerEmpty
	}

	// 找到时间范围内的数据库文件
	dbInfos := ds.getDBInfosForTimeRange(startTime, endTime)
	if len(dbInfos) == 0 {
		return nil, errors.TimeRangeNotFound(startTime, endTime)
	}

	// 解析sender参数，支持多个发送者（以英文逗号分隔）
	senders := util.Str2List(sender, ",")

	// 预编译正则表达式（如果有keyword）
	var regex *regexp.Regexp
	if keyword != "" {
		var err error
		regex, err = regexp.Compile(keyword)
		if err != nil {
			return nil, errors.QueryFailed("invalid regex pattern", err)
		}
	}

	// 确定 SQL 排序方向
	sqlOrder := "ASC"
	if order == "desc" {
		sqlOrder = "DESC"
	}

	// desc 时从最新的数据库文件开始查，asc 时从最旧的开始
	if order == "desc" {
		for i, j := 0, len(dbInfos)-1; i < j; i, j = i+1, j-1 {
			dbInfos[i], dbInfos[j] = dbInfos[j], dbInfos[i]
		}
	}

	// 需要收集的总数量
	needed := 0
	if limit > 0 {
		needed = offset + limit
	}

	// 从每个相关数据库中查询消息
	filteredMessages := []*model.Message{}

	for _, dbInfo := range dbInfos {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// 如果已经收集够了，提前退出
		if needed > 0 && len(filteredMessages) >= needed {
			break
		}

		db, err := ds.dbm.OpenDB(dbInfo.FilePath)
		if err != nil {
			continue
		}

		func() {
			defer db.Close()
			for _, talkerItem := range talkers {
				_talkerMd5Bytes := md5.Sum([]byte(talkerItem))
				talkerMd5 := hex.EncodeToString(_talkerMd5Bytes[:])
				tableName := "Msg_" + talkerMd5

				var exists bool
				err = db.QueryRowContext(ctx,
					"SELECT 1 FROM sqlite_master WHERE type='table' AND name=?",
					tableName).Scan(&exists)
				if err != nil {
					continue
				}

				conditions := []string{"create_time >= ? AND create_time <= ?"}
				args := []interface{}{startTime.Unix(), endTime.Unix()}

				// 在 SQL 层面加 LIMIT 减少读取量
				sqlLimit := ""
				if needed > 0 {
					remaining := needed - len(filteredMessages)
					if remaining > 0 {
						sqlLimit = fmt.Sprintf(" LIMIT %d", remaining)
					}
				}

				query := fmt.Sprintf(`
					SELECT m.sort_seq, m.server_id, m.local_type, IFNULL(n.user_name, ''), m.create_time, m.message_content, m.packed_info_data, m.status
					FROM %s m
					LEFT JOIN Name2Id n ON m.real_sender_id = n.rowid
					WHERE %s
					ORDER BY m.sort_seq %s%s
				`, tableName, strings.Join(conditions, " AND "), sqlOrder, sqlLimit)

				rows, err := db.QueryContext(ctx, query, args...)
				if err != nil {
					continue
				}

				for rows.Next() {
					var msg model.MessageV4
					err := rows.Scan(
						&msg.SortSeq,
						&msg.ServerID,
						&msg.LocalType,
						&msg.UserName,
						&msg.CreateTime,
						&msg.MessageContent,
						&msg.PackedInfoData,
						&msg.Status,
					)
					if err != nil {
						rows.Close()
						return
					}

					msg.SelfID = selfID
					msg.SenderName = ds.contactCache[msg.UserName]
					message := msg.Wrap(talkerItem)

					if len(senders) > 0 {
						senderMatch := false
						for _, s := range senders {
							if message.Sender == s {
								senderMatch = true
								break
							}
						}
						if !senderMatch {
							continue
						}
					}

					if regex != nil {
						if !regex.MatchString(message.PlainTextContent()) {
							continue
						}
					}

					filteredMessages = append(filteredMessages, message)

					// 单库内如果已经够了就停止读取
					if needed > 0 && len(filteredMessages) >= needed {
						rows.Close()
						return
					}
				}
				rows.Close()
			}
		}()
	}

	// 跨多个数据库文件时需要重新排序
	if len(dbInfos) > 1 {
		if order == "asc" {
			sort.Slice(filteredMessages, func(i, j int) bool {
				return filteredMessages[i].Seq < filteredMessages[j].Seq
			})
		} else {
			sort.Slice(filteredMessages, func(i, j int) bool {
				return filteredMessages[i].Seq > filteredMessages[j].Seq
			})
		}
	}

	// 处理分页
	if limit > 0 {
		if offset >= len(filteredMessages) {
			return []*model.Message{}, nil
		}
		end := offset + limit
		if end > len(filteredMessages) {
			end = len(filteredMessages)
		}
		return filteredMessages[offset:end], nil
	}

	return filteredMessages, nil
}

func (ds *DataSource) GetMessagesCount(ctx context.Context, startTime, endTime time.Time, speakerto string, talker string, sender string, keyword string) (int, error) {
	messages, err := ds.GetMessages(ctx, startTime, endTime, speakerto, talker, sender, keyword, 0, 0, "")
	if err != nil {
		return 0, err
	}
	return len(messages), nil
}

// 联系人
func (ds *DataSource) GetContacts(ctx context.Context, key string, limit, offset int) ([]*model.Contact, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT username, local_type, flag, delete_flag, IFNULL(is_in_chat_room,0), alias, remark, nick_name, IFNULL(small_head_url,''), IFNULL(big_head_url,'')
				FROM contact 
				WHERE username = ? OR alias = ? OR remark = ? OR nick_name = ?`
		args = []interface{}{key, key, key, key}
	} else {
		// 查询所有联系人
		query = `SELECT username, local_type, flag, delete_flag, IFNULL(is_in_chat_room,0), alias, remark, nick_name, IFNULL(small_head_url,''), IFNULL(big_head_url,'') FROM contact`
	}

	// 添加排序、分页
	query += ` ORDER BY username`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	db, err := ds.dbm.GetDB(Contact)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.QueryFailed(query, err)
	}
	defer rows.Close()

	contacts := []*model.Contact{}
	for rows.Next() {
		var contactV4 model.ContactV4
		err := rows.Scan(
			&contactV4.UserName,
			&contactV4.LocalType,
			&contactV4.Flag,
			&contactV4.DeleteFlag,
			&contactV4.IsInChatRoom,
			&contactV4.Alias,
			&contactV4.Remark,
			&contactV4.NickName,
			&contactV4.SmallHeadUrl,
			&contactV4.BigHeadUrl,
		)

		if err != nil {
			return nil, errors.ScanRowFailed(err)
		}

		contacts = append(contacts, contactV4.Wrap())
	}

	return contacts, nil
}

func (ds *DataSource) GetContactsCount(ctx context.Context, key string) (int, error) {
	var query string
	var args []interface{}

	if key != "" {
		query = `SELECT COUNT(*) FROM contact WHERE username = ? OR alias = ? OR remark = ? OR nick_name = ?`
		args = []interface{}{key, key, key, key}
	} else {
		query = `SELECT COUNT(*) FROM contact`
	}

	db, err := ds.dbm.GetDB(Contact)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	err = db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, errors.QueryFailed(query, err)
	}

	return count, nil
}

func (ds *DataSource) GetAddressBookContacts(ctx context.Context, key string, isInChatRoom, limit, offset int) ([]*model.Contact, error) {
	query := `SELECT username, local_type, flag, delete_flag, IFNULL(is_in_chat_room,0), alias, remark, nick_name, IFNULL(small_head_url,''), IFNULL(big_head_url,'')
			FROM contact
			WHERE flag in (2,3,2051) and delete_flag = 0
			  AND NOT (username LIKE '%@chatroom' AND IFNULL(is_in_chat_room,0) = 0)`
	args := make([]interface{}, 0, 4)
	if isInChatRoom >= 0 {
		query += ` AND IFNULL(is_in_chat_room,0) = ?`
		args = append(args, isInChatRoom)
	}
	if strings.TrimSpace(key) != "" {
		query += ` AND (
			username LIKE '%' || ? || '%' OR
			alias LIKE '%' || ? || '%' OR
			remark LIKE '%' || ? || '%' OR
			nick_name LIKE '%' || ? || '%'
		)`
		args = append(args, key, key, key, key)
	}
	query += ` ORDER BY username`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	db, err := ds.dbm.GetDB(Contact)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.QueryFailed(query, err)
	}
	defer rows.Close()

	contacts := make([]*model.Contact, 0)
	for rows.Next() {
		var contactV4 model.ContactV4
		err := rows.Scan(
			&contactV4.UserName,
			&contactV4.LocalType,
			&contactV4.Flag,
			&contactV4.DeleteFlag,
			&contactV4.IsInChatRoom,
			&contactV4.Alias,
			&contactV4.Remark,
			&contactV4.NickName,
			&contactV4.SmallHeadUrl,
			&contactV4.BigHeadUrl,
		)
		if err != nil {
			return nil, errors.ScanRowFailed(err)
		}
		contacts = append(contacts, contactV4.Wrap())
	}
	return contacts, nil
}

func (ds *DataSource) GetAddressBookContactsCount(ctx context.Context, key string, isInChatRoom int) (int, error) {
	query := `SELECT COUNT(*)
			FROM contact
			WHERE flag in (2,3,2051) and delete_flag = 0
			  AND NOT (username LIKE '%@chatroom' AND IFNULL(is_in_chat_room,0) = 0)`
	args := make([]interface{}, 0, 4)
	if isInChatRoom >= 0 {
		query += ` AND IFNULL(is_in_chat_room,0) = ?`
		args = append(args, isInChatRoom)
	}
	if strings.TrimSpace(key) != "" {
		query += ` AND (
			username LIKE '%' || ? || '%' OR
			alias LIKE '%' || ? || '%' OR
			remark LIKE '%' || ? || '%' OR
			nick_name LIKE '%' || ? || '%'
		)`
		args = append(args, key, key, key, key)
	}

	db, err := ds.dbm.GetDB(Contact)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, errors.QueryFailed(query, err)
	}
	return count, nil
}

// 群聊
func (ds *DataSource) GetChatRooms(ctx context.Context, key string, limit, offset int) ([]*model.ChatRoom, error) {
	var query string
	var args []interface{}

	// 执行查询
	db, err := ds.dbm.GetDB(Contact)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if key != "" {
		// 按照关键字查询
		query = `SELECT username, owner, ext_buffer FROM chat_room WHERE username = ?`
		args = []interface{}{key}

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, errors.QueryFailed(query, err)
		}
		defer rows.Close()

		chatRooms := []*model.ChatRoom{}
		for rows.Next() {
			var chatRoomV4 model.ChatRoomV4
			err := rows.Scan(
				&chatRoomV4.UserName,
				&chatRoomV4.Owner,
				&chatRoomV4.ExtBuffer,
			)

			if err != nil {
				return nil, errors.ScanRowFailed(err)
			}

			chatRooms = append(chatRooms, chatRoomV4.Wrap())
		}

		// 如果没有找到群聊，尝试通过联系人查找
		if len(chatRooms) == 0 {
			contacts, err := ds.GetContacts(ctx, key, 1, 0)
			if err == nil && len(contacts) > 0 && strings.HasSuffix(contacts[0].UserName, "@chatroom") {
				// 再次尝试通过用户名查找群聊
				rows, err := db.QueryContext(ctx,
					`SELECT username, owner, ext_buffer FROM chat_room WHERE username = ?`,
					contacts[0].UserName)

				if err != nil {
					return nil, errors.QueryFailed(query, err)
				}
				defer rows.Close()

				for rows.Next() {
					var chatRoomV4 model.ChatRoomV4
					err := rows.Scan(
						&chatRoomV4.UserName,
						&chatRoomV4.Owner,
						&chatRoomV4.ExtBuffer,
					)

					if err != nil {
						return nil, errors.ScanRowFailed(err)
					}

					chatRooms = append(chatRooms, chatRoomV4.Wrap())
				}

				// 如果群聊记录不存在，但联系人记录存在，创建一个模拟的群聊对象
				if len(chatRooms) == 0 {
					chatRooms = append(chatRooms, &model.ChatRoom{
						Name:             contacts[0].UserName,
						Users:            make([]model.ChatRoomUser, 0),
						User2DisplayName: make(map[string]string),
					})
				}
			}
		}

		return chatRooms, nil
	} else {
		// 查询所有群聊
		query = `SELECT username, owner, ext_buffer FROM chat_room`

		// 添加排序、分页
		query += ` ORDER BY username`
		if limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", limit)
			if offset > 0 {
				query += fmt.Sprintf(" OFFSET %d", offset)
			}
		}

		// 执行查询
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, errors.QueryFailed(query, err)
		}
		defer rows.Close()

		chatRooms := []*model.ChatRoom{}
		for rows.Next() {
			var chatRoomV4 model.ChatRoomV4
			err := rows.Scan(
				&chatRoomV4.UserName,
				&chatRoomV4.Owner,
				&chatRoomV4.ExtBuffer,
			)

			if err != nil {
				return nil, errors.ScanRowFailed(err)
			}

			chatRooms = append(chatRooms, chatRoomV4.Wrap())
		}

		return chatRooms, nil
	}
}

func (ds *DataSource) GetChatRoomsCount(ctx context.Context, key string) (int, error) {
	// 由于 GetChatRooms 逻辑包含从联系人表中模拟群聊，这里直接复用 GetChatRooms 逻辑取长度
	// 对于群聊数量来说，量级通常较小，直接取列表长度尚可接受
	rooms, err := ds.GetChatRooms(ctx, key, 0, 0)
	if err != nil {
		return 0, err
	}
	return len(rooms), nil
}

// 最近会话
func (ds *DataSource) GetSessions(ctx context.Context, key string, limit, offset int) ([]*model.Session, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT username, summary, last_timestamp, last_msg_sender, last_sender_display_name, last_msg_type, last_msg_sub_type, status, IFNULL(last_msg_locald_id, 0)
				FROM SessionTable 
				WHERE username LIKE '%' || ? || '%' 
				   OR last_sender_display_name LIKE '%' || ? || '%' 
				   OR summary LIKE '%' || ? || '%'
				ORDER BY sort_timestamp DESC`
		args = []interface{}{key, key, key}
	} else {
		// 查询所有会话
		query = `SELECT username, summary, last_timestamp, last_msg_sender, last_sender_display_name, last_msg_type, last_msg_sub_type, status, IFNULL(last_msg_locald_id, 0)
				FROM SessionTable 
				ORDER BY sort_timestamp DESC`
	}

	// 添加分页
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	db, err := ds.dbm.GetDB(Session)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.QueryFailed(query, err)
	}
	defer rows.Close()

	sessions := []*model.Session{}
	for rows.Next() {
		var sessionV4 model.SessionV4
		err := rows.Scan(
			&sessionV4.Username,
			&sessionV4.Summary,
			&sessionV4.LastTimestamp,
			&sessionV4.LastMsgSender,
			&sessionV4.LastSenderDisplayName,
			&sessionV4.LastMsgType,
			&sessionV4.LastMsgSubType,
			&sessionV4.Status,
			&sessionV4.LastMsgLocaldID,
		)

		if err != nil {
			return nil, errors.ScanRowFailed(err)
		}

		sessions = append(sessions, sessionV4.Wrap())
	}

	return sessions, nil
}

func (ds *DataSource) GetSessionsCount(ctx context.Context, key string) (int, error) {
	var query string
	var args []interface{}

	if key != "" {
		query = `SELECT COUNT(*) FROM SessionTable 
				WHERE username LIKE '%' || ? || '%' 
				   OR last_sender_display_name LIKE '%' || ? || '%' 
				   OR summary LIKE '%' || ? || '%'`
		args = []interface{}{key, key, key}
	} else {
		query = `SELECT COUNT(*) FROM SessionTable`
	}

	db, err := ds.dbm.GetDB(Session)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	err = db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, errors.QueryFailed(query, err)
	}

	return count, nil
}

func (ds *DataSource) GetMedia(ctx context.Context, _type string, key string) (*model.Media, error) {
	if key == "" {
		return nil, errors.ErrKeyEmpty
	}

	var table string
	switch _type {
	case "image":
		table = "image_hardlink_info_v3"
		// 4.1.0 版本开始使用 v4 表
		if !ds.IsExist(Media, table) {
			table = "image_hardlink_info_v4"
		}
	case "video":
		table = "video_hardlink_info_v3"
		if !ds.IsExist(Media, table) {
			table = "video_hardlink_info_v4"
		}
	case "file":
		table = "file_hardlink_info_v3"
		if !ds.IsExist(Media, table) {
			table = "file_hardlink_info_v4"
		}
	case "voice":
		return ds.GetVoice(ctx, key)
	default:
		return nil, errors.MediaTypeUnsupported(_type)
	}

	query := fmt.Sprintf(`
	SELECT 
		f.md5,
		f.file_name,
		f.file_size,
		f.modify_time,
		IFNULL(d1.username,""),
		IFNULL(d2.username,"")
	FROM 
		%s f
	LEFT JOIN 
		dir2id d1 ON d1.rowid = f.dir1
	LEFT JOIN 
		dir2id d2 ON d2.rowid = f.dir2
	`, table)
	query += " WHERE f.md5 = ? OR f.file_name LIKE ? || '%'"
	args := []interface{}{key, key}

	// 执行查询
	db, err := ds.dbm.GetDB(Media)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, query, args...)

	var media *model.Media
	for rows.Next() {
		var mediaV4 model.MediaV4
		err := rows.Scan(
			&mediaV4.Key,
			&mediaV4.Name,
			&mediaV4.Size,
			&mediaV4.ModifyTime,
			&mediaV4.Dir1,
			&mediaV4.Dir2,
		)
		if err != nil {
			return nil, errors.ScanRowFailed(err)
		}
		mediaV4.Type = _type
		media = mediaV4.Wrap()

		// 优先返回高清图
		if _type == "image" && strings.HasSuffix(mediaV4.Name, "_h.dat") {
			break
		}
	}

	if media == nil {
		return nil, errors.ErrMediaNotFound
	}

	return media, nil
}

func (ds *DataSource) IsExist(_db string, table string) bool {
	db, err := ds.dbm.GetDB(_db)
	if err != nil {
		return false
	}
	defer db.Close()
	var tableName string
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?;"
	if err = db.QueryRow(query, table).Scan(&tableName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		return false
	}
	return true
}

func (ds *DataSource) GetVoice(ctx context.Context, key string) (*model.Media, error) {
	if key == "" {
		return nil, errors.ErrKeyEmpty
	}

	query := `
	SELECT voice_data
	FROM VoiceInfo
	WHERE svr_id = ? 
	`
	args := []interface{}{key}

	dbs, err := ds.dbm.GetDBs(Voice)
	if err != nil {
		return nil, errors.DBConnectFailed("", err)
	}
	defer func() {
		for _, db := range dbs {
			db.Close()
		}
	}()

	for _, db := range dbs {
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			log.Err(err).Msgf("Query media failed")
			continue
		}

		for rows.Next() {
			var voiceData []byte
			err := rows.Scan(
				&voiceData,
			)
			if err != nil {
				rows.Close() // 遇到错误关闭
				return nil, errors.ScanRowFailed(err)
			}
			if len(voiceData) > 0 {
				rows.Close() // 找到结果关闭
				return &model.Media{
					Type: "voice",
					Key:  key,
					Data: voiceData,
				}, nil
			}
		}
		rows.Close() // 未找到结果关闭
	}

	return nil, errors.ErrMediaNotFound
}

func (ds *DataSource) Close() error {
	return ds.dbm.Close()
}

func (ds *DataSource) CloseDB(path string) error {
	ds.dbm.CloseDB(path)
	return nil
}

func (ds *DataSource) LockDB(path string) error {
	ds.dbm.LockDB(path)
	return nil
}

func (ds *DataSource) UnlockDB(path string) error {
	ds.dbm.UnlockDB(path)
	return nil
}

// GetSenderByLocalID 通过 topicID 和 localID 查询消息表获取发送人 username
// 表名格式: Msg_ + md5(topicID)
// 查询: 通过 local_id 获取 real_sender_id，再从 Name2Id 表获取 username
func (ds *DataSource) GetSenderByLocalID(ctx context.Context, topicID string, localID int) (string, error) {
	if topicID == "" || localID == 0 {
		return "", nil
	}

	res, err := ds.GetSendersByLocalIDs(ctx, []model.SenderRequest{{TopicID: topicID, LocalID: localID}})
	if err != nil {
		return "", err
	}
	return res[model.SenderRequest{TopicID: topicID, LocalID: localID}], nil
}

// GetSendersByLocalIDs 批量通过 topicID 和 localID 查询发送人 username
func (ds *DataSource) GetSendersByLocalIDs(ctx context.Context, requests []model.SenderRequest) (map[model.SenderRequest]string, error) {
	result := make(map[model.SenderRequest]string)
	if len(requests) == 0 {
		return result, nil
	}

	// 按 TopicID 分组请求，减少重复计算 MD5 和构建查询
	groups := make(map[string][]int)
	for _, req := range requests {
		groups[req.TopicID] = append(groups[req.TopicID], req.LocalID)
	}

	// 找到 message_0.db
	var targetDBPath string
	for _, dbInfo := range ds.messageInfos {
		if strings.Contains(dbInfo.FilePath, "message_0.db") {
			targetDBPath = dbInfo.FilePath
			break
		}
	}

	if targetDBPath == "" {
		return result, nil
	}

	db, err := ds.dbm.OpenDB(targetDBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	for topicID, localIDs := range groups {
		if len(localIDs) == 0 {
			continue
		}

		_topicMd5Bytes := md5.Sum([]byte(topicID))
		topicMd5 := hex.EncodeToString(_topicMd5Bytes[:])
		tableName := "Msg_" + topicMd5

		// 构建 IN 子句
		placeholders := make([]string, len(localIDs))
		args := make([]interface{}, len(localIDs))
		for i, id := range localIDs {
			placeholders[i] = "?"
			args[i] = id
		}

		query := fmt.Sprintf(`
			SELECT m.local_id, IFNULL(n.user_name, '')
			FROM %s m
			LEFT JOIN Name2Id n ON m.real_sender_id = n.rowid
			WHERE m.local_id IN (%s)
		`, tableName, strings.Join(placeholders, ","))

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			log.Debug().Err(err).Str("topicID", topicID).Msg("Batch query senders failed")
			continue
		}

		for rows.Next() {
			var localID int
			var username string
			if err := rows.Scan(&localID, &username); err == nil {
				result[model.SenderRequest{TopicID: topicID, LocalID: localID}] = username
			}
		}
		rows.Close()
	}

	return result, nil
}

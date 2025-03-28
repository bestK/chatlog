package v4

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/pkg/util"

	_ "github.com/mattn/go-sqlite3"
)

const (
	MessageFilePattern = "^message_([0-9]?[0-9])?\\.db$"
	ContactFilePattern = "^contact\\.db$"
	SessionFilePattern = "^session\\.db$"
	MediaFilePattern   = "^hardlink\\.db$"
)

// MessageDBInfo 存储消息数据库的信息
type MessageDBInfo struct {
	FilePath  string
	StartTime time.Time
	EndTime   time.Time
	ID2Name   map[int]string
}

type DataSource struct {
	path       string
	messageDbs map[string]*sql.DB
	contactDb  *sql.DB
	sessionDb  *sql.DB
	mediaDb    *sql.DB

	// 消息数据库信息
	messageFiles []MessageDBInfo
}

func New(path string) (*DataSource, error) {
	ds := &DataSource{
		path:         path,
		messageDbs:   make(map[string]*sql.DB),
		messageFiles: make([]MessageDBInfo, 0),
	}

	if err := ds.initMessageDbs(path); err != nil {
		return nil, fmt.Errorf("初始化消息数据库失败: %w", err)
	}
	if err := ds.initContactDb(path); err != nil {
		return nil, fmt.Errorf("初始化联系人数据库失败: %w", err)
	}
	if err := ds.initSessionDb(path); err != nil {
		return nil, fmt.Errorf("初始化会话数据库失败: %w", err)
	}
	if err := ds.initMediaDb(path); err != nil {
		return nil, fmt.Errorf("初始化媒体数据库失败: %w", err)
	}

	return ds, nil
}

func (ds *DataSource) initMessageDbs(path string) error {
	// 查找所有消息数据库文件
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

		// 获取 Timestamp 表中的开始时间
		var startTime time.Time
		var timestamp int64

		row := db.QueryRow("SELECT timestamp FROM Timestamp LIMIT 1")
		if err := row.Scan(&timestamp); err != nil {
			log.Printf("警告: 获取数据库 %s 的时间戳失败: %v", filePath, err)
			db.Close()
			continue
		}
		startTime = time.Unix(timestamp, 0)

		// 获取 ID2Name 映射
		id2Name := make(map[int]string)
		rows, err := db.Query("SELECT user_name FROM Name2Id")
		if err != nil {
			log.Printf("警告: 获取数据库 %s 的 Name2Id 表失败: %v", filePath, err)
			db.Close()
			continue
		}

		i := 1
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				log.Printf("警告: 扫描 Name2Id 行失败: %v", err)
				continue
			}
			id2Name[i] = name
			i++
		}
		rows.Close()

		// 保存数据库信息
		ds.messageFiles = append(ds.messageFiles, MessageDBInfo{
			FilePath:  filePath,
			StartTime: startTime,
			ID2Name:   id2Name,
		})

		// 保存数据库连接
		ds.messageDbs[filePath] = db
	}

	// 按照 StartTime 排序数据库文件
	sort.Slice(ds.messageFiles, func(i, j int) bool {
		return ds.messageFiles[i].StartTime.Before(ds.messageFiles[j].StartTime)
	})

	// 设置结束时间
	for i := range ds.messageFiles {
		if i == len(ds.messageFiles)-1 {
			ds.messageFiles[i].EndTime = time.Now()
		} else {
			ds.messageFiles[i].EndTime = ds.messageFiles[i+1].StartTime
		}
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

// getDBInfosForTimeRange 获取时间范围内的数据库信息
func (ds *DataSource) getDBInfosForTimeRange(startTime, endTime time.Time) []MessageDBInfo {
	var dbs []MessageDBInfo
	for _, info := range ds.messageFiles {
		if info.StartTime.Before(endTime) && info.EndTime.After(startTime) {
			dbs = append(dbs, info)
		}
	}
	return dbs
}

func (ds *DataSource) GetMessages(ctx context.Context, startTime, endTime time.Time, talker string, limit, offset int) ([]*model.Message, error) {
	if talker == "" {
		return nil, fmt.Errorf("必须指定 talker 参数")
	}

	// 找到时间范围内的数据库文件
	dbInfos := ds.getDBInfosForTimeRange(startTime, endTime)
	if len(dbInfos) == 0 {
		return nil, fmt.Errorf("未找到时间范围 %v 到 %v 内的数据库文件", startTime, endTime)
	}

	if len(dbInfos) == 1 {
		// LIMIT 和 OFFSET 逻辑在单文件情况下可以直接在 SQL 里处理
		return ds.getMessagesSingleFile(ctx, dbInfos[0], startTime, endTime, talker, limit, offset)
	}

	// 从每个相关数据库中查询消息
	totalMessages := []*model.Message{}

	for _, dbInfo := range dbInfos {
		// 检查上下文是否已取消
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		db, ok := ds.messageDbs[dbInfo.FilePath]
		if !ok {
			log.Printf("警告: 数据库 %s 未打开", dbInfo.FilePath)
			continue
		}

		messages, err := ds.getMessagesFromDB(ctx, db, dbInfo, startTime, endTime, talker)
		if err != nil {
			log.Printf("警告: 从数据库 %s 获取消息失败: %v", dbInfo.FilePath, err)
			continue
		}

		totalMessages = append(totalMessages, messages...)

		if limit+offset > 0 && len(totalMessages) >= limit+offset {
			break
		}
	}

	// 对所有消息按时间排序
	sort.Slice(totalMessages, func(i, j int) bool {
		return totalMessages[i].Sequence < totalMessages[j].Sequence
	})

	// 处理分页
	if limit > 0 {
		if offset >= len(totalMessages) {
			return []*model.Message{}, nil
		}
		end := offset + limit
		if end > len(totalMessages) {
			end = len(totalMessages)
		}
		return totalMessages[offset:end], nil
	}

	return totalMessages, nil
}

// getMessagesSingleFile 从单个数据库文件获取消息
func (ds *DataSource) getMessagesSingleFile(ctx context.Context, dbInfo MessageDBInfo, startTime, endTime time.Time, talker string, limit, offset int) ([]*model.Message, error) {
	db, ok := ds.messageDbs[dbInfo.FilePath]
	if !ok {
		return nil, fmt.Errorf("数据库 %s 未打开", dbInfo.FilePath)
	}

	// 构建表名
	_talkerMd5Bytes := md5.Sum([]byte(talker))
	talkerMd5 := hex.EncodeToString(_talkerMd5Bytes[:])
	tableName := "Msg_" + talkerMd5

	// 构建查询条件
	conditions := []string{"create_time >= ? AND create_time <= ?"}
	args := []interface{}{startTime.Unix(), endTime.Unix()}

	query := fmt.Sprintf(`
		SELECT sort_seq, local_type, real_sender_id, create_time, message_content, packed_info_data, status
		FROM %s 
		WHERE %s 
		ORDER BY sort_seq ASC
	`, tableName, strings.Join(conditions, " AND "))

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询数据库 %s 失败: %w", dbInfo.FilePath, err)
	}
	defer rows.Close()

	// 处理查询结果
	messages := []*model.Message{}
	isChatRoom := strings.HasSuffix(talker, "@chatroom")

	for rows.Next() {
		var msg model.MessageV4
		err := rows.Scan(
			&msg.SortSeq,
			&msg.LocalType,
			&msg.RealSenderID,
			&msg.CreateTime,
			&msg.MessageContent,
			&msg.PackedInfoData,
			&msg.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描消息行失败: %w", err)
		}

		messages = append(messages, msg.Wrap(dbInfo.ID2Name, isChatRoom))
	}

	return messages, nil
}

// getMessagesFromDB 从数据库获取消息
func (ds *DataSource) getMessagesFromDB(ctx context.Context, db *sql.DB, dbInfo MessageDBInfo, startTime, endTime time.Time, talker string) ([]*model.Message, error) {
	// 构建表名
	_talkerMd5Bytes := md5.Sum([]byte(talker))
	talkerMd5 := hex.EncodeToString(_talkerMd5Bytes[:])
	tableName := "Msg_" + talkerMd5

	// 检查表是否存在
	var exists bool
	err := db.QueryRowContext(ctx,
		"SELECT 1 FROM sqlite_master WHERE type='table' AND name=?",
		tableName).Scan(&exists)

	if err != nil {
		if err == sql.ErrNoRows {
			// 表不存在，返回空结果
			return []*model.Message{}, nil
		}
		return nil, fmt.Errorf("检查表 %s 是否存在失败: %w", tableName, err)
	}

	// 构建查询条件
	conditions := []string{"create_time >= ? AND create_time <= ?"}
	args := []interface{}{startTime.Unix(), endTime.Unix()}

	query := fmt.Sprintf(`
		SELECT sort_seq, local_type, real_sender_id, create_time, message_content, packed_info_data, status
		FROM %s 
		WHERE %s 
		ORDER BY sort_seq ASC
	`, tableName, strings.Join(conditions, " AND "))

	// 执行查询
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		// 如果表不存在，SQLite 会返回错误
		if strings.Contains(err.Error(), "no such table") {
			return []*model.Message{}, nil
		}
		return nil, fmt.Errorf("查询数据库失败: %w", err)
	}
	defer rows.Close()

	// 处理查询结果
	messages := []*model.Message{}
	isChatRoom := strings.HasSuffix(talker, "@chatroom")

	for rows.Next() {
		var msg model.MessageV4
		err := rows.Scan(
			&msg.SortSeq,
			&msg.LocalType,
			&msg.RealSenderID,
			&msg.CreateTime,
			&msg.MessageContent,
			&msg.PackedInfoData,
			&msg.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描消息行失败: %w", err)
		}

		messages = append(messages, msg.Wrap(dbInfo.ID2Name, isChatRoom))
	}

	return messages, nil
}

// 联系人
func (ds *DataSource) GetContacts(ctx context.Context, key string, limit, offset int) ([]*model.Contact, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT username, local_type, alias, remark, nick_name 
				FROM contact 
				WHERE username = ? OR alias = ? OR remark = ? OR nick_name = ?`
		args = []interface{}{key, key, key, key}
	} else {
		// 查询所有联系人
		query = `SELECT username, local_type, alias, remark, nick_name FROM contact`
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
	rows, err := ds.contactDb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询联系人失败: %w", err)
	}
	defer rows.Close()

	contacts := []*model.Contact{}
	for rows.Next() {
		var contactV4 model.ContactV4
		err := rows.Scan(
			&contactV4.UserName,
			&contactV4.LocalType,
			&contactV4.Alias,
			&contactV4.Remark,
			&contactV4.NickName,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描联系人行失败: %w", err)
		}

		contacts = append(contacts, contactV4.Wrap())
	}

	return contacts, nil
}

// 群聊
func (ds *DataSource) GetChatRooms(ctx context.Context, key string, limit, offset int) ([]*model.ChatRoom, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT username, owner, ext_buffer FROM chat_room WHERE username = ?`
		args = []interface{}{key}

		// 执行查询
		rows, err := ds.contactDb.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("查询群聊失败: %w", err)
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
				return nil, fmt.Errorf("扫描群聊行失败: %w", err)
			}

			chatRooms = append(chatRooms, chatRoomV4.Wrap())
		}

		// 如果没有找到群聊，尝试通过联系人查找
		if len(chatRooms) == 0 {
			contacts, err := ds.GetContacts(ctx, key, 1, 0)
			if err == nil && len(contacts) > 0 && strings.HasSuffix(contacts[0].UserName, "@chatroom") {
				// 再次尝试通过用户名查找群聊
				rows, err := ds.contactDb.QueryContext(ctx,
					`SELECT username, owner, ext_buffer FROM chat_room WHERE username = ?`,
					contacts[0].UserName)

				if err != nil {
					return nil, fmt.Errorf("查询群聊失败: %w", err)
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
						return nil, fmt.Errorf("扫描群聊行失败: %w", err)
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
		rows, err := ds.contactDb.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("查询群聊失败: %w", err)
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
				return nil, fmt.Errorf("扫描群聊行失败: %w", err)
			}

			chatRooms = append(chatRooms, chatRoomV4.Wrap())
		}

		return chatRooms, nil
	}
}

// 最近会话
func (ds *DataSource) GetSessions(ctx context.Context, key string, limit, offset int) ([]*model.Session, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT username, summary, last_timestamp, last_msg_sender, last_sender_display_name 
				FROM SessionTable 
				WHERE username = ? OR last_sender_display_name = ?
				ORDER BY sort_timestamp DESC`
		args = []interface{}{key, key}
	} else {
		// 查询所有会话
		query = `SELECT username, summary, last_timestamp, last_msg_sender, last_sender_display_name 
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
	rows, err := ds.sessionDb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询会话失败: %w", err)
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
		)

		if err != nil {
			return nil, fmt.Errorf("扫描会话行失败: %w", err)
		}

		sessions = append(sessions, sessionV4.Wrap())
	}

	return sessions, nil
}

func (ds *DataSource) GetMedia(ctx context.Context, _type string, key string) (*model.Media, error) {
	if key == "" {
		return nil, fmt.Errorf("key 不能为空")
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("key 长度必须为 32")
	}

	var table string
	switch _type {
	case "image":
		table = "image_hardlink_info_v3"
	case "video":
		table = "video_hardlink_info_v3"
	case "file":
		table = "file_hardlink_info_v3"
	default:
		return nil, fmt.Errorf("不支持的媒体类型: %s", _type)
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

	rows, err := ds.mediaDb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询媒体失败: %w", err)
	}
	defer rows.Close()

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
			return nil, fmt.Errorf("扫描会话行失败: %w", err)
		}
		mediaV4.Type = _type
		media = mediaV4.Wrap()

		// 跳过缩略图
		if _type == "image" && !strings.Contains(media.Name, "_t") {
			break
		}
	}

	if media == nil {
		return nil, fmt.Errorf("未找到媒体 %s", key)
	}

	return media, nil
}

func (ds *DataSource) Close() error {
	var errs []error

	// 关闭消息数据库连接
	for path, db := range ds.messageDbs {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭消息数据库 %s 失败: %w", path, err))
		}
	}

	// 关闭联系人数据库连接
	if ds.contactDb != nil {
		if err := ds.contactDb.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭联系人数据库失败: %w", err))
		}
	}

	// 关闭会话数据库连接
	if ds.sessionDb != nil {
		if err := ds.sessionDb.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭会话数据库失败: %w", err))
		}
	}

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

package windowsv3

import (
	"context"
	"database/sql"
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
	MessageFilePattern = "^MSG([0-9]?[0-9])?\\.db$"
	ContactFilePattern = "^MicroMsg.db$"
)

// MessageDBInfo 保存消息数据库的信息
type MessageDBInfo struct {
	FilePath  string
	StartTime time.Time
	EndTime   time.Time
	TalkerMap map[string]int
}

// DataSource 实现了 DataSource 接口
type DataSource struct {
	// 消息数据库
	messageFiles []MessageDBInfo
	messageDbs   map[string]*sql.DB

	// 联系人数据库
	contactDbFile string
	contactDb     *sql.DB
}

// New 创建一个新的 WindowsV3DataSource
func New(path string) (*DataSource, error) {
	ds := &DataSource{
		messageFiles: make([]MessageDBInfo, 0),
		messageDbs:   make(map[string]*sql.DB),
	}

	// 初始化消息数据库
	if err := ds.initMessageDbs(path); err != nil {
		return nil, fmt.Errorf("初始化消息数据库失败: %w", err)
	}

	// 初始化联系人数据库
	if err := ds.initContactDb(path); err != nil {
		return nil, fmt.Errorf("初始化联系人数据库失败: %w", err)
	}

	return ds, nil
}

// initMessageDbs 初始化消息数据库
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

		// 获取 DBInfo 表中的开始时间
		var startTime time.Time

		rows, err := db.Query("SELECT tableIndex, tableVersion, tableDesc FROM DBInfo")
		if err != nil {
			log.Printf("警告: 查询数据库 %s 的 DBInfo 表失败: %v", filePath, err)
			db.Close()
			continue
		}

		for rows.Next() {
			var tableIndex int
			var tableVersion int64
			var tableDesc string

			if err := rows.Scan(&tableIndex, &tableVersion, &tableDesc); err != nil {
				log.Printf("警告: 扫描 DBInfo 行失败: %v", err)
				continue
			}

			// 查找描述为 "Start Time" 的记录
			if strings.Contains(tableDesc, "Start Time") {
				startTime = time.Unix(tableVersion/1000, (tableVersion%1000)*1000000)
				break
			}
		}
		rows.Close()

		// 组织 TalkerMap
		talkerMap := make(map[string]int)
		rows, err = db.Query("SELECT UsrName FROM Name2ID")
		if err != nil {
			log.Printf("警告: 查询数据库 %s 的 Name2ID 表失败: %v", filePath, err)
			db.Close()
			continue
		}

		i := 1
		for rows.Next() {
			var userName string
			if err := rows.Scan(&userName); err != nil {
				log.Printf("警告: 扫描 Name2ID 行失败: %v", err)
				continue
			}
			talkerMap[userName] = i
			i++
		}
		rows.Close()

		// 保存数据库信息
		ds.messageFiles = append(ds.messageFiles, MessageDBInfo{
			FilePath:  filePath,
			StartTime: startTime,
			TalkerMap: talkerMap,
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

// initContactDb 初始化联系人数据库
func (ds *DataSource) initContactDb(path string) error {
	files, err := util.FindFilesWithPatterns(path, ContactFilePattern, true)
	if err != nil {
		return fmt.Errorf("查找联系人数据库文件失败: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("未找到联系人数据库文件: %s", path)
	}

	ds.contactDbFile = files[0]

	ds.contactDb, err = sql.Open("sqlite3", ds.contactDbFile)
	if err != nil {
		return fmt.Errorf("连接联系人数据库失败: %w", err)
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

// GetMessages 实现 DataSource 接口的 GetMessages 方法
func (ds *DataSource) GetMessages(ctx context.Context, startTime, endTime time.Time, talker string, limit, offset int) ([]*model.Message, error) {
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

		// 构建查询条件
		conditions := []string{"Sequence >= ? AND Sequence <= ?"}
		args := []interface{}{startTime.Unix() * 1000, endTime.Unix() * 1000}

		if len(talker) > 0 {
			talkerID, ok := dbInfo.TalkerMap[talker]
			if ok {
				conditions = append(conditions, "TalkerId = ?")
				args = append(args, talkerID)
			} else {
				conditions = append(conditions, "StrTalker = ?")
				args = append(args, talker)
			}
		}

		query := fmt.Sprintf(`
            SELECT Sequence, CreateTime, TalkerId, StrTalker, IsSender, 
                Type, SubType, StrContent, CompressContent, BytesExtra
            FROM MSG 
            WHERE %s 
            ORDER BY Sequence ASC
        `, strings.Join(conditions, " AND "))

		// 执行查询
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			log.Printf("警告: 查询数据库 %s 失败: %v", dbInfo.FilePath, err)
			continue
		}

		// 处理查询结果
		for rows.Next() {
			var msg model.MessageV3
			var compressContent []byte
			var bytesExtra []byte

			err := rows.Scan(
				&msg.Sequence,
				&msg.CreateTime,
				&msg.TalkerID,
				&msg.StrTalker,
				&msg.IsSender,
				&msg.Type,
				&msg.SubType,
				&msg.StrContent,
				&compressContent,
				&bytesExtra,
			)
			if err != nil {
				log.Printf("警告: 扫描消息行失败: %v", err)
				continue
			}
			msg.CompressContent = compressContent
			msg.BytesExtra = bytesExtra

			totalMessages = append(totalMessages, msg.Wrap())
		}
		rows.Close()

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
	// 构建查询条件
	conditions := []string{"Sequence >= ? AND Sequence <= ?"}
	args := []interface{}{startTime.Unix() * 1000, endTime.Unix() * 1000}
	if len(talker) > 0 {
		// TalkerId 有索引，优先使用
		talkerID, ok := dbInfo.TalkerMap[talker]
		if ok {
			conditions = append(conditions, "TalkerId = ?")
			args = append(args, talkerID)
		} else {
			conditions = append(conditions, "StrTalker = ?")
			args = append(args, talker)
		}
	}
	query := fmt.Sprintf(`
        SELECT Sequence, CreateTime, TalkerId, StrTalker, IsSender, 
            Type, SubType, StrContent, CompressContent, BytesExtra
        FROM MSG 
        WHERE %s 
        ORDER BY Sequence ASC
    `, strings.Join(conditions, " AND "))

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)

		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	rows, err := ds.messageDbs[dbInfo.FilePath].QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询数据库 %s 失败: %w", dbInfo.FilePath, err)
	}
	defer rows.Close()

	// 处理查询结果
	totalMessages := []*model.Message{}
	for rows.Next() {
		var msg model.MessageV3
		var compressContent []byte
		var bytesExtra []byte
		err := rows.Scan(
			&msg.Sequence,
			&msg.CreateTime,
			&msg.TalkerID,
			&msg.StrTalker,
			&msg.IsSender,
			&msg.Type,
			&msg.SubType,
			&msg.StrContent,
			&compressContent,
			&bytesExtra,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描消息行失败: %w", err)
		}
		msg.CompressContent = compressContent
		msg.BytesExtra = bytesExtra
		totalMessages = append(totalMessages, msg.Wrap())
	}
	return totalMessages, nil
}

// GetContacts 实现获取联系人信息的方法
func (ds *DataSource) GetContacts(ctx context.Context, key string, limit, offset int) ([]*model.Contact, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT UserName, Alias, Remark, NickName, Reserved1 FROM Contact 
                WHERE UserName = ? OR Alias = ? OR Remark = ? OR NickName = ?`
		args = []interface{}{key, key, key, key}
	} else {
		// 查询所有联系人
		query = `SELECT UserName, Alias, Remark, NickName, Reserved1 FROM Contact`
	}

	// 添加排序、分页
	query += ` ORDER BY UserName`
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
		var contactV3 model.ContactV3
		err := rows.Scan(
			&contactV3.UserName,
			&contactV3.Alias,
			&contactV3.Remark,
			&contactV3.NickName,
			&contactV3.Reserved1,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描联系人行失败: %w", err)
		}

		contacts = append(contacts, contactV3.Wrap())
	}

	return contacts, nil
}

// GetChatRooms 实现获取群聊信息的方法
func (ds *DataSource) GetChatRooms(ctx context.Context, key string, limit, offset int) ([]*model.ChatRoom, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT ChatRoomName, Reserved2, RoomData FROM ChatRoom WHERE ChatRoomName = ?`
		args = []interface{}{key}

		// 执行查询
		rows, err := ds.contactDb.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("查询群聊失败: %w", err)
		}
		defer rows.Close()

		chatRooms := []*model.ChatRoom{}
		for rows.Next() {
			var chatRoomV3 model.ChatRoomV3
			err := rows.Scan(
				&chatRoomV3.ChatRoomName,
				&chatRoomV3.Reserved2,
				&chatRoomV3.RoomData,
			)

			if err != nil {
				return nil, fmt.Errorf("扫描群聊行失败: %w", err)
			}

			chatRooms = append(chatRooms, chatRoomV3.Wrap())
		}

		// 如果没有找到群聊，尝试通过联系人查找
		if len(chatRooms) == 0 {
			contacts, err := ds.GetContacts(ctx, key, 1, 0)
			if err == nil && len(contacts) > 0 && strings.HasSuffix(contacts[0].UserName, "@chatroom") {
				// 再次尝试通过用户名查找群聊
				rows, err := ds.contactDb.QueryContext(ctx,
					`SELECT ChatRoomName, Reserved2, RoomData FROM ChatRoom WHERE ChatRoomName = ?`,
					contacts[0].UserName)

				if err != nil {
					return nil, fmt.Errorf("查询群聊失败: %w", err)
				}
				defer rows.Close()

				for rows.Next() {
					var chatRoomV3 model.ChatRoomV3
					err := rows.Scan(
						&chatRoomV3.ChatRoomName,
						&chatRoomV3.Reserved2,
						&chatRoomV3.RoomData,
					)

					if err != nil {
						return nil, fmt.Errorf("扫描群聊行失败: %w", err)
					}

					chatRooms = append(chatRooms, chatRoomV3.Wrap())
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
		query = `SELECT ChatRoomName, Reserved2, RoomData FROM ChatRoom`

		// 添加排序、分页
		query += ` ORDER BY ChatRoomName`
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
			var chatRoomV3 model.ChatRoomV3
			err := rows.Scan(
				&chatRoomV3.ChatRoomName,
				&chatRoomV3.Reserved2,
				&chatRoomV3.RoomData,
			)

			if err != nil {
				return nil, fmt.Errorf("扫描群聊行失败: %w", err)
			}

			chatRooms = append(chatRooms, chatRoomV3.Wrap())
		}

		return chatRooms, nil
	}
}

// GetSessions 实现获取会话信息的方法
func (ds *DataSource) GetSessions(ctx context.Context, key string, limit, offset int) ([]*model.Session, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT strUsrName, nOrder, strNickName, strContent, nTime 
                FROM Session 
                WHERE strUsrName = ? OR strNickName = ?
                ORDER BY nOrder DESC`
		args = []interface{}{key, key}
	} else {
		// 查询所有会话
		query = `SELECT strUsrName, nOrder, strNickName, strContent, nTime 
                FROM Session 
                ORDER BY nOrder DESC`
	}

	// 添加分页
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	rows, err := ds.contactDb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}
	defer rows.Close()

	sessions := []*model.Session{}
	for rows.Next() {
		var sessionV3 model.SessionV3
		err := rows.Scan(
			&sessionV3.StrUsrName,
			&sessionV3.NOrder,
			&sessionV3.StrNickName,
			&sessionV3.StrContent,
			&sessionV3.NTime,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描会话行失败: %w", err)
		}

		sessions = append(sessions, sessionV3.Wrap())
	}

	return sessions, nil
}

// Close 实现 DataSource 接口的 Close 方法
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

	if len(errs) > 0 {
		return fmt.Errorf("关闭数据库连接时发生错误: %v", errs)
	}

	return nil
}

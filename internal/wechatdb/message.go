package wechatdb

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/sjzar/chatlog/pkg/model"
	"github.com/sjzar/chatlog/pkg/util"

	_ "github.com/mattn/go-sqlite3"
)

const (
	MessageFileV3 = "^MSG([0-9]?[0-9])?\\.db$"
	MessageFileV4 = "^messages_([0-9]?[0-9])+\\.db$"
)

type Message struct {
	version int
	files   []MsgDBInfo
	dbs     map[string]*sql.DB
}

type MsgDBInfo struct {
	FilePath  string
	StartTime time.Time
	EndTime   time.Time
	TalkerMap map[string]int
}

func NewMessage(path string, version int) (*Message, error) {
	m := &Message{
		version: version,
		files:   make([]MsgDBInfo, 0),
		dbs:     make(map[string]*sql.DB),
	}

	// 查找所有 MSG[0-13].db 文件
	files, err := util.FindFilesWithPatterns(path, MessageFileV3, true)
	if err != nil {
		return nil, fmt.Errorf("查找数据库文件失败: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("未找到任何数据库文件: %s", path)
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
		// 首先检查表结构
		var startTime time.Time

		// 尝试从 DBInfo 表中查找 Start Time 对应的记录
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

		// 保存数据库信息
		m.files = append(m.files, MsgDBInfo{
			FilePath:  filePath,
			StartTime: startTime,
			TalkerMap: talkerMap,
		})

		// 保存数据库连接
		m.dbs[filePath] = db
	}

	// 按照 StartTime 排序数据库文件
	sort.Slice(m.files, func(i, j int) bool {
		return m.files[i].StartTime.Before(m.files[j].StartTime)
	})

	for i := range m.files {
		if i == len(m.files)-1 {
			m.files[i].EndTime = time.Now()
		} else {
			m.files[i].EndTime = m.files[i+1].StartTime
		}
	}

	return m, nil
}

// GetMessages 根据时间段和 talker 查询聊天记录
func (m *Message) GetMessages(startTime, endTime time.Time, talker string, limit, offset int) ([]*model.Message, error) {
	// 找到时间范围内的数据库文件
	dbInfos := m.getDBInfosForTimeRange(startTime, endTime)
	if len(dbInfos) == 0 {
		return nil, fmt.Errorf("未找到时间范围 %v 到 %v 内的数据库文件", startTime, endTime)
	}

	if len(dbInfos) == 1 {
		// LIMIT 和 OFFSET 逻辑在单文件情况下可以直接在 SQL 里处理
		return m.getMessagesSingleFile(dbInfos[0], startTime, endTime, talker, limit, offset)
	}

	// 从每个相关数据库中查询消息
	totalMessages := []*model.Message{}

	for _, dbInfo := range dbInfos {
		db, ok := m.dbs[dbInfo.FilePath]
		if !ok {
			log.Printf("警告: 数据库 %s 未打开", dbInfo.FilePath)
			continue
		}

		// 构建查询条件
		// 使用 Sequence 查询，有索引
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
		rows, err := db.Query(query, args...)
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

	// FIXME limit 和 offset 逻辑，在多文件边界条件下不好处理，直接查询全量数据后在进程里处理
	if limit > 0 {
		if offset >= len(totalMessages) {
			return []*model.Message{}, nil
		}
		end := offset + limit
		if end > len(totalMessages) || limit == 0 {
			end = len(totalMessages)
		}
		return totalMessages[offset:end], nil
	}

	return totalMessages, nil

}

func (m *Message) getMessagesSingleFile(dbInfo MsgDBInfo, startTime, endTime time.Time, talker string, limit, offset int) ([]*model.Message, error) {
	// 构建查询条件
	// 使用 Sequence 查询，有索引
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
	rows, err := m.dbs[dbInfo.FilePath].Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询数据库 %s 失败: %v", dbInfo.FilePath, err)
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
			return nil, fmt.Errorf("扫描消息行失败: %v", err)
		}
		msg.CompressContent = compressContent
		msg.BytesExtra = bytesExtra
		totalMessages = append(totalMessages, msg.Wrap())
	}
	return totalMessages, nil
}

func (m *Message) getDBInfosForTimeRange(startTime, endTime time.Time) []MsgDBInfo {
	var dbs []MsgDBInfo
	for _, info := range m.files {
		if info.StartTime.Before(endTime) && info.EndTime.After(startTime) {
			dbs = append(dbs, info)
		}
	}
	return dbs
}

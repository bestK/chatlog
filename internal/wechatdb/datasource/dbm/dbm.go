package dbm

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/pkg/filemonitor"
)

type DBManager struct {
	path    string
	id      string
	fm      *filemonitor.FileMonitor
	fgs     map[string]*filemonitor.FileGroup
	dbs     map[string]*sql.DB
	dbPaths map[string][]string
	locks   map[string]bool // 文件锁定状态
	lockMut sync.Mutex      // 锁状态的互斥锁
	mutex   sync.RWMutex
}

func NewDBManager(path string) *DBManager {
	return &DBManager{
		path:    path,
		id:      filepath.Base(path),
		fm:      filemonitor.NewFileMonitor(),
		fgs:     make(map[string]*filemonitor.FileGroup),
		dbs:     make(map[string]*sql.DB),
		dbPaths: make(map[string][]string),
		locks:   make(map[string]bool),
	}
}

func (d *DBManager) AddGroup(g *Group) error {
	fg, err := filemonitor.NewFileGroup(g.Name, d.path, g.Pattern, g.BlackList)
	if err != nil {
		return err
	}
	fg.AddCallback(d.Callback)
	d.fm.AddGroup(fg)
	d.mutex.Lock()
	d.fgs[g.Name] = fg
	d.mutex.Unlock()
	return nil
}

func (d *DBManager) AddCallback(group string, callback func(event fsnotify.Event) error) error {
	d.mutex.RLock()
	fg, ok := d.fgs[group]
	d.mutex.RUnlock()
	if !ok {
		return errors.FileGroupNotFound(group)
	}
	fg.AddCallback(callback)
	return nil
}

func (d *DBManager) GetDB(name string) (*sql.DB, error) {
	dbPaths, err := d.GetDBPath(name)
	if err != nil {
		return nil, err
	}
	return d.OpenDB(dbPaths[0])
}

func (d *DBManager) GetDBs(name string) ([]*sql.DB, error) {
	dbPaths, err := d.GetDBPath(name)
	if err != nil {
		return nil, err
	}
	dbs := make([]*sql.DB, 0)
	for _, file := range dbPaths {
		db, err := d.OpenDB(file)
		if err != nil {
			return nil, err
		}
		dbs = append(dbs, db)
	}
	return dbs, nil
}

func (d *DBManager) GetDBPath(name string) ([]string, error) {
	d.mutex.RLock()
	dbPaths, ok := d.dbPaths[name]
	d.mutex.RUnlock()
	if !ok {
		d.mutex.RLock()
		fg, ok := d.fgs[name]
		d.mutex.RUnlock()
		if !ok {
			return nil, errors.FileGroupNotFound(name)
		}
		list, err := fg.List()
		if err != nil {
			return nil, errors.DBFileNotFound(d.path, fg.PatternStr, err)
		}
		if len(list) == 0 {
			return nil, errors.DBFileNotFound(d.path, fg.PatternStr, nil)
		}
		dbPaths = list
		d.mutex.Lock()
		d.dbPaths[name] = dbPaths
		d.mutex.Unlock()
	}
	return dbPaths, nil
}

func (d *DBManager) CloseDB(path string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 规范化路径以避免斜杠问题
	normalizedPath := filepath.Clean(path)

	// 尝试直接查找
	db, ok := d.dbs[path]
	if !ok {
		// 尝试规范化路径查找
		db, ok = d.dbs[normalizedPath]
	}
	// 如果还是没找到，遍历查找（忽略大小写和斜杠差异，针对 Windows）
	if !ok && runtime.GOOS == "windows" {
		lowerPath := strings.ToLower(normalizedPath)
		for k, v := range d.dbs {
			if strings.ToLower(filepath.Clean(k)) == lowerPath {
				db = v
				ok = true
				// 更新 map key 以便后续删除
				delete(d.dbs, k)
				break
			}
		}
	} else if ok {
		delete(d.dbs, path)
		if path != normalizedPath {
			delete(d.dbs, normalizedPath)
		}
	}

	if ok {
		// 必须同步关闭，否则在Windows上无法立即释放文件锁
		if err := db.Close(); err != nil {
			log.Debug().Err(err).Str("path", path).Msg("dbm: close db failed")
		}
	}
}

// getLockKey 统一获取锁的 key
func (d *DBManager) getLockKey(path string) string {
	p := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(p)
	}
	return p
}

// LockDB 锁定指定的数据库路径，禁止 OpenDB
func (d *DBManager) LockDB(path string) {
	d.lockMut.Lock()
	defer d.lockMut.Unlock()
	d.locks[d.getLockKey(path)] = true
}

// UnlockDB 解锁指定的数据库路径
func (d *DBManager) UnlockDB(path string) {
	d.lockMut.Lock()
	defer d.lockMut.Unlock()
	delete(d.locks, d.getLockKey(path))
}

func (d *DBManager) OpenDB(path string) (*sql.DB, error) {
	// 检查是否被锁定，如果被锁定则等待
	lockKey := d.getLockKey(path)
	for {
		d.lockMut.Lock()
		locked := d.locks[lockKey]
		d.lockMut.Unlock()

		if !locked {
			break
		}
		// 如果被锁定，等待一下再试
		time.Sleep(100 * time.Millisecond)
	}

	// 在 Windows 上不使用并发缓存，确保每次获取都是新连接，以便在 Close 后立即释放文件锁
	if runtime.GOOS == "windows" {
		db, err := d.openDB(path)
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	d.mutex.RLock()
	db, ok := d.dbs[path]
	d.mutex.RUnlock()
	if ok {
		return db, nil
	}

	db, err := d.openDB(path)
	if err != nil {
		return nil, err
	}

	d.mutex.Lock()
	d.dbs[path] = db
	d.mutex.Unlock()

	return db, nil
}

func (d *DBManager) openDB(path string) (*sql.DB, error) {
	// 构建连接字符串
	var connStr string
	if runtime.GOOS == "windows" {
		// 在 Windows 上使用 immutable=1 参数绕过独占锁
		// 这样可以避免复制大文件，大幅节省磁盘空间和时间
		uri := filepath.ToSlash(path)
		connStr = fmt.Sprintf("file:%s?immutable=1&mode=ro", uri)
	} else {
		// 其他平台直接使用路径
		connStr = path
	}

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		log.Err(err).Msgf("连接数据库 %s 失败", path)
		return nil, err
	}

	// 在 Windows 上，我们将连接池限制为 1 个连接以最大程度减少文件句柄占用
	if runtime.GOOS == "windows" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(time.Minute)
	}

	return db, nil
}

func (d *DBManager) Callback(event fsnotify.Event) error {
	// 监听 Create, Write 和 Rename 事件，当文件变化时关闭旧连接
	// 这样下次访问时会重新打开，读取最新数据
	if !(event.Op.Has(fsnotify.Create) || event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Rename)) {
		return nil
	}

	d.mutex.Lock()
	db, ok := d.dbs[event.Name]
	if ok {
		delete(d.dbs, event.Name)
		// 在 Windows 上不再使用 goroutine 异步关闭，避免 database is closed 竞争
		// 我们依赖 CloseDB 被显式调用或者连接自然失效
		if runtime.GOOS != "windows" {
			go func(db *sql.DB) {
				time.Sleep(time.Second * 5)
				db.Close()
			}(db)
		} else {
			// Windows 下同步尝试关闭（如果当前无活跃查询，Close 会成功并释放锁）
			// 但由于我们使用了缓存池，这里其实很难直接关闭成功，
			// 正确的做法是在 DecryptDBFile 的 defer 中调用 CloseDB。
		}
	}
	d.mutex.Unlock()

	return nil
}

func (d *DBManager) Start() error {
	return d.fm.Start()
}

func (d *DBManager) Stop() error {
	return d.fm.Stop()
}

func (d *DBManager) Close() error {
	for _, db := range d.dbs {
		db.Close()
	}
	return d.fm.Stop()
}

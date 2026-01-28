package dbm

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
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

func (d *DBManager) OpenDB(path string) (*sql.DB, error) {
	d.mutex.RLock()
	db, ok := d.dbs[path]
	d.mutex.RUnlock()
	if ok {
		return db, nil
	}

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

	d.mutex.Lock()
	d.dbs[path] = db
	d.mutex.Unlock()
	return db, nil
}

func (d *DBManager) Callback(event fsnotify.Event) error {
	if !event.Op.Has(fsnotify.Create) {
		return nil
	}

	d.mutex.Lock()
	db, ok := d.dbs[event.Name]
	if ok {
		delete(d.dbs, event.Name)
		go func(db *sql.DB) {
			time.Sleep(time.Second * 5)
			db.Close()
		}(db)
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

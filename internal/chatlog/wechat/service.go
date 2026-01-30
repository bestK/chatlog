package wechat

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat"
	"github.com/sjzar/chatlog/internal/wechat/decrypt"
	"github.com/sjzar/chatlog/pkg/filemonitor"
	"github.com/sjzar/chatlog/pkg/util"
)

var (
	DebounceTime = 500 * time.Millisecond
	MaxWaitTime  = 10 * time.Second
)

type DBController interface {
	CloseDB(path string) error
	LockDB(path string) error
	UnlockDB(path string) error
}

type Service struct {
	conf           Config
	lastEvents     map[string]time.Time
	pendingActions map[string]bool
	mutex          sync.Mutex
	fm             *filemonitor.FileMonitor
	dbController   DBController
}

type Config interface {
	GetDataKey() string
	GetDataDir() string
	GetWorkDir() string
	GetPlatform() string
	GetVersion() int
}

func NewService(conf Config) *Service {
	return &Service{
		conf:           conf,
		lastEvents:     make(map[string]time.Time),
		pendingActions: make(map[string]bool),
	}
}

func (s *Service) SetDBController(ctrl DBController) {
	s.dbController = ctrl
}

// GetWeChatInstances returns all running WeChat instances
func (s *Service) GetWeChatInstances() []*wechat.Account {
	wechat.Load()
	return wechat.GetAccounts()
}

// GetDataKey extracts the encryption key from a WeChat process
func (s *Service) GetDataKey(info *wechat.Account) (string, string, error) {
	if info == nil {
		return "", "", fmt.Errorf("no WeChat instance selected")
	}

	// 设置90秒超时（比底层的60秒多留余量）
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	dataKey, imgKey, err := info.GetKey(ctx)
	if err != nil {
		return "", "", err
	}

	return dataKey, imgKey, nil
}

// GetImgKey 仅获取图片密钥（不会重启微信）
func (s *Service) GetImgKey(info *wechat.Account) (string, error) {
	if info == nil {
		return "", fmt.Errorf("no WeChat instance selected")
	}

	// 设置60秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	imgKey, err := info.GetImgKey(ctx)
	if err != nil {
		return "", err
	}

	return imgKey, nil
}

func (s *Service) StartAutoDecrypt() error {
	log.Info().Msgf("start auto decrypt, data dir: %s", s.conf.GetDataDir())
	// Update pattern to match .db and .db-wal files
	dbGroup, err := filemonitor.NewFileGroup("wechat", s.conf.GetDataDir(), `.*\.db(-wal)?$`, []string{"fts"})
	if err != nil {
		return err
	}
	dbGroup.AddCallback(s.DecryptFileCallback)

	s.fm = filemonitor.NewFileMonitor()
	s.fm.AddGroup(dbGroup)
	if err := s.fm.Start(); err != nil {
		log.Debug().Err(err).Msg("failed to start file monitor")
		return err
	}
	return nil
}

func (s *Service) StopAutoDecrypt() error {
	if s.fm != nil {
		if err := s.fm.Stop(); err != nil {
			return err
		}
	}
	s.fm = nil
	return nil
}

func (s *Service) DecryptFileCallback(event fsnotify.Event) error {
	// Local file system and Syncthing checks...
	if !(event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Create) || event.Op.Has(fsnotify.Rename)) {
		return nil
	}

	targetFile := event.Name
	// If it's a WAL file, target the main DB file
	if strings.HasSuffix(targetFile, "-wal") {
		targetFile = strings.TrimSuffix(targetFile, "-wal")
		// log.Info().Msgf("WAL file change detected, targeting main DB: %s", targetFile)
	}

	s.mutex.Lock()
	s.lastEvents[targetFile] = time.Now()

	if !s.pendingActions[targetFile] {
		s.pendingActions[targetFile] = true
		s.mutex.Unlock()
		go s.waitAndProcess(targetFile)
	} else {
		s.mutex.Unlock()
	}

	return nil
}

func (s *Service) waitAndProcess(dbFile string) {
	start := time.Now()
	for {
		time.Sleep(DebounceTime)

		s.mutex.Lock()
		lastEventTime := s.lastEvents[dbFile]
		elapsed := time.Since(lastEventTime)
		totalElapsed := time.Since(start)

		if elapsed >= DebounceTime || totalElapsed >= MaxWaitTime {
			s.pendingActions[dbFile] = false
			s.mutex.Unlock()

			log.Debug().Msgf("Processing file: %s", dbFile)
			s.DecryptDBFile(dbFile)
			return
		}
		s.mutex.Unlock()
	}
}
func (s *Service) DecryptDBFile(dbFile string) error {
	decryptor, err := decrypt.NewDecryptor(s.conf.GetPlatform(), s.conf.GetVersion())
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(s.conf.GetDataDir(), dbFile)
	if err != nil {
		return err
	}
	output := filepath.Join(s.conf.GetWorkDir(), rel)

	if err := util.PrepareDir(filepath.Dir(output)); err != nil {
		return err
	}

	tmp := output + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	err = decryptor.Decrypt(context.Background(), dbFile, s.conf.GetDataKey(), f)
	f.Close()

	if err != nil {
		if err == errors.ErrAlreadyDecrypted {
			data, _ := os.ReadFile(dbFile)
			_ = os.WriteFile(tmp, data, 0644)
		} else {
			_ = os.Remove(tmp)
			return err
		}
	}

	if err := s.replaceDB(tmp, output); err != nil {
		log.Logger.Error().Err(err).Msgf("failed to replace db %s", output)

		return err
	}

	// 清理工作目录下残留的 WAL/SHM 文件，防止 SQLite 读取加密的 WAL 导致失败
	s.removeWalFiles(output)
	return nil
}

func (s *Service) replaceDB(tmp, target string) error {
	// 在替换之前清理目标目录的 WAL 文件，防止 SQLite 在替换后立即读取旧的加密 WAL
	s.removeWalFiles(target)

	if s.dbController != nil {
		s.dbController.LockDB(target)
		defer s.dbController.UnlockDB(target)
		s.dbController.CloseDB(target)
		time.Sleep(100 * time.Millisecond)
	}

	for i := 0; i < 5; i++ {
		err := os.Rename(tmp, target)
		if err == nil {
			return nil
		}

		if errors.Is(err, fs.ErrPermission) {
			_ = os.Remove(target)
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("failed to replace db %s", target)
}

func (s *Service) DecryptDBFiles() error {
	dbGroup, err := filemonitor.NewFileGroup("wechat", s.conf.GetDataDir(), `.*\.db$`, []string{"fts"})
	if err != nil {
		return err
	}

	dbFiles, err := dbGroup.List()
	if err != nil {
		return err
	}

	for _, dbFile := range dbFiles {
		if err := s.DecryptDBFile(dbFile); err != nil {
			log.Debug().Msgf("DecryptDBFile %s failed: %v", dbFile, err)
			continue
		}
	}

	return nil
}

// removeWalFiles 删除数据库对应的 WAL 和 SHM 文件
// 这是必要的，因为微信使用加密的 WAL 模式，解密后这些文件仍是加密的
// 如果不删除，SQLite 会尝试读取加密的 WAL 导致查询失败或返回空数据
func (s *Service) removeWalFiles(dbFile string) {
	walFile := dbFile + "-wal"
	shmFile := dbFile + "-shm"
	if err := os.Remove(walFile); err != nil && !os.IsNotExist(err) {
		log.Debug().Err(err).Msgf("failed to remove wal file %s", walFile)
	}
	if err := os.Remove(shmFile); err != nil && !os.IsNotExist(err) {
		log.Debug().Err(err).Msgf("failed to remove shm file %s", shmFile)
	}
}

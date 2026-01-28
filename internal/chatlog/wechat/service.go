package wechat

import (
	"context"
	"fmt"
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
	DebounceTime = 1 * time.Second
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
	if !(event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Create)) {
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

	output := filepath.Join(s.conf.GetWorkDir(), dbFile[len(s.conf.GetDataDir()):])
	if err := util.PrepareDir(filepath.Dir(output)); err != nil {
		return err
	}

	outputTemp := output + ".tmp"
	outputFile, err := os.Create(outputTemp)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer func() {
		outputFile.Close()

		// 使用文件锁机制保证替换过程的原子性
		if s.dbController != nil {
			// 1. 锁定路径，阻止新的 OpenDB
			s.dbController.LockDB(output)
			defer s.dbController.UnlockDB(output)

			// 2. 关闭现有连接
			s.dbController.CloseDB(output)

			// 稍微给一点时间让连接关闭
			time.Sleep(100 * time.Millisecond)
		}

		// 3. 执行替换（简单的重试机制以防万一）
		maxRetries := 5
		for i := 0; i < maxRetries; i++ {
			// 尝试直接重命名
			err := os.Rename(outputTemp, output)
			if err == nil {
				// log.Info().Msgf("successfully replaced %s", output)
				return // 成功，退出
			}

			// 如果重命名失败，尝试删除目标文件
			if os.IsExist(err) || strings.Contains(err.Error(), "exist") || strings.Contains(err.Error(), "access") || strings.Contains(err.Error(), "used by another process") {
				_ = os.Remove(output)
				if renameErr := os.Rename(outputTemp, output); renameErr == nil {
					// log.Info().Msgf("successfully replaced %s after remove", output)
					return // 成功，退出
				}
			}

			// 如果还有 Controller，再次尝试关闭（可能锁加晚了？）
			if s.dbController != nil {
				s.dbController.CloseDB(output)
			}

			time.Sleep(200 * time.Millisecond)
		}

		// 4. 所有重试都失败，清理临时文件
		log.Error().Msgf("failed to replace %s after %d retries, cleaning up", output, maxRetries)
		os.Remove(outputTemp)
	}()

	if err := decryptor.Decrypt(context.Background(), dbFile, s.conf.GetDataKey(), outputFile); err != nil {
		if err == errors.ErrAlreadyDecrypted {
			if data, err := os.ReadFile(dbFile); err == nil {
				outputFile.Write(data)
			}
			return nil
		}
		log.Err(err).Msgf("failed to decrypt %s", dbFile)
		return err
	}

	log.Debug().Msgf("Decrypted %s to %s", dbFile, output)

	return nil
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

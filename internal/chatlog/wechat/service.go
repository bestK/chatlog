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

type Service struct {
	conf            Config
	lastEvents      map[string]time.Time
	pendingActions  map[string]bool
	mutex           sync.Mutex
	fm              *filemonitor.FileMonitor
	CloseDBCallback func(string) error
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

// GetWeChatInstances returns all running WeChat instances
func (s *Service) GetWeChatInstances() []*wechat.Account {
	wechat.Load()
	return wechat.GetAccounts()
}

// GetDataKey extracts the encryption key from a WeChat process
func (s *Service) GetDataKey(info *wechat.Account) (string, error) {
	if info == nil {
		return "", fmt.Errorf("no WeChat instance selected")
	}

	key, _, err := info.GetKey(context.Background())
	if err != nil {
		return "", err
	}

	return key, nil
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

		// 重试机制：尝试多次替换文件
		maxRetries := 10
		for i := 0; i < maxRetries; i++ {
			// 1. 尝试关闭连接
			if s.CloseDBCallback != nil {
				s.CloseDBCallback(output)
				// 稍微等待文件锁释放
				time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			}

			// 2. 尝试直接重命名
			err := os.Rename(outputTemp, output)
			if err == nil {
				// log.Info().Msgf("successfully replaced %s", output)
				return // 成功，退出
			}

			// 3. 如果重命名失败，尝试删除目标文件
			if os.IsExist(err) || strings.Contains(err.Error(), "exist") || strings.Contains(err.Error(), "access") || strings.Contains(err.Error(), "used by another process") {
				if removeErr := os.Remove(output); removeErr != nil {
					log.Debug().Err(removeErr).Msgf("failed to remove target file %s (retry %d)", output, i)
				} else {
					// 删除成功后再次尝试重命名
					if renameErr := os.Rename(outputTemp, output); renameErr == nil {
						// log.Info().Msgf("successfully replaced %s after remove", output)
						return // 成功，退出
					}
				}
			}

			log.Debug().Err(err).Msgf("retry %d: replace file failed, waiting...", i)
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

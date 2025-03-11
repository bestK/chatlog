package wechat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/wechat"
	"github.com/sjzar/chatlog/pkg/util"
)

type Service struct {
	ctx *ctx.Context
}

func NewService(ctx *ctx.Context) *Service {
	return &Service{
		ctx: ctx,
	}
}

// GetWeChatInstances returns all running WeChat instances
func (s *Service) GetWeChatInstances() []*wechat.Info {
	wechat.Load()
	return wechat.Items
}

// GetDataKey extracts the encryption key from a WeChat process
func (s *Service) GetDataKey(info *wechat.Info) (string, error) {
	if info == nil {
		return "", fmt.Errorf("no WeChat instance selected")
	}

	key, err := info.GetKey()
	if err != nil {
		return "", err
	}

	return key, nil
}

// FindDBFiles finds all .db files in the specified directory
func (s *Service) FindDBFiles(rootDir string, recursive bool) ([]string, error) {
	// Check if directory exists
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("cannot access directory %s: %w", rootDir, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", rootDir)
	}

	var dbFiles []string

	// Define walk function
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// If a file or directory can't be accessed, log the error but continue
			fmt.Printf("Warning: Cannot access %s: %v\n", path, err)
			return nil
		}

		// If it's a directory and not the root directory, and we're not recursively searching, skip it
		if info.IsDir() && path != rootDir && !recursive {
			return filepath.SkipDir
		}

		// Check if file extension is .db
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".db" {
			dbFiles = append(dbFiles, path)
		}

		return nil
	}

	// Start traversal
	err = filepath.Walk(rootDir, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("error traversing directory: %w", err)
	}

	if len(dbFiles) == 0 {
		return nil, fmt.Errorf("no .db files found")
	}

	return dbFiles, nil
}

func (s *Service) DecryptDBFiles(dataDir string, workDir string, key string, version int) error {

	dbfiles, err := s.FindDBFiles(dataDir, true)
	if err != nil {
		return err
	}

	for _, dbfile := range dbfiles {
		output := filepath.Join(workDir, dbfile[len(dataDir):])
		if err := util.PrepareDir(filepath.Dir(output)); err != nil {
			return err
		}
		if err := wechat.DecryptDBFileToFile(dbfile, output, key, version); err != nil {
			if err == wechat.ErrAlreadyDecrypted {
				continue
			}
			return err
		}
	}

	return nil
}

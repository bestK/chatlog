package wechat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat"
	"github.com/sjzar/chatlog/internal/wechat/decrypt"
	"github.com/sjzar/chatlog/pkg/util"

	"github.com/rs/zerolog/log"
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
func (s *Service) GetWeChatInstances() []*wechat.Account {
	wechat.Load()
	return wechat.GetAccounts()
}

// GetDataKey extracts the encryption key from a WeChat process
func (s *Service) GetDataKey(info *wechat.Account) (string, error) {
	if info == nil {
		return "", fmt.Errorf("no WeChat instance selected")
	}

	key, err := info.GetKey(context.Background())
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
			log.Err(err).Msgf("Warning: Cannot access %s", path)
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

func (s *Service) DecryptDBFiles(dataDir string, workDir string, key string, platform string, version int) error {

	ctx := context.Background()

	dbfiles, err := s.FindDBFiles(dataDir, true)
	if err != nil {
		return err
	}

	decryptor, err := decrypt.NewDecryptor(platform, version)
	if err != nil {
		return err
	}

	for _, dbfile := range dbfiles {
		output := filepath.Join(workDir, dbfile[len(dataDir):])
		if err := util.PrepareDir(filepath.Dir(output)); err != nil {
			return err
		}

		outputFile, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer outputFile.Close()

		if err := decryptor.Decrypt(ctx, dbfile, key, outputFile); err != nil {
			log.Err(err).Msgf("failed to decrypt %s", dbfile)
			if err == errors.ErrAlreadyDecrypted {
				if data, err := os.ReadFile(dbfile); err == nil {
					outputFile.Write(data)
				}
				continue
			}
			continue
			// return err
		}
	}

	return nil
}

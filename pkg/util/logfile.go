package util

import (
	"fmt"
	"os"
	"path/filepath"
)

func OpenLogFile() (*os.File, string, error) {
	var errs []error
	for _, dir := range logDirCandidates() {
		if err := PrepareDir(dir); err != nil {
			errs = append(errs, err)
			continue
		}
		path := filepath.Join(dir, "chatlog.log")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return f, path, nil
	}
	if len(errs) == 0 {
		return nil, "", fmt.Errorf("open log file failed")
	}
	return nil, "", fmt.Errorf("open log file failed: %w", errs[0])
}

func logDirCandidates() []string {
	defaultDir := DefaultWorkDir("")
	tempDir := filepath.Join(os.TempDir(), "chatlog")
	if defaultDir == tempDir {
		return []string{defaultDir}
	}
	return []string{defaultDir, tempDir}
}

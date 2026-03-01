package util

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
)

func Is64Bit(handle windows.Handle) (bool, error) {
	var is32Bit bool
	if err := windows.IsWow64Process(handle, &is32Bit); err != nil {
		return false, fmt.Errorf("检查进程位数失败: %w", err)
	}
	return !is32Bit, nil
}

// CleanExtendedLengthPath 移除 Windows 扩展长度路径前缀。
// 处理两种格式：
//   - `\\?\C:\path` → `C:\path` （普通扩展长度路径）
//   - `\\?\UNC\server\share\path` → `\\server\share\path` （UNC 扩展长度路径）
//
// 在 Parallels Desktop 等虚拟机环境中，p.OpenFiles() 返回的路径可能包含 UNC 格式，
// 如果不正确处理会导致路径无效（如 `UNC\psf\Home\...`）。
func CleanExtendedLengthPath(path string) string {
	// \\?\UNC\server\share → \\server\share
	if strings.HasPrefix(path, `\\?\UNC\`) {
		return `\\` + path[8:]
	}
	// \\?\C:\path → C:\path
	if strings.HasPrefix(path, `\\?\`) {
		return path[4:]
	}
	return path
}

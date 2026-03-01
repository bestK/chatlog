//go:build !windows

package util

// CleanExtendedLengthPath 在非 Windows 平台上直接返回原始路径。
// Windows 实现位于 os_windows.go，用于处理 \\?\UNC\ 等扩展长度路径前缀。
func CleanExtendedLengthPath(path string) string {
	return path
}

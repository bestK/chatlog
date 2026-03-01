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

func NormalizeDataDirPath(path string) string {
	trimmedPath := trimWindowsExtendedPathPrefix(path)
	return mapParallelsHomeUNCToDrive(trimmedPath)
}

func trimWindowsExtendedPathPrefix(path string) string {
	if strings.HasPrefix(path, `\\?\UNC\`) {
		return `\\` + path[len(`\\?\UNC\`):]
	}
	if strings.HasPrefix(path, `\\?\`) {
		return path[len(`\\?\`):]
	}
	return path
}

func mapParallelsHomeUNCToDrive(path string) string {
	const parallelsHomeUNC = `\\psf\Home\`
	if strings.HasPrefix(path, parallelsHomeUNC) {
		return `C:\Mac\Home\` + path[len(parallelsHomeUNC):]
	}
	return path
}

func CleanExtendedLengthPath(path string) string {
	return NormalizeDataDirPath(path)
}

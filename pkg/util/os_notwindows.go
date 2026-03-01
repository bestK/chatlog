//go:build !windows

package util

func NormalizeDataDirPath(path string) string {
	return path
}

func CleanExtendedLengthPath(path string) string {
	return NormalizeDataDirPath(path)
}

package main

import (
	"strings"

	"golang.org/x/sys/windows"
)

// FindProcessIdsByName 根据进程名查找所有匹配的进程 ID
func FindProcessIdsByName(processName string) ([]uint32, error) {
	processIDs, err := EnumProcesses()
	if err != nil {
		return nil, err
	}

	var matchedPIDs []uint32
	targetName := strings.ToLower(processName)

	for _, pid := range processIDs {
		if pid == 0 {
			continue
		}

		hProcess, err := OpenProcess(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, false, pid)
		if err != nil {
			continue
		}

		name, err := GetModuleBaseName(hProcess)
		CloseHandle(hProcess)

		if err != nil {
			continue
		}

		if strings.ToLower(name) == targetName {
			matchedPIDs = append(matchedPIDs, pid)
		}
	}

	return matchedPIDs, nil
}

// IsProcessRunning 检查指定进程是否正在运行
func IsProcessRunning(processName string) bool {
	pids, err := FindProcessIdsByName(processName)
	if err != nil {
		return false
	}
	return len(pids) > 0
}

// OpenProcessByPID 通过 PID 打开进程（全权限）
func OpenProcessByPID(pid uint32) (windows.Handle, error) {
	return OpenProcess(PROCESS_ALL_ACCESS, false, pid)
}

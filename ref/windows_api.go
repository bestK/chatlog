package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	PROCESS_ALL_ACCESS        = 0x1F0FFF
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ           = 0x0010

	MEM_COMMIT  = 0x1000
	MEM_PRIVATE = 0x20000
	MEM_MAPPED  = 0x40000
	MEM_IMAGE   = 0x1000000

	PAGE_NOACCESS = 0x01
	PAGE_GUARD    = 0x100
)

// MEMORY_BASIC_INFORMATION 内存区域信息结构
type MEMORY_BASIC_INFORMATION struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	PartitionId       uint16
	_                 uint16
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
}

var (
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modPsapi    = windows.NewLazySystemDLL("psapi.dll")

	procVirtualQueryEx     = modKernel32.NewProc("VirtualQueryEx")
	procReadProcessMemory  = modKernel32.NewProc("ReadProcessMemory")
	procEnumProcesses      = modPsapi.NewProc("EnumProcesses")
	procGetModuleBaseNameW = modPsapi.NewProc("GetModuleBaseNameW")
)

// OpenProcess 打开进程
func OpenProcess(desiredAccess uint32, inheritHandle bool, processID uint32) (windows.Handle, error) {
	inherit := uint32(0)
	if inheritHandle {
		inherit = 1
	}
	handle, err := windows.OpenProcess(desiredAccess, inherit != 0, processID)
	return handle, err
}

// CloseHandle 关闭句柄
func CloseHandle(handle windows.Handle) error {
	return windows.CloseHandle(handle)
}

// VirtualQueryEx 查询进程内存区域
func VirtualQueryEx(hProcess windows.Handle, lpAddress uintptr, lpBuffer *MEMORY_BASIC_INFORMATION) (uintptr, error) {
	size := unsafe.Sizeof(*lpBuffer)
	ret, _, err := procVirtualQueryEx.Call(
		uintptr(hProcess),
		lpAddress,
		uintptr(unsafe.Pointer(lpBuffer)),
		size,
	)
	if ret == 0 {
		return 0, err
	}
	return ret, nil
}

// ReadProcessMemory 读取进程内存
func ReadProcessMemory(hProcess windows.Handle, baseAddress uintptr, buffer []byte) (int, error) {
	var bytesRead uintptr
	ret, _, err := procReadProcessMemory.Call(
		uintptr(hProcess),
		baseAddress,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&bytesRead)),
	)
	if ret == 0 {
		return 0, err
	}
	return int(bytesRead), nil
}

// EnumProcesses 枚举所有进程 ID
func EnumProcesses() ([]uint32, error) {
	processIDs := make([]uint32, 1024)
	var cbNeeded uint32

	ret, _, err := procEnumProcesses.Call(
		uintptr(unsafe.Pointer(&processIDs[0])),
		uintptr(len(processIDs)*4),
		uintptr(unsafe.Pointer(&cbNeeded)),
	)
	if ret == 0 {
		return nil, err
	}

	count := cbNeeded / 4
	return processIDs[:count], nil
}

// GetModuleBaseName 获取进程的模块名
func GetModuleBaseName(hProcess windows.Handle) (string, error) {
	var buffer [260]uint16 // MAX_PATH

	ret, _, err := procGetModuleBaseNameW.Call(
		uintptr(hProcess),
		0,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if ret == 0 {
		return "", err
	}

	return windows.UTF16ToString(buffer[:ret]), nil
}

// GetMemoryRegions 获取进程的所有可读内存区域
func GetMemoryRegions(hProcess windows.Handle) []MemoryRegion {
	var regions []MemoryRegion
	var address uintptr = 0
	var mbi MEMORY_BASIC_INFORMATION

	for {
		// 64位地址空间上限
		if address >= 0x7FFFFFFFFFFF {
			break
		}

		result, err := VirtualQueryEx(hProcess, address, &mbi)
		if result == 0 || err != nil {
			break
		}

		// 只收集已提交的可读内存
		if mbi.State == MEM_COMMIT &&
			isReadableProtect(mbi.Protect) &&
			isCandidateRegionType(mbi.Type) {
			regions = append(regions, MemoryRegion{
				BaseAddress: mbi.BaseAddress,
				RegionSize:  mbi.RegionSize,
			})
		}

		// 移动到下一个区域
		nextAddress := address + mbi.RegionSize
		if nextAddress <= address {
			break
		}
		address = nextAddress
	}

	return regions
}

// MemoryRegion 内存区域
type MemoryRegion struct {
	BaseAddress uintptr
	RegionSize  uintptr
}

func isReadableProtect(protect uint32) bool {
	if protect == PAGE_NOACCESS {
		return false
	}
	if (protect & PAGE_GUARD) != 0 {
		return false
	}
	return true
}

func isCandidateRegionType(memType uint32) bool {
	return memType == MEM_PRIVATE || memType == MEM_MAPPED || memType == MEM_IMAGE
}

// ReadMemoryChunk 读取指定地址的内存块
func ReadMemoryChunk(hProcess windows.Handle, address uintptr, size int) ([]byte, error) {
	buffer := make([]byte, size)
	bytesRead, err := ReadProcessMemory(hProcess, address, buffer)
	if err != nil {
		return nil, err
	}
	if bytesRead == 0 {
		return nil, fmt.Errorf("读取内存失败：读取字节数为0")
	}
	return buffer[:bytesRead], nil
}

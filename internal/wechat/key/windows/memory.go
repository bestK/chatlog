//go:build windows

package windows

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// MemoryRegion represents a memory region in a process.
type MemoryRegion struct {
	BaseAddress uintptr
	RegionSize  uintptr
	State       uint32
	Protect     uint32
	Type        uint32
}

// GetMemoryRegions enumerates all committed memory regions of a process.
func GetMemoryRegions(handle windows.Handle) []MemoryRegion {
	var regions []MemoryRegion
	var address uintptr = 0x10000
	const maxAddr uintptr = 0x7FFFFFFFFFFF

	for address < maxAddr {
		var mbi windows.MemoryBasicInformation
		err := windows.VirtualQueryEx(handle, address, &mbi, unsafe.Sizeof(mbi))
		if err != nil {
			break
		}

		if mbi.State == windows.MEM_COMMIT {
			regions = append(regions, MemoryRegion{
				BaseAddress: mbi.BaseAddress,
				RegionSize:  mbi.RegionSize,
				State:       mbi.State,
				Protect:     mbi.Protect,
				Type:        mbi.Type,
			})
		}
		address = mbi.BaseAddress + mbi.RegionSize
	}
	return regions
}

// ReadMemoryChunk reads a chunk of memory from a process.
func ReadMemoryChunk(handle windows.Handle, address uintptr, size int) ([]byte, error) {
	buffer := make([]byte, size)
	var bytesRead uintptr
	err := windows.ReadProcessMemory(handle, address, &buffer[0], uintptr(size), &bytesRead)
	if err != nil {
		return nil, err
	}
	return buffer[:bytesRead], nil
}
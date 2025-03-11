package wechat

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/sjzar/chatlog/pkg/util"

	"github.com/shirou/gopsutil/v4/process"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

const (
	V3ModuleName = "WeChatWin.dll"
	V3DBFile     = "Msg\\Misc.db"
	V4DBFile     = "db_storage\\message\\message_0.db"

	MaxWorkers = 16

	// Windows memory protection constants
	MEM_PRIVATE = 0x20000
)

// Common error definitions
var (
	ErrWeChatOffline    = errors.New("wechat is not logged in")
	ErrOpenProcess      = errors.New("failed to open process")
	ErrReaddecryptor    = errors.New("failed to read database header")
	ErrCheckProcessBits = errors.New("failed to check process architecture")
	ErrFindWeChatDLL    = errors.New("WeChatWin.dll module not found")
	ErrNoValidKey       = errors.New("no valid key found")
	ErrInvalidFilePath  = errors.New("invalid file path format")
)

// GetKey is the entry point for retrieving the WeChat database key
func (i *Info) GetKey() (string, error) {
	if i.Status == StatusOffline {
		return "", ErrWeChatOffline
	}

	// Choose key retrieval method based on WeChat version
	if i.Version.FileMajorVersion == 4 {
		return i.getKeyV4()
	}
	return i.getKeyV3()
}

// initialize initializes WeChat information
func (i *Info) initialize(p *process.Process) error {
	files, err := p.OpenFiles()
	if err != nil {
		log.Error("Failed to get open file list: ", err)
		return err
	}

	dbPath := V3DBFile
	if i.Version.FileMajorVersion == 4 {
		dbPath = V4DBFile
	}

	for _, f := range files {
		if strings.HasSuffix(f.Path, dbPath) {
			filePath := f.Path[4:] // Remove "\\?\" prefix
			parts := strings.Split(filePath, string(filepath.Separator))
			if len(parts) < 4 {
				log.Debug("Invalid file path format: " + filePath)
				continue
			}

			i.Status = StatusOnline
			if i.Version.FileMajorVersion == 4 {
				i.DataDir = strings.Join(parts[:len(parts)-3], string(filepath.Separator))
				i.AccountName = parts[len(parts)-4]
			} else {
				i.DataDir = strings.Join(parts[:len(parts)-2], string(filepath.Separator))
				i.AccountName = parts[len(parts)-3]
			}
		}
	}

	return nil
}

// getKeyV3 retrieves the database key for WeChat V3 version
func (i *Info) getKeyV3() (string, error) {
	// Read database header for key validation
	dbPath := filepath.Join(i.DataDir, V3DBFile)
	decryptor, err := NewDecryptor(dbPath, i.Version.FileMajorVersion)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrReaddecryptor, err)
	}
	log.Debug("V3 database path: ", dbPath)

	// Open WeChat process
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, i.PID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOpenProcess, err)
	}
	defer windows.CloseHandle(handle)

	// Check process architecture
	is64Bit, err := util.Is64Bit(handle)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCheckProcessBits, err)
	}

	// Create context to control all goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels for memory data and results
	memoryChannel := make(chan []byte, 100)
	resultChannel := make(chan string, 1)

	// Determine number of worker goroutines
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2
	}
	if workerCount > MaxWorkers {
		workerCount = MaxWorkers
	}
	log.Debug("Starting ", workerCount, " workers for V3 key search")

	// Start consumer goroutines
	var workerWaitGroup sync.WaitGroup
	workerWaitGroup.Add(workerCount)
	for index := 0; index < workerCount; index++ {
		go func() {
			defer workerWaitGroup.Done()
			workerV3(ctx, handle, decryptor, is64Bit, memoryChannel, resultChannel)
		}()
	}

	// Start producer goroutine
	var producerWaitGroup sync.WaitGroup
	producerWaitGroup.Add(1)
	go func() {
		defer producerWaitGroup.Done()
		defer close(memoryChannel) // Close channel when producer is done
		err := i.findMemoryV3(ctx, handle, memoryChannel)
		if err != nil {
			log.Error(err)
		}
	}()

	// Wait for producer and consumers to complete
	go func() {
		producerWaitGroup.Wait()
		workerWaitGroup.Wait()
		close(resultChannel)
	}()

	// Wait for result
	result, ok := <-resultChannel
	if ok && result != "" {
		i.Key = result
		return result, nil
	}

	return "", ErrNoValidKey
}

// getKeyV4 retrieves the database key for WeChat V4 version
func (i *Info) getKeyV4() (string, error) {
	// Read database header for key validation
	dbPath := filepath.Join(i.DataDir, V4DBFile)
	decryptor, err := NewDecryptor(dbPath, i.Version.FileMajorVersion)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrReaddecryptor, err)
	}
	log.Debug("V4 database path: ", dbPath)

	// Open process handle
	handle, err := windows.OpenProcess(windows.PROCESS_VM_READ|windows.PROCESS_QUERY_INFORMATION, false, i.PID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOpenProcess, err)
	}
	defer windows.CloseHandle(handle)

	// Create context to control all goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels for memory data and results
	memoryChannel := make(chan []byte, 100)
	resultChannel := make(chan string, 1)

	// Determine number of worker goroutines
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2
	}
	if workerCount > MaxWorkers {
		workerCount = MaxWorkers
	}
	log.Debug("Starting ", workerCount, " workers for V4 key search")

	// Start consumer goroutines
	var workerWaitGroup sync.WaitGroup
	workerWaitGroup.Add(workerCount)
	for index := 0; index < workerCount; index++ {
		go func() {
			defer workerWaitGroup.Done()
			workerV4(ctx, handle, decryptor, memoryChannel, resultChannel)
		}()
	}

	// Start producer goroutine
	var producerWaitGroup sync.WaitGroup
	producerWaitGroup.Add(1)
	go func() {
		defer producerWaitGroup.Done()
		defer close(memoryChannel) // Close channel when producer is done
		err := i.findMemoryV4(ctx, handle, memoryChannel)
		if err != nil {
			log.Error(err)
		}
	}()

	// Wait for producer and consumers to complete
	go func() {
		producerWaitGroup.Wait()
		workerWaitGroup.Wait()
		close(resultChannel)
	}()

	// Wait for result
	result, ok := <-resultChannel
	if ok && result != "" {
		i.Key = result
		return result, nil
	}

	return "", ErrNoValidKey
}

// findMemoryV3 searches for writable memory regions in WeChatWin.dll for V3 version
func (i *Info) findMemoryV3(ctx context.Context, handle windows.Handle, memoryChannel chan<- []byte) error {
	// Find WeChatWin.dll module
	module, isFound := FindModule(i.PID, V3ModuleName)
	if !isFound {
		return ErrFindWeChatDLL
	}
	log.Debug("Found WeChatWin.dll module at base address: 0x", fmt.Sprintf("%X", module.ModBaseAddr))

	// Read writable memory regions
	baseAddr := uintptr(module.ModBaseAddr)
	endAddr := baseAddr + uintptr(module.ModBaseSize)
	currentAddr := baseAddr

	for currentAddr < endAddr {
		var mbi windows.MemoryBasicInformation
		err := windows.VirtualQueryEx(handle, currentAddr, &mbi, unsafe.Sizeof(mbi))
		if err != nil {
			break
		}

		// Skip small memory regions
		if mbi.RegionSize < 100*1024 {
			currentAddr += uintptr(mbi.RegionSize)
			continue
		}

		// Check if memory region is writable
		isWritable := (mbi.Protect & (windows.PAGE_READWRITE | windows.PAGE_WRITECOPY | windows.PAGE_EXECUTE_READWRITE | windows.PAGE_EXECUTE_WRITECOPY)) > 0
		if isWritable && uint32(mbi.State) == windows.MEM_COMMIT {
			// Calculate region size, ensure it doesn't exceed DLL bounds
			regionSize := uintptr(mbi.RegionSize)
			if currentAddr+regionSize > endAddr {
				regionSize = endAddr - currentAddr
			}

			// Read writable memory region
			memory := make([]byte, regionSize)
			if err = windows.ReadProcessMemory(handle, currentAddr, &memory[0], regionSize, nil); err == nil {
				select {
				case memoryChannel <- memory:
					log.Debug("Sent memory region for analysis, size: ", regionSize, " bytes")
				case <-ctx.Done():
					return nil
				}
			}
		}

		// Move to next memory region
		currentAddr = uintptr(mbi.BaseAddress) + uintptr(mbi.RegionSize)
	}

	return nil
}

// workerV3 processes memory regions to find V3 version key
func workerV3(ctx context.Context, handle windows.Handle, decryptor *Decryptor, is64Bit bool, memoryChannel <-chan []byte, resultChannel chan<- string) {

	// Define search pattern
	keyPattern := []byte{0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ptrSize := 8
	littleEndianFunc := binary.LittleEndian.Uint64

	// Adjust for 32-bit process
	if !is64Bit {
		keyPattern = keyPattern[:4]
		ptrSize = 4
		littleEndianFunc = func(b []byte) uint64 { return uint64(binary.LittleEndian.Uint32(b)) }
	}

	for {
		select {
		case <-ctx.Done():
			return
		case memory, ok := <-memoryChannel:
			if !ok {
				return
			}

			index := len(memory)
			for {
				select {
				case <-ctx.Done():
					return // Exit if result found
				default:
				}

				// Find pattern from end to beginning
				index = bytes.LastIndex(memory[:index], keyPattern)
				if index == -1 || index-ptrSize < 0 {
					break
				}

				// Extract and validate pointer value
				ptrValue := littleEndianFunc(memory[index-ptrSize : index])
				if ptrValue > 0x10000 && ptrValue < 0x7FFFFFFFFFFF {
					if key := validateKey(handle, decryptor, ptrValue); key != "" {
						select {
						case resultChannel <- key:
							log.Debug("Valid key found for V3 database")
						default:
						}
						return
					}
				}
				index -= 1 // Continue searching from previous position
			}
		}
	}
}

// findMemoryV4 searches for writable memory regions for V4 version
func (i *Info) findMemoryV4(ctx context.Context, handle windows.Handle, memoryChannel chan<- []byte) error {
	// Define search range
	minAddr := uintptr(0x10000)    // Process space usually starts from 0x10000
	maxAddr := uintptr(0x7FFFFFFF) // 32-bit process space limit

	if runtime.GOARCH == "amd64" {
		maxAddr = uintptr(0x7FFFFFFFFFFF) // 64-bit process space limit
	}
	log.Debug("Scanning memory regions from 0x", fmt.Sprintf("%X", minAddr), " to 0x", fmt.Sprintf("%X", maxAddr))

	currentAddr := minAddr

	for currentAddr < maxAddr {
		var memInfo windows.MemoryBasicInformation
		err := windows.VirtualQueryEx(handle, currentAddr, &memInfo, unsafe.Sizeof(memInfo))
		if err != nil {
			break
		}

		// Skip small memory regions
		if memInfo.RegionSize < 1024*1024 {
			currentAddr += uintptr(memInfo.RegionSize)
			continue
		}

		// Check if memory region is readable and private
		if memInfo.State == windows.MEM_COMMIT && (memInfo.Protect&windows.PAGE_READWRITE) != 0 && memInfo.Type == MEM_PRIVATE {
			// Calculate region size, ensure it doesn't exceed limit
			regionSize := uintptr(memInfo.RegionSize)
			if currentAddr+regionSize > maxAddr {
				regionSize = maxAddr - currentAddr
			}

			// Read memory region
			memory := make([]byte, regionSize)
			if err = windows.ReadProcessMemory(handle, currentAddr, &memory[0], regionSize, nil); err == nil {
				select {
				case memoryChannel <- memory:
					log.Debug("Sent memory region for analysis, size: ", regionSize, " bytes")
				case <-ctx.Done():
					return nil
				}
			}
		}

		// Move to next memory region
		currentAddr = uintptr(memInfo.BaseAddress) + uintptr(memInfo.RegionSize)
	}

	return nil
}

// workerV4 processes memory regions to find V4 version key
func workerV4(ctx context.Context, handle windows.Handle, decryptor *Decryptor, memoryChannel <-chan []byte, resultChannel chan<- string) {

	// Define search pattern for V4
	keyPattern := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x2F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	ptrSize := 8
	littleEndianFunc := binary.LittleEndian.Uint64

	for {
		select {
		case <-ctx.Done():
			return
		case memory, ok := <-memoryChannel:
			if !ok {
				return
			}

			index := len(memory)
			for {
				select {
				case <-ctx.Done():
					return // Exit if result found
				default:
				}

				// Find pattern from end to beginning
				index = bytes.LastIndex(memory[:index], keyPattern)
				if index == -1 || index-ptrSize < 0 {
					break
				}

				// Extract and validate pointer value
				ptrValue := littleEndianFunc(memory[index-ptrSize : index])
				if ptrValue > 0x10000 && ptrValue < 0x7FFFFFFFFFFF {
					if key := validateKey(handle, decryptor, ptrValue); key != "" {
						select {
						case resultChannel <- key:
							log.Debug("Valid key found for V4 database")
						default:
						}
						return
					}
				}
				index -= 1 // Continue searching from previous position
			}
		}
	}
}

// validateKey validates a single key candidate
func validateKey(handle windows.Handle, decryptor *Decryptor, addr uint64) string {
	keyData := make([]byte, 0x20) // 32-byte key
	if err := windows.ReadProcessMemory(handle, uintptr(addr), &keyData[0], uintptr(len(keyData)), nil); err != nil {
		return ""
	}

	// Validate key against database header
	if decryptor.Validate(keyData) {
		return hex.EncodeToString(keyData)
	}

	return ""
}

// FindModule searches for a specified module in the process
// Used to find WeChatWin.dll module for V3 version
func FindModule(pid uint32, name string) (module windows.ModuleEntry32, isFound bool) {
	// Create module snapshot
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE|windows.TH32CS_SNAPMODULE32, pid)
	if err != nil {
		log.Debug("Failed to create module snapshot: ", err)
		return module, false
	}
	defer windows.CloseHandle(snapshot)

	// Initialize module entry structure
	module.Size = uint32(windows.SizeofModuleEntry32)

	// Get the first module
	if err := windows.Module32First(snapshot, &module); err != nil {
		log.Debug("Failed to get first module: ", err)
		return module, false
	}

	// Iterate through all modules to find WeChatWin.dll
	for ; err == nil; err = windows.Module32Next(snapshot, &module) {
		if windows.UTF16ToString(module.Module[:]) == name {
			return module, true
		}
	}
	return module, false
}

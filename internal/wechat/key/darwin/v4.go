package darwin

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/sjzar/chatlog/internal/wechat/decrypt"
	"github.com/sjzar/chatlog/internal/wechat/key/darwin/glance"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

const (
	MaxWorkers = 8
)

type V4Extractor struct {
	validator *decrypt.Validator
}

func NewV4Extractor() *V4Extractor {
	return &V4Extractor{}
}

func (e *V4Extractor) Extract(ctx context.Context, proc *model.Process) (string, error) {
	if proc.Status == model.StatusOffline {
		return "", fmt.Errorf("WeChat is offline")
	}

	// Check if SIP is disabled, as it's required for memory reading on macOS
	if !glance.IsSIPDisabled() {
		return "", fmt.Errorf("System Integrity Protection (SIP) is enabled, cannot read process memory")
	}

	if e.validator == nil {
		return "", fmt.Errorf("validator not set")
	}

	// Create context to control all goroutines
	searchCtx, cancel := context.WithCancel(ctx)
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
	logrus.Debug("Starting ", workerCount, " workers for V4 key search")

	// Start consumer goroutines
	var workerWaitGroup sync.WaitGroup
	workerWaitGroup.Add(workerCount)
	for index := 0; index < workerCount; index++ {
		go func() {
			defer workerWaitGroup.Done()
			e.worker(searchCtx, memoryChannel, resultChannel)
		}()
	}

	// Start producer goroutine
	var producerWaitGroup sync.WaitGroup
	producerWaitGroup.Add(1)
	go func() {
		defer producerWaitGroup.Done()
		defer close(memoryChannel) // Close channel when producer is done
		err := e.findMemory(searchCtx, uint32(proc.PID), memoryChannel)
		if err != nil {
			logrus.Error(err)
		}
	}()

	// Wait for producer and consumers to complete
	go func() {
		producerWaitGroup.Wait()
		workerWaitGroup.Wait()
		close(resultChannel)
	}()

	// Wait for result
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result, ok := <-resultChannel:
		if ok && result != "" {
			return result, nil
		}
	}

	return "", fmt.Errorf("no valid key found")
}

// findMemory searches for memory regions using Glance
func (e *V4Extractor) findMemory(ctx context.Context, pid uint32, memoryChannel chan<- []byte) error {
	// Initialize a Glance instance to read process memory
	g := glance.NewGlance(pid)

	// Read memory data
	memory, err := g.Read()
	if err != nil {
		return fmt.Errorf("failed to read process memory: %w", err)
	}

	logrus.Debug("Read memory region, size: ", len(memory), " bytes")

	// Send memory data to channel for processing
	select {
	case memoryChannel <- memory:
		logrus.Debug("Sent memory region for analysis")
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// worker processes memory regions to find V4 version key
func (e *V4Extractor) worker(ctx context.Context, memoryChannel <-chan []byte, resultChannel chan<- string) {
	keyPattern := []byte{0x20, 0x66, 0x74, 0x73, 0x35, 0x28, 0x25, 0x00}

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
					return // Exit if context cancelled
				default:
				}

				// Find pattern from end to beginning
				index = bytes.LastIndex(memory[:index], keyPattern)
				if index == -1 {
					break // No more matches found
				}

				// Check if we have enough space for the key
				if index+16+32 > len(memory) {
					index -= 1
					continue
				}

				// Extract the key data, which is 16 bytes after the pattern and 32 bytes long
				keyOffset := index + 16
				keyData := memory[keyOffset : keyOffset+32]

				// Validate key against database header
				if e.validator.Validate(keyData) {
					select {
					case resultChannel <- hex.EncodeToString(keyData):
						logrus.Debug("Valid key found for V4 database")
						return
					default:
					}
				}

				index -= 1
			}
		}
	}
}

func (e *V4Extractor) SetValidate(validator *decrypt.Validator) {
	e.validator = validator
}

package wechat

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

// Constants for WeChat database decryption
const (
	// Common constants
	PageSize     = 4096
	KeySize      = 32
	SaltSize     = 16
	AESBlockSize = 16
	SQLiteHeader = "SQLite format 3\x00"

	// Version specific constants
	V3IterCount = 64000
	V4IterCount = 256000

	IVSize         = 16 // Same for both versions
	HmacSHA1Size   = 20 // Used in V3
	HmacSHA512Size = 64 // Used in V4
)

// Error definitions
var (
	ErrHashVerificationFailed = errors.New("hash verification failed")
	ErrInvalidVersion         = errors.New("invalid version, must be 3 or 4")
	ErrInvalidKey             = errors.New("invalid key format")
	ErrIncorrectKey           = errors.New("incorrect decryption key")
	ErrReadFile               = errors.New("failed to read database file")
	ErrOpenFile               = errors.New("failed to open database file")
	ErrIncompleteRead         = errors.New("incomplete header read")
	ErrCreateCipher           = errors.New("failed to create cipher")
	ErrDecodeKey              = errors.New("failed to decode hex key")
	ErrWriteOutput            = errors.New("failed to write output")
	ErrSeekFile               = errors.New("failed to seek in file")
	ErrOperationCanceled      = errors.New("operation was canceled")
	ErrAlreadyDecrypted       = errors.New("file is already decrypted")
)

// Decryptor handles the decryption of WeChat database files
type Decryptor struct {
	// Database file path
	dbPath string

	// Database properties
	version int
	salt    []byte
	page1   []byte
	reserve int

	// Calculated fields
	hashFunc    func() hash.Hash
	hmacSize    int
	currentPage int64
	totalPages  int64
}

// NewDecryptor creates a new Decryptor for the specified database file and version
func NewDecryptor(dbPath string, version int) (*Decryptor, error) {
	// Validate version
	if version != 3 && version != 4 {
		return nil, ErrInvalidVersion
	}

	// Open database file
	fp, err := os.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOpenFile, err)
	}
	defer fp.Close()

	// Get file size
	fileInfo, err := fp.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	// Calculate total pages
	fileSize := fileInfo.Size()
	totalPages := fileSize / PageSize
	if fileSize%PageSize > 0 {
		totalPages++
	}

	// Read first page
	buffer := make([]byte, PageSize)
	n, err := io.ReadFull(fp, buffer)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReadFile, err)
	}
	if n != PageSize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrIncompleteRead, PageSize, n)
	}

	// Check if file is already decrypted
	if bytes.Equal(buffer[:len(SQLiteHeader)-1], []byte(SQLiteHeader[:len(SQLiteHeader)-1])) {
		return nil, ErrAlreadyDecrypted
	}

	// Initialize hash function and HMAC size based on version
	var hashFunc func() hash.Hash
	var hmacSize int

	if version == 4 {
		hashFunc = sha512.New
		hmacSize = HmacSHA512Size
	} else {
		hashFunc = sha1.New
		hmacSize = HmacSHA1Size
	}

	// Calculate reserve size and MAC offset
	reserve := IVSize + hmacSize
	if reserve%AESBlockSize != 0 {
		reserve = ((reserve / AESBlockSize) + 1) * AESBlockSize
	}

	return &Decryptor{
		dbPath:     dbPath,
		version:    version,
		salt:       buffer[:SaltSize],
		page1:      buffer,
		reserve:    reserve,
		hashFunc:   hashFunc,
		hmacSize:   hmacSize,
		totalPages: totalPages,
	}, nil
}

// GetTotalPages returns the total number of pages in the database
func (d *Decryptor) GetTotalPages() int64 {
	return d.totalPages
}

// Validate checks if the provided key is valid for this database
func (d *Decryptor) Validate(key []byte) bool {
	if len(key) != KeySize {
		return false
	}
	_, macKey := d.calcPBKDF2Key(key)
	return d.validate(macKey)
}

func (d *Decryptor) calcPBKDF2Key(key []byte) ([]byte, []byte) {
	// Generate encryption key from password
	var encKey []byte
	if d.version == 4 {
		encKey = pbkdf2.Key(key, d.salt, V4IterCount, KeySize, sha512.New)
	} else {
		encKey = pbkdf2.Key(key, d.salt, V3IterCount, KeySize, sha1.New)
	}

	// Generate MAC key
	macSalt := xorBytes(d.salt, 0x3a)
	macKey := pbkdf2.Key(encKey, macSalt, 2, KeySize, d.hashFunc)
	return encKey, macKey
}

func (d *Decryptor) validate(macKey []byte) bool {
	// Calculate HMAC
	hashMac := hmac.New(d.hashFunc, macKey)

	dataEnd := PageSize - d.reserve + IVSize
	hashMac.Write(d.page1[SaltSize:dataEnd])

	// Page number is fixed as 1
	pageNoBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(pageNoBytes, 1)
	hashMac.Write(pageNoBytes)

	calculatedMAC := hashMac.Sum(nil)
	storedMAC := d.page1[dataEnd : dataEnd+d.hmacSize]

	return hmac.Equal(calculatedMAC, storedMAC)
}

// Decrypt decrypts the database using the provided key and writes the result to the writer
func (d *Decryptor) Decrypt(ctx context.Context, hexKey string, w io.Writer) error {
	// Decode key
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDecodeKey, err)
	}

	encKey, macKey := d.calcPBKDF2Key(key)

	// Validate key first
	if !d.validate(macKey) {
		return ErrIncorrectKey
	}

	// Open input file
	dbFile, err := os.Open(d.dbPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOpenFile, err)
	}
	defer dbFile.Close()

	// Write SQLite header to output
	_, err = w.Write([]byte(SQLiteHeader))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWriteOutput, err)
	}

	// Process each page
	pageBuf := make([]byte, PageSize)
	d.currentPage = 0

	for curPage := int64(0); curPage < d.totalPages; curPage++ {
		// Check for cancellation before processing each page
		select {
		case <-ctx.Done():
			return ErrOperationCanceled
		default:
			// Continue processing
		}

		// For the first page, we need to skip the salt
		if curPage == 0 {
			// Read the first page
			_, err = io.ReadFull(dbFile, pageBuf)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrReadFile, err)
			}
		} else {
			// Read a full page
			n, err := io.ReadFull(dbFile, pageBuf)
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// Handle last partial page
					if n > 0 {
						// Process partial page
						// For simplicity, we'll just break here
						break
					}
				}
				return fmt.Errorf("%w: %v", ErrReadFile, err)
			}
		}

		// check if page contains only zeros (v3 & v4 both have this behavior)
		allZeros := true
		for _, b := range pageBuf {
			if b != 0 {
				allZeros = false
				break
			}
		}

		if allZeros {
			// Write the zeros page to output
			_, err = w.Write(pageBuf)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrWriteOutput, err)
			}

			// Update progress
			d.currentPage = curPage + 1
			continue

			// // Set current page to total pages to indicate completion
			// d.currentPage = d.totalPages
			// return nil
		}

		// Decrypt the page
		decryptedPage, err := d.decryptPage(encKey, macKey, pageBuf, curPage)
		if err != nil {
			return err
		}

		// Write decrypted page to output
		_, err = w.Write(decryptedPage)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrWriteOutput, err)
		}

		// Update progress
		d.currentPage = curPage + 1
	}

	return nil
}

// decryptPage decrypts a single page of the database
func (d *Decryptor) decryptPage(key, macKey []byte, pageBuf []byte, pageNum int64) ([]byte, error) {
	offset := 0
	if pageNum == 0 {
		offset = SaltSize
	}

	// Verify HMAC
	mac := hmac.New(d.hashFunc, macKey)
	mac.Write(pageBuf[offset : PageSize-d.reserve+IVSize])

	// Convert page number and update HMAC
	pageNumBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(pageNumBytes, uint32(pageNum+1))
	mac.Write(pageNumBytes)

	hashMac := mac.Sum(nil)

	hashMacStartOffset := PageSize - d.reserve + IVSize
	hashMacEndOffset := hashMacStartOffset + len(hashMac)

	if !bytes.Equal(hashMac, pageBuf[hashMacStartOffset:hashMacEndOffset]) {
		return nil, ErrHashVerificationFailed
	}

	// Decrypt content using AES-256-CBC
	iv := pageBuf[PageSize-d.reserve : PageSize-d.reserve+IVSize]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreateCipher, err)
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	// Create a copy of encrypted data for decryption
	encrypted := make([]byte, PageSize-d.reserve-offset)
	copy(encrypted, pageBuf[offset:PageSize-d.reserve])

	// Decrypt in place
	mode.CryptBlocks(encrypted, encrypted)

	// Combine decrypted data with reserve part
	decryptedPage := append(encrypted, pageBuf[PageSize-d.reserve:PageSize]...)

	return decryptedPage, nil
}

// xorBytes performs XOR operation on each byte of the array with the specified byte
func xorBytes(a []byte, b byte) []byte {
	result := make([]byte, len(a))
	for i := range a {
		result[i] = a[i] ^ b
	}
	return result
}

// Utility functions for backward compatibility

// DecryptDBFile decrypts a WeChat database file and returns the decrypted content
func DecryptDBFile(dbPath string, hexKey string, version int) ([]byte, error) {
	// Create a buffer to store the decrypted content
	var buf bytes.Buffer

	// Create a decryptor
	d, err := NewDecryptor(dbPath, version)
	if err != nil {
		return nil, err
	}

	// Decrypt the database
	err = d.Decrypt(context.Background(), hexKey, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecryptDBFileToFile decrypts a WeChat database file and saves the result to the specified output file
func DecryptDBFileToFile(dbPath, outputPath, hexKey string, version int) error {
	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Create a decryptor
	d, err := NewDecryptor(dbPath, version)
	if err != nil {
		return err
	}

	// Decrypt the database
	return d.Decrypt(context.Background(), hexKey, outputFile)
}

// ValidateDBKey validates if the provided key is correct for the database
func ValidateDBKey(dbPath string, hexKey string, version int) bool {
	// Create a decryptor
	d, err := NewDecryptor(dbPath, version)
	if err != nil {
		return false
	}

	// Decode key
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return false
	}

	// Validate the key
	return d.Validate(key)
}

package dat2img

import (
	"crypto/aes"
	"encoding/binary"
	"testing"
)

var (
	testAESKey = []byte("c43524e3f4ee209e") // 16-byte ASCII key (same format as real key)
	testXorKey = byte(0x88)

	// Minimal recognizable JPEG: FF D8 FF ... FF D9
	testJPGPayload = func() []byte {
		p := make([]byte, 88)
		p[0], p[1], p[2] = 0xFF, 0xD8, 0xFF
		p[86], p[87] = 0xFF, 0xD9
		return p
	}()

	// Minimal recognizable PNG
	testPNGPayload = func() []byte {
		p := make([]byte, 88)
		copy(p, []byte{0x89, 0x50, 0x4E, 0x47})
		copy(p[76:], []byte("IEND\xae\x42\x60\x82"))
		return p
	}()
)

// buildV2Dat constructs a synthetic V2 .dat file for testing.
// Layout: [6B magic][4B aes_size LE][4B xor_size LE][1B pad][AES-ECB padded][raw][XOR]
func buildV2Dat(plaintext []byte, aesSize, xorSize int, aesKey []byte, xorKey byte) []byte {
	if aesSize+xorSize > len(plaintext) {
		panic("aesSize + xorSize > len(plaintext)")
	}

	aesPlain := plaintext[:aesSize]
	rawPlain := plaintext[aesSize : len(plaintext)-xorSize]
	xorPlain := plaintext[len(plaintext)-xorSize:]

	// PKCS7 pad
	padLen := aes.BlockSize - (len(aesPlain) % aes.BlockSize)
	padded := make([]byte, len(aesPlain)+padLen)
	copy(padded, aesPlain)
	for i := len(aesPlain); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	// AES-ECB encrypt
	block, _ := aes.NewCipher(aesKey[:16])
	aesCipher := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		block.Encrypt(aesCipher[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}

	// XOR encrypt
	xorCipher := make([]byte, len(xorPlain))
	for i, b := range xorPlain {
		xorCipher[i] = b ^ xorKey
	}

	// Build header
	header := make([]byte, 15)
	copy(header[:6], []byte{0x07, 0x08, 0x56, 0x32, 0x08, 0x07}) // V2 magic
	binary.LittleEndian.PutUint32(header[6:10], uint32(aesSize))
	binary.LittleEndian.PutUint32(header[10:14], uint32(xorSize))
	header[14] = 0x00

	result := make([]byte, 0, len(header)+len(aesCipher)+len(rawPlain)+len(xorCipher))
	result = append(result, header...)
	result = append(result, aesCipher...)
	result = append(result, rawPlain...)
	result = append(result, xorCipher...)
	return result
}

func TestV2RoundTrip(t *testing.T) {
	// Save and restore global state
	origKey := V4Format2.AesKey
	origXor := V4XorKey
	defer func() {
		V4Format2.AesKey = origKey
		V4XorKey = origXor
	}()

	V4Format2.AesKey = testAESKey
	V4XorKey = testXorKey

	dat := buildV2Dat(testJPGPayload, 32, 16, testAESKey, testXorKey)

	out, ext, err := Dat2Image(dat)
	if err != nil {
		t.Fatalf("Dat2Image failed: %v", err)
	}
	if ext != "jpg" {
		t.Fatalf("expected ext=jpg, got %s", ext)
	}
	if len(out) != len(testJPGPayload) {
		t.Fatalf("output length mismatch: got %d, want %d", len(out), len(testJPGPayload))
	}
	for i := range out {
		if out[i] != testJPGPayload[i] {
			t.Fatalf("byte mismatch at offset %d: got 0x%02x, want 0x%02x", i, out[i], testJPGPayload[i])
		}
	}
}

func TestV2PNG(t *testing.T) {
	origKey := V4Format2.AesKey
	origXor := V4XorKey
	defer func() {
		V4Format2.AesKey = origKey
		V4XorKey = origXor
	}()

	V4Format2.AesKey = testAESKey
	V4XorKey = testXorKey

	dat := buildV2Dat(testPNGPayload, 32, 16, testAESKey, testXorKey)

	out, ext, err := Dat2Image(dat)
	if err != nil {
		t.Fatalf("Dat2Image failed: %v", err)
	}
	if ext != "png" {
		t.Fatalf("expected ext=png, got %s", ext)
	}
	if len(out) != len(testPNGPayload) {
		t.Fatalf("output length mismatch: got %d, want %d", len(out), len(testPNGPayload))
	}
}

func TestV2WrongKeyFails(t *testing.T) {
	origKey := V4Format2.AesKey
	origXor := V4XorKey
	defer func() {
		V4Format2.AesKey = origKey
		V4XorKey = origXor
	}()

	V4Format2.AesKey = []byte("wrongkey00000000")
	V4XorKey = testXorKey

	dat := buildV2Dat(testJPGPayload, 32, 16, testAESKey, testXorKey)

	_, _, err := Dat2Image(dat)
	if err == nil {
		t.Fatal("expected error with wrong key, got nil")
	}
}

func TestV1FixedKey(t *testing.T) {
	origXor := V4XorKey
	defer func() { V4XorKey = origXor }()

	V4XorKey = testXorKey
	v1Key := []byte("cfcd208495d565ef")

	// Build V1 dat (same structure, different magic and fixed key)
	dat := buildV1Dat(testJPGPayload, 32, 16, v1Key, testXorKey)

	out, ext, err := Dat2Image(dat)
	if err != nil {
		t.Fatalf("V1 Dat2Image failed: %v", err)
	}
	if ext != "jpg" {
		t.Fatalf("expected ext=jpg, got %s", ext)
	}
	if len(out) != len(testJPGPayload) {
		t.Fatalf("output length mismatch: got %d, want %d", len(out), len(testJPGPayload))
	}
}

func buildV1Dat(plaintext []byte, aesSize, xorSize int, aesKey []byte, xorKey byte) []byte {
	if aesSize+xorSize > len(plaintext) {
		panic("aesSize + xorSize > len(plaintext)")
	}

	aesPlain := plaintext[:aesSize]
	rawPlain := plaintext[aesSize : len(plaintext)-xorSize]
	xorPlain := plaintext[len(plaintext)-xorSize:]

	padLen := aes.BlockSize - (len(aesPlain) % aes.BlockSize)
	padded := make([]byte, len(aesPlain)+padLen)
	copy(padded, aesPlain)
	for i := len(aesPlain); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	block, _ := aes.NewCipher(aesKey[:16])
	aesCipher := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		block.Encrypt(aesCipher[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}

	xorCipher := make([]byte, len(xorPlain))
	for i, b := range xorPlain {
		xorCipher[i] = b ^ xorKey
	}

	header := make([]byte, 15)
	copy(header[:6], []byte{0x07, 0x08, 0x56, 0x31, 0x08, 0x07}) // V1 magic
	binary.LittleEndian.PutUint32(header[6:10], uint32(aesSize))
	binary.LittleEndian.PutUint32(header[10:14], uint32(xorSize))
	header[14] = 0x00

	result := make([]byte, 0, len(header)+len(aesCipher)+len(rawPlain)+len(xorCipher))
	result = append(result, header...)
	result = append(result, aesCipher...)
	result = append(result, rawPlain...)
	result = append(result, xorCipher...)
	return result
}

func TestSetAesKeyFormats(t *testing.T) {
	origKey := V4Format2.AesKey
	defer func() { V4Format2.AesKey = origKey }()

	// 16-char ASCII key
	SetAesKey("c43524e3f4ee209e")
	if string(V4Format2.AesKey) != "c43524e3f4ee209e" {
		t.Fatalf("16-char key: got %q", V4Format2.AesKey)
	}

	// 32-char valid hex key
	SetAesKey("c43524e3f4ee209ec43524e3f4ee209e")
	if len(V4Format2.AesKey) != 16 {
		t.Fatalf("32-char hex key: expected 16 bytes, got %d", len(V4Format2.AesKey))
	}

	// 32-char non-hex key (contains g-z) → should take first 16 chars
	SetAesKey("a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6")
	if string(V4Format2.AesKey) != "a1b2c3d4e5f6g7h8" {
		t.Fatalf("32-char non-hex key: got %q, want first 16 chars", V4Format2.AesKey)
	}

	// Empty key should not change
	prev := V4Format2.AesKey
	SetAesKey("")
	if string(V4Format2.AesKey) != string(prev) {
		t.Fatal("empty key should not change stored key")
	}
}
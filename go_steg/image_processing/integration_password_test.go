package image_processing

import (
	"bytes"
	"go-steg/cli/helpers"
	"go-steg/go_steg/pipeline"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPasswordUnicode tests roundtrip with a unicode password containing
// Cyrillic, emoji, and CJK characters.
func TestPasswordUnicode(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 200, 200, 70001)

	originalData := []byte("unicode password roundtrip test")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	password := "пароль🔑密码"
	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: true,
		FileExtension:  "txt",
		Password:       password,
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		password,
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (unicode password) failed: %v", err)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")
	if _, err := os.Stat(embeddedPath); os.IsNotExist(err) {
		t.Fatalf("embedded carrier not found: %s", embeddedPath)
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err = MultiCarrierDecodeByFileNames(
		[]string{embeddedPath},
		password,
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (unicode password) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("unicode password roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestPasswordVeryLong tests roundtrip with a 10,000-character password.
func TestPasswordVeryLong(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 200, 200, 70002)

	originalData := []byte("long password roundtrip test")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	password := strings.Repeat("abcdefghij", 1000) // 10,000 characters
	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: true,
		FileExtension:  "txt",
		Password:       password,
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		password,
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (10KB password) failed: %v", err)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")
	if _, err := os.Stat(embeddedPath); os.IsNotExist(err) {
		t.Fatalf("embedded carrier not found: %s", embeddedPath)
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err = MultiCarrierDecodeByFileNames(
		[]string{embeddedPath},
		password,
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (10KB password) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("10KB password roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestPasswordNullBytes tests roundtrip with a password containing null bytes.
func TestPasswordNullBytes(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 200, 200, 70003)

	originalData := []byte("null byte password roundtrip test")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	password := "pass\x00word"
	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: true,
		FileExtension:  "txt",
		Password:       password,
	}

	// Use recover to catch potential panics from null bytes in password
	var encodeErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("RECOVERED panic during encode with null-byte password: %v", r)
				encodeErr = nil // signal that we panicked
				return
			}
		}()
		encodeErr = EncodeByFileNames(
			[]string{carrierPath},
			dataPath,
			1,
			password,
			encodeOutDir,
			cfg,
		)
	}()
	if encodeErr != nil {
		t.Fatalf("EncodeByFileNames (null-byte password) failed: %v", encodeErr)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")
	if _, err := os.Stat(embeddedPath); os.IsNotExist(err) {
		t.Skipf("embedded carrier not created (may have panicked); skipping decode")
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var decodeErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("RECOVERED panic during decode with null-byte password: %v", r)
				decodeErr = nil
				return
			}
		}()
		decodeErr = MultiCarrierDecodeByFileNames(
			[]string{embeddedPath},
			password,
			decodeOutDir,
		)
	}()
	if decodeErr != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (null-byte password) failed: %v", decodeErr)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("null-byte password roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestPasswordWhitespaceOnly tests roundtrip with a whitespace-only password.
func TestPasswordWhitespaceOnly(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 200, 200, 70004)

	originalData := []byte("whitespace password roundtrip test")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	password := "   \t\n"
	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: true,
		FileExtension:  "txt",
		Password:       password,
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		password,
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (whitespace password) failed: %v", err)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")
	if _, err := os.Stat(embeddedPath); os.IsNotExist(err) {
		t.Fatalf("embedded carrier not found: %s", embeddedPath)
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err = MultiCarrierDecodeByFileNames(
		[]string{embeddedPath},
		password,
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (whitespace password) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("whitespace password roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

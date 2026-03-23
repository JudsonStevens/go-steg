package image_processing

import (
	"bytes"
	"go-steg/cli/helpers"
	"go-steg/go_steg/pipeline"
	"go-steg/go_steg/reed_solomon"
	"os"
	"path/filepath"
	"testing"
)

// TestMaskBitDepth1 tests roundtrip encoding/decoding with UseMask=true and BitDepth=1.
func TestMaskBitDepth1(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 300, 300, 60001)

	originalData := []byte("mask + bit depth 1 roundtrip test")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      1,
		FileExtension: "txt",
		Password:      "testpassword",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"testpassword",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (mask + bit depth 1) failed: %v", err)
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
		"testpassword",
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (mask + bit depth 1) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("mask + bit depth 1 roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestMaskBitDepth3 tests roundtrip encoding/decoding with UseMask=true and BitDepth=3.
func TestMaskBitDepth3(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 300, 300, 60002)

	originalData := []byte("mask + bit depth 3 roundtrip test")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      3,
		FileExtension: "txt",
		Password:      "testpassword",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"testpassword",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (mask + bit depth 3) failed: %v", err)
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
		"testpassword",
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (mask + bit depth 3) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("mask + bit depth 3 roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestMaskBitDepth4 tests roundtrip encoding/decoding with UseMask=true and BitDepth=4.
func TestMaskBitDepth4(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 300, 300, 60003)

	originalData := []byte("mask + bit depth 4 roundtrip test")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      4,
		FileExtension: "txt",
		Password:      "testpassword",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"testpassword",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (mask + bit depth 4) failed: %v", err)
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
		"testpassword",
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (mask + bit depth 4) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("mask + bit depth 4 roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestMaskHuffmanRS tests roundtrip with UseMask=true, HuffmanEnabled=true, and RSEnabled=true.
func TestMaskHuffmanRS(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 400, 400, 60004)

	originalData := []byte("mask + huffman + reed-solomon roundtrip test payload")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: true,
		RSEnabled:      true,
		RSLevel:        reed_solomon.High,
		FileExtension:  "txt",
		Password:       "testpassword",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"testpassword",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (mask + huffman + RS) failed: %v", err)
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
		"testpassword",
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (mask + huffman + RS) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("mask + huffman + RS roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestMaskInsufficientCapacity tests that encoding fails with an error when the
// carrier is too small to hold the data with mask enabled. The mask reduces
// capacity significantly because it filters out pixels based on a password-derived pattern.
func TestMaskInsufficientCapacity(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	// Use a small 40x40 carrier — mask will reduce capacity substantially.
	carrierPath := filepath.Join(tmpDir, "small_carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 40, 40, 60005)

	// Create data that is too large for a 40x40 carrier with mask enabled.
	// A 40x40 image has 40*6=240 data pixels (height 40, header 34, so 6 data rows).
	// With mask ~50% are usable, and at bit depth 2 that's ~240*0.5*3*2/8 = ~90 bytes max.
	// Use 500 bytes to ensure it exceeds capacity.
	largeData := make([]byte, 500)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	dataPath := filepath.Join(tmpDir, "large_data.bin")
	createDataFile(t, dataPath, largeData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      1,
		FileExtension: "bin",
		Password:      "testpassword",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"testpassword",
		encodeOutDir,
		cfg,
	)
	if err == nil {
		t.Error("expected error for mask + insufficient capacity, but encoding succeeded")
	} else {
		t.Logf("got expected error for mask + insufficient capacity: %v", err)
	}
}

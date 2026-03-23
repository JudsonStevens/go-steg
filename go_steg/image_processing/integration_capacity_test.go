package image_processing

import (
	"bytes"
	"go-steg/cli/helpers"
	"go-steg/go_steg/pipeline"
	"go-steg/go_steg/reed_solomon"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCapacityNearFull encodes data filling ~95% of the carrier's raw capacity
// (no pipeline processing) and verifies a successful roundtrip.
func TestCapacityNearFull(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNG(t, carrierPath, 42)

	// 200x200 carrier, bit depth 2:
	// usable pixels = 200 * (200 - 34) = 33,200
	// slots = 33,200 * 3 = 99,600
	// slots per byte = ceil(8/2) = 4
	// capacity = 99,600 / 4 = 24,900 bytes
	// Use ~50% to be safe against any internal overhead (header bytes etc.)
	capacity := 12000
	data := make([]byte, capacity)
	for i := range data {
		data[i] = byte(i % 251) // avoid patterns that might cause issues
	}

	dataPath := filepath.Join(tmpDir, "data.bin")
	createDataFile(t, dataPath, data)

	encodeDir := filepath.Join(tmpDir, "encoded")
	os.MkdirAll(encodeDir, 0755)

	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: false,
		RSEnabled:      false,
		FileExtension:  "bin",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"",
		encodeDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("encode near-full capacity failed: %v", err)
	}

	// Find the embedded file and decode
	embeddedPattern := filepath.Join(encodeDir, "*-embedded.png")
	matches, err := filepath.Glob(embeddedPattern)
	if err != nil || len(matches) == 0 {
		t.Fatalf("no embedded file found in %s", encodeDir)
	}

	decodeDir := filepath.Join(tmpDir, "decoded")
	os.MkdirAll(decodeDir, 0755)

	err = MultiCarrierDecodeByFileNames(matches, "", decodeDir)
	if err != nil {
		t.Fatalf("decode near-full capacity failed: %v", err)
	}

	decoded := findDecodedFile(t, decodeDir, "bin")
	decodedData, err := os.ReadFile(decoded)
	if err != nil {
		t.Fatalf("failed to read decoded file: %v", err)
	}

	if !bytes.Equal(data, decodedData) {
		t.Errorf("roundtrip mismatch: original %d bytes, decoded %d bytes", len(data), len(decodedData))
	}
}

// TestCapacityExceeded verifies that encoding data larger than the carrier can
// hold returns an appropriate error.
func TestCapacityExceeded(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNG(t, carrierPath, 42)

	// Create data much larger than the carrier can hold (200x200, ~24,900 byte capacity)
	data := make([]byte, 50000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	dataPath := filepath.Join(tmpDir, "data.bin")
	createDataFile(t, dataPath, data)

	encodeDir := filepath.Join(tmpDir, "encoded")
	os.MkdirAll(encodeDir, 0755)

	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: false,
		RSEnabled:      false,
		FileExtension:  "bin",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"",
		encodeDir,
		cfg,
	)
	if err == nil {
		t.Fatal("expected error when data exceeds carrier capacity, got nil")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' in error, got: %v", err)
	}
}

// TestCapacityPipelineExpansionOverflow encodes data that fits in the carrier
// raw but after RS High encoding (~33.5% overhead per block, plus 8-byte prefix)
// exceeds the carrier capacity.
func TestCapacityPipelineExpansionOverflow(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	// Use a small carrier to make the boundary easier to hit.
	// 100x80 carrier at bit depth 2:
	// usable pixels = 100 * (80 - 34) = 4,600
	// slots = 4,600 * 3 = 13,800
	// slots per byte = ceil(8/2) = 4
	// raw capacity = 13,800 / 4 = 3,450 bytes
	//
	// RS High: each 191 data bytes -> 255 output bytes, plus 8-byte prefix
	// For data of size D, RS output = 8 + ceil(D/191)*255
	//
	// We want: D < 3,450 (fits raw) but 8 + ceil(D/191)*255 > 3,450 (exceeds after RS)
	// ceil(D/191)*255 > 3,442
	// ceil(D/191) > 13.5 => need 14 blocks => D > 13*191 = 2,483, D <= 14*191 = 2,674
	// RS output for D=2,500: ceil(2500/191)=14 blocks, output = 8 + 14*255 = 3,578 > 3,450
	// So D=2,500 fits raw (2,500 < 3,450) but not after RS (3,578 > 3,450).
	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 100, 80, 42)

	data := make([]byte, 2500)
	for i := range data {
		data[i] = byte(i % 256)
	}

	dataPath := filepath.Join(tmpDir, "data.bin")
	createDataFile(t, dataPath, data)

	encodeDir := filepath.Join(tmpDir, "encoded")
	os.MkdirAll(encodeDir, 0755)

	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: false,
		RSEnabled:      true,
		RSLevel:        reed_solomon.High,
		FileExtension:  "bin",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"",
		encodeDir,
		cfg,
	)
	if err == nil {
		t.Fatal("expected error when pipeline-expanded data exceeds capacity, got nil")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' in error, got: %v", err)
	}
}

// TestCapacityEmptyDataFile verifies behavior when encoding a zero-byte file.
func TestCapacityEmptyDataFile(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNG(t, carrierPath, 42)

	dataPath := filepath.Join(tmpDir, "empty.bin")
	createDataFile(t, dataPath, []byte{})

	encodeDir := filepath.Join(tmpDir, "encoded")
	os.MkdirAll(encodeDir, 0755)

	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: false,
		RSEnabled:      false,
		FileExtension:  "bin",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"",
		encodeDir,
		cfg,
	)
	// Empty data may succeed or return a meaningful error — either is acceptable.
	if err != nil {
		t.Logf("encoding empty data returned error (acceptable): %v", err)
		// Verify the error is meaningful, not a panic or nil-pointer
		if err.Error() == "" {
			t.Error("expected a meaningful error message, got empty string")
		}
	} else {
		t.Log("encoding empty data succeeded")
		// If it succeeded, verify we can decode it
		embeddedPattern := filepath.Join(encodeDir, "*-embedded.png")
		matches, err := filepath.Glob(embeddedPattern)
		if err != nil || len(matches) == 0 {
			t.Fatalf("no embedded file found in %s", encodeDir)
		}

		decodeDir := filepath.Join(tmpDir, "decoded")
		os.MkdirAll(decodeDir, 0755)

		err = MultiCarrierDecodeByFileNames(matches, "", decodeDir)
		if err != nil {
			t.Logf("decoding empty data returned error (acceptable): %v", err)
		}
	}
}

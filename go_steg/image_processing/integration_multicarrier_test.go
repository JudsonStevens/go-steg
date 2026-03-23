package image_processing

import (
	"bytes"
	"go-steg/cli/helpers"
	"go-steg/go_steg/pipeline"
	"go-steg/go_steg/reed_solomon"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// TestMultiCarrierWrongOrder encodes with [carrier1, carrier2] and decodes with
// [carrier2, carrier1]. The decoded data should either error or differ from the original.
func TestMultiCarrierWrongOrder(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrier1Path := filepath.Join(tmpDir, "carrier1.png")
	carrier2Path := filepath.Join(tmpDir, "carrier2.png")
	createCarrierPNG(t, carrier1Path, 1001)
	createCarrierPNG(t, carrier2Path, 1002)

	originalData := make([]byte, 300)
	rng := rand.New(rand.NewSource(100))
	for i := range originalData {
		originalData[i] = byte(rng.Intn(256))
	}
	dataPath := filepath.Join(tmpDir, "data.bin")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      2,
		FileExtension: "bin",
	}

	err := EncodeByFileNames(
		[]string{carrier1Path, carrier2Path},
		dataPath,
		1,
		"",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames failed: %v", err)
	}

	embedded1 := filepath.Join(encodeOutDir, "carrier1-0-embedded.png")
	embedded2 := filepath.Join(encodeOutDir, "carrier2-1-embedded.png")

	for _, p := range []string{embedded1, embedded2} {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Fatalf("expected embedded file not found: %s", p)
		}
	}

	// Decode with carriers in WRONG order: [carrier2, carrier1] instead of [carrier1, carrier2]
	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	decErr := MultiCarrierDecodeByFileNames(
		[]string{embedded2, embedded1},
		"",
		decodeOutDir,
	)
	if decErr != nil {
		// Error is acceptable — wrong order should not decode cleanly.
		t.Logf("decode with wrong carrier order returned error (expected): %v", decErr)
		return
	}

	// If no error, verify the output does NOT match the original.
	decodedPath := findDecodedFile(t, decodeOutDir, "bin")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if bytes.Equal(decodedData, originalData) {
		t.Error("decoded data matches original despite wrong carrier order — order should matter")
	} else {
		t.Logf("decoded data differs from original as expected (original len=%d, decoded len=%d)",
			len(originalData), len(decodedData))
	}
}

// TestMultiCarrierMissingCarrier encodes with 2 carriers but decodes with only 1.
// The decoded data should either error or not match the original.
func TestMultiCarrierMissingCarrier(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrier1Path := filepath.Join(tmpDir, "carrier1.png")
	carrier2Path := filepath.Join(tmpDir, "carrier2.png")
	createCarrierPNG(t, carrier1Path, 2001)
	createCarrierPNG(t, carrier2Path, 2002)

	originalData := make([]byte, 300)
	rng := rand.New(rand.NewSource(200))
	for i := range originalData {
		originalData[i] = byte(rng.Intn(256))
	}
	dataPath := filepath.Join(tmpDir, "data.bin")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      2,
		FileExtension: "bin",
	}

	err := EncodeByFileNames(
		[]string{carrier1Path, carrier2Path},
		dataPath,
		1,
		"",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames failed: %v", err)
	}

	embedded1 := filepath.Join(encodeOutDir, "carrier1-0-embedded.png")

	if _, err := os.Stat(embedded1); os.IsNotExist(err) {
		t.Fatalf("expected embedded file not found: %s", embedded1)
	}

	// Decode with only the first carrier (missing the second).
	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	decErr := MultiCarrierDecodeByFileNames(
		[]string{embedded1},
		"",
		decodeOutDir,
	)
	if decErr != nil {
		// Error is acceptable — missing a carrier should fail.
		t.Logf("decode with missing carrier returned error (expected): %v", decErr)
		return
	}

	// If no error, verify the output does NOT match the original.
	decodedPath := findDecodedFile(t, decodeOutDir, "bin")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if bytes.Equal(decodedData, originalData) {
		t.Error("decoded data matches original despite missing a carrier — multi-carrier split is not effective")
	} else {
		t.Logf("decoded data differs from original as expected (original len=%d, decoded len=%d)",
			len(originalData), len(decodedData))
	}
}

// TestMultiCarrierThreeCarriers encodes data split across 3 carriers and verifies roundtrip.
func TestMultiCarrierThreeCarriers(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrier1Path := filepath.Join(tmpDir, "carrier1.png")
	carrier2Path := filepath.Join(tmpDir, "carrier2.png")
	carrier3Path := filepath.Join(tmpDir, "carrier3.png")
	createCarrierPNG(t, carrier1Path, 3001)
	createCarrierPNG(t, carrier2Path, 3002)
	createCarrierPNG(t, carrier3Path, 3003)

	originalData := make([]byte, 450)
	rng := rand.New(rand.NewSource(300))
	for i := range originalData {
		originalData[i] = byte(rng.Intn(256))
	}
	dataPath := filepath.Join(tmpDir, "data.bin")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      2,
		FileExtension: "bin",
	}

	err := EncodeByFileNames(
		[]string{carrier1Path, carrier2Path, carrier3Path},
		dataPath,
		1,
		"",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames with 3 carriers failed: %v", err)
	}

	embedded1 := filepath.Join(encodeOutDir, "carrier1-0-embedded.png")
	embedded2 := filepath.Join(encodeOutDir, "carrier2-1-embedded.png")
	embedded3 := filepath.Join(encodeOutDir, "carrier3-2-embedded.png")

	for _, p := range []string{embedded1, embedded2, embedded3} {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Fatalf("expected embedded file not found: %s", p)
		}
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	decErr := MultiCarrierDecodeByFileNames(
		[]string{embedded1, embedded2, embedded3},
		"",
		decodeOutDir,
	)
	if decErr != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames with 3 carriers failed: %v", decErr)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "bin")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("3-carrier roundtrip mismatch: original len=%d, decoded len=%d",
			len(originalData), len(decodedData))
		showSnippet(t, "original", originalData)
		showSnippet(t, "decoded", decodedData)
	}
}

// TestMultiCarrierWithPipelineFeatures encodes with 2 carriers + Huffman + RS enabled,
// decodes, and verifies roundtrip.
func TestMultiCarrierWithPipelineFeatures(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrier1Path := filepath.Join(tmpDir, "carrier1.png")
	carrier2Path := filepath.Join(tmpDir, "carrier2.png")
	createCarrierPNG(t, carrier1Path, 4001)
	createCarrierPNG(t, carrier2Path, 4002)

	originalData := []byte("Multi-carrier with Huffman and Reed-Solomon pipeline features test payload data.")
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
	}

	err := EncodeByFileNames(
		[]string{carrier1Path, carrier2Path},
		dataPath,
		1,
		"",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames with pipeline features failed: %v", err)
	}

	embedded1 := filepath.Join(encodeOutDir, "carrier1-0-embedded.png")
	embedded2 := filepath.Join(encodeOutDir, "carrier2-1-embedded.png")

	for _, p := range []string{embedded1, embedded2} {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Fatalf("expected embedded file not found: %s", p)
		}
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	decErr := MultiCarrierDecodeByFileNames(
		[]string{embedded1, embedded2},
		"",
		decodeOutDir,
	)
	if decErr != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames with pipeline features failed: %v", decErr)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("multi-carrier + pipeline features roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestMultiCarrierUnevenDataSplit encodes data whose length is not evenly divisible
// by the carrier count, and verifies roundtrip integrity.
func TestMultiCarrierUnevenDataSplit(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrier1Path := filepath.Join(tmpDir, "carrier1.png")
	carrier2Path := filepath.Join(tmpDir, "carrier2.png")
	createCarrierPNG(t, carrier1Path, 5001)
	createCarrierPNG(t, carrier2Path, 5002)

	// 301 bytes is not evenly divisible by 2 carriers
	originalData := make([]byte, 301)
	rng := rand.New(rand.NewSource(500))
	for i := range originalData {
		originalData[i] = byte(rng.Intn(256))
	}
	dataPath := filepath.Join(tmpDir, "data.bin")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      2,
		FileExtension: "bin",
	}

	err := EncodeByFileNames(
		[]string{carrier1Path, carrier2Path},
		dataPath,
		1,
		"",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames with uneven data failed: %v", err)
	}

	embedded1 := filepath.Join(encodeOutDir, "carrier1-0-embedded.png")
	embedded2 := filepath.Join(encodeOutDir, "carrier2-1-embedded.png")

	for _, p := range []string{embedded1, embedded2} {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Fatalf("expected embedded file not found: %s", p)
		}
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	decErr := MultiCarrierDecodeByFileNames(
		[]string{embedded1, embedded2},
		"",
		decodeOutDir,
	)
	if decErr != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames with uneven data failed: %v", decErr)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "bin")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("uneven data split roundtrip mismatch: original len=%d, decoded len=%d",
			len(originalData), len(decodedData))
		showSnippet(t, "original", originalData)
		showSnippet(t, "decoded", decodedData)
	}
}

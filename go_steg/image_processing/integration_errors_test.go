package image_processing

import (
	"go-steg/cli/helpers"
	"go-steg/go_steg/pipeline"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// TestCorruptedHeader encodes data normally, then corrupts the header pixels
// (y=0..13, x=0) in the embedded PNG. Decoding should either return an error
// or produce garbage output (not the original data).
func TestCorruptedHeader(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNG(t, carrierPath, 70001)

	originalData := []byte("Header corruption test payload data.")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("failed to create encode output dir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      2,
		FileExtension: "txt",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames failed: %v", err)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")

	// Open the embedded PNG and corrupt header pixels
	f, err := os.Open(embeddedPath)
	if err != nil {
		t.Fatalf("failed to open embedded file: %v", err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatalf("failed to decode embedded PNG: %v", err)
	}

	rgbaImg, ok := img.(*image.RGBA)
	if !ok {
		// Convert to RGBA
		bounds := img.Bounds()
		rgbaImg = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgbaImg.Set(x, y, img.At(x, y))
			}
		}
	}

	// Corrupt header pixels at (0, 0..13) with random values
	rng := rand.New(rand.NewSource(99999))
	for y := 0; y < 14; y++ {
		rgbaImg.SetRGBA(0, y, color.RGBA{
			R: uint8(rng.Intn(256)),
			G: uint8(rng.Intn(256)),
			B: uint8(rng.Intn(256)),
			A: 255,
		})
	}

	// Save corrupted image back
	corruptedPath := filepath.Join(tmpDir, "corrupted.png")
	cf, err := os.Create(corruptedPath)
	if err != nil {
		t.Fatalf("failed to create corrupted file: %v", err)
	}
	if err := png.Encode(cf, rgbaImg); err != nil {
		cf.Close()
		t.Fatalf("failed to encode corrupted PNG: %v", err)
	}
	cf.Close()

	// Attempt to decode from corrupted image — expect error or garbage
	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("failed to create decode output dir: %v", err)
	}

	var decodeErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("decode panicked (expected with corrupted header): %v", r)
			}
		}()
		decodeErr = MultiCarrierDecodeByFileNames(
			[]string{corruptedPath},
			"",
			decodeOutDir,
		)
	}()

	if decodeErr != nil {
		t.Logf("decode returned error as expected: %v", decodeErr)
		return
	}

	// If no error, check that the output is garbage (not the original data)
	matches, _ := filepath.Glob(filepath.Join(decodeOutDir, "decoded_file-*"))
	if len(matches) == 0 {
		t.Log("no decoded file produced after header corruption — acceptable behavior")
		return
	}

	decodedData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Logf("could not read decoded file: %v", err)
		return
	}

	if string(decodedData) == string(originalData) {
		t.Error("decoded data matches original despite corrupted header — header corruption had no effect")
	} else {
		t.Log("decoded data differs from original as expected with corrupted header")
	}
}

// TestNonImageCarrier tries to encode into a file that is not an image
// (a text file renamed to .png). The encode should return an error.
func TestNonImageCarrier(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	// Create a text file disguised as a PNG
	fakePNGPath := filepath.Join(tmpDir, "notanimage.png")
	if err := os.WriteFile(fakePNGPath, []byte("This is not a PNG image, just plain text."), 0644); err != nil {
		t.Fatalf("failed to write fake PNG: %v", err)
	}

	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, []byte("test payload"))

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("failed to create encode output dir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      2,
		FileExtension: "txt",
	}

	var encodeErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("encode panicked with non-image carrier (acceptable): %v", r)
			}
		}()
		encodeErr = EncodeByFileNames(
			[]string{fakePNGPath},
			dataPath,
			1,
			"",
			encodeOutDir,
			cfg,
		)
	}()

	if encodeErr == nil {
		t.Error("expected error when encoding into a non-image carrier, but got nil")
	} else {
		t.Logf("got expected error: %v", encodeErr)
	}
}

// TestChecksumValidation encodes data with Huffman enabled, then decodes with
// the wrong password. The decode should either error or produce garbage.
func TestChecksumValidation(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNG(t, carrierPath, 70003)

	originalData := []byte("Checksum validation test payload with some content.")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("failed to create encode output dir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: true,
		FileExtension:  "txt",
		Password:       "correctpassword",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"correctpassword",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames failed: %v", err)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")

	// Decode with wrong password — should error or produce garbage
	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("failed to create decode output dir: %v", err)
	}

	var decodeErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("decode panicked with wrong password (acceptable): %v", r)
			}
		}()
		decodeErr = MultiCarrierDecodeByFileNames(
			[]string{embeddedPath},
			"wrongpassword",
			decodeOutDir,
		)
	}()

	if decodeErr != nil {
		t.Logf("decode with wrong password returned error (expected): %v", decodeErr)
		return
	}

	// If no error, verify output doesn't match original
	matches, _ := filepath.Glob(filepath.Join(decodeOutDir, "decoded_file-*"))
	if len(matches) == 0 {
		t.Log("no decoded file produced with wrong password — acceptable")
		return
	}

	decodedData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Logf("could not read decoded file: %v", err)
		return
	}

	if string(decodedData) == string(originalData) {
		t.Error("decoded data matches original despite wrong password — Huffman password protection is ineffective")
	} else {
		t.Logf("decoded data differs from original as expected (original=%d bytes, decoded=%d bytes)",
			len(originalData), len(decodedData))
	}
}

// TestTruncatedCarrier creates a valid PNG, truncates it, then tries to decode.
// This should return an error since the PNG data is incomplete.
func TestTruncatedCarrier(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	// Create a valid PNG first
	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNG(t, carrierPath, 70004)

	// Read the full PNG data
	fullData, err := os.ReadFile(carrierPath)
	if err != nil {
		t.Fatalf("failed to read carrier: %v", err)
	}

	// Truncate to roughly half
	truncatedPath := filepath.Join(tmpDir, "truncated.png")
	truncLen := len(fullData) / 2
	if err := os.WriteFile(truncatedPath, fullData[:truncLen], 0644); err != nil {
		t.Fatalf("failed to write truncated PNG: %v", err)
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("failed to create decode output dir: %v", err)
	}

	var decodeErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("decode panicked with truncated carrier (acceptable): %v", r)
			}
		}()
		decodeErr = MultiCarrierDecodeByFileNames(
			[]string{truncatedPath},
			"",
			decodeOutDir,
		)
	}()

	if decodeErr == nil {
		t.Error("expected error when decoding from truncated carrier, but got nil")
	} else {
		t.Logf("got expected error from truncated carrier: %v", decodeErr)
	}
}

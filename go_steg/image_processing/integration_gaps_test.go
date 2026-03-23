package image_processing

import (
	"bytes"
	"go-steg/cli/helpers"
	"go-steg/go_steg/pipeline"
	"go-steg/go_steg/reed_solomon"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// TestIntegrationGapsMultiCarrier tests encoding data split across 2 carriers and
// decoding from both. The current multi-carrier encoding code has known bugs:
//   - embeddedCarrierFileNames is built with make([]string, N) + append, producing
//     N empty-string entries followed by N real entries
//   - embeddedCarrierFileNames[1:] therefore iterates over (N-1) empty + N real names
//   - The chunking logic in MultiCarrierEncode can double-append chunks
//
// This test documents the current behavior; failures indicate pre-existing bugs.
func TestIntegrationGapsMultiCarrier(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrier1Path := filepath.Join(tmpDir, "carrier1.png")
	carrier2Path := filepath.Join(tmpDir, "carrier2.png")
	createCarrierPNG(t, carrier1Path, 111)
	createCarrierPNG(t, carrier2Path, 222)

	originalData := make([]byte, 300)
	rng := rand.New(rand.NewSource(42))
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

	encErr := EncodeByFileNames(
		[]string{carrier1Path, carrier2Path},
		dataPath,
		1,
		"",
		encodeOutDir,
		cfg,
	)
	if encErr != nil {
		// Multi-carrier encoding has known bugs; record and stop.
		t.Fatalf("EncodeByFileNames with 2 carriers failed (known multi-carrier bug): %v", encErr)
	}

	// If encoding succeeded, try to decode from both embedded carriers.
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
		t.Fatalf("MultiCarrierDecodeByFileNames failed: %v", decErr)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "bin")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("multi-carrier roundtrip mismatch: original len=%d, decoded len=%d",
			len(originalData), len(decodedData))
		showSnippet(t, "original", originalData)
		showSnippet(t, "decoded", decodedData)
	}
}

// TestIntegrationGapsMaskEnabled tests roundtrip with helpers.UseMask = true.
func TestIntegrationGapsMaskEnabled(t *testing.T) {
	helpers.UseMask = true
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 200, 200, 54321)

	originalData := []byte("Mask-enabled steganography roundtrip test payload.")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      2,
		FileExtension: "txt",
		Password:      "masktest",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"masktest",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (mask) failed: %v", err)
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
		"masktest",
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (mask) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("mask roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestIntegrationGapsLargePayload tests encoding ~10,000 bytes of random data into
// a 200x200 carrier (capacity ~29,899 bytes at bit depth 2).
func TestIntegrationGapsLargePayload(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 200, 200, 77777)

	rng := rand.New(rand.NewSource(99))
	originalData := make([]byte, 10000)
	for i := range originalData {
		originalData[i] = byte(rng.Intn(256))
	}
	dataPath := filepath.Join(tmpDir, "bigdata.bin")
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
		[]string{carrierPath},
		dataPath,
		1,
		"",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (large payload) failed: %v", err)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")
	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err = MultiCarrierDecodeByFileNames(
		[]string{embeddedPath},
		"",
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("decode (large payload) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "bin")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("large payload mismatch: original len=%d, decoded len=%d",
			len(originalData), len(decodedData))
		showSnippet(t, "original", originalData)
		showSnippet(t, "decoded", decodedData)
	}
}

// TestIntegrationGapsRSCorruptionRecovery encodes data with Reed-Solomon, corrupts
// pixel data in the payload region, and verifies RS can correct the errors.
func TestIntegrationGapsRSCorruptionRecovery(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNGWithSize(t, carrierPath, 200, 200, 33333)

	originalData := []byte("Reed-Solomon corruption recovery integration test data payload.")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      2,
		RSEnabled:     true,
		RSLevel:       reed_solomon.High,
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
		t.Fatalf("EncodeByFileNames (RS) failed: %v", err)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")

	// Load the embedded PNG, corrupt some pixels in the payload region, save back.
	corruptedPath := filepath.Join(tmpDir, "corrupted.png")
	corruptPixels(t, embeddedPath, corruptedPath, 5)

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err = MultiCarrierDecodeByFileNames(
		[]string{corruptedPath},
		"",
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("decode from corrupted carrier failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("RS corruption recovery failed:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// TestIntegrationGapsWrongPassword encodes with one password and decodes with another.
// The decoded output should either error or differ from the original.
func TestIntegrationGapsWrongPassword(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNG(t, carrierPath, 44444)

	originalData := []byte("This data should not be recoverable with the wrong password.")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: true,
		FileExtension:  "txt",
		Password:       "correct",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		1,
		"correct",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames failed: %v", err)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")
	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Decode with the wrong password. This may error or produce garbage.
	decErr := MultiCarrierDecodeByFileNames(
		[]string{embeddedPath},
		"wrong",
		decodeOutDir,
	)
	if decErr != nil {
		// An error is acceptable — wrong password should not decode cleanly.
		t.Logf("decode with wrong password returned error (expected): %v", decErr)
		return
	}

	// If no error, verify the output does NOT match the original.
	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if bytes.Equal(decodedData, originalData) {
		t.Error("decoded data matches original despite using the wrong password — password protection is ineffective")
	} else {
		t.Logf("decoded data differs from original as expected (original len=%d, decoded len=%d)",
			len(originalData), len(decodedData))
	}
}

// TestIntegrationGapsFileExtensions verifies that various file extensions are preserved
// through the encode/decode roundtrip.
func TestIntegrationGapsFileExtensions(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	extensions := []struct {
		ext  string
		data []byte
	}{
		{"json", []byte(`{"key":"value"}`)},
		{"tar", []byte("fake tar content")},
		{"a", []byte("single char ext")},
		{"docx", []byte("fake docx content")},
		{"markdown", []byte("# heading")},
	}

	for _, tc := range extensions {
		tc := tc
		t.Run("ext_"+tc.ext, func(t *testing.T) {
			tmpDir := t.TempDir()

			carrierPath := filepath.Join(tmpDir, "carrier.png")
			createCarrierPNG(t, carrierPath, 12345)

			dataPath := filepath.Join(tmpDir, "testdata."+tc.ext)
			createDataFile(t, dataPath, tc.data)

			encodeOutDir := filepath.Join(tmpDir, "encoded")
			if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			cfg := pipeline.Config{
				BitDepth:      2,
				FileExtension: tc.ext,
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
				t.Fatalf("EncodeByFileNames failed for ext %q: %v", tc.ext, err)
			}

			embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")
			decodeOutDir := filepath.Join(tmpDir, "decoded")
			if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			err = MultiCarrierDecodeByFileNames(
				[]string{embeddedPath},
				"",
				decodeOutDir,
			)
			if err != nil {
				t.Fatalf("decode failed for ext %q: %v", tc.ext, err)
			}

			decodedPath := findDecodedFile(t, decodeOutDir, tc.ext)
			decodedData, err := os.ReadFile(decodedPath)
			if err != nil {
				t.Fatalf("read decoded: %v", err)
			}

			if !bytes.Equal(decodedData, tc.data) {
				t.Errorf("extension %q roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
					tc.ext, len(tc.data), tc.data, len(decodedData), decodedData)
			}

			ext := filepath.Ext(decodedPath)
			if ext != "."+tc.ext {
				t.Errorf("decoded file extension = %q, want %q", ext, "."+tc.ext)
			}
		})
	}
}

// TestIntegrationGapsJPEGCarrier tests that a JPEG carrier can be used for encoding.
// The output is always PNG regardless of the input carrier format.
func TestIntegrationGapsJPEGCarrier(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	tmpDir := t.TempDir()

	// Create a JPEG carrier image.
	jpegPath := filepath.Join(tmpDir, "carrier.jpg")
	createCarrierJPEG(t, jpegPath, 200, 200, 55555)

	originalData := []byte("JPEG carrier input test — output should be PNG.")
	dataPath := filepath.Join(tmpDir, "data.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:      2,
		FileExtension: "txt",
	}

	err := EncodeByFileNames(
		[]string{jpegPath},
		dataPath,
		1,
		"",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (JPEG carrier) failed: %v", err)
	}

	// The output keeps the original carrier extension in the filename,
	// but the file content is actually PNG.
	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.jpg")
	if _, err := os.Stat(embeddedPath); os.IsNotExist(err) {
		t.Fatalf("embedded carrier not found: %s", embeddedPath)
	}

	// Verify the file is actually a PNG by checking magic bytes.
	f, err := os.Open(embeddedPath)
	if err != nil {
		t.Fatalf("open embedded: %v", err)
	}
	magic := make([]byte, 4)
	f.Read(magic)
	f.Close()
	// PNG magic: 0x89 0x50 0x4E 0x47
	if magic[0] != 0x89 || magic[1] != 0x50 || magic[2] != 0x4E || magic[3] != 0x47 {
		t.Errorf("embedded file is not PNG format; magic bytes: %x", magic)
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err = MultiCarrierDecodeByFileNames(
		[]string{embeddedPath},
		"",
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("decode (JPEG carrier) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("JPEG carrier roundtrip mismatch:\n  original (%d bytes): %q\n  decoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

// --- Helper functions ---

// createCarrierJPEG creates a JPEG file with random pixel data.
func createCarrierJPEG(t *testing.T, path string, width, height int, seed int64) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	rng := rand.New(rand.NewSource(seed))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(rng.Intn(256)),
				G: uint8(rng.Intn(256)),
				B: uint8(rng.Intn(256)),
				A: 255,
			})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create JPEG file: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatalf("encode JPEG: %v", err)
	}
}

// corruptPixels loads a PNG, flips bits in some pixels in the payload region
// (y >= 34), and saves it back as PNG.
func corruptPixels(t *testing.T, srcPath, dstPath string, numPixels int) {
	t.Helper()

	f, err := os.Open(srcPath)
	if err != nil {
		t.Fatalf("open for corruption: %v", err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatalf("decode for corruption: %v", err)
	}

	rgba, ok := img.(*image.RGBA)
	if !ok {
		// Convert to RGBA
		bounds := img.Bounds()
		rgba = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}

	bounds := rgba.Bounds()
	rng := rand.New(rand.NewSource(12345))

	for i := 0; i < numPixels; i++ {
		// Pick a pixel in the payload region (y >= 34)
		x := rng.Intn(bounds.Dx())
		y := 34 + rng.Intn(bounds.Dy()-34)
		c := rgba.RGBAAt(x, y)
		// Flip the lowest 2 bits of each channel
		c.R ^= 0x03
		c.G ^= 0x03
		c.B ^= 0x03
		rgba.SetRGBA(x, y, c)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		t.Fatalf("create corrupted file: %v", err)
	}
	defer out.Close()
	if err := png.Encode(out, rgba); err != nil {
		t.Fatalf("encode corrupted PNG: %v", err)
	}
}

// showSnippet logs the first few bytes of a slice for debugging.
func showSnippet(t *testing.T, label string, data []byte) {
	t.Helper()
	n := len(data)
	if n > 32 {
		n = 32
	}
	t.Logf("  %s (first %d of %d bytes): %x", label, n, len(data), data[:n])
}

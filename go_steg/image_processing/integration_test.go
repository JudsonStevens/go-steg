package image_processing

import (
	"bytes"
	"go-steg/cli/helpers"
	"go-steg/go_steg/pipeline"
	"go-steg/go_steg/reed_solomon"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// createCarrierPNG creates a 200x200 PNG file filled with pseudo-random pixel
// data and writes it to the given path. The deterministic seed ensures
// reproducibility across test runs.
func createCarrierPNG(t *testing.T, path string, seed int64) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	rng := rand.New(rand.NewSource(seed))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
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
		t.Fatalf("failed to create carrier file: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode carrier PNG: %v", err)
	}
}

// createDataFile writes the given content to the specified path.
func createDataFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write data file: %v", err)
	}
}

// findDecodedFile finds the decoded output file in the given directory using a
// glob pattern.
func findDecodedFile(t *testing.T, dir, ext string) string {
	t.Helper()
	pattern := filepath.Join(dir, "decoded_file-*."+ext)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no decoded file found matching %s", pattern)
	}
	return matches[0]
}

func TestIntegrationRoundtrip(t *testing.T) {
	// Disable mask for deterministic, simpler tests.
	helpers.UseMask = false

	testCases := []struct {
		name      string
		cfg       pipeline.Config
		dataExt   string
		makeData  func() []byte
	}{
		{
			name: "default_bitdepth2_txt",
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: false,
				RSEnabled:      false,
				FileExtension:  "txt",
			},
			dataExt:  "txt",
			makeData: func() []byte { return []byte("Hello, steganography! This is a basic roundtrip test.") },
		},
		{
			name: "huffman_only_txt",
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: true,
				RSEnabled:      false,
				FileExtension:  "txt",
				Password:       "huffman-test",
			},
			dataExt:  "txt",
			makeData: func() []byte { return []byte("Huffman encoding test payload with repeated characters aaaaabbbbcccc.") },
		},
		{
			name: "rs_only_standard_txt",
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: false,
				RSEnabled:      true,
				RSLevel:        reed_solomon.Standard,
				FileExtension:  "txt",
			},
			dataExt:  "txt",
			makeData: func() []byte { return []byte("Reed-Solomon standard level test data.") },
		},
		{
			name: "huffman_rs_bitdepth1",
			cfg: pipeline.Config{
				BitDepth:       1,
				HuffmanEnabled: true,
				RSEnabled:      true,
				RSLevel:        reed_solomon.Standard,
				FileExtension:  "txt",
				Password:       "depth1",
			},
			dataExt:  "txt",
			makeData: func() []byte { return []byte("Bit depth 1 test.") },
		},
		{
			name: "huffman_rs_bitdepth3",
			cfg: pipeline.Config{
				BitDepth:       3,
				HuffmanEnabled: true,
				RSEnabled:      true,
				RSLevel:        reed_solomon.Standard,
				FileExtension:  "txt",
				Password:       "depth3",
			},
			dataExt:  "txt",
			makeData: func() []byte { return []byte("Bit depth 3 with Huffman and Reed-Solomon.") },
		},
		{
			name: "huffman_rs_bitdepth4",
			cfg: pipeline.Config{
				BitDepth:       4,
				HuffmanEnabled: true,
				RSEnabled:      true,
				RSLevel:        reed_solomon.Standard,
				FileExtension:  "txt",
				Password:       "depth4",
			},
			dataExt:  "txt",
			makeData: func() []byte { return []byte("Bit depth 4 with Huffman and Reed-Solomon encoding applied.") },
		},
		{
			name: "binary_data_bin",
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: false,
				RSEnabled:      false,
				FileExtension:  "bin",
			},
			dataExt: "bin",
			makeData: func() []byte {
				rng := rand.New(rand.NewSource(42))
				data := make([]byte, 256)
				for i := range data {
					data[i] = byte(rng.Intn(256))
				}
				return data
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create carrier image and data file.
			carrierPath := filepath.Join(tmpDir, "carrier.png")
			createCarrierPNG(t, carrierPath, 12345)

			originalData := tc.makeData()
			dataPath := filepath.Join(tmpDir, "testdata."+tc.dataExt)
			createDataFile(t, dataPath, originalData)

			// Encode.
			encodeOutDir := filepath.Join(tmpDir, "encoded")
			if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
				t.Fatalf("failed to create encode output dir: %v", err)
			}

			err := EncodeByFileNames(
				[]string{carrierPath},
				dataPath,
				1,
				tc.cfg.Password,
				encodeOutDir,
				tc.cfg,
			)
			if err != nil {
				t.Fatalf("EncodeByFileNames failed: %v", err)
			}

			// Find the embedded carrier file.
			embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")
			if _, err := os.Stat(embeddedPath); os.IsNotExist(err) {
				t.Fatalf("embedded carrier not found at %s", embeddedPath)
			}

			// Decode.
			decodeOutDir := filepath.Join(tmpDir, "decoded")
			if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
				t.Fatalf("failed to create decode output dir: %v", err)
			}

			err = MultiCarrierDecodeByFileNames(
				[]string{embeddedPath},
				tc.cfg.Password,
				decodeOutDir,
			)
			if err != nil {
				t.Fatalf("MultiCarrierDecodeByFileNames failed: %v", err)
			}

			// Find and verify the decoded file.
			decodedPath := findDecodedFile(t, decodeOutDir, tc.dataExt)
			decodedData, err := os.ReadFile(decodedPath)
			if err != nil {
				t.Fatalf("failed to read decoded file: %v", err)
			}

			if !bytes.Equal(decodedData, originalData) {
				t.Errorf("decoded data does not match original.\nOriginal (%d bytes): %q\nDecoded  (%d bytes): %q",
					len(originalData), originalData,
					len(decodedData), decodedData)
			}

			// Verify file extension in output name.
			ext := filepath.Ext(decodedPath)
			expectedExt := "." + tc.dataExt
			if ext != expectedExt {
				t.Errorf("decoded file extension = %q, want %q", ext, expectedExt)
			}
		})
	}
}

func TestIntegrationLegacyCompatibility(t *testing.T) {
	// Verify that the default config (bit depth 2, no features) encodes and
	// decodes correctly, preserving backward compatibility.
	helpers.UseMask = false

	tmpDir := t.TempDir()

	carrierPath := filepath.Join(tmpDir, "carrier.png")
	createCarrierPNG(t, carrierPath, 99999)

	originalData := []byte("Legacy compatibility test: this should decode correctly with the default configuration.")
	dataPath := filepath.Join(tmpDir, "legacy.txt")
	createDataFile(t, dataPath, originalData)

	encodeOutDir := filepath.Join(tmpDir, "encoded")
	if err := os.MkdirAll(encodeOutDir, 0755); err != nil {
		t.Fatalf("failed to create encode output dir: %v", err)
	}

	cfg := pipeline.Config{
		BitDepth:       2,
		HuffmanEnabled: false,
		RSEnabled:      false,
		FileExtension:  "txt",
	}

	err := EncodeByFileNames(
		[]string{carrierPath},
		dataPath,
		42,
		"",
		encodeOutDir,
		cfg,
	)
	if err != nil {
		t.Fatalf("EncodeByFileNames (legacy) failed: %v", err)
	}

	embeddedPath := filepath.Join(encodeOutDir, "carrier-0-embedded.png")
	if _, err := os.Stat(embeddedPath); os.IsNotExist(err) {
		t.Fatalf("embedded carrier not found at %s", embeddedPath)
	}

	decodeOutDir := filepath.Join(tmpDir, "decoded")
	if err := os.MkdirAll(decodeOutDir, 0755); err != nil {
		t.Fatalf("failed to create decode output dir: %v", err)
	}

	err = MultiCarrierDecodeByFileNames(
		[]string{embeddedPath},
		"",
		decodeOutDir,
	)
	if err != nil {
		t.Fatalf("MultiCarrierDecodeByFileNames (legacy) failed: %v", err)
	}

	decodedPath := findDecodedFile(t, decodeOutDir, "txt")
	decodedData, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("failed to read decoded file: %v", err)
	}

	if !bytes.Equal(decodedData, originalData) {
		t.Errorf("legacy decoded data does not match original.\nOriginal (%d bytes): %q\nDecoded  (%d bytes): %q",
			len(originalData), originalData,
			len(decodedData), decodedData)
	}
}

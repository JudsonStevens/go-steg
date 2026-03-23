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

func TestDataPatternRoundtrip(t *testing.T) {
	helpers.UseMask = false
	defer func() { helpers.UseMask = false }()

	testCases := []struct {
		name     string
		makeData func() []byte
		cfg      pipeline.Config
	}{
		{
			name: "all_zero_bytes",
			makeData: func() []byte {
				return make([]byte, 500)
			},
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: true,
				RSEnabled:      true,
				RSLevel:        reed_solomon.High,
				FileExtension:  "bin",
			},
		},
		{
			name: "all_0xFF_bytes",
			makeData: func() []byte {
				data := make([]byte, 500)
				for i := range data {
					data[i] = 0xFF
				}
				return data
			},
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: true,
				RSEnabled:      true,
				RSLevel:        reed_solomon.High,
				FileExtension:  "bin",
			},
		},
		{
			name: "alternating_0xAA_0x55",
			makeData: func() []byte {
				data := make([]byte, 500)
				for i := range data {
					if i%2 == 0 {
						data[i] = 0xAA
					} else {
						data[i] = 0x55
					}
				}
				return data
			},
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: true,
				RSEnabled:      true,
				RSLevel:        reed_solomon.High,
				FileExtension:  "bin",
			},
		},
		{
			name: "single_repeated_byte_0x42",
			makeData: func() []byte {
				data := make([]byte, 500)
				for i := range data {
					data[i] = 0x42
				}
				return data
			},
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: true,
				RSEnabled:      true,
				RSLevel:        reed_solomon.High,
				FileExtension:  "bin",
			},
		},
		{
			name: "empty_password_huffman",
			makeData: func() []byte {
				return []byte("Huffman with empty password test data for roundtrip verification.")
			},
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: true,
				RSEnabled:      false,
				FileExtension:  "bin",
				Password:       "",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			carrierPath := filepath.Join(tmpDir, "carrier.png")
			createCarrierPNG(t, carrierPath, 12345)

			originalData := tc.makeData()
			dataPath := filepath.Join(tmpDir, "testdata."+tc.cfg.FileExtension)
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
			decodedPath := findDecodedFile(t, decodeOutDir, tc.cfg.FileExtension)
			decodedData, err := os.ReadFile(decodedPath)
			if err != nil {
				t.Fatalf("failed to read decoded file: %v", err)
			}

			if !bytes.Equal(decodedData, originalData) {
				origSnip := len(originalData)
				if origSnip > 64 {
					origSnip = 64
				}
				decSnip := len(decodedData)
				if decSnip > 64 {
					decSnip = 64
				}
				t.Errorf("decoded data does not match original.\nOriginal (%d bytes): %x\nDecoded  (%d bytes): %x",
					len(originalData), originalData[:origSnip],
					len(decodedData), decodedData[:decSnip])
			}
		})
	}
}


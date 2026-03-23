package image_processing

import (
	"bytes"
	"go-steg/cli/helpers"
	"go-steg/go_steg/bit_manipulation"
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

// TestPixelLSBVerification encodes known data, opens the embedded PNG,
// reads pixels in the payload region (y >= 34), extracts the last N bits
// from R/G/B channels, and verifies they match the expected pipeline-encoded
// data chunks.
func TestPixelLSBVerification(t *testing.T) {
	helpers.UseMask = false

	testCases := []struct {
		name string
		cfg  pipeline.Config
		data []byte
	}{
		{
			name: "bitdepth2_no_pipeline",
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: false,
				RSEnabled:      false,
				FileExtension:  "txt",
			},
			data: []byte("Hello pixel verification!"),
		},
		{
			name: "bitdepth1_no_pipeline",
			cfg: pipeline.Config{
				BitDepth:       1,
				HuffmanEnabled: false,
				RSEnabled:      false,
				FileExtension:  "bin",
			},
			data: []byte("bit depth one test"),
		},
		{
			name: "bitdepth4_no_pipeline",
			cfg: pipeline.Config{
				BitDepth:       4,
				HuffmanEnabled: false,
				RSEnabled:      false,
				FileExtension:  "dat",
			},
			data: []byte("four bit depth test data"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			carrierPath := filepath.Join(tmpDir, "carrier.png")
			outputDir := tmpDir
			dataPath := filepath.Join(tmpDir, "data."+tc.cfg.FileExtension)

			createCarrierPNG(t, carrierPath, 99)
			createDataFile(t, dataPath, tc.data)

			err := EncodeByFileNames(
				[]string{carrierPath},
				dataPath,
				42,
				"",
				outputDir,
				tc.cfg,
			)
			if err != nil {
				t.Fatalf("EncodeByFileNames failed: %v", err)
			}

			// Find the embedded file
			embeddedPath := filepath.Join(outputDir, "carrier-0-embedded.png")
			f, err := os.Open(embeddedPath)
			if err != nil {
				t.Fatalf("failed to open embedded file: %v", err)
			}
			defer f.Close()

			img, err := png.Decode(f)
			if err != nil {
				t.Fatalf("failed to decode embedded PNG: %v", err)
			}
			rgbaImg := img.(*image.RGBA)
			if rgbaImg == nil {
				// Convert if needed
				bounds := img.Bounds()
				rgbaImg = image.NewRGBA(bounds)
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						rgbaImg.Set(x, y, img.At(x, y))
					}
				}
			}

			// Compute expected pipeline output
			pipelineOutput, err := pipeline.Encode(tc.data, tc.cfg)
			if err != nil {
				t.Fatalf("pipeline.Encode failed: %v", err)
			}

			// Split pipeline output into chunks
			bitDepth := tc.cfg.BitDepth
			if bitDepth < 1 || bitDepth > 4 {
				bitDepth = 2
			}
			var expectedChunks []byte
			for _, b := range pipelineOutput {
				chunks := bit_manipulation.SplitByte(b, bitDepth)
				expectedChunks = append(expectedChunks, chunks...)
			}

			// Read actual LSBs from the image in encoding traversal order
			bounds := rgbaImg.Bounds()
			var actualChunks []byte
			for x := 0; x < bounds.Dx() && len(actualChunks) < len(expectedChunks); x++ {
				for y := totalReservedPixels; y < bounds.Dy() && len(actualChunks) < len(expectedChunks); y++ {
					c := rgbaImg.RGBAAt(x, y)
					// R, G, B in order
					actualChunks = append(actualChunks, bit_manipulation.GetLastNBits(c.R, bitDepth))
					if len(actualChunks) < len(expectedChunks) {
						actualChunks = append(actualChunks, bit_manipulation.GetLastNBits(c.G, bitDepth))
					}
					if len(actualChunks) < len(expectedChunks) {
						actualChunks = append(actualChunks, bit_manipulation.GetLastNBits(c.B, bitDepth))
					}
				}
			}

			if len(actualChunks) < len(expectedChunks) {
				t.Fatalf("not enough pixels to read all expected chunks: got %d, want %d", len(actualChunks), len(expectedChunks))
			}

			for i := 0; i < len(expectedChunks); i++ {
				if actualChunks[i] != expectedChunks[i] {
					t.Errorf("chunk %d mismatch: got %d, want %d", i, actualChunks[i], expectedChunks[i])
				}
			}
		})
	}
}

// TestHeaderFieldsVerification encodes data, opens the embedded PNG, calls
// readHeader() on the image, and verifies all header fields match what was
// encoded.
func TestHeaderFieldsVerification(t *testing.T) {
	helpers.UseMask = false

	testCases := []struct {
		name          string
		cfg           pipeline.Config
		photoID       uint64
		data          []byte
	}{
		{
			name: "basic_txt_bitdepth2",
			cfg: pipeline.Config{
				BitDepth:       2,
				HuffmanEnabled: false,
				RSEnabled:      false,
				FileExtension:  "txt",
			},
			photoID: 12345,
			data:    []byte("header test data"),
		},
		{
			name: "huffman_rs_high_png",
			cfg: pipeline.Config{
				BitDepth:       3,
				HuffmanEnabled: true,
				RSEnabled:      true,
				RSLevel:        reed_solomon.High,
				FileExtension:  "png",
				Password:       "testpass",
			},
			photoID: 9876543210,
			data:    []byte("header test with huffman and RS encoding features enabled"),
		},
		{
			name: "rs_standard_bitdepth1",
			cfg: pipeline.Config{
				BitDepth:       1,
				HuffmanEnabled: false,
				RSEnabled:      true,
				RSLevel:        reed_solomon.Standard,
				FileExtension:  "bin",
			},
			photoID: 255,
			data:    []byte("RS standard test"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			carrierPath := filepath.Join(tmpDir, "carrier.png")
			outputDir := tmpDir
			dataPath := filepath.Join(tmpDir, "data."+tc.cfg.FileExtension)

			createCarrierPNG(t, carrierPath, 42)
			createDataFile(t, dataPath, tc.data)

			err := EncodeByFileNames(
				[]string{carrierPath},
				dataPath,
				tc.photoID,
				"",
				outputDir,
				tc.cfg,
			)
			if err != nil {
				t.Fatalf("EncodeByFileNames failed: %v", err)
			}

			embeddedPath := filepath.Join(outputDir, "carrier-0-embedded.png")
			f, err := os.Open(embeddedPath)
			if err != nil {
				t.Fatalf("failed to open embedded file: %v", err)
			}
			defer f.Close()

			img, err := png.Decode(f)
			if err != nil {
				t.Fatalf("failed to decode embedded PNG: %v", err)
			}
			bounds := img.Bounds()
			rgbaImg := image.NewRGBA(bounds)
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					rgbaImg.Set(x, y, img.At(x, y))
				}
			}

			header := readHeader(rgbaImg)

			// Compute expected values
			pipelineOutput, err := pipeline.Encode(tc.data, tc.cfg)
			if err != nil {
				t.Fatalf("pipeline.Encode failed: %v", err)
			}
			expectedChecksum := computeChecksum(pipelineOutput)
			expectedByteCountMod := uint16(len(pipelineOutput) % 4096)

			bitDepth := tc.cfg.BitDepth
			if bitDepth < 1 || bitDepth > 4 {
				bitDepth = 2
			}

			// Compute expected data count: number of chunks written
			var totalChunks uint32
			for range pipelineOutput {
				chunks := bit_manipulation.SplitByte(0, bitDepth) // just to get count
				totalChunks += uint32(len(chunks))
			}

			// Verify header fields
			if header.PhotoID != tc.photoID {
				t.Errorf("PhotoID: got %d, want %d", header.PhotoID, tc.photoID)
			}
			if header.PhotoNumber != 0 {
				t.Errorf("PhotoNumber: got %d, want 0", header.PhotoNumber)
			}
			if header.DataCount != totalChunks {
				t.Errorf("DataCount: got %d, want %d", header.DataCount, totalChunks)
			}
			if !header.IsNewFormat {
				t.Error("IsNewFormat: got false, want true")
			}
			if header.FileExtension != tc.cfg.FileExtension {
				t.Errorf("FileExtension: got %q, want %q", header.FileExtension, tc.cfg.FileExtension)
			}
			if header.BitDepth != bitDepth {
				t.Errorf("BitDepth: got %d, want %d", header.BitDepth, bitDepth)
			}
			if header.HuffmanEnabled != tc.cfg.HuffmanEnabled {
				t.Errorf("HuffmanEnabled: got %v, want %v", header.HuffmanEnabled, tc.cfg.HuffmanEnabled)
			}
			if header.RSEnabled != tc.cfg.RSEnabled {
				t.Errorf("RSEnabled: got %v, want %v", header.RSEnabled, tc.cfg.RSEnabled)
			}
			if header.RSLevel != tc.cfg.RSLevel {
				t.Errorf("RSLevel: got %v, want %v", header.RSLevel, tc.cfg.RSLevel)
			}
			if header.Checksum != expectedChecksum {
				t.Errorf("Checksum: got %d, want %d", header.Checksum, expectedChecksum)
			}
			if header.ByteCountMod != expectedByteCountMod {
				t.Errorf("ByteCountMod: got %d, want %d", header.ByteCountMod, expectedByteCountMod)
			}
		})
	}
}

// TestUpperBitsPreserved creates a carrier with known pixel values, encodes
// data into it, and verifies that the upper bits (above bit depth) of each
// pixel remain unchanged.
func TestUpperBitsPreserved(t *testing.T) {
	helpers.UseMask = false

	testCases := []struct {
		name     string
		bitDepth int
	}{
		{"bitdepth1", 1},
		{"bitdepth2", 2},
		{"bitdepth3", 3},
		{"bitdepth4", 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			carrierPath := filepath.Join(tmpDir, "carrier.png")
			outputDir := tmpDir
			dataPath := filepath.Join(tmpDir, "data.txt")

			width, height := 100, 100
			seed := int64(7777)

			// Create carrier with known pixels
			rng := rand.New(rand.NewSource(seed))
			originalImg := image.NewRGBA(image.Rect(0, 0, width, height))
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					originalImg.SetRGBA(x, y, color.RGBA{
						R: uint8(rng.Intn(256)),
						G: uint8(rng.Intn(256)),
						B: uint8(rng.Intn(256)),
						A: 255,
					})
				}
			}

			// Save the carrier
			cf, err := os.Create(carrierPath)
			if err != nil {
				t.Fatalf("failed to create carrier: %v", err)
			}
			if err := png.Encode(cf, originalImg); err != nil {
				cf.Close()
				t.Fatalf("failed to encode carrier: %v", err)
			}
			cf.Close()

			// Store original pixel values (reload to get exact PNG round-trip values)
			cf2, err := os.Open(carrierPath)
			if err != nil {
				t.Fatalf("failed to open carrier: %v", err)
			}
			carrierDecoded, err := png.Decode(cf2)
			cf2.Close()
			if err != nil {
				t.Fatalf("failed to decode carrier: %v", err)
			}
			carrierBounds := carrierDecoded.Bounds()
			originalRGBA := image.NewRGBA(carrierBounds)
			for y := carrierBounds.Min.Y; y < carrierBounds.Max.Y; y++ {
				for x := carrierBounds.Min.X; x < carrierBounds.Max.X; x++ {
					originalRGBA.Set(x, y, carrierDecoded.At(x, y))
				}
			}

			// Create data and encode
			testData := bytes.Repeat([]byte("ABCDEFGH"), 20)
			createDataFile(t, dataPath, testData)

			cfg := pipeline.Config{
				BitDepth:       tc.bitDepth,
				HuffmanEnabled: false,
				RSEnabled:      false,
				FileExtension:  "txt",
			}

			err = EncodeByFileNames(
				[]string{carrierPath},
				dataPath,
				123,
				"",
				outputDir,
				cfg,
			)
			if err != nil {
				t.Fatalf("EncodeByFileNames failed: %v", err)
			}

			// Open the embedded image
			embeddedPath := filepath.Join(outputDir, "carrier-0-embedded.png")
			ef, err := os.Open(embeddedPath)
			if err != nil {
				t.Fatalf("failed to open embedded: %v", err)
			}
			defer ef.Close()

			embeddedImg, err := png.Decode(ef)
			if err != nil {
				t.Fatalf("failed to decode embedded: %v", err)
			}
			embBounds := embeddedImg.Bounds()
			embeddedRGBA := image.NewRGBA(embBounds)
			for y := embBounds.Min.Y; y < embBounds.Max.Y; y++ {
				for x := embBounds.Min.X; x < embBounds.Max.X; x++ {
					embeddedRGBA.Set(x, y, embeddedImg.At(x, y))
				}
			}

			// Verify upper bits are preserved for payload region
			upperMask := byte(0xFF) << tc.bitDepth
			errCount := 0
			for x := 0; x < embBounds.Dx(); x++ {
				for y := totalReservedPixels; y < embBounds.Dy(); y++ {
					origC := originalRGBA.RGBAAt(x, y)
					embC := embeddedRGBA.RGBAAt(x, y)

					if (origC.R & upperMask) != (embC.R & upperMask) {
						if errCount < 10 {
							t.Errorf("pixel (%d,%d) R upper bits changed: orig=%08b, emb=%08b", x, y, origC.R, embC.R)
						}
						errCount++
					}
					if (origC.G & upperMask) != (embC.G & upperMask) {
						if errCount < 10 {
							t.Errorf("pixel (%d,%d) G upper bits changed: orig=%08b, emb=%08b", x, y, origC.G, embC.G)
						}
						errCount++
					}
					if (origC.B & upperMask) != (embC.B & upperMask) {
						if errCount < 10 {
							t.Errorf("pixel (%d,%d) B upper bits changed: orig=%08b, emb=%08b", x, y, origC.B, embC.B)
						}
						errCount++
					}
					// Alpha should be completely unchanged
					if origC.A != embC.A {
						if errCount < 10 {
							t.Errorf("pixel (%d,%d) Alpha changed: orig=%d, emb=%d", x, y, origC.A, embC.A)
						}
						errCount++
					}
				}
			}
			if errCount > 10 {
				t.Errorf("... and %d more upper-bit errors", errCount-10)
			}
		})
	}
}

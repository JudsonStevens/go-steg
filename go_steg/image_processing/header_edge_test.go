package image_processing

import (
	"go-steg/go_steg/reed_solomon"
	"image"
	"testing"
)

func TestHeaderAllBitDepths(t *testing.T) {
	for depth := 1; depth <= 4; depth++ {
		t.Run("", func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 100, 100))
			info := HeaderInfo{
				IsNewFormat: true,
				BitDepth:    depth,
			}
			writeHeader(img, info)
			got := readHeader(img)
			if !got.IsNewFormat {
				t.Fatal("expected new format")
			}
			if got.BitDepth != depth {
				t.Errorf("BitDepth: got %d, want %d", got.BitDepth, depth)
			}
		})
	}
}

func TestHeaderAllFlagCombinations(t *testing.T) {
	tests := []struct {
		name    string
		huffman bool
		rs      bool
		rsLevel reed_solomon.RedundancyLevel
	}{
		{"none", false, false, reed_solomon.Standard},
		{"huffman only", true, false, reed_solomon.Standard},
		{"RS standard only", false, true, reed_solomon.Standard},
		{"RS high only", false, true, reed_solomon.High},
		{"huffman + RS standard", true, true, reed_solomon.Standard},
		{"huffman + RS high", true, true, reed_solomon.High},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 100, 100))
			info := HeaderInfo{
				IsNewFormat:    true,
				BitDepth:       2,
				HuffmanEnabled: tt.huffman,
				RSEnabled:      tt.rs,
				RSLevel:        tt.rsLevel,
			}
			writeHeader(img, info)
			got := readHeader(img)
			if got.HuffmanEnabled != tt.huffman {
				t.Errorf("HuffmanEnabled: got %v, want %v", got.HuffmanEnabled, tt.huffman)
			}
			if got.RSEnabled != tt.rs {
				t.Errorf("RSEnabled: got %v, want %v", got.RSEnabled, tt.rs)
			}
			if got.RSLevel != tt.rsLevel {
				t.Errorf("RSLevel: got %v, want %v", got.RSLevel, tt.rsLevel)
			}
		})
	}
}

func TestHeaderU12BoundaryValues(t *testing.T) {
	tests := []struct {
		name     string
		checksum uint16
		byteMod  uint16
	}{
		{"zeros", 0, 0},
		{"max 12-bit", 0xFFF, 0xFFF},
		{"mid value", 0x800, 0x800},
		{"one", 1, 1},
		{"alternating bits", 0xAAA, 0x555},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 100, 100))
			info := HeaderInfo{
				IsNewFormat:  true,
				BitDepth:     2,
				Checksum:     tt.checksum,
				ByteCountMod: tt.byteMod,
			}
			writeHeader(img, info)
			got := readHeader(img)
			if got.Checksum != tt.checksum {
				t.Errorf("Checksum: got %d, want %d", got.Checksum, tt.checksum)
			}
			if got.ByteCountMod != tt.byteMod {
				t.Errorf("ByteCountMod: got %d, want %d", got.ByteCountMod, tt.byteMod)
			}
		})
	}
}

func TestHeaderMaxPhotoID(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Max value that can be encoded: 6 bytes = 48 bits (only 24 quarters used)
	// The QuartersOfBytes64 only stores 6 bytes, so max is 2^48-1
	maxID := uint64(1<<48 - 1)
	info := HeaderInfo{
		IsNewFormat: true,
		BitDepth:    2,
		PhotoID:     maxID,
	}
	writeHeader(img, info)
	got := readHeader(img)
	if got.PhotoID != maxID {
		t.Errorf("PhotoID: got %d, want %d", got.PhotoID, maxID)
	}
}

func TestHeaderMaxPhotoNumber(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Photo number uses 6 bits (3 channels * 2 bits), max = 63
	info := HeaderInfo{
		IsNewFormat: true,
		BitDepth:    2,
		PhotoNumber: 63,
	}
	writeHeader(img, info)
	got := readHeader(img)
	if got.PhotoNumber != 63 {
		t.Errorf("PhotoNumber: got %d, want 63", got.PhotoNumber)
	}
}

func TestHeaderMaxDataCount(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// DataCount uses 12 quarters = 3 bytes (24 bits), encoded as uint32 with LE
	// But only 3 bytes are reconstructed from 12 quarters, so max is 2^24-1
	maxDC := uint32(1<<24 - 1)
	info := HeaderInfo{
		IsNewFormat: true,
		BitDepth:    2,
		DataCount:   maxDC,
	}
	writeHeader(img, info)
	got := readHeader(img)
	if got.DataCount != maxDC {
		t.Errorf("DataCount: got %d, want %d", got.DataCount, maxDC)
	}
}

func TestHeaderExtensionMaxLength(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	info := HeaderInfo{
		IsNewFormat:   true,
		BitDepth:      2,
		FileExtension: "markdown", // exactly 8 chars
	}
	writeHeader(img, info)
	got := readHeader(img)
	if got.FileExtension != "markdown" {
		t.Errorf("FileExtension: got %q, want %q", got.FileExtension, "markdown")
	}
}

func TestHeaderExtensionSingleChar(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	info := HeaderInfo{
		IsNewFormat:   true,
		BitDepth:      2,
		FileExtension: "c",
	}
	writeHeader(img, info)
	got := readHeader(img)
	if got.FileExtension != "c" {
		t.Errorf("FileExtension: got %q, want %q", got.FileExtension, "c")
	}
}

func TestHeaderExtensionVariousFormats(t *testing.T) {
	extensions := []string{"txt", "pdf", "png", "jpg", "bin", "dat", "go", "rs"}
	for _, ext := range extensions {
		t.Run(ext, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 100, 100))
			info := HeaderInfo{
				IsNewFormat:   true,
				BitDepth:      2,
				FileExtension: ext,
			}
			writeHeader(img, info)
			got := readHeader(img)
			if got.FileExtension != ext {
				t.Errorf("FileExtension: got %q, want %q", got.FileExtension, ext)
			}
		})
	}
}

func TestHeaderVersionMarkerDetection(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	info := HeaderInfo{
		IsNewFormat: true,
		BitDepth:    2,
	}
	writeHeader(img, info)

	// Verify new format is detected
	got := readHeader(img)
	if !got.IsNewFormat {
		t.Error("expected new format to be detected")
	}

	// Corrupt the version marker and verify legacy detection
	img2 := image.NewRGBA(image.Rect(0, 0, 100, 100))
	got2 := readHeader(img2)
	if got2.IsNewFormat {
		t.Error("expected legacy format for blank image")
	}
}

func TestWriteU12ReadU12Roundtrip(t *testing.T) {
	// Test all 12-bit values at boundaries
	values := []uint16{0, 1, 2, 3, 4, 0x3FF, 0x400, 0x7FF, 0x800, 0xFFF}
	for _, val := range values {
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		writeU12(img, 0, val)
		got := readU12(img, 0)
		if got != val {
			t.Errorf("writeU12/readU12 roundtrip failed for %d: got %d", val, got)
		}
	}
}

func TestHeaderMinimalImage(t *testing.T) {
	// Minimum image that can hold a header: 1 pixel wide, 34 pixels tall
	img := image.NewRGBA(image.Rect(0, 0, 1, 34))
	info := HeaderInfo{
		IsNewFormat:   true,
		BitDepth:      1,
		FileExtension: "txt",
		Checksum:      42,
		ByteCountMod:  100,
	}
	writeHeader(img, info)
	got := readHeader(img)
	if !got.IsNewFormat {
		t.Error("expected new format on minimal image")
	}
	if got.BitDepth != 1 {
		t.Errorf("BitDepth: got %d, want 1", got.BitDepth)
	}
	if got.FileExtension != "txt" {
		t.Errorf("FileExtension: got %q, want 'txt'", got.FileExtension)
	}
}

func TestHeaderFullRoundtripAllFields(t *testing.T) {
	// Test with all fields at non-zero, non-max values
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	info := HeaderInfo{
		PhotoID:        999999,
		PhotoNumber:    42,
		DataCount:      100000,
		IsNewFormat:    true,
		FileExtension:  "json",
		BitDepth:       4,
		HuffmanEnabled: true,
		RSEnabled:      true,
		RSLevel:        reed_solomon.High,
		Checksum:       0x123,
		ByteCountMod:   2048,
	}
	writeHeader(img, info)
	got := readHeader(img)

	if got.PhotoID != info.PhotoID {
		t.Errorf("PhotoID: got %d, want %d", got.PhotoID, info.PhotoID)
	}
	if got.PhotoNumber != info.PhotoNumber {
		t.Errorf("PhotoNumber: got %d, want %d", got.PhotoNumber, info.PhotoNumber)
	}
	if got.DataCount != info.DataCount {
		t.Errorf("DataCount: got %d, want %d", got.DataCount, info.DataCount)
	}
	if got.FileExtension != info.FileExtension {
		t.Errorf("FileExtension: got %q, want %q", got.FileExtension, info.FileExtension)
	}
	if got.BitDepth != info.BitDepth {
		t.Errorf("BitDepth: got %d, want %d", got.BitDepth, info.BitDepth)
	}
	if got.HuffmanEnabled != info.HuffmanEnabled {
		t.Errorf("HuffmanEnabled: got %v, want %v", got.HuffmanEnabled, info.HuffmanEnabled)
	}
	if got.RSEnabled != info.RSEnabled {
		t.Errorf("RSEnabled: got %v, want %v", got.RSEnabled, info.RSEnabled)
	}
	if got.RSLevel != info.RSLevel {
		t.Errorf("RSLevel: got %v, want %v", got.RSLevel, info.RSLevel)
	}
	if got.Checksum != info.Checksum {
		t.Errorf("Checksum: got %d, want %d", got.Checksum, info.Checksum)
	}
	if got.ByteCountMod != info.ByteCountMod {
		t.Errorf("ByteCountMod: got %d, want %d", got.ByteCountMod, info.ByteCountMod)
	}
}

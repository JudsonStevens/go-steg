package image_processing

import (
	"go-steg/go_steg/reed_solomon"
	"image"
	"testing"
)

func TestHeaderRoundtrip(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	info := HeaderInfo{
		PhotoID:        12345,
		PhotoNumber:    3,
		DataCount:      5000,
		IsNewFormat:    true,
		FileExtension:  "pdf",
		BitDepth:       3,
		HuffmanEnabled: true,
		RSEnabled:      true,
		RSLevel:        reed_solomon.High,
		Checksum:       0xABC,
		ByteCountMod:   1234,
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
	if !got.IsNewFormat {
		t.Error("expected IsNewFormat=true")
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

func TestHeaderLegacyDetection(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	got := readHeader(img)
	if got.IsNewFormat {
		t.Error("expected legacy format for blank image")
	}
}

func TestHeaderEmptyExtension(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	info := HeaderInfo{
		IsNewFormat:   true,
		FileExtension: "",
		BitDepth:      2,
	}
	writeHeader(img, info)
	got := readHeader(img)
	if got.FileExtension != "" {
		t.Errorf("expected empty extension, got %q", got.FileExtension)
	}
}

func TestHeaderLongExtension(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	info := HeaderInfo{
		IsNewFormat:   true,
		FileExtension: "markdown", // 8 chars, max length
		BitDepth:      2,
	}
	writeHeader(img, info)
	got := readHeader(img)
	if got.FileExtension != "markdown" {
		t.Errorf("got %q, want %q", got.FileExtension, "markdown")
	}
}

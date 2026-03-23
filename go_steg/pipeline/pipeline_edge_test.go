package pipeline

import (
	"bytes"
	"go-steg/go_steg/reed_solomon"
	"strings"
	"testing"
)

func TestPipelineEncodeDecodeEmpty(t *testing.T) {
	configs := []struct {
		name   string
		config Config
	}{
		{"no transforms", Config{Password: "test"}},
		{"huffman only", Config{HuffmanEnabled: true, Password: "test"}},
		{"RS only", Config{RSEnabled: true, RSLevel: reed_solomon.Standard, Password: "test"}},
		{"huffman + RS", Config{HuffmanEnabled: true, RSEnabled: true, RSLevel: reed_solomon.Standard, Password: "test"}},
	}
	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := Encode([]byte{}, tc.config)
			if err != nil {
				t.Fatalf("Encode error: %v", err)
			}
			decoded, err := Decode(encoded, tc.config)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}
			if len(decoded) != 0 {
				t.Errorf("expected empty decoded, got len %d", len(decoded))
			}
		})
	}
}

func TestPipelineEncodeDecodeSingleByte(t *testing.T) {
	configs := []struct {
		name   string
		config Config
	}{
		{"no transforms", Config{Password: "test"}},
		{"huffman only", Config{HuffmanEnabled: true, Password: "test"}},
		{"RS only", Config{RSEnabled: true, RSLevel: reed_solomon.Standard, Password: "test"}},
		{"huffman + RS", Config{HuffmanEnabled: true, RSEnabled: true, RSLevel: reed_solomon.Standard, Password: "test"}},
		{"huffman + RS high", Config{HuffmanEnabled: true, RSEnabled: true, RSLevel: reed_solomon.High, Password: "test"}},
	}
	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{0x42}
			encoded, err := Encode(data, tc.config)
			if err != nil {
				t.Fatalf("Encode error: %v", err)
			}
			decoded, err := Decode(encoded, tc.config)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}
			if !bytes.Equal(decoded, data) {
				t.Errorf("roundtrip failed: got %v, want %v", decoded, data)
			}
		})
	}
}

func TestPipelineDecodeWrongPassword(t *testing.T) {
	data := []byte("secret message for pipeline wrong password test")
	cfg := Config{HuffmanEnabled: true, Password: "rightPassword"}
	encoded, err := Encode(data, cfg)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	wrongCfg := Config{HuffmanEnabled: true, Password: "wrongPassword"}
	decoded, err := Decode(encoded, wrongCfg)
	if err == nil && bytes.Equal(decoded, data) {
		t.Error("decoding with wrong password should not produce original data")
	}
}

func TestPipelineDecodeRSErrorPropagation(t *testing.T) {
	// Feed garbage data to Decode with RS enabled - should propagate RS error.
	garbage := []byte{0, 0, 0, 0, 0, 0, 0, 0} // valid prefix but no blocks
	cfg := Config{RSEnabled: true, RSLevel: reed_solomon.Standard}
	_, err := Decode(garbage, cfg)
	// The prefix says 0 blocks and 0 bytes, which means empty data
	// Let's try with a corrupted prefix instead
	garbage2 := []byte{1, 0, 0, 0, 100, 0, 0, 0} // 1 block, 100 bytes, but no block data
	_, err = Decode(garbage2, cfg)
	if err == nil {
		t.Error("expected error when RS decode gets truncated data")
	}
}

func TestPipelineDecodeHuffmanErrorPropagation(t *testing.T) {
	// Feed data too short for huffman (< 4 bytes)
	cfg := Config{HuffmanEnabled: true, Password: "test"}
	_, err := Decode([]byte{1, 2, 3}, cfg)
	if err == nil {
		t.Error("expected error from huffman decode with truncated data")
	}
	if err != nil && !strings.Contains(err.Error(), "huffman") {
		t.Errorf("error = %q, want to contain 'huffman'", err.Error())
	}
}

func TestPipelineLargeDataMultiBlock(t *testing.T) {
	// Data that spans multiple RS blocks
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	cfg := Config{
		HuffmanEnabled: true,
		RSEnabled:      true,
		RSLevel:        reed_solomon.Standard,
		Password:       "multiblock",
	}
	encoded, err := Encode(data, cfg)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	decoded, err := Decode(encoded, cfg)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Errorf("roundtrip failed for large multi-block data: got len %d, want len %d", len(decoded), len(data))
	}
}

func TestPipelineHuffmanOnlyEmptyPassword(t *testing.T) {
	data := []byte("empty password pipeline test")
	cfg := Config{HuffmanEnabled: true, Password: ""}
	encoded, err := Encode(data, cfg)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	decoded, err := Decode(encoded, cfg)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Errorf("roundtrip with empty password failed")
	}
}

func TestPipelineRSOnlyHighLevel(t *testing.T) {
	data := []byte("RS High level only pipeline test data")
	cfg := Config{RSEnabled: true, RSLevel: reed_solomon.High}
	encoded, err := Encode(data, cfg)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	decoded, err := Decode(encoded, cfg)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Errorf("roundtrip failed for RS High only")
	}
}

func TestPipelineDecodeWithWrongRSLevel(t *testing.T) {
	data := []byte("test wrong RS level")
	cfg := Config{RSEnabled: true, RSLevel: reed_solomon.Standard}
	encoded, err := Encode(data, cfg)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	wrongCfg := Config{RSEnabled: true, RSLevel: reed_solomon.High}
	decoded, err := Decode(encoded, wrongCfg)
	if err == nil && bytes.Equal(decoded, data) {
		t.Error("decoding with wrong RS level should not produce correct data")
	}
}

func TestPipelineAllByteValues(t *testing.T) {
	// Ensure all 256 byte values survive the pipeline
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	cfg := Config{
		HuffmanEnabled: true,
		RSEnabled:      true,
		RSLevel:        reed_solomon.Standard,
		Password:       "allbytes",
	}
	encoded, err := Encode(data, cfg)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	decoded, err := Decode(encoded, cfg)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Errorf("roundtrip failed for all byte values")
	}
}

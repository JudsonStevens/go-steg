package huffman

import (
	"reflect"
	"testing"
)

func TestGenerateTreeFromPassword(t *testing.T) {
	tree1, leaves1 := GenerateTreeFromPassword("testPassword")
	tree2, leaves2 := GenerateTreeFromPassword("testPassword")

	if tree1 == nil || tree2 == nil {
		t.Fatal("tree should not be nil")
	}
	if len(leaves1) != 256 || len(leaves2) != 256 {
		t.Fatalf("expected 256 leaves, got %d and %d", len(leaves1), len(leaves2))
	}

	// Verify determinism
	for i := 0; i < 256; i++ {
		code1, bits1 := leaves1[i].ReturnCode()
		code2, bits2 := leaves2[i].ReturnCode()
		if code1 != code2 || bits1 != bits2 {
			t.Errorf("byte %d: codes differ for same password", i)
		}
	}

	// Different password should produce different tree
	_, leaves3 := GenerateTreeFromPassword("differentPassword")
	differences := 0
	for i := 0; i < 256; i++ {
		code1, bits1 := leaves1[i].ReturnCode()
		code3, bits3 := leaves3[i].ReturnCode()
		if code1 != code3 || bits1 != bits3 {
			differences++
		}
	}
	if differences == 0 {
		t.Error("different passwords produced identical trees")
	}
}

func TestHuffmanEncodeDecodeRoundtrip(t *testing.T) {
	password := "testPassword"
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single byte", []byte{42}},
		{"hello", []byte("hello world")},
		{"all byte values", func() []byte {
			b := make([]byte, 256)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}()},
		{"repeated bytes", make([]byte, 1000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := HuffmanEncode(tt.data, password)
			decoded, err := HuffmanDecode(encoded, password)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if len(tt.data) == 0 && len(decoded) == 0 {
				return
			}
			if !reflect.DeepEqual(decoded, tt.data) {
				t.Errorf("roundtrip failed: got len %d, want len %d", len(decoded), len(tt.data))
			}
		})
	}
}

func TestHuffmanDecodeWrongPassword(t *testing.T) {
	data := []byte("secret message")
	encoded := HuffmanEncode(data, "rightPassword")
	decoded, err := HuffmanDecode(encoded, "wrongPassword")
	if err == nil && reflect.DeepEqual(decoded, data) {
		t.Error("expected different output or error with wrong password")
	}
}

func TestHuffmanDecodeTruncated(t *testing.T) {
	data := []byte("hello world")
	encoded := HuffmanEncode(data, "password")
	_, err := HuffmanDecode(encoded[:len(encoded)/2], "password")
	if err == nil {
		t.Error("expected error for truncated data")
	}
}

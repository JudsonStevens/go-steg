package huffman

import (
	"reflect"
	"strings"
	"testing"
)

func TestHuffmanDecodeDataTooShortForPrefix(t *testing.T) {
	// Data shorter than 4 bytes should return a specific error.
	inputs := [][]byte{
		{0},
		{0, 1},
		{0, 1, 2},
	}
	for _, input := range inputs {
		_, err := HuffmanDecode(input, "password")
		if err == nil {
			t.Errorf("HuffmanDecode(%v, ...) should return error for data shorter than 4 bytes", input)
		}
		if err != nil && !strings.Contains(err.Error(), "too short for length prefix") {
			t.Errorf("HuffmanDecode(%v, ...) error = %q, want 'too short for length prefix'", input, err.Error())
		}
	}
}

func TestHuffmanDecodeEmptyInput(t *testing.T) {
	decoded, err := HuffmanDecode([]byte{}, "password")
	if err != nil {
		t.Fatalf("HuffmanDecode(empty) returned error: %v", err)
	}
	if len(decoded) != 0 {
		t.Errorf("HuffmanDecode(empty) returned %d bytes, want 0", len(decoded))
	}
}

func TestHuffmanEncodeEmptyInput(t *testing.T) {
	encoded := HuffmanEncode([]byte{}, "password")
	if len(encoded) != 0 {
		t.Errorf("HuffmanEncode(empty) returned %d bytes, want 0", len(encoded))
	}
}

func TestHuffmanRoundtripSingleByte(t *testing.T) {
	// Test every possible single byte value
	password := "test"
	for b := 0; b < 256; b++ {
		data := []byte{byte(b)}
		encoded := HuffmanEncode(data, password)
		decoded, err := HuffmanDecode(encoded, password)
		if err != nil {
			t.Fatalf("HuffmanDecode failed for byte %d: %v", b, err)
		}
		if !reflect.DeepEqual(decoded, data) {
			t.Errorf("Roundtrip failed for byte %d: got %v", b, decoded)
		}
	}
}

func TestHuffmanRoundtripAllSameBytes(t *testing.T) {
	// All same byte value: tests the Huffman tree with very skewed frequency.
	password := "samebyte"
	for _, val := range []byte{0, 127, 255} {
		data := make([]byte, 500)
		for i := range data {
			data[i] = val
		}
		encoded := HuffmanEncode(data, password)
		decoded, err := HuffmanDecode(encoded, password)
		if err != nil {
			t.Fatalf("decode error for all-%d: %v", val, err)
		}
		if !reflect.DeepEqual(decoded, data) {
			t.Errorf("roundtrip failed for all-%d data", val)
		}
	}
}

func TestHuffmanRoundtripEmptyPassword(t *testing.T) {
	data := []byte("test with empty password")
	encoded := HuffmanEncode(data, "")
	decoded, err := HuffmanDecode(encoded, "")
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded, data) {
		t.Errorf("roundtrip with empty password failed")
	}
}

func TestHuffmanRoundtripSingleCharPassword(t *testing.T) {
	data := []byte("single char password test")
	encoded := HuffmanEncode(data, "x")
	decoded, err := HuffmanDecode(encoded, "x")
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded, data) {
		t.Errorf("roundtrip with single-char password failed")
	}
}

func TestHuffmanRoundtripLongPassword(t *testing.T) {
	data := []byte("testing with a very long password")
	longPass := strings.Repeat("abcdef1234567890", 100) // 1600 chars
	encoded := HuffmanEncode(data, longPass)
	decoded, err := HuffmanDecode(encoded, longPass)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded, data) {
		t.Errorf("roundtrip with long password failed")
	}
}

func TestHuffmanDecodeCorruptedLengthPrefix(t *testing.T) {
	// Encode normally, then corrupt the length prefix to a huge value.
	data := []byte("hello")
	encoded := HuffmanEncode(data, "password")

	// Set length prefix to a very large value
	encoded[0] = 0xFF
	encoded[1] = 0xFF
	encoded[2] = 0xFF
	encoded[3] = 0x0F // ~268 million bytes

	_, err := HuffmanDecode(encoded, "password")
	if err == nil {
		t.Error("expected error when length prefix exceeds available data")
	}
}

func TestHuffmanDecodeWrongPasswordProducesDifferentData(t *testing.T) {
	data := []byte("secret message for wrong password test")
	encoded := HuffmanEncode(data, "rightPassword")

	// With wrong password, decode should either error or produce different data
	decoded, err := HuffmanDecode(encoded, "wrongPassword")
	if err == nil && reflect.DeepEqual(decoded, data) {
		t.Error("decoding with wrong password should not produce original data")
	}
}

func TestHuffmanDecodeTruncatedAtVariousPoints(t *testing.T) {
	data := []byte("this is a longer message to test truncation at various points")
	encoded := HuffmanEncode(data, "password")

	// Truncate at the 4-byte header boundary (just the length prefix, no bits)
	_, err := HuffmanDecode(encoded[:4], "password")
	if err == nil {
		t.Error("expected error when truncated to just length prefix")
	}

	// Truncate at 5 bytes (length prefix + 1 byte of bits)
	if len(encoded) > 5 {
		_, err := HuffmanDecode(encoded[:5], "password")
		if err == nil {
			t.Error("expected error when truncated to 5 bytes")
		}
	}

	// Truncate at 75% of the encoded data
	cutPoint := len(encoded) * 3 / 4
	if cutPoint > 4 {
		_, err := HuffmanDecode(encoded[:cutPoint], "password")
		if err == nil {
			t.Error("expected error when truncated to 75%")
		}
	}
}

func TestHuffmanRoundtripLargeData(t *testing.T) {
	// Test with a large dataset to exercise the bit packing thoroughly.
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	password := "largetest"
	encoded := HuffmanEncode(data, password)
	decoded, err := HuffmanDecode(encoded, password)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded, data) {
		t.Errorf("roundtrip failed for large data: got len %d, want len %d", len(decoded), len(data))
	}
}

func TestGenerateTreeFromPasswordDeterminism(t *testing.T) {
	// Run multiple times with the same password to ensure determinism.
	password := "determinism_test"
	_, leaves1 := GenerateTreeFromPassword(password)

	for trial := 0; trial < 10; trial++ {
		_, leaves2 := GenerateTreeFromPassword(password)
		for i := 0; i < 256; i++ {
			code1, bits1 := leaves1[i].ReturnCode()
			code2, bits2 := leaves2[i].ReturnCode()
			if code1 != code2 || bits1 != bits2 {
				t.Fatalf("trial %d: byte %d codes differ for same password", trial, i)
			}
		}
	}
}

func TestGenerateTreeFromPasswordLeafValidity(t *testing.T) {
	_, leaves := GenerateTreeFromPassword("test")
	for i := 0; i < 256; i++ {
		if leaves[i] == nil {
			t.Fatalf("leaf %d is nil", i)
		}
		if leaves[i].Value != int32(i) {
			t.Errorf("leaf %d has Value %d, want %d", i, leaves[i].Value, i)
		}
		// Every leaf should have a non-zero code length
		_, bits := leaves[i].ReturnCode()
		if bits == 0 {
			t.Errorf("leaf %d has 0 bits code length", i)
		}
	}
}

func TestBuildTreeNilForEmptySlice(t *testing.T) {
	root := BuildTree([]*Node{})
	if root != nil {
		t.Error("BuildTree(empty) should return nil")
	}
}

func TestBuildTreeSingleNode(t *testing.T) {
	node := &Node{Count: 1, Value: 42}
	root := BuildTree([]*Node{node})
	if root != node {
		t.Error("BuildTree with single node should return that node")
	}
}

func TestHuffmanDecodeInvalidBitSequence(t *testing.T) {
	// Create encoded data, then flip bits in the encoded stream to create invalid paths.
	data := []byte("test")
	encoded := HuffmanEncode(data, "password")

	// Corrupt all bit stream bytes
	if len(encoded) > 4 {
		corrupted := make([]byte, len(encoded))
		copy(corrupted, encoded)
		for i := 4; i < len(corrupted); i++ {
			corrupted[i] ^= 0xFF
		}
		// With corrupted bit stream, decode may error or produce wrong data
		decoded, err := HuffmanDecode(corrupted, "password")
		if err == nil && reflect.DeepEqual(decoded, data) {
			t.Error("corrupted bit stream should not produce original data")
		}
	}
}

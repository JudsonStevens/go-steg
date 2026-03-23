package reed_solomon

import (
	"bytes"
	"strings"
	"testing"
)

func TestRSDecodeDataTooShortForPrefix(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"1 byte", []byte{1}},
		{"4 bytes", []byte{1, 2, 3, 4}},
		{"7 bytes", []byte{1, 2, 3, 4, 5, 6, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RSDecode(tt.data, Standard)
			if err == nil {
				t.Error("expected error for data too short for prefix")
			}
			if err != nil && !strings.Contains(err.Error(), "too short") {
				t.Errorf("error = %q, want to contain 'too short'", err.Error())
			}
		})
	}
}

func TestRSDecodeTruncatedBlocks(t *testing.T) {
	data := []byte("test data for truncation")
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}

	// Truncate so blocks are incomplete
	truncated := encoded[:prefixLen+100] // less than one full 255-byte block
	_, err = RSDecode(truncated, Standard)
	if err == nil {
		t.Error("expected error for truncated block data")
	}
	if err != nil && !strings.Contains(err.Error(), "expected") {
		t.Errorf("error = %q, want to contain 'expected'", err.Error())
	}
}

func TestRSDecodeWithWrongLevel(t *testing.T) {
	data := []byte("test data for wrong level")
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}

	// Decode with High level (different block size) - should produce
	// incorrect data or an error because block boundaries don't match.
	decoded, err := RSDecode(encoded, High)
	if err == nil && bytes.Equal(decoded, data) {
		t.Error("decoding with wrong RS level should not produce correct data")
	}
}

func TestRSRoundtripExactlyTwoBlocks(t *testing.T) {
	// Exactly 2 * 223 = 446 bytes for Standard
	data := make([]byte, 446)
	for i := range data {
		data[i] = byte(i % 256)
	}
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Error("roundtrip failed for exactly two Standard blocks")
	}
}

func TestRSRoundtripExactlyTwoBlocksHigh(t *testing.T) {
	// Exactly 2 * 191 = 382 bytes for High
	data := make([]byte, 382)
	for i := range data {
		data[i] = byte((i * 3 + 7) % 256)
	}
	encoded, err := RSEncode(data, High)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, High)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Error("roundtrip failed for exactly two High blocks")
	}
}

func TestRSRoundtripExactBlockBoundaryHigh(t *testing.T) {
	// Exactly 191 bytes (one full High block)
	data := make([]byte, 191)
	for i := range data {
		data[i] = byte(i)
	}
	encoded, err := RSEncode(data, High)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, High)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Error("roundtrip failed for exact High block size")
	}
}

func TestRSRoundtripOneByteOverBlock(t *testing.T) {
	// 224 bytes = 1 byte more than one Standard block (223)
	data := make([]byte, 224)
	for i := range data {
		data[i] = byte(i % 256)
	}
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Error("roundtrip failed for data one byte over block boundary")
	}
}

func TestRSCorruptionBeyondRecovery(t *testing.T) {
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}

	// Corrupt 17 bytes in the first block (beyond nsym=32/2=16 correctable)
	for i := 0; i < 17; i++ {
		encoded[prefixLen+i*10] ^= 0xFF
	}

	_, err = RSDecode(encoded, Standard)
	if err == nil {
		t.Error("expected error for corruption beyond recovery capability")
	}
}

func TestRSCorruptionBeyondRecoveryHigh(t *testing.T) {
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	encoded, err := RSEncode(data, High)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}

	// Corrupt 33 bytes in the first block (beyond nsym=64/2=32 correctable)
	for i := 0; i < 33; i++ {
		encoded[prefixLen+i*5] ^= 0xFF
	}

	_, err = RSDecode(encoded, High)
	if err == nil {
		t.Error("expected error for corruption beyond High recovery capability")
	}
}

func TestRSDecodeCorruptedPrefix(t *testing.T) {
	data := []byte("test prefix corruption")
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}

	// Corrupt the block count to a huge number
	corrupted := make([]byte, len(encoded))
	copy(corrupted, encoded)
	corrupted[0] = 0xFF
	corrupted[1] = 0xFF
	corrupted[2] = 0xFF
	corrupted[3] = 0x0F

	_, err = RSDecode(corrupted, Standard)
	if err == nil {
		t.Error("expected error for corrupted block count in prefix")
	}
}

func TestRSDecodeOriginalLengthExceedsDecoded(t *testing.T) {
	data := []byte("test")
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}

	// Corrupt the original length to a huge value
	corrupted := make([]byte, len(encoded))
	copy(corrupted, encoded)
	corrupted[4] = 0xFF
	corrupted[5] = 0xFF
	corrupted[6] = 0xFF
	corrupted[7] = 0x0F

	_, err = RSDecode(corrupted, Standard)
	if err == nil {
		t.Error("expected error when original length exceeds decoded data")
	}
	if err != nil && !strings.Contains(err.Error(), "original length") {
		t.Errorf("error = %q, want to contain 'original length'", err.Error())
	}
}

func TestRSEncodeSingleByte(t *testing.T) {
	// Single byte should produce exactly 1 block
	data := []byte{42}
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	expectedLen := prefixLen + 255 // 1 block
	if len(encoded) != expectedLen {
		t.Errorf("RSEncode(1 byte) length = %d, want %d", len(encoded), expectedLen)
	}
	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Errorf("roundtrip failed for single byte: got %v, want %v", decoded, data)
	}
}

func TestRSEncodeOutputFormat(t *testing.T) {
	data := make([]byte, 500)
	for i := range data {
		data[i] = byte(i % 256)
	}
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}

	// Verify prefix contains correct block count and length
	numBlocks := int(encoded[0]) | int(encoded[1])<<8 | int(encoded[2])<<16 | int(encoded[3])<<24
	origLen := int(encoded[4]) | int(encoded[5])<<8 | int(encoded[6])<<16 | int(encoded[7])<<24

	expectedBlocks := 3 // 500/223 = 2 remainder 54, so 3 blocks
	if numBlocks != expectedBlocks {
		t.Errorf("block count = %d, want %d", numBlocks, expectedBlocks)
	}
	if origLen != 500 {
		t.Errorf("original length = %d, want 500", origLen)
	}
	if len(encoded) != prefixLen+expectedBlocks*255 {
		t.Errorf("total length = %d, want %d", len(encoded), prefixLen+expectedBlocks*255)
	}
}

func TestParamsForLevelUnknown(t *testing.T) {
	// Unknown level should default to Standard
	d, p := paramsForLevel(RedundancyLevel(99))
	if d != 223 || p != 32 {
		t.Errorf("unknown level: got (%d, %d), want (223, 32)", d, p)
	}
}

func TestRSRoundtripAllZeroData(t *testing.T) {
	data := make([]byte, 223) // one block of all zeros
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Error("roundtrip failed for all-zero data")
	}
}

func TestRSRoundtripAllFFData(t *testing.T) {
	data := make([]byte, 223)
	for i := range data {
		data[i] = 0xFF
	}
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Error("roundtrip failed for all-0xFF data")
	}
}

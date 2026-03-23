package reed_solomon

import (
	"bytes"
	"testing"
)

func TestRoundtripStandard(t *testing.T) {
	data := []byte("Hello, Reed-Solomon!")
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Errorf("roundtrip failed: got %q, want %q", decoded, data)
	}
}

func TestRoundtripHigh(t *testing.T) {
	data := []byte("High redundancy test data!")
	encoded, err := RSEncode(data, High)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, High)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Errorf("roundtrip failed: got %q, want %q", decoded, data)
	}
}

func TestRoundtripExactBlock(t *testing.T) {
	// Exactly 223 bytes (one full Standard block)
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i)
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
		t.Error("roundtrip failed for exact block size")
	}
}

func TestRoundtripMultiBlock(t *testing.T) {
	// 500 bytes = 3 blocks at Standard (223 + 223 + 54)
	data := make([]byte, 500)
	for i := range data {
		data[i] = byte((i * 13 + 7) % 256)
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
		t.Error("roundtrip failed for multi-block data")
	}
}

func TestRoundtripSmall(t *testing.T) {
	data := []byte{42}
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Errorf("roundtrip failed: got %v, want %v", decoded, data)
	}
}

func TestRoundtripEmpty(t *testing.T) {
	data := []byte{}
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}
	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Errorf("roundtrip failed for empty data: got %v", decoded)
	}
}

func TestCorruptionRecoveryStandard(t *testing.T) {
	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i)
	}
	encoded, err := RSEncode(data, Standard)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}

	// Corrupt up to 16 bytes in the first block (max correctable for nsym=32)
	for i := 0; i < 16; i++ {
		encoded[8+i*10] ^= 0xFF // skip 8-byte prefix
	}

	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode with corruption: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Error("failed to recover from corruption")
	}
}

func TestCorruptionRecoveryHigh(t *testing.T) {
	data := make([]byte, 150)
	for i := range data {
		data[i] = byte(i)
	}
	encoded, err := RSEncode(data, High)
	if err != nil {
		t.Fatalf("RSEncode: %v", err)
	}

	// Corrupt up to 32 bytes (max correctable for nsym=64)
	for i := 0; i < 32; i++ {
		encoded[8+i*4] ^= 0xAB
	}

	decoded, err := RSDecode(encoded, High)
	if err != nil {
		t.Fatalf("RSDecode with corruption: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Error("failed to recover from corruption")
	}
}

func TestParamsForLevel(t *testing.T) {
	d, p := paramsForLevel(Standard)
	if d != 223 || p != 32 {
		t.Errorf("Standard: got (%d, %d), want (223, 32)", d, p)
	}
	d, p = paramsForLevel(High)
	if d != 191 || p != 64 {
		t.Errorf("High: got (%d, %d), want (191, 64)", d, p)
	}
}

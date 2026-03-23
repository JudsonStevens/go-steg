package reed_solomon

import (
	"testing"
)

func TestDecodeBlockAllZeros(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	decoded, err := decodeBlock(codeword, nsym)
	if err != nil {
		t.Fatalf("decodeBlock error: %v", err)
	}
	for i := 0; i < 223; i++ {
		if decoded[i] != 0 {
			t.Errorf("decoded[%d] = %d, want 0", i, decoded[i])
		}
	}
}

func TestDecodeBlockSingleByteErrorAtEveryPosition(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte((i * 13 + 5) % 256)
	}
	parity := encodeBlock(data, nsym)

	// Test error correction at first, middle, and last positions
	positions := []int{0, 1, 111, 222, 223, 240, 254}
	for _, pos := range positions {
		codeword := make([]byte, 255)
		copy(codeword, data)
		copy(codeword[223:], parity)
		codeword[pos] ^= 0x42

		decoded, err := decodeBlock(codeword, nsym)
		if err != nil {
			t.Errorf("decodeBlock error at pos %d: %v", pos, err)
			continue
		}
		for i := 0; i < 223; i++ {
			if decoded[i] != data[i] {
				t.Errorf("pos %d: decoded[%d] = %d, want %d", pos, i, decoded[i], data[i])
			}
		}
	}
}

func TestDecodeBlockExactMaxErrors(t *testing.T) {
	// For nsym=64 (High), max correctable = 32
	nsym := 64
	data := make([]byte, 191)
	for i := range data {
		data[i] = byte(i % 256)
	}
	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[191:], parity)

	// Corrupt exactly 32 positions
	for i := 0; i < 32; i++ {
		codeword[i*7%255] ^= byte(i + 1)
	}

	decoded, err := decodeBlock(codeword, nsym)
	if err != nil {
		t.Fatalf("decodeBlock error: %v", err)
	}
	for i := 0; i < 191; i++ {
		if decoded[i] != data[i] {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], data[i])
		}
	}
}

func TestDecodeBlockOnePastMaxErrors(t *testing.T) {
	// For nsym=64, 33 errors should fail
	nsym := 64
	data := make([]byte, 191)
	for i := range data {
		data[i] = byte(i % 256)
	}
	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[191:], parity)

	// Corrupt 33 positions
	for i := 0; i < 33; i++ {
		codeword[i*7%255] ^= byte(i + 1)
	}

	_, err := decodeBlock(codeword, nsym)
	if err == nil {
		t.Error("expected error for 33 errors with nsym=64")
	}
}

func TestDecodeBlockMinNsym(t *testing.T) {
	// nsym=2: can correct 1 error
	nsym := 2
	data := make([]byte, 253)
	for i := range data {
		data[i] = byte(i % 256)
	}
	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[253:], parity)

	// Single error
	codeword[100] ^= 0xFF

	decoded, err := decodeBlock(codeword, nsym)
	if err != nil {
		t.Fatalf("decodeBlock error: %v", err)
	}
	for i := 0; i < 253; i++ {
		if decoded[i] != data[i] {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], data[i])
		}
	}
}

func TestDecodeBlockTwoErrorsWithMinNsym(t *testing.T) {
	// nsym=2: 2 errors should fail (max correctable = 1)
	nsym := 2
	data := make([]byte, 253)
	for i := range data {
		data[i] = byte(i % 256)
	}
	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[253:], parity)

	codeword[50] ^= 0xFF
	codeword[100] ^= 0xFF

	_, err := decodeBlock(codeword, nsym)
	if err == nil {
		t.Error("expected error for 2 errors with nsym=2")
	}
}

func TestDecodeBlockErrorAtFirstAndLastPosition(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}
	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	// Corrupt first and last byte
	codeword[0] ^= 0xAB
	codeword[254] ^= 0xCD

	decoded, err := decodeBlock(codeword, nsym)
	if err != nil {
		t.Fatalf("decodeBlock error: %v", err)
	}
	for i := 0; i < 223; i++ {
		if decoded[i] != data[i] {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], data[i])
		}
	}
}

func TestDecodeBlockConsecutiveErrors(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}
	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	// Corrupt 16 consecutive bytes
	for i := 0; i < 16; i++ {
		codeword[i] ^= 0xFF
	}

	decoded, err := decodeBlock(codeword, nsym)
	if err != nil {
		t.Fatalf("decodeBlock error: %v", err)
	}
	for i := 0; i < 223; i++ {
		if decoded[i] != data[i] {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], data[i])
		}
	}
}

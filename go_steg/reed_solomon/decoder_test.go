package reed_solomon

import "testing"

func TestDecodeNoErrors(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}

	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	decoded, err := decodeBlock(codeword, nsym)
	if err != nil {
		t.Fatalf("decodeBlock returned error: %v", err)
	}

	for i := 0; i < 223; i++ {
		if decoded[i] != data[i] {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], data[i])
		}
	}
}

func TestDecode16Errors(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}

	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	// Corrupt 16 positions (max correctable for nsym=32)
	for i := 0; i < 16; i++ {
		codeword[i*10] ^= 0xFF
	}

	decoded, err := decodeBlock(codeword, nsym)
	if err != nil {
		t.Fatalf("decodeBlock returned error: %v", err)
	}

	for i := 0; i < 223; i++ {
		if decoded[i] != data[i] {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], data[i])
		}
	}
}

func TestDecode17ErrorsFails(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}

	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	// Corrupt 17 positions (beyond max correctable)
	for i := 0; i < 17; i++ {
		codeword[i*15%255] ^= 0xFF
	}

	_, err := decodeBlock(codeword, nsym)
	if err == nil {
		t.Error("decodeBlock should return error for 17 errors with nsym=32")
	}
}

func TestDecodeSingleError(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte((i * 7 + 3) % 256)
	}

	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	// Single error
	codeword[100] ^= 0x42

	decoded, err := decodeBlock(codeword, nsym)
	if err != nil {
		t.Fatalf("decodeBlock returned error: %v", err)
	}

	for i := 0; i < 223; i++ {
		if decoded[i] != data[i] {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], data[i])
		}
	}
}

func TestDecodeErrorInParity(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}

	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	// Corrupt a parity byte
	codeword[230] ^= 0xAB

	decoded, err := decodeBlock(codeword, nsym)
	if err != nil {
		t.Fatalf("decodeBlock returned error: %v", err)
	}

	// Data portion should be correct
	for i := 0; i < 223; i++ {
		if decoded[i] != data[i] {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], data[i])
		}
	}
}

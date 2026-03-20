package reed_solomon

import (
	"testing"
)

func TestGeneratorPolyLeadingCoefficient(t *testing.T) {
	// All generator polynomials should have leading coefficient 1
	for _, nsym := range []int{1, 2, 4, 8, 16, 32, 64} {
		g := generatorPoly(nsym)
		if g[0] != 1 {
			t.Errorf("generatorPoly(%d)[0] = %d, want 1", nsym, g[0])
		}
		if len(g) != nsym+1 {
			t.Errorf("generatorPoly(%d) length = %d, want %d", nsym, len(g), nsym+1)
		}
	}
}

func TestPolyMulByZero(t *testing.T) {
	initTables()
	a := []byte{1, 2, 3}
	zero := []byte{0}
	result := polyMul(a, zero)
	for i, v := range result {
		if v != 0 {
			t.Errorf("polyMul by zero: result[%d] = %d, want 0", i, v)
		}
	}
}

func TestEncodeBlockAllZeros(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	parity := encodeBlock(data, nsym)
	if len(parity) != nsym {
		t.Fatalf("parity length = %d, want %d", len(parity), nsym)
	}
	// All-zero data should produce all-zero parity
	for i, v := range parity {
		if v != 0 {
			t.Errorf("parity[%d] = %d for all-zero data, want 0", i, v)
		}
	}
}

func TestEncodeBlockAllOnes(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = 0xFF
	}
	parity := encodeBlock(data, nsym)
	if len(parity) != nsym {
		t.Fatalf("parity length = %d, want %d", len(parity), nsym)
	}
	// Verify syndromes are zero for the codeword
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)
	syndromes := computeSyndromes(codeword, nsym)
	for i, s := range syndromes {
		if s != 0 {
			t.Errorf("syndrome[%d] = %d for all-0xFF data, want 0", i, s)
		}
	}
}

func TestEncodeBlockSmallNsym(t *testing.T) {
	// nsym=2: minimal error correction (correct 1 error)
	nsym := 2
	data := make([]byte, 253)
	for i := range data {
		data[i] = byte(i % 256)
	}
	parity := encodeBlock(data, nsym)
	if len(parity) != nsym {
		t.Fatalf("parity length = %d, want %d", len(parity), nsym)
	}
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[253:], parity)
	syndromes := computeSyndromes(codeword, nsym)
	for i, s := range syndromes {
		if s != 0 {
			t.Errorf("syndrome[%d] = %d, want 0", i, s)
		}
	}
}

func TestEncodeBlockNsym64(t *testing.T) {
	// nsym=64: High redundancy
	nsym := 64
	data := make([]byte, 191)
	for i := range data {
		data[i] = byte(i % 256)
	}
	parity := encodeBlock(data, nsym)
	if len(parity) != nsym {
		t.Fatalf("parity length = %d, want %d", len(parity), nsym)
	}
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[191:], parity)
	syndromes := computeSyndromes(codeword, nsym)
	for i, s := range syndromes {
		if s != 0 {
			t.Errorf("syndrome[%d] = %d, want 0", i, s)
		}
	}
}

func TestComputeSyndromesDeterministic(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}
	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	// Compute syndromes twice and verify they match
	syn1 := computeSyndromes(codeword, nsym)
	syn2 := computeSyndromes(codeword, nsym)
	for i := range syn1 {
		if syn1[i] != syn2[i] {
			t.Errorf("syndromes not deterministic: syn[%d] = %d vs %d", i, syn1[i], syn2[i])
		}
	}
}

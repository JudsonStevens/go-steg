package reed_solomon

import "testing"

func TestGeneratorPolyLength(t *testing.T) {
	g := generatorPoly(32)
	if len(g) != 33 {
		t.Errorf("generatorPoly(32) length = %d, want 33", len(g))
	}
	// Leading coefficient should be 1
	if g[0] != 1 {
		t.Errorf("generatorPoly(32)[0] = %d, want 1", g[0])
	}
}

func TestPolyMulIdentity(t *testing.T) {
	initTables()
	a := []byte{1, 2, 3}
	// Multiply by [1] (identity)
	result := polyMul(a, []byte{1})
	if len(result) != len(a) {
		t.Fatalf("polyMul identity: length %d, want %d", len(result), len(a))
	}
	for i := range a {
		if result[i] != a[i] {
			t.Errorf("polyMul identity: result[%d] = %d, want %d", i, result[i], a[i])
		}
	}
}

func TestEncodeBlockSyndromesAllZero(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}

	parity := encodeBlock(data, nsym)
	if len(parity) != nsym {
		t.Fatalf("encodeBlock parity length = %d, want %d", len(parity), nsym)
	}

	// Full codeword = data + parity
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	syndromes := computeSyndromes(codeword, nsym)
	for i, s := range syndromes {
		if s != 0 {
			t.Errorf("syndrome[%d] = %d, want 0", i, s)
		}
	}
}

func TestEncodeBlockSyndromesNonZeroOnCorruption(t *testing.T) {
	nsym := 32
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}

	parity := encodeBlock(data, nsym)
	codeword := make([]byte, 255)
	copy(codeword, data)
	copy(codeword[223:], parity)

	// Corrupt one byte
	codeword[0] ^= 0xFF

	syndromes := computeSyndromes(codeword, nsym)
	allZero := true
	for _, s := range syndromes {
		if s != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("syndromes should be nonzero after corruption")
	}
}

func TestEncodeBlockDifferentData(t *testing.T) {
	nsym := 32
	// Two different data blocks should produce different parity
	data1 := make([]byte, 223)
	data2 := make([]byte, 223)
	data2[0] = 1

	p1 := encodeBlock(data1, nsym)
	p2 := encodeBlock(data2, nsym)

	same := true
	for i := range p1 {
		if p1[i] != p2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different data should produce different parity")
	}
}

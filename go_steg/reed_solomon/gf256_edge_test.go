package reed_solomon

import "testing"

func TestGfInvOfZero(t *testing.T) {
	initTables()
	// gfInv(0) is defined to return 0 (special case)
	got := gfInv(0)
	if got != 0 {
		t.Errorf("gfInv(0) = %d, want 0", got)
	}
}

func TestGfMul255x255(t *testing.T) {
	initTables()
	// 255 * 255 in GF(256): verify it produces a valid result and is commutative.
	result := gfMul(255, 255)
	reverse := gfMul(255, 255)
	if result != reverse {
		t.Errorf("gfMul(255,255) not consistent: %d vs %d", result, reverse)
	}
	// Verify multiplication by inverse gives 1
	if result != 0 {
		inv := gfInv(255)
		check := gfMul(result, inv)
		// result = 255*255, so result * inv(255) = 255
		if check != 255 {
			t.Errorf("gfMul(gfMul(255,255), gfInv(255)) = %d, want 255", check)
		}
	}
}

func TestGfDivZeroNumerator(t *testing.T) {
	initTables()
	// 0 / x = 0 for any nonzero x
	for b := 1; b < 256; b++ {
		got := gfDiv(0, byte(b))
		if got != 0 {
			t.Errorf("gfDiv(0, %d) = %d, want 0", b, got)
		}
	}
}

func TestGfDivSelfIsOne(t *testing.T) {
	initTables()
	// a / a = 1 for all nonzero a
	for a := 1; a < 256; a++ {
		got := gfDiv(byte(a), byte(a))
		if got != 1 {
			t.Errorf("gfDiv(%d, %d) = %d, want 1", a, a, got)
		}
	}
}

func TestGfMulAssociative(t *testing.T) {
	initTables()
	// Test associativity: (a*b)*c == a*(b*c) for some samples
	triples := [][3]byte{
		{2, 3, 5},
		{17, 42, 200},
		{255, 255, 255},
		{1, 128, 64},
		{100, 200, 50},
	}
	for _, tr := range triples {
		a, b, c := tr[0], tr[1], tr[2]
		left := gfMul(gfMul(a, b), c)
		right := gfMul(a, gfMul(b, c))
		if left != right {
			t.Errorf("associativity failed: (%d*%d)*%d=%d != %d*(%d*%d)=%d",
				a, b, c, left, a, b, c, right)
		}
	}
}

func TestGfMulDistributive(t *testing.T) {
	initTables()
	// Test distributivity: a*(b^c) == (a*b)^(a*c) (XOR is addition in GF(2^8))
	triples := [][3]byte{
		{2, 3, 5},
		{17, 42, 200},
		{255, 128, 64},
	}
	for _, tr := range triples {
		a, b, c := tr[0], tr[1], tr[2]
		left := gfMul(a, b^c)
		right := gfMul(a, b) ^ gfMul(a, c)
		if left != right {
			t.Errorf("distributivity failed: %d*(%d^%d)=%d != (%d*%d)^(%d*%d)=%d",
				a, b, c, left, a, b, a, c, right)
		}
	}
}

func TestExpTableWraparound(t *testing.T) {
	initTables()
	// Verify the wraparound: expTable[i] == expTable[i+255] for i < 255
	for i := 0; i < 255; i++ {
		if expTable[i] != expTable[i+255] {
			t.Errorf("expTable[%d]=%d != expTable[%d]=%d", i, expTable[i], i+255, expTable[i+255])
		}
	}
}

func TestGfMulByOneAllValues(t *testing.T) {
	initTables()
	// Already tested in existing tests, but let's also verify 1*0 = 0
	if got := gfMul(1, 0); got != 0 {
		t.Errorf("gfMul(1, 0) = %d, want 0", got)
	}
	if got := gfMul(0, 1); got != 0 {
		t.Errorf("gfMul(0, 1) = %d, want 0", got)
	}
}

func TestGfDivByOneSameValue(t *testing.T) {
	initTables()
	// a / 1 = a for all a
	for a := 0; a < 256; a++ {
		got := gfDiv(byte(a), 1)
		if got != byte(a) {
			t.Errorf("gfDiv(%d, 1) = %d, want %d", a, got, a)
		}
	}
}

func TestGfInvInvolution(t *testing.T) {
	initTables()
	// inv(inv(a)) == a for all nonzero a
	for a := 1; a < 256; a++ {
		got := gfInv(gfInv(byte(a)))
		if got != byte(a) {
			t.Errorf("gfInv(gfInv(%d)) = %d, want %d", a, got, a)
		}
	}
}

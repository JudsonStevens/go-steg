package reed_solomon

import "testing"

func TestGfMulOverflow(t *testing.T) {
	initTables()
	// 2 * 128 = 256, reduced mod 0x11D (x^8+x^4+x^3+x^2+1) => 256 XOR 285 = 29
	got := gfMul(2, 128)
	if got != 29 {
		t.Errorf("gfMul(2, 128) = %d, want 29", got)
	}
}

func TestGfMulByZero(t *testing.T) {
	initTables()
	if got := gfMul(0, 42); got != 0 {
		t.Errorf("gfMul(0, 42) = %d, want 0", got)
	}
	if got := gfMul(42, 0); got != 0 {
		t.Errorf("gfMul(42, 0) = %d, want 0", got)
	}
}

func TestGfMulByOne(t *testing.T) {
	initTables()
	for a := 0; a < 256; a++ {
		got := gfMul(byte(a), 1)
		if got != byte(a) {
			t.Errorf("gfMul(%d, 1) = %d, want %d", a, got, a)
		}
	}
}

func TestGfInvAllNonzero(t *testing.T) {
	initTables()
	for a := 1; a < 256; a++ {
		inv := gfInv(byte(a))
		product := gfMul(byte(a), inv)
		if product != 1 {
			t.Errorf("gfMul(%d, gfInv(%d)) = %d, want 1", a, a, product)
		}
	}
}

func TestGfDivByZeroPanics(t *testing.T) {
	initTables()
	defer func() {
		if r := recover(); r == nil {
			t.Error("gfDiv(1, 0) did not panic")
		}
	}()
	gfDiv(1, 0)
}

func TestGfDivRoundtrip(t *testing.T) {
	initTables()
	for a := 1; a < 256; a++ {
		for b := 1; b < 256; b++ {
			product := gfMul(byte(a), byte(b))
			quotient := gfDiv(product, byte(b))
			if quotient != byte(a) {
				t.Errorf("gfDiv(gfMul(%d, %d), %d) = %d, want %d", a, b, b, quotient, a)
			}
		}
	}
}

func TestGfMulCommutative(t *testing.T) {
	initTables()
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			ab := gfMul(byte(a), byte(b))
			ba := gfMul(byte(b), byte(a))
			if ab != ba {
				t.Errorf("gfMul(%d,%d)=%d != gfMul(%d,%d)=%d", a, b, ab, b, a, ba)
			}
		}
	}
}

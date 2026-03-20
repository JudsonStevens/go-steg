package image_processing

import (
	"reflect"
	"testing"
)

func TestAlignNVariousValues(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		n    int
		want []byte
	}{
		{"n=3, len=1", []byte{1}, 3, []byte{1, 0, 0}},
		{"n=3, len=2", []byte{1, 2}, 3, []byte{1, 2, 0}},
		{"n=3, len=3", []byte{1, 2, 3}, 3, []byte{1, 2, 3}},
		{"n=3, len=4", []byte{1, 2, 3, 4}, 3, []byte{1, 2, 3, 4, 0, 0}},
		{"n=3, len=5", []byte{1, 2, 3, 4, 5}, 3, []byte{1, 2, 3, 4, 5, 0}},
		{"n=3, len=6", []byte{1, 2, 3, 4, 5, 6}, 3, []byte{1, 2, 3, 4, 5, 6}},
		{"n=8, len=1", []byte{1}, 8, []byte{1, 0, 0, 0, 0, 0, 0, 0}},
		{"n=8, len=8", []byte{1, 2, 3, 4, 5, 6, 7, 8}, 8, []byte{1, 2, 3, 4, 5, 6, 7, 8}},
		{"n=1, len=1", []byte{1}, 1, []byte{1}},
		{"n=1, len=5", []byte{1, 2, 3, 4, 5}, 1, []byte{1, 2, 3, 4, 5}},
		{"n=2, len=1", []byte{1}, 2, []byte{1, 0}},
		{"n=2, len=2", []byte{1, 2}, 2, []byte{1, 2}},
		{"n=4, len=0", []byte{}, 4, []byte{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := alignN(tt.data, tt.n)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("alignN(%v, %d) = %v, want %v", tt.data, tt.n, got, tt.want)
			}
		})
	}
}

func TestAlignEmpty(t *testing.T) {
	got := align([]byte{})
	if len(got) != 0 {
		t.Errorf("align(empty) returned len %d, want 0", len(got))
	}
}

func TestAlignLargeInput(t *testing.T) {
	// 7 bytes should pad to 8
	data := []byte{1, 2, 3, 4, 5, 6, 7}
	got := align(data)
	if len(got) != 8 {
		t.Errorf("align(7 bytes) returned len %d, want 8", len(got))
	}
	// 8 bytes should stay 8
	data = []byte{1, 2, 3, 4, 5, 6, 7, 8}
	got = align(data)
	if len(got) != 8 {
		t.Errorf("align(8 bytes) returned len %d, want 8", len(got))
	}
}

func TestAlignNIdempotent(t *testing.T) {
	// Aligning already-aligned data should be a no-op
	for n := 1; n <= 8; n++ {
		data := make([]byte, n*3) // already aligned to n
		for i := range data {
			data[i] = byte(i)
		}
		got := alignN(data, n)
		if len(got) != len(data) {
			t.Errorf("alignN(aligned data, %d): len changed from %d to %d", n, len(data), len(got))
		}
	}
}

func TestMultiCarrierDecodeByFileNamesEmptyCarriers(t *testing.T) {
	err := MultiCarrierDecodeByFileNames([]string{}, "password", "/tmp")
	if err == nil {
		t.Error("expected error for empty carrier list")
	}
}

func TestComputeChecksumDeterministic(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	c1 := computeChecksum(data)
	c2 := computeChecksum(data)
	if c1 != c2 {
		t.Errorf("computeChecksum not deterministic: %d vs %d", c1, c2)
	}
}

func TestComputeChecksumDifferentData(t *testing.T) {
	data1 := []byte{1, 2, 3, 4}
	data2 := []byte{5, 6, 7, 8}
	c1 := computeChecksum(data1)
	c2 := computeChecksum(data2)
	if c1 == c2 {
		t.Error("different data should produce different checksums (with high probability)")
	}
}

func TestComputeChecksumMaxBits(t *testing.T) {
	// Checksum should be at most 12 bits (0x0FFF)
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	c := computeChecksum(data)
	if c > 0x0FFF {
		t.Errorf("checksum %d exceeds 12 bits (max 4095)", c)
	}
}

func TestComputeChecksumShortData(t *testing.T) {
	// 1 byte data
	c := computeChecksum([]byte{42})
	if c > 0x0FFF {
		t.Errorf("checksum %d exceeds 12 bits", c)
	}
	// 0 bytes - should not panic
	// computeChecksum takes first min(4, len) bytes; for empty, n=0
	// This would cause crc32.ChecksumIEEE(data[:0]) which is valid
}

func TestComputeChecksumEmpty(t *testing.T) {
	c := computeChecksum([]byte{})
	if c > 0x0FFF {
		t.Errorf("checksum %d exceeds 12 bits", c)
	}
}

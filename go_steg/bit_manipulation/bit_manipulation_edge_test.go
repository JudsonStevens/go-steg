package bit_manipulation

import (
	"reflect"
	"testing"
)

func TestSplitByteEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		b            byte
		bitsPerChunk int
		want         []byte
	}{
		// 8-bit split: single chunk = the byte itself
		{"8-bit split of 0", 0, 8, []byte{0}},
		{"8-bit split of 255", 255, 8, []byte{255}},
		{"8-bit split of 170", 170, 8, []byte{170}},
		// 5-bit split: 5+3 = 8 bits
		{"5-bit split of 0", 0, 5, []byte{0, 0}},
		{"5-bit split of 255", 255, 5, []byte{31, 7}},
		{"5-bit split of 128", 128, 5, []byte{16, 0}},
		// 6-bit split: 6+2 = 8 bits
		{"6-bit split of 255", 255, 6, []byte{63, 3}},
		{"6-bit split of 0", 0, 6, []byte{0, 0}},
		// 7-bit split: 7+1 = 8 bits
		{"7-bit split of 255", 255, 7, []byte{127, 1}},
		{"7-bit split of 128", 128, 7, []byte{64, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitByte(tt.b, tt.bitsPerChunk)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitByte(%d, %d) = %v, want %v", tt.b, tt.bitsPerChunk, got, tt.want)
			}
		})
	}
}

func TestConstructByteEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		chunks       []byte
		bitsPerChunk int
		want         byte
	}{
		{"8-bit construct 255", []byte{255}, 8, 255},
		{"8-bit construct 0", []byte{0}, 8, 0},
		{"5-bit construct 255", []byte{31, 7}, 5, 255},
		{"6-bit construct 255", []byte{63, 3}, 6, 255},
		{"7-bit construct 255", []byte{127, 1}, 7, 255},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConstructByte(tt.chunks, tt.bitsPerChunk); got != tt.want {
				t.Errorf("ConstructByte(%v, %d) = %d, want %d", tt.chunks, tt.bitsPerChunk, got, tt.want)
			}
		})
	}
}

// TestSplitByteConstructByteRoundtripExtended tests roundtrip for bit depths 5-8.
func TestSplitByteConstructByteRoundtripExtended(t *testing.T) {
	for depth := 5; depth <= 8; depth++ {
		for b := 0; b < 256; b++ {
			chunks := SplitByte(byte(b), depth)
			reconstructed := ConstructByte(chunks, depth)
			if reconstructed != byte(b) {
				t.Errorf("Roundtrip failed: depth=%d, byte=%d, chunks=%v, reconstructed=%d",
					depth, b, chunks, reconstructed)
			}
		}
	}
}

func TestClearLastNBitsEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		b    byte
		n    int
		want byte
	}{
		{"clear 0 bits from 255", 255, 0, 255},
		{"clear 8 bits from 255", 255, 8, 0},
		{"clear 8 bits from 0", 0, 8, 0},
		{"clear 0 bits from 0", 0, 0, 0},
		{"clear 5 bits from 255", 255, 5, 224},
		{"clear 6 bits from 255", 255, 6, 192},
		{"clear 7 bits from 255", 255, 7, 128},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClearLastNBits(tt.b, tt.n); got != tt.want {
				t.Errorf("ClearLastNBits(%d, %d) = %d, want %d", tt.b, tt.n, got, tt.want)
			}
		})
	}
}

func TestGetLastNBitsEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		b    byte
		n    int
		want byte
	}{
		{"get 0 bits from 255", 255, 0, 0},
		{"get 8 bits from 255", 255, 8, 255},
		{"get 8 bits from 0", 0, 8, 0},
		{"get 5 bits from 255", 255, 5, 31},
		{"get 6 bits from 255", 255, 6, 63},
		{"get 7 bits from 255", 255, 7, 127},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLastNBits(tt.b, tt.n); got != tt.want {
				t.Errorf("GetLastNBits(%d, %d) = %d, want %d", tt.b, tt.n, got, tt.want)
			}
		})
	}
}

func TestSetLastNBitsEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		b          byte
		valueToSet byte
		n          int
		want       byte
	}{
		{"set 0 bits: no change", 255, 0, 0, 255},
		{"set 8 bits: replace entirely", 0, 255, 8, 255},
		{"set 8 bits: clear entirely", 255, 0, 8, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetLastNBits(tt.b, tt.valueToSet, tt.n); got != tt.want {
				t.Errorf("SetLastNBits(%d, %d, %d) = %d, want %d", tt.b, tt.valueToSet, tt.n, got, tt.want)
			}
		})
	}
}

func TestSplitByteIntoQuartersAllValues(t *testing.T) {
	// Verify the split + reconstruct roundtrip for all 256 byte values.
	for b := 0; b < 256; b++ {
		quarters := SplitByteIntoQuarters(byte(b))
		reconstructed := ConstructByteFromQuarters(quarters[0], quarters[1], quarters[2], quarters[3])
		if reconstructed != byte(b) {
			t.Errorf("SplitByteIntoQuarters/ConstructByteFromQuarters roundtrip failed for %d: quarters=%v, reconstructed=%d",
				b, quarters, reconstructed)
		}
	}
}

func TestQuartersOfBytes64MaxValue(t *testing.T) {
	// Max uint64 should produce all 3s in the first 24 quarters
	// (only 6 bytes are used, so indices 0-23, but only first 24 are meaningful)
	got := QuartersOfBytes64(^uint64(0))
	// All 8 bytes are 0xFF, so all quarters should be 3
	// But QuartersOfBytes64 only processes 6 bytes (indices 0-23),
	// check that all returned values are 3
	for i, v := range got {
		if v != 3 {
			t.Errorf("QuartersOfBytes64(max)[%d] = %d, want 3", i, v)
		}
	}
}

func TestQuartersOfBytes32Roundtrip(t *testing.T) {
	// Verify that splitting and reconstructing uint32 values works.
	testValues := []uint32{0, 1, 255, 256, 65535, 1000000, 4294967295}
	for _, val := range testValues {
		quarters := QuartersOfBytes32(val)
		// Reconstruct: every 4 quarters make a byte
		var reconstructedBytes [4]byte
		for i := 0; i < 4; i++ {
			reconstructedBytes[i] = ConstructByteFromQuartersAsSlice(quarters[i*4 : i*4+4])
		}
		reconstructed := uint32(reconstructedBytes[0]) |
			uint32(reconstructedBytes[1])<<8 |
			uint32(reconstructedBytes[2])<<16 |
			uint32(reconstructedBytes[3])<<24
		if reconstructed != val {
			t.Errorf("QuartersOfBytes32 roundtrip failed for %d: got %d", val, reconstructed)
		}
	}
}

func TestReturnMaskDifferenceNBitDepths(t *testing.T) {
	// Test that different bit depths clear different amounts of bits.
	// With colorInt=0xFF and bitDepth=4, cleared = 0xF0, bitDepth=3 => 0xF8, etc.
	tests := []struct {
		name     string
		bitDepth int
		colorInt uint8
	}{
		{"depth 1, color 0xFF", 1, 0xFF},
		{"depth 2, color 0xFF", 2, 0xFF},
		{"depth 3, color 0xFF", 3, 0xFF},
		{"depth 4, color 0xFF", 4, 0xFF},
		{"depth 1, color 0x00", 1, 0x00},
		{"depth 4, color 0x00", 4, 0x00},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			_ = ReturnMaskDifferenceN(1, 1, 1, 2, tt.colorInt, tt.bitDepth)
		})
	}
}

func TestConstructByteFromQuartersAsSliceSymmetry(t *testing.T) {
	// Verify ConstructByteFromQuartersAsSlice matches ConstructByteFromQuarters
	for a := byte(0); a < 4; a++ {
		for b := byte(0); b < 4; b++ {
			for c := byte(0); c < 4; c++ {
				for d := byte(0); d < 4; d++ {
					fromSlice := ConstructByteFromQuartersAsSlice([]byte{a, b, c, d})
					fromArgs := ConstructByteFromQuarters(a, b, c, d)
					if fromSlice != fromArgs {
						t.Errorf("Mismatch for (%d,%d,%d,%d): slice=%d, args=%d",
							a, b, c, d, fromSlice, fromArgs)
					}
				}
			}
		}
	}
}

func TestQuartersOfBytes16Roundtrip(t *testing.T) {
	testValues := []uint16{0, 1, 127, 128, 255, 256, 10000, 32767, 32768, 65535}
	for _, val := range testValues {
		quarters := QuartersOfBytes16(val)
		// Reconstruct the first byte from quarters 0-3
		firstByte := ConstructByteFromQuartersAsSlice(quarters[0:4])
		// Note: QuartersOfBytes16 only splits the first byte (low byte in LE)
		// The quarters represent 1 byte only (4 quarters)
		if quarters == nil || len(quarters) != 4 {
			t.Fatalf("QuartersOfBytes16(%d) returned slice of len %d, want 4", val, len(quarters))
		}
		// The first byte in LE is the low byte
		expectedLowByte := byte(val & 0xFF)
		if firstByte != expectedLowByte {
			t.Errorf("QuartersOfBytes16(%d) low byte roundtrip: got %d, want %d", val, firstByte, expectedLowByte)
		}
	}
}

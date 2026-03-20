# Pipeline & Features Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a layered transform pipeline to go-steg supporting arbitrary file types, Huffman compression, Reed-Solomon error correction, and adjustable bit depth (1-4 bits/channel).

**Architecture:** Each transform stage is a standalone `[]byte → []byte` function. The pipeline orchestrator in `go_steg/pipeline/` chains them: raw bytes → Huffman → RS → bit-depth writer → pixels. The header (always 2-bit depth) stores metadata for decoding. Existing image_processing package calls into the pipeline but retains pixel traversal and header I/O.

**Tech Stack:** Go 1.26, no new external dependencies. RS-ECC is implemented in-house over GF(256).

**Spec:** `docs/superpowers/specs/2026-03-20-pipeline-and-features-design.md`

---

## Chunk 1: Generalized Bit Manipulation

### Task 1: Add N-bit clear/set/get functions

**Files:**
- Modify: `go_steg/bit_manipulation/bit_manipulation.go`
- Modify: `go_steg/bit_manipulation/bit_manipulation_test.go`

- [ ] **Step 1: Write failing tests for ClearLastNBits**

Add to `go_steg/bit_manipulation/bit_manipulation_test.go`:

```go
func TestClearLastNBits(t *testing.T) {
	tests := []struct {
		name string
		b    byte
		n    int
		want byte
	}{
		{"clear 1 bit from 255", 255, 1, 254},   // 1111_1111 -> 1111_1110
		{"clear 2 bits from 255", 255, 2, 252},   // 1111_1111 -> 1111_1100
		{"clear 3 bits from 255", 255, 3, 248},   // 1111_1111 -> 1111_1000
		{"clear 4 bits from 255", 255, 4, 240},   // 1111_1111 -> 1111_0000
		{"clear 2 bits from 0", 0, 2, 0},
		{"clear 2 bits from 66", 66, 2, 64},      // same as existing clearLastTwoBits
		{"clear 1 bit from 1", 1, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClearLastNBits(tt.b, tt.n); got != tt.want {
				t.Errorf("ClearLastNBits(%d, %d) = %d, want %d", tt.b, tt.n, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go_steg/bit_manipulation/ -run TestClearLastNBits -v`
Expected: FAIL — `ClearLastNBits` not defined.

- [ ] **Step 3: Implement ClearLastNBits**

Add to `go_steg/bit_manipulation/bit_manipulation.go`:

```go
// ClearLastNBits clears the last n bits of the byte.
// For n=2, this is equivalent to clearLastTwoBits (mask 1111_1100).
// For n=3, mask is 1111_1000, etc.
func ClearLastNBits(b byte, n int) byte {
	mask := byte(0xFF) << n
	return b & mask
}
```

Update `clearLastTwoBits` to be a wrapper:

```go
func clearLastTwoBits(b byte) byte {
	return ClearLastNBits(b, 2)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./go_steg/bit_manipulation/ -run TestClearLastNBits -v`
Expected: PASS

- [ ] **Step 5: Write failing tests for SetLastNBits and GetLastNBits**

```go
func TestSetLastNBits(t *testing.T) {
	tests := []struct {
		name       string
		b          byte
		valueToSet byte
		n          int
		want       byte
	}{
		{"set 2 bits: 0 with 3", 0, 3, 2, 3},
		{"set 2 bits: 255 with 2", 255, 2, 2, 254},       // same as existing
		{"set 1 bit: 0 with 1", 0, 1, 1, 1},
		{"set 1 bit: 255 with 0", 255, 0, 1, 254},
		{"set 3 bits: 0 with 7", 0, 7, 3, 7},
		{"set 3 bits: 255 with 5", 255, 5, 3, 253},       // 1111_1000 | 101 = 1111_1101
		{"set 4 bits: 0 with 15", 0, 15, 4, 15},
		{"set 4 bits: 255 with 10", 255, 10, 4, 250},     // 1111_0000 | 1010 = 1111_1010
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetLastNBits(tt.b, tt.valueToSet, tt.n); got != tt.want {
				t.Errorf("SetLastNBits(%d, %d, %d) = %d, want %d", tt.b, tt.valueToSet, tt.n, got, tt.want)
			}
		})
	}
}

func TestGetLastNBits(t *testing.T) {
	tests := []struct {
		name string
		b    byte
		n    int
		want byte
	}{
		{"get 2 bits from 255", 255, 2, 3},
		{"get 2 bits from 0", 0, 2, 0},
		{"get 1 bit from 255", 255, 1, 1},
		{"get 1 bit from 0", 0, 1, 0},
		{"get 3 bits from 255", 255, 3, 7},
		{"get 3 bits from 5", 5, 3, 5},
		{"get 4 bits from 255", 255, 4, 15},
		{"get 4 bits from 170", 170, 4, 10},  // 1010_1010 -> 1010
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLastNBits(tt.b, tt.n); got != tt.want {
				t.Errorf("GetLastNBits(%d, %d) = %d, want %d", tt.b, tt.n, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./go_steg/bit_manipulation/ -run "TestSetLastNBits|TestGetLastNBits" -v`
Expected: FAIL

- [ ] **Step 7: Implement SetLastNBits and GetLastNBits**

```go
// SetLastNBits clears the last n bits of b, then sets them to valueToSet.
func SetLastNBits(b byte, valueToSet byte, n int) byte {
	return ClearLastNBits(b, n) | valueToSet
}

// GetLastNBits returns the last n bits of b.
func GetLastNBits(b byte, n int) byte {
	mask := byte((1 << n) - 1)
	return b & mask
}
```

Update `SetLastTwoBits` and `GetLastTwoBits` to be wrappers:

```go
func SetLastTwoBits(b byte, valueToSet byte) byte {
	return SetLastNBits(b, valueToSet, 2)
}

func GetLastTwoBits(b byte) byte {
	return GetLastNBits(b, 2)
}
```

- [ ] **Step 8: Run all bit_manipulation tests to verify nothing broke**

Run: `go test ./go_steg/bit_manipulation/ -v`
Expected: All PASS

- [ ] **Step 9: Commit**

```bash
git add go_steg/bit_manipulation/
git commit -m "feat: add generalized N-bit clear/set/get functions"
```

### Task 2: Add SplitByte and ConstructByte for variable bit depth

**Files:**
- Modify: `go_steg/bit_manipulation/bit_manipulation.go`
- Modify: `go_steg/bit_manipulation/bit_manipulation_test.go`

- [ ] **Step 1: Write failing tests for SplitByte**

```go
func TestSplitByte(t *testing.T) {
	tests := []struct {
		name          string
		b             byte
		bitsPerChunk  int
		want          []byte
	}{
		// 1-bit: 8 chunks
		{"1-bit split of 0", 0, 1, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{"1-bit split of 255", 255, 1, []byte{1, 1, 1, 1, 1, 1, 1, 1}},
		{"1-bit split of 170", 170, 1, []byte{1, 0, 1, 0, 1, 0, 1, 0}},
		// 2-bit: 4 chunks (same as SplitByteIntoQuarters)
		{"2-bit split of 0", 0, 2, []byte{0, 0, 0, 0}},
		{"2-bit split of 255", 255, 2, []byte{3, 3, 3, 3}},
		{"2-bit split of 1", 1, 2, []byte{0, 0, 0, 1}},
		// 3-bit: 3 chunks (3+3+2, last padded on MSB)
		{"3-bit split of 0", 0, 3, []byte{0, 0, 0}},
		{"3-bit split of 255", 255, 3, []byte{7, 7, 3}},  // 111_111_11 -> [111, 111, 11(0-padded)=011]
		{"3-bit split of 170", 170, 3, []byte{5, 2, 2}},  // 101_010_10 -> [101, 010, 10->010]
		// 4-bit: 2 chunks
		{"4-bit split of 0", 0, 4, []byte{0, 0}},
		{"4-bit split of 255", 255, 4, []byte{15, 15}},
		{"4-bit split of 170", 170, 4, []byte{10, 10}},   // 1010_1010
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go_steg/bit_manipulation/ -run TestSplitByte -v`
Expected: FAIL

- [ ] **Step 3: Implement SplitByte**

```go
// SplitByte splits a byte into chunks of bitsPerChunk bits each.
// For bitsPerChunk=2, this is equivalent to SplitByteIntoQuarters.
// For bitsPerChunk=3, produces 3 chunks: two 3-bit chunks from the top 6 bits,
// and one chunk with the remaining 2 bits (zero-padded on MSB side).
func SplitByte(b byte, bitsPerChunk int) []byte {
	totalBits := 8
	remaining := totalBits
	var chunks []byte

	for remaining > 0 {
		bitsToTake := bitsPerChunk
		if remaining < bitsToTake {
			bitsToTake = remaining
		}
		// Shift right to align the bits we want to the LSB position
		shift := remaining - bitsToTake
		chunk := (b >> shift) & byte((1<<bitsToTake)-1)
		chunks = append(chunks, chunk)
		remaining -= bitsToTake
	}

	return chunks
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./go_steg/bit_manipulation/ -run TestSplitByte -v`
Expected: PASS

- [ ] **Step 5: Write failing tests for ConstructByte**

```go
func TestConstructByte(t *testing.T) {
	tests := []struct {
		name         string
		chunks       []byte
		bitsPerChunk int
		want         byte
	}{
		// 1-bit
		{"1-bit construct 0", []byte{0, 0, 0, 0, 0, 0, 0, 0}, 1, 0},
		{"1-bit construct 255", []byte{1, 1, 1, 1, 1, 1, 1, 1}, 1, 255},
		{"1-bit construct 170", []byte{1, 0, 1, 0, 1, 0, 1, 0}, 1, 170},
		// 2-bit
		{"2-bit construct 0", []byte{0, 0, 0, 0}, 2, 0},
		{"2-bit construct 255", []byte{3, 3, 3, 3}, 2, 255},
		{"2-bit construct 65", []byte{1, 0, 0, 1}, 2, 65},
		// 3-bit
		{"3-bit construct 0", []byte{0, 0, 0}, 3, 0},
		{"3-bit construct 255", []byte{7, 7, 3}, 3, 255},
		{"3-bit construct 170", []byte{5, 2, 2}, 3, 170},
		// 4-bit
		{"4-bit construct 0", []byte{0, 0}, 4, 0},
		{"4-bit construct 255", []byte{15, 15}, 4, 255},
		{"4-bit construct 170", []byte{10, 10}, 4, 170},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConstructByte(tt.chunks, tt.bitsPerChunk); got != tt.want {
				t.Errorf("ConstructByte(%v, %d) = %d, want %d", tt.chunks, tt.bitsPerChunk, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./go_steg/bit_manipulation/ -run TestConstructByte -v`
Expected: FAIL

- [ ] **Step 7: Implement ConstructByte**

```go
// ConstructByte reconstructs a byte from chunks of bitsPerChunk bits each.
// For bitsPerChunk=3, the last chunk has only 2 real bits.
func ConstructByte(chunks []byte, bitsPerChunk int) byte {
	var result byte
	remaining := 8

	for _, chunk := range chunks {
		bitsToPlace := bitsPerChunk
		if remaining < bitsToPlace {
			bitsToPlace = remaining
		}
		result = (result << bitsToPlace) | chunk
		remaining -= bitsToPlace
	}

	return result
}
```

- [ ] **Step 8: Write roundtrip test**

```go
func TestSplitByteConstructByteRoundtrip(t *testing.T) {
	for depth := 1; depth <= 4; depth++ {
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
```

- [ ] **Step 9: Run all bit_manipulation tests**

Run: `go test ./go_steg/bit_manipulation/ -v`
Expected: All PASS

- [ ] **Step 10: Commit**

```bash
git add go_steg/bit_manipulation/
git commit -m "feat: add variable bit depth SplitByte and ConstructByte"
```

### Task 3: Update ReturnMaskDifference for variable bit depth

**Files:**
- Modify: `go_steg/bit_manipulation/bit_manipulation.go`
- Modify: `go_steg/bit_manipulation/bit_manipulation_test.go`

- [ ] **Step 1: Write failing test for ReturnMaskDifferenceN**

```go
func TestReturnMaskDifferenceN(t *testing.T) {
	// At depth=2, should match existing ReturnMaskDifference behavior
	tests := []struct {
		name        string
		maskInt     int32
		multiplier  int32
		firstIndex  int16
		secondIndex int16
		colorInt    uint8
		bitDepth    int
		want        bool
	}{
		{
			name: "depth 2: same as original false case",
			maskInt: 1, multiplier: 1, firstIndex: 0, secondIndex: 1,
			colorInt: 1, bitDepth: 2, want: false,
		},
		{
			name: "depth 2: same as original true case",
			maskInt: 8, multiplier: 1, firstIndex: 28, secondIndex: 27,
			colorInt: 16, bitDepth: 2, want: true,
		},
		{
			name: "depth 1: clears only 1 bit",
			maskInt: 8, multiplier: 1, firstIndex: 28, secondIndex: 27,
			colorInt: 17, bitDepth: 1, want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReturnMaskDifferenceN(tt.maskInt, tt.multiplier, tt.firstIndex, tt.secondIndex, tt.colorInt, tt.bitDepth)
			if got != tt.want {
				t.Errorf("ReturnMaskDifferenceN() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go_steg/bit_manipulation/ -run TestReturnMaskDifferenceN -v`
Expected: FAIL

- [ ] **Step 3: Implement ReturnMaskDifferenceN**

Add a new function that takes `bitDepth int` and uses `ClearLastNBits` instead of `clearLastTwoBits`. Copy the logic from `ReturnMaskDifference` but replace line 94's `clearLastTwoBits(colorInt)` with `ClearLastNBits(colorInt, bitDepth)`. Then update `ReturnMaskDifference` to call `ReturnMaskDifferenceN(..., 2)`.

- [ ] **Step 4: Run all bit_manipulation tests**

Run: `go test ./go_steg/bit_manipulation/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add go_steg/bit_manipulation/
git commit -m "feat: add ReturnMaskDifferenceN with variable bit depth"
```

---

## Chunk 2: Huffman Encode/Decode

### Task 4: Add password-derived tree generation

**Files:**
- Create: `go_steg/huffman/huffman_codec.go`
- Create: `go_steg/huffman/huffman_codec_test.go`

- [ ] **Step 1: Write failing test for GenerateTreeFromPassword**

In `go_steg/huffman/huffman_codec_test.go`:

```go
package huffman

import "testing"

func TestGenerateTreeFromPassword(t *testing.T) {
	// Same password must produce same tree
	tree1, leaves1 := GenerateTreeFromPassword("testPassword")
	tree2, leaves2 := GenerateTreeFromPassword("testPassword")

	if tree1 == nil || tree2 == nil {
		t.Fatal("tree should not be nil")
	}
	if len(leaves1) != 256 || len(leaves2) != 256 {
		t.Fatalf("expected 256 leaves, got %d and %d", len(leaves1), len(leaves2))
	}

	// Verify determinism: same codes for same password
	for i := 0; i < 256; i++ {
		code1, bits1 := leaves1[i].ReturnCode()
		code2, bits2 := leaves2[i].ReturnCode()
		if code1 != code2 || bits1 != bits2 {
			t.Errorf("byte %d: codes differ for same password: (%d,%d) vs (%d,%d)",
				i, code1, bits1, code2, bits2)
		}
	}

	// Different password must produce different tree
	_, leaves3 := GenerateTreeFromPassword("differentPassword")
	differences := 0
	for i := 0; i < 256; i++ {
		code1, bits1 := leaves1[i].ReturnCode()
		code3, bits3 := leaves3[i].ReturnCode()
		if code1 != code3 || bits1 != bits3 {
			differences++
		}
	}
	if differences == 0 {
		t.Error("different passwords produced identical trees")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go_steg/huffman/ -run TestGenerateTreeFromPassword -v`
Expected: FAIL

- [ ] **Step 3: Implement GenerateTreeFromPassword**

In `go_steg/huffman/huffman_codec.go`:

```go
package huffman

import (
	"crypto/sha256"
	"encoding/binary"
	mathrand "math/rand"
)

// GenerateTreeFromPassword creates a deterministic Huffman tree from a password.
// Returns the root node and a slice of 256 leaf nodes indexed by byte value.
func GenerateTreeFromPassword(password string) (*Node, [256]*Node) {
	hash := sha256.Sum256([]byte(password))
	seed := binary.BigEndian.Uint64(hash[:8])
	rng := mathrand.New(mathrand.NewSource(int64(seed)))

	var leaves [256]*Node
	nodeSlice := make([]*Node, 256)
	for i := 0; i < 256; i++ {
		freq := rng.Intn(1000) + 1 // 1-1000, avoid zero
		leaves[i] = &Node{Count: freq, Value: int32(i)}
		nodeSlice[i] = leaves[i]
	}

	root := BuildTree(nodeSlice)
	return root, leaves
}
```

Note: `BuildTree` sorts the slice and builds the tree, returning the root node. The `leaves` array still points to the original leaf nodes, which now have Parent pointers set by the tree building process. We must use the return value of `BuildTree` (not `nodeSlice[0]`) because `BuildPreSortedTree` shrinks its local slice copy — the caller's slice may not have the root at index 0.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./go_steg/huffman/ -run TestGenerateTreeFromPassword -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go_steg/huffman/
git commit -m "feat: add password-derived Huffman tree generation"
```

### Task 5: Implement HuffmanEncode and HuffmanDecode

**Files:**
- Modify: `go_steg/huffman/huffman_codec.go`
- Modify: `go_steg/huffman/huffman_codec_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestHuffmanEncodeDecodeRoundtrip(t *testing.T) {
	password := "testPassword"

	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single byte", []byte{42}},
		{"hello", []byte("hello world")},
		{"all byte values", func() []byte {
			b := make([]byte, 256)
			for i := range b { b[i] = byte(i) }
			return b
		}()},
		{"repeated bytes", make([]byte, 1000)}, // all zeros
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := HuffmanEncode(tt.data, password)
			decoded, err := HuffmanDecode(encoded, password)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if len(tt.data) == 0 && len(decoded) == 0 {
				return // both empty is fine
			}
			if !reflect.DeepEqual(decoded, tt.data) {
				t.Errorf("roundtrip failed: got %v, want %v", decoded, tt.data)
			}
		})
	}
}

func TestHuffmanDecodeWrongPassword(t *testing.T) {
	data := []byte("secret message")
	encoded := HuffmanEncode(data, "rightPassword")
	decoded, err := HuffmanDecode(encoded, "wrongPassword")
	if err == nil && reflect.DeepEqual(decoded, data) {
		t.Error("expected different output or error with wrong password")
	}
}

func TestHuffmanDecodeTruncated(t *testing.T) {
	data := []byte("hello world")
	encoded := HuffmanEncode(data, "password")
	// Truncate the encoded data
	_, err := HuffmanDecode(encoded[:len(encoded)/2], "password")
	if err == nil {
		t.Error("expected error for truncated data")
	}
}
```

Add `"reflect"` to the imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./go_steg/huffman/ -run "TestHuffmanEncode|TestHuffmanDecode" -v`
Expected: FAIL

- [ ] **Step 3: Implement HuffmanEncode**

```go
// HuffmanEncode encodes data using a password-derived Huffman tree.
// Format: [4-byte LE original length][packed huffman bits]
func HuffmanEncode(data []byte, password string) []byte {
	if len(data) == 0 {
		return []byte{}
	}

	_, leaves := GenerateTreeFromPassword(password)

	// Build code lookup table: byte value -> (code uint64, bits byte)
	type codeEntry struct {
		code uint64
		bits byte
	}
	var codeTable [256]codeEntry
	for i := 0; i < 256; i++ {
		code, bits := leaves[i].ReturnCode()
		codeTable[i] = codeEntry{code, bits}
	}

	// Prepend 4-byte LE original length
	result := make([]byte, 4)
	binary.LittleEndian.PutUint32(result, uint32(len(data)))

	// Pack bits into bytes.
	// IMPORTANT: ReturnCode() returns codes in LSB-first order (bit 0 is closest to leaf,
	// highest bit is root decision). We must emit bits from highest to lowest so that
	// the decoder (which walks root-to-leaf) reads them in the correct order.
	// ReturnCode() already stores the root decision in the highest bit position (bits-1),
	// so we iterate from bits-1 down to 0.
	var currentByte byte
	var bitsInCurrent byte

	for _, b := range data {
		entry := codeTable[b]
		// Emit from MSB (root decision) to LSB (leaf decision).
		// ReturnCode() stores: bit position (bits-1) = first decision from root.
		for i := int(entry.bits) - 1; i >= 0; i-- {
			bit := byte((entry.code >> i) & 1)
			currentByte = (currentByte << 1) | bit
			bitsInCurrent++
			if bitsInCurrent == 8 {
				result = append(result, currentByte)
				currentByte = 0
				bitsInCurrent = 0
			}
		}
	}

	// Flush remaining bits
	if bitsInCurrent > 0 {
		currentByte <<= (8 - bitsInCurrent)
		result = append(result, currentByte)
	}

	return result
}
```

- [ ] **Step 4: Implement HuffmanDecode**

```go
// HuffmanDecode decodes data encoded by HuffmanEncode.
func HuffmanDecode(data []byte, password string) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("huffman: data too short, need at least 4 bytes for length prefix")
	}

	originalLen := binary.LittleEndian.Uint32(data[:4])
	bitStream := data[4:]

	root, _ := GenerateTreeFromPassword(password)

	result := make([]byte, 0, originalLen)
	node := root

	for _, b := range bitStream {
		for bitPos := 7; bitPos >= 0; bitPos-- {
			bit := (b >> bitPos) & 1
			if bit == 0 {
				node = node.Left
			} else {
				node = node.Right
			}
			if node == nil {
				return nil, fmt.Errorf("huffman: invalid bit sequence, reached nil node")
			}
			// Leaf node: no children
			if node.Left == nil && node.Right == nil {
				result = append(result, byte(node.Value))
				node = root
				if uint32(len(result)) == originalLen {
					return result, nil
				}
			}
		}
	}

	if uint32(len(result)) < originalLen {
		return nil, fmt.Errorf("huffman: decoded %d bytes but expected %d", len(result), originalLen)
	}

	return result, nil
}
```

Add `"encoding/binary"` and `"fmt"` to imports.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./go_steg/huffman/ -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add go_steg/huffman/
git commit -m "feat: implement HuffmanEncode and HuffmanDecode"
```

---

## Chunk 3: Reed-Solomon Error Correction

### Task 6: Implement GF(256) arithmetic

**Files:**
- Create: `go_steg/reed_solomon/gf256.go`
- Create: `go_steg/reed_solomon/gf256_test.go`

This implements Galois Field GF(2^8) arithmetic using the standard irreducible polynomial 0x11D (x^8 + x^4 + x^3 + x^2 + 1), which is used by QR codes and many RS implementations.

- [ ] **Step 1: Write failing tests for GF(256) operations**

In `go_steg/reed_solomon/gf256_test.go`:

```go
package reed_solomon

import "testing"

func TestGF256Multiply(t *testing.T) {
	initTables()
	tests := []struct {
		a, b, want byte
	}{
		{0, 0, 0},
		{1, 1, 1},
		{2, 3, 6},       // basic
		{0, 255, 0},     // multiply by zero
		{1, 255, 255},   // multiply by one
		{2, 128, 27},    // overflow case using polynomial reduction
	}
	for _, tt := range tests {
		got := gfMul(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("gfMul(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestGF256Inverse(t *testing.T) {
	initTables()
	// a * inverse(a) == 1 for all nonzero a
	for a := 1; a < 256; a++ {
		inv := gfInv(byte(a))
		product := gfMul(byte(a), inv)
		if product != 1 {
			t.Errorf("gfMul(%d, gfInv(%d)) = %d, want 1", a, a, product)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go_steg/reed_solomon/ -run "TestGF256" -v`
Expected: FAIL

- [ ] **Step 3: Implement GF(256) with log/exp tables**

In `go_steg/reed_solomon/gf256.go`:

```go
package reed_solomon

import "sync"

// GF(2^8) with irreducible polynomial x^8 + x^4 + x^3 + x^2 + 1 = 0x11D
const gfPoly = 0x11D

var (
	expTable [512]byte // exp[i] = alpha^i, doubled for convenience
	logTable [256]byte // log[i] = discrete log base alpha of i
	initOnce sync.Once
)

func initTables() {
	initOnce.Do(func() {
		x := 1
		for i := 0; i < 255; i++ {
			expTable[i] = byte(x)
			logTable[x] = byte(i)
			x <<= 1
			if x >= 256 {
				x ^= gfPoly
			}
		}
		// Double the table for modular reduction convenience
		for i := 255; i < 512; i++ {
			expTable[i] = expTable[i-255]
		}
	})
}

func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return expTable[int(logTable[a])+int(logTable[b])]
}

func gfInv(a byte) byte {
	if a == 0 {
		return 0
	}
	return expTable[255-int(logTable[a])]
}

func gfDiv(a, b byte) byte {
	if b == 0 {
		panic("reed_solomon: division by zero in GF(256)")
	}
	if a == 0 {
		return 0
	}
	return expTable[(int(logTable[a])-int(logTable[b])+255)%255]
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./go_steg/reed_solomon/ -run "TestGF256" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go_steg/reed_solomon/
git commit -m "feat: implement GF(256) arithmetic for Reed-Solomon"
```

### Task 7: Implement RS encoder (polynomial evaluation)

**Files:**
- Create: `go_steg/reed_solomon/encoder.go`
- Modify: `go_steg/reed_solomon/gf256_test.go` (or create `encoder_test.go`)

- [ ] **Step 1: Write failing test for RS encoding of a single block**

In `go_steg/reed_solomon/encoder_test.go`:

```go
package reed_solomon

import "testing"

func TestEncodeBlock(t *testing.T) {
	initTables()
	// Encode a block, then verify that evaluating the codeword polynomial
	// at the roots of the generator produces zeros
	data := make([]byte, 223)
	for i := range data {
		data[i] = byte(i % 256)
	}

	parity := encodeBlock(data, 32)
	if len(parity) != 32 {
		t.Fatalf("expected 32 parity bytes, got %d", len(parity))
	}

	// Codeword = data || parity should have zero syndromes
	codeword := append(append([]byte{}, data...), parity...)
	syndromes := computeSyndromes(codeword, 32)
	for i, s := range syndromes {
		if s != 0 {
			t.Errorf("syndrome[%d] = %d, want 0", i, s)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go_steg/reed_solomon/ -run TestEncodeBlock -v`
Expected: FAIL

- [ ] **Step 3: Implement encodeBlock and computeSyndromes**

In `go_steg/reed_solomon/encoder.go`:

Implement the RS encoder using the generator polynomial approach:
1. Build generator polynomial `g(x) = (x - alpha^0)(x - alpha^1)...(x - alpha^(2t-1))` where `2t` = number of parity bytes.
2. Encode by computing `data(x) * x^(2t) mod g(x)` — the remainder is the parity.
3. `computeSyndromes` evaluates the codeword at `alpha^0` through `alpha^(2t-1)`.

This is standard textbook RS. The generator polynomial is computed once per block size (can cache for RS(255,223) and RS(255,191)).

```go
// generatorPoly computes the generator polynomial for nsym parity symbols.
func generatorPoly(nsym int) []byte {
	initTables()
	g := []byte{1}
	for i := 0; i < nsym; i++ {
		g = polyMul(g, []byte{1, expTable[i]})
	}
	return g
}

// polyMul multiplies two polynomials over GF(256).
func polyMul(a, b []byte) []byte {
	result := make([]byte, len(a)+len(b)-1)
	for i, av := range a {
		for j, bv := range b {
			result[i+j] ^= gfMul(av, bv)
		}
	}
	return result
}

// encodeBlock computes parity bytes for data using RS encoding.
func encodeBlock(data []byte, nsym int) []byte {
	gen := generatorPoly(nsym)
	// Pad data with nsym zeros
	padded := make([]byte, len(data)+nsym)
	copy(padded, data)
	// Polynomial long division
	for i := 0; i < len(data); i++ {
		coef := padded[i]
		if coef != 0 {
			for j := 1; j < len(gen); j++ {
				padded[i+j] ^= gfMul(gen[j], coef)
			}
		}
	}
	return padded[len(data):]
}

// computeSyndromes evaluates codeword at alpha^0..alpha^(nsym-1).
func computeSyndromes(codeword []byte, nsym int) []byte {
	syndromes := make([]byte, nsym)
	for i := 0; i < nsym; i++ {
		val := byte(0)
		for _, c := range codeword {
			val = gfMul(val, expTable[i]) ^ c
		}
		syndromes[i] = val
	}
	return syndromes
}
```

- [ ] **Step 4: Run test**

Run: `go test ./go_steg/reed_solomon/ -run TestEncodeBlock -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go_steg/reed_solomon/
git commit -m "feat: implement RS block encoder with generator polynomial"
```

### Task 8: Implement RS decoder (Berlekamp-Massey + Forney)

**Files:**
- Create: `go_steg/reed_solomon/decoder.go`
- Create: `go_steg/reed_solomon/decoder_test.go`

- [ ] **Step 1: Write failing tests for error correction**

In `go_steg/reed_solomon/decoder_test.go`:

```go
package reed_solomon

import (
	"reflect"
	"testing"
)

func TestDecodeBlockNoErrors(t *testing.T) {
	initTables()
	data := make([]byte, 223)
	for i := range data { data[i] = byte(i) }
	parity := encodeBlock(data, 32)
	codeword := append(append([]byte{}, data...), parity...)

	corrected, err := decodeBlock(codeword, 32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(corrected[:223], data) {
		t.Error("data corrupted after decode with no errors")
	}
}

func TestDecodeBlockWithErrors(t *testing.T) {
	initTables()
	data := make([]byte, 223)
	for i := range data { data[i] = byte(i) }
	parity := encodeBlock(data, 32)
	codeword := append(append([]byte{}, data...), parity...)

	// Introduce up to 16 byte errors (max correctable for 32 parity)
	corrupted := append([]byte{}, codeword...)
	for i := 0; i < 16; i++ {
		corrupted[i*10] ^= 0xFF
	}

	corrected, err := decodeBlock(corrupted, 32)
	if err != nil {
		t.Fatalf("failed to correct errors: %v", err)
	}
	if !reflect.DeepEqual(corrected[:223], data) {
		t.Error("data not correctly restored after error correction")
	}
}

func TestDecodeBlockTooManyErrors(t *testing.T) {
	initTables()
	data := make([]byte, 223)
	parity := encodeBlock(data, 32)
	codeword := append(append([]byte{}, data...), parity...)

	// Introduce 17 errors (beyond correctable limit of 16)
	corrupted := append([]byte{}, codeword...)
	for i := 0; i < 17; i++ {
		corrupted[i] ^= byte(i + 1)
	}

	_, err := decodeBlock(corrupted, 32)
	if err == nil {
		t.Error("expected error for too many corrupted bytes")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./go_steg/reed_solomon/ -run "TestDecodeBlock" -v`
Expected: FAIL

- [ ] **Step 3: Implement decodeBlock**

In `go_steg/reed_solomon/decoder.go`:

Implement the standard RS decoding pipeline:
1. Compute syndromes. If all zero, no errors.
2. Berlekamp-Massey algorithm to find the error locator polynomial.
3. Chien search to find error positions.
4. Forney algorithm to find error magnitudes.
5. Apply corrections.

This is ~100-150 lines of GF(256) polynomial operations. Follow standard textbook RS decoding (e.g., Wicker & Bhargava, or the Wikipedia RS article). Key functions:

```go
// decodeBlock attempts to correct errors in a codeword.
// Returns the corrected codeword or an error if too many errors.
func decodeBlock(codeword []byte, nsym int) ([]byte, error) {
	syndromes := computeSyndromes(codeword, nsym)

	// Check if all syndromes are zero (no errors)
	allZero := true
	for _, s := range syndromes {
		if s != 0 { allZero = false; break }
	}
	if allZero {
		return codeword, nil
	}

	// Berlekamp-Massey to find error locator polynomial
	errLoc, err := berlekampMassey(syndromes, nsym)
	if err != nil {
		return nil, err
	}

	// Chien search to find error positions
	errPos, err := chienSearch(errLoc, len(codeword))
	if err != nil {
		return nil, err
	}

	// Forney algorithm for error magnitudes
	corrected := make([]byte, len(codeword))
	copy(corrected, codeword)
	err = forney(corrected, syndromes, errLoc, errPos)
	if err != nil {
		return nil, err
	}

	return corrected, nil
}
```

Implement `berlekampMassey`, `chienSearch`, and `forney` as separate functions. Each is a well-documented algorithm with standard pseudocode available.

- [ ] **Step 4: Run tests**

Run: `go test ./go_steg/reed_solomon/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add go_steg/reed_solomon/
git commit -m "feat: implement RS decoder with Berlekamp-Massey and Forney"
```

### Task 9: Implement public RSEncode/RSDecode API

**Files:**
- Create: `go_steg/reed_solomon/rs.go`
- Create: `go_steg/reed_solomon/rs_test.go`

- [ ] **Step 1: Write failing tests for the public API**

In `go_steg/reed_solomon/rs_test.go`:

```go
package reed_solomon

import (
	"reflect"
	"testing"
)

func TestRSEncodeDecodeRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		level RedundancyLevel
	}{
		{"small standard", []byte("hello world"), Standard},
		{"small high", []byte("hello world"), High},
		{"exact block standard", make([]byte, 223), Standard},
		{"exact block high", make([]byte, 191), High},
		{"multi block standard", make([]byte, 500), Standard},
		{"multi block high", make([]byte, 500), High},
		{"large data", make([]byte, 2000), Standard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill with non-zero data
			for i := range tt.data { tt.data[i] = byte(i) }

			encoded, err := RSEncode(tt.data, tt.level)
			if err != nil {
				t.Fatalf("RSEncode error: %v", err)
			}

			decoded, err := RSDecode(encoded, tt.level)
			if err != nil {
				t.Fatalf("RSDecode error: %v", err)
			}

			if !reflect.DeepEqual(decoded, tt.data) {
				t.Errorf("roundtrip failed: decoded length %d, want %d", len(decoded), len(tt.data))
			}
		})
	}
}

func TestRSEncodeDecodeWithCorruption(t *testing.T) {
	data := make([]byte, 500)
	for i := range data { data[i] = byte(i) }

	encoded, _ := RSEncode(data, Standard)

	// Corrupt a few bytes in the first block (skip 8-byte prefix)
	for i := 8; i < 18; i++ {
		encoded[i] ^= 0xFF
	}

	decoded, err := RSDecode(encoded, Standard)
	if err != nil {
		t.Fatalf("RSDecode should correct minor corruption: %v", err)
	}
	if !reflect.DeepEqual(decoded, data) {
		t.Error("decoded data does not match original after correction")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./go_steg/reed_solomon/ -run "TestRSEncode" -v`
Expected: FAIL

- [ ] **Step 3: Implement RSEncode and RSDecode**

In `go_steg/reed_solomon/rs.go`:

```go
package reed_solomon

import (
	"encoding/binary"
	"fmt"
)

type RedundancyLevel int

const (
	Standard RedundancyLevel = iota // RS(255,223): 32 parity, corrects 16 errors
	High                            // RS(255,191): 64 parity, corrects 32 errors
)

func paramsForLevel(level RedundancyLevel) (dataBytes, parityBytes int) {
	switch level {
	case High:
		return 191, 64
	default:
		return 223, 32
	}
}

// RSEncode encodes data with Reed-Solomon error correction.
// Format: [4-byte LE block count][4-byte LE original data length][blocks...]
// Each block is 255 bytes (data + parity).
func RSEncode(data []byte, level RedundancyLevel) ([]byte, error) {
	initTables()
	dataPerBlock, parityPerBlock := paramsForLevel(level)

	// Split into blocks
	var blocks [][]byte
	for i := 0; i < len(data); i += dataPerBlock {
		end := i + dataPerBlock
		if end > len(data) {
			end = len(data)
		}
		block := make([]byte, dataPerBlock)
		copy(block, data[i:end]) // zero-pads if last block is short
		blocks = append(blocks, block)
	}
	if len(blocks) == 0 {
		blocks = append(blocks, make([]byte, dataPerBlock))
	}

	// 8-byte prefix
	prefix := make([]byte, 8)
	binary.LittleEndian.PutUint32(prefix[0:4], uint32(len(blocks)))
	binary.LittleEndian.PutUint32(prefix[4:8], uint32(len(data)))

	result := append([]byte{}, prefix...)

	for _, block := range blocks {
		parity := encodeBlock(block, parityPerBlock)
		result = append(result, block...)
		result = append(result, parity...)
	}

	return result, nil
}

// RSDecode decodes RS-encoded data, correcting errors where possible.
// The level parameter must match the level used during encoding — it is read
// from the header's encoding flags by the caller.
func RSDecode(data []byte, level RedundancyLevel) ([]byte, error) {
	initTables()
	if len(data) < 8 {
		return nil, fmt.Errorf("reed_solomon: data too short for prefix")
	}

	blockCount := binary.LittleEndian.Uint32(data[0:4])
	originalLen := binary.LittleEndian.Uint32(data[4:8])

	payload := data[8:]
	blockSize := 255 // always 255 for RS(255,k)

	expectedLen := int(blockCount) * blockSize
	if len(payload) < expectedLen {
		return nil, fmt.Errorf("reed_solomon: payload too short: have %d, need %d", len(payload), expectedLen)
	}

	dataPerBlock, nsym := paramsForLevel(level)

	var result []byte
	for i := 0; i < int(blockCount); i++ {
		start := i * blockSize
		end := start + blockSize
		codeword := payload[start:end]

		corrected, err := decodeBlock(codeword, nsym)
		if err != nil {
			return nil, fmt.Errorf("reed_solomon: block %d: %w", i, err)
		}

		result = append(result, corrected[:dataPerBlock]...)
	}

	// Trim to original length
	if int(originalLen) < len(result) {
		result = result[:originalLen]
	}

	return result, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./go_steg/reed_solomon/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add go_steg/reed_solomon/
git commit -m "feat: implement public RSEncode/RSDecode API with block splitting"
```

---

## Chunk 4: Pipeline Orchestration & Header

### Task 10: Create pipeline package

**Files:**
- Create: `go_steg/pipeline/pipeline.go`
- Create: `go_steg/pipeline/pipeline_test.go`

- [ ] **Step 1: Write failing tests for pipeline encode/decode**

In `go_steg/pipeline/pipeline_test.go`:

```go
package pipeline

import (
	"reflect"
	"testing"

	"go-steg/go_steg/reed_solomon"
)

func TestPipelineEncodeDecodeRoundtrip(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		data   []byte
	}{
		{
			name: "no transforms",
			config: Config{Password: "test"},
			data: []byte("hello world"),
		},
		{
			name: "huffman only",
			config: Config{HuffmanEnabled: true, Password: "test"},
			data: []byte("hello world"),
		},
		{
			name: "RS only",
			config: Config{RSEnabled: true, RSLevel: reed_solomon.Standard, Password: "test"},
			data: []byte("hello world"),
		},
		{
			name: "huffman + RS",
			config: Config{
				HuffmanEnabled: true, RSEnabled: true,
				RSLevel: reed_solomon.Standard, Password: "test",
			},
			data: []byte("hello world"),
		},
		{
			name: "all byte values",
			config: Config{HuffmanEnabled: true, RSEnabled: true, RSLevel: reed_solomon.High, Password: "test"},
			data: func() []byte {
				b := make([]byte, 256)
				for i := range b { b[i] = byte(i) }
				return b
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := Encode(tt.data, tt.config)
			if err != nil {
				t.Fatalf("Encode error: %v", err)
			}
			decoded, err := Decode(encoded, tt.config)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}
			if !reflect.DeepEqual(decoded, tt.data) {
				t.Errorf("roundtrip failed")
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go_steg/pipeline/ -v`
Expected: FAIL

- [ ] **Step 3: Implement pipeline**

In `go_steg/pipeline/pipeline.go`:

```go
package pipeline

import (
	"go-steg/go_steg/huffman"
	"go-steg/go_steg/reed_solomon"
)

// Config holds the pipeline configuration for encode/decode.
type Config struct {
	BitDepth       int                        // 1-4, used by caller for pixel I/O
	HuffmanEnabled bool
	RSEnabled      bool
	RSLevel        reed_solomon.RedundancyLevel
	FileExtension  string
	Password       string
}

// Encode runs the encode pipeline: Huffman → RS.
// Returns transformed bytes ready to be written to pixels.
func Encode(data []byte, cfg Config) ([]byte, error) {
	result := data

	if cfg.HuffmanEnabled {
		result = huffman.HuffmanEncode(result, cfg.Password)
	}

	if cfg.RSEnabled {
		var err error
		result, err = reed_solomon.RSEncode(result, cfg.RSLevel)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Decode runs the decode pipeline: RS → Huffman (reverse order).
func Decode(data []byte, cfg Config) ([]byte, error) {
	result := data

	if cfg.RSEnabled {
		var err error
		result, err = reed_solomon.RSDecode(result, cfg.RSLevel)
		if err != nil {
			return nil, err
		}
	}

	if cfg.HuffmanEnabled {
		var err error
		result, err = huffman.HuffmanDecode(result, cfg.Password)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./go_steg/pipeline/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add go_steg/pipeline/
git commit -m "feat: implement pipeline package with Huffman and RS stages"
```

### Task 11: Update header constants and layout

**Files:**
- Modify: `go_steg/image_processing/constants.go`

- [ ] **Step 1: Update constants**

Replace the contents of `go_steg/image_processing/constants.go`:

```go
package image_processing

// Legacy header layout (pixels 0-12)
const photoIDHeaderReservedPixels = 8
const photoNumberHeaderReservedPixels = 1
const dataSizeHeaderReservedPixels = 4
const legacyTotalReservedPixels = photoIDHeaderReservedPixels + photoNumberHeaderReservedPixels + dataSizeHeaderReservedPixels // 13

// New header layout (pixels 13-33)
const versionMarkerPixels = 2         // pixels 13-14
const fileExtensionPixels = 11        // pixels 15-25
const encodingFlagsPixels = 1         // pixel 26
const checksumPixels = 2              // pixels 27-28
const byteCountModuloPixels = 2       // pixels 29-30
const reservedPixels = 3              // pixels 31-33

const totalReservedPixels = legacyTotalReservedPixels + versionMarkerPixels + fileExtensionPixels +
	encodingFlagsPixels + checksumPixels + byteCountModuloPixels + reservedPixels // 34

// Version marker magic pattern: 101010 110011 (across 2 pixels, 12 bits)
// Pixel 13 channels: R=10, G=10, B=10 -> quarters [2, 2, 2]
// Pixel 14 channels: R=11, G=00, B=11 -> quarters [3, 0, 3]
var versionMarkerBytes = [6]byte{2, 2, 2, 3, 0, 3}

// Image size constants
const instagramMaxImageWidth = 1080
const instagramMaxImageHeight = 1350
const instagramHalfMaxWidth = instagramMaxImageWidth / 2
const instagramHalfMaxHeight = instagramMaxImageHeight / 2

// Minimum carrier image height (must fit all header pixels)
const minCarrierHeight = totalReservedPixels
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Success (existing code uses `totalReservedPixels` which still exists)

- [ ] **Step 3: Commit**

```bash
git add go_steg/image_processing/constants.go
git commit -m "feat: update header constants for new 34-pixel layout"
```

### Task 12: Implement new header read/write

**Files:**
- Create: `go_steg/image_processing/header.go`
- Create: `go_steg/image_processing/header_test.go`

- [ ] **Step 1: Define HeaderInfo struct and write failing test**

In `go_steg/image_processing/header_test.go`:

```go
package image_processing

import (
	"image"
	"testing"

	"go-steg/go_steg/reed_solomon"
)

func TestHeaderRoundtrip(t *testing.T) {
	// Create a test RGBA image large enough for the header
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	info := HeaderInfo{
		PhotoID:        12345,
		PhotoNumber:    3,
		DataCount:      5000,
		IsNewFormat:    true,
		FileExtension:  "pdf",
		BitDepth:       3,
		HuffmanEnabled: true,
		RSEnabled:      true,
		RSLevel:        reed_solomon.High,
		Checksum:       0xABC,
		ByteCountMod:   1234,
	}

	writeHeader(img, info)
	got := readHeader(img)

	if got.PhotoID != info.PhotoID {
		t.Errorf("PhotoID: got %d, want %d", got.PhotoID, info.PhotoID)
	}
	if got.PhotoNumber != info.PhotoNumber {
		t.Errorf("PhotoNumber: got %d, want %d", got.PhotoNumber, info.PhotoNumber)
	}
	if got.DataCount != info.DataCount {
		t.Errorf("DataCount: got %d, want %d", got.DataCount, info.DataCount)
	}
	if !got.IsNewFormat {
		t.Error("expected IsNewFormat=true")
	}
	if got.FileExtension != info.FileExtension {
		t.Errorf("FileExtension: got %q, want %q", got.FileExtension, info.FileExtension)
	}
	if got.BitDepth != info.BitDepth {
		t.Errorf("BitDepth: got %d, want %d", got.BitDepth, info.BitDepth)
	}
	if got.HuffmanEnabled != info.HuffmanEnabled {
		t.Errorf("HuffmanEnabled: got %v, want %v", got.HuffmanEnabled, info.HuffmanEnabled)
	}
	if got.RSEnabled != info.RSEnabled {
		t.Errorf("RSEnabled: got %v, want %v", got.RSEnabled, info.RSEnabled)
	}
	if got.RSLevel != info.RSLevel {
		t.Errorf("RSLevel: got %v, want %v", got.RSLevel, info.RSLevel)
	}
	if got.Checksum != info.Checksum {
		t.Errorf("Checksum: got %d, want %d", got.Checksum, info.Checksum)
	}
	if got.ByteCountMod != info.ByteCountMod {
		t.Errorf("ByteCountMod: got %d, want %d", got.ByteCountMod, info.ByteCountMod)
	}
}

func TestHeaderLegacyDetection(t *testing.T) {
	// Image with no version marker should be detected as legacy
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	got := readHeader(img)
	if got.IsNewFormat {
		t.Error("expected legacy format detection for blank image")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./go_steg/image_processing/ -run "TestHeader" -v`
Expected: FAIL

- [ ] **Step 3: Implement HeaderInfo struct and read/write functions**

In `go_steg/image_processing/header.go`:

Create a `HeaderInfo` struct with all fields from the spec. Implement `writeHeader(img *image.RGBA, info HeaderInfo)` and `readHeader(img *image.RGBA) HeaderInfo`.

Key implementation details:
- Use `bit_manipulation.SetLastTwoBits(c.R, ...)` with `c.R` as the base (NOT `c.B` — fix the existing bug).
- Same pattern for G and B channels.
- Version marker: write `versionMarkerBytes` into pixels 13-14 (R, G, B channels).
- File extension: encode ASCII string into 2-bit quarters across pixels 15-25.
- Encoding flags pixel 26: pack bit depth (2 bits), huffman (1 bit), RS (1 bit), RS level (1 bit), unused (1 bit).
- Checksum pixels 27-28: store low 12 bits across 2 pixels.
- Byte count modulo pixels 29-30: store 12 bits across 2 pixels.
- Legacy detection: read pixels 13-14, compare to `versionMarkerBytes`. If mismatch, read bit depth from pixel 26 — if not valid (1-4), treat as legacy.

- [ ] **Step 4: Run tests**

Run: `go test ./go_steg/image_processing/ -run "TestHeader" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go_steg/image_processing/header.go go_steg/image_processing/header_test.go
git commit -m "feat: implement new header read/write with version marker and encoding flags"
```

---

## Chunk 5: Integration, CLI, and README

### Task 13: Refactor Encode/Decode to use pipeline and new header

**Files:**
- Modify: `go_steg/image_processing/encoding.go`
- Modify: `go_steg/image_processing/decoding.go`

This is the largest integration task. The changes:

- [ ] **Step 1: Remove old header functions and update Encode signature**

**IMPORTANT:** The old `extractDataCount`, `extractPhotoID`, `extractPhotoNumber`, and `setHeaderInformation` functions use `totalReservedPixels` as a loop bound. Since `totalReservedPixels` changed from 13 to 34, these functions will break. Replace all of them with the new `readHeader`/`writeHeader` from Task 12. Delete the old functions entirely.

Add a `pipeline.Config` parameter to `Encode`, `MultiCarrierEncode`, `EncodeByFileNames`, and `MultiCarrierEncodeByFileNames`. The config carries bit depth, huffman, RS, file extension, and password.

- [ ] **Step 2: Update encoding pipeline integration**

In `MultiCarrierEncode`:
1. Run `pipeline.Encode(dataBytes, config)` on the full data BEFORE splitting into chunks.
2. Compute CRC-16 checksum of the pipeline output's first 4 bytes.
3. Compute byte count modulo 4096 of the pipeline output.
4. Split pipeline output into chunks (existing logic).
5. For each chunk, call `Encode` which now uses `SplitByte`/`SetLastNBits` with `config.BitDepth` instead of hardcoded 2-bit operations.
6. Write the new header via `writeHeader` instead of `setHeaderInformation`.

- [ ] **Step 3: Update the pixel traversal in Encode to use N-bit operations**

Replace:
- `setColorSegment` calls to use `SetLastNBits` with `config.BitDepth`
- `SplitByteIntoQuarters` → `SplitByte(b, config.BitDepth)` in `readData`
- Mask checks: `ReturnMaskDifference` → `ReturnMaskDifferenceN(..., config.BitDepth)`

- [ ] **Step 4: Update Decode function**

1. Read header via `readHeader`. If legacy format, use existing logic.
2. If new format, use `GetLastNBits` with the header's bit depth.
3. After extracting all bytes and reconstructing, run `pipeline.Decode(data, config)`.
4. Use the byte count modulo from the header to determine exact byte count for 3-bit depth.
5. Verify CRC-16 checksum. If mismatch, return "wrong password or corrupted data" error.

- [ ] **Step 5: Update MultiCarrierDecodeByFileNames to use header's file extension**

Change output filename from hardcoded `decoded_image-<time>.png` to `decoded_file-<time>.<ext>` using the file extension read from the header.

- [ ] **Step 6: Read embed file as raw bytes**

In `MultiCarrierEncodeByFileNames`, read the embed file with `os.ReadFile` (raw bytes) instead of going through `readData`'s image decoder. The `readData` goroutine-based channel approach remains for feeding bytes to the pixel writer, but it receives raw bytes, not image-decoded bytes.

- [ ] **Step 7: Add capacity validation**

After running the pipeline, before writing pixels, check: `pipelineOutputLen * chunksPerByte(bitDepth)` must not exceed available channel slots in the carrier (accounting for mask if enabled). Return `ErrDataTooLarge` if exceeded.

- [ ] **Step 8: Run existing tests**

Run: `go test ./go_steg/image_processing/ -v`
Expected: Existing tests should still pass (they use bit depth 2, no Huffman, no RS — legacy behavior).

- [ ] **Step 9: Commit**

```bash
git add go_steg/image_processing/
git commit -m "feat: integrate pipeline and new header into encode/decode"
```

### Task 14: Update CLI commands

**Files:**
- Modify: `cli/cmd/encode.go`
- Modify: `cli/cmd/decode.go`

- [ ] **Step 1: Add new flags to encode command**

In `cli/cmd/encode.go`, add to `init()`:

```go
var bitDepth int
var huffmanEnabled bool
var rsEnabled bool
var rsLevel string

encodeCmd.PersistentFlags().IntVarP(&bitDepth, "bitDepth", "b", 2,
	"Bits per channel (1-4). Higher values increase capacity but reduce stealth")
encodeCmd.PersistentFlags().BoolVar(&huffmanEnabled, "huffman", false,
	"Enable Huffman compression (password-derived)")
encodeCmd.PersistentFlags().BoolVar(&rsEnabled, "rs", false,
	"Enable Reed-Solomon error correction")
encodeCmd.PersistentFlags().StringVar(&rsLevel, "rsLevel", "standard",
	"RS redundancy level: 'standard' (~14%) or 'high' (~34%)")
```

- [ ] **Step 2: Update encode Run function to build PipelineConfig**

In the `Run` function of `encodeCmd`, construct a `pipeline.Config` from the flags and pass it through to `EncodeByFileNames`.

- [ ] **Step 3: Update `-e` flag description**

Change `"The name of the file to embed into the carrier file(s)"` — no reference to "photo".

- [ ] **Step 4: Update decode command**

No new flags needed. Pass password through to the decode functions. The header provides all other config.

- [ ] **Step 5: Verify CLI builds**

Run: `go build -o go-steg .`
Expected: Success

- [ ] **Step 6: Manual smoke test**

```bash
# Encode a text file with default settings
echo "test data" > /tmp/test.txt
./go-steg encode -e /tmp/test.txt -c <carrier.png> -p testpass -o /tmp -u

# Decode
./go-steg decode -c /tmp/<embedded.png> -p testpass -o /tmp
# Should produce /tmp/decoded_file-<time>.txt containing "test data"
```

- [ ] **Step 7: Commit**

```bash
git add cli/cmd/encode.go cli/cmd/decode.go
git commit -m "feat: add bitDepth, huffman, rs, rsLevel CLI flags"
```

### Task 15: Update README with Reed-Solomon documentation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add Reed-Solomon section**

Add a section after the existing "Resources" section explaining:

1. **What is Reed-Solomon error correction** — polynomial codes over finite fields (Galois Fields), originally designed for deep-space communication. Each block of data is treated as coefficients of a polynomial, and parity bytes are computed by evaluating the polynomial at specific points in GF(256).

2. **How it works in go-steg** — data is split into blocks (223 bytes for standard, 191 for high), parity bytes are appended. On decode, syndromes are computed; if non-zero, the Berlekamp-Massey algorithm finds the error locator polynomial, Chien search finds error positions, and the Forney algorithm computes error magnitudes.

3. **What it can and cannot protect against:**
   - PNG re-save with color space rounding: YES
   - Minor bit-level corruption: YES (up to 16 byte errors per 255-byte block at standard, 32 at high)
   - JPEG recompression: NO (DCT quantization destroys LSBs entirely)
   - Image cropping affecting header area: NO (header is not RS-protected)

4. **Choosing a redundancy level:**
   - Standard (~14% overhead): good default, handles minor corruption
   - High (~34% overhead): when the carrier may undergo multiple re-saves or slight processing

- [ ] **Step 2: Update Getting Started section**

Update the CLI examples to show new flags:

```bash
# Basic encode (backwards compatible)
go-steg encode -e embed.png -c carrier.png -p password -o output/ -u

# Encode with all features
go-steg encode -e document.pdf -c carrier.png -p password -o output/ -u -b 3 --huffman --rs --rsLevel high

# Decode (automatically detects features from header)
go-steg decode -c output/carrier-0-embedded.png -p password -o output/
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add Reed-Solomon explanation and updated CLI examples"
```

### Task 16: End-to-end integration tests

**Files:**
- Create: `go_steg/image_processing/integration_test.go`

- [ ] **Step 1: Write integration tests**

Test the full encode → decode cycle with various configurations:

```go
func TestEncodeDecodeNewFormat(t *testing.T) {
	// Tests: text file, huffman, RS, bit depths 1-4
	// For each: encode into a test carrier, decode, verify output matches input
}

func TestEncodeDecodeLegacyCompatibility(t *testing.T) {
	// Test: encode with default settings (no new features)
	// Verify the output can be decoded the same as before
}

func TestEncodeDecodeMultiCarrier(t *testing.T) {
	// Test: encode with pipeline enabled, split across 2 carriers, decode
}
```

These tests need test fixture images. Use programmatically generated solid-color images (e.g., `image.NewRGBA(image.Rect(0, 0, 200, 200))`) to avoid depending on external files.

- [ ] **Step 2: Run tests**

Run: `go test ./go_steg/image_processing/ -run "TestEncodeDecode" -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add go_steg/image_processing/integration_test.go
git commit -m "test: add end-to-end integration tests for pipeline features"
```

- [ ] **Step 4: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS

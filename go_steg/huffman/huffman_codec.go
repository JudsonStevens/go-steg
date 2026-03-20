package huffman

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	mathrand "math/rand"
)

// GenerateTreeFromPassword creates a deterministic Huffman tree from a password.
// Returns the root node and a fixed-size array of 256 leaf nodes indexed by byte value.
func GenerateTreeFromPassword(password string) (*Node, [256]*Node) {
	hash := sha256.Sum256([]byte(password))
	seed := binary.BigEndian.Uint64(hash[:8])
	rng := mathrand.New(mathrand.NewSource(int64(seed)))

	var leaves [256]*Node
	nodeSlice := make([]*Node, 256)
	for i := 0; i < 256; i++ {
		freq := rng.Intn(1000) + 1
		leaves[i] = &Node{Count: freq, Value: int32(i)}
		nodeSlice[i] = leaves[i]
	}

	root := BuildTree(nodeSlice)
	return root, leaves
}

// HuffmanEncode encodes data using a password-derived Huffman tree.
// Format: [4-byte LE original length][packed huffman bits]
func HuffmanEncode(data []byte, password string) []byte {
	if len(data) == 0 {
		return []byte{}
	}

	_, leaves := GenerateTreeFromPassword(password)

	type codeEntry struct {
		code uint64
		bits byte
	}
	var codeTable [256]codeEntry
	for i := 0; i < 256; i++ {
		code, bits := leaves[i].ReturnCode()
		codeTable[i] = codeEntry{code, bits}
	}

	result := make([]byte, 4)
	binary.LittleEndian.PutUint32(result, uint32(len(data)))

	var currentByte byte
	var bitsInCurrent byte

	for _, b := range data {
		entry := codeTable[b]
		// Emit from MSB (root decision at bits-1) to LSB (leaf decision at 0)
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

	if bitsInCurrent > 0 {
		currentByte <<= (8 - bitsInCurrent)
		result = append(result, currentByte)
	}

	return result
}

// HuffmanDecode decodes data encoded by HuffmanEncode.
func HuffmanDecode(data []byte, password string) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("huffman: data too short for length prefix")
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
				return nil, fmt.Errorf("huffman: invalid bit sequence")
			}
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

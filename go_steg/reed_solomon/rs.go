package reed_solomon

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// RedundancyLevel controls the amount of error correction applied.
type RedundancyLevel int

const (
	// Standard uses RS(255,223): 32 parity bytes, corrects up to 16 byte errors per block.
	Standard RedundancyLevel = iota
	// High uses RS(255,191): 64 parity bytes, corrects up to 32 byte errors per block.
	High
)

const prefixLen = 8 // uint32 block count + uint32 original data length

func paramsForLevel(level RedundancyLevel) (dataBytes, parityBytes int) {
	switch level {
	case High:
		return 191, 64
	default:
		return 223, 32
	}
}

// RSEncode encodes data with Reed-Solomon error correction.
// The output format is:
//   - 8-byte prefix: uint32 block count (LE) + uint32 original data length (LE)
//   - Each block: 255 bytes (data + parity)
func RSEncode(data []byte, level RedundancyLevel) ([]byte, error) {
	initTables()
	dataPerBlock, parityPerBlock := paramsForLevel(level)

	// Calculate number of blocks
	numBlocks := len(data) / dataPerBlock
	if len(data)%dataPerBlock != 0 || len(data) == 0 {
		numBlocks++
	}

	// Allocate output: prefix + numBlocks * 255
	output := make([]byte, prefixLen+numBlocks*255)

	// Write prefix
	binary.LittleEndian.PutUint32(output[0:4], uint32(numBlocks))
	binary.LittleEndian.PutUint32(output[4:8], uint32(len(data)))

	// Encode each block
	for i := 0; i < numBlocks; i++ {
		start := i * dataPerBlock
		end := start + dataPerBlock
		if end > len(data) {
			end = len(data)
		}

		// Prepare block data (zero-padded to dataPerBlock)
		blockData := make([]byte, dataPerBlock)
		copy(blockData, data[start:end])

		// Compute parity
		parity := encodeBlock(blockData, parityPerBlock)

		// Write data + parity to output
		outStart := prefixLen + i*255
		copy(output[outStart:], blockData)
		copy(output[outStart+dataPerBlock:], parity)
	}

	return output, nil
}

// RSDecode decodes Reed-Solomon encoded data, correcting errors if possible.
func RSDecode(data []byte, level RedundancyLevel) ([]byte, error) {
	initTables()

	if len(data) < prefixLen {
		return nil, errors.New("reed_solomon: data too short for prefix")
	}

	dataPerBlock, parityPerBlock := paramsForLevel(level)
	_ = parityPerBlock // used implicitly via 255 - dataPerBlock

	// Read prefix
	numBlocks := int(binary.LittleEndian.Uint32(data[0:4]))
	origLen := int(binary.LittleEndian.Uint32(data[4:8]))

	expectedLen := prefixLen + numBlocks*255
	if len(data) < expectedLen {
		return nil, fmt.Errorf("reed_solomon: expected %d bytes, got %d", expectedLen, len(data))
	}

	// Decode each block
	result := make([]byte, 0, numBlocks*dataPerBlock)
	nsym := 255 - dataPerBlock

	for i := 0; i < numBlocks; i++ {
		blockStart := prefixLen + i*255
		codeword := data[blockStart : blockStart+255]

		decoded, err := decodeBlock(codeword, nsym)
		if err != nil {
			return nil, fmt.Errorf("reed_solomon: block %d: %w", i, err)
		}

		// Append only the data portion
		result = append(result, decoded[:dataPerBlock]...)
	}

	// Trim to original length
	if origLen > len(result) {
		return nil, fmt.Errorf("reed_solomon: original length %d exceeds decoded data %d", origLen, len(result))
	}

	return result[:origLen], nil
}

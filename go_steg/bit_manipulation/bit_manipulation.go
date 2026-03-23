package bit_manipulation

import (
	"encoding/binary"
)

// These constants are the max value at each quarter with just those as 1s
// Ex - 192 = 1100 0000, 48 = 0011 0000, 12 = 0000 1100, 3 = 0000 0011
const (
	firstQuarterOfByteMax  = 192
	secondQuarterOfByteMax = 48
	thirdQuarterOfByteMax  = 12
	fourthQuarterOfByteMax = 3
)

// SplitByteIntoQuarters returns a byte split into it's four quarters.
//
// Use the & (AND) operator to get just the bits we want and zero out the rest. That is, 1011 0011 & 1100 0000 (192)
// will return 1000 0000. Then we shift the bits to the right to move the bits we want to the end. Similarly,
// for the second quarter of the byte, we use 0011 0000 (48) to get the bits we want, then shift them to the right.
// That is, 1011 0011 & 0011 0000 (48) will return 0011 0000. Then we shift the bits to the right to move the bits
// to the end again. We do this for the third and fourth quarters of the byte as well. The purpose of this is to
// generate bytes that contain the data we want to store as the last two bits. So for the above example, we would end
// up with [0000 0010, 0000 0011, 0000 0000, 0000 0011]. The last two bits of those bytes are the data we want
// to store.
//
// Some reading on the bitwise AND operator (and others):
// - https://yourbasic.org/golang/operators/
// - https://yourbasic.org/golang/bitwise-operator-cheat-sheet/
func SplitByteIntoQuarters(b byte) [4]byte {
	return [4]byte{b & firstQuarterOfByteMax >> 6, b & secondQuarterOfByteMax >> 4, b & thirdQuarterOfByteMax >> 2, b & fourthQuarterOfByteMax}
}

// ClearLastNBits clears the last n bits of b.
func ClearLastNBits(b byte, n int) byte {
	return b & (byte(0xFF) << n)
}

// SetLastNBits clears the last n bits of b and then sets them to valueToSet.
func SetLastNBits(b byte, valueToSet byte, n int) byte {
	return ClearLastNBits(b, n) | valueToSet
}

// GetLastNBits returns the last n bits of b.
func GetLastNBits(b byte, n int) byte {
	return b & byte((1<<n)-1)
}

// SplitByte splits a byte into chunks of bitsPerChunk bits, MSB-first.
// For bitsPerChunk=3, produces 3 chunks (3+3+2 bits — last chunk has only 2 remaining bits).
func SplitByte(b byte, bitsPerChunk int) []byte {
	totalBits := 8
	remaining := totalBits
	var chunks []byte
	for remaining > 0 {
		bitsToTake := bitsPerChunk
		if remaining < bitsToTake {
			bitsToTake = remaining
		}
		shift := remaining - bitsToTake
		chunk := (b >> shift) & byte((1<<bitsToTake)-1)
		chunks = append(chunks, chunk)
		remaining -= bitsToTake
	}
	return chunks
}

// ConstructByte reconstructs a byte from chunks produced by SplitByte.
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

// clearLastTwoBits will clear the last two bits of the passed in byte
// We do this with a mask of 1111 1100, i.e., 0100 1101 & 1111 1100 -> 0100 1100
func clearLastTwoBits(b byte) byte {
	return ClearLastNBits(b, 2)
}

// SetLastTwoBits will change the last two bits of the passed in byte to the value
//
// We do this by first clearing the last two bits of the passed in byte, and then using the OR operator
// to set the last two bits to the value of the last two bits of the valueToSet byte.
//
// For example:
// - We start with a byte of 0100 1101
// - Our value to set is 0000 0011
// - We first clear the last two bits of the byte, so we have 0100 1100
// - Then we use the OR operator to set the last two bits to 11
// - 0100 1100 | 0000 0011 -> 0100 1111
func SetLastTwoBits(b byte, valueToSet byte) byte {
	return SetLastNBits(b, valueToSet, 2)
}

// GetLastTwoBits will return a byte that is all 0s except for the last two bits of the passed in byte
func GetLastTwoBits(b byte) byte {
	return GetLastNBits(b, 2)
}

// ConstructByteFromQuartersAsSlice builds a byte out of a slice of bytes
func ConstructByteFromQuartersAsSlice(b []byte) byte {
	return ConstructByteFromQuarters(b[0], b[1], b[2], b[3])
}

// ConstructByteFromQuarters builds a byte based on the quarters passed in
//
// This is done by left shifting the first byte by 6, the second by 4, the third by 2, and then ORing them together
func ConstructByteFromQuarters(first, second, third, fourth byte) byte {
	return (((first << 6) | (second << 4)) | third<<2) | fourth
}

// ReturnMaskDifferenceN is the variable-bit-depth version of ReturnMaskDifference.
// It clears the last bitDepth bits of colorInt instead of a hardcoded 2.
func ReturnMaskDifferenceN(maskInt int32, multiplier int32, firstIndex int16, secondIndex int16, colorInt uint8, bitDepth int) bool {
	uMaskInt, uMultiplier := uint32(maskInt), uint32(multiplier)
	uFirstIndex, uSecondIndex := uint16(firstIndex), uint16(secondIndex)

	clearedDataByte := ClearLastNBits(colorInt, bitDepth)
	colorInt = clearedDataByte

	multipliedColorInt := uint32(colorInt) * uMultiplier
	var maxIndex uint16 = 31

	xorValue := uMaskInt ^ multipliedColorInt
	firstIndexShift := maxIndex - uFirstIndex
	firstShiftValue := xorValue >> firstIndexShift
	secondIndexShift := maxIndex - uSecondIndex
	secondShiftValue := xorValue >> secondIndexShift
	shiftedDataByte := firstShiftValue & 1
	secondShiftedDataByte := secondShiftValue & 1

	return shiftedDataByte == 1 && secondShiftedDataByte == 1
}

// ReturnMaskDifference will take in two bytes and identify if the data should be read or not by determining
// if the data byte indexes are different from the mask byte indexes.
//
// The XOR operator will only set the bit to 1 if the bits are different. We use the XOR operator
// to set all bits that are different to 1, then rotate the result and use the AND operator to zero
// out all the bits besides the least significant bit. If that bit is 1, then the data bit was different
// from the bit in the mask. We do that for both indexes, and then return true if both of them are 1 (i.e.,
// the data bit was different from both bits in the mask).
func ReturnMaskDifference(maskInt int32, multiplier int32, firstIndex int16, secondIndex int16, colorInt uint8) bool {
	return ReturnMaskDifferenceN(maskInt, multiplier, firstIndex, secondIndex, colorInt, 2)
}

// QuartersOfBytes32 will take in a 32-bit unsigned integer and return a byte slice of length 16
// TODO: Rename this function
func QuartersOfBytes32(intToSplit uint32) []byte {
	//Create a byte slice buffer with length of 4 (32-bit integer => 4 bytes)
	bs := make([]byte, 4)
	// LittleEndian designates the order of the importance of bits - this is the non-typical
	// right to left implementation, with the most important bit on the right side, leading to the
	// least important bit on the left side
	// The PutUint32 method takes in a slice of bytes with length 4, the unsigned 32-bit integer and returns
	// a byte slice. So basically takes the integer in and returns a slice made of the bytes of the integer
	// A 32-bit unsigned integer allows us to store how many channels we used to store the original data.
	// For example - we could potentially use 1,425,600 channels for a 1080 x 1350 picture (removing the first 30 pixels/90 channels for the header info)
	// This method takes an unsigned integer and fills the byte slice with its binary representation,
	// in the little endian order (right to left, most sig to least sig)
	// However this doesn't matter for reading it as a string, as it's mostly about bytes and their order, not bits
	// https://medium.com/go-walkthrough/go-walkthrough-encoding-binary-96dc5d4abb5d
	// We put the binary representation of the 32-bit integer into the byte slice buffer
	binary.LittleEndian.PutUint32(bs, intToSplit)
	//This generates a byte slice buffer with length of 16, so 128 bits total
	//This allows us to store two bits, but in a 0 buffered byte at each index
	//We split each byte of the 4 bytes (32-bit number/counter) up into 4 pieces, so we need 16 bytes to hold all of it
	quarters := make([]byte, 16)
	//This will iterate 4 times - 0, 4, 8, 12 for the counter
	//This is done so that we can access the correct index
	for i := 0; i < 16; i += 4 {
		//First iteration i = 0 - so we take the first element in bs(0/4 = 0) and then take the first element of the returned value for the first quarter, etc
		//Second iteration i = 4 - so we take the second element in bs(4/4 = 1) and then take the first element of returned value, etc
		//Use 4 as the increment so we can use the i + 1, etc to set the right quarter index
		//For example 1100 1010 => [00000011, 00000000, 00000010, 00000010] - quarters of byte will right shift each of the two bits to the far right
		//QuartersOfBytes return statement - return [4]byte{b & firstQuarterOfByteMax >> 6, b & secondQuarterOfByteMax >> 4, b & thirdQuarterOfByteMax >> 2, b & fourthQuarterOfByteMax}
		// firstQuarterOfByteMax = 192, secondQuarterOfByteMax = 48, thirdQuarterOfByteMax = 12, fourthQuarterOfByteMax = 3 - these are the max values utilizing just those two bits in a byte
		// 192 = 1100 0000, 48 = 0011 0000, 12 = 0000 1100, 3 = 0000 0011
		//Right shift operator moves the bits however many to the right - i.e. 1100 0000 >> 6 moves them 6 bits right - 0000 0011
		//The & operator is the bitwise AND - https://yourbasic.org/golang/operators/ - https://yourbasic.org/golang/bitwise-operator-cheat-sheet/
		//For bitwise AND, where both values are 1 we get a 1 back, otherwise we get 0 - i.e. 0101 & 1011 = 0001
		//The AND mask is used to zero out all of the other bits except the ones we want
		//To reverse, do the same thing but left shift and only have to use fourthQuarterOfByteMax
		//For example - 0100 1100 & 1100 0000 -> 0100 0000 >> 6 -> 0000 0001 & 0000 0011 -> 0000 0001 << 6 -> 0100 0000
		quarters[i] = SplitByteIntoQuarters(bs[i/4])[0]
		quarters[i+1] = SplitByteIntoQuarters(bs[i/4])[1]
		quarters[i+2] = SplitByteIntoQuarters(bs[i/4])[2]
		quarters[i+3] = SplitByteIntoQuarters(bs[i/4])[3]
	}
	return quarters
}

// QuartersOfBytes64 will take in a 64-bit unsigned integer and return a byte slice of length 24
func QuartersOfBytes64(intToSplit uint64) []byte {
	byteSlice := make([]byte, 8)

	binary.LittleEndian.PutUint64(byteSlice, intToSplit)

	quarters := make([]byte, 24)

	for i := 0; i < 24; i += 4 {
		quarters[i] = SplitByteIntoQuarters(byteSlice[i/4])[0]
		quarters[i+1] = SplitByteIntoQuarters(byteSlice[i/4])[1]
		quarters[i+2] = SplitByteIntoQuarters(byteSlice[i/4])[2]
		quarters[i+3] = SplitByteIntoQuarters(byteSlice[i/4])[3]
	}

	return quarters
}

// QuartersOfBytes16 will take in a 16-bit unsigned integer and return a byte slice of length 4
func QuartersOfBytes16(intToSplit uint16) []byte {
	byteSlice := make([]byte, 2)

	binary.LittleEndian.PutUint16(byteSlice, intToSplit)

	quarters := make([]byte, 4)

	for i := 0; i < 4; i += 4 {
		quarters[i] = SplitByteIntoQuarters(byteSlice[i/4])[0]
		quarters[i+1] = SplitByteIntoQuarters(byteSlice[i/4])[1]
		quarters[i+2] = SplitByteIntoQuarters(byteSlice[i/4])[2]
		quarters[i+3] = SplitByteIntoQuarters(byteSlice[i/4])[3]
	}

	return quarters
}

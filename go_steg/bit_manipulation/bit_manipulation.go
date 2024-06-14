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

// clearLastTwoBits will clear the last two bits of the passed in byte
// We do this with a mask of 1111 1100, i.e., 0100 1101 & 1111 1100 -> 0100 1100
func clearLastTwoBits(b byte) byte {
	return b & byte(252)
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
	return clearLastTwoBits(b) | valueToSet
}

// GetLastTwoBits will return a byte that is all 0s except for the last two bits of the passed in byte
func GetLastTwoBits(b byte) byte {
	return b & fourthQuarterOfByteMax
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

// ReturnMaskDifference will take in two bytes and identify if the data should be read or not by determining
// if the data byte indexes are different from the mask byte indexes.
//
// The XOR operator will only set the bit to 1 if the bits are different. We use the XOR operator
// to set all bits that are different to 1, then rotate the result and use the AND operator to zero
// out all the bits besides the least significant bit. If that bit is 1, then the data bit was different
// from the bit in the mask. We do that for both indexes, and then return true if both of them are 1 (i.e.,
// the data bit was different from both bits in the mask).
func ReturnMaskDifference(maskInt int32, multiplier int32, firstIndex int16, secondIndex int16, colorInt uint8) bool {
	//First convert everything to uint with an unsafe pointer
	//https://old.reddit.com/r/golang/comments/29iir54/convert_from_int32_to_uint32/cilcok1
	//uMaskInt, uMultiplier := *(*uint32)(unsafe.Pointer(&maskInt)), *(*uint32)(unsafe.Pointer(&multiplier))
	//uFirstIndex, uSecondIndex := *(*uint16)(unsafe.Pointer(&firstIndex)), *(*uint16)(unsafe.Pointer(&secondIndex))
	uMaskInt, uMultiplier := uint32(maskInt), uint32(multiplier)
	uFirstIndex, uSecondIndex := uint16(firstIndex), uint16(secondIndex)

	// Clear out the last two bits to make sure the changes in data don't bite us - since we are multiplying
	// the data integer we could end up with different results - i.e. if it was 198 before, now it's 200,
	// and our multiplier is 10, then we would be comparing 1980 initially and 2000 after, which wouldn't
	// give us the correct results.
	// This could happen if this byte was used to store data from the embed image.
	// Clearing the last two bites means regardless of any scenario, the number used should be the same.
	clearedDataByte := clearLastTwoBits(colorInt)

	// Set colorInt back to the clearedDataByte value but as an uint8,
	// so we can multiply this with the multiplier.
	// A byte is an alias for uint8 in Golang.
	colorInt = clearedDataByte

	// Multiply the colorInt with the multiplier which gives us a 32-bit number.
	// The multiplier is maxed at 8421504 so that we don't go over signed 32-bit max. That is,
	// 255 * 8421504 = 2,147,483,520, and we don't go over 2,147,483,647.
	// TODO: Investigate whether we can actually use unsigned 32-bit max as the limit
	//multipliedColorInt := *(*uint32)(unsafe.Pointer(&colorInt)) * uMultiplier
	multipliedColorInt := uint32(colorInt) * uMultiplier
	// MaxIndex of a 32-bit number in a byte slice is 31 for 0 indexed, 16 bits to match the incoming indexes
	var maxIndex uint16 = 31

	// Run XOR on the colorInt - this will set the bits to 1 if the bits of the two numbers are different.
	// After that, shift the bits to the right and then zero out all bits except the last one.
	// Using maxIndex - uFirstIndex and maxIndex - uSecondIndex will shift the bits to the right
	// by the amount of the index.
	// For example, if the index is 2, then we will shift the bits to the right by 29 (31-2) bits.
	// This will leave us with the last bit of the XORed number.
	xorValue := uMaskInt ^ multipliedColorInt
	firstIndexShift := maxIndex - uFirstIndex
	firstShiftValue := xorValue >> firstIndexShift
	secondIndexShift := maxIndex - uSecondIndex
	secondShiftValue := xorValue >> secondIndexShift
	shiftedDataByte := firstShiftValue & 1
	secondShiftedDataByte := secondShiftValue & 1

	return shiftedDataByte == 1 && secondShiftedDataByte == 1
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

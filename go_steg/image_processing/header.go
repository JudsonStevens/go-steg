package image_processing

import (
	"encoding/binary"
	"go-steg/go_steg/bit_manipulation"
	"go-steg/go_steg/reed_solomon"
	"image"
	"strings"
)

// HeaderInfo contains all header metadata for encode/decode.
type HeaderInfo struct {
	// Legacy fields
	PhotoID     uint64
	PhotoNumber uint16
	DataCount   uint32

	// New format fields
	IsNewFormat    bool
	FileExtension  string // up to 8 chars, no dot
	BitDepth       int    // 1-4
	HuffmanEnabled bool
	RSEnabled      bool
	RSLevel        reed_solomon.RedundancyLevel
	Checksum       uint16 // low 12 bits of CRC-16
	ByteCountMod   uint16 // pipeline output byte count modulo 4096
}

// writeHeader writes all header metadata into the first 34 pixels of column 0.
// Header always uses 2-bit operations.
func writeHeader(img *image.RGBA, info HeaderInfo) {
	// y=0..7: photo ID (24 quarter-values across 8 pixels, 3 per pixel)
	photoIDQuarters := bit_manipulation.QuartersOfBytes64(info.PhotoID)
	for y := 0; y < 8; y++ {
		c := img.RGBAAt(0, y)
		idx := y * 3
		c.R = bit_manipulation.SetLastTwoBits(c.R, photoIDQuarters[idx])
		c.G = bit_manipulation.SetLastTwoBits(c.G, photoIDQuarters[idx+1])
		c.B = bit_manipulation.SetLastTwoBits(c.B, photoIDQuarters[idx+2])
		img.SetRGBA(0, y, c)
	}

	// y=8: photo number (6 bits across 3 channels: R=bits[5:4], G=bits[3:2], B=bits[1:0])
	{
		c := img.RGBAAt(0, 8)
		pn := info.PhotoNumber
		c.R = bit_manipulation.SetLastTwoBits(c.R, byte((pn>>4)&0x3))
		c.G = bit_manipulation.SetLastTwoBits(c.G, byte((pn>>2)&0x3))
		c.B = bit_manipulation.SetLastTwoBits(c.B, byte(pn&0x3))
		img.SetRGBA(0, 8, c)
	}

	// y=9..12: data count (16 quarter-values across 4 pixels)
	dataCountQuarters := bit_manipulation.QuartersOfBytes32(info.DataCount)
	for y := 9; y < 13; y++ {
		c := img.RGBAAt(0, y)
		idx := (y - 9) * 3
		c.R = bit_manipulation.SetLastTwoBits(c.R, dataCountQuarters[idx])
		c.G = bit_manipulation.SetLastTwoBits(c.G, dataCountQuarters[idx+1])
		c.B = bit_manipulation.SetLastTwoBits(c.B, dataCountQuarters[idx+2])
		img.SetRGBA(0, y, c)
	}

	// y=13..14: version marker
	for y := 13; y < 15; y++ {
		c := img.RGBAAt(0, y)
		idx := (y - 13) * 3
		c.R = bit_manipulation.SetLastTwoBits(c.R, versionMarkerBytes[idx])
		c.G = bit_manipulation.SetLastTwoBits(c.G, versionMarkerBytes[idx+1])
		c.B = bit_manipulation.SetLastTwoBits(c.B, versionMarkerBytes[idx+2])
		img.SetRGBA(0, y, c)
	}

	// y=15..25: file extension (up to 8 bytes, each split into 4 quarters, written across 11 pixels = 33 channels)
	// We have 11 pixels * 3 channels = 33 quarter-values, enough for 8 bytes (32 quarters) + 1 spare
	extBytes := make([]byte, 8)
	copy(extBytes, []byte(info.FileExtension))
	extQuarters := make([]byte, 0, 32)
	for _, b := range extBytes {
		q := bit_manipulation.SplitByteIntoQuarters(b)
		extQuarters = append(extQuarters, q[0], q[1], q[2], q[3])
	}
	qi := 0
	for y := 15; y < 26; y++ {
		c := img.RGBAAt(0, y)
		if qi < len(extQuarters) {
			c.R = bit_manipulation.SetLastTwoBits(c.R, extQuarters[qi])
		}
		qi++
		if qi < len(extQuarters) {
			c.G = bit_manipulation.SetLastTwoBits(c.G, extQuarters[qi])
		}
		qi++
		if qi < len(extQuarters) {
			c.B = bit_manipulation.SetLastTwoBits(c.B, extQuarters[qi])
		}
		qi++
		img.SetRGBA(0, y, c)
	}

	// y=26: encoding flags
	// R = (bitDepth-1) & 0x3, G = huffman(MSB) | rs(LSB), B = rsLevel(MSB) | 0(LSB)
	{
		c := img.RGBAAt(0, 26)
		bd := byte(0)
		if info.BitDepth >= 1 && info.BitDepth <= 4 {
			bd = byte(info.BitDepth - 1)
		}
		c.R = bit_manipulation.SetLastTwoBits(c.R, bd&0x3)

		var gVal byte
		if info.HuffmanEnabled {
			gVal |= 0x2
		}
		if info.RSEnabled {
			gVal |= 0x1
		}
		c.G = bit_manipulation.SetLastTwoBits(c.G, gVal)

		var bVal byte
		if info.RSLevel == reed_solomon.High {
			bVal |= 0x2
		}
		c.B = bit_manipulation.SetLastTwoBits(c.B, bVal)
		img.SetRGBA(0, 26, c)
	}

	// y=27..28: checksum (12 bits across 2 pixels, 6 channels)
	writeU12(img, 27, info.Checksum)

	// y=29..30: byte count modulo (12 bits across 2 pixels)
	writeU12(img, 29, info.ByteCountMod)

	// y=31..33: reserved (leave as-is)
}

// writeU12 writes a 12-bit value across 2 pixels (6 channels) starting at the given y.
func writeU12(img *image.RGBA, startY int, val uint16) {
	// 12 bits => 6 two-bit values
	vals := [6]byte{
		byte((val >> 10) & 0x3),
		byte((val >> 8) & 0x3),
		byte((val >> 6) & 0x3),
		byte((val >> 4) & 0x3),
		byte((val >> 2) & 0x3),
		byte(val & 0x3),
	}
	for i := 0; i < 2; i++ {
		c := img.RGBAAt(0, startY+i)
		idx := i * 3
		c.R = bit_manipulation.SetLastTwoBits(c.R, vals[idx])
		c.G = bit_manipulation.SetLastTwoBits(c.G, vals[idx+1])
		c.B = bit_manipulation.SetLastTwoBits(c.B, vals[idx+2])
		img.SetRGBA(0, startY+i, c)
	}
}

// readU12 reads a 12-bit value from 2 pixels (6 channels) starting at the given y.
func readU12(img *image.RGBA, startY int) uint16 {
	var vals [6]byte
	for i := 0; i < 2; i++ {
		c := img.RGBAAt(0, startY+i)
		idx := i * 3
		vals[idx] = bit_manipulation.GetLastTwoBits(c.R)
		vals[idx+1] = bit_manipulation.GetLastTwoBits(c.G)
		vals[idx+2] = bit_manipulation.GetLastTwoBits(c.B)
	}
	var val uint16
	for i := 0; i < 6; i++ {
		val = (val << 2) | uint16(vals[i])
	}
	return val
}

// readHeader reads all header metadata from the first 34 pixels of column 0.
func readHeader(img *image.RGBA) HeaderInfo {
	var info HeaderInfo

	// y=0..7: photo ID
	photoIDQuarters := make([]byte, 0, 24)
	for y := 0; y < 8; y++ {
		c := img.RGBAAt(0, y)
		photoIDQuarters = append(photoIDQuarters,
			bit_manipulation.GetLastTwoBits(c.R),
			bit_manipulation.GetLastTwoBits(c.G),
			bit_manipulation.GetLastTwoBits(c.B),
		)
	}
	// Reconstruct uint64 from 24 quarters (6 bytes)
	idBytes := make([]byte, 8)
	for i := 0; i < 6; i++ {
		idBytes[i] = bit_manipulation.ConstructByteFromQuartersAsSlice(photoIDQuarters[i*4 : i*4+4])
	}
	info.PhotoID = binary.LittleEndian.Uint64(idBytes)

	// y=8: photo number (6 bits from 3 channels)
	{
		c := img.RGBAAt(0, 8)
		r := bit_manipulation.GetLastTwoBits(c.R)
		g := bit_manipulation.GetLastTwoBits(c.G)
		b := bit_manipulation.GetLastTwoBits(c.B)
		info.PhotoNumber = uint16(r)<<4 | uint16(g)<<2 | uint16(b)
	}

	// y=9..12: data count
	dataCountQuarters := make([]byte, 0, 12)
	for y := 9; y < 13; y++ {
		c := img.RGBAAt(0, y)
		dataCountQuarters = append(dataCountQuarters,
			bit_manipulation.GetLastTwoBits(c.R),
			bit_manipulation.GetLastTwoBits(c.G),
			bit_manipulation.GetLastTwoBits(c.B),
		)
	}
	dcBytes := make([]byte, 4)
	dcBytes[0] = bit_manipulation.ConstructByteFromQuartersAsSlice(dataCountQuarters[0:4])
	dcBytes[1] = bit_manipulation.ConstructByteFromQuartersAsSlice(dataCountQuarters[4:8])
	dcBytes[2] = bit_manipulation.ConstructByteFromQuartersAsSlice(dataCountQuarters[8:12])
	info.DataCount = binary.LittleEndian.Uint32(dcBytes)

	// y=13..14: check version marker
	var markerVals [6]byte
	for i := 0; i < 2; i++ {
		c := img.RGBAAt(0, 13+i)
		idx := i * 3
		markerVals[idx] = bit_manipulation.GetLastTwoBits(c.R)
		markerVals[idx+1] = bit_manipulation.GetLastTwoBits(c.G)
		markerVals[idx+2] = bit_manipulation.GetLastTwoBits(c.B)
	}

	markerMatch := markerVals == versionMarkerBytes

	// Read bit depth from y=26 to check validity
	flagC := img.RGBAAt(0, 26)
	bdRaw := bit_manipulation.GetLastTwoBits(flagC.R)

	if markerMatch && bdRaw <= 3 {
		info.IsNewFormat = true
	} else {
		// Legacy mode: only legacy fields populated
		return info
	}

	// y=15..25: file extension
	extQuarters := make([]byte, 0, 33)
	for y := 15; y < 26; y++ {
		c := img.RGBAAt(0, y)
		extQuarters = append(extQuarters,
			bit_manipulation.GetLastTwoBits(c.R),
			bit_manipulation.GetLastTwoBits(c.G),
			bit_manipulation.GetLastTwoBits(c.B),
		)
	}
	extBuf := make([]byte, 8)
	for i := 0; i < 8; i++ {
		qi := i * 4
		extBuf[i] = bit_manipulation.ConstructByteFromQuartersAsSlice(extQuarters[qi : qi+4])
	}
	info.FileExtension = strings.TrimRight(string(extBuf), "\x00")

	// y=26: encoding flags (already read flagC)
	info.BitDepth = int(bdRaw) + 1

	gVal := bit_manipulation.GetLastTwoBits(flagC.G)
	info.HuffmanEnabled = (gVal & 0x2) != 0
	info.RSEnabled = (gVal & 0x1) != 0

	bVal := bit_manipulation.GetLastTwoBits(flagC.B)
	if (bVal & 0x2) != 0 {
		info.RSLevel = reed_solomon.High
	} else {
		info.RSLevel = reed_solomon.Standard
	}

	// y=27..28: checksum
	info.Checksum = readU12(img, 27)

	// y=29..30: byte count modulo
	info.ByteCountMod = readU12(img, 29)

	return info
}

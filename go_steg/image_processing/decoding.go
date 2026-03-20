package image_processing

import (
	"fmt"
	"go-steg/cli/helpers"
	"go-steg/go_steg/bit_manipulation"
	"go-steg/go_steg/pipeline"
	"image"
	"io"
	"math"
	"os"
	"time"
)

// MultiCarrierDecodeByFileNames performs steganography decoding of data previously encoded by the MultiCarrierEncode function.
// The data is decoded from carrier files, and it is saved in a new file.
// NOTE: The order of the carriers MUST be the same as the one when encoding.
func MultiCarrierDecodeByFileNames(carrierFileNames []string, password string, outputFileDir string) (err error) {
	if len(carrierFileNames) == 0 {
		return fmt.Errorf("missing carriers names")
	}

	carriers := make([]io.Reader, 0, len(carrierFileNames))
	for _, name := range carrierFileNames {
		carrier, err := os.Open(name)
		if err != nil {
			logger.Errorf("Error opening carrier file: %v", err)
			return fmt.Errorf("error opening carrier file %s: %v", name, err)
		}
		defer func() {
			closeErr := carrier.Close()
			if err == nil {
				err = closeErr
			}
		}()
		carriers = append(carriers, carrier)
	}

	// Peek at the first carrier to read the header for file extension
	firstCarrierForHeader, err := os.Open(carrierFileNames[0])
	if err != nil {
		return fmt.Errorf("error opening first carrier for header: %v", err)
	}
	firstRGBA, _, err := getImageAsRGBA(firstCarrierForHeader)
	firstCarrierForHeader.Close()
	if err != nil {
		return fmt.Errorf("error reading first carrier image: %v", err)
	}
	header := readHeader(firstRGBA)

	// Determine file extension for output
	ext := "png" // default for legacy
	if header.IsNewFormat && header.FileExtension != "" {
		ext = header.FileExtension
	}

	currentTime := time.Now()
	currentTimeString := currentTime.Format("2006-01-02 15:04:05")
	resultName := fmt.Sprintf("%s/decoded_file-%s.%s", outputFileDir, currentTimeString, ext)

	result, err := os.Create(resultName)
	if err != nil {
		logger.Errorf("Error creating the result file: %v", err)
		return fmt.Errorf("error creating result file: %v", err)
	}
	defer func() {
		closeErr := result.Close()
		if err == nil {
			err = closeErr
		}
	}()

	if err != nil {
		logger.Errorf("Error closing the results file: %v", err)
		return fmt.Errorf("issue closing the result file: %w", err)
	}

	err = MultiCarrierDecode(carriers, result, password)
	if err != nil {
		logger.Errorf("Error decoding files: %v", err)
		_ = os.Remove(resultName)
	}
	return err
}

// MultiCarrierDecode performs steganography decoding of Readers with previously encoded data chunks by the
// MultiCarrierEncode function and writes to result Writer.
//
// NOTE: The order of the carriers MUST be the same as the one when encoding.
func MultiCarrierDecode(carriers []io.Reader, result io.Writer, password string) error {
	mask := generateMaskingInfo(password)

	fmt.Println("Masking info: ", mask)

	// Collect raw decoded bytes from each carrier
	var allBytes []byte
	var firstHeader HeaderInfo

	for i := 0; i < len(carriers); i++ {
		decoded, header, err := DecodeRaw(carriers[i], mask)
		if err != nil {
			logger.Errorf("Error decoding chunk: %v", err)
			return fmt.Errorf("error decoding chunk with index %d: %v", i, err)
		}
		if i == 0 {
			firstHeader = header
		}
		allBytes = append(allBytes, decoded...)
	}

	// If new format, run pipeline decode
	if firstHeader.IsNewFormat {
		cfg := pipeline.Config{
			BitDepth:       firstHeader.BitDepth,
			HuffmanEnabled: firstHeader.HuffmanEnabled,
			RSEnabled:      firstHeader.RSEnabled,
			RSLevel:        firstHeader.RSLevel,
			Password:       password,
		}
		decoded, err := pipeline.Decode(allBytes, cfg)
		if err != nil {
			return fmt.Errorf("error in pipeline decode: %w", err)
		}
		allBytes = decoded
	}

	if _, err := result.Write(allBytes); err != nil {
		logger.Errorf("Error writing result file: %v", err)
		return err
	}

	return nil
}

// DecodeRaw extracts the raw embedded bytes from a single carrier, returning the bytes and the header info.
func DecodeRaw(carrier io.Reader, mask Mask) ([]byte, HeaderInfo, error) {
	RGBAImage, _, err := getImageAsRGBA(carrier)
	if err != nil {
		logger.Errorf("Error parsing carrier image: %v", err)
		return nil, HeaderInfo{}, fmt.Errorf("error parsing carrier image: %w", err)
	}

	dx := RGBAImage.Bounds().Dx()
	dy := RGBAImage.Bounds().Dy()

	header := readHeader(RGBAImage)

	dataCount := int(header.DataCount)
	fmt.Printf("Data count for this carrier: %v\n", dataCount)

	if helpers.UseMask {
		openSlots := DetermineOpenSlotsWithMask(RGBAImage, dx, dy, mask)
		fmt.Printf("Number of slots availabe with mask: %v\n", openSlots)
	}

	if !header.IsNewFormat {
		// Legacy 2-bit extraction
		return decodeLegacy(RGBAImage, dx, dy, dataCount, mask), header, nil
	}

	// New format: use variable bit depth
	bitDepth := header.BitDepth
	if bitDepth < 1 || bitDepth > 4 {
		bitDepth = 2
	}

	dataBytes := make([]byte, 0, dataCount)

	for x := 0; x < dx && dataCount > 0; x++ {
		for y := totalReservedPixels; y < dy && dataCount > 0; y++ {
			c := RGBAImage.RGBAAt(x, y)
			if helpers.UseMask && bit_manipulation.ReturnMaskDifferenceN(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.R, bitDepth) == mask.changeBoolean {
				dataBytes = append(dataBytes, bit_manipulation.GetLastNBits(c.R, bitDepth))
				dataCount--
			} else if !helpers.UseMask {
				dataBytes = append(dataBytes, bit_manipulation.GetLastNBits(c.R, bitDepth))
				dataCount--
			}
			if dataCount > 0 {
				if helpers.UseMask && bit_manipulation.ReturnMaskDifferenceN(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.G, bitDepth) == mask.changeBoolean {
					dataBytes = append(dataBytes, bit_manipulation.GetLastNBits(c.G, bitDepth))
					dataCount--
				} else if !helpers.UseMask {
					dataBytes = append(dataBytes, bit_manipulation.GetLastNBits(c.G, bitDepth))
					dataCount--
				}
			}
			if dataCount > 0 {
				if helpers.UseMask && bit_manipulation.ReturnMaskDifferenceN(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.B, bitDepth) == mask.changeBoolean {
					dataBytes = append(dataBytes, bit_manipulation.GetLastNBits(c.B, bitDepth))
					dataCount--
				} else if !helpers.UseMask {
					dataBytes = append(dataBytes, bit_manipulation.GetLastNBits(c.B, bitDepth))
					dataCount--
				}
			}
			if dataCount <= 0 {
				fmt.Printf("Last decoded pixel location - (%v, %v)\n", x, y)
			}
		}
	}

	fmt.Printf("Data count after loop is: %v\n", dataCount)
	fmt.Printf("Data bytes length after loop is: %v\n", len(dataBytes))

	if dataCount < 0 {
		dataBytes = dataBytes[:len(dataBytes)+dataCount]
	}

	// Align chunks based on bit depth
	chunksPerByte := int(math.Ceil(8.0 / float64(bitDepth)))
	dataBytes = alignN(dataBytes, chunksPerByte)

	// Reconstruct bytes from chunks
	resultBytes := make([]byte, 0, len(dataBytes)/chunksPerByte)
	for i := 0; i+chunksPerByte <= len(dataBytes); i += chunksPerByte {
		resultBytes = append(resultBytes, bit_manipulation.ConstructByte(dataBytes[i:i+chunksPerByte], bitDepth))
	}

	fmt.Printf("Result bytes length - %v\n\n", len(resultBytes))

	return resultBytes, header, nil
}

// Decode reverses the Encode method and extracts the embed image data from the carrier file.
// This is kept for backward compatibility; it calls DecodeRaw internally.
func Decode(carrier io.Reader, result io.Writer, mask Mask) error {
	decoded, _, err := DecodeRaw(carrier, mask)
	if err != nil {
		return err
	}

	if _, err = result.Write(decoded); err != nil {
		logger.Errorf("Error writing result file: %v", err)
		return err
	}

	return nil
}

// decodeLegacy extracts data using the legacy 2-bit method with legacyTotalReservedPixels bounds.
func decodeLegacy(RGBAImage *image.RGBA, dx, dy, dataCount int, mask Mask) []byte {
	dataBytes := make([]byte, 0, 100000)

	for x := 0; x < dx && dataCount > 0; x++ {
		for y := totalReservedPixels; y < dy && dataCount > 0; y++ {
			c := RGBAImage.RGBAAt(x, y)
			if helpers.UseMask && bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.R) == mask.changeBoolean {
				dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.R))
				dataCount--
			} else if !helpers.UseMask {
				dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.R))
				dataCount--
			}
			if dataCount > 0 {
				if helpers.UseMask && bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.G) == mask.changeBoolean {
					dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.G))
					dataCount--
				} else if !helpers.UseMask {
					dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.G))
					dataCount--
				}
			}
			if dataCount > 0 {
				if helpers.UseMask && bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.B) == mask.changeBoolean {
					dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.B))
					dataCount--
				} else if !helpers.UseMask {
					dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.B))
					dataCount--
				}
			}
			if dataCount <= 0 {
				fmt.Printf("Last decoded pixel location - (%v, %v)\n", x, y)
			}
		}
	}

	fmt.Printf("Data count after loop is: %v\n", dataCount)
	fmt.Printf("Data bytes length after loop is: %v\n", len(dataBytes))

	if dataCount < 0 {
		dataBytes = dataBytes[:len(dataBytes)+dataCount]
	}

	dataBytes = align(dataBytes)

	resultBytes := make([]byte, 0, len(dataBytes)/4)
	for i := 0; i < len(dataBytes); i += 4 {
		resultBytes = append(resultBytes, bit_manipulation.ConstructByteFromQuartersAsSlice(dataBytes[i:i+4]))
	}

	fmt.Printf("Result bytes length - %v\n\n", len(resultBytes))

	return resultBytes
}

// align ensures that the slice is divisible by 4 (for legacy 2-bit mode).
func align(dataBytes []byte) []byte {
	switch len(dataBytes) % 4 {
	case 1:
		dataBytes = append(dataBytes, byte(0), byte(0), byte(0))
	case 2:
		dataBytes = append(dataBytes, byte(0), byte(0))
	case 3:
		dataBytes = append(dataBytes, byte(0))
	}
	return dataBytes
}

// alignN ensures that the slice length is divisible by n (for variable bit depth).
func alignN(dataBytes []byte, n int) []byte {
	remainder := len(dataBytes) % n
	if remainder != 0 {
		padding := n - remainder
		for i := 0; i < padding; i++ {
			dataBytes = append(dataBytes, byte(0))
		}
	}
	return dataBytes
}

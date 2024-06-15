package image_processing

import (
	"encoding/binary"
	"fmt"
	"go-steg/cli/helpers"
	"go-steg/go_steg/bit_manipulation"
	"go-steg/go_steg/logging"
	"image"
	"io"
	"os"
	"time"
)

func init() {
	logger = logging.NewLogger("")
}

// MultiCarrierDecodeByFileNames performs steganography decoding of data previously encoded by the MultiCarrierEncode function.
// The data is decoded from carrier files, and it is saved in separate new file
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

	currentTime := time.Now()
	currentTimeString := currentTime.Format("2006-01-02 15:04:05")
	resultName := fmt.Sprintf("%s/decoded_image-%s.png", outputFileDir, currentTimeString)

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
// TODO: Eliminate the need for ordering by using the photoId values in the header to determine order
func MultiCarrierDecode(carriers []io.Reader, result io.Writer, password string) error {
	mask := generateMaskingInfo(password)

	fmt.Println("Masking info: ", mask)

	for i := 0; i < len(carriers); i++ {
		if err := Decode(carriers[i], result, mask); err != nil {
			logger.Errorf("Error decoding chunk: %v", err)
			return fmt.Errorf("error decoding chunk with index %d: %v", i, err)
		}
	}
	return nil
}

// Decode reverses the Encode method and extracts the embed image data from the carrier file
func Decode(carrier io.Reader, result io.Writer, mask Mask) error {
	RGBAImage, _, err := getImageAsRGBA(carrier)
	if err != nil {
		logger.Errorf("Error parsing carrier image: %v", err)
		return fmt.Errorf("error parsing carrier image: %w", err)
	}

	dx := RGBAImage.Bounds().Dx()
	dy := RGBAImage.Bounds().Dy()

	dataBytes := make([]byte, 0, 100000)
	resultBytes := make([]byte, 0, 100000)

	dataCount := extractDataCount(RGBAImage)
	fmt.Printf("Data count for this carrier: %v\n", dataCount)

	if helpers.UseMask {
		openSlots := DetermineOpenSlotsWithMask(RGBAImage, dx, dy, mask)
		fmt.Printf("Number of slots availabe with mask: %v\n", openSlots)
	}

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
		dataBytes = dataBytes[:len(dataBytes)+dataCount] //remove bytes that are not part of data and mistakenly added
	}

	dataBytes = align(dataBytes) // len(dataBytes) must be aliquot of 4/divisible by 4

	for i := 0; i < len(dataBytes); i += 4 {
		resultBytes = append(resultBytes, bit_manipulation.ConstructByteFromQuartersAsSlice(dataBytes[i:i+4]))
	}

	fmt.Printf("Result bytes length - %v\n\n", len(resultBytes))

	if _, err = result.Write(resultBytes); err != nil {
		logger.Errorf("Error writing result file: %v", err)
		return err
	}

	return nil
}

// align ensures that the slice is divisible by 4, or in other words, an aliquot of 4.
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

// extractDataCount extracts the data count from the carrier image.
func extractDataCount(RGBAImage *image.RGBA) int {
	//We initialize a slice that's 12 bytes long, to be able to fit the bits we have to capture
	// since there are 2 bits per byte/channel from the picture
	dataCountBytes := make([]byte, 0, 12)

	//We want to start on the y-axis after the photoID and the photoNumber embedded information
	x := 0
	for y := photoIDHeaderReservedPixels + photoNumberHeaderReservedPixels; y < totalReservedPixels; y++ {
		c := RGBAImage.RGBAAt(x, y)
		dataCountBytes = append(dataCountBytes, bit_manipulation.GetLastTwoBits(c.R), bit_manipulation.GetLastTwoBits(c.G), bit_manipulation.GetLastTwoBits(c.B))
	}

	// Empty byte at the end to make the LittleEndian conversion work
	var bs = []byte{
		bit_manipulation.ConstructByteFromQuartersAsSlice(dataCountBytes[:4]),
		bit_manipulation.ConstructByteFromQuartersAsSlice(dataCountBytes[4:8]),
		bit_manipulation.ConstructByteFromQuartersAsSlice(dataCountBytes[8:]),
		byte(0),
	}

	return int(binary.LittleEndian.Uint32(bs))
}

// extractPhotoID extracts the photo ID from the carrier image.
func extractPhotoID(RGBAImage *image.RGBA) int {
	photoIDBytes := make([]byte, 0, 24)
	x := 0
	for y := 0; y < photoIDHeaderReservedPixels; y++ {
		c := RGBAImage.RGBAAt(x, y)
		photoIDBytes = append(photoIDBytes, bit_manipulation.GetLastTwoBits(c.R), bit_manipulation.GetLastTwoBits(c.G), bit_manipulation.GetLastTwoBits(c.B))
	}

	// Empty bytes at the end to make the LittleEndian conversion work
	var bs = []byte{
		bit_manipulation.ConstructByteFromQuartersAsSlice(photoIDBytes[:4]),
		bit_manipulation.ConstructByteFromQuartersAsSlice(photoIDBytes[4:8]),
		bit_manipulation.ConstructByteFromQuartersAsSlice(photoIDBytes[8:12]),
		bit_manipulation.ConstructByteFromQuartersAsSlice(photoIDBytes[12:16]),
		bit_manipulation.ConstructByteFromQuartersAsSlice(photoIDBytes[16:20]),
		bit_manipulation.ConstructByteFromQuartersAsSlice(photoIDBytes[20:]),
		byte(0),
		byte(0),
	}

	return int(binary.LittleEndian.Uint64(bs))
}

// extractPhotoNumber extracts the photo number from the carrier image.
func extractPhotoNumber(RGBAImage *image.RGBA) int {
	photoNumberBytes := make([]byte, 0, 4)
	x := 0
	for y := photoIDHeaderReservedPixels; y < photoIDHeaderReservedPixels+photoNumberHeaderReservedPixels; y++ {
		c := RGBAImage.RGBAAt(x, y)
		photoNumberBytes = append(photoNumberBytes, bit_manipulation.GetLastTwoBits(c.R), bit_manipulation.GetLastTwoBits(c.G), bit_manipulation.GetLastTwoBits(c.B))
	}
	// Add another byte on the end to make the byte slice work with the construct method
	photoNumberBytes = append(photoNumberBytes, byte(0))

	var bs = []byte{bit_manipulation.ConstructByteFromQuartersAsSlice(photoNumberBytes), byte(0), byte(0), byte(0)}

	return int(binary.LittleEndian.Uint16(bs))
}

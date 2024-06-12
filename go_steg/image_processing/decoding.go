package image_processing

import (
	"encoding/binary"
	"fmt"
	"github.com/disintegration/imaging"
	"go-steg/cli/helpers"
	"go-steg/go_steg/bit_manipulation"
	"go-steg/go_steg/logging"
	"image"
	"image/color"
	"image/png"
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
	currentTimeString := currentTime.Format("2006-01-02-15-04-05")
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

	newResultName := fmt.Sprintf("%s/new-decoded_image-%s.png", outputFileDir, currentTimeString)

	newResult, err := os.Create(newResultName)
	if err != nil {
		logger.Errorf("Error creating the result file: %v", err)
		return fmt.Errorf("error creating result file: %v", err)
	}
	defer func() {
		closeErr := newResult.Close()
		if err == nil {
			err = closeErr
		}
	}()

	if err != nil {
		logger.Errorf("Error closing the results file: %v", err)
		return fmt.Errorf("issue closing the result file: %w", err)
	}

	err = MultiCarrierDecode(carriers, result, newResult, password)
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
func MultiCarrierDecode(carriers []io.Reader, result io.Writer, newResult io.Writer, password string) error {
	mask := generateMaskingInfo(password)
	fmt.Println("Masking info: ", mask)

	for i := 0; i < len(carriers); i++ {
		if err := Decode(carriers[i], result, newResult, mask); err != nil {
			logger.Errorf("Error decoding chunk: %v", err)
			return fmt.Errorf("error decoding chunk with index %d: %v", i, err)
		}
	}
	return nil
}

// Decode reverses the Encode method and extracts the embed image data from the carrier file
func Decode(carrier io.Reader, result io.Writer, newResult io.Writer, mask Mask) error {
	img, format, err := image.Decode(carrier)
	if err != nil {
		logger.Errorf("Error parsing carrier image: %v", err)
		return fmt.Errorf("error parsing carrier image: %w", err)
	}
	// Open the carrier image as an NRGBA image, along with getting the format of the carrier image
	nrgbaCarrierImage := imaging.Clone(img)
	if format != "png" && format != "jpeg" {
		logger.Errorf("Unsupported carrier format")
		return fmt.Errorf("Unsupported carrier format\n")
	}

	dx := nrgbaCarrierImage.Bounds().Dx()
	dy := nrgbaCarrierImage.Bounds().Dy()

	dataCount := extractDataCount(nrgbaCarrierImage)
	dataBytes := make([]byte, 0, 100000)
	resultBytes := make([]byte, 0, 100000)

	// Generate a new image struct to add the data to
	resultImage := image.NewNRGBA(image.Rect(0, 0, 540, 810))
	fmt.Printf("Data count for this carrier is - %v\n", dataCount)

	if helpers.UseMask {
		openSlots := DetermineOpenSlotsWithMask(nrgbaCarrierImage, dx, dy, mask)
		fmt.Printf("Number of slots availabe with mask: %v\n", openSlots)
	}

	for x := 0; x < dx && dataCount > 0; x++ {
		for y := 0; y < dy && dataCount > 0; y++ {
			if x == 0 && y < totalReservedPixels {
				continue
			}
			c := nrgbaCarrierImage.NRGBAAt(x, y)
			_ = resultImage.NRGBAAt(x, y)
			if helpers.UseMask && bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.R) == mask.changeBoolean {
				dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.R))
				dataCount--
			} else {
				dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.R))
				dataCount--
			}
			if dataCount > 0 && helpers.UseMask && bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.G) == mask.changeBoolean {
				dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.G))
				dataCount--
			} else if dataCount > 0 {
				dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.G))
				dataCount--
			}
			if dataCount > 0 && helpers.UseMask && bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.B) == mask.changeBoolean {
				dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.B))
				dataCount--
			} else if dataCount > 0 {
				dataBytes = append(dataBytes, bit_manipulation.GetLastTwoBits(c.G))
				dataCount--
			}
			if dataCount <= 0 {
				fmt.Printf("Last decoded pixel location - (%v, %v)\n", x, y)
			}
		}
		if dataCount <= 0 {
			break
		}
	}

	fmt.Printf("Data count after loop is - %v\n", dataCount)
	fmt.Printf("Data bytes length after loop is - %v\n", len(dataBytes))

	if dataCount < 0 {
		dataBytes = dataBytes[:len(dataBytes)+dataCount] //remove bytes that are not part of data and mistakenly added
	}

	// Align ensures that the dataBytes length is divisible by 4 by 0 padding any extra bits at the end.
	// This is necessary because we use the constructByteFromQuartersAsSlice method which requires a slice of length 4.
	dataBytes = align(dataBytes)

	for i := 0; i < len(dataBytes); i += 4 {
		resultByte := bit_manipulation.ConstructByteFromQuartersAsSlice(dataBytes[i : i+4])
		resultBytes = append(resultBytes, resultByte)
	}

	// Iterate over the resultBytes and set the appropriate pixels
	for x := 0; x < 540; x++ {
		for y := 0; y < 810; y++ {
			if x == 0 && y < totalReservedPixels {
				continue
			}
			if len(resultBytes) > 0 {
				redByte, ok := safeReadFromArray(resultBytes, 0)
				if !ok {
					break
				}
				greenByte, ok := safeReadFromArray(resultBytes, 1)
				if !ok {
					break
				}
				blueByte, ok := safeReadFromArray(resultBytes, 2)
				if !ok {
					break
				}
				resultBytes = resultBytes[3:]
				resultImage.SetNRGBA(x, y, color.NRGBA{R: redByte, G: greenByte, B: blueByte, A: 255})
			}
		}
	}

	fmt.Printf("Result bytes length - %v\n\n", len(resultBytes))

	if _, err = result.Write(resultBytes); err != nil {
		logger.Errorf("Error writing result file: %v", err)
		return err
	}

	err = png.Encode(newResult, resultImage)
	if err != nil {
		logger.Errorf("Error encoding result image: %v", err)
		return fmt.Errorf("error encoding result image: %v", err)
	}

	return nil
}

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

func extractDataCount(RGBAImage *image.NRGBA) int {
	// We initialize a slice that's 12 bytes long, to be able to fit the bits we have to capture
	// since there are 2 bits per byte from the carrier
	dataCountBytes := make([]byte, 0, 12)

	// We want to start on the y-axis after the photoID and the photoNumber embedded information
	// We'll never increment the X as we only read the data from the first column
	x := 0
	count := 0
	for y := photoIDHeaderReservedPixels + photoNumberHeaderReservedPixels; count < dataSizeHeaderReservedPixels; y++ {
		c := RGBAImage.NRGBAAt(x, y)
		dataCountBytes = append(dataCountBytes, bit_manipulation.GetLastTwoBits(c.R), bit_manipulation.GetLastTwoBits(c.G), bit_manipulation.GetLastTwoBits(c.B))
		count++
	}

	// Even though we retrieve the data bytes in chunks of 3, when we construct it, we need to do it in chunks of 4.
	// This is because the dataCount number was stored consecutively.
	// That means for us to get a full 32-bit integer, we need to construct it from 4 bytes, not 3, and so we
	// need to add a 0 byte at the end.
	var bs = []byte{
		bit_manipulation.ConstructByteFromQuartersAsSlice(dataCountBytes[:4]),
		bit_manipulation.ConstructByteFromQuartersAsSlice(dataCountBytes[4:8]),
		bit_manipulation.ConstructByteFromQuartersAsSlice(dataCountBytes[8:]),
		byte(0),
	}

	return int(binary.LittleEndian.Uint32(bs))
}

func extractPhotoID(RGBAImage *image.NRGBA) int {
	photoIDBytes := make([]byte, 24)

	dx := RGBAImage.Bounds().Dx()
	dy := RGBAImage.Bounds().Dy()

	count := 0

	for x := 0; x < dx && count < photoIDHeaderReservedPixels; x++ {
		for y := 0; y < dy && count < photoIDHeaderReservedPixels; y++ {
			c := RGBAImage.NRGBAAt(x, y)
			photoIDBytes = append(photoIDBytes, bit_manipulation.GetLastTwoBits(c.R), bit_manipulation.GetLastTwoBits(c.G), bit_manipulation.GetLastTwoBits(c.B))
			count++
		}
	}
	// Not sure why this was here, it seems to just be adding one last byte of 0 - may need to do that if
	// the number doesn't come out to the total amount of bytes in the variable type
	//The little endian function should be able to return the number even if it isn't as long as it's supposed to be
	// photoIDBytes = append(photoIDBytes, byte(0), byte(0), byte(0))

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

func extractPhotoNumber(RGBAImage *image.NRGBA) int {
	photoNumberBytes := make([]byte, 4)

	dx := RGBAImage.Bounds().Dx()
	dy := RGBAImage.Bounds().Dy()

	count := 0

	for x := 0; x < dx && count < photoNumberHeaderReservedPixels; x++ {
		for y := photoIDHeaderReservedPixels; y < dy && count < photoNumberHeaderReservedPixels; y++ {
			c := RGBAImage.NRGBAAt(x, y)
			photoNumberBytes = append(photoNumberBytes, bit_manipulation.GetLastTwoBits(c.R), bit_manipulation.GetLastTwoBits(c.G), bit_manipulation.GetLastTwoBits(c.B))
			count++
		}
	}
	//This one only has 3 bytes worth of data, so we pad it with a zero byte to allow the quarters method to work
	photoNumberBytes = append(photoNumberBytes, byte(0))

	var bs = []byte{bit_manipulation.ConstructByteFromQuartersAsSlice(photoNumberBytes)}

	return int(binary.LittleEndian.Uint64(bs))
}

func safeReadFromArray(arr []byte, index int) (byte, bool) {
	if index >= 0 && index < len(arr) {
		return arr[index], true
	}
	return 0, false
}

package image_processing

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"go-steg/cli/helpers"
	"go-steg/go_steg/bit_manipulation"
	"go-steg/go_steg/logging"
	"go.uber.org/zap"
	"image"
	"image/draw"
	mathrand "math/rand"
	"path/filepath"
	"strings"

	"bytes"
	// Blank to justify
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
)

var logger *zap.SugaredLogger

// Mask - this struct will let us store information about the mask
type Mask struct {
	maskInt       int32
	multiplier    int32
	firstIndex    int16
	secondIndex   int16
	changeBoolean bool
}

func init() {
	logger = logging.NewLogger("")
}

// EncodeByFileNames will take in a list of carrier file names, a data image, and a list of the resulting image file names
func EncodeByFileNames(carrierFileNames []string, dataFileName string, uniquePhotoID uint64, password string, outputFileDir string) (err error) {
	return MultiCarrierEncodeByFileNames(carrierFileNames, dataFileName, uniquePhotoID, password, outputFileDir)
}

// MultiCarrierEncodeByFileNames takes in a series of files, a data file, and a series of strings to name the resulting files
// and passes everything to the Encode methods
func MultiCarrierEncodeByFileNames(
	carrierFileNames []string,
	dataFileName string,
	uniquePhotoID uint64,
	password string,
	outputFileDir string) (err error) {
	if len(carrierFileNames) == 0 {
		logger.Errorf("Missing carrier file names")
		return fmt.Errorf("missing carrier file names")
	}

	logger.Infof("Carrier file names: %v", carrierFileNames)
	logger.Infof("Data file name: %v", dataFileName)
	logger.Infof("Unique photo ID: %v", uniquePhotoID)
	logger.Infof("Number of carrier files: %v", len(carrierFileNames))

	//Make a slice to hold the names of the embedded carrier image files
	embeddedCarrierFileNames := make([]string, len(carrierFileNames))

	//Make a slice to hold the carrier files after they've been read/opened
	carriers := make([]io.Reader, 0, len(carrierFileNames))

	//Iterate through and open each carrier file
	for idx, name := range carrierFileNames {
		carrier, err := os.Open(name)
		fileName := filepath.Base(name)
		fileExtension := filepath.Ext(fileName)
		baseFileName := strings.TrimSuffix(fileName, fileExtension)
		embeddedCarrierName := fmt.Sprintf("%s/%s-%d-embedded%s", outputFileDir, baseFileName, idx, fileExtension)
		embeddedCarrierFileNames = append(embeddedCarrierFileNames, embeddedCarrierName)
		if err != nil {
			logger.Errorf("Error opening carrier file: %v", err)
		}
		defer func() {
			closeErr := carrier.Close()
			if err == nil {
				err = closeErr
			}
		}()
		if err != nil {
			logger.Errorf("Error closing the carrier file: %v", err)
		}
		carriers = append(carriers, carrier)
	}

	embedFile, err := os.Open(dataFileName)
	if err != nil {
		logger.Errorf("Error opening the data file: %v", err)
		return fmt.Errorf("error opening data file %s: %w", dataFileName, err)
	}
	defer func() {
		closeErr := embedFile.Close()
		if err == nil {
			err = closeErr
		}
	}()

	if err != nil {
		logger.Errorf("Error closing the data file: %v", err)
		return fmt.Errorf("issue closing the data file: %w", err)
	}

	//Make a slice of io writers in order to create the new files that are embedded
	embeddedCarrierWriters := make([]io.Writer, 0, len(embeddedCarrierFileNames[1:]))

	for _, name := range embeddedCarrierFileNames[1:] {
		result, err := os.Create(name)
		if err != nil {
			logger.Errorf("Error creating result file %s: %s", name, err)
			return fmt.Errorf("error creating result file %s: %w", name, err)
		}
		defer func() {
			closeErr := result.Close()
			if err == nil {
				err = closeErr
			}
		}()

		if err != nil {
			logger.Errorf("Error closing the carrier image: %s", err)
		}
		embeddedCarrierWriters = append(embeddedCarrierWriters, result)
	}

	//Here is where we encode the data into multiple carriers
	// If we receive an error, make sure to remove all the result files
	err = MultiCarrierEncode(carriers, embedFile, embeddedCarrierWriters, uniquePhotoID, password)
	if err != nil {
		for _, name := range embeddedCarrierFileNames {
			_ = os.Remove(name)
			logger.Errorf("Error encoding file name: %s, error: %v", name, err)
			return fmt.Errorf("issue encoding file %s: %w", name, err)
		}
	}
	return err
}

// MultiCarrierEncode will split the information into pieces and then use the encode
// function to encode that information into separate files.
// It does this by splitting the dataBytes reader into separate io.Readers based on how many
// carrier files there are
func MultiCarrierEncode(carriers []io.Reader, data io.Reader, results []io.Writer, uniquePhotoID uint64, password string) error {
	// Read all the data from the embed file
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("Error reading data %w\n", err)
	}

	//Make the chunk size the length of the byte slices divided by the number of carrier files
	chunkSize := len(dataBytes) / len(carriers)

	//Make a slice of readers with a starting length of 0 and a cap using the number of carrier files
	dataChunks := make([]io.Reader, 0, len(carriers))

	//Initialize a counter for the for loop in order to track the number of chunks put into encoder
	chunksCount := 0
	//Run a loop while the increment counter is less than the total length of the byte array
	// and the chunks count is less than the total number of carriers
	for i := 0; i < len(dataBytes) && chunksCount < len(carriers); i += chunkSize {
		chunksCount++
		//If the increment counter plus the chunkSize is larger than the length of the dataBytes slice,
		// or if the number of chunks already added is equal to the number of carriers, append whatever is left
		// of the data size file
		// TODO: Figure out why this is the last iteration - would it still go on to do the next step as well?
		if i+chunkSize >= len(dataBytes) || chunksCount == len(carriers) {
			dataChunks = append(dataChunks, bytes.NewReader(dataBytes[i:]))
		}
		dataChunks = append(dataChunks, bytes.NewReader(dataBytes[i:i+chunkSize]))
	}

	//Generate the mask information
	mask := generateMaskingInfo(password)

	fmt.Println("Masking info: ", mask)

	//Initialize a variable for the count since we need it to be an uint16
	var photoNumber uint16
	//Use another loop to actually encode everything
	for i := 0; i < len(carriers); i++ {
		if err := Encode(carriers[i], dataChunks[i], results[i], photoNumber, uniquePhotoID, mask); err != nil {
			return fmt.Errorf("error encoding chunk with index %d: %w", i, err)
		}
		photoNumber++
	}
	return err
}

// Encode will take in a carrier reader, data reader, and a result file writer and encode the data reader into the
// carrier, writing the result to the result file
//
// When encoding, we can use an indiscernability mask - this entails choosing two bits to function as the mask, and if
// the channel fits, change the last two bits.
// For example, consider a mask of 00[11] 0000 where the [] highlight the mask deciders and a change boolean of true.
// We then compare the mask to the channel/byte - 00[11] 0000 <=> 0010 0000
// In this case, the bits 11 and 10 are compared - 1 bit has changed,
// and so we change the last two bits of the channel.
// That means we use this channel/byte to hide information - if the data byte is 0000 0011, then our channel
// goes from 0010 0000 to 0010 0011
func Encode(carrier io.Reader, data io.Reader, result io.Writer, photoNumber uint16, uniquePhotoID uint64, mask Mask) error {
	// Open the carrier image as an RGBA image, along with getting the format of the carrier image
	RGBAImage, format, err := getImageAsRGBA(carrier)
	if err != nil {
		return fmt.Errorf("Error parsing carrier image: %w\n", err)
	}
	if format != "png" && format != "jpeg" {
		return fmt.Errorf("Unsupported carrier format\n")
	}

	//Open a buffered channel for the data - if the channel is full it will block until there's space
	//This makes an empty byte channel with a length of 128 bytes
	dataBytesChannel := make(chan byte, 128)

	//Open an unbuffered channel for errors we encounter
	errChannel := make(chan error)

	//Read the image data to make sure it's good and fill the channel
	go readData(data, dataBytesChannel, errChannel)

	//Set a boolean to tell if we have more data in the for loop
	hasMoreBytes := true

	//dataCount keeps track of the data size to store that information in the dataSizeHeader area of the carrier file
	var dataCount uint32

	//Get the bounds of the image
	bounds := RGBAImage.Bounds()

	if helpers.UseMask {
		//TODO: Occasionally the mask is too small - do we set a floor? Reduces the random range.
		openSlots := DetermineOpenSlotsWithMask(RGBAImage, bounds.Dx(), bounds.Dy(), mask)

		fmt.Printf("Number of slots availabe with mask: %v\n", openSlots)
	}

	// Iterate over every pixel starting at the 13th y pixel.
	// This gives room for the photo id and count to be stored.
	// Iterate over each pixel of the data we've received, top to bottom, left to right
	//	[
	//		1, 4, 7, 10,
	//		2, 5, 8, 11,
	//		3, 6, 9, 12
	//	]
	for x := 0; x < bounds.Dx() && hasMoreBytes; x++ {
		for y := totalReservedPixels; y < bounds.Dy() && hasMoreBytes; y++ {
			//Get the pixel at the current location
			c := RGBAImage.RGBAAt(x, y)
			if helpers.UseMask && bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.R) == mask.changeBoolean {
				//Set hasMoreBytes equal to the return of the setColorSegment method, which lets us know if there is more data
				hasMoreBytes, err = setColorSegment(&c.R, dataBytesChannel, errChannel)
				if err != nil {
					logger.Errorf("Error in setting red color segment: %v", err)
					return err
				}
				if hasMoreBytes {
					dataCount++
				}
			} else if !helpers.UseMask {
				hasMoreBytes, err = setColorSegment(&c.R, dataBytesChannel, errChannel)
				if err != nil {
					logger.Errorf("Error in setting red color segment: %v", err)
					return err
				}
				if hasMoreBytes {
					dataCount++
				}
			}
			if hasMoreBytes {
				if helpers.UseMask && bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.G) == mask.changeBoolean {
					hasMoreBytes, err = setColorSegment(&c.G, dataBytesChannel, errChannel)
					if err != nil {
						logger.Errorf("Error in setting green color segment: %v", err)
						return err
					}
					if hasMoreBytes {
						dataCount++
					}
				} else if !helpers.UseMask {
					hasMoreBytes, err = setColorSegment(&c.G, dataBytesChannel, errChannel)
					if err != nil {
						logger.Errorf("Error in setting green color segment: %v", err)
						return err
					}
					if hasMoreBytes {
						dataCount++
					}
				}
			}
			if hasMoreBytes {
				if helpers.UseMask && bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.B) == mask.changeBoolean {
					hasMoreBytes, err = setColorSegment(&c.B, dataBytesChannel, errChannel)
					if err != nil {
						logger.Errorf("Error in setting blue color segment: %v", err)
						return err
					}
					if hasMoreBytes {
						dataCount++
					}
				} else if !helpers.UseMask {
					hasMoreBytes, err = setColorSegment(&c.B, dataBytesChannel, errChannel)
					if err != nil {
						logger.Errorf("Error in setting blue color segment: %v", err)
						return err
					}
					if hasMoreBytes {
						dataCount++
					}
				}
			}
			//Set the pixel at this location equal to the former data with the last two bits of each channel modified
			RGBAImage.SetRGBA(x, y, c)

			if !hasMoreBytes {
				fmt.Printf("Last encoded pixel - (%v, %v)\n", x, y)
				openSlots := DetermineOpenSlotsWithMask(RGBAImage, bounds.Dx(), bounds.Dy(), mask)
				fmt.Printf("Number of slots availabe with mask: %v\n", openSlots)
			}
		}
	}
	fmt.Printf("Picture number - %v - Data count for encoding - %v\n\n", photoNumber, dataCount)
	//If we still have bytes left in the channel, we weren't able to fill the carrier file
	// TODO: investigate the default line here
	select {
	case _, ok := <-dataBytesChannel:
		if ok {
			fmt.Printf("Length of data left - %v\n", len(dataBytesChannel))
			return fmt.Errorf("Data file is too large for this carrier file\n")
		}
	default:
	}

	//Set the header area with the size of the data we just embedded into the carrier along with the ID and
	// the count/order of the photo. The ID comes from the database row number
	setHeaderInformation(RGBAImage, dataCount, uniquePhotoID, photoNumber)

	//If the format we get isn't either png or jpeg, return an error
	switch format {
	case "png", "jpeg":
		return png.Encode(result, RGBAImage)
	default:
		return fmt.Errorf("Unsupported carrier format\n")
	}
}

func setHeaderInformation(RGBAImage *image.RGBA, dataCount uint32, uniqueID uint64, photoNumber uint16) {
	// Set our counts to 0
	photoIdCount := 0
	photoNumberCount := 0
	dataCountBytesCount := 0
	//Get the bytes that make up each integer
	//The photo ID will be 48 bits, but we use a 64-bit variable as that's what the method calls for
	// 48 unsigned bits:
	// photoID - 0 - 281,474,976,710,655 or 281.474 trillion
	photoIDBytes := bit_manipulation.QuartersOfBytes64(uniqueID)
	//The number is only 6 bits, but the lowest we can go for the PutUint method is 16
	// photoNumber - 0 - 63
	photoNumberBytes := bit_manipulation.QuartersOfBytes16(photoNumber)
	//The data count can be up to 24 bits long, and we store it in a 32-bit unsigned integer
	// dataCount - 0 - 16,777,215 or 16.777 million
	dataCountBytes := bit_manipulation.QuartersOfBytes32(dataCount)

	x := 0
	for y := 0; y < totalReservedPixels; y++ {
		//For each pixel, get the channels at that pixel
		//Send each pixel to the set last two bits method to set the last two bits
		//We have to increment by 3 in order to have the right numbers for indexes in the byte arrays
		//This will use the first 8 pixels (0-7) in the y column to store the ID, the
		// next 1 pixel to store the order of the photo, and the last 4 pixels to store
		// the amount of data stored in this photo in bits
		c := RGBAImage.RGBAAt(x, y)
		if y < 8 {
			c.R = bit_manipulation.SetLastTwoBits(c.B, photoIDBytes[photoIdCount])
			c.G = bit_manipulation.SetLastTwoBits(c.B, photoIDBytes[photoIdCount+1])
			c.B = bit_manipulation.SetLastTwoBits(c.B, photoIDBytes[photoIdCount+2])
			photoIdCount += 3
		} else if y == 8 {
			c.R = bit_manipulation.SetLastTwoBits(c.B, photoNumberBytes[photoNumberCount])
			c.G = bit_manipulation.SetLastTwoBits(c.B, photoNumberBytes[photoNumberCount+1])
			c.B = bit_manipulation.SetLastTwoBits(c.B, photoNumberBytes[photoNumberCount+2])
			photoNumberCount += 3
		} else {
			c.R = bit_manipulation.SetLastTwoBits(c.B, dataCountBytes[dataCountBytesCount])
			c.G = bit_manipulation.SetLastTwoBits(c.B, dataCountBytes[dataCountBytesCount+1])
			c.B = bit_manipulation.SetLastTwoBits(c.B, dataCountBytes[dataCountBytesCount+2])
			dataCountBytesCount += 3
		}
		RGBAImage.SetRGBA(x, y, c)
	}

}

// setColorSegment will set the last two bits to the values pulled from the embed image.
// It takes in the pointer to the byte, a data channel with the bytes in it, and the error channel
// It returns hasMoreBytes, a simple boolean that checks to see if the dataChannel is empty
func setColorSegment(colorSegment *byte, dataChannel <-chan byte, errChan <-chan error) (hasMoreBytes bool, err error) {
	select {
	// Check the next byte in the dataChannel, if it's empty, return false to indicate there is no more data
	case chanByte, ok := <-dataChannel:
		if !ok {
			return false, nil
		}
		// If we're ok, set the last two bits of this segment to the bits of the embed image pulled from the channel
		// The byte being passed in as the value in this case is a byte that has the value needed right shifted,
		// or the equivalent of 0 padding the value, so it occupies the last two bits of the byte
		*colorSegment = bit_manipulation.SetLastTwoBits(*colorSegment, chanByte)
		// Return true because there is more data left
		return true, nil
	// If we have an error in the errorChannel, return false and the error itself to stop this operation
	case err := <-errChan:
		return false, err
	}
}

// This is reading the data of the input image into the bytesChannel - not sure why it's closed at the end,
// but it may just be so nothing else can get written to it by accident
// Also it looks like the close statement satisfies the return requirement, so whatever channel that's passed
// in is still accessible elsewhere
// Also, this loop will continue to run while the rest of the methods are running, providing data to the channel
// until EOF
func readData(reader io.Reader, bytesChannel chan<- byte, errChan chan<- error) {
	// Create a byte slice buffer with length of 1, so 8 bits total in the slice but in the first index
	byteArray := make([]byte, 1)
	for {
		// Iterate in an unending loop util EOF, where we break
		// The reader method will populate the byte slice with the number of bytes allowed until EOF
		// The method will return the number of bytes populated
		if _, err := reader.Read(byteArray); err != nil {
			if errors.Is(err, io.EOF) {
				// If we hit EOF, break the outer loop
				break
			}
			// Store the error if we see an error that isn't EOF
			errChan <- fmt.Errorf("error reading data %w", err)
		}
		// The length of the array is 4 elements, take each element and put it into the byte channel
		// This continues until EOF
		for _, byteArray := range bit_manipulation.SplitByteIntoQuarters(byteArray[0]) {
			bytesChannel <- byteArray
		}
	}
	// Close the channel as we shouldn't have to write anything else to it
	close(bytesChannel)
}

// This method receives a reader object and makes an image -
// it will always format the image as an RGBA image
func getImageAsRGBA(reader io.Reader) (*image.RGBA, string, error) {
	// Use the decode method to get the data from the reader object that was passed in
	// This includes the format of the image
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, format, fmt.Errorf("Error decoding carrier image: %v", err)
	}
	// Make a new RGBA image with the data, using the bounds of the image itself
	// TODO: here is where we may want to resize the image to some portion of the carrier image
	RGBAImage := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	// Draw the image out into the image object
	draw.Draw(RGBAImage, RGBAImage.Bounds(), img, img.Bounds().Min, draw.Src)
	// Return the RGBA image and the original format of the image
	return RGBAImage, format, nil
}

// DetermineOpenSlotsWithMask returns the number of open slots for a given carrier image by applying
// a mask
// Pass in the carrier information in order to determine how many open slots we have according
// to a certain mask - use the XOR operator to determine if they are different
// i.e. Mask = 00[10] 0000 - carrierByte = 0001 0000
// Initially going to just do a nested for loop, but may be a better way to handle this
func DetermineOpenSlotsWithMask(RGBAImage *image.RGBA, dx, dy int, mask Mask) (openSlotCount int64) {
	for x := 0; x < dx; x++ {
		for y := totalReservedPixels; y < dy; y++ {
			// For each pixel, use the mask to check if we can change it or not, skipping the first 0-12 that are used for
			// the header information
			c := RGBAImage.RGBAAt(x, y)
			if bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.R) == mask.changeBoolean {
				openSlotCount++
			}
			if bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.B) == mask.changeBoolean {
				openSlotCount++
			}
			if bit_manipulation.ReturnMaskDifference(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.G) == mask.changeBoolean {
				openSlotCount++
			}
		}
	}
	return openSlotCount
}

// generateMaskingInfo will generate a "mostly" (yeah, I know) cryptographically safe random number to use
// as a seed value for the random number generator in the math package, and then generate
// multiple random numbers to use for the masking operation.
// It uses the passed in password to generate a reproducible seed value for the random number generation.
//
// We then generate a random 32-bit number to serve as the mask, a random 32-bit number that is less than 8421504
// (the largest number that can be multiplied by the data byte value (max 255) and still not go over signed max)
// and two random numbers between 0 and 29 to serve as the index values to check for the mask.
// These index values will determine what indexes in the 32-bit mask will be used to determine
// if the data byte value will be changed or not.
func generateMaskingInfo(password string) Mask {
	//Make a slice of int16 numbers for our index compare
	var indexRange = make([]int16, 30)

	hashedPassword := hashPassword(password)

	seedValue := generateNumbersFromHash(hashedPassword)

	rng := mathrand.New(mathrand.NewSource(int64(seedValue)))

	// Create a range of numbers from 1-30, with indexes 0-29
	var i int16
	for i = 1; i < 31; i++ {
		indexRange[i-1] = i
	}
	//Generate a random 32-bit number
	mask := rng.Int31()
	// This is the largest number that can be multiplied by the data byte value and still not go over signed max
	// i.e. 8421504 * 255 < 2,147,483,647 (signed 32-bit max)
	multiplier := rng.Int31n(8421504)
	//Get a random number from 0-28 (array is 0-29 indexes, we want to exclude the last one for the further step)
	randomIndex := rng.Intn(28)
	// Get the first index we'll compare
	firstIndex := indexRange[randomIndex]
	// Set the value of the index that we just pulled to the value of the last index
	indexRange[randomIndex] = indexRange[29]
	//Remake the range of numbers to exclude the last value (slice is exclusive)
	indexRange = indexRange[:29]
	//Generate a random value from 0-28 and grab the second index we'll compare
	randomIndex = rng.Intn(28)
	secondIndex := indexRange[randomIndex]
	// Randomly decide the boolean value that we'll use to determine if we change the data byte value or not
	var changeBoolean bool
	if rng.Intn(2) == 1 {
		changeBoolean = true
	} else {
		changeBoolean = false
	}

	return Mask{mask, multiplier, firstIndex, secondIndex, changeBoolean}
}

// hashPassword will take in a password and hash it using the sha256 hashing algorithm
func hashPassword(password string) []byte {
	hashFunction := sha256.New()
	_, err := hashFunction.Write([]byte(password))
	if err != nil {
		logger.Errorf("Error hashing password: %v", err)
		panic(err)
	}

	return hashFunction.Sum(nil)
}

// generateNumbersFromHash will take in a hashed password and generate a seed value for the random number generator
// from the first eight bytes of the hash
func generateNumbersFromHash(hash []byte) uint64 {
	subHash := hash[:8]

	return binary.BigEndian.Uint64(subHash)
}

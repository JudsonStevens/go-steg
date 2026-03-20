package image_processing

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"go-steg/cli/helpers"
	"go-steg/go_steg/bit_manipulation"
	"go-steg/go_steg/logging"
	"go-steg/go_steg/pipeline"
	"hash/crc32"
	"image"
	"image/draw"
	"io"
	mathrand "math/rand"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	// Blank to justify
	_ "image/jpeg"
	"image/png"
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

type EncodingError struct {
	Type    string
	Message string
	Err     error
}

func (e *EncodingError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Error types
var (
	ErrCarrierTooSmall = &EncodingError{
		Type:    "CarrierSizeError",
		Message: "carrier image dimensions too small for data",
	}
	ErrInvalidFormat = &EncodingError{
		Type:    "FormatError",
		Message: "unsupported carrier image format, must be PNG or JPEG",
	}
	ErrDataTooLarge = &EncodingError{
		Type:    "DataSizeError",
		Message: "data file too large for carrier image capacity",
	}
	ErrHeaderSpace = &EncodingError{
		Type:    "HeaderError",
		Message: "insufficient space for header information",
	}
	ErrMaskGeneration = &EncodingError{
		Type:    "MaskError",
		Message: "error generating steganographic mask",
	}
	ErrIOOperation = &EncodingError{
		Type:    "IOError",
		Message: "error during file read/write operation",
	}
)

func wrapError(err error, errType *EncodingError, context string) error {
	return &EncodingError{
		Type:    errType.Type,
		Message: fmt.Sprintf("%s: %s", errType.Message, context),
		Err:     err,
	}
}

func init() {
	logger = logging.NewLogger("")
}

// computeChecksum computes a CRC-16 style checksum from pipeline output.
// Takes CRC-32 of first min(4, len) bytes and returns the low 12 bits.
func computeChecksum(data []byte) uint16 {
	n := len(data)
	if n > 4 {
		n = 4
	}
	crc := crc32.ChecksumIEEE(data[:n])
	return uint16(crc & 0x0FFF)
}

// EncodeByFileNames will take in a list of carrier file names, a data image, and a list of the resulting image file names
func EncodeByFileNames(carrierFileNames []string, dataFileName string, uniquePhotoID uint64, password string, outputFileDir string, cfg pipeline.Config) (err error) {
	return MultiCarrierEncodeByFileNames(carrierFileNames, dataFileName, uniquePhotoID, password, outputFileDir, cfg)
}

// MultiCarrierEncodeByFileNames takes in a series of files, a data file, and a series of strings to name the resulting files
// and passes everything to the Encode methods
func MultiCarrierEncodeByFileNames(
	carrierFileNames []string,
	dataFileName string,
	uniquePhotoID uint64,
	password string,
	outputFileDir string,
	cfg pipeline.Config) (err error) {
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
	err = MultiCarrierEncode(carriers, embedFile, embeddedCarrierWriters, uniquePhotoID, password, cfg)
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
func MultiCarrierEncode(carriers []io.Reader, data io.Reader, results []io.Writer, uniquePhotoID uint64, password string, cfg pipeline.Config) error {
	// Read all the data from the embed file
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("Error reading data %w\n", err)
	}

	// Run the pipeline encoding (huffman, reed-solomon, etc.)
	pipelineOutput, err := pipeline.Encode(dataBytes, cfg)
	if err != nil {
		return fmt.Errorf("error in pipeline encode: %w", err)
	}

	// Compute checksum and byte count modulo for the header
	checksum := computeChecksum(pipelineOutput)
	byteCountMod := uint16(len(pipelineOutput) % 4096)

	//Make the chunk size the length of the byte slices divided by the number of carrier files
	chunkSize := len(pipelineOutput) / len(carriers)

	//Make a slice of readers with a starting length of 0 and a cap using the number of carrier files
	dataChunks := make([]io.Reader, 0, len(carriers))

	//Initialize a counter for the for loop in order to track the number of chunks put into encoder
	chunksCount := 0
	//Run a loop while the increment counter is less than the total length of the byte array
	// and the chunks count is less than the total number of carriers
	for i := 0; i < len(pipelineOutput) && chunksCount < len(carriers); i += chunkSize {
		chunksCount++
		//If the increment counter plus the chunkSize is larger than the length of the pipelineOutput slice,
		// or if the number of chunks already added is equal to the number of carriers, append whatever is left
		if i+chunkSize >= len(pipelineOutput) || chunksCount == len(carriers) {
			dataChunks = append(dataChunks, bytes.NewReader(pipelineOutput[i:]))
		}
		dataChunks = append(dataChunks, bytes.NewReader(pipelineOutput[i:i+chunkSize]))
	}

	//Generate the mask information
	mask := generateMaskingInfo(password)

	fmt.Println("Masking info: ", mask)

	//Initialize a variable for the count since we need it to be an uint16
	var photoNumber uint16
	//Use another loop to actually encode everything
	for i := 0; i < len(carriers); i++ {
		if err := Encode(carriers[i], dataChunks[i], results[i], photoNumber, uniquePhotoID, mask, cfg, checksum, byteCountMod); err != nil {
			return fmt.Errorf("error encoding chunk with index %d: %w", i, err)
		}
		photoNumber++
	}
	return err
}

// Encode will take in a carrier reader, data reader, and a result file writer and encode the data reader into the
// carrier, writing the result to the result file
func Encode(carrier io.Reader, data io.Reader, result io.Writer, photoNumber uint16, uniquePhotoID uint64, mask Mask, cfg pipeline.Config, checksum uint16, byteCountMod uint16) error {
	bitDepth := cfg.BitDepth
	if bitDepth < 1 || bitDepth > 4 {
		bitDepth = 2
	}

	// Open the carrier image as an RGBA image, along with getting the format of the carrier image
	RGBAImage, format, err := getImageAsRGBA(carrier)
	if err != nil {
		return fmt.Errorf("Error parsing carrier image: %w\n", err)
	}
	if format != "png" && format != "jpeg" {
		return fmt.Errorf("Unsupported carrier format\n")
	}

	//Get the bounds of the image
	bounds := RGBAImage.Bounds()

	// Validate carrier height
	if bounds.Dy() < minCarrierHeight {
		return wrapError(nil, ErrCarrierTooSmall, fmt.Sprintf("carrier height %d < minimum %d", bounds.Dy(), minCarrierHeight))
	}

	//Open a buffered channel for the data - if the channel is full it will block until there's space
	dataBytesChannel := make(chan byte, 128)

	//Open an unbuffered channel for errors we encounter
	errChannel := make(chan error)

	//Read the image data to make sure it's good and fill the channel
	go readData(data, dataBytesChannel, errChannel, bitDepth)

	//Set a boolean to tell if we have more data in the for loop
	hasMoreBytes := true

	//dataCount keeps track of the data size to store that information in the header
	var dataCount uint32

	if helpers.UseMask {
		openSlots := DetermineOpenSlotsWithMask(RGBAImage, bounds.Dx(), bounds.Dy(), mask)
		fmt.Printf("Number of slots availabe with mask: %v\n", openSlots)
	}

	// Iterate over every pixel starting at the reserved header pixels.
	for x := 0; x < bounds.Dx() && hasMoreBytes; x++ {
		for y := totalReservedPixels; y < bounds.Dy() && hasMoreBytes; y++ {
			c := RGBAImage.RGBAAt(x, y)
			if helpers.UseMask && bit_manipulation.ReturnMaskDifferenceN(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.R, bitDepth) == mask.changeBoolean {
				hasMoreBytes, err = setColorSegment(&c.R, dataBytesChannel, errChannel, bitDepth)
				if err != nil {
					logger.Errorf("Error in setting red color segment: %v", err)
					return err
				}
				if hasMoreBytes {
					dataCount++
				}
			} else if !helpers.UseMask {
				hasMoreBytes, err = setColorSegment(&c.R, dataBytesChannel, errChannel, bitDepth)
				if err != nil {
					logger.Errorf("Error in setting red color segment: %v", err)
					return err
				}
				if hasMoreBytes {
					dataCount++
				}
			}
			if hasMoreBytes {
				if helpers.UseMask && bit_manipulation.ReturnMaskDifferenceN(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.G, bitDepth) == mask.changeBoolean {
					hasMoreBytes, err = setColorSegment(&c.G, dataBytesChannel, errChannel, bitDepth)
					if err != nil {
						logger.Errorf("Error in setting green color segment: %v", err)
						return err
					}
					if hasMoreBytes {
						dataCount++
					}
				} else if !helpers.UseMask {
					hasMoreBytes, err = setColorSegment(&c.G, dataBytesChannel, errChannel, bitDepth)
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
				if helpers.UseMask && bit_manipulation.ReturnMaskDifferenceN(mask.maskInt, mask.multiplier, mask.firstIndex, mask.secondIndex, c.B, bitDepth) == mask.changeBoolean {
					hasMoreBytes, err = setColorSegment(&c.B, dataBytesChannel, errChannel, bitDepth)
					if err != nil {
						logger.Errorf("Error in setting blue color segment: %v", err)
						return err
					}
					if hasMoreBytes {
						dataCount++
					}
				} else if !helpers.UseMask {
					hasMoreBytes, err = setColorSegment(&c.B, dataBytesChannel, errChannel, bitDepth)
					if err != nil {
						logger.Errorf("Error in setting blue color segment: %v", err)
						return err
					}
					if hasMoreBytes {
						dataCount++
					}
				}
			}
			RGBAImage.SetRGBA(x, y, c)

			if !hasMoreBytes {
				fmt.Printf("Last encoded pixel - (%v, %v)\n", x, y)
				openSlots := DetermineOpenSlotsWithMask(RGBAImage, bounds.Dx(), bounds.Dy(), mask)
				fmt.Printf("Number of slots availabe with mask: %v\n", openSlots)
			}
		}
	}
	fmt.Printf("Picture number - %v - Data count for encoding - %v\n\n", photoNumber, dataCount)

	select {
	case _, ok := <-dataBytesChannel:
		if ok {
			fmt.Printf("Length of data left - %v\n", len(dataBytesChannel))
			return fmt.Errorf("Data file is too large for this carrier file\n")
		}
	default:
	}

	// Write the new header with all metadata
	headerInfo := HeaderInfo{
		PhotoID:        uniquePhotoID,
		PhotoNumber:    photoNumber,
		DataCount:      dataCount,
		IsNewFormat:    true,
		FileExtension:  cfg.FileExtension,
		BitDepth:       bitDepth,
		HuffmanEnabled: cfg.HuffmanEnabled,
		RSEnabled:      cfg.RSEnabled,
		RSLevel:        cfg.RSLevel,
		Checksum:       checksum,
		ByteCountMod:   byteCountMod,
	}
	writeHeader(RGBAImage, headerInfo)

	switch format {
	case "png", "jpeg":
		return png.Encode(result, RGBAImage)
	default:
		return fmt.Errorf("Unsupported carrier format\n")
	}
}

// setColorSegment will set the last N bits to the values pulled from the embed image.
func setColorSegment(colorSegment *byte, dataChannel <-chan byte, errChan <-chan error, bitDepth int) (hasMoreBytes bool, err error) {
	select {
	case chanByte, ok := <-dataChannel:
		if !ok {
			return false, nil
		}
		*colorSegment = bit_manipulation.SetLastNBits(*colorSegment, chanByte, bitDepth)
		return true, nil
	case err := <-errChan:
		return false, err
	}
}

// readData reads the data from the reader and splits each byte into chunks based on bitDepth,
// sending the chunks through the bytesChannel.
func readData(reader io.Reader, bytesChannel chan<- byte, errChan chan<- error, bitDepth int) {
	byteArray := make([]byte, 1)
	for {
		if _, err := reader.Read(byteArray); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			errChan <- fmt.Errorf("error reading data %w", err)
		}
		for _, chunk := range bit_manipulation.SplitByte(byteArray[0], bitDepth) {
			bytesChannel <- chunk
		}
	}
	close(bytesChannel)
}

// getImageAsRGBA receives a reader object and makes an RGBA image
func getImageAsRGBA(reader io.Reader) (*image.RGBA, string, error) {
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, format, fmt.Errorf("Error decoding carrier image: %v", err)
	}
	RGBAImage := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(RGBAImage, RGBAImage.Bounds(), img, img.Bounds().Min, draw.Src)
	return RGBAImage, format, nil
}

// DetermineOpenSlotsWithMask returns the number of open slots for a given carrier image by applying a mask
func DetermineOpenSlotsWithMask(RGBAImage *image.RGBA, dx, dy int, mask Mask) (openSlotCount int64) {
	for x := 0; x < dx; x++ {
		for y := totalReservedPixels; y < dy; y++ {
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

// generateMaskingInfo will generate masking information from the password
func generateMaskingInfo(password string) Mask {
	var indexRange = make([]int16, 30)

	hashedPassword := hashPassword(password)
	seedValue := generateNumbersFromHash(hashedPassword)
	rng := mathrand.New(mathrand.NewSource(int64(seedValue)))

	var i int16
	for i = 1; i < 31; i++ {
		indexRange[i-1] = i
	}
	mask := rng.Int31()
	multiplier := rng.Int31n(8421504)
	randomIndex := rng.Intn(28)
	firstIndex := indexRange[randomIndex]
	indexRange[randomIndex] = indexRange[29]
	indexRange = indexRange[:29]
	randomIndex = rng.Intn(28)
	secondIndex := indexRange[randomIndex]
	var changeBoolean bool = false
	if rng.Intn(2) == 1 {
		changeBoolean = true
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

// generateNumbersFromHash will take in a hashed password and generate a seed value
func generateNumbersFromHash(hash []byte) uint64 {
	subHash := hash[:8]
	return binary.BigEndian.Uint64(subHash)
}

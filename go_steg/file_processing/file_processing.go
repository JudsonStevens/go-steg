package file_processing

import (
	"fmt"
	"github.com/google/uuid"
	"go-steg/go_steg/image_processing"
	"go.uber.org/zap"
	"mime/multipart"
)

// SaveFiles will save files from the requests in the main package
func SaveFiles(files []*multipart.FileHeader, logger zap.SugaredLogger, outputFileDir string) (fileNames []string, err error) {
	var count uint8

	for _, file := range files {
		fileName, err := SaveAndResizeFile(file, count, true, logger, outputFileDir)
		if err != nil {
			logger.Errorf("Error saving the carrier file - %v", err)
			return nil, err
		}

		fileNames = append(fileNames, fileName)
		count++
	}

	return fileNames, nil
}

// SaveAndResizeFile will save the file passed in from the requests in the main package and resize if necessary
func SaveAndResizeFile(file *multipart.FileHeader, fileCount uint8, carrierPhoto bool, logger zap.SugaredLogger, outputFileDir string) (carrierFileName string, err error) {
	receivedFile, err := file.Open()
	if err != nil {
		logger.Errorf("Error opening received file - %v", err)
		return "", fmt.Errorf("issue with opening a file: %v", err)
	}
	defer func(receivedFile multipart.File) {
		err := receivedFile.Close()
		if err != nil {
			logger.Errorf("Error closing received file - %v", err)
		}
	}(receivedFile)

	//Create a new UUID to save the file - this is mostly so files saved at the same time don't somehow conflict
	// over naming. TODO: Evaluate taking this out
	newUUID, err := uuid.NewRandom()
	if err != nil {
		logger.Errorf("Error with UUID creation: %v", err)
		return "", fmt.Errorf("issue with UUID creation: %v", err)
	}

	if carrierPhoto {
		resizedFilename, err := image_processing.ResizeCarrierImage(receivedFile, newUUID.String(), fileCount, outputFileDir)
		if err != nil {
			logger.Errorf("Error with resizing a file: %v", err)
			return "", fmt.Errorf("issue with resizing a file: %v", err)
		}
		return resizedFilename, nil
	}
	resizedFilename, err := image_processing.ResizeEmbedImage(receivedFile, newUUID.String(), outputFileDir)
	if err != nil {
		logger.Errorf("Error with resizing a file: %v", err)
		return "", fmt.Errorf("issue with resizing a file: %v", err)
	}
	return resizedFilename, nil
}

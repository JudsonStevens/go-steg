package helpers

import (
	"errors"
	"fmt"
	"os"
)

var UseMask bool

// ValidateIsValidDirectory checks if the directory path is valid and exists
func ValidateIsValidDirectory(directoryPath string) error {
	fmt.Println("Validating directory path: ", directoryPath)
	dir, err := os.Stat(directoryPath)
	if err != nil {
		return err
	}

	if !dir.IsDir() {
		return errors.New("the output directory must be a valid, existing directory")
	}

	return nil
}

// ValidateIsValidFile checks if the file path is valid and exists
func ValidateIsValidFile(filePath string) error {
	_, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	return nil
}

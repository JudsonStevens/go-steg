package image_processing

import (
	"fmt"
	"go-steg/go_steg/logging"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"os"

	"github.com/disintegration/imaging"
)

func init() {
	logger = logging.NewLogger("")
}

// ResizeEmbedImage will resize an embed image to something equal to or smaller than 540 x 675
// TODO: We need to check both dimensions in case the picture is formatted weirdly
func ResizeEmbedImage(file io.Reader, UUID string, outputFileDir string) (fileName string, err error) {
	//Get the image from the file passed in
	RGBAImage, _, err := getImageAsRGBA(file)
	if err != nil {
		logger.Errorf("Error getting the file as an image for file: %v", err)
		return "", fmt.Errorf("error getting the file as an image for file: %v", err)
	}

	//Create the new file name using the files count and the UUID
	embedFileName := fmt.Sprintf("%s/%v.png", outputFileDir, UUID)

	//Create the file we're going to save the image into, whether resized or not
	embedFile, err := os.Create(embedFileName)
	if err != nil {
		logger.Errorf("Error creating the new file: %v", err)
		return "", fmt.Errorf("error creating the new file: %v", err)
	}

	//Check the bounds of the image
	dx := RGBAImage.Bounds().Dx()
	dy := RGBAImage.Bounds().Dy()

	//Create a new RGBA image to store the resized image
	//TODO: Make sure we can't just resize into a regular RGBA?
	var resizedNRGBAImage *image.NRGBA
	if dx > instagramHalfMaxWidth && dy > instagramHalfMaxHeight {
		//Resize the image
		resizedNRGBAImage = imaging.Resize(RGBAImage, instagramHalfMaxWidth, instagramHalfMaxHeight, imaging.Lanczos)

		//Get the bounds of the new image
		bounds := resizedNRGBAImage.Bounds()
		dx = bounds.Dx()
		dy = bounds.Dy()

		//Create a blank RGBA image to draw the resized image into
		RGBAImage = image.NewRGBA(image.Rect(0, 0, dx, dy))
		draw.Draw(RGBAImage, RGBAImage.Bounds(), resizedNRGBAImage, bounds.Min, draw.Src)

	} else if dy > instagramHalfMaxHeight {
		//Resize the image
		resizedNRGBAImage = imaging.Resize(RGBAImage, 0, instagramHalfMaxHeight, imaging.Lanczos)

		//Get the bounds of the new image
		bounds := resizedNRGBAImage.Bounds()
		dx = bounds.Dx()
		dy = bounds.Dy()

		//Create a blank RGBA image to draw the resized image into
		RGBAImage = image.NewRGBA(image.Rect(0, 0, dx, dy))
		draw.Draw(RGBAImage, RGBAImage.Bounds(), resizedNRGBAImage, bounds.Min, draw.Src)
	} else if dx > instagramHalfMaxWidth {
		//Resize the image
		resizedNRGBAImage = imaging.Resize(RGBAImage, instagramHalfMaxWidth, 0, imaging.Lanczos)

		//Get the bounds of the new image
		bounds := resizedNRGBAImage.Bounds()
		dx = bounds.Dx()
		dy = bounds.Dy()

		//Create a blank RGBA image to draw the resized image into
		RGBAImage = image.NewRGBA(image.Rect(0, 0, dx, dy))
		draw.Draw(RGBAImage, RGBAImage.Bounds(), resizedNRGBAImage, bounds.Min, draw.Src)
	}

	fmt.Printf("New bounds are x - %v, y - %v\n", dx, dy)
	//Encode the file as a png and save it to the file name
	err = png.Encode(embedFile, RGBAImage)
	if err != nil {
		logger.Errorf("Issues saving the embed file: %v", err)
	}

	//Close the file since we've saved it
	// err = file.Close()
	// if err != nil {
	// 	log.Debug("Issue closing the embed file: %v", err)
	// }

	return embedFileName, err
}

// ResizeCarrierImage will resize a carrier image to something equal to or smaller than 1080x1350.
//
// This is the Instagram max size for the 4:5 ratio.
func ResizeCarrierImage(file io.Reader, UUID string, fileNumber uint8, outputFileDir string) (fileName string, err error) {
	//Get the image from the file passed in
	RGBAImage, _, err := getImageAsRGBA(file)
	if err != nil {
		logger.Errorf("Error getting the file as an image for file: %v", err)
		return "", fmt.Errorf("error getting the file as an image for file: %v", err)
	}

	//Create the new file name using the files count and the UUID
	carrierFileName := fmt.Sprintf("%s/%v-%v.png", outputFileDir, fileNumber, UUID)

	//Create the file we're going to save the image into, whether resized or not
	carrierFile, err := os.Create(carrierFileName)
	if err != nil {
		logger.Errorf("Error creating the new file: %v", err)
		return "", fmt.Errorf("error creating the new file: %v", err)
	}

	//Check the bounds of the image
	dx := RGBAImage.Bounds().Dx()
	dy := RGBAImage.Bounds().Dy()

	//Create a new NRGBA image to store the resized image
	//TODO: Make sure we can't just resize into a regular RGBA?
	var resizedNRGBAImage *image.NRGBA
	if dy > instagramMaxImageHeight && dx > instagramMaxImageWidth {
		//Resize the image in both dimensions
		resizedNRGBAImage = imaging.Resize(RGBAImage, instagramHalfMaxWidth, instagramHalfMaxHeight, imaging.Lanczos)
	} else if dy > instagramMaxImageHeight {
		//Resize the image in the y dimension while preserving the aspect ratio
		resizedNRGBAImage = imaging.Resize(RGBAImage, 0, instagramMaxImageHeight, imaging.Lanczos)
	} else if dx > instagramMaxImageWidth {
		//Resize the image in the x dimension while preserving the aspect ratio
		resizedNRGBAImage = imaging.Resize(RGBAImage, instagramMaxImageWidth, 0, imaging.Lanczos)
	} else {
		//The image is already small enough, so just use the original
		resizedNRGBAImage = imaging.Resize(RGBAImage, dx, dy, imaging.Lanczos)
	}
	//Get the bounds of the new image
	bounds := resizedNRGBAImage.Bounds()
	dx = bounds.Dx()
	dy = bounds.Dy()
	//Create a blank RGBA image to draw the resized image into
	RGBAImage = image.NewRGBA(image.Rect(0, 0, dx, dy))
	draw.Draw(RGBAImage, RGBAImage.Bounds(), resizedNRGBAImage, bounds.Min, draw.Src)

	// Encode the file as a png and save it to the file name
	// We use JPEG as Instagram recompresses/converts PNGs
	err = jpeg.Encode(carrierFile, RGBAImage, nil)
	if err != nil {
		logger.Errorf("issues saving the carrier file: %v", err)
	}
	return carrierFileName, err
}

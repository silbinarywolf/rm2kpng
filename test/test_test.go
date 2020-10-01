package rm2kpngtest

import (
	"fmt"
	"image"
	"os"
	"testing"

	"github.com/silbinarywolf/rm2kpng"
)

func convertImageByFilename(src string) (*image.Paletted, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	convertedImage, err := rm2kpng.ConvertToRm2kImage(srcFile)
	if err != nil {
		srcFile.Close()
		return nil, err
	}
	srcFile.Close()
	return convertedImage, nil
}

// todo(Jae): Write actual tests for the api
func TestData(t *testing.T) {
	image, err := convertImageByFilename("object3_taken_from_rm2k_survivor.png")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Palette %d, err: %v", len(image.Palette), err)
}

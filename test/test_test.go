package rm2kpngtest

import (
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
	convertedImage, err := rm2kpng.ConvertPNGToRm2kPNG(srcFile)
	if err != nil {
		srcFile.Close()
		return nil, err
	}
	srcFile.Close()
	return convertedImage, nil
}

func TestAlreadyValidData(t *testing.T) {
	_, err := convertImageByFilename("object3_taken_from_rm2k_survivor.png")
	if _, ok := err.(rm2kpng.ErrRm2kCompatiblePNG); !ok {
		t.Fail()
		return
	}
	//fmt.Printf("Palette %d, err: %v", len(image.Palette), err)
}

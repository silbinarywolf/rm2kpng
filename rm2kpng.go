package rm2kpng

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
)

// ErrRm2kCompatiblePNG is an error that occurs if you're trying to convert an already converted image
type ErrRm2kCompatiblePNG struct {
	paletteLen int
}

// PaletteLen is the size of the palette of the image you tried to convert
func (err ErrRm2kCompatiblePNG) PaletteLen() int {
	return err.paletteLen
}

func (err ErrRm2kCompatiblePNG) Error() string {
	if err.paletteLen == maxPaletteLen {
		return fmt.Sprintf("PNG is valid Rm2k png with %d colors.", err.paletteLen)
	}
	return fmt.Sprintf("PNG is valid Rm2k png with less than %d colors in its palette. It has %d.", maxPaletteLen, err.paletteLen)
}

const (
	// maxPaletteLen is the expected palette size of RPG Maker assets
	maxPaletteLen = 256
)

// getRm2kPaletteList will build a Rm2k-compatible palette for an image by
// looping over every pixel.
//
// An error is returned if there are more than 256 colors used.
func getRm2kPaletteList(src image.Image) (color.Palette, error) {
	paletteList := make(color.Palette, 0, maxPaletteLen)
	paletteMap := make(map[color.RGBA]int32)
	imageSize := src.Bounds().Size()

	// Detect type and first palette color (ie. transparency)
	if imageSize.X == 480 &&
		imageSize.Y == 256 {
		// Detect Rm2k/3 chipset
		const (
			transparentTileX = 296
			transparentTileY = 135
		)
		r, g, b, a := src.At(transparentTileX, transparentTileY).RGBA()
		rgba := color.RGBA{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
			A: uint8(a),
		}
		paletteList = append(paletteList, rgba)
		paletteMap[rgba]++
	} else {
		// *Assume* Charset/Picture/Faceset/etc and assume top-left pixel is transparent

		// note(Jae):
		// may want to improve this heuristic but for now lets do the simplest thing
		// might experiment with the idea that most-used pixel == transparent?
		//
		// i could also provide a config file users can place in their working directory
		// so that the transparent color used can be fixed to a specific color
		const (
			transparentX = 0
			transparentY = 0
		)
		r, g, b, a := src.At(transparentX, transparentY).RGBA()
		rgba := color.RGBA{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
			A: uint8(a),
		}
		paletteList = append(paletteList, rgba)
		paletteMap[rgba]++
	}

	// Loop over all the pixels and build a palette
	for y := 0; y < imageSize.Y; y++ {
		for x := 0; x < imageSize.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			rgba := color.RGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: uint8(a),
			}
			if _, ok := paletteMap[rgba]; ok {
				paletteMap[rgba]++
				continue
			}
			paletteList = append(paletteList, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
			paletteMap[rgba]++
		}
	}
	if len(paletteList) > maxPaletteLen {
		return nil, fmt.Errorf("Palette size is %d, which is too big", len(paletteList))
	}

	// NOTE(Jae): Most Rm2k assets I've seen have 256 colors in its palette
	// regardless of whether they're all used or not but our conversion process
	// can go lower and still work, so we leave it for now.
	// We may need to pad out the paletteList with 256 colors latter though.

	return paletteList, nil
}

func comparePixels(src image.Image, dest image.Image) error {
	srcImageSize := src.Bounds().Size()
	destImageSize := dest.Bounds().Size()
	if srcImageSize.X != destImageSize.X {
		return errors.New("Src and dest image do not match in width")
	}
	if srcImageSize.Y != destImageSize.Y {
		return errors.New("Src and dest image do not match in height")
	}
	for y := 0; y < srcImageSize.Y; y++ {
		for x := 0; x < srcImageSize.X; x++ {
			srcR, srcG, srcB, srcA := src.At(x, y).RGBA()
			destR, destG, destB, destA := dest.At(x, y).RGBA()
			if srcR == destR && srcG == destG && srcB == destB && srcA == destA {
				// if match, continue
				continue
			}
			return fmt.Errorf("source and destination do not match at pixel: %dx%d", x, y)
		}
	}
	return nil
}

// ConvertPNGToRm2kPNG will losslessly convert any PNG file
//
// If the PNG provided is already an 8-bit PNG that uses 256 colors or less in its
// palette, then return an error
func ConvertPNGToRm2kPNG(srcFile io.Reader) (*image.Paletted, error) {
	src, err := png.Decode(srcFile)
	if err != nil {
		return nil, err
	}
	if srcPaletted, ok := src.(*image.Paletted); ok {
		// NOTE(Jae): Most Rm2k assets I've seen have 256 colors in its palette
		// regardless of whether they're all used or not but our conversion process
		// can go lower and still work, so if this PNG uses 256 colors or less, do
		// not convert.
		if len(srcPaletted.Palette) <= maxPaletteLen {
			return nil, ErrRm2kCompatiblePNG{
				paletteLen: len(srcPaletted.Palette),
			}
		}
	}

	paletteList, err := getRm2kPaletteList(src)
	if err != nil {
		return nil, err
	}

	dst := image.NewPaletted(src.Bounds(), paletteList)
	drawer := draw.Drawer(draw.Src)
	// NOTE(Jae):
	// May want to provide the ability to do lossy conversion later?
	// It'd have to be explicit opt-in.
	//if dither {
	//	drawer = draw.FloydSteinberg
	//}
	drawer.Draw(dst, dst.Bounds(), src, src.Bounds().Min)

	// Sanity check that the image is the same pixel-by-pixel
	if err := comparePixels(src, dst); err != nil {
		return nil, err
	}

	return dst, nil
}

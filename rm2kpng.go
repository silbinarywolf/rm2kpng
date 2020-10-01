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

// RGBA uses uint32 naively because computers are fast enough for this tool to
// go fast for my use-case, who cares
type RGBA struct {
	R, G, B, A uint32
}

// getRm2kPaletteList will retrieve an Rm2k-compatible palette for an image
func getRm2kPaletteList(src image.Image) ([]color.Color, error) {
	paletteList := make([]color.Color, 0, 255)
	paletteMap := make(map[RGBA]int32)
	imageSize := src.Bounds()

	// Detect type and first palette color (ie. transparency)
	if imageSize.Size().X == 480 &&
		imageSize.Size().Y == 256 {
		// Detect Rm2k/3 chipset
		const (
			transparentTileX = 296
			transparentTileY = 135
		)
		r, g, b, a := src.At(transparentTileX, transparentTileY).RGBA()
		rgba := RGBA{
			R: r,
			G: g,
			B: b,
			A: a,
		}
		paletteList = append(paletteList, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
		paletteMap[rgba]++
	} else {
		// *Assume* Charset and assume top-left pixel is transparent
		// note(Jae): may want to improve this heuristic but for now lets do the simplest thing
		//			  might experiment with the idea that most-used pixel == transparent?
		const (
			transparentX = 0
			transparentY = 0
		)
		r, g, b, a := src.At(transparentX, transparentY).RGBA()
		rgba := RGBA{
			R: r,
			G: g,
			B: b,
			A: a,
		}
		paletteList = append(paletteList, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
		paletteMap[rgba]++
	}

	for y := 0; y < imageSize.Size().Y; y++ {
		for x := 0; x < imageSize.Size().X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			rgba := RGBA{
				R: r,
				G: g,
				B: b,
				A: a,
			}
			if _, ok := paletteMap[rgba]; ok {
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

func ConvertToRm2kImage(srcFile io.Reader) (*image.Paletted, error) {
	src, err := png.Decode(srcFile)
	if err != nil {
		return nil, err
	}
	if srcPaletted, ok := src.(*image.Paletted); ok {
		// NOTE(Jae): Most Rm2k assets I've seen have 256 colors in its palette
		// regardless of whether they're all used or not but our conversion process
		// can go lower and still work, so we leave it for now.
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

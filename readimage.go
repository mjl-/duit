package duit

import (
	"fmt"
	"image"
	imagedraw "image/draw"
	"io"
	"os"

	"9fans.net/go/draw"
)

// ReadImage decodes an image from f for use on display. The returned image is ready for use in an Image UI.
func ReadImage(display *draw.Display, f io.Reader) (*draw.Image, error) {
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %s", err)
	}
	var rgba *image.RGBA
	switch i := img.(type) {
	case *image.RGBA:
		rgba = i
	default:
		b := img.Bounds()
		rgba = image.NewRGBA(image.Rectangle{image.ZP, b.Size()})
		imagedraw.Draw(rgba, rgba.Bounds(), img, b.Min, imagedraw.Src)
	}

	// todo: package image claims data is in r,g,b,a.  who is reversing the bytes? devdraw? will this work on big endian?
	ni, err := display.AllocImage(rgba.Bounds(), draw.ABGR32, false, draw.White)
	if err != nil {
		return nil, fmt.Errorf("allocimage: %s", err)
	}
	_, err = ni.Load(rgba.Bounds(), rgba.Pix)
	if err != nil {
		return nil, fmt.Errorf("load image: %s", err)
	}
	return ni, nil
}

// ReadImagePath is a convenience function that opens path and calls ReadImage.
func ReadImagePath(display *draw.Display, path string) (*draw.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %s", path, err)
	}
	defer f.Close()
	return ReadImage(display, f)
}

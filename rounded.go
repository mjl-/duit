package duit

import (
	"image"

	"9fans.net/go/draw"
)

// draw border with rounded corners, on the inside of `r`.
func drawRoundedBorder(img *draw.Image, r image.Rectangle, color *draw.Image) {
	offset := 2
	radius := 2 * offset
	x0 := r.Min.X
	x1 := r.Max.X
	y0 := r.Min.Y
	y1 := r.Max.Y
	tl := image.Pt(x0+radius, y0+radius)
	bl := image.Pt(x0+radius, y1-radius)
	br := image.Pt(x1-radius, y1-radius)
	tr := image.Pt(x1-radius, y0+radius)
	img.Arc(tl, radius, radius, 0, color, image.ZP, 90, 90)
	img.Arc(bl, radius, radius, 0, color, image.ZP, 180, 90)
	img.Arc(br, radius, radius, 0, color, image.ZP, 270, 90)
	img.Arc(tr, radius, radius, 0, color, image.ZP, 0, 90)
	img.Line(image.Pt(x0, y0+offset), image.Pt(x0, y1-offset), 0, 0, 0, color, image.ZP)
	img.Line(image.Pt(x0+offset, y1), image.Pt(x1-offset, y1), 0, 0, 0, color, image.ZP)
	img.Line(image.Pt(x1, y1-offset), image.Pt(x1, y0+offset), 0, 0, 0, color, image.ZP)
	img.Line(image.Pt(x1-offset, y0), image.Pt(x0+offset, y0), 0, 0, 0, color, image.ZP)
}

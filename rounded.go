package duit

import (
	"image"

	"9fans.net/go/draw"
)

// draw border with rounded corners, on the inside of `r`.
func drawRoundedBorder(img *draw.Image, r image.Rectangle, color *draw.Image) {
	radius := 3
	x0 := r.Min.X
	x1 := r.Max.X - 1
	y0 := r.Min.Y
	y1 := r.Max.Y - 1
	tl := image.Pt(x0+radius, y0+radius)
	bl := image.Pt(x0+radius, y1-radius)
	br := image.Pt(x1-radius, y1-radius)
	tr := image.Pt(x1-radius, y0+radius)
	img.Arc(tl, radius, radius, 0, color, image.ZP, 90, 90)
	img.Arc(bl, radius, radius, 0, color, image.ZP, 180, 90)
	img.Arc(br, radius, radius, 0, color, image.ZP, 270, 90)
	img.Arc(tr, radius, radius, 0, color, image.ZP, 0, 90)
	img.Line(image.Pt(x0, y0+radius), image.Pt(x0, y1-radius), 0, 0, 0, color, image.ZP)
	img.Line(image.Pt(x0+radius, y1), image.Pt(x1-radius, y1), 0, 0, 0, color, image.ZP)
	img.Line(image.Pt(x1, y1-radius), image.Pt(x1, y0+radius), 0, 0, 0, color, image.ZP)
	img.Line(image.Pt(x1-radius, y0), image.Pt(x0+radius, y0), 0, 0, 0, color, image.ZP)
}

package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Image struct {
	Image *draw.Image
}

func (ui *Image) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	return ui.Image.R.Size()
}

func (ui *Image) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(image.Rectangle{orig, orig.Add(ui.Image.R.Size())}, ui.Image, nil, image.ZP)
}

func (ui *Image) Mouse(m draw.Mouse) Result {
	return Result{Hit: ui}
}

func (ui *Image) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{Hit: ui}
}

func (ui *Image) FirstFocus() *image.Point {
	return nil
}

func (ui *Image) Focus(o UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

func (ui *Image) Print(indent int, r image.Rectangle) {
	uiPrint("Image", indent, r)
}

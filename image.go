package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Image struct {
	Image *draw.Image
}

var _ UI = &Image{}

func (ui *Image) Layout(dui *DUI, size image.Point) image.Point {
	if ui.Image == nil {
		return image.ZP
	}
	return ui.Image.R.Size()
}

func (ui *Image) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	if ui.Image == nil {
		return
	}
	img.Draw(image.Rectangle{orig, orig.Add(ui.Image.R.Size())}, ui.Image, nil, image.ZP)
}

func (ui *Image) Mouse(dui *DUI, origM, m draw.Mouse) Result {
	return Result{Hit: ui}
}

func (ui *Image) Key(dui *DUI, orig image.Point, m draw.Mouse, c rune) Result {
	return Result{Hit: ui}
}

func (ui *Image) FirstFocus(dui *DUI) *image.Point {
	return nil
}

func (ui *Image) Focus(dui *DUI, o UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

func (ui *Image) Print(indent int, r image.Rectangle) {
	PrintUI("Image", indent, r)
}

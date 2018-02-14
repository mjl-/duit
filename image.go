package duit

import (
	"image"

	"9fans.net/go/draw"
)

// Image shows an image. Currently always in its original size.
type Image struct {
	Image *draw.Image `json:"-"`
}

var _ UI = &Image{}

func (ui *Image) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	if ui.Image == nil {
		self.R = image.ZR
	} else {
		self.R = rect(ui.Image.R.Size())
	}
	return
}

func (ui *Image) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)
	if ui.Image == nil {
		return
	}
	img.Draw(ui.Image.R.Add(orig), ui.Image, nil, image.ZP)
}

func (ui *Image) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	return
}

func (ui *Image) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return
}

func (ui *Image) FirstFocus(dui *DUI, self *Kid) *image.Point {
	return nil
}

func (ui *Image) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

func (ui *Image) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Image) Print(self *Kid, indent int) {
	PrintUI("Image", self, indent)
}

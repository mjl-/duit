package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Horizontal struct {
	Split func(width int) (widths []int)
	Kids  []*Kid

	size   image.Point
	widths []int
}

var _ UI = &Horizontal{}

func (ui *Horizontal) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("Horizontal", self)
	if kidsLayout(dui, self, ui.Kids, force) {
		return
	}

	ui.widths = ui.Split(sizeAvail.X)
	if len(ui.widths) != len(ui.Kids) {
		panic("bad number of widths from split")
	}
	ui.size = image.ZP
	for i, k := range ui.Kids {
		k.UI.Layout(dui, k, image.Pt(ui.widths[i], sizeAvail.Y), true)
		p := image.Pt(ui.size.X, 0)
		k.R = k.R.Add(p)
		ui.size.X += ui.widths[i]
		if k.R.Dy() > ui.size.Y {
			ui.size.Y = k.R.Dy()
		}
	}
	self.R = rect(ui.size)
}

func (ui *Horizontal) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	kidsDraw("Horizontal", dui, self, ui.Kids, ui.size, img, orig, m, force)
}

func (ui *Horizontal) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	return kidsMouse(dui, self, ui.Kids, m, origM, orig)
}

func (ui *Horizontal) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return kidsKey(dui, self, ui.Kids, k, m, orig)
}

func (ui *Horizontal) FirstFocus(dui *DUI) *image.Point {
	return kidsFirstFocus(dui, ui.Kids)
}

func (ui *Horizontal) Focus(dui *DUI, o UI) *image.Point {
	return kidsFocus(dui, ui.Kids, o)
}

func (ui *Horizontal) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return kidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Horizontal) Print(self *Kid, indent int) {
	PrintUI("Horizontal", self, indent)
	kidsPrint(ui.Kids, indent+1)
}

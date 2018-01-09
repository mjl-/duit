package duit

import (
	"image"

	"9fans.net/go/draw"
)

// Middle lays out a single child in the middle of the available space, both vertically and horizontally.
type Middle struct {
	Kid *Kid

	kids []*Kid
	size image.Point
}

func NewMiddle(ui UI) *Middle {
	return &Middle{Kid: &Kid{UI: ui}}
}

func (ui *Middle) ensure() {
	if len(ui.kids) != 1 {
		ui.kids = make([]*Kid, 1)
	}
	ui.kids[0] = ui.Kid
}

func (ui *Middle) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	ui.ensure()
	dui.debugLayout("Middle", self)

	if kidsLayout(dui, self, ui.kids, force) {
		return
	}

	ui.Kid.UI.Layout(dui, ui.Kid, sizeAvail, true)
	left := sizeAvail.Sub(ui.Kid.R.Size())
	ui.Kid.R = ui.Kid.R.Add(image.Pt(maximum(0, left.X/2), maximum(0, left.Y/2)))
	ui.size = image.Pt(maximum(ui.Kid.R.Dx(), sizeAvail.X), maximum(ui.Kid.R.Dy(), sizeAvail.Y))
	self.R = rect(ui.size)
}

func (ui *Middle) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	ui.ensure()
	dui.debugDraw("Middle", self)
	kidsDraw("Middle", dui, self, ui.kids, ui.size, img, orig, m, force)
}

func (ui *Middle) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	ui.ensure()
	return kidsMouse(dui, self, ui.kids, m, origM, orig)
}

func (ui *Middle) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	ui.ensure()
	return kidsKey(dui, self, ui.kids, k, m, orig)
}

func (ui *Middle) FirstFocus(dui *DUI) (warp *image.Point) {
	ui.ensure()
	return kidsFirstFocus(dui, ui.kids)
}

func (ui *Middle) Focus(dui *DUI, o UI) (warp *image.Point) {
	ui.ensure()
	return kidsFocus(dui, ui.kids, o)
}

func (ui *Middle) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	ui.ensure()
	return kidsMark(self, ui.kids, o, forLayout)
}

func (ui *Middle) Print(self *Kid, indent int) {
	ui.ensure()
	PrintUI("Middle", self, indent)
	kidsPrint(ui.kids, indent+1)
}

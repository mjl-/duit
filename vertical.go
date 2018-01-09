package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Vertical struct {
	Split func(height int) (heights []int) `json:"-"`
	Kids  []*Kid

	size    image.Point
	heights []int
}

var _ UI = &Vertical{}

func (ui *Vertical) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("Vertical", self)
	if kidsLayout(dui, self, ui.Kids, force) {
		return
	}

	heights := ui.Split(sizeAvail.Y)
	if len(heights) != len(ui.Kids) {
		panic("bad number of heights from split")
	}
	ui.heights = heights
	cur := image.ZP
	for i, k := range ui.Kids {
		k.UI.Layout(dui, k, image.Pt(sizeAvail.X, heights[i]), true)
		k.R = k.R.Add(cur)
		cur.Y += heights[i]
	}
	ui.size = image.Pt(sizeAvail.X, cur.Y)
	self.R = rect(ui.size)
}

func (ui *Vertical) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	kidsDraw("Vertical", dui, self, ui.Kids, ui.size, img, orig, m, force)
}

func (ui *Vertical) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	return kidsMouse(dui, self, ui.Kids, m, origM, orig)
}

func (ui *Vertical) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return kidsKey(dui, self, ui.Kids, k, m, orig)
}

func (ui *Vertical) FirstFocus(dui *DUI) *image.Point {
	return kidsFirstFocus(dui, ui.Kids)
}

func (ui *Vertical) Focus(dui *DUI, o UI) *image.Point {
	return kidsFocus(dui, ui.Kids, o)
}

func (ui *Vertical) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return kidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Vertical) Print(self *Kid, indent int) {
	PrintUI("Vertical", self, indent)
	kidsPrint(ui.Kids, indent+1)
}

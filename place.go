package duit

import (
	"image"

	"9fans.net/go/draw"
)

// Place contains other UIs it can position absolute, possibly on top of each other.
type Place struct {
	// Place is called during layout. It must configure Kids, and set self.R, based on sizeAvail.
	Place      func(self *Kid, sizeAvail image.Point) `json:"-"`
	Kids       []*Kid                                 // Kids to draw, set by the Place function.
	Background *draw.Image                            `json:"-"` // For background color.

	kidsReversed []*Kid
	size         image.Point
}

var _ UI = &Place{}

func (ui *Place) ensure() {
	if len(ui.kidsReversed) == len(ui.Kids) {
		return
	}
	ui.kidsReversed = make([]*Kid, len(ui.Kids))
	for i, k := range ui.Kids {
		ui.kidsReversed[len(ui.Kids)-1-i] = k
	}
}

func (ui *Place) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	ui.ensure()
	dui.debugLayout(self)

	ui.Place(self, sizeAvail)
}

func (ui *Place) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	// xxx place should copies of its kids images, so it doesn't have to ask them to redraw all the time
	if self.Draw == DirtyKid {
		force = true
	}
	KidsDraw(dui, self, ui.Kids, ui.size, ui.Background, img, orig, m, force)
}

func (ui *Place) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	return KidsMouse(dui, self, ui.kidsReversed, m, origM, orig)
}

func (ui *Place) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return KidsKey(dui, self, ui.kidsReversed, k, m, orig)
}

func (ui *Place) FirstFocus(dui *DUI, self *Kid) (warp *image.Point) {
	return KidsFirstFocus(dui, self, ui.Kids)
}

func (ui *Place) Focus(dui *DUI, self *Kid, o UI) (warp *image.Point) {
	return KidsFocus(dui, self, ui.Kids, o)
}

func (ui *Place) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return KidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Place) Print(self *Kid, indent int) {
	PrintUI("Place", self, indent)
	KidsPrint(ui.Kids, indent+1)
}

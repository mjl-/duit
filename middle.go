package duit

import (
	"image"

	"9fans.net/go/draw"
)

// Middle lays out a single child in the middle of the available space, both vertically and horizontally.
type Middle struct {
	UI UI

	kids []*Kid
	size image.Point
}

func NewMiddle(ui UI) *Middle {
	return &Middle{UI: ui, kids: NewKids(ui)}
}

func (ui *Middle) ensure() {
	if len(ui.kids) != 1 || ui.kids[0].UI != ui.UI {
		ui.kids = NewKids(ui.UI)
	}
}

func (ui *Middle) Layout(dui *DUI, sizeAvail image.Point) (sizeTaken image.Point) {
	size := ui.UI.Layout(dui, sizeAvail)
	left := sizeAvail.Sub(size)
	ui.kids[0].R = rect(size).Add(image.Pt(maximum(0, left.X/2), maximum(0, left.Y/2)))
	ui.size = image.Pt(maximum(size.X, sizeAvail.X), maximum(size.Y, sizeAvail.Y))
	return ui.size
}

func (ui *Middle) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(dui, ui.kids, ui.size, img, orig, m)
}

func (ui *Middle) Mouse(dui *DUI, m draw.Mouse, origM draw.Mouse) (r Result) {
	return kidsMouse(dui, ui.kids, m, origM)
}

func (ui *Middle) Key(dui *DUI, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return kidsKey(dui, ui, ui.kids, k, m, orig)
}

func (ui *Middle) FirstFocus(dui *DUI) (warp *image.Point) {
	return kidsFirstFocus(dui, ui.kids)
}

func (ui *Middle) Focus(dui *DUI, o UI) (warp *image.Point) {
	return kidsFocus(dui, ui.kids, o)
}

func (ui *Middle) Print(indent int, r image.Rectangle) {
	PrintUI("Middle", indent, r)
	kidsPrint(ui.kids, indent+1)
}

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

func (ui *Middle) Layout(env *Env, sizeAvail image.Point) (sizeTaken image.Point) {
	size := ui.UI.Layout(env, sizeAvail)
	left := sizeAvail.Sub(size)
	ui.kids[0].r = rect(size).Add(image.Pt(maximum(0, left.X/2), maximum(0, left.Y/2)))
	ui.size = image.Pt(maximum(size.X, sizeAvail.X), maximum(size.Y, sizeAvail.Y))
	return ui.size
}

func (ui *Middle) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(env, ui.kids, ui.size, img, orig, m)
}

func (ui *Middle) Mouse(env *Env, origM, m draw.Mouse) (r Result) {
	return kidsMouse(env, ui.kids, origM, m)
}

func (ui *Middle) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result) {
	return kidsKey(env, ui, ui.kids, orig, m, k)
}

func (ui *Middle) FirstFocus(env *Env) (warp *image.Point) {
	return kidsFirstFocus(env, ui.kids)
}

func (ui *Middle) Focus(env *Env, o UI) (warp *image.Point) {
	return kidsFocus(env, ui.kids, o)
}

func (ui *Middle) Print(indent int, r image.Rectangle) {
	uiPrint("Middle", indent, r)
	kidsPrint(ui.kids, indent+1)
}

package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Vertical struct {
	Split func(height int) (heights []int)
	Kids  []*Kid

	size    image.Point
	heights []int
}

var _ UI = &Vertical{}

func (ui *Vertical) Layout(dui *DUI, size image.Point) image.Point {
	heights := ui.Split(size.Y)
	if len(heights) != len(ui.Kids) {
		panic("bad number of heights from split")
	}
	ui.heights = heights
	cur := image.ZP
	for i, k := range ui.Kids {
		childSize := k.UI.Layout(dui, image.Pt(size.X, heights[i]))
		k.R = rect(childSize).Add(cur)
		cur.Y += heights[i]
	}
	ui.size = image.Pt(size.X, cur.Y)
	return ui.size
}

func (ui *Vertical) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(dui, ui.Kids, ui.size, img, orig, m)
}

func (ui *Vertical) Mouse(dui *DUI, m draw.Mouse, origM draw.Mouse) (r Result) {
	return kidsMouse(dui, ui.Kids, m, origM)
}

func (ui *Vertical) Key(dui *DUI, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return kidsKey(dui, ui, ui.Kids, k, m, orig)
}

func (ui *Vertical) FirstFocus(dui *DUI) *image.Point {
	return kidsFirstFocus(dui, ui.Kids)
}

func (ui *Vertical) Focus(dui *DUI, o UI) *image.Point {
	return kidsFocus(dui, ui.Kids, o)
}

func (ui *Vertical) Print(indent int, r image.Rectangle) {
	PrintUI("Vertical", indent, r)
	kidsPrint(ui.Kids, indent+1)
}

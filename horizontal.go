package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Horizontal struct {
	Kids  []*Kid
	Split func(width int) (widths []int)

	size   image.Point
	widths []int
}

var _ UI = &Horizontal{}

func (ui *Horizontal) Layout(dui *DUI, size image.Point) image.Point {
	ui.widths = ui.Split(size.X)
	if len(ui.widths) != len(ui.Kids) {
		panic("bad number of widths from split")
	}
	ui.size = image.ZP
	for i, k := range ui.Kids {
		childSize := k.UI.Layout(dui, image.Pt(ui.widths[i], size.Y))
		p := image.Pt(ui.size.X, 0)
		k.R = image.Rectangle{p, p.Add(childSize)}
		ui.size.X += ui.widths[i]
		if k.R.Dy() > ui.size.Y {
			ui.size.Y = k.R.Dy()
		}
	}
	return ui.size
}

func (ui *Horizontal) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(dui, ui.Kids, ui.size, img, orig, m)
}

func (ui *Horizontal) Mouse(dui *DUI, origM, m draw.Mouse) (result Result) {
	return kidsMouse(dui, ui.Kids, origM, m)
}

func (ui *Horizontal) Key(dui *DUI, orig image.Point, m draw.Mouse, k rune) (result Result) {
	return kidsKey(dui, ui, ui.Kids, orig, m, k)
}

func (ui *Horizontal) FirstFocus(dui *DUI) *image.Point {
	return kidsFirstFocus(dui, ui.Kids)
}

func (ui *Horizontal) Focus(dui *DUI, o UI) *image.Point {
	return kidsFocus(dui, ui.Kids, o)
}

func (ui *Horizontal) Print(indent int, r image.Rectangle) {
	PrintUI("Horizontal", indent, r)
	kidsPrint(ui.Kids, indent+1)
}

package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Place struct {
	Place func(sizeAvail image.Point) (sizeTaken image.Point)
	Kids  []*Kid

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

func (ui *Place) Layout(dui *DUI, sizeAvail image.Point) (sizeTaken image.Point) {
	ui.ensure()
	ui.size = ui.Place(sizeAvail)
	return ui.size
}

func (ui *Place) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(dui, ui.Kids, ui.size, img, orig, m)
}

func (ui *Place) Mouse(dui *DUI, origM, m draw.Mouse) (r Result) {
	return kidsMouse(dui, ui.kidsReversed, origM, m)
}

func (ui *Place) Key(dui *DUI, orig image.Point, m draw.Mouse, k rune) (r Result) {
	return kidsKey(dui, ui, ui.kidsReversed, orig, m, k)
}

func (ui *Place) FirstFocus(dui *DUI) (warp *image.Point) {
	return kidsFirstFocus(dui, ui.Kids)
}

func (ui *Place) Focus(dui *DUI, o UI) (warp *image.Point) {
	return kidsFocus(dui, ui.Kids, o)
}

func (ui *Place) Print(indent int, r image.Rectangle) {
	PrintUI("Place", indent, r)
	kidsPrint(ui.Kids, indent+1)
}

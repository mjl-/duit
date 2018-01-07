package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Pick struct {
	Pick func(sizeAvail image.Point) UI

	ui UI
}

func (ui *Pick) Layout(dui *DUI, sizeAvail image.Point) (sizeTaken image.Point) {
	ui.ui = ui.Pick(sizeAvail)
	return ui.ui.Layout(dui, sizeAvail)
}

func (ui *Pick) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	ui.ui.Draw(dui, img, orig, m)
}

func (ui *Pick) Mouse(dui *DUI, m draw.Mouse, origM draw.Mouse) (r Result) {
	return ui.ui.Mouse(dui, m, origM)
}

func (ui *Pick) Key(dui *DUI, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return ui.ui.Key(dui, k, m, orig)
}

func (ui *Pick) FirstFocus(dui *DUI) (warp *image.Point) {
	return ui.ui.FirstFocus(dui)
}

func (ui *Pick) Focus(dui *DUI, o UI) (warp *image.Point) {
	return ui.ui.Focus(dui, o)
}

func (ui *Pick) Print(indent int, r image.Rectangle) {
	PrintUI("Pick", indent, r)
	ui.ui.Print(indent+1, r)
}

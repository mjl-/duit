package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Pick struct {
	Pick func(sizeAvail image.Point) UI
	UI   UI
}

func (ui *Pick) Layout(dui *DUI, sizeAvail image.Point) (sizeTaken image.Point) {
	ui.UI = ui.Pick(sizeAvail)
	return ui.UI.Layout(dui, sizeAvail)
}

func (ui *Pick) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	ui.UI.Draw(dui, img, orig, m)
}

func (ui *Pick) Mouse(dui *DUI, origM, m draw.Mouse) (r Result) {
	return ui.UI.Mouse(dui, origM, m)
}

func (ui *Pick) Key(dui *DUI, orig image.Point, m draw.Mouse, k rune) (r Result) {
	return ui.UI.Key(dui, orig, m, k)
}

func (ui *Pick) FirstFocus(dui *DUI) (warp *image.Point) {
	return ui.UI.FirstFocus(dui)
}

func (ui *Pick) Focus(dui *DUI, o UI) (warp *image.Point) {
	return ui.UI.Focus(dui, o)
}

func (ui *Pick) Print(indent int, r image.Rectangle) {
	PrintUI("Pick", indent, r)
	ui.UI.Print(indent+1, r)
}

package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Label struct {
	Text string
	Font *draw.Font
}

var _ UI = &Label{}

func (ui *Label) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Label) Layout(dui *DUI, size image.Point) image.Point {
	return ui.font(dui).StringSize(ui.Text)
}

func (ui *Label) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.String(orig, dui.Normal.Text, image.ZP, ui.font(dui), ui.Text)
}

func (ui *Label) Mouse(dui *DUI, origM, m draw.Mouse) Result {
	return Result{Hit: ui}
}

func (ui *Label) Key(dui *DUI, orig image.Point, m draw.Mouse, c rune) Result {
	return Result{Hit: ui}
}

func (ui *Label) FirstFocus(dui *DUI) *image.Point {
	return nil
}

func (ui *Label) Focus(dui *DUI, o UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

func (ui *Label) Print(indent int, r image.Rectangle) {
	PrintUI("Label", indent, r)
}

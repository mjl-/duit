package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Label struct {
	Text string

	sizes sizes
}

func (ui *Label) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	setSizes(display, &ui.sizes)
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*ui.sizes.margin + 2*ui.sizes.border, 2 * ui.sizes.space})
}
func (ui *Label) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.String(orig.Add(image.Point{ui.sizes.margin + ui.sizes.border, ui.sizes.space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Label) Mouse(m draw.Mouse) Result {
	return Result{Hit: ui}
}
func (ui *Label) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{Hit: ui}
}
func (ui *Label) FirstFocus() *image.Point {
	return nil
}
func (ui *Label) Focus(o UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

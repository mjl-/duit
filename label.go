package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Label struct {
	Text string
}

func (ui *Label) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*Margin + 2*Border, 2 * Space})
}
func (ui *Label) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.String(orig.Add(image.Point{Margin + Border, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
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

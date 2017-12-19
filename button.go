package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Button struct {
	Text  string
	Click func()

	m draw.Mouse
}

func (ui *Button) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2 * Space, 2 * Space})
}
func (ui *Button) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	size := display.DefaultFont.StringSize(ui.Text)

	grey, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, draw.Palegreygreen)
	check(err, "allocimage grey")

	r := image.Rectangle{
		orig.Add(image.Point{Margin + Border, Margin + Border}),
		orig.Add(size).Add(image.Point{2*Padding + Margin + Border, 2*Padding + Margin + Border}),
	}
	hover := m.In(image.Rectangle{image.ZP, size.Add(image.Pt(2*Space, 2*Space))})
	borderColor := grey
	if hover {
		borderColor = display.Black
	}
	img.Draw(r, grey, nil, image.ZP)
	img.Border(image.Rectangle{orig.Add(image.Point{Margin, Margin}), orig.Add(size).Add(image.Point{Margin + 2*Padding + 2*Border, Margin + 2*Padding + 2*Border})}, 1, borderColor, image.ZP)
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Button) Mouse(m draw.Mouse) Result {
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 && ui.Click != nil {
		ui.Click()
	}
	ui.m = m
	return Result{Hit: ui}
}
func (ui *Button) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{Hit: ui}
}
func (ui *Button) FirstFocus() *image.Point {
	p := image.Pt(Space, Space)
	return &p
}

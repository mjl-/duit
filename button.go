package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Button struct {
	Text  string
	Click func(r *Result)

	m     draw.Mouse
	sizes sizes
}

func (ui *Button) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	setSizes(display, &ui.sizes)
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2 * ui.sizes.space, 2 * ui.sizes.space})
}

func (ui *Button) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	size := display.DefaultFont.StringSize(ui.Text)

	grey, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, draw.Palegreygreen)
	check(err, "allocimage grey")

	r := image.Rectangle{
		orig.Add(image.Point{ui.sizes.margin + ui.sizes.border, ui.sizes.margin + ui.sizes.border}),
		orig.Add(size).Add(image.Point{2*Padding + ui.sizes.margin + ui.sizes.border, 2*Padding + ui.sizes.margin + ui.sizes.border}),
	}
	hover := m.In(image.Rectangle{image.ZP, size.Add(image.Pt(2*ui.sizes.space, 2*ui.sizes.space))})
	borderColor := grey
	if hover {
		borderColor = display.Black
	}
	img.Draw(r, grey, nil, image.ZP)
	img.Border(image.Rectangle{orig.Add(image.Point{ui.sizes.margin, ui.sizes.margin}), orig.Add(size).Add(image.Point{ui.sizes.margin + 2*Padding + 2*ui.sizes.border, ui.sizes.margin + 2*Padding + 2*ui.sizes.border})}, 1, borderColor, image.ZP)
	img.String(orig.Add(image.Point{ui.sizes.space, ui.sizes.space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}

func (ui *Button) Mouse(m draw.Mouse) Result {
	r := Result{Hit: ui}
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 && ui.Click != nil {
		ui.Click(&r)
	}
	ui.m = m
	return r
}

func (ui *Button) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{Hit: ui}
}

func (ui *Button) FirstFocus() *image.Point {
	p := image.Pt(ui.sizes.space, ui.sizes.space)
	return &p
}

func (ui *Button) Focus(o UI) *image.Point {
	if o != ui {
		return nil
	}
	p := image.Pt(ui.sizes.space, ui.sizes.space)
	return &p
}

func (ui *Button) Print(indent int, r image.Rectangle) {
	uiPrint("Button", indent, r)
}

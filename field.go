package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Field struct {
	Text    string
	Changed func(string, *Result)

	size image.Point // including space
}

func (ui *Field) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	ui.size = image.Point{r.Dx(), 2*Space + display.DefaultFont.Height}
	return ui.size
}
func (ui *Field) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	hover := m.In(image.Rectangle{image.ZP, ui.size})
	r := image.Rectangle{orig, orig.Add(ui.size)}
	img.Draw(r, display.White, nil, image.ZP)

	color := display.Black
	if hover {
		var err error
		color, err = display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, draw.Blue)
		check(err, "allocimage")
	}
	img.Border(
		image.Rectangle{
			orig.Add(image.Point{Margin, Margin}),
			orig.Add(ui.size).Sub(image.Point{Margin, Margin}),
		},
		1, color, image.ZP)
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Field) Mouse(m draw.Mouse) Result {
	return Result{Hit: ui}
}
func (ui *Field) Key(orig image.Point, m draw.Mouse, c rune) Result {
	switch c {
	case PageUp, PageDown, ArrowUp, ArrowDown:
		return Result{Hit: ui}
	case '\t':
		return Result{Hit: ui}
	case 8:
		if ui.Text != "" {
			ui.Text = ui.Text[:len(ui.Text)-1]
		}
	default:
		ui.Text += string(c)
	}
	result := Result{Hit: ui, Consumed: true, Redraw: true}
	if ui.Changed != nil {
		ui.Changed(ui.Text, &result)
	}
	return result
}
func (ui *Field) FirstFocus() *image.Point {
	p := image.Pt(Space, Space)
	return &p
}
func (ui *Field) Focus(o UI) *image.Point {
	if o != ui {
		return nil
	}
	p := image.Pt(Space, Space)
	return &p
}

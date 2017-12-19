package duit

import (
	"image"

	"9fans.net/go/draw"
)

type ListValue struct {
	Label    string
	Value    interface{}
	Selected bool
}

type List struct {
	Values   []*ListValue
	Multiple bool
	Changed  func(index int, result *Result)

	lineHeight int
	size       image.Point
	padding    image.Point
}

func (ui *List) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	font := display.DefaultFont
	ui.lineHeight = font.Height
	ui.padding = image.Pt(font.Height/4, font.Height/4)
	ui.size = image.Pt(r.Dx(), len(ui.Values)*ui.lineHeight).Add(ui.padding).Add(ui.padding)
	return ui.size
}
func (ui *List) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	font := display.DefaultFont
	r := image.Rectangle{orig, orig.Add(ui.size)}
	img.Draw(r, display.White, nil, image.ZP)
	cur := orig.Add(ui.padding)
	for _, v := range ui.Values {
		color := display.Black
		if v.Selected {
			img.Draw(image.Rectangle{cur, cur.Add(image.Pt(ui.size.X-2*ui.padding.X, font.Height))}, display.Black, nil, image.ZP)
			color = display.White
		}
		img.String(cur, color, image.ZP, font, v.Label)
		cur.Y += ui.lineHeight
	}
}
func (ui *List) Mouse(m draw.Mouse) (result Result) {
	result.Hit = ui
	m.Point = m.Point.Sub(ui.padding)
	if m.In(image.Rectangle{image.ZP, ui.size.Sub(ui.padding).Sub(ui.padding)}) {
		index := m.Y / ui.lineHeight
		if m.Buttons == 1 {
			v := ui.Values[index]
			v.Selected = !v.Selected
			if v.Selected && !ui.Multiple {
				for _, vv := range ui.Values {
					if vv != v {
						vv.Selected = false
					}
				}
			}
			if ui.Changed != nil {
				ui.Changed(index, &result)
			}
			result.Redraw = true
			result.Consumed = true
		}
	}
	return
}
func (ui *List) Key(orig image.Point, m draw.Mouse, k rune) (result Result) {
	result.Hit = ui
	return
}
func (ui *List) FirstFocus() *image.Point {
	return &ui.padding
}
func (ui *List) Focus(o UI) *image.Point {
	if o != ui {
		return nil
	}
	return &ui.padding
}

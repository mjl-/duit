package duit

import (
	"image"

	"9fans.net/go/draw"
)

// Scroll shows a part of its single child, typically a box, and lets you scroll the content.
type Scroll struct {
	Child UI

	r             image.Rectangle // entire ui
	barR          image.Rectangle
	childSize     image.Point
	offset        int         // current scroll offset in pixels
	img           *draw.Image // for child to draw on
	scrollbarSize int
}

func NewScroll(ui UI) *Scroll {
	return &Scroll{Child: ui}
}

func (ui *Scroll) Layout(d *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	ui.scrollbarSize = scale(d, ScrollbarSize)
	ui.r = image.Rectangle{image.ZP, image.Pt(r.Dx(), r.Max.Y-cur.Y)}
	ui.barR = ui.r
	ui.barR.Max.X = ui.barR.Min.X + ui.scrollbarSize
	ui.childSize = ui.Child.Layout(d, image.Rectangle{image.ZP, image.Pt(ui.r.Dx()-ui.barR.Dx(), ui.r.Dy())}, image.ZP)
	if ui.r.Dy() > ui.childSize.Y {
		ui.barR.Max.Y = ui.childSize.Y
		ui.r.Max.Y = ui.childSize.Y
	}
	return ui.r.Size()
}

func (ui *Scroll) Draw(d *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	if ui.childSize.X == 0 || ui.childSize.Y == 0 {
		return
	}

	// draw scrollbar
	lightGrey, err := d.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0xEEEEEEFF)
	check(err, "allowimage lightgrey")
	darkerGrey, err := d.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0xAAAAAAFF)
	check(err, "allowimage darkergrey")
	barR := ui.barR.Add(orig)
	img.Draw(barR, lightGrey, nil, image.ZP)
	barRActive := barR
	h := ui.r.Dy()
	uih := ui.childSize.Y
	if uih > h {
		barH := int((float32(h) / float32(uih)) * float32(h))
		barY := int((float32(ui.offset) / float32(uih)) * float32(h))
		barRActive.Min.Y += barY
		barRActive.Max.Y = barRActive.Min.Y + barH
	}
	img.Draw(barRActive, darkerGrey, nil, image.ZP)

	// draw child ui
	if ui.childSize.X == 0 || ui.childSize.Y == 0 {
		return
	}
	if ui.img == nil || ui.childSize != ui.img.R.Size() {
		var err error
		ui.img, err = d.AllocImage(image.Rectangle{image.ZP, ui.childSize}, draw.ARGB32, false, draw.White)
		check(err, "allocimage")
	}

	ui.img.Draw(ui.img.R, d.White, nil, image.ZP)
	m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
	ui.Child.Draw(d, ui.img, image.Pt(0, -ui.offset), m)
	img.Draw(ui.img.R.Add(orig).Add(image.Pt(ui.scrollbarSize, 0)), ui.img, nil, image.ZP)
}

func (ui *Scroll) scroll(delta int) bool {
	o := ui.offset
	ui.offset += delta
	if ui.offset < 0 {
		ui.offset = 0
	}
	offsetMax := ui.childSize.Y - ui.r.Dy()
	if ui.offset > offsetMax {
		ui.offset = offsetMax
	}
	return o != ui.offset
}

func (ui *Scroll) scrollKey(c rune) (consumed bool) {
	switch c {
	case ArrowUp:
		return ui.scroll(-50)
	case ArrowDown:
		return ui.scroll(50)
	case PageUp:
		return ui.scroll(-200)
	case PageDown:
		return ui.scroll(200)
	}
	return false
}

func (ui *Scroll) scrollMouse(m draw.Mouse) (consumed bool) {
	switch m.Buttons {
	case WheelUp:
		return ui.scroll(-50)
	case WheelDown:
		return ui.scroll(50)
	}
	return false
}

func (ui *Scroll) Mouse(m draw.Mouse) Result {
	if m.Point.In(ui.barR) {
		consumed := ui.scrollMouse(m)
		redraw := consumed
		return Result{Hit: ui, Consumed: consumed, Redraw: redraw}
	}
	if m.Point.In(ui.r) {
		m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r := ui.Child.Mouse(m)
		if !r.Consumed {
			r.Consumed = ui.scrollMouse(m)
			r.Redraw = r.Redraw || r.Consumed
		}
		return r
	}
	return Result{}
}

func (ui *Scroll) Key(orig image.Point, m draw.Mouse, c rune) Result {
	if m.Point.In(ui.barR) {
		consumed := ui.scrollKey(c)
		redraw := consumed
		return Result{Hit: ui, Consumed: consumed, Redraw: redraw}
	}
	if m.Point.In(ui.r) {
		m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r := ui.Child.Key(orig.Add(image.Pt(ui.scrollbarSize, -ui.offset)), m, c)
		if !r.Consumed {
			r.Consumed = ui.scrollKey(c)
			r.Redraw = r.Redraw || r.Consumed
		}
		return r
	}
	return Result{}
}

func (ui *Scroll) FirstFocus() *image.Point {
	first := ui.Child.FirstFocus()
	if first == nil {
		return nil
	}
	p := first.Add(image.Pt(ui.scrollbarSize, -ui.offset))
	return &p
}

func (ui *Scroll) Focus(o UI) *image.Point {
	p := ui.Child.Focus(o)
	if p == nil {
		return nil
	}
	pp := p.Add(image.Pt(ui.scrollbarSize, -ui.offset))
	return &pp
}

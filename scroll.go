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

var _ UI = &Scroll{}

func NewScroll(ui UI) *Scroll {
	return &Scroll{Child: ui}
}

func (ui *Scroll) Layout(env *Env, size image.Point) image.Point {
	ui.scrollbarSize = scale(env.Display, ScrollbarSize)
	ui.r = image.Rectangle{image.ZP, size}
	ui.barR = ui.r
	ui.barR.Max.X = ui.barR.Min.X + ui.scrollbarSize
	ui.childSize = ui.Child.Layout(env, image.Pt(ui.r.Dx()-ui.barR.Dx(), ui.r.Dy()))
	if ui.r.Dy() > ui.childSize.Y {
		ui.barR.Max.Y = ui.childSize.Y
		ui.r.Max.Y = ui.childSize.Y
	}
	return ui.r.Size()
}

func (ui *Scroll) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	if ui.childSize.X == 0 || ui.childSize.Y == 0 {
		return
	}

	ui.scroll(0)
	hover := m.In(ui.barR)

	bg := env.ScrollBGNormal
	vis := env.ScrollVisibleNormal
	if hover {
		bg = env.ScrollBGHover
		vis = env.ScrollVisibleHover
	}

	barR := ui.barR.Add(orig)
	img.Draw(barR, bg, nil, image.ZP)
	barRActive := barR
	h := ui.r.Dy()
	uih := ui.childSize.Y
	if uih > h {
		barH := int((float32(h) / float32(uih)) * float32(h))
		barY := int((float32(ui.offset) / float32(uih)) * float32(h))
		barRActive.Min.Y += barY
		barRActive.Max.Y = barRActive.Min.Y + barH
	}
	img.Draw(barRActive, vis, nil, image.ZP)

	// draw child ui
	if ui.childSize.X == 0 || ui.childSize.Y == 0 {
		return
	}
	if ui.img == nil || ui.childSize != ui.img.R.Size() {
		var err error
		ui.img, err = env.Display.AllocImage(image.Rectangle{image.ZP, ui.childSize}, draw.ARGB32, false, env.BackgroundColor)
		check(err, "allocimage")
	} else {
		ui.img.Draw(ui.img.R, env.Normal.Background, nil, image.ZP)
	}
	m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
	ui.Child.Draw(env, ui.img, image.Pt(0, -ui.offset), m)
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
	case draw.KeyUp:
		return ui.scroll(-50)
	case draw.KeyDown:
		return ui.scroll(50)
	case draw.KeyPageUp:
		return ui.scroll(-200)
	case draw.KeyPageDown:
		return ui.scroll(200)
	}
	return false
}

func (ui *Scroll) scrollMouse(m draw.Mouse) (consumed bool) {
	if m.Buttons == Button4 {
		return ui.scroll(-80)
	}
	if m.Buttons == Button5 {
		return ui.scroll(80)
	}
	return false
}

func (ui *Scroll) Mouse(env *Env, m draw.Mouse) Result {
	if m.Point.In(ui.barR) {
		consumed := ui.scrollMouse(m)
		redraw := consumed
		return Result{Hit: ui, Consumed: consumed, Redraw: redraw}
	}
	if m.Point.In(ui.r) {
		m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r := ui.Child.Mouse(env, m)
		if !r.Consumed {
			r.Consumed = ui.scrollMouse(m)
			r.Redraw = r.Redraw || r.Consumed
		}
		return r
	}
	return Result{}
}

func (ui *Scroll) Key(env *Env, orig image.Point, m draw.Mouse, c rune) Result {
	if m.Point.In(ui.barR) {
		consumed := ui.scrollKey(c)
		redraw := consumed
		return Result{Hit: ui, Consumed: consumed, Redraw: redraw}
	}
	if m.Point.In(ui.r) {
		m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r := ui.Child.Key(env, orig.Add(image.Pt(ui.scrollbarSize, -ui.offset)), m, c)
		if !r.Consumed {
			r.Consumed = ui.scrollKey(c)
			r.Redraw = r.Redraw || r.Consumed
		}
		return r
	}
	return Result{}
}

func (ui *Scroll) FirstFocus(env *Env) *image.Point {
	first := ui.Child.FirstFocus(env)
	if first == nil {
		return nil
	}
	p := first.Add(image.Pt(ui.scrollbarSize, -ui.offset))
	return &p
}

func (ui *Scroll) Focus(env *Env, o UI) *image.Point {
	p := ui.Child.Focus(env, o)
	if p == nil {
		return nil
	}
	pp := p.Add(image.Pt(ui.scrollbarSize, -ui.offset))
	return &pp
}

func (ui *Scroll) Print(indent int, r image.Rectangle) {
	uiPrint("Scroll", indent, r)
	ui.Child.Print(indent+1, image.Rectangle{image.ZP, ui.childSize})
}

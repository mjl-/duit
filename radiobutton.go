package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Radiobutton struct {
	Selected bool
	Value    interface{}
	Disabled bool
	Group    []*Radiobutton
	Changed  func(v interface{}, r *Result) // only the change function of the newly selected radiobutton in the group will be called

	m draw.Mouse
}

var _ UI = &Radiobutton{}

func (ui *Radiobutton) Layout(env *Env, size image.Point) image.Point {
	hit := image.Point{0, 1}
	return pt(2*BorderSize + 4*env.Display.DefaultFont.Height/5).Add(hit)
}

func (ui *Radiobutton) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	r := rect(pt(2*BorderSize + 4*env.Display.DefaultFont.Height/5))
	hover := m.In(r)
	r = r.Add(orig)

	colors := env.Normal
	color := colors.Text
	if ui.Disabled {
		colors = env.Disabled
		color = colors.Border
	} else if hover {
		colors = env.Hover
		color = colors.Border
	}

	hit := pt(0)
	if hover && m.Buttons&1 == 1 {
		hit = image.Pt(0, 1)
	}

	img.Draw(extendY(r, 1), colors.Background, nil, image.ZP)
	r = r.Add(hit)

	radius := r.Dx() / 2
	img.Arc(r.Min.Add(pt(radius)), radius, radius, 0, color, image.ZP, 0, 360)

	cr := r.Inset((4 * env.Display.DefaultFont.Height / 5) / 5).Add(hit)
	if ui.Selected {
		radius = cr.Dx() / 2
		img.FillArc(cr.Min.Add(pt(radius)), radius, radius, 0, color, image.ZP, 0, 360)
	}
}

func (ui *Radiobutton) check(r *Result) {
	ui.Selected = true
	for _, r := range ui.Group {
		if r != ui {
			r.Selected = false
		}
	}
	if ui.Changed != nil {
		ui.Changed(ui.Value, r)
	}
}

func (ui *Radiobutton) Mouse(env *Env, m draw.Mouse) (r Result) {
	r.Hit = ui
	if ui.Disabled {
		return
	}
	rr := rect(pt(2*BorderSize + 4*env.Display.DefaultFont.Height/5))
	hover := m.In(rr)
	if hover != ui.m.In(rr) {
		r.Redraw = true
	}
	if hover && ui.m.Buttons&1 != m.Buttons&1 {
		r.Redraw = true
		if m.Buttons&1 == 0 {
			r.Consumed = true
			ui.check(&r)
		}
	}
	ui.m = m
	return
}

func (ui *Radiobutton) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result) {
	r.Hit = ui
	if k == ' ' {
		r.Consumed = true
		r.Redraw = true
		ui.check(&r)
	}
	return
}

func (ui *Radiobutton) FirstFocus(env *Env) *image.Point {
	return &image.ZP
}

func (ui *Radiobutton) Focus(env *Env, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(env)
}

func (ui *Radiobutton) Print(indent int, r image.Rectangle) {
	uiPrint("Radiobutton", indent, r)
}

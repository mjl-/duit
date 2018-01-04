package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Checkbox struct {
	Checked  bool
	Disabled bool
	Changed  func(r *Result)

	m draw.Mouse
}

var _ UI = &Checkbox{}

func (ui *Checkbox) Layout(env *Env, size image.Point) image.Point {
	hit := image.Point{0, 1}
	return pt(2*BorderSize + 4*env.Display.DefaultFont.Height/5).Add(hit)
}

func (ui *Checkbox) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
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
	drawRoundedBorder(img, r, color)

	cr := r.Inset((4 * env.Display.DefaultFont.Height / 5) / 5)
	if ui.Checked {
		p0 := image.Pt(cr.Min.X, cr.Min.Y+2*cr.Dy()/3)
		p1 := image.Pt(cr.Min.X+1*cr.Dx()/3, cr.Max.Y)
		p2 := image.Pt(cr.Max.X, cr.Min.Y)
		img.Line(p0, p1, 0, 0, 1, color, image.ZP)
		img.Line(p1, p2, 0, 0, 1, color, image.ZP)
	}
}

func (ui *Checkbox) Mouse(env *Env, origM, m draw.Mouse) (r Result) {
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
			ui.Checked = !ui.Checked
			if ui.Changed != nil {
				ui.Changed(&r)
			}
		}
	}
	ui.m = m
	return
}

func (ui *Checkbox) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result) {
	r.Hit = ui
	if k == ' ' {
		r.Consumed = true
		r.Redraw = true
		ui.Checked = !ui.Checked
		if ui.Changed != nil {
			ui.Changed(&r)
		}
	}
	return
}

func (ui *Checkbox) FirstFocus(env *Env) *image.Point {
	return &image.ZP
}

func (ui *Checkbox) Focus(env *Env, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(env)
}

func (ui *Checkbox) Print(indent int, r image.Rectangle) {
	PrintUI("Checkbox", indent, r)
}

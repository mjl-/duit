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

func (ui *Radiobutton) Layout(dui *DUI, size image.Point) image.Point {
	hit := image.Point{0, 1}
	return pt(2*BorderSize + 4*dui.Display.DefaultFont.Height/5).Add(hit)
}

func (ui *Radiobutton) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	r := rect(pt(2*BorderSize + 4*dui.Display.DefaultFont.Height/5))
	hover := m.In(r)
	r = r.Add(orig)

	colors := dui.Normal
	color := colors.Text
	if ui.Disabled {
		colors = dui.Disabled
		color = colors.Border
	} else if hover {
		colors = dui.Hover
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

	cr := r.Inset((4 * dui.Display.DefaultFont.Height / 5) / 5).Add(hit)
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

func (ui *Radiobutton) Mouse(dui *DUI, origM, m draw.Mouse) (r Result) {
	r.Hit = ui
	if ui.Disabled {
		return
	}
	rr := rect(pt(2*BorderSize + 4*dui.Display.DefaultFont.Height/5))
	hover := m.In(rr)
	if hover != ui.m.In(rr) {
		r.Draw = true
	}
	if hover && ui.m.Buttons&1 != m.Buttons&1 {
		r.Draw = true
		if m.Buttons&1 == 0 {
			r.Consumed = true
			ui.check(&r)
		}
	}
	ui.m = m
	return
}

func (ui *Radiobutton) Key(dui *DUI, orig image.Point, m draw.Mouse, k rune) (r Result) {
	r.Hit = ui
	if k == ' ' {
		r.Consumed = true
		r.Draw = true
		ui.check(&r)
	}
	return
}

func (ui *Radiobutton) FirstFocus(dui *DUI) *image.Point {
	return &image.ZP
}

func (ui *Radiobutton) Focus(dui *DUI, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui)
}

func (ui *Radiobutton) Print(indent int, r image.Rectangle) {
	PrintUI("Radiobutton", indent, r)
}

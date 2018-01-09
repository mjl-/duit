package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Checkbox struct {
	Checked  bool
	Disabled bool
	Changed  func(e *Event)

	m draw.Mouse
}

var _ UI = &Checkbox{}

func (ui *Checkbox) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("Checkbox", self)
	hit := image.Point{0, 1}
	size := pt(2*BorderSize + 4*dui.Display.DefaultFont.Height/5).Add(hit)
	self.R = rect(size)
	return
}

func (ui *Checkbox) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw("Checkbox", self)

	r := rect(pt(2*BorderSize + 4*dui.Display.DefaultFont.Height/5))
	hover := m.In(r)
	r = r.Add(orig)

	colors := dui.Regular.Normal
	color := colors.Text
	if ui.Disabled {
		colors = dui.Disabled
		color = colors.Border
	} else if hover {
		colors = dui.Regular.Hover
		color = colors.Border
	}

	hit := pt(0)
	if hover && m.Buttons&1 == 1 {
		hit = image.Pt(0, 1)
	}

	img.Draw(extendY(r, 1), colors.Background, nil, image.ZP)
	r = r.Add(hit)
	drawRoundedBorder(img, r, color)

	cr := r.Inset((4 * dui.Display.DefaultFont.Height / 5) / 5)
	if ui.Checked {
		p0 := image.Pt(cr.Min.X, cr.Min.Y+2*cr.Dy()/3)
		p1 := image.Pt(cr.Min.X+1*cr.Dx()/3, cr.Max.Y)
		p2 := image.Pt(cr.Max.X, cr.Min.Y)
		img.Line(p0, p1, 0, 0, 1, color, image.ZP)
		img.Line(p1, p2, 0, 0, 1, color, image.ZP)
	}
}

func (ui *Checkbox) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	if ui.Disabled {
		return
	}
	rr := rect(pt(2*BorderSize + 4*dui.Display.DefaultFont.Height/5))
	hover := m.In(rr)
	if hover != ui.m.In(rr) {
		self.Draw = Dirty
	}
	if hover && ui.m.Buttons&1 != m.Buttons&1 {
		self.Draw = Dirty
		if m.Buttons&1 == 0 {
			r.Consumed = true
			ui.Checked = !ui.Checked
			if ui.Changed != nil {
				var e Event
				ui.Changed(&e)
				propagateEvent(self, &r, e)
			}
		}
	}
	ui.m = m
	return
}

func (ui *Checkbox) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if ui.Disabled {
		return
	}
	if k == ' ' {
		r.Consumed = true
		self.Draw = Dirty
		ui.Checked = !ui.Checked
		if ui.Changed != nil {
			var e Event
			ui.Changed(&e)
			propagateEvent(self, &r, e)
		}
	}
	return
}

func (ui *Checkbox) FirstFocus(dui *DUI) *image.Point {
	return &image.ZP
}

func (ui *Checkbox) Focus(dui *DUI, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui)
}

func (ui *Checkbox) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Checkbox) Print(self *Kid, indent int) {
	PrintUI("Checkbox", self, indent)
}

package duit

import (
	"image"

	"9fans.net/go/draw"
)

// Radiobutton is typically part of a group of radiobuttons, with exactly one of them selected. Labels are not part of the radiobutton itself.
type Radiobutton struct {
	Selected bool
	Disabled bool             // If set, cannot be selected.
	Group    RadiobuttonGroup // Other radiobuttons as part of this group. If a radiobutton is selected, others in the group are unselected.
	Font     *draw.Font       `json:"-"` // Used only to determine size of radiobutton to draw.
	Value    interface{}      `json:"-"` // Auxiliary data.

	// Called for the radiobutton in the group that is newly selected, not for the other radiobuttons in the group.
	// Not called if selected with Select().
	Changed func(v interface{}) (e Event) `json:"-"`

	m draw.Mouse
}

var _ UI = &Radiobutton{}

// RadiobuttonGroup is the group of all possible radiobuttons of which only one can be selected.
type RadiobuttonGroup []*Radiobutton

// Selected returns the currently selected radiobutton in the group.
func (g RadiobuttonGroup) Selected() *Radiobutton {
	for _, r := range g {
		if r.Selected {
			return r
		}
	}
	return nil
}

func (ui *Radiobutton) font(dui *DUI) *draw.Font {
	if ui.Font != nil {
		return ui.Font
	}
	return dui.Display.DefaultFont
}

func (ui *Radiobutton) size(dui *DUI) image.Point {
	return pt(2*BorderSize + ui.innerDim(dui))
}

func (ui *Radiobutton) innerDim(dui *DUI) int {
	return 7 * dui.Display.DefaultFont.Height / 10
}

func (ui *Radiobutton) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)

	hit := image.Point{0, 1}
	size := pt(2*BorderSize + 7*dui.Display.DefaultFont.Height/10).Add(hit)
	self.R = rect(size)
}

func (ui *Radiobutton) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)

	r := rect(pt(2*BorderSize + 7*dui.Display.DefaultFont.Height/10))
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

	radius := r.Dx() / 2
	img.Arc(r.Min.Add(pt(radius)), radius, radius, 0, color, image.ZP, 0, 360)

	cr := r.Inset((7 * dui.Display.DefaultFont.Height / 10) / 5)
	if ui.Selected {
		radius = cr.Dx() / 2
		img.FillArc(cr.Min.Add(pt(radius)), radius, radius, 0, color, image.ZP, 0, 360)
	}
}

// Select this radiobutton from the group, unselecting the previously selected radiobutton.
// Select does not call Changed.
func (ui *Radiobutton) Select(dui *DUI) {
	if ui.Disabled {
		return
	}
	ui.Selected = true
	for _, o := range ui.Group {
		if o != ui {
			o.Selected = false
		}
		dui.MarkDraw(o)
	}
}

func (ui *Radiobutton) check(self *Kid, r *Result) {
	ui.Selected = true
	for _, r := range ui.Group {
		if r != ui {
			r.Selected = false
		}
	}
	if ui.Changed != nil {
		e := ui.Changed(ui.Value)
		propagateEvent(self, r, e)
	}
}

func (ui *Radiobutton) markDraw(dui *DUI) {
	for _, o := range ui.Group {
		if o != ui {
			dui.MarkDraw(o)
		}
	}
}

func (ui *Radiobutton) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	if ui.Disabled {
		return
	}
	rr := rect(ui.size(dui))
	hover := m.In(rr)
	if hover != ui.m.In(rr) {
		self.Draw = Dirty
	}
	if hover && ui.m.Buttons&1 != m.Buttons&1 {
		self.Draw = Dirty
		ui.markDraw(dui)
		if m.Buttons&1 == 0 {
			r.Consumed = true
			ui.check(self, &r)
		}
	}
	ui.m = m
	return
}

func (ui *Radiobutton) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if k == ' ' {
		r.Consumed = true
		self.Draw = Dirty
		ui.markDraw(dui)
		ui.check(self, &r)
	}
	return
}

func (ui *Radiobutton) FirstFocus(dui *DUI, self *Kid) *image.Point {
	p := ui.size(dui).Mul(3).Div(4)
	return &p
}

func (ui *Radiobutton) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui, self)
}

func (ui *Radiobutton) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Radiobutton) Print(self *Kid, indent int) {
	PrintUI("Radiobutton", self, indent)
}

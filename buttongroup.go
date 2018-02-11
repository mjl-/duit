package duit

import (
	"image"

	"9fans.net/go/draw"
)

// Buttongroup shows multiple joined buttons, with one of them active.
type Buttongroup struct {
	Texts    []string                  // Texts to display on the buttons.
	Selected int                       // Index in Text of the currently selected button.
	Disabled bool                      // If disabled, the duit.Disabled colors are used and clicks have no effect.
	Font     *draw.Font                `json:"-"` // Used for drawing Texts.
	Changed  func(index int) (e Event) `json:"-"` // Called on click on a different button in the group then previously selected.

	m    draw.Mouse
	size image.Point
}

var _ UI = &Buttongroup{}

func (ui *Buttongroup) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Buttongroup) padding(dui *DUI) image.Point {
	fontHeight := ui.font(dui).Height
	return image.Pt(fontHeight/2, fontHeight/4)
}

func (ui *Buttongroup) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	pad2 := ui.padding(dui).Mul(2)
	size := image.Pt(2*BorderSize, 2*BorderSize+pad2.Y+ui.font(dui).Height)
	font := ui.font(dui)
	for i, t := range ui.Texts {
		size.X += font.StringSize(t).X + pad2.X
		if i > 0 {
			size.X += BorderSize
		}
	}
	ui.size = size
	self.R = rect(size)
	return
}

func (ui *Buttongroup) selected() int {
	if ui.Selected < 0 || ui.Selected >= len(ui.Texts) {
		return 0
	}
	return ui.Selected
}

func (ui *Buttongroup) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)

	if len(ui.Texts) == 0 {
		return
	}

	r := rect(ui.size)

	hover := m.In(r)
	colors := dui.Regular.Normal
	if ui.Disabled {
		colors = dui.Disabled
	} else if hover {
		colors = dui.Regular.Hover
	}

	r = r.Add(orig)
	drawRoundedBorder(img, r, colors.Border)

	sel := ui.selected()
	font := ui.font(dui)
	pad := ui.padding(dui)
	pad2 := pad.Mul(2)
	p := r.Min.Add(pt(BorderSize)).Add(pad)
	for i, t := range ui.Texts {
		dx := font.StringWidth(t)
		selR := image.Rect(p.X-pad.X, r.Min.Y+BorderSize, p.X+dx+pad.X, r.Max.Y-BorderSize)
		col := colors
		if i == sel {
			col = dui.Primary.Normal
			img.Draw(selR, col.Background, nil, image.ZP)
		}
		if i > 0 {
			p0 := image.Pt(p.X-pad.X-BorderSize, r.Min.Y+BorderSize)
			p1 := p0.Add(image.Pt(0, r.Dy()-2*BorderSize))
			img.Line(p0, p1, 0, 0, 0, col.Border, image.ZP)
		}
		pp := p
		if !ui.Disabled && m.Buttons == Button1 && m.Add(orig).In(selR) {
			pp = pp.Add(image.Pt(0, 1))
		}
		p0 := img.String(pp, col.Text, image.ZP, font, t)
		p.X = p0.X + pad2.X + BorderSize
	}
}

// findIndex returns the index of the text under the mouse, and the start and end X of the text
func (ui *Buttongroup) findIndex(dui *DUI, m draw.Mouse) (int, int, int) {
	offset := 0
	pad2 := ui.padding(dui).Mul(2)
	font := ui.font(dui)
	for i, t := range ui.Texts {
		end := offset + font.StringSize(t).X + pad2.X + BorderSize
		if m.X >= offset && m.X < end {
			return i, offset, end
		}
		offset = end
	}
	return -1, 0, 0
}

func (ui *Buttongroup) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	if ui.m.Buttons^m.Buttons != 0 {
		self.Draw = Dirty
	}
	if ui.m.Buttons == Button1 && m.Buttons == 0 && !ui.Disabled {
		index, _, _ := ui.findIndex(dui, m)
		if index >= 0 {
			ui.Selected = index
			if ui.Changed != nil {
				e := ui.Changed(ui.Selected)
				propagateEvent(self, &r, e)
			}
			self.Draw = Dirty
			r.Consumed = true
		}
	}
	ui.m = m
	return
}

func (ui *Buttongroup) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if ui.Disabled {
		return
	}
	switch k {
	case ' ', '\n':
		index, _, _ := ui.findIndex(dui, m)
		if index < 0 {
			break
		}
		r.Consumed = true
		self.Draw = Dirty
		ui.Selected = index
		if ui.Changed != nil {
			e := ui.Changed(ui.Selected)
			propagateEvent(self, &r, e)
		}
	case '\t':
		index, _, end := ui.findIndex(dui, m)
		if index < 0 {
			break
		}
		index++
		if index < len(ui.Texts) {
			p := orig.Add(image.Pt(end+BorderSize*2+ui.padding(dui).X, m.Y))
			r.Warp = &p
			r.Consumed = true
			self.Draw = Dirty
		}
	}
	return
}

func (ui *Buttongroup) FirstFocus(dui *DUI, self *Kid) *image.Point {
	p := ui.padding(dui)
	// todo: move to active item
	return &p
}

func (ui *Buttongroup) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui, self)
}

func (ui *Buttongroup) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Buttongroup) Print(self *Kid, indent int) {
	PrintUI("Buttongroup", self, indent)
}

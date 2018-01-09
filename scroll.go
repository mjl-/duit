package duit

import (
	"fmt"
	"image"

	"9fans.net/go/draw"
)

// Scroll shows a part of its single child, typically a box, and lets you scroll the content.
type Scroll struct {
	Kid    Kid
	Height int // < 0 means full height, 0 means as much as necessary, >0 means exactly that many lowdpi pixels

	r             image.Rectangle // entire ui
	barR          image.Rectangle
	barActiveR    image.Rectangle
	childR        image.Rectangle
	offset        int         // current scroll offset in pixels
	img           *draw.Image // for child to draw on
	scrollbarSize int
}

var _ UI = &Scroll{}

func NewScroll(ui UI) *Scroll {
	return &Scroll{Kid: Kid{UI: ui}}
}

func (ui *Scroll) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("Scroll", self)

	if self.Layout == Clean && !force {
		return
	}
	self.Layout = Clean
	self.Draw = Dirty
	// todo: be smarter about DirtyKid

	ui.scrollbarSize = scale(dui.Display, ScrollbarSize)
	scaledHeight := scale(dui.Display, ui.Height)
	if scaledHeight > 0 && scaledHeight < sizeAvail.Y {
		sizeAvail.Y = scaledHeight
	}
	ui.r = rect(sizeAvail)
	ui.barR = ui.r
	ui.barR.Max.X = ui.barR.Min.X + ui.scrollbarSize
	ui.childR = ui.r
	ui.childR.Min.X = ui.barR.Max.X

	// todo: only force when sizeAvail or childR changed?
	ui.Kid.UI.Layout(dui, &ui.Kid, image.Pt(ui.r.Dx()-ui.barR.Dx(), ui.r.Dy()), force)
	ui.Kid.Layout = Clean
	ui.Kid.Draw = Dirty

	kY := ui.Kid.R.Dy()
	if ui.r.Dy() > kY && ui.Height == 0 {
		ui.barR.Max.Y = kY
		ui.r.Max.Y = kY
		ui.childR.Max.Y = kY
	}
	self.R = rect(ui.r.Size())
}

func (ui *Scroll) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw("Scroll", self)

	if self.Draw == Clean {
		return
	}
	self.Draw = Clean

	if ui.r.Empty() {
		return
	}

	ui.scroll(0)
	barHover := m.In(ui.barR)

	bg := dui.ScrollBGNormal
	vis := dui.ScrollVisibleNormal
	if barHover {
		bg = dui.ScrollBGHover
		vis = dui.ScrollVisibleHover
	}

	h := ui.r.Dy()
	uih := ui.Kid.R.Dy()
	if uih > h {
		barR := ui.barR.Add(orig)
		img.Draw(barR, bg, nil, image.ZP)
		barH := int((float32(h) / float32(uih)) * float32(h))
		barY := int((float32(ui.offset) / float32(uih)) * float32(h))
		ui.barActiveR = ui.barR
		ui.barActiveR.Min.Y += barY
		ui.barActiveR.Max.Y = ui.barActiveR.Min.Y + barH
		img.Draw(ui.barActiveR.Add(orig), vis, nil, image.ZP)
	}

	// draw child ui
	if ui.childR.Empty() {
		return
	}
	if ui.img == nil || ui.Kid.R.Size() != ui.img.R.Size() {
		var err error
		if ui.img != nil {
			ui.img.Free()
			ui.img = nil
		}
		ui.img, err = dui.Display.AllocImage(ui.Kid.R, draw.ARGB32, false, dui.BackgroundColor)
		check(err, "allocimage")
		ui.Kid.Draw = Dirty
	} else if ui.Kid.Draw == Dirty {
		ui.img.Draw(ui.img.R, dui.Background, nil, image.ZP)
	}
	m.Point = m.Point.Add(image.Pt(-ui.childR.Min.X, ui.offset))
	if ui.Kid.Draw != Clean {
		if force {
			ui.Kid.Draw = Dirty
		}
		ui.Kid.UI.Draw(dui, &ui.Kid, ui.img, image.ZP, m, ui.Kid.Draw == Dirty)
		ui.Kid.Draw = Clean
	}
	img.Draw(ui.childR.Add(orig), ui.img, nil, image.Pt(0, ui.offset))
}

func (ui *Scroll) scroll(delta int) bool {
	o := ui.offset
	ui.offset += delta
	if ui.offset < 0 {
		ui.offset = 0
	}
	offsetMax := ui.Kid.R.Dy() - ui.childR.Dy()
	if offsetMax < 0 {
		offsetMax = 0
	}
	if ui.offset > offsetMax {
		ui.offset = offsetMax
	}
	return o != ui.offset
}

func (ui *Scroll) scrollKey(k rune) (consumed bool) {
	switch k {
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

func (ui *Scroll) scrollMouse(m draw.Mouse, scrollOnly bool) (consumed bool) {
	switch m.Buttons {
	case Button4:
		return ui.scroll(-m.Y / 4)
	case Button5:
		return ui.scroll(m.Y / 4)
	}

	if scrollOnly {
		return false
	}
	switch m.Buttons {
	case Button1:
		return ui.scroll(-m.Y)
	case Button2:
		offset := m.Y * ui.Kid.R.Dy() / ui.barR.Dy()
		offsetMax := ui.Kid.R.Dy() - ui.childR.Dy()
		if offset < 0 {
			offset = 0
		} else if offset > offsetMax {
			offset = offsetMax
		}
		o := ui.offset
		ui.offset = offset
		return o != ui.offset
	case Button3:
		return ui.scroll(m.Y)
	}
	return false
}

func (ui *Scroll) result(dui *DUI, self *Kid, r *Result, scrolled bool) {
	if ui.Kid.Layout != Clean {
		ui.Kid.UI.Layout(dui, &ui.Kid, ui.childR.Size(), false)
		ui.Kid.Layout = Clean
		ui.Kid.Draw = Dirty
		self.Draw = Dirty
	} else if ui.Kid.Draw != Clean || scrolled {
		self.Draw = Dirty
	}
}

func (ui *Scroll) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	if m.Point.In(ui.barR) {
		r.Hit = ui
		r.Consumed = ui.scrollMouse(m, false)
		self.Draw = Dirty
		return
	}
	if m.Point.In(ui.childR) {
		nOrigM := origM
		nOrigM.Point = nOrigM.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		nm := m
		nm.Point = nm.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r = ui.Kid.UI.Mouse(dui, &ui.Kid, nm, nOrigM, orig)
		scrolled := false
		if !r.Consumed {
			scrolled = ui.scrollMouse(m, true)
			r.Consumed = scrolled
		}
		ui.result(dui, self, &r, scrolled)
	}
	return
}

func (ui *Scroll) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if m.Point.In(ui.barR) {
		r.Hit = ui
		r.Consumed = ui.scrollKey(k)
		if r.Consumed {
			self.Draw = Dirty
		}
	}
	if m.Point.In(ui.childR) {
		m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r = ui.Kid.UI.Key(dui, &ui.Kid, k, m, orig.Add(image.Pt(ui.scrollbarSize, -ui.offset)))
		scrolled := false
		if !r.Consumed {
			scrolled = ui.scrollKey(k)
			r.Consumed = scrolled
		}
		ui.result(dui, self, &r, scrolled)
	}
	return
}

func (ui *Scroll) FirstFocus(dui *DUI) *image.Point {
	first := ui.Kid.UI.FirstFocus(dui)
	if first == nil {
		return nil
	}
	p := first.Add(image.Pt(ui.scrollbarSize, -ui.offset))
	return &p
}

func (ui *Scroll) Focus(dui *DUI, o UI) *image.Point {
	p := ui.Kid.UI.Focus(dui, o)
	if p == nil {
		return nil
	}
	pp := p.Add(image.Pt(ui.scrollbarSize, -ui.offset))
	return &pp
}

func (ui *Scroll) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	if self.Mark(o, forLayout) {
		return true
	}
	marked = ui.Kid.UI.Mark(&ui.Kid, o, forLayout)
	if marked {
		if forLayout {
			if self.Layout == Clean {
				self.Layout = DirtyKid
			}
		} else {
			if self.Layout == Clean {
				self.Draw = DirtyKid
			}
		}
	}
	return
}

func (ui *Scroll) Print(self *Kid, indent int) {
	what := fmt.Sprintf("Scroll offset=%d childR=%v", ui.offset, ui.childR)
	PrintUI(what, self, indent)
	ui.Kid.UI.Print(&ui.Kid, indent+1)
}

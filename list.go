package duit

import (
	"image"

	"9fans.net/go/draw"
)

type ListValue struct {
	Text     string
	Value    interface{} `json:"-"`
	Selected bool
}

type List struct {
	Values   []*ListValue
	Multiple bool
	Font     *draw.Font                                      `json:"-"`
	Changed  func(index int, e *Event)                       `json:"-"`
	Click    func(index int, m draw.Mouse, e *Event)         `json:"-"`
	Keys     func(index int, k rune, m draw.Mouse, e *Event) `json:"-"`

	m    draw.Mouse
	size image.Point
}

var _ UI = &List{}

func (ui *List) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *List) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("List", self)
	ui.size = image.Pt(sizeAvail.X, len(ui.Values)*(4*ui.font(dui).Height/3))
	self.R = rect(ui.size)
}

func (ui *List) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw("Label", self)

	font := ui.font(dui)
	r := rect(ui.size).Add(orig)
	img.Draw(r, dui.Background, nil, image.ZP)
	lineR := r
	lineR.Max.Y = lineR.Min.Y + 4*font.Height/3

	for _, v := range ui.Values {
		colors := dui.Regular.Normal
		if v.Selected {
			colors = dui.Inverse
			img.Draw(lineR, colors.Background, nil, image.ZP)
		}
		img.String(lineR.Min.Add(pt(font.Height/4)), colors.Text, image.ZP, font, v.Text)
		lineR = lineR.Add(image.Pt(0, 4*font.Height/3))
	}
}

func (ui *List) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	prevM := ui.m
	ui.m = m
	if !m.In(rect(ui.size)) {
		return
	}
	font := ui.font(dui)
	index := m.Y / (4 * font.Height / 3)
	if m.Buttons != 0 && prevM.Buttons^m.Buttons != 0 && ui.Click != nil {
		var e Event
		ui.Click(index, m, &e)
		propagateEvent(self, &r, e)
	}
	if !r.Consumed && prevM.Buttons == 0 && m.Buttons == Button1 {
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
			var e Event
			ui.Changed(index, &e)
			propagateEvent(self, &r, e)
		}
		self.Draw = Dirty
		r.Consumed = true
	}
	return
}

func (ui *List) selectedIndices() (l []int) {
	for i, lv := range ui.Values {
		if lv.Selected {
			l = append(l, i)
		}
	}
	return
}

func (ui *List) Selected() (indices []int) {
	return ui.selectedIndices()
}

// unselect indices, or if indices is nil, unselect all
func (ui *List) Unselect(indices []int) {
	if indices == nil {
		for _, lv := range ui.Values {
			lv.Selected = false
		}
	} else {
		for _, i := range indices {
			ui.Values[i].Selected = false
		}
	}
}

func (ui *List) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if !m.In(rect(ui.size)) {
		return
	}
	if ui.Keys != nil {
		// xxx what should "index" be? especially for multiple: true...
		sel := ui.selectedIndices()
		index := -1
		if len(sel) == 1 {
			index = sel[0]
		}
		var e Event
		ui.Keys(index, k, m, &e)
		propagateEvent(self, &r, e)
		if r.Consumed {
			return
		}
	}
	switch k {
	case draw.KeyUp, draw.KeyDown:
		if len(ui.Values) == 0 {
			return
		}
		sel := ui.selectedIndices()
		oindex := -1
		nindex := -1
		switch k {
		case draw.KeyUp:
			r.Consumed = true
			if len(sel) == 0 {
				nindex = len(ui.Values) - 1
			} else {
				oindex = sel[0]
				nindex = (sel[0] - 1 + len(ui.Values)) % len(ui.Values)
			}
		case draw.KeyDown:
			r.Consumed = true
			if len(sel) == 0 {
				nindex = 0
			} else {
				oindex = sel[len(sel)-1]
				nindex = (sel[len(sel)-1] + 1) % len(ui.Values)
			}
		}
		if oindex >= 0 {
			ui.Values[oindex].Selected = false
			self.Draw = Dirty
		}
		if nindex >= 0 {
			ui.Values[nindex].Selected = true
			self.Draw = Dirty
			if ui.Changed != nil {
				var e Event
				ui.Changed(nindex, &e)
				propagateEvent(self, &r, e)
			}
			// xxx orig probably should not be a part in this...
			font := ui.font(dui)
			p := orig.Add(image.Pt(m.X, nindex*(4*font.Height/3)+font.Height/2))
			r.Warp = &p
		}
	}
	return
}

func (ui *List) FirstFocus(dui *DUI) *image.Point {
	return &image.ZP
}

func (ui *List) Focus(dui *DUI, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui)
}

func (ui *List) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *List) Print(self *Kid, indent int) {
	PrintUI("List", self, indent)
}

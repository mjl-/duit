package duit

import (
	"image"

	"9fans.net/go/draw"
)

// ListValue is used for values in a List.
type ListValue struct {
	Text     string      // Text shown, as single line.
	Value    interface{} `json:"-"` // Auxiliary data.
	Selected bool
}

// List shows values, allowing for single or multiple selection, with callbacks when the selection changes.
//
// Keys:
//	arrow up, move selection up
//	arrow down, move selection down
//	home, move selection to first element
//	end, move selection to last element
type List struct {
	Values   []*ListValue                            // Values, each contains whether it is selected.
	Multiple bool                                    // Whether multiple values can be selected at a time.
	Font     *draw.Font                              `json:"-"` // For drawing the values.
	Changed  func(index int) (e Event)               `json:"-"` // Called after the selection changes, index being the new single selected item if >= 0.
	Click    func(index int, m draw.Mouse) (e Event) `json:"-"` // Called on click at value at index, before handling selection change. If consumed, processing stops.
	Keys     func(k rune, m draw.Mouse) (e Event)    `json:"-"` // Called on key. If consumed, processing stops.

	m    draw.Mouse
	size image.Point
}

var _ UI = &List{}

func (ui *List) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *List) rowHeight(dui *DUI) int {
	return 4 * ui.font(dui).Height / 3
}

func (ui *List) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	ui.size = image.Pt(sizeAvail.X, len(ui.Values)*ui.rowHeight(dui))
	self.R = rect(ui.size)
}

func (ui *List) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)

	rowHeight := ui.rowHeight(dui)
	font := ui.font(dui)
	r := rect(ui.size).Add(orig)
	img.Draw(r, dui.Background, nil, image.ZP)
	lineR := r
	lineR.Max.Y = lineR.Min.Y + rowHeight

	for _, v := range ui.Values {
		colors := dui.Regular.Normal
		if v.Selected {
			colors = dui.Inverse
			img.Draw(lineR, colors.Background, nil, image.ZP)
		}
		img.String(lineR.Min.Add(pt(font.Height/4)), colors.Text, image.ZP, font, v.Text)
		lineR = lineR.Add(image.Pt(0, rowHeight))
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
		e := ui.Click(index, m)
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
			e := ui.Changed(index)
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

// Selected returns the indices of the selected values.
func (ui *List) Selected() (indices []int) {
	return ui.selectedIndices()
}

// Unselect indices, or if indices is nil, unselects all.
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
		e := ui.Keys(k, m)
		propagateEvent(self, &r, e)
		if r.Consumed {
			return
		}
	}
	switch k {
	case draw.KeyUp, draw.KeyDown, draw.KeyHome, draw.KeyEnd:
		if len(ui.Values) == 0 {
			return
		}
		sel := ui.selectedIndices()
		oindex := -1
		nindex := -1
		switch k {
		case draw.KeyUp:
			if len(sel) == 0 {
				nindex = len(ui.Values) - 1
			} else {
				oindex = sel[0]
				nindex = maximum(0, sel[0]-1)
			}
		case draw.KeyDown:
			if len(sel) == 0 {
				nindex = 0
			} else {
				oindex = sel[len(sel)-1]
				nindex = minimum(sel[len(sel)-1]+1, len(ui.Values)-1)
			}
		case draw.KeyHome:
			nindex = 0
		case draw.KeyEnd:
			nindex = len(ui.Values) - 1
		}
		r.Consumed = oindex != nindex
		if !r.Consumed {
			return
		}
		if oindex >= 0 {
			ui.Values[oindex].Selected = false
			self.Draw = Dirty
		}
		if nindex >= 0 {
			ui.Values[nindex].Selected = true
			self.Draw = Dirty
			if ui.Changed != nil {
				e := ui.Changed(nindex)
				propagateEvent(self, &r, e)
			}
			// xxx orig probably should not be a part in this...
			font := ui.font(dui)
			p := orig.Add(image.Pt(m.X, nindex*ui.rowHeight(dui)+font.Height/2))
			r.Warp = &p
		}
	}
	return
}

func (ui *List) firstSelected() int {
	for i, lv := range ui.Values {
		if lv.Selected {
			return i
		}
	}
	return -1
}

func (ui *List) FirstFocus(dui *DUI, self *Kid) *image.Point {
	rowHeight := ui.rowHeight(dui)
	p := image.Pt(self.R.Dx()/2, maximum(0, ui.firstSelected())*rowHeight+rowHeight/2)
	return &p
}

func (ui *List) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui, self)
}

func (ui *List) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *List) Print(self *Kid, indent int) {
	PrintUI("List", self, indent)
}

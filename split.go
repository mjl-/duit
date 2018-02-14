package duit

import (
	"image"

	"9fans.net/go/draw"
)

// Split is a horizontal or vertical split of the available space, with 1 or more UIs.
type Split struct {
	// Space between the UIs, in lowDPI pixels.
	// If >0, users can drag the gutter. Manual changes are automatically stored and restored on next load, if you set ID in the containing Kid.
	Gutter int

	// Optional, must return the division of available space. Sum of dims must be dim.
	Split func(dim int) (dims []int) `json:"-"`

	Vertical   bool
	Kids       []*Kid      // Hold UIs shown in split.
	Background *draw.Image `json:"-"` // For background color.

	size   image.Point
	dims   []int
	manual struct {
		uiDim int // total of dims + gutters, to see if we need to recalculate dims during layout
		dims  []int
	}
	m             draw.Mouse
	dragging      bool
	draggingIndex int
}

var _ UI = &Split{}

func (ui *Split) ensureManual(dui *DUI) {
	if len(ui.manual.dims) != len(ui.Kids) {
		ui.manual.dims = make([]int, len(ui.dims))
	}
	copy(ui.manual.dims, ui.dims)
	gut := dui.Scale(ui.Gutter)
	ui.manual.uiDim = (len(ui.Kids) - 1) * gut
	for _, d := range ui.dims {
		ui.manual.uiDim += d
	}
}

func (ui *Split) dim(p image.Point) int {
	if ui.Vertical {
		return p.Y
	}
	return p.X
}

func (ui *Split) Dimensions(dui *DUI, dims []int) []int {
	if dims != nil {
		if len(dims) != len(ui.Kids) {
			panic("bad dimensions")
		}
		if len(ui.dims) != len(dims) {
			ui.dims = make([]int, len(dims))
		}
		copy(ui.dims, dims)
		if len(ui.manual.dims) != len(dims) {
			ui.manual.dims = make([]int, len(dims))
		}
		copy(ui.manual.dims, dims)
	}
	r := make([]int, len(ui.dims))
	copy(r, ui.dims)
	return r
}

func (ui *Split) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	if KidsLayout(dui, self, ui.Kids, force) {
		return
	}

	gut := dui.Scale(ui.Gutter)

	// from manual.dims to dims
	reassign := func() {
		if len(ui.dims) != len(ui.Kids) {
			ui.dims = make([]int, len(ui.Kids))
		}
		if sizeAvail.X == ui.manual.uiDim {
			copy(ui.dims, ui.manual.dims)
			return
		}

		had := 0
		for _, d := range ui.manual.dims {
			had += d
		}
		have := ui.dim(sizeAvail) - (len(ui.Kids)-1)*gut
		left := have
		for i, d := range ui.manual.dims {
			if i == len(ui.manual.dims)-1 {
				ui.dims[i] = left
			} else {
				ui.dims[i] = d * have / had
				left -= ui.dims[i]
			}
		}
		ui.manual.uiDim = ui.dim(sizeAvail)
		copy(ui.manual.dims, ui.dims)
	}

	split := func() {
		if ui.Split == nil {
			have := ui.dim(sizeAvail) - (len(ui.Kids)-1)*gut
			ui.dims = make([]int, len(ui.Kids))
			for i := range ui.dims {
				ui.dims[i] = have / len(ui.Kids)
			}
			ui.dims[len(ui.dims)-1] = have - (len(ui.dims)-1)*(have/len(ui.Kids))
		} else {
			ui.dims = ui.Split(ui.dim(sizeAvail) - (len(ui.Kids)-1)*gut)
			if len(ui.dims) != len(ui.Kids) {
				panic("bad number of dims from split")
			}
		}
		ui.manual.dims = nil
		ui.manual.uiDim = 0
	}

	var r []int
	if len(ui.manual.dims) == len(ui.Kids) {
		reassign()
	} else if self.ID != "" && dui.ReadSettings(self, &r) && len(r) == len(ui.Kids) {
		ui.manual.uiDim = (len(ui.Kids) - 1) * gut
		for _, d := range r {
			ui.manual.uiDim += d
		}
		ui.manual.dims = r
		reassign()
	} else {
		split()
	}

	ui.size = image.ZP
	if ui.Vertical {
		for i, k := range ui.Kids {
			k.UI.Layout(dui, k, image.Pt(sizeAvail.X, ui.dims[i]), true)
			k.R = k.R.Add(image.Pt(0, ui.size.Y))
			ui.size.Y += ui.dims[i]
			if i < len(ui.dims)-1 {
				ui.size.Y += gut
			}
			ui.size.X = maximum(ui.size.X, k.R.Dx())
		}
	} else {
		for i, k := range ui.Kids {
			k.UI.Layout(dui, k, image.Pt(ui.dims[i], sizeAvail.Y), true)
			k.R = k.R.Add(image.Pt(ui.size.X, 0))
			ui.size.X += ui.dims[i]
			if i < len(ui.dims)-1 {
				ui.size.X += gut
			}
			ui.size.Y = maximum(ui.size.Y, k.R.Dy())
		}
	}
	self.R = rect(ui.size)
}

func (ui *Split) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	KidsDraw(dui, self, ui.Kids, ui.size, ui.Background, img, orig, m, force)
}

func (ui *Split) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	findGutter := func(d int) int {
		gut := dui.Scale(ui.Gutter)
		o := 0
		slack := dui.Scale(1)
		for i, dd := range ui.dims[:len(ui.dims)-1] {
			o += dd
			if d >= o-slack && d < o+gut+slack {
				return i
			}
			o += gut
		}
		return -1
	}

	if ui.Gutter > 0 && m.Buttons == Button1 && ui.m.Buttons == 0 {
		index := findGutter(ui.dim(m.Point))
		if index >= 0 {
			ui.dragging = true
			ui.draggingIndex = index
		}
	} else if ui.dragging {
		if m.Buttons == Button1 {
			delta := ui.dim(m.Point) - ui.dim(ui.m.Point)
			if delta != 0 {
				ui.ensureManual(dui)
				if ui.manual.dims[ui.draggingIndex]+delta >= 0 && ui.manual.dims[ui.draggingIndex+1]-delta >= 0 {
					ui.manual.dims[ui.draggingIndex] += delta
					ui.manual.dims[ui.draggingIndex+1] -= delta
					dui.WriteSettings(self, ui.manual.dims)
				}
				r.Consumed = true
				r.Hit = ui
				self.Layout = Dirty
			}
			ui.m = m
			return
		}
		ui.dragging = false
	}
	r = KidsMouse(dui, self, ui.Kids, m, origM, orig)
	ui.m = m
	return r
}

func (ui *Split) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return KidsKey(dui, self, ui.Kids, k, m, orig)
}

func (ui *Split) FirstFocus(dui *DUI, self *Kid) *image.Point {
	return KidsFirstFocus(dui, self, ui.Kids)
}

func (ui *Split) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	return KidsFocus(dui, self, ui.Kids, o)
}

func (ui *Split) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return KidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Split) Print(self *Kid, indent int) {
	how := "horizontal"
	if ui.Vertical {
		how = "vertical"
	}
	PrintUI("Split "+how, self, indent)
	KidsPrint(ui.Kids, indent+1)
}

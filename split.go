package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Split struct {
	Gutter int // in lowDPI pixels
	Vertical bool
	Split  func(dim int) (dims []int) `json:"-"`
	Kids   []*Kid

	size   image.Point
	dims []int
	manual struct {
		uiDim int // total of dims + gutters, to see if we need to recalculate dims during layout
		dims  []int
	}
	m draw.Mouse
	dragging bool
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

func (ui *Split) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("Split", self)
	if kidsLayout(dui, self, ui.Kids, force) {
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
		ui.dims = ui.Split(ui.dim(sizeAvail) - (len(ui.Kids)-1)*gut)
		if len(ui.dims) != len(ui.Kids) {
			panic("bad number of dims from split")
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
	draw := self.Draw == Dirty || force
	kidsDraw("Split", dui, self, ui.Kids, ui.size, img, orig, m, force)
	if draw && ui.Gutter > 0 {
		gut := dui.Scale(ui.Gutter)
		if ui.Vertical {
			r := image.Rect(0, -gut, ui.size.X, 0).Add(orig)
			for _, d := range ui.dims[:len(ui.dims)-1] {
				r = r.Add(image.Pt(0, d+gut))
				img.Draw(r, dui.Display.White, nil, image.ZP)
			}
		} else {
			r := image.Rect(-gut, 0, 0, ui.size.Y).Add(orig)
			for _, d := range ui.dims[:len(ui.dims)-1] {
				r = r.Add(image.Pt(d+gut, 0))
				img.Draw(r, dui.Display.White, nil, image.ZP)
			}
		}
	}
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
				if ui.manual.dims[ui.draggingIndex] + delta >= 0 && ui.manual.dims[ui.draggingIndex+1] - delta >= 0 {
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
		} else {
			ui.dragging = false
		}
	}
	r = kidsMouse(dui, self, ui.Kids, m, origM, orig)
	ui.m = m
	return r
}

func (ui *Split) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return kidsKey(dui, self, ui.Kids, k, m, orig)
}

func (ui *Split) FirstFocus(dui *DUI) *image.Point {
	return kidsFirstFocus(dui, ui.Kids)
}

func (ui *Split) Focus(dui *DUI, o UI) *image.Point {
	return kidsFocus(dui, ui.Kids, o)
}

func (ui *Split) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return kidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Split) Print(self *Kid, indent int) {
	how := "horizontal"
	if ui.Vertical {
		how = "vertical"
	}
	PrintUI("Split "+how, self, indent)
	kidsPrint(ui.Kids, indent+1)
}

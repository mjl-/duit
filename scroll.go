package duit

import (
	"fmt"
	"image"

	"9fans.net/go/draw"
)

// Scroll shows a part of its single child, typically a box, and lets you scroll the content.
type Scroll struct {
	Child  UI
	Height int // < 0 means full height, 0 means as much as necessary, >0 means exactly that many lowdpi pixels

	r             image.Rectangle // entire ui
	barR          image.Rectangle
	barActiveR    image.Rectangle
	childR        image.Rectangle
	childSize     image.Point
	offset        int         // current scroll offset in pixels
	img           *draw.Image // for child to draw on
	scrollbarSize int
}

var _ UI = &Scroll{}

func NewScroll(ui UI) *Scroll {
	return &Scroll{Child: ui}
}

func (ui *Scroll) Layout(dui *DUI, size image.Point) image.Point {
	ui.scrollbarSize = scale(dui.Display, ScrollbarSize)
	scaledHeight := scale(dui.Display, ui.Height)
	if scaledHeight > 0 && scaledHeight < size.Y {
		size.Y = scaledHeight
	}
	ui.r = rect(size)
	ui.barR = ui.r
	ui.barR.Max.X = ui.barR.Min.X + ui.scrollbarSize
	ui.childR = ui.r
	ui.childR.Min.X = ui.barR.Max.X
	ui.childSize = ui.Child.Layout(dui, image.Pt(ui.r.Dx()-ui.barR.Dx(), ui.r.Dy()))
	if ui.r.Dy() > ui.childSize.Y && ui.Height == 0 {
		ui.barR.Max.Y = ui.childSize.Y
		ui.r.Max.Y = ui.childSize.Y
		ui.childR.Max.Y = ui.childSize.Y
	}
	return ui.r.Size()
}

func (ui *Scroll) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
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
	uih := ui.childSize.Y
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
	if ui.img == nil || ui.childSize != ui.img.R.Size() {
		var err error
		if ui.img != nil {
			ui.img.Free()
			ui.img = nil
		}
		ui.img, err = dui.Display.AllocImage(rect(ui.childSize), draw.ARGB32, false, dui.BackgroundColor)
		check(err, "allocimage")
	} else {
		ui.img.Draw(ui.img.R, dui.Background, nil, image.ZP)
	}
	m.Point = m.Point.Add(image.Pt(-ui.childR.Min.X, ui.offset))
	ui.Child.Draw(dui, ui.img, image.ZP, m)
	img.Draw(ui.childR.Add(orig), ui.img, nil, image.Pt(0, ui.offset))
}

func (ui *Scroll) scroll(delta int) bool {
	o := ui.offset
	ui.offset += delta
	if ui.offset < 0 {
		ui.offset = 0
	}
	offsetMax := ui.childSize.Y - ui.childR.Dy()
	if offsetMax < 0 {
		offsetMax = 0
	}
	if ui.offset > offsetMax {
		ui.offset = offsetMax
	}
	return o != ui.offset
}

func (ui *Scroll) scrollKey(c rune) (consumed bool) {
	switch c {
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
		offset := m.Y * ui.childSize.Y / ui.barR.Dy()
		offsetMax := ui.childSize.Y - ui.childR.Dy()
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

func (ui *Scroll) Mouse(dui *DUI, origM, m draw.Mouse) (r Result) {
	r.Hit = ui
	if m.Point.In(ui.barR) {
		r.Consumed = ui.scrollMouse(m, false)
		r.Draw = r.Consumed
		return
	}
	if m.Point.In(ui.r) {
		nOrigM := origM
		nOrigM.Point = nOrigM.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		nm := m
		nm.Point = nm.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r = ui.Child.Mouse(dui, nOrigM, nm)
		if !r.Consumed {
			r.Consumed = ui.scrollMouse(m, true)
			r.Draw = r.Draw || r.Consumed
		}
		return
	}
	return
}

func (ui *Scroll) Key(dui *DUI, orig image.Point, m draw.Mouse, c rune) Result {
	if m.Point.In(ui.barR) {
		consumed := ui.scrollKey(c)
		redraw := consumed
		return Result{Hit: ui, Consumed: consumed, Draw: redraw}
	}
	if m.Point.In(ui.r) {
		m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r := ui.Child.Key(dui, orig.Add(image.Pt(ui.scrollbarSize, -ui.offset)), m, c)
		if !r.Consumed {
			r.Consumed = ui.scrollKey(c)
			r.Draw = r.Draw || r.Consumed
		}
		return r
	}
	return Result{}
}

func (ui *Scroll) FirstFocus(dui *DUI) *image.Point {
	first := ui.Child.FirstFocus(dui)
	if first == nil {
		return nil
	}
	p := first.Add(image.Pt(ui.scrollbarSize, -ui.offset))
	return &p
}

func (ui *Scroll) Focus(dui *DUI, o UI) *image.Point {
	p := ui.Child.Focus(dui, o)
	if p == nil {
		return nil
	}
	pp := p.Add(image.Pt(ui.scrollbarSize, -ui.offset))
	return &pp
}

func (ui *Scroll) Print(indent int, r image.Rectangle) {
	PrintUI(fmt.Sprintf("Scroll offset=%d childR=%v childSize=%v", ui.offset, ui.childR, ui.childSize), indent, r)
	ui.Child.Print(indent+1, image.Rectangle{image.ZP, ui.childSize})
}

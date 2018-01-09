package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Label struct {
	Text string
	Font *draw.Font

	lines []string
}

var _ UI = &Label{}

func (ui *Label) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Label) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("Label", self)

	font := ui.font(dui)
	ui.lines = []string{}
	s := 0
	x := 0
	xmax := 0
	for i, c := range ui.Text {
		if c == '\n' {
			xmax = maximum(xmax, x)
			ui.lines = append(ui.lines, ui.Text[s:i])
			s = i + 1
			x = 0
			continue
		}
		dx := font.StringWidth(string(c))
		x += dx
		if i-s == 0 || x <= sizeAvail.X {
			continue
		}
		xmax = maximum(xmax, x-dx)
		ui.lines = append(ui.lines, ui.Text[s:i])
		s = i
		x = dx
	}
	if s < len(ui.Text) || s == 0 {
		ui.lines = append(ui.lines, ui.Text[s:])
		xmax = maximum(xmax, x)
	}
	self.R = rect(image.Pt(xmax, len(ui.lines)*font.Height))
}

func (ui *Label) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw("Label", self)

	p := orig
	font := ui.font(dui)
	for _, line := range ui.lines {
		img.String(p, dui.Regular.Normal.Text, image.ZP, font, line)
		p.Y += font.Height
	}
}

func (ui *Label) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	return
}

func (ui *Label) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return
}

func (ui *Label) FirstFocus(dui *DUI) *image.Point {
	return nil
}

func (ui *Label) Focus(dui *DUI, o UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

func (ui *Label) Mark(self *Kid, o UI, forLayout bool, state State) (marked bool) {
	return self.Mark(o, forLayout, state)
}

func (ui *Label) Print(self *Kid, indent int) {
	PrintUI("Label", self, indent)
}

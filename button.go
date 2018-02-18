package duit

import (
	"image"

	"9fans.net/go/draw"
)

// Button with text and optional icon with a click function.
type Button struct {
	Text     string           // Displayed on button.
	Icon     Icon             `json:"-"` // Displayed before text, if Icon.Font is not nil.
	Disabled bool             // If disabled, colors used indicate disabledness, clicks don't result in Click being called.
	Colorset *Colorset        `json:"-"` // Colors used, for example DUI.Primary. Defaults to DUI.Regular.
	Font     *draw.Font       `json:"-"` // Used to draw Text, if not nil.
	Click    func() (e Event) `json:"-"` // Called on click on the button.

	m    draw.Mouse
	size image.Point
}

var _ UI = &Button{}

func (ui *Button) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Button) space(dui *DUI) image.Point {
	return ui.padding(dui).Add(pt(BorderSize))
}

func (ui *Button) padding(dui *DUI) image.Point {
	fontHeight := ui.font(dui).Height
	return image.Pt(fontHeight/2, fontHeight/4)
}

func (ui *Button) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)

	size := ui.font(dui).StringSize(ui.Text).Add(ui.space(dui).Mul(2))
	if ui.Icon.Font != nil {
		size.X += ui.Icon.Font.StringSize(string(ui.Icon.Rune)).X
		size.X += ui.font(dui).StringSize("  ").X
	}
	ui.size = size
	self.R = rect(size)
}

func (ui *Button) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)

	text := ui.Text
	iconSize := image.ZP
	if ui.Icon.Font != nil {
		text = "  " + text
		iconSize = ui.Icon.Font.StringSize(string(ui.Icon.Rune))
	}
	textSize := ui.font(dui).StringSize(text)
	r := rect(image.Pt(iconSize.X, 0).Add(textSize).Add(ui.space(dui).Mul(2)))

	hover := m.In(r)
	var colors Colors
	if ui.Disabled {
		colors = dui.Disabled
	} else {
		cs := ui.Colorset
		if cs == nil {
			cs = &dui.Regular
		}
		colors = cs.Normal
		if hover {
			colors = cs.Hover
		}
	}

	r = r.Add(orig)
	img.Draw(r.Inset(1), colors.Background, nil, image.ZP)
	drawRoundedBorder(img, r, colors.Border)

	hit := image.ZP
	if hover && !ui.Disabled && m.Buttons&1 == 1 {
		hit = image.Pt(0, 1)
	}
	p := r.Min.Add(ui.space(dui)).Add(hit)
	if ui.Icon.Font != nil {
		dy := (iconSize.Y - textSize.Y) / 2
		img.String(p.Sub(image.Pt(0, dy)), colors.Text, image.ZP, ui.Icon.Font, string(ui.Icon.Rune))
	}
	p.X += iconSize.X
	img.String(p, colors.Text, image.ZP, ui.font(dui), text)
}

func (ui *Button) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	if ui.Disabled {
		return
	}
	rr := rect(ui.size)
	hover := m.In(rr)
	if ui.m.Buttons != m.Buttons {
		self.Draw = Dirty
	}
	if hover && ui.m.Buttons&Button1 == Button1 && m.Buttons&Button1 == 0 && ui.Click != nil {
		e := ui.Click()
		propagateEvent(self, &r, e)
	}
	ui.m = m
	return r
}

func (ui *Button) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if !ui.Disabled && (k == ' ' || k == '\n') {
		r.Consumed = true
		if ui.Click != nil {
			e := ui.Click()
			propagateEvent(self, &r, e)
		}
	}
	return
}

func (ui *Button) FirstFocus(dui *DUI, self *Kid) *image.Point {
	p := ui.space(dui)
	return &p
}

func (ui *Button) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui, self)
}

func (ui *Button) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Button) Print(self *Kid, indent int) {
	PrintUI("Button", self, indent)
}

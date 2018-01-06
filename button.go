package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Button struct {
	Text     string
	Icon     Icon // drawn before text
	Disabled bool
	Primary  bool
	Font     *draw.Font
	Click    func(r *Result)

	m draw.Mouse
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

func (ui *Button) Layout(dui *DUI, sizeAvail image.Point) (sizeTaken image.Point) {
	sizeTaken = ui.font(dui).StringSize(ui.Text).Add(ui.space(dui).Mul(2))
	if ui.Icon.Font != nil {
		sizeTaken.X += ui.Icon.Font.StringSize(string(ui.Icon.Rune)).X
		sizeTaken.X += ui.font(dui).StringSize("  ").X
	}
	return sizeTaken
}

func (ui *Button) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	text := ui.Text
	iconSize := image.ZP
	if ui.Icon.Font != nil {
		text = "  " + text
		iconSize = ui.Icon.Font.StringSize(string(ui.Icon.Rune))
	}
	textSize := ui.font(dui).StringSize(text)
	r := rect(image.Pt(iconSize.X, 0).Add(textSize).Add(ui.space(dui).Mul(2)))

	hover := m.In(r)
	colors := dui.Normal
	if ui.Disabled {
		colors = dui.Disabled
	} else if ui.Primary {
		colors = dui.Primary
	} else if hover {
		colors = dui.Hover
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

func (ui *Button) Mouse(dui *DUI, origM, m draw.Mouse) Result {
	r := Result{Hit: ui}
	if ui.m.Buttons&1 != m.Buttons&1 {
		r.Draw = true
	}
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 && ui.Click != nil && !ui.Disabled && m.Point.In(rect(ui.Layout(dui, image.ZP))) {
		ui.Click(&r)
	}
	ui.m = m
	return r
}

func (ui *Button) Key(dui *DUI, orig image.Point, m draw.Mouse, k rune) (r Result) {
	r.Hit = ui
	if !ui.Disabled && (k == ' ' || k == '\n') {
		r.Consumed = true
		if ui.Click != nil {
			ui.Click(&r)
		}
	}
	return
}

func (ui *Button) FirstFocus(dui *DUI) *image.Point {
	p := ui.space(dui)
	return &p
}

func (ui *Button) Focus(dui *DUI, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui)
}

func (ui *Button) Print(indent int, r image.Rectangle) {
	PrintUI("Button", indent, r)
}

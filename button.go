package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Button struct {
	Text     string
	Disabled bool
	Primary  bool
	Font     *draw.Font
	Click    func(r *Result)

	m draw.Mouse
}

var _ UI = &Button{}

func (ui *Button) font(env *Env) *draw.Font {
	return env.Font(ui.Font)
}

func (ui *Button) space(env *Env) image.Point {
	// padding + border
	return ui.padding(env).Add(pt(BorderSize))
}

func (ui *Button) padding(env *Env) image.Point {
	fontHeight := ui.font(env).Height
	return image.Pt(fontHeight/2, fontHeight/4)
}

func (ui *Button) Layout(env *Env, size image.Point) image.Point {
	return ui.font(env).StringSize(ui.Text).Add(ui.space(env).Mul(2))
}

func (ui *Button) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	textSize := ui.font(env).StringSize(ui.Text)
	r := rect(textSize.Add(ui.space(env).Mul(2)))

	hover := m.In(r)
	colors := env.Normal
	if ui.Disabled {
		colors = env.Disabled
	} else if ui.Primary {
		colors = env.Primary
	} else if hover {
		colors = env.Hover
	}

	r = r.Add(orig)
	img.Draw(r.Inset(1), colors.Background, nil, image.ZP)
	drawRoundedBorder(img, r, colors.Border)

	hit := image.ZP
	if hover && !ui.Disabled && m.Buttons&1 == 1 {
		hit = image.Pt(0, 1)
	}
	img.String(r.Min.Add(ui.space(env)).Add(hit), colors.Text, image.ZP, ui.font(env), ui.Text)
}

func (ui *Button) Mouse(env *Env, origM, m draw.Mouse) Result {
	r := Result{Hit: ui}
	if ui.m.Buttons&1 != m.Buttons&1 {
		r.Redraw = true
	}
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 && ui.Click != nil && !ui.Disabled && m.Point.In(rect(ui.Layout(env, image.ZP))) {
		ui.Click(&r)
	}
	ui.m = m
	return r
}

func (ui *Button) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result) {
	r.Hit = ui
	if !ui.Disabled && (k == ' ' || k == '\n') {
		r.Consumed = true
		ui.Click(&r)
	}
	return
}

func (ui *Button) FirstFocus(env *Env) *image.Point {
	p := ui.space(env)
	return &p
}

func (ui *Button) Focus(env *Env, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(env)
}

func (ui *Button) Print(indent int, r image.Rectangle) {
	uiPrint("Button", indent, r)
}

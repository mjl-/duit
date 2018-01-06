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

func (ui *Button) font(env *Env) *draw.Font {
	return env.Font(ui.Font)
}

func (ui *Button) space(env *Env) image.Point {
	return ui.padding(env).Add(pt(BorderSize))
}

func (ui *Button) padding(env *Env) image.Point {
	fontHeight := ui.font(env).Height
	return image.Pt(fontHeight/2, fontHeight/4)
}

func (ui *Button) Layout(env *Env, sizeAvail image.Point) (sizeTaken image.Point) {
	sizeTaken = ui.font(env).StringSize(ui.Text).Add(ui.space(env).Mul(2))
	if ui.Icon.Font != nil {
		sizeTaken.X += ui.Icon.Font.StringSize(string(ui.Icon.Rune)).X
		sizeTaken.X += ui.font(env).StringSize("  ").X
	}
	return sizeTaken
}

func (ui *Button) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	text := ui.Text
	iconSize := image.ZP
	if ui.Icon.Font != nil {
		text = "  " + text
		iconSize = ui.Icon.Font.StringSize(string(ui.Icon.Rune))
	}
	textSize := ui.font(env).StringSize(text)
	r := rect(image.Pt(iconSize.X, 0).Add(textSize).Add(ui.space(env).Mul(2)))

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
	p := r.Min.Add(ui.space(env)).Add(hit)
	if ui.Icon.Font != nil {
		dy := (iconSize.Y - textSize.Y) / 2
		img.String(p.Sub(image.Pt(0, dy)), colors.Text, image.ZP, ui.Icon.Font, string(ui.Icon.Rune))
	}
	p.X += iconSize.X
	img.String(p, colors.Text, image.ZP, ui.font(env), text)
}

func (ui *Button) Mouse(env *Env, origM, m draw.Mouse) Result {
	r := Result{Hit: ui}
	if ui.m.Buttons&1 != m.Buttons&1 {
		r.Draw = true
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
		if ui.Click != nil {
			ui.Click(&r)
		}
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
	PrintUI("Button", indent, r)
}

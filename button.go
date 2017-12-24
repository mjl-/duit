package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Button struct {
	Text     string
	Disabled bool
	Primary  bool
	Click    func(r *Result)

	m draw.Mouse
}

func (ui *Button) Layout(env *Env, r image.Rectangle, cur image.Point) image.Point {
	return env.Display.DefaultFont.StringSize(ui.Text).Add(pt(2 * env.Size.Space))
}

func (ui *Button) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	size := env.Display.DefaultFont.StringSize(ui.Text)

	hover := m.In(rect(size.Add(pt(2 * env.Size.Space))))
	colors := env.Normal
	if ui.Disabled {
		colors = env.Disabled
	} else if ui.Primary {
		colors = env.Primary
	} else if hover {
		colors = env.Hover
	}

	r := image.Rectangle{
		orig.Add(pt(env.Size.Margin + env.Size.Border)),
		orig.Add(size).Add(pt(env.Size.Margin + 2*env.Size.Padding + 2*env.Size.Border)),
	}
	img.Draw(r, colors.Background, nil, image.ZP)
	drawRoundedBorder(img, image.Rectangle{
		orig.Add(pt(env.Size.Margin)),
		orig.Add(size).Add(pt(2*env.Size.Space - env.Size.Margin)),
	}, colors.Border)

	hit := image.ZP
	if hover && !ui.Disabled && m.Buttons&1 == 1 {
		hit = image.Pt(0, 1)
	}
	img.String(orig.Add(pt(env.Size.Space)).Add(hit), colors.Text, image.ZP, env.Display.DefaultFont, ui.Text)
}

func (ui *Button) Mouse(env *Env, m draw.Mouse) Result {
	r := Result{Hit: ui}
	if ui.m.Buttons&1 != m.Buttons&1 {
		r.Redraw = true
	}
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 && ui.Click != nil {
		ui.Click(&r)
	}
	ui.m = m
	return r
}

func (ui *Button) Key(env *Env, orig image.Point, m draw.Mouse, c rune) Result {
	return Result{Hit: ui}
}

func (ui *Button) FirstFocus(env *Env) *image.Point {
	p := image.Pt(env.Size.Space, env.Size.Space)
	return &p
}

func (ui *Button) Focus(env *Env, o UI) *image.Point {
	if o != ui {
		return nil
	}
	p := image.Pt(env.Size.Space, env.Size.Space)
	return &p
}

func (ui *Button) Print(indent int, r image.Rectangle) {
	uiPrint("Button", indent, r)
}

package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Field struct {
	Text     string
	Disabled bool
	Changed  func(string, *Result)

	size image.Point // including space
}

func (ui *Field) Layout(env *Env, r image.Rectangle, cur image.Point) image.Point {
	ui.size = image.Point{r.Dx(), 2*env.Size.Space + env.Display.DefaultFont.Height}
	return ui.size
}

func (ui *Field) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	hover := m.In(image.Rectangle{image.ZP, ui.size})
	r := image.Rectangle{orig, orig.Add(ui.size)}

	colors := env.Normal
	if ui.Disabled {
		colors = env.Disabled
	} else if hover {
		colors = env.Hover
	}
	img.Draw(r, colors.Background, nil, image.ZP)
	drawRoundedBorder(img, image.Rectangle{
		orig.Add(image.Pt(env.Size.Margin, env.Size.Margin)),
		orig.Add(ui.size).Sub(image.Pt(env.Size.Margin, env.Size.Margin)),
	}, colors.Border)
	img.String(orig.Add(image.Point{env.Size.Space, env.Size.Space}), colors.Text, image.ZP, env.Display.DefaultFont, ui.Text)
}

func (ui *Field) Mouse(env *Env, m draw.Mouse) Result {
	return Result{Hit: ui}
}

func (ui *Field) Key(env *Env, orig image.Point, m draw.Mouse, c rune) Result {
	switch c {
	case draw.KeyPageUp, draw.KeyPageDown, draw.KeyUp, draw.KeyDown:
		return Result{Hit: ui}
	case '\t':
		return Result{Hit: ui}
	case 8:
		if ui.Text != "" {
			ui.Text = ui.Text[:len(ui.Text)-1]
		}
	default:
		ui.Text += string(c)
	}
	result := Result{Hit: ui, Consumed: true, Redraw: true}
	if ui.Changed != nil {
		ui.Changed(ui.Text, &result)
	}
	return result
}

func (ui *Field) FirstFocus(env *Env) *image.Point {
	p := image.Pt(env.Size.Space, env.Size.Space)
	return &p
}

func (ui *Field) Focus(env *Env, o UI) *image.Point {
	if o != ui {
		return nil
	}
	p := image.Pt(env.Size.Space, env.Size.Space)
	return &p
}

func (ui *Field) Print(indent int, r image.Rectangle) {
	uiPrint("Field", indent, r)
}

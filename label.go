package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Label struct {
	Text string
}

func (ui *Label) Layout(env *Env, r image.Rectangle, cur image.Point) image.Point {
	return env.Display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*env.Size.Margin + 2*env.Size.Border, 2 * env.Size.Space})
}

func (ui *Label) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.String(orig.Add(image.Point{env.Size.Margin + env.Size.Border, env.Size.Space}), env.Normal.Text, image.ZP, env.Display.DefaultFont, ui.Text)
}

func (ui *Label) Mouse(env *Env, m draw.Mouse) Result {
	return Result{Hit: ui}
}

func (ui *Label) Key(env *Env, orig image.Point, m draw.Mouse, c rune) Result {
	return Result{Hit: ui}
}

func (ui *Label) FirstFocus(env *Env) *image.Point {
	return nil
}

func (ui *Label) Focus(env *Env, o UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

func (ui *Label) Print(indent int, r image.Rectangle) {
	uiPrint("Label", indent, r)
}

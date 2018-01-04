package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Label struct {
	Text string
	Font *draw.Font
}

var _ UI = &Label{}

func (ui *Label) font(env *Env) *draw.Font {
	return env.Font(ui.Font)
}

func (ui *Label) Layout(env *Env, size image.Point) image.Point {
	return ui.font(env).StringSize(ui.Text)
}

func (ui *Label) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.String(orig, env.Normal.Text, image.ZP, ui.font(env), ui.Text)
}

func (ui *Label) Mouse(env *Env, origM, m draw.Mouse) Result {
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
	PrintUI("Label", indent, r)
}

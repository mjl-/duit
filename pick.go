package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Pick struct {
	Pick func(sizeAvail image.Point) UI
	UI   UI
}

func (ui *Pick) Layout(env *Env, sizeAvail image.Point) (sizeTaken image.Point) {
	ui.UI = ui.Pick(sizeAvail)
	return ui.UI.Layout(env, sizeAvail)
}

func (ui *Pick) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	ui.UI.Draw(env, img, orig, m)
}

func (ui *Pick) Mouse(env *Env, origM, m draw.Mouse) (r Result) {
	return ui.UI.Mouse(env, origM, m)
}

func (ui *Pick) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result) {
	return ui.UI.Key(env, orig, m, k)
}

func (ui *Pick) FirstFocus(env *Env) (warp *image.Point) {
	return ui.UI.FirstFocus(env)
}

func (ui *Pick) Focus(env *Env, o UI) (warp *image.Point) {
	return ui.UI.Focus(env, o)
}

func (ui *Pick) Print(indent int, r image.Rectangle) {
	PrintUI("Pick", indent, r)
	ui.UI.Print(indent+1, r)
}

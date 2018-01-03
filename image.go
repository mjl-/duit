package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Image struct {
	Image *draw.Image
}

var _ UI = &Image{}

func (ui *Image) Layout(env *Env, size image.Point) image.Point {
	return ui.Image.R.Size()
}

func (ui *Image) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(image.Rectangle{orig, orig.Add(ui.Image.R.Size())}, ui.Image, nil, image.ZP)
}

func (ui *Image) Mouse(env *Env, origM, m draw.Mouse) Result {
	return Result{Hit: ui}
}

func (ui *Image) Key(env *Env, orig image.Point, m draw.Mouse, c rune) Result {
	return Result{Hit: ui}
}

func (ui *Image) FirstFocus(env *Env) *image.Point {
	return nil
}

func (ui *Image) Focus(env *Env, o UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

func (ui *Image) Print(indent int, r image.Rectangle) {
	uiPrint("Image", indent, r)
}

package duit

import (
	"fmt"
	"image"

	"9fans.net/go/draw"
)

type Grid struct {
	Kids    []*Kid
	Columns int

	widths  []int
	heights []int
	size    image.Point
}

var _ UI = &Grid{}

func (ui *Grid) Layout(env *Env, size image.Point) image.Point {
	ui.widths = make([]int, ui.Columns)
	width := 0
	for col := 0; col < ui.Columns; col++ {
		ui.widths[col] = 0
		for i := col; i < len(ui.Kids); i += ui.Columns {
			k := ui.Kids[i]
			childSize := k.UI.Layout(env, image.Pt(size.X-width, size.Y))
			if childSize.X > ui.widths[col] {
				ui.widths[col] = childSize.X
			}
		}
		width += ui.widths[col]
	}

	ui.heights = make([]int, (len(ui.Kids)+ui.Columns-1)/ui.Columns)
	height := 0
	for i := 0; i < len(ui.Kids); i += ui.Columns {
		row := i / ui.Columns
		ui.heights[row] = 0
		for col := 0; col < ui.Columns; col++ {
			k := ui.Kids[i+col]
			childSize := k.UI.Layout(env, image.Pt(ui.widths[col], size.Y))
			if childSize.Y > ui.heights[row] {
				ui.heights[row] = childSize.Y
			}
		}
		height += ui.heights[row]
	}

	x := make([]int, len(ui.widths))
	for col := range x {
		if col > 0 {
			x[col] = x[col-1] + ui.widths[col-1]
		}
	}
	y := make([]int, len(ui.heights))
	for row := range y {
		if row > 0 {
			y[row] = y[row-1] + ui.heights[row-1]
		}
	}

	for i, k := range ui.Kids {
		row := i / ui.Columns
		col := i % ui.Columns
		p := image.Pt(x[col], y[row])
		k.r = image.Rectangle{p, p.Add(image.Pt(ui.widths[col], ui.heights[row]))}
	}

	ui.size = image.Pt(width, height)
	return ui.size
}

func (ui *Grid) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(env, ui.Kids, ui.size, img, orig, m)
}

func (ui *Grid) Mouse(env *Env, m draw.Mouse) (result Result) {
	return kidsMouse(env, ui.Kids, m)
}

func (ui *Grid) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (result Result) {
	return kidsKey(env, ui, ui.Kids, orig, m, k)
}

func (ui *Grid) FirstFocus(env *Env) *image.Point {
	return kidsFirstFocus(env, ui.Kids)
}

func (ui *Grid) Focus(env *Env, o UI) *image.Point {
	return kidsFocus(env, ui.Kids, o)
}

func (ui *Grid) Print(indent int, r image.Rectangle) {
	uiPrint(fmt.Sprintf("Grid columns=%d", ui.Columns), indent, r)
	kidsPrint(ui.Kids, indent+1)
}

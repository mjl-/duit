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

func (ui *Grid) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	r.Min = image.Pt(0, cur.Y)

	ui.widths = make([]int, ui.Columns)
	width := 0
	for col := 0; col < ui.Columns; col++ {
		ui.widths[col] = 0
		for i := col; i < len(ui.Kids); i += ui.Columns {
			k := ui.Kids[i]
			kr := image.Rectangle{image.ZP, image.Pt(r.Dx()-width, r.Dy())}
			size := k.UI.Layout(display, kr, image.ZP)
			if size.X > ui.widths[col] {
				ui.widths[col] = size.X
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
			kr := image.Rectangle{image.ZP, image.Pt(ui.widths[col], r.Dy())}
			size := k.UI.Layout(display, kr, image.ZP)
			if size.Y > ui.heights[row] {
				ui.heights[row] = size.Y
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
func (ui *Grid) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(display, ui.Kids, ui.size, img, orig, m)
}
func (ui *Grid) Mouse(m draw.Mouse) (result Result) {
	return kidsMouse(ui.Kids, m)
}
func (ui *Grid) Key(orig image.Point, m draw.Mouse, k rune) (result Result) {
	return kidsKey(ui, ui.Kids, orig, m, k)
}
func (ui *Grid) FirstFocus() *image.Point {
	return kidsFirstFocus(ui.Kids)
}
func (ui *Grid) Focus(o UI) *image.Point {
	return kidsFocus(ui.Kids, o)
}
func (ui *Grid) Print(indent int, r image.Rectangle) {
	uiPrint(fmt.Sprintf("Grid columns=%d", ui.Columns), indent, r)
	kidsPrint(ui.Kids, indent+1)
}

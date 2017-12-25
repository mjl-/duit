package duit

import (
	"fmt"
	"image"

	"9fans.net/go/draw"
)

type Grid struct {
	Kids    []*Kid
	Columns int
	Valign  []Valign
	Halign  []Halign
	Padding image.Point // in low DPI pixels, will be adjusted for high DPI

	widths  []int
	heights []int
	size    image.Point
}

var _ UI = &Grid{}

func (ui *Grid) Layout(env *Env, size image.Point) image.Point {
	if ui.Valign != nil && len(ui.Valign) != ui.Columns {
		panic(fmt.Sprintf("len(valign) = %d, should be ui.Columns = %d", len(ui.Valign), ui.Columns))
	}
	if ui.Halign != nil && len(ui.Halign) != ui.Columns {
		panic(fmt.Sprintf("len(halign) = %d, should be ui.Columns = %d", len(ui.Halign), ui.Columns))
	}

	ui.widths = make([]int, ui.Columns)         // widths include padding
	padding := scalePt(env.Display, ui.Padding) // single padding
	pad2 := padding.Mul(2)
	width := 0                       // total width so far
	x := make([]int, len(ui.widths)) // x offsets per column
	x[0] = 0

	// first determine the column widths
	for col := 0; col < ui.Columns; col++ {
		if col > 0 {
			x[col] = x[col-1] + ui.widths[col-1]
		}
		ui.widths[col] = 0
		newDx := 0
		for i := col; i < len(ui.Kids); i += ui.Columns {
			k := ui.Kids[i]
			childSize := k.UI.Layout(env, image.Pt(size.X-width-pad2.X, size.Y-pad2.Y))
			if childSize.X+pad2.X > newDx {
				newDx = childSize.X + pad2.X
			}
		}
		ui.widths[col] = newDx
		width += ui.widths[col]
	}

	// now determine row heights
	ui.heights = make([]int, (len(ui.Kids)+ui.Columns-1)/ui.Columns)
	height := 0                       // total height so far
	y := make([]int, len(ui.heights)) // including padding
	y[0] = 0
	for i := 0; i < len(ui.Kids); i += ui.Columns {
		row := i / ui.Columns
		if row > 0 {
			y[row] = y[row-1] + ui.heights[row-1]
		}
		rowDy := 0
		for col := 0; col < ui.Columns; col++ {
			k := ui.Kids[i+col]
			childSize := k.UI.Layout(env, image.Pt(ui.widths[col]-pad2.X, size.Y-y[row]-pad2.Y))
			offset := image.Pt(x[col], y[row]).Add(padding)
			k.r = rect(childSize).Add(offset) // aligned in top left, fixed for halign/valign later on
			if childSize.Y+pad2.Y > rowDy {
				rowDy = childSize.Y + pad2.Y
			}
		}
		ui.heights[row] = rowDy
		height += ui.heights[row]
	}

	// now shift the kids for right valign/halign
	for i, k := range ui.Kids {
		row := i / ui.Columns
		col := i % ui.Columns

		valign := ValignTop
		halign := HalignLeft
		if ui.Valign != nil {
			valign = ui.Valign[col]
		}
		if ui.Halign != nil {
			halign = ui.Halign[col]
		}
		cellSize := image.Pt(ui.widths[col], ui.heights[row]).Sub(padding.Mul(2))
		spaceX := 0
		switch halign {
		case HalignLeft:
		case HalignMiddle:
			spaceX = (cellSize.X - k.r.Dx()) / 2
		case HalignRight:
			spaceX = cellSize.X - k.r.Dx()
		}
		spaceY := 0
		switch valign {
		case ValignTop:
		case ValignMiddle:
			spaceY = (cellSize.Y - k.r.Dy()) / 2
		case ValignBottom:
			spaceY = cellSize.Y - k.r.Dy()
		}
		k.r = k.r.Add(image.Pt(spaceX, spaceY))
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
	uiPrint(fmt.Sprintf("Grid columns=%d padding=%v", ui.Columns, ui.Padding), indent, r)
	kidsPrint(ui.Kids, indent+1)
}

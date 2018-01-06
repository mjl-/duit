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
	Padding []Space // in low DPI pixels
	Width   int     // -1 means full width, 0 means automatic width, >0 means exactly that many lowdpi pixels

	widths  []int
	heights []int
	size    image.Point
}

var _ UI = &Grid{}

func (ui *Grid) Layout(dui *DUI, size image.Point) image.Point {
	if ui.Valign != nil && len(ui.Valign) != ui.Columns {
		panic(fmt.Sprintf("len(valign) = %d, should be ui.Columns = %d", len(ui.Valign), ui.Columns))
	}
	if ui.Halign != nil && len(ui.Halign) != ui.Columns {
		panic(fmt.Sprintf("len(halign) = %d, should be ui.Columns = %d", len(ui.Halign), ui.Columns))
	}
	if ui.Padding != nil && len(ui.Padding) != ui.Columns {
		panic(fmt.Sprintf("len(padding) = %d, should be ui.Columns = %d", len(ui.Padding), ui.Columns))
	}
	if len(ui.Kids)%ui.Columns != 0 {
		panic(fmt.Sprintf("len(kids) = %d, should be multiple of ui.Columns = %d", len(ui.Kids), ui.Columns))
	}

	scaledWidth := dui.Scale(ui.Width)
	if scaledWidth > 0 && scaledWidth < size.X {
		ui.size.X = scaledWidth
	}

	ui.widths = make([]int, ui.Columns) // widths include padding
	spaces := make([]Space, ui.Columns)
	if ui.Padding != nil {
		for i, pad := range ui.Padding {
			spaces[i] = dui.ScaleSpace(pad)
		}
	}
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
		space := spaces[col]
		for i := col; i < len(ui.Kids); i += ui.Columns {
			k := ui.Kids[i]
			childSize := k.UI.Layout(dui, image.Pt(size.X-width-space.Dx(), size.Y-space.Dy()))
			if childSize.X+space.Dx() > newDx {
				newDx = childSize.X + space.Dx()
			}
		}
		ui.widths[col] = newDx
		width += ui.widths[col]
	}
	if scaledWidth < 0 && width < size.X {
		leftover := size.X - width
		given := 0
		for i, _ := range ui.widths {
			x[i] += given
			var dx int
			if i == len(ui.widths)-1 {
				dx = leftover - given
			} else {
				dx = leftover / len(ui.widths)
			}
			ui.widths[i] += dx
			given += dx
		}
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
			space := spaces[col]
			k := ui.Kids[i+col]
			childSize := k.UI.Layout(dui, image.Pt(ui.widths[col]-space.Dx(), size.Y-y[row]-space.Dy()))
			offset := image.Pt(x[col], y[row]).Add(space.Topleft())
			k.R = rect(childSize).Add(offset) // aligned in top left, fixed for halign/valign later on
			if childSize.Y+space.Dy() > rowDy {
				rowDy = childSize.Y + space.Dy()
			}
		}
		ui.heights[row] = rowDy
		height += ui.heights[row]
	}

	// now shift the kids for right valign/halign
	for i, k := range ui.Kids {
		row := i / ui.Columns
		col := i % ui.Columns
		space := spaces[col]

		valign := ValignTop
		halign := HalignLeft
		if ui.Valign != nil {
			valign = ui.Valign[col]
		}
		if ui.Halign != nil {
			halign = ui.Halign[col]
		}
		cellSize := image.Pt(ui.widths[col], ui.heights[row]).Sub(space.Size())
		spaceX := 0
		switch halign {
		case HalignLeft:
		case HalignMiddle:
			spaceX = (cellSize.X - k.R.Dx()) / 2
		case HalignRight:
			spaceX = cellSize.X - k.R.Dx()
		}
		spaceY := 0
		switch valign {
		case ValignTop:
		case ValignMiddle:
			spaceY = (cellSize.Y - k.R.Dy()) / 2
		case ValignBottom:
			spaceY = cellSize.Y - k.R.Dy()
		}
		k.R = k.R.Add(image.Pt(spaceX, spaceY))
	}

	ui.size = image.Pt(width, height)
	if ui.Width < 0 && ui.size.X < size.X {
		ui.size.X = size.X
	}
	return ui.size
}

func (ui *Grid) Draw(dui *DUI, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(dui, ui.Kids, ui.size, img, orig, m)
}

func (ui *Grid) Mouse(dui *DUI, origM, m draw.Mouse) (result Result) {
	return kidsMouse(dui, ui.Kids, origM, m)
}

func (ui *Grid) Key(dui *DUI, orig image.Point, m draw.Mouse, k rune) (result Result) {
	return kidsKey(dui, ui, ui.Kids, orig, m, k)
}

func (ui *Grid) FirstFocus(dui *DUI) *image.Point {
	return kidsFirstFocus(dui, ui.Kids)
}

func (ui *Grid) Focus(dui *DUI, o UI) *image.Point {
	return kidsFocus(dui, ui.Kids, o)
}

func (ui *Grid) Print(indent int, r image.Rectangle) {
	PrintUI(fmt.Sprintf("Grid columns=%d padding=%v", ui.Columns, ui.Padding), indent, r)
	kidsPrint(ui.Kids, indent+1)
}

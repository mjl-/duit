package duit

import (
	"fmt"
	"image"
	"strings"

	"9fans.net/go/draw"
)

const (
	// note: these are not adjusted for low/hidpi, we want them as slim as possible
	separatorWidth  = 1
	separatorHeight = 1
)

type Gridrow struct {
	Selected bool
	Values   []string
	Value    interface{}
}

type Gridlist struct {
	Header   Gridrow
	Rows     []*Gridrow
	Multiple bool
	Halign   []Halign
	Padding  Space // in low DPI pixels
	Striped  bool
	Font     *draw.Font

	Changed func(index int, r *Result)
	Click   func(index, buttons int, r *Result)
	Keys    func(index int, m draw.Mouse, k rune, r *Result)

	m                draw.Mouse
	colWidths        []int // set the first time there are rows
	size             image.Point
	draggingColStart int         // x offset of column being dragged, so 1 means the first column is being dragged.
	cellImage        *draw.Image // scratch image to draw cells on if they are too big
}

var _ UI = &Gridlist{}

func (ui *Gridlist) font(env *Env) *draw.Font {
	return env.Font(ui.Font)
}

// rowHeight without separator
func (ui *Gridlist) rowHeight(env *Env) int {
	return ui.font(env).Height + env.ScaleSpace(ui.Padding).Dy()
}

func (ui *Gridlist) makeWidthOffsets(env *Env, widths []int) []int {
	offsets := make([]int, len(widths))
	pad := env.ScaleSpace(ui.Padding)
	for i := range widths {
		if i > 0 {
			offsets[i] = offsets[i-1] + widths[i-1] + pad.Dx() + separatorWidth
		}
	}
	return offsets
}

func (ui *Gridlist) columnWidths(env *Env, width int) []int {
	if ui.colWidths != nil {
		if width == ui.size.X {
			return ui.colWidths
		}
		// log.Printf("making new columns, ui.size.X %d, width %d\n", ui.size.X, width)

		// reassign sizes, same relative size, just new absolute widths
		ncol := len(ui.Header.Values)
		pad := env.ScaleSpace(ui.Padding)
		avail := width - ncol*pad.Dx() - (ncol-1)*separatorWidth
		prevTotal := 0
		for _, v := range ui.colWidths {
			prevTotal += v
		}
		oavail := avail
		for i, v := range ui.colWidths {
			dx := oavail * v / prevTotal
			avail -= dx
			ui.colWidths[i] = dx
		}
		ui.colWidths[0] += avail
		return ui.colWidths
	}

	makeWidths := func(rows []*Gridrow) ([]int, bool) {
		// first determine max & avg size of first 50 columns. there is always at least one row.
		if len(rows) > 50 {
			rows = rows[:50]
		}
		font := ui.font(env)
		ncol := len(rows[0].Values)
		max := make([]int, ncol)
		avg := make([]int, ncol)
		maxTotal := 0
		for _, row := range rows {
			for col, v := range row.Values {
				dx := font.StringWidth(v)
				if dx > max[col] {
					max[col] = dx
				}
				avg[col] += dx // divided by rows later
			}
		}
		for i := range avg {
			avg[i] /= len(rows)
		}
		for _, v := range max {
			maxTotal += v
		}

		// log.Printf("making widths, ncol %d, max %v, avg %v, maxTotal %d, width avail %d\n", ncol, max, avg, maxTotal, width)

		// give out minimum width to all cols
		pad := env.ScaleSpace(ui.Padding)
		minWidth := font.StringWidth("mmm")

		widths := make([]int, ncol)
		for i := range widths {
			widths[i] = minWidth
		}

		overhead := ncol*pad.Dx() - (ncol-1)*separatorWidth
		remain := width - ncol*minWidth - overhead
		// log.Printf("gave minwidths, widths %v, remain %d\n", widths, remain)

		// then see if we can fit them all
		need := 0
		for i := range widths {
			dx := max[i] - widths[i]
			if dx > 0 {
				need += dx
			}
		}
		if need <= remain {
			for i := range widths {
				dx := max[i] - widths[i]
				if dx > 0 {
					widths[i] += dx
					remain -= dx
				}
			}
			// log.Printf("cols did fit, widths %v, remain %d\n", widths, remain)
		}

		// then give half remaining width to cols that would then fit without growing them to twice their previous size
		give := remain / 2
		for i := range widths {
			if widths[i] >= max[i] || 2*widths[i] < max[i] {
				continue
			}
			dx := max[i] - widths[i]
			if dx > give {
				dx = give
			}
			widths[i] += dx
			give -= dx
			if give <= 0 {
				break
			}
		}
		remain = remain - remain/2 + give
		// log.Printf("gave half remainig to fit small cols, widths %v, remain %d\n", widths, remain)

		// give remaining half evenly based on average size of columns that don't yet fit
		avgTotal := 0
		for i := range widths {
			if widths[i] >= max[i] {
				continue
			}
			avgTotal += avg[i]
		}
		if avgTotal > 0 {
			oremain := remain
			for i := range widths {
				if widths[i] >= max[i] {
					continue
				}
				dx := oremain * avg[i] / avgTotal
				if dx > max[i]-widths[i] {
					dx = max[i] - widths[i]
				}
				widths[i] += dx
				remain -= dx
			}
			// log.Printf("gave remaining to non-fitting, widths %v, remain %d\n", widths, remain)
		}

		oremain := remain
		for i := range widths {
			dx := oremain * max[i] / maxTotal
			widths[i] += dx
			remain -= dx
		}
		widths[0] += remain
		// log.Printf("gave remaining, widths %v, remain %d\n", widths, remain)
		total := 0
		for _, w := range widths {
			total += w
		}
		if total != width-overhead {
			panic(fmt.Sprintf("widths don't add up, total %d, width %d - overhead %d = %d\n", total, width, overhead, width-overhead))
		}
		fit := true
		for i, w := range widths {
			if w < max[i] {
				fit = false
				break
			}
		}
		return widths, fit
	}

	if len(ui.Rows) == 0 {
		widths, _ := makeWidths([]*Gridrow{&ui.Header})
		return widths
	}
	var fit bool
	ui.colWidths, fit = makeWidths(ui.Rows)
	if fit {
		widths, fit := makeWidths(append([]*Gridrow{&ui.Header}, ui.Rows...))
		if fit {
			ui.colWidths = widths
		}
	}
	return ui.colWidths
}

func (ui *Gridlist) Layout(env *Env, sizeAvail image.Point) (sizeTaken image.Point) {
	if ui.Halign != nil && len(ui.Halign) != len(ui.Header.Values) {
		panic(fmt.Sprintf("len(halign) = %d, should be len(ui.Header.Values) = %d", len(ui.Halign), len(ui.Header.Values)))
	}

	n := 1 + len(ui.Rows)
	ui.columnWidths(env, sizeAvail.X) // calculate widths, possibly remembering
	ui.size = image.Pt(sizeAvail.X, n*ui.rowHeight(env)+(n-1)*separatorHeight)
	return ui.size
}

func (ui *Gridlist) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	ncol := len(ui.Header.Values)
	if ncol == 0 {
		return
	}

	r := rect(ui.size).Add(orig)

	rowHeight := ui.rowHeight(env)
	pad := env.ScaleSpace(ui.Padding)

	widths := ui.columnWidths(env, ui.size.X) // widths, excluding separator and padding
	x := ui.makeWidthOffsets(env, widths)

	font := ui.font(env)
	rowSize := image.Pt(r.Dx(), rowHeight)
	lineR := rect(rowSize).Add(orig)

	ensureCellImage := func(size image.Point) *draw.Image {
		if ui.cellImage != nil {
			csize := ui.cellImage.R.Size()
			if csize.X >= size.X && csize.Y >= size.Y {
				return ui.cellImage
			}
		}
		maxDx := 0
		for _, dx := range widths {
			if dx > maxDx {
				maxDx = dx
			}
		}
		var err error
		ui.cellImage, err = env.Display.AllocImage(rect(image.Pt(maxDx, size.Y)), draw.ARGB32, false, draw.Transparent)
		check(err, "allocimage")
		return ui.cellImage
	}

	drawRow := func(row *Gridrow, odd bool) {
		if len(row.Values) != ncol {
			panic(fmt.Sprintf("row with wrong number of values, expect %d, saw %d", ncol, len(row.Values)))
		}
		colors := env.Normal
		if row.Selected {
			colors = env.Inverse
			img.Draw(lineR, colors.Background, nil, image.ZP)
		} else if odd && ui.Striped {
			colors = env.Striped
			img.Draw(lineR, colors.Background, nil, image.ZP)
		}
		for i, s := range row.Values {
			cellR := lineR
			cellR.Min.X = lineR.Min.X + x[i] + separatorWidth
			cellR.Max.X = cellR.Min.X + widths[i] + pad.Dx()
			cellR = pad.Inset(cellR)
			alignOffset := pt(0)
			dx := font.StringWidth(s)
			if ui.Halign != nil {
				leftover := widths[i] - dx
				switch ui.Halign[i] {
				case HalignLeft:
				case HalignMiddle:
					alignOffset.X += leftover / 2
				case HalignRight:
					alignOffset.X += leftover
				default:
					panic(fmt.Sprintf("unknown halign %d", ui.Halign[i]))
				}
			}
			if dx > widths[i] {
				cellImg := ensureCellImage(cellR.Size())
				cellImg.Draw(cellImg.R, colors.Background, nil, image.ZP)
				cellImg.String(alignOffset, colors.Text, image.ZP, font, s)
				img.Draw(cellR, cellImg, nil, image.ZP)
			} else {
				img.String(cellR.Min.Add(alignOffset), colors.Text, image.ZP, font, s)
			}
		}
		lineR = lineR.Add(image.Pt(0, rowHeight+separatorHeight))
	}

	drawRow(&ui.Header, false)

	// print separators
	for i := 1; i < ncol; i++ {
		p0 := image.Pt(x[i], 0).Add(orig).Add(image.Pt(0, pad.Top))
		p1 := p0
		p1.Y += rowHeight - pad.Dy()
		img.Line(p0, p1, 0, 0, 0, env.Normal.Border, image.ZP)
	}
	lp0 := lineR.Min.Sub(image.Pt(0, separatorHeight))
	lp1 := lp0
	lp1.X += r.Dx()
	img.Line(lp0, lp1, 0, 0, 0, env.Normal.Border, image.ZP)

	for i, row := range ui.Rows {
		drawRow(row, i%2 == 1)
	}
}

func (ui *Gridlist) Mouse(env *Env, origM, m draw.Mouse) (r Result) {
	r.Hit = ui
	prevM := ui.m
	ui.m = m
	if !m.In(rect(ui.size)) {
		return
	}
	rowHeight := ui.rowHeight(env)
	index := m.Y / (rowHeight + separatorHeight)
	if ui.draggingColStart > 0 || index == 0 {
		// xxx todo: on double click, max column before fit (but at most twice as large)
		// xxx todo: should probably show the grid separator with hover style

		b1 := m.Buttons&Button1 == 1
		if !b1 {
			ui.draggingColStart = 0
			return
		}
		widths := ui.columnWidths(env, ui.size.X)
		offsets := ui.makeWidthOffsets(env, widths)
		if ui.draggingColStart > 0 {
			// user was dragging, move the grid sizes
			dx := m.X - offsets[ui.draggingColStart]
			widths[ui.draggingColStart] -= dx
			widths[ui.draggingColStart-1] += dx

			// might have to move other columns
			if dx > 0 {
				// ui.draggingColStart became smaller, must check if later ones still have positive size
				for i := ui.draggingColStart; i < len(widths)-1 && widths[i] < 0; i++ {
					dx = -widths[i]
					widths[i] = 0
					widths[i+1] -= dx
				}
			} else {
				// ui.draggingColStart-1 became smaller
				for i := ui.draggingColStart - 1; i > 0 && widths[i] < 0; i-- {
					dx = -widths[i]
					widths[i] = 0
					widths[i-1] -= dx
				}
			}

			ui.colWidths = widths // note: this sets colWidths even if it wasn't set before
			r.Consumed = true
			r.Draw = true
			return
		}

		// start dragging, find the column if any
		slack := ui.font(env).StringWidth("x")
		for i, x := range offsets {
			x -= m.X
			if x >= -slack && x <= slack {
				ui.draggingColStart = i
				r.Consumed = true
				return
			}
		}

		return
	}
	index-- // adjust for header
	if m.Buttons != 0 && prevM.Buttons^m.Buttons != 0 && ui.Click != nil {
		ui.Click(index, m.Buttons, &r)
	}
	if !r.Consumed && prevM.Buttons == 0 && m.Buttons == Button1 {
		row := ui.Rows[index]
		row.Selected = !row.Selected
		if row.Selected && !ui.Multiple {
			for _, vv := range ui.Rows {
				if vv != row {
					vv.Selected = false
				}
			}
		}
		if ui.Changed != nil {
			ui.Changed(index, &r)
		}
		r.Draw = true
		r.Consumed = true
	}
	return
}

func (ui *Gridlist) selectedIndices() (l []int) {
	for i, row := range ui.Rows {
		if row.Selected {
			l = append(l, i)
		}
	}
	return
}

func (ui *Gridlist) Selected() (indices []int) {
	return ui.selectedIndices()
}

func (ui *Gridlist) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result) {
	r.Hit = ui
	if !m.In(rect(ui.size)) {
		return
	}
	if ui.Keys != nil {
		// xxx what should "index" be? especially for multiple: true...
		sel := ui.selectedIndices()
		index := -1
		if len(sel) == 1 {
			index = sel[0]
		}
		ui.Keys(index, m, k, &r)
		if r.Consumed {
			return
		}
	}
	switch k {
	case draw.KeyCmd + 'n':
		// clear selection
		for _, row := range ui.Rows {
			row.Selected = false
		}
		r.Consumed = true
		r.Draw = true
	case draw.KeyCmd + 'a':
		// select all
		for _, row := range ui.Rows {
			row.Selected = true
		}
		r.Consumed = true
		r.Draw = true
	case draw.KeyCmd + 'c':
		// snarf selection
		s := ""
		for _, row := range ui.Rows {
			if !row.Selected {
				continue
			}
			s += strings.Join(row.Values, "\t") + "\n"
		}
		if s != "" {
			env.Display.WriteSnarf([]byte(s))
			r.Consumed = true
			r.Draw = true
		}

	case draw.KeyUp, draw.KeyDown:
		if len(ui.Rows) == 0 {
			return
		}
		sel := ui.selectedIndices()
		oindex := -1
		nindex := -1
		switch k {
		case draw.KeyUp:
			r.Consumed = true
			if len(sel) == 0 {
				nindex = len(ui.Rows) - 1
			} else {
				oindex = sel[0]
				nindex = (sel[0] - 1 + len(ui.Rows)) % len(ui.Rows)
			}
		case draw.KeyDown:
			r.Consumed = true
			if len(sel) == 0 {
				nindex = 0
			} else {
				oindex = sel[len(sel)-1]
				nindex = (sel[len(sel)-1] + 1) % len(ui.Rows)
			}
		}
		if oindex >= 0 {
			ui.Rows[oindex].Selected = false
			r.Draw = true
		}
		if nindex >= 0 {
			font := ui.font(env)
			rowHeight := ui.rowHeight(env)
			pad := env.ScaleSpace(ui.Padding)

			ui.Rows[nindex].Selected = true
			r.Draw = true
			if ui.Changed != nil {
				ui.Changed(nindex, &r)
			}
			// xxx orig probably should not be a part in this...
			p := orig.Add(image.Pt(m.X, (1+nindex)*(rowHeight+separatorHeight)+(font.Height+pad.Dy())/2))
			r.Warp = &p
		}
	}
	return
}

func (ui *Gridlist) FirstFocus(env *Env) (warp *image.Point) {
	return &image.ZP
}

func (ui *Gridlist) Focus(env *Env, o UI) (warp *image.Point) {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(env)
}

func (ui *Gridlist) Print(indent int, r image.Rectangle) {
	PrintUI("Gridlist", indent, r)
}

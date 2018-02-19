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

// Gridrow is used for each row in a Gridlist.
type Gridrow struct {
	Selected bool        // If currently selected.
	Values   []string    // Values displayed in the row.
	Value    interface{} `json:"-"` // Auxiliary data.
}

// Gridfit is the layout strategy for a Gridlist.
type Gridfit byte

const (
	FitNormal Gridfit = iota // FitNormal lays out over full available width.
	FitSlim                  // FitSlim lays out only as much as needed.
)

// Gridlist is a table-like list of selectable values.
// Currently each cell in each row is drawn as a single-line string.
// Column widths can be adjusted by dragging the separator in the header.
//
// Keys:
// 	arrow up, move selection up
// 	arrow down, move selection down
// 	home, move selection to first element
// 	end, move selection to last element
// 	cmd-n, clear selection
// 	cmd-a, select all
// 	cmd-c, copy selected rows, as tab-separated values
type Gridlist struct {
	Header   *Gridrow   // Optional header to display at the the top.
	Rows     []*Gridrow // Rows, each holds whether it is selected.
	Multiple bool       // Whether multiple rows can be selected at a time.
	Halign   []Halign   // Horizontal alignment for the values.
	Padding  Space      // Padding for each cell, in lowDPI pixels.
	Striped  bool       // If set, odd cells have a slightly contrasting background color.
	Fit      Gridfit    // Layout strategy, how much space columns receive.
	Font     *draw.Font `json:"-"` // Used for drawing text.

	Changed func(index int) (e Event)               `json:"-"` // Called after the selection changed. -1 is multiple may have changed.
	Click   func(index int, m draw.Mouse) (e Event) `json:"-"` // Called on click at given index. If consumed, processing stops.
	Keys    func(k rune, m draw.Mouse) (e Event)    `json:"-"` // Called before handling a key event. If consumed, processing stops.

	m                draw.Mouse
	colWidths        []int // set the first time there are rows
	size             image.Point
	draggingColStart int         // x offset of column being dragged, so 1 means the first column is being dragged.
	cellImage        *draw.Image // scratch image to draw cells on if they are too big
}

var _ UI = &Gridlist{}

func (ui *Gridlist) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

// rowHeight without separator
func (ui *Gridlist) rowHeight(dui *DUI) int {
	return ui.font(dui).Height + dui.ScaleSpace(ui.Padding).Dy()
}

func (ui *Gridlist) makeWidthOffsets(dui *DUI, widths []int) []int {
	offsets := make([]int, len(widths))
	pad := dui.ScaleSpace(ui.Padding)
	for i := range widths {
		if i > 0 {
			offsets[i] = offsets[i-1] + widths[i-1] + pad.Dx() + separatorWidth
		}
	}
	return offsets
}

func (ui *Gridlist) columnWidths(dui *DUI, width int) []int {
	if ui.colWidths != nil {
		if width == ui.size.X || ui.Fit == FitSlim {
			return ui.colWidths
		}
		// log.Printf("making new columns, ui.size.X %d, width %d\n", ui.size.X, width)

		// reassign sizes, same relative size, just new absolute widths
		row := ui.exampleRow()
		ncol := len(row.Values)
		pad := dui.ScaleSpace(ui.Padding)
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

	if ui.Fit == FitSlim {
		row := ui.exampleRow()
		if row == nil {
			return nil
		}

		widths := make([]int, len(row.Values))
		font := ui.font(dui)
		updateWidths := func(row *Gridrow) {
			for i, s := range row.Values {
				widths[i] = maximum(widths[i], font.StringWidth(s))
			}
		}

		if ui.Header != nil {
			updateWidths(ui.Header)
		}
		for _, row := range ui.Rows {
			updateWidths(row)
		}
		left := width
		for i := range widths {
			widths[i] = minimum(widths[i], left)
			left -= widths[i]
		}
		if len(ui.Rows) > 0 {
			ui.colWidths = widths
		}
		return widths
	}

	makeWidths := func(rows []*Gridrow) ([]int, bool) {
		if len(rows[0].Values) == 0 {
			panic("makeWidths on empty rows")
		}

		// first determine max & avg size of first 50 columns. there is always at least one row.
		if len(rows) > 50 {
			rows = rows[:50]
		}
		font := ui.font(dui)
		ncol := len(rows[0].Values)
		max := make([]int, ncol)
		avg := make([]int, ncol)
		maxTotal := 0
		for _, row := range rows {
			for col, v := range row.Values {
				dx := font.StringWidth(v)
				max[col] = maximum(max[col], dx)
				avg[col] += dx // divided by rows later
			}
		}
		for i := range avg {
			avg[i] /= len(rows)
		}
		for _, v := range max {
			maxTotal += v
		}
		if maxTotal == 0 {
			return nil, false
		}

		// log.Printf("making widths, ncol %d, max %v, avg %v, maxTotal %d, width avail %d\n", ncol, max, avg, maxTotal, width)

		// give out minimum width to all cols
		pad := dui.ScaleSpace(ui.Padding)
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
			dx = minimum(dx, give)
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
				dx = minimum(dx, max[i]-widths[i])
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
		if ui.Header == nil {
			return nil
		}
		widths, _ := makeWidths([]*Gridrow{ui.Header})
		return widths
	}
	var fit bool
	ui.colWidths, fit = makeWidths(ui.Rows)
	if fit && ui.Header != nil {
		widths, fit := makeWidths(append([]*Gridrow{ui.Header}, ui.Rows...))
		if fit {
			ui.colWidths = widths
		}
	}
	return ui.colWidths
}

func (ui *Gridlist) exampleRow() *Gridrow {
	if ui.Header != nil {
		return ui.Header
	}
	if len(ui.Rows) == 0 {
		return nil
	}
	return ui.Rows[0]
}

func (ui *Gridlist) rowCount() int {
	n := len(ui.Rows)
	if ui.Header != nil {
		n++
	}
	return n
}

func (ui *Gridlist) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)

	row := ui.exampleRow()
	if ui.Halign != nil && row != nil && len(ui.Halign) != len(row.Values) {
		panic(fmt.Sprintf("len(halign) = %d, should be len(row.Values) = %d", len(ui.Halign), len(row.Values)))
	}

	n := ui.rowCount()
	ui.columnWidths(dui, sizeAvail.X) // calculate widths, possibly remembering
	ui.size = image.Pt(sizeAvail.X, n*ui.rowHeight(dui)+(n-1)*separatorHeight)
	self.R = rect(ui.size)
}

func (ui *Gridlist) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)

	row := ui.exampleRow()
	if row == nil || len(row.Values) == 0 {
		return
	}
	ncol := len(row.Values)

	r := rect(ui.size).Add(orig)

	rowHeight := ui.rowHeight(dui)
	pad := dui.ScaleSpace(ui.Padding)

	widths := ui.columnWidths(dui, ui.size.X) // widths, excluding separator and padding
	x := ui.makeWidthOffsets(dui, widths)

	font := ui.font(dui)
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
			maxDx = maximum(maxDx, dx)
		}
		var err error
		ui.cellImage, err = dui.Display.AllocImage(rect(image.Pt(maxDx, size.Y)), draw.ARGB32, false, draw.Transparent)
		if dui.error(err, "allocimage") {
			return nil
		}
		return ui.cellImage
	}

	drawRow := func(row *Gridrow, odd bool) {
		if len(row.Values) != ncol {
			panic(fmt.Sprintf("row with wrong number of values, expect %d, saw %d", ncol, len(row.Values)))
		}
		colors := dui.Regular.Normal
		if row.Selected {
			colors = dui.Inverse
			img.Draw(lineR, colors.Background, nil, image.ZP)
		} else if odd && ui.Striped {
			colors = dui.Striped
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
				if cellImg == nil {
					return
				}
				cellImg.Draw(cellImg.R, colors.Background, nil, image.ZP)
				cellImg.String(alignOffset, colors.Text, image.ZP, font, s)
				img.Draw(cellR, cellImg, nil, image.ZP)
			} else {
				img.String(cellR.Min.Add(alignOffset), colors.Text, image.ZP, font, s)
			}
		}
		lineR = lineR.Add(image.Pt(0, rowHeight+separatorHeight))
	}

	if ui.Header != nil {
		drawRow(ui.Header, false)
		// print separators
		for i := 1; i < ncol; i++ {
			p0 := image.Pt(x[i], 0).Add(orig).Add(image.Pt(0, pad.Top))
			p1 := p0
			p1.Y += rowHeight - pad.Dy()
			img.Line(p0, p1, 0, 0, 0, dui.Regular.Normal.Border, image.ZP)
		}
		lp0 := lineR.Min.Sub(image.Pt(0, separatorHeight))
		lp1 := lp0
		lp1.X += r.Dx()
		img.Line(lp0, lp1, 0, 0, 0, dui.Regular.Normal.Border, image.ZP)
	}

	for i, row := range ui.Rows {
		drawRow(row, i%2 == 1)
	}
}

func (ui *Gridlist) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	prevM := ui.m
	ui.m = m
	if !m.In(rect(ui.size)) {
		return
	}
	rowHeight := ui.rowHeight(dui)
	index := m.Y / (rowHeight + separatorHeight)
	if ui.draggingColStart > 0 || (index == 0 && ui.Header != nil) {
		// xxx todo: on double click, max column before fit (but at most twice as large)
		// xxx todo: should probably show the grid separator with hover style

		b1 := m.Buttons&Button1 == 1
		if !b1 {
			ui.draggingColStart = 0
			return
		}
		widths := ui.columnWidths(dui, ui.size.X)
		offsets := ui.makeWidthOffsets(dui, widths)
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
			self.Draw = Dirty
			return
		}

		// start dragging, find the column if any
		slack := ui.font(dui).StringWidth("x")
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
	if ui.Header != nil {
		index--
	}
	if m.Buttons != 0 && prevM.Buttons^m.Buttons != 0 && ui.Click != nil {
		e := ui.Click(index, m)
		propagateEvent(self, &r, e)
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
			e := ui.Changed(index)
			propagateEvent(self, &r, e)
		}
		self.Draw = Dirty
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

func (ui *Gridlist) firstSelected() int {
	for i, row := range ui.Rows {
		if row.Selected {
			return i
		}
	}
	return -1
}

func (ui *Gridlist) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if !m.In(rect(ui.size)) {
		return
	}
	if ui.Keys != nil {
		e := ui.Keys(k, m)
		propagateEvent(self, &r, e)
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
		if ui.Changed != nil {
			ui.Changed(-1)
		}
		r.Consumed = true
		self.Draw = Dirty
	case draw.KeyCmd + 'a':
		// select all
		for _, row := range ui.Rows {
			row.Selected = true
		}
		if ui.Changed != nil {
			ui.Changed(-1)
		}
		r.Consumed = true
		self.Draw = Dirty
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
			dui.WriteSnarf([]byte(s))
			r.Consumed = true
			self.Draw = Dirty
		}

	case draw.KeyUp, draw.KeyDown, draw.KeyHome, draw.KeyEnd:
		if len(ui.Rows) == 0 {
			return
		}
		sel := ui.selectedIndices()
		oindex := -1
		nindex := -1
		switch k {
		case draw.KeyUp:
			if len(sel) == 0 {
				nindex = 0
			} else {
				oindex = sel[0]
				nindex = maximum(0, sel[0]-1)
			}
		case draw.KeyDown:
			if len(sel) == 0 {
				nindex = 0
			} else {
				oindex = sel[len(sel)-1]
				nindex = minimum(sel[len(sel)-1]+1, len(ui.Rows)-1)
			}
		case draw.KeyHome:
			nindex = 0
		case draw.KeyEnd:
			nindex = len(ui.Rows) - 1
		}
		r.Consumed = oindex != nindex
		if !r.Consumed {
			return
		}
		if oindex >= 0 {
			ui.Rows[oindex].Selected = false
			self.Draw = Dirty
		}
		if nindex >= 0 {
			font := ui.font(dui)
			rowHeight := ui.rowHeight(dui)
			pad := dui.ScaleSpace(ui.Padding)

			ui.Rows[nindex].Selected = true
			self.Draw = Dirty
			if ui.Changed != nil {
				e := ui.Changed(nindex)
				propagateEvent(self, &r, e)
			}
			// xxx orig probably should not be a part in this...
			n := nindex
			if ui.Header != nil {
				n++
			}
			p := orig.Add(image.Pt(m.X, n*(rowHeight+separatorHeight)+(font.Height+pad.Dy())/2))
			r.Warp = &p
		}
	}
	return
}

func (ui *Gridlist) FirstFocus(dui *DUI, self *Kid) (warp *image.Point) {
	i := maximum(0, ui.firstSelected())
	if ui.Header != nil {
		i++
	}
	// focus on first selected item
	p := image.Pt(0, i*(ui.rowHeight(dui)+separatorHeight))
	return &p
}

func (ui *Gridlist) Focus(dui *DUI, self *Kid, o UI) (warp *image.Point) {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui, self)
}

func (ui *Gridlist) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Gridlist) Print(self *Kid, indent int) {
	PrintUI("Gridlist", self, indent)
}

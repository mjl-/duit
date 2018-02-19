package duit

import (
	"image"
	"strings"
	"unicode/utf8"

	"9fans.net/go/draw"
)

// Cursor and SelectionStart start at 1 for sane behaviour of an empty Field struct.

// Field is a single line text field. The cursor is always visible, and determines which part of the text is shown.
type Field struct {
	Text            string                               // Current text.
	Placeholder     string                               // Text displayed in lighter color as example.
	Disabled        bool                                 // If disabled, mouse and key input have no effect.
	Cursor1         int                                  // Index in string of cursor in bytes, start at 1, 0 means end of string.
	SelectionStart1 int                                  // If > 0, 1 beyond the start of the selection in bytes, with Cursor being the end.
	Password        bool                                 // Render text as bullet items to hide the password (but not length).
	Font            *draw.Font                           `json:"-"` // Font to use for drawing text.
	Changed         func(text string) (e Event)          `json:"-"` // Called after contents of field have changed.
	Keys            func(k rune, m draw.Mouse) (e Event) `json:"-"` // Called before handling key. If you consume the event, Changed will not be called.

	size            image.Point // including space
	m               draw.Mouse
	prevB1Release   draw.Mouse
	img             *draw.Image // in case text is too big
	prevTextOffset  int         // offset for text for previous draw, used to determine whether to realign the cursor
	lastCursorPoint image.Point // location of last cursor draw, for FirstFocus() and cmd+t
}

var _ UI = &Field{}

func (ui *Field) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Field) padding(dui *DUI) image.Point {
	fontHeight := ui.font(dui).Height
	return image.Pt(fontHeight/4, fontHeight/4)
}

func (ui *Field) space(dui *DUI) image.Point {
	// padding + border
	return ui.padding(dui).Add(pt(1))
}

// cursor adjusted to start at 0 index
func (ui *Field) cursor0() int {
	ui.fixCursor()
	if ui.Cursor1 == 0 {
		return len(ui.Text)
	}
	return ui.Cursor1 - 1
}

// selection with start & end with 0 indices
func (ui *Field) selection0() (start int, end int, text string) {
	if ui.SelectionStart1 <= 0 {
		return 0, 0, ""
	}
	s, e := ui.cursor0(), ui.SelectionStart1-1
	if s > e {
		s, e = e, s
	}
	return s, e, ui.Text[s:e]
}

func (ui *Field) removeSelection() {
	if ui.SelectionStart1 <= 0 {
		return
	}
	s, e, _ := ui.selection0()
	ui.Text = ui.Text[:s] + ui.Text[e:]
	ui.Cursor1 = 1 + s
	ui.SelectionStart1 = 0
}

func (ui *Field) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)

	ui.size = image.Point{sizeAvail.X, ui.font(dui).Height + 2*ui.space(dui).Y}
	self.R = rect(ui.size)
	return
}

func (ui *Field) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)

	if ui.size.X <= 0 || ui.size.Y <= 0 {
		return
	}
	r := rect(ui.size)
	hover := m.In(r)
	r = r.Add(orig)

	ui.fixCursor()
	s, e, sel := ui.selection0()
	f := ui.font(dui)

	colors := dui.Regular.Normal
	selColors := dui.Selection
	if ui.Disabled {
		colors = dui.Disabled
	} else if hover {
		colors = dui.Regular.Hover
		selColors = dui.SelectionHover
	}
	text := ui.Text
	c0 := ui.cursor0()
	if text == "" {
		text = ui.Placeholder
		if !ui.Disabled {
			colors = dui.Placeholder
			if hover {
				colors.Border = dui.Regular.Hover.Border
			}
		}
	} else if ui.Password {
		// ugh
		nt := ""
		sel = ""
		inSel := false
		ns := -1
		ne := -1
		nc0 := -1
		for o := range text {
			if s == o {
				ns = len(nt)
				inSel = true
			}
			if e == o {
				ne = len(nt)
				inSel = false
			}
			if c0 == o {
				nc0 = len(nt)
			}
			nt += "•"
			if inSel {
				sel += "•"
			}
		}
		if nc0 < 0 {
			nc0 = len(nt)
		}
		if ns < 0 {
			ns = len(nt)
		}
		if ne < 0 {
			ne = len(nt)
		}
		text, s, e, c0 = nt, ns, ne, nc0
	}
	img.Draw(r, colors.Background, nil, image.ZP)
	drawRoundedBorder(img, r, colors.Border)

	space := ui.space(dui)

	drawString := func(i *draw.Image, p, cp image.Point) {
		p = p.Add(space)
		if sel == "" {
			i.String(p, colors.Text, image.ZP, f, text)
		} else {
			before := text[:s]
			after := text[e:]
			p = i.String(p, colors.Text, image.ZP, f, before)
			selR := outsetPt(rect(f.StringSize(sel)).Add(p), image.Pt(0, space.Y/2))
			i.Draw(selR, selColors.Background, nil, image.ZP)
			p = i.String(p, selColors.Text, image.ZP, f, sel)
			i.String(p, colors.Text, image.ZP, f, after)
		}

		if hover && !ui.Disabled {
			// draw cursor
			cp = cp.Add(space)
			cp1 := cp
			cp1.Y += f.Height
			i.Line(cp, cp1, 1, 1, 0, dui.Regular.Hover.Border, image.ZP)
			ui.lastCursorPoint = cp1.Sub(orig)
		}
	}

	width := f.StringWidth(text)
	if width <= r.Dx()-2*space.X {
		cp := r.Min.Add(image.Pt(f.StringWidth(text[:c0]), 0))
		drawString(img, r.Min, cp)
	} else {
		if ui.img == nil || !ui.img.R.Size().Eq(ui.size) {
			var err error
			ui.img, err = dui.Display.AllocImage(rect(ui.size), draw.ARGB32, false, draw.Transparent)
			if dui.error(err, "allocimage") {
				return
			}
		}
		ui.img.Draw(ui.img.R, colors.Background, nil, image.ZP)

		// first, determine cursor given previous draw
		width := ui.img.R.Dx() - 2*space.X
		stringWidth := f.StringWidth(text[:c0])
		cursorOffset := stringWidth + ui.prevTextOffset
		var textOffset int
		if cursorOffset < 0 {
			// before start, realign to left
			textOffset = -stringWidth
			cursorOffset = 0
		} else if cursorOffset > width {
			// after start, realign to right
			textOffset = width - stringWidth
			cursorOffset = width - 1
		} else {
			// don't reallign
			textOffset = ui.prevTextOffset
		}

		drawString(ui.img, image.Pt(textOffset, 0), image.Pt(cursorOffset, 0))
		img.Draw(SpacePt(space).Inset(r), ui.img, nil, space)
		ui.prevTextOffset = textOffset
	}
}

func expandSelection(t string, i int) (s, e int) {
	if i == 0 || i == len(t) {
		return 0, len(t)
	}

	const (
		Starts = "[{(<\"'`"
		Ends   = "]})>\"'`"
	)

	index := strings.IndexByte(Starts, t[i-1])
	if index >= 0 {
		s = i
		e = s
		n := len(t)
		up := Starts[index]
		down := Ends[index]
		nested := 1
		for {
			if e >= n {
				return i, i
			}
			// note: order of comparison matters, for quotes, down is the same as up
			if t[e] == down {
				nested--
			} else if t[e] == up {
				nested++
			}
			if nested == 0 {
				return
			}
			e++
		}
	}

	index = strings.IndexByte(Ends, t[i])
	if index >= 0 {
		e = i
		s = i - 1
		up := Ends[index]
		down := Starts[index]
		nested := 1
		for {
			if s == 0 {
				return i, i
			}
			// note: order of comparison matters, for quotes, down is the same as up
			if t[s] == down {
				nested--
			} else if t[s] == up {
				nested++
			}
			if nested == 0 {
				s++
				return
			}
			s--
		}
	}

	s = i
	e = i

	const Space = " \t\r\n\f"
	skip := func(isSpace bool) bool {
		return !isSpace
	}

	if strings.ContainsAny(t[s-1:s], Space) && strings.ContainsAny(t[e:e+1], Space) {
		skip = func(isSpace bool) bool {
			return isSpace
		}
	}
	for ; s > 0 && skip(strings.ContainsAny(t[s-1:s], Space)) && !strings.ContainsAny(t[s-1:s], Starts+Ends); s-- {
	}
	for ; e < len(t) && skip(strings.ContainsAny(t[e:e+1], Space)) && !strings.ContainsAny(t[e:e+1], Starts+Ends); e++ {
	}
	return
}

func (ui *Field) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	if ui.Disabled {
		return
	}
	if !origM.In(rect(ui.size)) {
		return
	}
	space := ui.space(dui)
	locateCursor := func() int {
		f := ui.font(dui)
		mX := m.X - space.X - ui.prevTextOffset
		x := 0
		for i, c := range ui.Text {
			if ui.Password {
				c = '•'
			}
			dx := f.StringWidth(string(c))
			if mX <= x+dx/2 {
				return i
			}
			x += dx
		}
		return len(ui.Text)
	}
	if ui.m.Buttons&1 == 0 && m.Buttons&1 == 1 {
		// b1 down, start selection
		ui.Cursor1 = 1 + locateCursor()
		ui.SelectionStart1 = ui.Cursor1
		r.Consumed = true
		self.Draw = Dirty
	} else if ui.m.Buttons&1 == 1 || m.Buttons&1 == 1 {
		// continue selection
		ui.Cursor1 = 1 + locateCursor()
		r.Consumed = true
		self.Draw = Dirty
		if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 {
			if m.Msec-ui.prevB1Release.Msec < 400 {
				s, e := expandSelection(ui.Text, ui.cursor0())
				ui.Cursor1 = 1 + s
				ui.SelectionStart1 = 1 + e
			}
			ui.prevB1Release = m
		}
	}
	ui.m = m
	return
}

func (ui *Field) fixCursor() {
	if ui.Cursor1 < 0 {
		ui.Cursor1 = 1
	}
	ui.Cursor1 = minimum(ui.Cursor1, 1+len(ui.Text))
	if ui.SelectionStart1 < 0 {
		ui.SelectionStart1 = 0
	}
	if ui.SelectionStart1-1 > len(ui.Text) {
		ui.SelectionStart1 = len(ui.Text) + 1
	}
}

func (ui *Field) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if ui.Disabled {
		return
	}
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

	origText := ui.Text

	const Ctrl = 0x1f
	ui.fixCursor()
	cursor0 := ui.cursor0()

	cursorPrev := func() int {
		_, n := utf8.DecodeLastRuneInString(ui.Text[:cursor0])
		return cursor0 - n
	}
	cursorNext := func() int {
		_, n := utf8.DecodeRuneInString(ui.Text[cursor0:])
		return cursor0 + n
	}
	removeSelection := func() int {
		ui.removeSelection()
		ui.fixCursor()
		return ui.cursor0()
	}
	switch k {
	case draw.KeyPageUp, draw.KeyPageDown, draw.KeyUp, draw.KeyDown, '\t':
		return
	case draw.KeyLeft:
		cursor0 = cursorPrev()
		ui.SelectionStart1 = 0
	case draw.KeyRight:
		cursor0 = cursorNext()
		ui.SelectionStart1 = 0
	case Ctrl & 'a', draw.KeyHome:
		cursor0 = 0
		ui.SelectionStart1 = 0
	case Ctrl & 'e', draw.KeyEnd:
		cursor0 = len(ui.Text)
		ui.SelectionStart1 = 0

	case Ctrl & 'h':
		// remove char before cursor0
		cursor0 = removeSelection()
		if cursor0 > 0 {
			prev := cursorPrev()
			ui.Text = ui.Text[:cursorPrev()] + ui.Text[cursor0:]
			cursor0 = prev
		}
	case Ctrl & 'w':
		// remove to start of space+word
		cursor0 = removeSelection()
		for cursor0 > 0 && strings.ContainsAny(ui.Text[cursorPrev():cursor0], " \t\r\n") {
			cursor0 = cursorPrev()
		}
		for cursor0 > 0 && !strings.ContainsAny(ui.Text[cursorPrev():cursor0], " \t\r\n") {
			cursor0 = cursorPrev()
		}
		ui.Text = ui.Text[:cursor0]
	case Ctrl & 'u':
		// remove entire line
		cursor0 = removeSelection()
		ui.Text = ""
		cursor0 = 0
	case Ctrl & 'k':
		// remove to end of line
		cursor0 = removeSelection()
		ui.Text = ui.Text[:cursor0]

	case draw.KeyDelete:
		// remove char after cursor0
		cursor0 = removeSelection()
		if cursor0 < len(ui.Text) {
			ui.Text = ui.Text[:cursor0] + ui.Text[cursorNext():]
		}

	case draw.KeyCmd + 'a':
		// select all
		cursor0 = 0
		ui.SelectionStart1 = 1 + len(ui.Text)

	case draw.KeyCmd + 'c':
		_, _, t := ui.selection0()
		if t != "" {
			dui.WriteSnarf([]byte(t))
		}

	case draw.KeyCmd + 'x':
		s, e, t := ui.selection0()
		if t != "" {
			dui.WriteSnarf([]byte(t))
			ui.Text = ui.Text[:s] + ui.Text[e:]
			cursor0 = s
			ui.SelectionStart1 = 0
		}

	case draw.KeyCmd + 'v':
		cursor0 = removeSelection()
		buf, ok := dui.ReadSnarf()
		if ok {
			t := string(buf)
			ui.Text = ui.Text[:cursor0] + t + ui.Text[cursor0:]
			ui.SelectionStart1 = 1 + cursor0
			cursor0 = 1 + cursor0 + len(t)
		}

	case draw.KeyCmd + 'm':
		p := ui.lastCursorPoint.Add(orig)
		r.Warp = &p

	case '\n':
		return

	default:
		if k >= draw.KeyCmd && k < draw.KeyCmd+128 {
			return
		}
		cursor0 = removeSelection()
		ks := string(k)
		if cursor0 >= len(ui.Text) {
			ui.Text += ks
		} else {
			ui.Text = ui.Text[:cursor0] + ks + ui.Text[cursor0:]
		}
		cursor0 += len(ks)
	}
	ui.Cursor1 = 1 + cursor0
	ui.fixCursor()
	r.Consumed = true
	self.Draw = Dirty
	if ui.Changed != nil && origText != ui.Text {
		e := ui.Changed(ui.Text)
		propagateEvent(self, &r, e)
	}
	return
}

func (ui *Field) FirstFocus(dui *DUI, self *Kid) *image.Point {
	p := ui.lastCursorPoint
	if p == image.ZP {
		p = ui.space(dui)
	}
	return &p
}

func (ui *Field) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui, self)
}

func (ui *Field) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Field) Print(self *Kid, indent int) {
	PrintUI("Field", self, indent)
}

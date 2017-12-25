package duit

import (
	"image"
	"log"
	"strings"

	"9fans.net/go/draw"
)

// Cursor and SelectionStart start at 1 for sane behaviour of an empty Field struct.

type Field struct {
	Text            string
	Disabled        bool
	Cursor1         int                                   // index in string of cursor, start at 1. 0 means end of string.
	SelectionStart1 int                                   // if > 0, 1 beyond the start of the selection, with Cursor being the end.
	Changed         func(string, *Result)                 // called after contents of field have changed
	Keys            func(m draw.Mouse, k rune, r *Result) // called before handling key. if you consume the event, Changed will not be called

	size          image.Point // including space
	m             draw.Mouse
	prevB1Release draw.Mouse
}

var _ UI = &Field{}

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

func (ui *Field) Layout(env *Env, size image.Point) image.Point {
	ui.size = image.Point{size.X, 2*env.Size.Space + env.Display.DefaultFont.Height}
	return ui.size
}

func (ui *Field) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	r := rect(ui.size)
	hover := m.In(r)
	r = r.Add(orig)

	colors := env.Normal
	invColors := env.Normal
	if ui.Disabled {
		colors = env.Disabled
	} else if hover {
		colors = env.Hover
		invColors = env.Inverse
	}
	img.Draw(r, colors.Background, nil, image.ZP)
	drawRoundedBorder(img, r, colors.Border)

	s, e, sel := ui.selection0()
	tp := orig.Add(pt(env.Size.Space))
	f := env.Display.DefaultFont
	if sel != "" {
		before := ui.Text[:s]
		after := ui.Text[e:]
		tp = img.String(tp, colors.Text, image.ZP, f, before)
		selR := outsetPt(rect(f.StringSize(sel)).Add(tp), image.Pt(0, env.Size.Space/2))
		img.Draw(selR, invColors.Background, nil, image.ZP)
		tp = img.String(tp, invColors.Text, image.ZP, f, sel)
		img.String(tp, colors.Text, image.ZP, f, after)
	} else {
		img.String(tp, colors.Text, image.ZP, f, ui.Text)
	}

	if hover && !ui.Disabled {
		ui.fixCursor()
		f := env.Display.DefaultFont
		p0 := r.Min.Add(pt(env.Size.Space))
		p0.X += f.StringWidth(ui.Text[:ui.cursor0()])
		p1 := p0
		p1.Y += f.Height
		img.Line(p0, p1, 1, 1, 0, env.Hover.Border, image.ZP)
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

func (ui *Field) Mouse(env *Env, m draw.Mouse) (r Result) {
	if !m.In(rect(ui.size)) {
		return
	}
	r.Hit = ui
	locateCursor := func() int {
		f := env.Display.DefaultFont
		n := len(ui.Text)
		mX := m.X - env.Size.Space
		for i := 0; i < n; i++ {
			x := f.StringWidth(ui.Text[:i])
			if mX <= x {
				return i
			}
		}
		return len(ui.Text)
	}
	if ui.m.Buttons&1 == 0 && m.Buttons&1 == 1 {
		// b1 down, start selection
		ui.Cursor1 = 1 + locateCursor()
		ui.SelectionStart1 = ui.Cursor1
		r.Consumed = true
		r.Redraw = true
	} else if ui.m.Buttons&1 == 1 || m.Buttons&1 == 1 {
		// continue selection
		ui.Cursor1 = 1 + locateCursor()
		r.Consumed = true
		r.Redraw = true
		if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 {
			if m.Msec-ui.prevB1Release.Msec < 400 {
				s, e := expandSelection(ui.Text, ui.cursor0())
				ui.Cursor1 = 1 + s
				ui.SelectionStart1 = 1 + e
				log.Printf("double click\n")
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
	if ui.Cursor1 > 1+len(ui.Text) {
		ui.Cursor1 = 1 + len(ui.Text)
	}
	if ui.SelectionStart1 < 0 {
		ui.SelectionStart1 = 0
	}
	if ui.SelectionStart1-1 > len(ui.Text) {
		ui.SelectionStart1 = len(ui.Text) + 1
	}
}

func (ui *Field) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result) {
	if !m.In(rect(ui.size)) {
		return
	}
	r.Hit = ui
	if ui.Disabled {
		return
	}

	if ui.Keys != nil {
		ui.Keys(m, k, &r)
		if r.Consumed {
			return
		}
	}

	origText := ui.Text

	const Ctrl = 0x1f
	ui.fixCursor()
	cursor0 := ui.cursor0()
	switch k {
	case draw.KeyPageUp, draw.KeyPageDown, draw.KeyUp, draw.KeyDown, '\t':
		return Result{Hit: ui}
	case draw.KeyLeft:
		cursor0--
		ui.SelectionStart1 = 0
	case draw.KeyRight:
		cursor0++
		ui.SelectionStart1 = 0
	case Ctrl & 'a':
		cursor0 = 0
		ui.SelectionStart1 = 0
	case Ctrl & 'e':
		cursor0 = len(ui.Text)
		ui.SelectionStart1 = 0

	case Ctrl & 'h':
		// remove char before cursor0
		ui.removeSelection()
		if cursor0 > 0 {
			ui.Text = ui.Text[:cursor0-1] + ui.Text[cursor0:]
			cursor0--
		}
	case Ctrl & 'w':
		// remove to start of space+word
		ui.removeSelection()
		for cursor0 > 0 && strings.ContainsAny(ui.Text[cursor0-1:cursor0], " \t\r\n") {
			cursor0--
		}
		for cursor0 > 0 && !strings.ContainsAny(ui.Text[cursor0-1:cursor0], " \t\r\n") {
			cursor0--
		}
		ui.Text = ui.Text[:cursor0]
	case Ctrl & 'u':
		// remove entire line
		ui.removeSelection()
		ui.Text = ""
		cursor0 = 0
	case Ctrl & 'k':
		// remove to end of line
		ui.removeSelection()
		ui.Text = ui.Text[cursor0:]

	case draw.KeyDelete:
		// remove char after cursor0
		ui.removeSelection()
		if cursor0 < len(ui.Text) {
			ui.Text = ui.Text[:cursor0] + ui.Text[cursor0+1:]
		}

	case draw.KeyCmd + 'a':
		// select all
		cursor0 = 0
		ui.SelectionStart1 = 1 + len(ui.Text)

	case draw.KeyCmd + 'c':
		_, _, t := ui.selection0()
		if t != "" {
			env.Display.WriteSnarf([]byte(t))
		}

	case draw.KeyCmd + 'x':
		s, e, t := ui.selection0()
		if t != "" {
			env.Display.WriteSnarf([]byte(t))
			ui.Text = ui.Text[:s] + ui.Text[e:]
			cursor0 = s
			ui.SelectionStart1 = 0
		}

	case draw.KeyCmd + 'v':
		ui.removeSelection()
		buf := make([]byte, 128)
		have, total, err := env.Display.ReadSnarf(buf)
		if err != nil {
			log.Printf("duit: readsnarf: %s\n", err)
			break
		}
		var t string
		if have >= total {
			t = string(buf[:have])
		} else {
			buf = make([]byte, total)
			have, _, err = env.Display.ReadSnarf(buf)
			if err != nil {
				log.Printf("duit: readsnarf entire buffer: %s\n", err)
			}
			t = string(buf[:have])
		}
		ui.Text = ui.Text[:cursor0] + t + ui.Text[cursor0:]

	case '\n':
		return

	default:
		ui.removeSelection()
		if cursor0 > len(ui.Text) {
			ui.Text += string(k)
		} else {
			ui.Text = ui.Text[:cursor0] + string(k) + ui.Text[cursor0:]
		}
		cursor0 += 1
	}
	ui.Cursor1 = 1 + cursor0
	ui.fixCursor()
	r.Consumed = true
	r.Redraw = true
	if ui.Changed != nil && origText != ui.Text {
		ui.Changed(ui.Text, &r)
	}
	return
}

func (ui *Field) FirstFocus(env *Env) *image.Point {
	return &image.ZP
}

func (ui *Field) Focus(env *Env, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(env)
}

func (ui *Field) Print(indent int, r image.Rectangle) {
	uiPrint("Field", indent, r)
}

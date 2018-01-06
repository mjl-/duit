package duit

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

var (
	cmdBadNumber  = errors.New("bad number")
	cmdUnfinished = errors.New("command not completed")
	cmdNoNumber   = errors.New("no number allowed")
	cmdBadMove    = errors.New("bad move")
)

type cmd struct {
	r         *strings.Reader
	number    int
	numberStr string
}

func (c *cmd) get(peek bool) (rune, bool) {
	r, n, _ := c.r.ReadRune()
	if n > 0 {
		if peek {
			c.r.UnreadRune()
		}
		return r, false
	}
	return 0, true
}

func (c *cmd) Number() {
	c.numberStr = ""
	for {
		r, eof := c.get(true)
		if eof || !(r >= '1' && r <= '9' || c.numberStr != "" && r == '0') {
			break
		}
		c.numberStr += string(r)
		c.get(false)
	}
	if c.numberStr == "" {
		c.number = 1
		return
	}
	v, err := strconv.ParseInt(c.numberStr, 10, 32)
	if err != nil {
		panic(cmdBadNumber)
	}
	if v == 0 {
		panic(cmdBadNumber)
	}
	c.number = int(v)
}

func (c *cmd) Get() rune {
	r, eof := c.get(false)
	if eof {
		panic(cmdUnfinished)
	}
	return r
}

func (c *cmd) NoNumber() {
	if c.number != 1 {
		panic(cmdNoNumber)
	}
}

func (c *cmd) Times(fn func()) {
	for i := 0; i < c.number; i++ {
		fn()
	}
}

// commandMove returns new file offset after applying moves.
// it panics with cmd* errors on invalid/incomplete.
func (ui *Edit) commandMove(dui *DUI, cmd *cmd, br, fr *reader, endLineChar rune) int64 {
	r := cmd.Get()
	// log.Printf("commandMove, number %d, r %c\n", cmd.number, r)
	switch r {
	case '0':
		// start of line
		br.Line(false)
		return br.Offset()
	case '$':
		// end of line
		cmd.NoNumber()
		fr.Line(false)
		return fr.Offset()
	case 'w':
		// word or interpunction
		cmd.Times(func() {
			o := fr.Offset()
			fr.Nonwhitespacepunct()
			if o == fr.Offset() {
				fr.TryGet()
			}
			fr.Whitespace()
		})
		return fr.Offset()
	case 'W':
		// word
		cmd.Times(func() {
			fr.Nonwhitespace()
			fr.Whitespace()
		})
		return fr.Offset()
	case 'b':
		// to begin of (previous) word
		cmd.Times(func() {
			br.Whitespace()
			br.Nonwhitespacepunct()
		})
		return br.Offset()
	case 'B':
		// like 'b', skip interpunction too
		cmd.Times(func() {
			br.Whitespace()
			br.Nonwhitespace()
		})
		return br.Offset()
	case 'e':
		// to end of (next) word
		cmd.Times(func() {
			fr.Whitespace()
			fr.Nonwhitespacepunct()
		})
		return fr.Offset()
	case 'E':
		// like 'e', skip interpunction too
		cmd.Times(func() {
			fr.Whitespace()
			fr.Nonwhitespace()
		})
		return fr.Offset()
	case 'h':
		// left
		cmd.Times(func() {
			br.TryGet()
		})
		return br.Offset()
	case 'l':
		// right
		cmd.Times(func() {
			fr.TryGet()
		})
		return fr.Offset()
	case 'k':
		// up
		runes, _, _ := br.Line(false)
		cmd.Times(func() {
			br.Line(true)
			br.Line(false)
		})
		rr := ui.reader(br.Offset(), ui.text.Size())
		for ; runes > 0; runes-- {
			c, eof := rr.Peek()
			if eof || c == '\n' {
				break
			}
			rr.Get()
		}
		return rr.Offset()
	case 'j':
		// down
		runes, _, _ := br.Line(false)
		cmd.Times(func() {
			fr.Line(true)
		})
		for ; runes > 0; runes-- {
			c, eof := fr.Peek()
			if eof || c == '\n' {
				break
			}
			fr.Get()
		}
		return fr.Offset()
	case 'G':
		if cmd.numberStr == "" {
			// to eof
			return ui.text.Size()
		}
		// to absolute line number (1 is first)
		r := ui.reader(0, ui.text.Size())
		for i := 1; i < cmd.number; i++ {
			_, _, eof := r.Line(true)
			if eof {
				break
			}
		}
		return r.Offset()
	case '%':
		// to matching struct key
		cmd.NoNumber()
		c, eof := fr.Peek()
		if eof {
			break
		}
		const Starts = "{[(<"
		const Ends = "}])>"
		if index := strings.IndexRune(Starts, c); index >= 0 {
			if ui.expandNested(fr, rune(Starts[index]), rune(Ends[index])) > 0 {
				return fr.Offset()
			}

		} else if index = strings.IndexRune(Ends, c); index >= 0 {
			if ui.expandNested(br, rune(Ends[index]), rune(Starts[index])) > 0 {
				return br.Offset()
			}
		}
	default:
		if r == endLineChar {
			br.Line(false)
			cmd.Times(func() {
				fr.Line(true)
			})
			return fr.Offset()
		}
	}
	panic(cmdBadMove)
}

func (ui *Edit) visualKey(dui *DUI, k rune, line bool, result *Result) {
	ui.visual += string(k)
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		switch err {
		case cmdUnfinished:
			return
		case cmdBadNumber, cmdBadMove, cmdNoNumber:
			ui.visual = ""
			return
		default:
			panic(err)
		}
	}()
	cmd := &cmd{r: strings.NewReader(ui.visual), number: 1}

	fr := ui.reader(ui.cursor, ui.text.Size())
	br := ui.revReader(ui.cursor)

	c0, c1 := ui.orderedCursor()

	switch k {
	case 'i':
		ui.mode = ModeInsert
	case 'd':
		ui.text.Replace(c0, c1, nil)
		ui.cursor0 = c0
		ui.cursor = c0
	case 's':
		ui.text.Replace(c0, c1, nil)
		ui.mode = ModeInsert
		ui.cursor0 = c0
		ui.cursor = c0
	case 'y':
		s := ui.readText(c0, c1)
		ui.writeSnarf(dui, []byte(s))
	case 'p':
		buf, ok := ui.readSnarf(dui)
		if ok {
			ui.text.Replace(c0, c1, buf)
			ui.cursor0 = c0
			ui.cursor = c0 + int64(len(buf))
		}
	case '<':
		n := ui.unindent(c0, c1)
		ui.cursor0 = c0
		ui.cursor = c0 + n
	case '>':
		n := ui.indent(c0, c1)
		ui.cursor0 = c0
		ui.cursor = c0 + n
	case 'J':
		s := ui.readText(c0, c1)
		newline := s != "" && s[len(s)-1] == '\n'
		if newline {
			s = s[:len(s)-1]
		}
		s = strings.Replace(s, "\n", " ", -1)
		if newline {
			s += "\n"
		}
		ui.text.Replace(c0, c1, []byte(s))
		ui.cursor0 = c0
		ui.cursor = ui.cursor0 + int64(len(s))
	case '~':
		s := ""
		for _, r := range ui.readText(c0, c1) {
			if unicode.IsUpper(r) {
				r = unicode.ToLower(r)
			} else if unicode.IsLower(r) {
				r = unicode.ToUpper(r)
			}
			s += string(r)
		}
		ui.text.Replace(c0, c1, []byte(s))
		ui.cursor0 = c0
		ui.cursor = ui.cursor0 + int64(len(s))
	case 'o':
		ui.cursor, ui.cursor0 = ui.cursor0, ui.cursor
	default:
		cmd.Number()
		offset := ui.commandMove(dui, cmd, br, fr, -1)
		if line {
			if offset < ui.cursor {
				r := ui.revReader(offset)
				r.Line(false)
				ui.cursor0 = ui.cursor
				ui.cursor = r.Offset()
			} else {
				r := ui.reader(offset, ui.text.Size())
				r.Line(false)
				ui.cursor = r.Offset()
			}
		} else {
			ui.cursor = offset
		}
	}

	ui.visual = ""
}

func (ui *Edit) commandKey(dui *DUI, k rune, result *Result) {
	ui.command += string(k)

	defer func() {
		err := recover()
		if err == nil {
			return
		}
		switch err {
		case cmdUnfinished:
			return
		case cmdBadNumber, cmdBadMove, cmdNoNumber:
			ui.command = ""
			return
		default:
			panic(err)
		}
	}()
	cmd := &cmd{r: strings.NewReader(ui.command), number: 1}

	fr := ui.reader(ui.cursor, ui.text.Size())
	br := ui.revReader(ui.cursor)

	cursor := func(offset int64) {
		ui.cursor = offset
		ui.cursor0 = ui.cursor
		ui.scrollCursor(dui)
	}

	order := func(c0, c1 int64) (int64, int64) {
		if c0 < c1 {
			return c0, c1
		}
		return c1, c0
	}

	const Ctrl = 0x1f

	cmd.Number()
	r, _ := cmd.get(true)
	switch r {
	case 'i':
		ui.mode = ModeInsert
	case 'I':
		br.Line(false)
		cursor(br.Offset())
		ui.mode = ModeInsert
	case 'a':
		fr.TryGet()
		cursor(fr.Offset())
		ui.mode = ModeInsert
	case 'A':
		fr.Line(false)
		cursor(fr.Offset())
		ui.mode = ModeInsert
	case 'o':
		fr.Line(true)
		ui.text.Replace(fr.Offset(), fr.Offset(), []byte("\n"))
		cursor(fr.Offset())
		ui.mode = ModeInsert
	case 'O':
		br.Line(false)
		ui.text.Replace(br.Offset(), br.Offset(), []byte("\n"))
		cursor(br.Offset())
		ui.mode = ModeInsert
	case 's':
		cmd.Times(func() {
			fr.TryGet()
		})
		ui.text.Replace(br.Offset(), fr.Offset(), nil)
		cursor(fr.Offset())
		ui.mode = ModeInsert
	case 'S':
		cmd.Times(func() {
			fr.Line(true)
		})
		ui.text.Replace(br.Offset(), fr.Offset(), []byte("\n"))
		cursor(fr.Offset())
		ui.mode = ModeInsert
	// case 'R': // replace, not sure if this is a useful enough
	case 'D':
		// delete to end of line
		fr.Line(false)
		ui.text.Replace(ui.cursor, fr.Offset(), nil)
	case 'd':
		// delete movement
		cmd.Get()
		cmd.Number()
		c0, c1 := order(ui.cursor, ui.commandMove(dui, cmd, br, fr, 'd'))
		ui.text.Replace(c0, c1, nil)
		cursor(br.Offset())
	case 'c':
		// replace movement
		cmd.Get()
		cmd.Number()
		c0, c1 := order(ui.cursor, ui.commandMove(dui, cmd, br, fr, 'c'))
		ui.text.Replace(c0, c1, nil)
		ui.mode = ModeInsert
	case 'x':
		// delete
		cmd.Get()
		cmd.Times(func() {
			fr.TryGet()
		})
		ui.text.Replace(ui.cursor, fr.Offset(), nil)
	case 'X':
		// backspace
		cmd.Get()
		cmd.Times(func() {
			br.TryGet()
		})
		ui.text.Replace(br.Offset(), ui.cursor, nil)
		cursor(br.Offset())
	case 'y':
		// yank
		cmd.Get()
		cmd.Number()
		c0, c1 := order(ui.cursor, ui.commandMove(dui, cmd, br, fr, 'y'))
		ui.writeSnarf(dui, []byte(ui.readText(c0, c1)))
	case 'Y':
		// whole lines
		cmd.Get()
		br.Line(false)
		cmd.Times(func() {
			fr.Line(true)
		})
		ui.writeSnarf(dui, []byte(ui.readText(br.Offset(), fr.Offset())))
	case 'p':
		// paste
		cmd.Get()
		buf, ok := ui.readSnarf(dui)
		if ok {
			ui.text.Replace(ui.cursor, ui.cursor, buf)
		}
	case 'P':
		// paste before
		cmd.Get()
		buf, ok := ui.readSnarf(dui)
		if ok {
			br.TryGet()
			ui.text.Replace(br.Offset(), br.Offset(), buf)
			cursor(br.Offset())
		}
	case '<':
		// unindent
		cmd.Get()
		cmd.Number()
		br.Line(false)
		c0, c1 := order(ui.cursor, ui.commandMove(dui, cmd, br, fr, '<'))
		ui.unindent(c0, c1)
		cursor(br.Offset())
	case '>':
		// indent
		cmd.Get()
		cmd.Number()
		br.Line(false)
		c0, c1 := order(ui.cursor, ui.commandMove(dui, cmd, br, fr, '>'))
		ui.indent(c0, c1)
		cursor(br.Offset())
	case 'J':
		// join with next line
		fr.Line(false)
		o := fr.Offset()
		fr.TryGet()
		if o != fr.Offset() {
			ui.text.Replace(o, fr.Offset(), []byte(" "))
		}
	case '~':
		// swap case of single char
		start := fr.Offset()
		r, err := fr.TryGet()
		if err == nil {
			or := r
			if unicode.IsUpper(r) {
				r = unicode.ToLower(r)
			} else if unicode.IsLower(r) {
				r = unicode.ToUpper(r)
			}
			if or != r {
				ui.text.Replace(start, fr.Offset(), []byte(string(r)))
				cursor(start + int64(len(string(r))))
			}
		}
	case '.':
		// repeat last modification at current cursor
		// xxx todo: need to keep track of changes
	case 'v':
		ui.mode = ModeVisual
	case 'V':
		br.Line(false)
		fr.Line(false)
		ui.cursor0 = br.Offset()
		ui.cursor = fr.Offset()
		ui.mode = ModeVisualLine
	case 'u':
		// xxx todo: undo
	case Ctrl & 'e':
		// viewport down
		r := ui.reader(ui.offset, ui.text.Size())
		cmd.Times(func() {
			r.Line(true)
		})
		ui.offset = r.Offset()
	case Ctrl & 'r':
		// xxx todo: redo
	case Ctrl & 'g':
		// xxx todo: show location status in bar

	default:
		cursor(ui.commandMove(dui, cmd, br, fr, -1))
	}
	ui.command = ""
}

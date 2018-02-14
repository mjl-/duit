package duit

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

var (
	errCmdBadNumber  = errors.New("bad number")
	errCmdUnfinished = errors.New("command not completed")
	errCmdNoNumber   = errors.New("no number allowed")
	errCmdBadMove    = errors.New("bad move")
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
		panic(errCmdBadNumber)
	}
	if v == 0 {
		panic(errCmdBadNumber)
	}
	c.number = int(v)
}

func (c *cmd) Get() rune {
	r, eof := c.get(false)
	if eof {
		panic(errCmdUnfinished)
	}
	return r
}

func (c *cmd) NoNumber() {
	if c.number != 1 {
		panic(errCmdNoNumber)
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
				fr.Punctuation()
			}
			fr.Whitespace(true)
		})
		return fr.Offset()
	case 'W':
		// word
		cmd.Times(func() {
			fr.Nonwhitespace()
			fr.Whitespace(true)
		})
		return fr.Offset()
	case 'b':
		// to begin of (previous) word
		cmd.Times(func() {
			o := br.Offset()
			br.Whitespacepunct(true)
			if o == br.Offset() {
				br.Nonwhitespacepunct()
			}
		})
		return br.Offset()
	case 'B':
		// like 'b', skip interpunction too
		cmd.Times(func() {
			br.Whitespace(true)
			br.Nonwhitespace()
		})
		return br.Offset()
	case 'e':
		// to end of (next) word
		cmd.Times(func() {
			fr.Whitespace(true)
			fr.Nonwhitespacepunct()
		})
		return fr.Offset()
	case 'E':
		// like 'e', skip interpunction too
		cmd.Times(func() {
			fr.Whitespace(true)
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
		const Starts = "{[(<"
		const Ends = "}])>"

		c, eof := fr.Peek()
		if index := strings.IndexRune(Starts, c); !eof && index >= 0 {
			fr.Get()
			if ui.expandNested(fr, rune(Starts[index]), rune(Ends[index])) > 0 {
				return fr.Offset()
			}
		}
		if index := strings.IndexRune(Ends, c); !eof && index >= 0 {
			br.Get()
			if ui.expandNested(br, rune(Ends[index]), rune(Starts[index])) > 0 {
				br.TryGet()
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
	panic(errCmdBadMove)
}

func (ui *Edit) visualKey(dui *DUI, line bool, result *Result) {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		switch err {
		case errCmdUnfinished:
			return
		case errCmdBadNumber, errCmdBadMove, errCmdNoNumber:
			ui.visual = ""
			return
		default:
			panic(err)
		}
	}()
	cmd := &cmd{r: strings.NewReader(ui.visual), number: 1}

	fr := ui.reader(ui.cursor.Cur, ui.text.Size())
	br := ui.revReader(ui.cursor.Cur)

	c0, _ := ui.cursor.Ordered()

	switch ui.visual[len(ui.visual)-1] {
	case 'i':
		ui.mode = modeInsert
	case 'd':
		ui.text.Replace(ui, &ui.dirty, ui.cursor, nil, false)
		ui.cursor = Cursor{c0, c0}
	case 's':
		ui.text.Replace(ui, &ui.dirty, ui.cursor, nil, false)
		ui.mode = modeInsert
		ui.cursor = Cursor{c0, c0}
	case 'y':
		buf, err := ui.readText(ui.cursor)
		if ui.error(err, "readText") {
			break
		}
		dui.WriteSnarf(buf)
	case 'p':
		buf, ok := dui.ReadSnarf()
		if ok {
			ui.text.Replace(ui, &ui.dirty, ui.cursor, buf, false)
			ui.cursor = Cursor{c0 + int64(len(buf)), c0}
		}
	case '<':
		n := ui.unindent(ui.cursor)
		ui.cursor = Cursor{c0 + n, c0}
	case '>':
		n := ui.indent(ui.cursor)
		ui.cursor = Cursor{c0 + n, c0}
	case 'J':
		buf, err := ui.readText(ui.cursor)
		if ui.error(err, "readText") {
			break
		}
		s := string(buf)
		newline := s != "" && s[len(s)-1] == '\n'
		if newline {
			s = s[:len(s)-1]
		}
		s = strings.Replace(s, "\n", " ", -1)
		if newline {
			s += "\n"
		}
		ui.text.Replace(ui, &ui.dirty, ui.cursor, []byte(s), false)
		ui.cursor = Cursor{c0 + int64(len(s)), c0}
	case '~':
		s := ""
		buf, err := ui.readText(ui.cursor)
		if ui.error(err, "readText") {
			break
		}
		for _, r := range string(buf) {
			if unicode.IsUpper(r) {
				r = unicode.ToLower(r)
			} else if unicode.IsLower(r) {
				r = unicode.ToUpper(r)
			}
			s += string(r)
		}
		ui.text.Replace(ui, &ui.dirty, ui.cursor, []byte(s), false)
		ui.cursor = Cursor{c0 + int64(len(s)), c0}
	case 'o':
		ui.cursor = Cursor{ui.cursor.Start, ui.cursor.Cur}
	default:
		cmd.Number()
		offset := ui.commandMove(dui, cmd, br, fr, -1)
		if line {
			if offset < ui.cursor.Cur {
				r := ui.revReader(offset)
				r.Line(false)
				ui.cursor = Cursor{r.Offset(), ui.cursor.Cur}
			} else {
				r := ui.reader(offset, ui.text.Size())
				r.Line(false)
				ui.cursor.Cur = r.Offset()
			}
		} else {
			ui.cursor.Cur = offset
		}
	}

	ui.visual = ""
}

func (ui *Edit) commandKey(dui *DUI, result *Result) (modified bool) {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		switch err {
		case errCmdUnfinished:
			return
		case errCmdBadNumber, errCmdBadMove, errCmdNoNumber:
			ui.command = ""
			return
		default:
			panic(err)
		}
	}()
	cmd := &cmd{r: strings.NewReader(ui.command), number: 1}

	fr := ui.reader(ui.cursor.Cur, ui.text.Size())
	br := ui.revReader(ui.cursor.Cur)

	setCursor := func(offset int64) {
		ui.cursor = Cursor{offset, offset}
		ui.ScrollCursor(dui)
	}

	const Ctrl = 0x1f

	cmd.Number()
	r, _ := cmd.get(true)
	switch r {
	case 'i':
		ui.mode = modeInsert
	case 'I':
		br.Line(false)
		setCursor(br.Offset())
		ui.mode = modeInsert
	case 'a':
		fr.TryGet()
		setCursor(fr.Offset())
		ui.mode = modeInsert
	case 'A':
		fr.Line(false)
		setCursor(fr.Offset())
		ui.mode = modeInsert
	case 'o':
		fr.Line(true)
		ui.text.Replace(ui, &modified, Cursor{fr.Offset(), fr.Offset()}, []byte("\n"), false)
		setCursor(fr.Offset() + 1)
		ui.mode = modeInsert
	case 'O':
		br.Line(false)
		ui.text.Replace(ui, &modified, Cursor{br.Offset(), br.Offset()}, []byte("\n"), false)
		setCursor(br.Offset())
		ui.mode = modeInsert
	case 's':
		cmd.Times(func() {
			fr.TryGet()
		})
		ui.text.Replace(ui, &modified, Cursor{br.Offset(), fr.Offset()}, nil, false)
		setCursor(fr.Offset())
		ui.mode = modeInsert
	case 'S':
		cmd.Times(func() {
			fr.Line(true)
		})
		ui.text.Replace(ui, &modified, Cursor{br.Offset(), fr.Offset()}, []byte("\n"), false)
		setCursor(fr.Offset())
		ui.mode = modeInsert
	// case 'R': // replace, not sure if this is a useful enough
	case 'D':
		// delete lines
		cmd.Times(func() {
			fr.Line(true)
		})
		ui.text.Replace(ui, &modified, Cursor{ui.cursor.Cur, fr.Offset()}, []byte("\n"), false)
	case 'd':
		// delete movement
		cmd.Get()
		cmd.Number()
		c := Cursor{ui.cursor.Cur, ui.commandMove(dui, cmd, br, fr, 'd')}
		ui.text.Replace(ui, &modified, c, nil, false)
		setCursor(br.Offset())
	case 'C':
		// replace lines
		cmd.Times(func() {
			fr.Line(true)
		})
		ui.text.Replace(ui, &modified, Cursor{ui.cursor.Cur, fr.Offset()}, []byte("\n"), false)
		ui.mode = modeInsert
	case 'c':
		// replace movement
		cmd.Get()
		cmd.Number()
		c := Cursor{ui.cursor.Cur, ui.commandMove(dui, cmd, br, fr, 'c')}
		ui.text.Replace(ui, &modified, c, nil, false)
		ui.mode = modeInsert
	case 'x':
		// delete
		cmd.Get()
		cmd.Times(func() {
			fr.TryGet()
		})
		ui.text.Replace(ui, &modified, Cursor{ui.cursor.Cur, fr.Offset()}, nil, false)
	case 'X':
		// backspace
		cmd.Get()
		cmd.Times(func() {
			br.TryGet()
		})
		ui.text.Replace(ui, &modified, Cursor{br.Offset(), ui.cursor.Cur}, nil, false)
		setCursor(br.Offset())
	case 'y':
		// yank
		cmd.Get()
		cmd.Number()
		c := Cursor{ui.cursor.Cur, ui.commandMove(dui, cmd, br, fr, 'y')}
		buf, err := ui.readText(c)
		if ui.error(err, "readText") {
			break
		}
		dui.WriteSnarf(buf)
	case 'Y':
		// whole lines
		cmd.Get()
		br.Line(false)
		cmd.Times(func() {
			fr.Line(true)
		})
		buf, err := ui.readText(Cursor{br.Offset(), fr.Offset()})
		if ui.error(err, "readText") {
			break
		}
		dui.WriteSnarf(buf)
	case 'p':
		// paste
		cmd.Get()
		buf, ok := dui.ReadSnarf()
		if ok {
			ui.text.Replace(ui, &modified, ui.cursor, buf, false)
		}
	case 'P':
		// paste before
		cmd.Get()
		buf, ok := dui.ReadSnarf()
		if ok {
			br.TryGet()
			ui.text.Replace(ui, &modified, Cursor{br.Offset(), br.Offset()}, buf, false)
			setCursor(br.Offset())
		}
	case '<':
		// unindent
		cmd.Get()
		cmd.Number()
		br.Line(false)
		c := Cursor{ui.cursor.Cur, ui.commandMove(dui, cmd, br, fr, '<')}
		ui.unindent(c)
		setCursor(br.Offset())
		modified = true
	case '>':
		// indent
		cmd.Get()
		cmd.Number()
		br.Line(false)
		c := Cursor{ui.cursor.Cur, ui.commandMove(dui, cmd, br, fr, '>')}
		ui.indent(c)
		setCursor(br.Offset())
		modified = true
	case 'J':
		// join with next line
		fr.Line(false)
		o := fr.Offset()
		fr.TryGet()
		if o != fr.Offset() {
			ui.text.Replace(ui, &modified, Cursor{o, fr.Offset()}, []byte(" "), false)
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
				ui.text.Replace(ui, &modified, Cursor{start, fr.Offset()}, []byte(string(r)), false)
				setCursor(start + int64(len(string(r))))
			}
		}
	case 'v':
		ui.mode = modeVisual
	case 'V':
		br.Line(false)
		fr.Line(false)
		ui.cursor = Cursor{fr.Offset(), br.Offset()}
		ui.mode = modeVisual
	case 'u':
		ui.text.undo(ui)
		ui.ScrollCursor(dui)
	case Ctrl & 'e':
		// viewport one line down, move cursor if necessary to keep in view
		r := ui.reader(ui.offset, ui.text.Size())
		cmd.Times(func() {
			r.Line(true)
		})
		ui.offset = r.Offset()
		if _, c := ui.cursor.Ordered(); c < ui.offset {
			ui.cursor = Cursor{ui.offset, ui.offset}
		}
	case Ctrl & 'r':
		ui.text.redo(ui)
		ui.ScrollCursor(dui)
	case Ctrl & 'g':
		// xxx todo: show location
	case '*':
		br.Nonwhitespacepunct()
		fr.Nonwhitespacepunct()
		buf, err := ui.text.get(Cursor{br.Offset(), fr.Offset()})
		if ui.error(err, "read") {
			return
		}
		ui.LastSearch = " " + string(buf)
		ui.Search(dui, false)
	case 'n':
		ui.Search(dui, true)
	case 'N':
		ui.Search(dui, false)
	case '.':
		cmd, lastText := ui.lastCommand, ui.lastCommandText
		ui.command = cmd
		defer func() {
			ui.lastCommand, ui.lastCommandText, ui.command = cmd, lastText, ""
		}()
		ui.commandKey(dui, result)
		if len(lastText) > 0 {
			ui.text.Replace(ui, &ui.dirty, Cursor{ui.cursor.Cur, ui.cursor.Cur}, lastText, false)
		}
		ui.mode = modeCommand
		return false
	default:
		setCursor(ui.commandMove(dui, cmd, br, fr, -1))
	}
	if modified {
		ui.dirty = true
		ui.lastCommand = ui.command
		ui.lastCommandText = nil
		ui.needLastCommandText = ui.mode == modeInsert
	}
	ui.command = ""
	return modified
}

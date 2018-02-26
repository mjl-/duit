package duit

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"9fans.net/go/draw"
)

// SeekReaderAt is used as a source for edits. The seeker is used to determine file size, the readerAt for reading.
type SeekReaderAt interface {
	io.Seeker
	io.ReaderAt
}

var (
	EditPadding = Space{0, 3, 0, 3} // LowDPI padding, drawn with a distinct color when in vi modes.
)

type editMode int

const (
	modeInsert     editMode = iota // Regular editing.
	modeCommand                    // vi commands, after escape without selection.
	modeVisual                     // vi visual mode, after 'v' in command mode, or escape with selection.
	modeVisualLine                 // vi visual line mode, after 'V' in command mode.
)

// EditColors hold all the colors used for rendering an Edit.
type EditColors struct {
	Fg, Bg,
	SelFg, SelBg,
	ScrollVis, ScrollBg,
	HoverScrollVis, HoverScrollBg,
	CommandBorder, VisualBorder *draw.Image
}

// Cursor represents the current editing location, and optionally text selection.
type Cursor struct {
	Cur   int64 // Current location/end of selection.
	Start int64 // Start of selection, not necessarily larger than Cur!
}

// Edit is a text editor inspired by acme, with vi key bindings. An edit has its own scrollbar, unlimited undo. It can read utf-8 encoded files of arbritary length, only reading data when necessary, to display or search.
//
// The usual arrow and pageup/pagedown keys can be used for navigation.
// Key shortcuts when combined with control:
//	a, to start of line
//	e, to end of line
//	h, remove character before cursor
//	w, remove word before cursor
//	u, delete to start of line
//	k, delete to end of line
//
// Key shortcuts when combined with the command key:
// 	a, select all text
// 	n, no selection
// 	c, copy selection
// 	x, cut selection
// 	v, paste selection
// 	z, undo last change
// 	Z, redo last undone change
// 	[, unindent selection or line
// 	], indent selection or line
// 	m, warp mouse to the cursor
// 	y, select last modification
// 	/, repeat last search forward
// 	?, repeat last search backward
//
// Edit has a vi command and visual mode, entered through the familiar escape key. Not all commands have been implemented yet, Edit does not aim to be feature-complete or a clone of any specific existing vi-clone.
type Edit struct {
	NoScrollbar  bool                                       // If set, no scrollbar is shown. Content will still scroll.
	LastSearch   string                                     // If starting with slash, the remainder is interpreted as regexp. used by cmd+[/?] and vi [*nN] commands. Literal text search should start with a space.
	Error        chan error                                 // If set, errors from Edit (including read errors from underlying files) are sent here. If nil, errors go to dui.Error.
	Colors       *EditColors                                `json:"-"` // Colors to use for drawing the Edit UI, allows for creating an acme look.
	Font         *draw.Font                                 `json:"-"` // Used for drawing all text.
	Keys         func(k rune, m draw.Mouse) (e Event)       `json:"-"` // Called before handling keys. If you set e.Consumed, the key is not handled further.
	Click        func(m draw.Mouse, offset int64) (e Event) `json:"-"` // Called for clicks with button 1,2,3. Offset is the file offset that was clicked on.
	DirtyChanged func(dirty bool)                           `json:"-"` // Called when the dirty-state of the underlying file changes.

	dui *DUI // Set at beginning of UI interface functions, for not having to pass dui around all the time.

	text   *text  // Wat we are rendering.  Offset & cursors index into this text.
	offset int64  // Byte offset of first line we draw.
	cursor Cursor // Current cursor.

	lastSearchRegexpString string // String used to create lastSearchRegexp.
	lastSearchRegexp       *regexp.Regexp

	mode    editMode
	command string // vi command so far.
	visual  string // vi visual command so far.

	// For repeat.
	lastCommand         string
	needLastCommandText bool
	lastCommandText     []byte // Text inserted as part of last command.  Used by vi repeat, filled by ui.text.

	dirty bool

	r,
	barR,
	barActiveR,
	textR image.Rectangle

	textM,
	prevTextB1 draw.Mouse

	lastCursorPoint image.Point
}

// Ordered returns the ordered start, end position of the cursor.
func (c Cursor) Ordered() (int64, int64) {
	if c.Cur > c.Start {
		return c.Start, c.Cur
	}
	return c.Cur, c.Start
}

func (c Cursor) size() int64 {
	c0, c1 := c.Ordered()
	return c1 - c0
}

func (ui *Edit) error(err error, msg string) bool {
	if err == nil {
		return false
	}
	err = fmt.Errorf("%s: %s", msg, err)
	go func() {
		if ui.Error != nil {
			ui.Error <- err
		} else {
			ui.dui.Error <- err
		}
	}()
	return true
}

func (ui *Edit) colors() EditColors {
	if ui.Colors != nil {
		return *ui.Colors
	}
	dui := ui.dui
	return EditColors{
		Fg:             dui.Regular.Normal.Text,
		Bg:             dui.Regular.Normal.Background,
		SelFg:          dui.Inverse.Text,
		SelBg:          dui.Inverse.Background,
		ScrollVis:      dui.ScrollVisibleNormal,
		ScrollBg:       dui.ScrollBGNormal,
		HoverScrollVis: dui.ScrollVisibleHover,
		HoverScrollBg:  dui.ScrollBGHover,
		CommandBorder:  dui.CommandMode,
		VisualBorder:   dui.VisualMode,
	}
}

// NewEdit returns an Edit initialized with f.
func NewEdit(f SeekReaderAt) (ui *Edit, err error) {
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return
	}
	parts := []textPart{}
	if size > 0 {
		parts = append(parts, &file{f, 0, size})
	}
	ui = &Edit{
		text: &text{file: f, l: parts},
	}
	return
}

type reverseReader struct {
	src    io.ReaderAt
	offset int64 // and going to 0
}

var _ io.Reader = &reverseReader{}

func readAtFull(src io.ReaderAt, buf []byte, offset int64) (read int, err error) {
	want := len(buf)
	for want > 0 {
		var n int
		n, err = src.ReadAt(buf, offset)
		if n > 0 {
			read += n
			offset += int64(n)
			want -= n
			buf = buf[n:]
		}
		if err == io.EOF {
			return
		}
	}
	return
}

func (r *reverseReader) Read(buf []byte) (int, error) {
	// log.Printf("reverseReader.Read, len buf %d, offset %d\n", len(buf), r.offset)
	want := int64(len(buf))
	want = minimum64(want, r.offset)
	if want == 0 {
		return 0, io.EOF
	}
	have, err := readAtFull(r.src, buf[:want], r.offset-want)
	if err != nil && err != io.EOF {
		return have, err
	}
	buf = buf[:have]

	// reverse the bytes, but keep utf8 valid
	// todo: should probably provide a reader like bufio that has ReadRune and UnReadRune and others and read backwards on demand.
	orignbuf := make([]byte, have)
	nbuf := orignbuf
	obuf := buf
	for len(buf) > 0 {
		_, size := utf8.DecodeLastRune(buf)
		if size == 0 {
			break
		}
		copy(nbuf[:], buf[len(buf)-size:])
		buf = buf[:len(buf)-size]
		nbuf = nbuf[size:]
	}
	have -= len(buf)
	copy(obuf[:], orignbuf[:have])
	if have > 0 {
		r.offset -= int64(have)
	}
	// log.Printf("reverseReader.Read, returning n %d, err %s, buf %s\n", have, err, string(buf[:have]))
	return have, err
}

type reader struct {
	ui      *Edit
	n       int64 // number of bytes read, excluding peek
	r       *bufio.Reader
	offset  int64
	forward bool
}

func (r *reader) Offset() int64 {
	if r.forward {
		return r.offset + r.n
	}
	return r.offset - r.n
}

func (r *reader) Peek() (rune, bool) {
	c, size, err := r.r.ReadRune()
	if size <= 0 && err == io.EOF {
		return 0, true
	}
	if r.ui.error(err, "readrune") {
		return -1, true
	}
	r.r.UnreadRune()
	return c, false
}

func (r *reader) Get() rune {
	c, size, err := r.r.ReadRune()
	if r.ui.error(err, "readrune") {
		return -1
	}
	r.n += int64(size)
	return c
}

func (r *reader) TryGet() (rune, error) {
	c, size, err := r.r.ReadRune()
	if err != nil {
		return 0, err
	}
	r.n += int64(size)
	return c, nil
}

func (r *reader) Line(includeNewline bool) (runes int, s string, eof bool) {
	var c rune
	for {
		c, eof = r.Peek()
		if eof {
			eof = s == ""
			break
		}
		if c == '\n' {
			if includeNewline {
				runes++
				r.Get()
			}
			break
		}
		r.Get()
		runes++
		s += string(c)
	}
	return
}

func (r *reader) gather(keep func(c rune) bool) (s string, eof bool) {
	var c rune
	for {
		c, eof = r.Peek()
		if eof {
			eof = s == ""
			break
		}
		if !keep(c) {
			break
		}
		r.Get()
		s += string(c)
	}
	return
}

// xxx todo: better (non)whitespace functions
func (r *reader) Whitespace(newline bool) (s string, eof bool) {
	return r.gather(func(c rune) (keep bool) {
		return (c != '\n' || newline) && unicode.IsSpace(c)
	})
}

func (r *reader) Nonwhitespace() (s string, eof bool) {
	return r.gather(func(c rune) (keep bool) {
		return !unicode.IsSpace(c)
	})
}

func (r *reader) Whitespacepunct(newline bool) (s string, eof bool) {
	return r.gather(func(c rune) (keep bool) {
		return (c != '\n' || newline) && (unicode.IsSpace(c) || unicode.IsPunct(c))
	})
}

func (r *reader) Nonwhitespacepunct() (s string, eof bool) {
	return r.gather(func(c rune) (keep bool) {
		return !unicode.IsSpace(c) && !unicode.IsPunct(c)
	})
}

func (r *reader) Punctuation() (s string, eof bool) {
	return r.gather(func(c rune) (keep bool) {
		return unicode.IsPunct(c)
	})
}

func (ui *Edit) ensureInit() {
	if ui.text == nil {
		ui.text = &text{}
	}
}

// EditReader provides a reader to the current contents of an Edit.
// It is used by navigation commands and keyboard shortcuts.
// Both Edit.EditReader and Edit.ReverseEditReader return an EditReader. ReverseEditReader reads utf-8 characters in reverse, towards the start of the file.
type EditReader interface {
	Peek() (r rune, eof bool)                                 // Return next character without consuming.
	TryGet() (r rune, err error)                              // Returns and consume next character.
	Get() (r rune)                                            // Return and consume next character. On error, returns -1 and sends on Edit.Error.
	Offset() (offset int64)                                   // Current offset.
	Whitespace(newline bool) (s string, eof bool)             // Consume and return whitespace, possibly including newlines.
	Nonwhitespace() (s string, eof bool)                      // Consume all except whitespace.
	Whitespacepunct(newline bool) (s string, eof bool)        // Consume whitespace and punctation.
	Nonwhitespacepunct() (s string, eof bool)                 // Consume non-whitespace and punctutation.
	Punctuation() (s string, eof bool)                        // Consume punctuation.
	Line(includeNewline bool) (runes int, s string, eof bool) // Reads to end of newline, possibly including the newline itself.
}

type ReaderReaderAt interface {
	io.Reader
	io.ReaderAt
}

// Text returns the entire contents.
func (ui *Edit) Text() ([]byte, error) {
	return ioutil.ReadAll(ui.Reader())
}

// Reader from which contents of edit can be read.
func (ui *Edit) Reader() ReaderReaderAt {
	ui.ensureInit()
	// xxx should make copy of ui.text
	return io.NewSectionReader(ui.text, 0, ui.text.Size()-0)
}

// EditReader from which contents of edit can be read, starting at offset.
func (ui *Edit) EditReader(offset int64) EditReader {
	ui.ensureInit()
	// xxx should make copy of ui.text
	return ui.reader(offset, ui.text.Size())
}

// ReverseEditReader from which contents of edit can be read in reverse (whole utf-8 characters), starting at offset, to 0.
func (ui *Edit) ReverseEditReader(offset int64) EditReader {
	ui.ensureInit()
	// xxx should make copy of ui.text
	return ui.revReader(offset)
}

func (ui *Edit) reader(offset, size int64) *reader {
	return &reader{ui, 0, bufio.NewReader(io.NewSectionReader(ui.text, offset, size-offset)), offset, true}
}

func (ui *Edit) revReader(offset int64) *reader {
	return &reader{ui, 0, bufio.NewReader(&reverseReader{ui.text, offset}), offset, false}
}

func (ui *Edit) font() *draw.Font {
	return ui.dui.Font(ui.Font)
}

func (ui *Edit) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	ui.dui = dui
	dui.debugLayout(self)

	ui.ensureInit()
	ui.r = rect(sizeAvail)
	ui.barR = ui.r
	if ui.NoScrollbar {
		ui.barR.Max.X = ui.barR.Min.X
	} else {
		ui.barR.Max.X = ui.barR.Min.X + dui.Scale(ScrollbarSize)
	}
	ui.barActiveR = ui.barR // Y's are filled in during draw
	ui.textR = ui.r
	ui.textR.Min.X = ui.barR.Max.X
	ui.textR = dui.ScaleSpace(EditPadding).Inset(ui.textR)
	self.R = ui.r
}

func (ui *Edit) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	ui.dui = dui
	dui.debugDraw(self)

	ui.ensureInit()
	if ui.r.Empty() {
		return
	}

	colors := ui.colors()
	pad := dui.ScaleSpace(EditPadding).Mul(-1)
	switch ui.mode {
	case modeInsert:
		img.Draw(pad.Inset(ui.textR.Add(orig)), colors.Bg, nil, image.ZP)
	case modeCommand:
		img.Draw(pad.Inset(ui.textR.Add(orig)), colors.CommandBorder, nil, image.ZP)
	case modeVisual, modeVisualLine:
		img.Draw(pad.Inset(ui.textR.Add(orig)), colors.VisualBorder, nil, image.ZP)
	}
	img.Draw(ui.textR.Add(orig), colors.Bg, nil, image.ZP)

	font := ui.font()
	s := ""
	sdx := 0
	lineWidth := ui.textR.Dx()
	line := 0

	size := ui.text.Size()
	rd := ui.reader(ui.offset, size)
	lines := ui.textR.Dy() / font.Height

	dropNewline := func(s string) string {
		if s != "" && s[len(s)-1] == '\n' {
			return s[:len(s)-1]
		}
		return s
	}

	c0, c1 := ui.cursor.Ordered()
	// log.Printf("drawing... c0 %d, c1 %d\n", c0, c1)
	drawLine := func(offsetEnd int64, eof bool) {
		origS := s

		n := len(s)
		offset := offsetEnd - int64(n)
		// log.Printf("drawLine, offset %d, offsetEnd %d, n %d\n", offset, offsetEnd, n)
		p := orig.Add(ui.textR.Min).Add(image.Pt(0, line*font.Height))
		cursorp := image.Pt(-1, -1)

		drawCursor := func(haveSel, cursorAtBegin bool) {
			// log.Printf("drawCursor, line %d c0 %d, c1 %d, cursor %d, cursor0 %d, offset %d, offsetEnd %d, s %s, n %d\n", line, c0, c1, ui.cursor, ui.cursor0, offset, offsetEnd, s, n)
			p0 := cursorp
			p1 := p0
			p1.Y += font.Height
			thick := 0
			if dui.Scale(1) > 1 {
				thick = 1
			}
			img.Line(p0, p1, 0, 0, thick, dui.Display.Black, image.ZP)
			ui.lastCursorPoint = p1.Sub(orig)
			if haveSel {
				pp := pt(dui.Scale(-2))
				if cursorAtBegin {
					pp.X *= -1
				}
				ui.lastCursorPoint = ui.lastCursorPoint.Add(pp)
			}
		}

		// we draw text before selection
		if offset < c0 {
			nn := minimum64(int64(n), c0-offset)
			// log.Printf("drawing %d before selection\n", nn)
			pp := img.String(p, colors.Fg, image.ZP, font, dropNewline(s[:nn]))
			p.X = pp.X
			s = s[nn:]
			offset += nn
		}

		if offset == ui.cursor.Cur && ui.cursor.Cur == c0 && c0 != c1 && offset < offsetEnd {
			//log.Printf("cursor A, offset %d, ui.cursor %d, c1 %d, offsetEnd %d, size %d\n", offset, ui.cursor.Cur, c1, offsetEnd, size)
			cursorp = p
		}

		// then selected text
		haveSelection := offset >= c0 && offset < c1 && c1-c0 > 0 && offset < offsetEnd
		if haveSelection {
			nn := minimum64(c1, offsetEnd) - offset
			// log.Printf("drawing %d as selection\n", nn)
			sels := s[:nn]
			toEnd := sels[len(sels)-1] == '\n'
			if toEnd {
				sels = sels[:len(sels)-1]
			}
			s = s[nn:]
			offset += nn
			if !(offset >= c1 && offsetEnd > offset) {
				toEnd = true
			}
			seldx := font.StringWidth(sels)
			selR := rect(image.Pt(seldx, font.Height)).Add(p)
			if toEnd {
				selR.Max.X = ui.textR.Max.X + orig.X
			}
			img.Draw(selR, colors.SelBg, nil, image.ZP)
			pp := img.String(p, colors.SelFg, image.ZP, font, sels)
			p.X = pp.X
		}
		if offset == ui.cursor.Cur && ui.cursor.Cur == c1 && (offset < offsetEnd || (offset == size && eof)) {
			// log.Printf("cursor B, offset %d, ui.cursor %d, c1 %d, offsetEnd %d, size %d\n", offset, ui.cursor.Cur, c1, offsetEnd, size)
			cursorp = p
		}
		if cursorp.X >= 0 {
			drawCursor(haveSelection, ui.cursor.Cur < ui.cursor.Start)
		}

		// then text after cursor
		if offset >= c1 && offsetEnd > offset {
			nn := int(offsetEnd - offset)
			// log.Printf("drawing %d after selection\n", nn)
			pp := img.String(p, colors.Fg, image.ZP, font, dropNewline(s))
			p.X = pp.X
			s = s[nn:]
			offset += int64(nn)
		}
		if s != "" || offset != offsetEnd {
			panic(fmt.Sprintf("bug in drawLine, s %v, offset %d, offsetEnd %d, c0 %d c1 %d, line %d, sdx %d, origS %s", s, offset, offsetEnd, c0, c1, line, sdx, origS))
		}

		s = ""
		sdx = 0
		line++
	}

	for line < lines {
		c, eof := rd.Peek()
		if eof {
			drawLine(rd.Offset(), eof)
			break
		}
		if c == '\n' {
			s += string(rd.Get())
			drawLine(rd.Offset(), false)
			continue
		}
		dx := font.StringWidth(string(c))
		if sdx+dx < lineWidth {
			sdx += dx
			s += string(rd.Get())
			continue
		}
		drawLine(rd.Offset(), false)
		rd.Get()
		s = string(c)
		sdx = dx
	}

	barHover := m.In(ui.barR)
	bg := colors.ScrollBg
	vis := colors.ScrollVis
	if barHover {
		bg = colors.HoverScrollBg
		vis = colors.HoverScrollVis
	}

	if size == 0 {
		ui.barActiveR = ui.barR
	} else {
		ui.barActiveR.Min.Y = int(int64(ui.barR.Dy()) * ui.offset / size)
		ui.barActiveR.Max.Y = int(int64(ui.barR.Dy()) * rd.Offset() / size)
	}
	if ui.barR.Dx() > 0 {
		barActiveR := ui.barActiveR.Add(orig)
		barActiveR.Max.X -= 1 // unscaled
		img.Draw(ui.barR.Add(orig), bg, nil, image.ZP)
		img.Draw(barActiveR, vis, nil, image.ZP)
	}
}

func (ui *Edit) scroll(lines int, self *Kid) {
	offset := ui.offset
	if lines > 0 {
		rd := ui.reader(ui.offset, ui.text.Size())
		eof := false
		for ; lines > 0 && !eof; lines-- {
			_, _, eof = rd.Line(true)
		}
		offset = rd.Offset()
	} else if lines < 0 {
		rd := ui.revReader(ui.offset)
		eof := false
		for ; lines < 0 && !eof; lines++ {
			rd.TryGet()
			_, _, eof = rd.Line(false)
		}
		offset = rd.Offset()
	}

	if offset != ui.offset {
		self.Draw = Dirty
	}
	ui.offset = offset
}

func (ui *Edit) expandNested(r *reader, up, down rune) int64 {
	nested := 1
	for {
		c, eof := r.Peek()
		if eof {
			return 0
		}
		if c == down {
			nested--
		} else if c == up {
			nested++
		}
		if nested == 0 {
			return r.n
		}
		r.Get()
	}
}

// todo: maybe not have this here?
func (ui *Edit) ExpandedText() ([]byte, error) {
	ui.ensureInit()
	br := ui.revReader(ui.cursor.Cur)
	br.Nonwhitespace()
	fr := ui.reader(ui.cursor.Cur, ui.text.Size())
	fr.Nonwhitespace()
	return ui.readText(Cursor{br.Offset(), fr.Offset()})
}

func (ui *Edit) expand(offset int64, fr, br *reader) Cursor {
	const (
		Starts = "[{(<\"'`"
		Ends   = "]})>\"'`"
	)

	c, eof := br.Peek()
	index := strings.IndexRune(Starts, c)
	if !eof && index >= 0 {
		br.Get()
		n := ui.expandNested(fr, rune(Starts[index]), rune(Ends[index]))
		return Cursor{offset + n, offset}
	}
	c, eof = fr.Peek()
	index = strings.IndexRune(Ends, c)
	if !eof && index >= 0 {
		fr.Get()
		n := ui.expandNested(br, rune(Ends[index]), rune(Starts[index]))
		return Cursor{offset, offset - n}
	}

	c, eof = br.Peek()
	if c == '\n' {
		// at start of line, select to end of line
		fr.Line(true)
		return Cursor{fr.Offset(), offset}
	} else if c, eof = fr.Peek(); c == '\n' || eof {
		// at end of line, select to start of line
		if !eof {
			fr.Get()
		}
		br.Line(false)
		return Cursor{fr.Offset(), br.Offset()}
	}

	bc, _ := br.Peek()
	fc, _ := fr.Peek()
	if unicode.IsSpace(bc) && unicode.IsSpace(fc) {
		br.Whitespace(true)
		fr.Whitespace(true)
	} else {
		br.Nonwhitespacepunct()
		fr.Nonwhitespacepunct()
	}
	return Cursor{fr.Offset(), br.Offset()}
}

func (ui *Edit) checkDirty(odirty bool) {
	if odirty != ui.dirty && ui.DirtyChanged != nil {
		ui.DirtyChanged(ui.dirty)
	}
}

func (ui *Edit) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	ui.dui = dui
	ui.ensureInit()
	font := ui.font()
	scrollLines := func(y int) int {
		lines := ui.textR.Dy() / font.Height
		n := lines * y / ui.textR.Dy()
		if n == 0 {
			return 1
		}
		return n
	}
	if origM.In(ui.barR) {
		o := ui.offset
		switch m.Buttons {
		case Button1:
			ui.scroll(-scrollLines(origM.Y), self)
		case Button2:
			y := maximum(0, minimum(m.Y, ui.textR.Dy()))
			rd := ui.revReader(ui.text.Size() * int64(y) / int64(ui.textR.Dy()))
			for {
				c, eof := rd.Peek()
				if eof || c == '\n' {
					break
				}
				rd.Get()
			}
			if rd.Offset() != ui.offset {
				self.Draw = Dirty
			}
			ui.offset = rd.Offset()
		case Button3:
			ui.scroll(scrollLines(origM.Y), self)
		case Button4:
			ui.scroll(-scrollLines(origM.Y/4), self)
		case Button5:
			ui.scroll(scrollLines(origM.Y/4), self)
		}
		r.Consumed = o != ui.offset
		return
	}
	if !origM.In(dui.ScaleSpace(EditPadding).Mul(-1).Inset(ui.textR)) {
		return
	}
	defer ui.checkDirty(ui.dirty)
	origM.Point = origM.Point.Sub(ui.textR.Min)
	origM.Point.X = maximum(0, origM.Point.X)
	m.Point = m.Point.Sub(ui.textR.Min)
	m.Point.X = maximum(0, m.Point.X)
	om := ui.textM
	ui.textM = m

	switch m.Buttons {
	case Button4, Button5:
		o := ui.offset
		switch m.Buttons {
		case Button4:
			ui.scroll(-scrollLines(m.Y/4), self)
		case Button5:
			ui.scroll(scrollLines(m.Y/4), self)
		}
		r.Consumed = ui.offset != o
		return
	}

	mouseOffset := func() int64 {
		line := m.Y / ui.font().Height
		xmax := ui.textR.Dx()
		if line < 0 {
			rd := ui.revReader(ui.offset)
			var s []rune // in reverse
			x := 0
			finishLine := func() (offset int64, done bool) {
				if line < 0 {
					line++
					s = nil
					x = 0
					return rd.Offset(), false
				}
				x := 0
				o := 0
				for i := len(s) - 1; i >= 0; i-- {
					dx := font.StringWidth(string(s[i]))
					if x+2*dx/3 > xmax {
						break
					}
					o += len(string(s[i]))
				}
				return rd.Offset() + int64(o), true
			}
			for {
				c, eof := rd.Peek()
				if eof {
					r, _ := finishLine()
					return r
				}
				if c == '\n' {
					if r, done := finishLine(); done {
						return r
					}
					rd.Get()
					continue
				}
				dx := font.StringWidth(string(c))
				if x+dx > xmax {
					if r, done := finishLine(); done {
						return r
					}
				}
				rd.Get()
				s = append(s, c)
				x += dx
			}
		}
		rd := ui.reader(ui.offset, ui.text.Size())
		x := 0
		mX := m.X
		for {
			c, eof := rd.Peek()
			if eof {
				break
			}
			if c == '\n' {
				if line == 0 {
					break
				}
				rd.Get()
				line--
				x = 0
				continue
			}
			dx := font.StringWidth(string(c))
			if line == 0 && (x+2*dx/3 > mX || x+dx > xmax) {
				break
			}
			x += dx
			if x > xmax {
				line--
				x = 0
			} else {
				rd.Get()
			}
		}
		return rd.Offset()
	}
	if m.Buttons^om.Buttons != 0 && ui.mode != modeInsert {
		ui.mode = modeInsert
		ui.command = ""
		ui.visual = ""
	}
	if m.Buttons == Button1 {
		ui.cursor.Cur = mouseOffset()
		ui.ScrollCursor(dui)
		if om.Buttons == 0 {
			if m.Msec-ui.prevTextB1.Msec < 350 {
				ui.cursor = ui.expand(ui.cursor.Cur, ui.reader(ui.cursor.Cur, ui.text.Size()), ui.revReader(ui.cursor.Cur))
			} else {
				ui.cursor.Start = ui.cursor.Cur
			}
			ui.prevTextB1 = m
		}
		self.Draw = Dirty
		r.Consumed = true
		return
	}
	if m.Buttons == 0 && om.Buttons&(Button1|Button2|Button3) != 0 && ui.Click != nil {
		e := ui.Click(om, mouseOffset())
		propagateEvent(self, &r, e)
	}
	if m.Buttons^om.Buttons != 0 {
		ui.text.closeHist(ui)
		// log.Printf("in text, mouse buttons changed %v ->  %v\n", om, m)
	} else if m.Buttons != 0 && m.Buttons == om.Buttons {
		// log.Printf("in text, mouse drag %v\n", m)
	} else if om.Buttons != 0 && m.Buttons == 0 {
		// log.Printf("in text, button release %v -> %v\n", om, m)
	}
	r.Consumed = true
	return
}

func (ui *Edit) readText(c Cursor) ([]byte, error) {
	c0, c1 := c.Ordered()
	r := io.NewSectionReader(ui.text, c0, c1-c0)
	return ioutil.ReadAll(r)
}

func (ui *Edit) selectionText() ([]byte, error) {
	c0, c1 := ui.cursor.Ordered()
	r := io.NewSectionReader(ui.text, c0, c1-c0)
	return ioutil.ReadAll(r)
}

// Selection returns the buffer of the current selection.
func (ui *Edit) Selection() ([]byte, error) {
	ui.ensureInit()
	return ui.selectionText()
}

// Cursor returns the current cursor position, including text selection.
func (ui *Edit) Cursor() Cursor {
	return ui.cursor
}

// SetCursor sets the new cursor or selection.
// Current is the new cursor. Start is the start of the selection.
// If start < 0, it is set to current.
func (ui *Edit) SetCursor(c Cursor) {
	if c.Start < 0 {
		c.Start = c.Cur
	}
	ui.cursor = c
}

// Append adds buf to the edit contents.
func (ui *Edit) Append(buf []byte) {
	ui.ensureInit()
	defer ui.checkDirty(ui.dirty)
	size := ui.text.Size()
	ui.text.Replace(ui, &ui.dirty, Cursor{size, size}, buf, false)
	ui.cursor.Cur = size + int64(len(buf))
	ui.cursor.Start = ui.cursor.Cur
}

// Replace replaces the selection from c with buf.
func (ui *Edit) Replace(c Cursor, buf []byte) {
	ui.ensureInit()
	defer ui.checkDirty(ui.dirty)
	ui.text.Replace(ui, &ui.dirty, c, buf, false)
}

// Saved marks content as saved, calling the DirtyChanged callback if set, and updating the history state.
func (ui *Edit) Saved() {
	ui.ensureInit()
	defer ui.checkDirty(ui.dirty)
	ui.dirty = false
	ui.text.saved(ui)
}

// ScrollCursor ensure cursor is visible, scrolling if necessary.
func (ui *Edit) ScrollCursor(dui *DUI) {
	ui.ensureInit()
	ui.dui = dui
	nbr := ui.revReader(ui.cursor.Cur)
	if ui.cursor.Cur < ui.offset {
		nbr.Line(false)
		ui.offset = nbr.Offset()
		return
	}

	nbr.Line(false)
	for lines := ui.textR.Dy() / ui.font().Height; lines > 1; lines-- {
		if nbr.Offset() <= ui.offset {
			return
		}
		nbr.Line(true)
		nbr.Line(false)
	}
	ui.offset = nbr.Offset()
}

func (ui *Edit) searchRegexp(re *regexp.Regexp, reverse bool) (match bool) {
	// todo: implement reverse search

	// xxx reading entire file is ridiculous, won't work for big files. regexp needs a better reader interface...
	buf, err := ui.Text()
	if ui.error(err, "read all text") {
		return
	}
	c0, c1 := ui.cursor.Ordered()
	m := re.FindIndex(buf[c1:])
	if m != nil {
		m[0] += int(c1)
		m[1] += int(c1)
	} else {
		m = re.FindIndex(buf[:c0])
	}
	if m == nil {
		return
	}
	ui.SetCursor(Cursor{Cur: int64(m[1]), Start: int64(m[0])})
	match = true
	return
}

func (ui *Edit) searchText(t string, reverse bool) (match bool) {
	// todo: implement reverse search

	c := ui.cursor
	if c.Cur > c.Start {
		c.Start = c.Cur
	}
	first := c.Start
	restarted := false
	r := ui.EditReader(c.Start)
	seen := ""
	for !restarted || c.Start != first {
		k, err := r.TryGet()
		if err == io.EOF {
			restarted = true
			c.Start = 0
			r = ui.EditReader(c.Start)
			seen = ""
			continue
		}
		if ui.error(err, "read") {
			return
		}
		seen += string(k)
		if t == seen {
			c.Cur = c.Start + int64(len(seen))
			ui.SetCursor(c)
			match = true
			return
		}
		if strings.HasPrefix(t, seen) {
			continue
		}
		found := false
		for o := range seen {
			if strings.HasPrefix(t, seen[o:]) {
				c.Start += int64(o)
				seen = seen[o:]
				found = true
				break
			}
		}
		if !found {
			c.Start += int64(len(seen))
			seen = ""
		}
	}
	return
}

// Search finds the next occurrence of LastSearch and selects it and scrolls to it.
// The first character determines the kind of search. If slash, the remainder is interpreted as regular expression. If space (and currently anything else), the remainder is interpreted as a literal string.
func (ui *Edit) Search(dui *DUI, reverse bool) (match bool) {
	ui.ensureInit()
	ui.dui = dui
	if ui.LastSearch == "" {
		return
	}
	t := ui.LastSearch[1:]
	if ui.LastSearch[0] != '/' {
		return ui.searchText(t, reverse)
	}
	if t != ui.lastSearchRegexpString {
		var err error
		ui.lastSearchRegexp, err = regexp.Compile(t)
		if dui.error(err, "compile regexp") {
			return
		}
		ui.lastSearchRegexpString = t
	}
	return ui.searchRegexp(ui.lastSearchRegexp, reverse)
}

func (ui *Edit) indent(c Cursor) int64 {
	buf, err := ui.readText(c)
	if ui.error(err, "readText") {
		return c.size()
	}
	r := &bytes.Buffer{}
	if len(buf) >= 1 && buf[0] != '\n' {
		r.WriteByte('\t')
	}
	for i, ch := range buf {
		r.WriteByte(ch)
		if ch == '\n' {
			if i+1 < len(buf) && buf[i+1] != '\n' {
				r.WriteByte('\t')
			}
		}
	}
	rbuf := r.Bytes()
	ui.text.Replace(ui, &ui.dirty, c, rbuf, false)
	return int64(len(rbuf))
}

func (ui *Edit) unindent(c Cursor) int64 {
	buf, err := ui.readText(c)
	if ui.error(err, "readText") {
		return c.size()
	}
	ns := bytes.Replace(buf, []byte{'\n', '\t'}, []byte{'\n'}, -1)
	if len(ns) > 0 && ns[0] == '\t' {
		ns = ns[1:]
	}
	ui.text.Replace(ui, &ui.dirty, c, ns, false)
	return int64(len(ns))
}

func (ui *Edit) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	ui.dui = dui
	ui.ensureInit()
	if m.In(ui.barR) {
		log.Printf("key in scrollbar\n")
		return
	}
	if !m.In(ui.textR) {
		return
	}

	defer ui.checkDirty(ui.dirty)

	if ui.Keys != nil {
		e := ui.Keys(k, m)
		propagateEvent(self, &r, e)
		if r.Consumed {
			return
		}
	}

	r.Consumed = true
	self.Draw = Dirty

	if ui.mode != modeInsert {
		ui.text.closeHist(ui)
	}
	switch ui.mode {
	case modeCommand:
		ui.command += string(k)
		ui.commandKey(dui, &r)
		return
	case modeVisual:
		ui.visual += string(k)
		ui.visualKey(dui, false, &r)
		return
	case modeVisualLine:
		ui.visual += string(k)
		ui.visualKey(dui, true, &r)
		return
	}

	c0, c1 := ui.cursor.Ordered()
	fr := ui.reader(c1, ui.text.Size())
	br := ui.revReader(c0)
	font := ui.font()
	lines := ui.textR.Dy() / font.Height
	const Ctrl = 0x1f

	switch k {
	case draw.KeyPageUp:
		ui.scroll(-lines/2, self)
	case draw.KeyPageDown:
		ui.scroll(lines/2, self)
	case draw.KeyUp:
		ui.scroll(-lines/5, self)
	case draw.KeyDown:
		ui.scroll(lines/5, self)
	case draw.KeyLeft:
		br.TryGet()
		ui.cursor.Cur = br.Offset()
		ui.cursor.Start = ui.cursor.Cur
		ui.ScrollCursor(dui)
	case draw.KeyRight:
		fr.TryGet()
		ui.cursor.Cur = fr.Offset()
		ui.cursor.Start = ui.cursor.Cur
		ui.ScrollCursor(dui)
	case Ctrl & 'a':
		br.Line(false)
		ui.cursor.Cur = br.Offset()
		ui.cursor.Start = ui.cursor.Cur
		ui.ScrollCursor(dui)
	case Ctrl & 'e':
		fr.Line(false)
		ui.cursor.Cur = fr.Offset()
		ui.cursor.Start = ui.cursor.Cur
		ui.ScrollCursor(dui)
	case Ctrl & 'h':
		br.TryGet()
		ui.text.Replace(ui, &ui.dirty, Cursor{br.Offset(), fr.Offset()}, nil, false)
		ui.cursor.Cur = br.Offset()
		ui.cursor.Start = ui.cursor.Cur
		ui.ScrollCursor(dui)
	case Ctrl & 'w':
		c, _ := br.Peek()
		if c == '\n' {
			br.Get()
		} else {
			br.Whitespacepunct(false)
			br.Nonwhitespacepunct()
		}
		ui.text.Replace(ui, &ui.dirty, Cursor{br.Offset(), fr.Offset()}, nil, false)
		ui.cursor.Cur = br.Offset()
		ui.cursor.Start = ui.cursor.Cur
		ui.ScrollCursor(dui)
	case Ctrl & 'u':
		o := br.Offset()
		br.Line(false)
		if o == br.Offset() && o > 0 {
			br.TryGet()
		}
		ui.text.Replace(ui, &ui.dirty, Cursor{br.Offset(), fr.Offset()}, nil, false)
		ui.cursor.Cur = br.Offset()
		ui.cursor.Start = ui.cursor.Cur
		ui.ScrollCursor(dui)
	case Ctrl & 'k':
		fr.Line(false)
		ui.text.Replace(ui, &ui.dirty, Cursor{br.Offset(), fr.Offset()}, nil, false)
		ui.ScrollCursor(dui)
	case draw.KeyDelete:
		fr.TryGet()
		ui.text.Replace(ui, &ui.dirty, Cursor{br.Offset(), fr.Offset()}, nil, false)
		ui.ScrollCursor(dui)
	case draw.KeyCmd + 'a':
		ui.cursor = Cursor{0, ui.text.Size()}
	case draw.KeyCmd + 'n':
		ui.cursor.Start = ui.cursor.Cur
	case draw.KeyCmd + 'c':
		buf, err := ui.selectionText()
		if ui.error(err, "selectionText") {
			break
		}
		dui.WriteSnarf(buf)
	case draw.KeyCmd + 'x':
		buf, err := ui.selectionText()
		if ui.error(err, "selectionText") {
			break
		}
		dui.WriteSnarf(buf)
		ui.text.Replace(ui, &ui.dirty, ui.cursor, nil, false)
		ui.cursor = Cursor{c0, c0}
	case draw.KeyCmd + 'v':
		buf, ok := dui.ReadSnarf()
		if ok {
			ui.text.Replace(ui, &ui.dirty, ui.cursor, buf, false)
			ui.cursor = Cursor{c0, c0 + int64(len(buf))} // todo: keep same order in cursor
		}
	case draw.KeyCmd + 'z':
		ui.text.undo(ui)
		ui.ScrollCursor(dui)
	case draw.KeyCmd + 'Z':
		ui.text.redo(ui)
		ui.ScrollCursor(dui)
	case draw.KeyCmd + '[':
		br.Line(false)
		n := ui.unindent(Cursor{br.Offset(), fr.Offset()})
		ui.cursor = Cursor{br.Offset(), br.Offset() + n}
	case draw.KeyCmd + ']':
		br.Line(false)
		n := ui.indent(Cursor{br.Offset(), fr.Offset()})
		ui.cursor = Cursor{br.Offset(), br.Offset() + n}
	case draw.KeyCmd + 'm':
		p := ui.lastCursorPoint.Add(orig)
		r.Warp = &p
	case draw.KeyCmd + 'y':
		if len(ui.text.history) > 0 {
			h := ui.text.history[len(ui.text.history)-1]
			c0, _ := h.c.Ordered()
			c1 := c0 + int64(len(h.nbuf))
			ui.cursor = Cursor{c0, c1}
		}
	case draw.KeyCmd + '/':
		ui.Search(dui, false)
	case draw.KeyCmd + '?':
		ui.Search(dui, true)
	case draw.KeyEscape:
		// oh yeah
		if ui.cursor.Cur == ui.cursor.Start {
			ui.mode = modeCommand
		} else {
			ui.mode = modeVisual
		}

	default:
		if k >= draw.KeyCmd && k < draw.KeyCmd+128 {
			r.Consumed = false
			return
		}
		ui.text.Replace(ui, &ui.dirty, ui.cursor, []byte(string(k)), true)
		c, _ := ui.cursor.Ordered()
		c += int64(len(string(k)))
		ui.cursor = Cursor{c, c}
		ui.ScrollCursor(dui)
	}

	c0, c1 = ui.cursor.Ordered()
	if c0 < 0 || c1 > ui.text.Size() {
		log.Printf("duit: edit: bug, bad cursor cur %d,start %d after key, size of text %d\n", ui.cursor.Cur, ui.cursor.Start, ui.text.Size())
		c0 = maximum64(0, c0)
		c1 = minimum64(c1, ui.text.Size())
	}

	return
}

func (ui *Edit) FirstFocus(dui *DUI, self *Kid) (warp *image.Point) {
	p := ui.lastCursorPoint
	return &p
}

func (ui *Edit) Focus(dui *DUI, self *Kid, o UI) (warp *image.Point) {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui, self)
}

func (ui *Edit) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Edit) Print(self *Kid, indent int) {
	PrintUI("Edit", self, indent)
}

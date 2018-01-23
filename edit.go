package duit

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"9fans.net/go/draw"
)

type SeekReaderAt interface {
	io.Seeker
	io.ReaderAt
}

var (
	EditPadding = Space{0, 3, 0, 3} // lowDPI padding
)

type editMode int

const (
	modeInsert     = editMode(iota) // regular editing
	modeCommand                     // vi commands, after escape without selection
	modeVisual                      // vi visual mode, after 'v' in command mode, or escape with selection
	modeVisualLine                  // vi visual line mode, after 'V' in command mode
)

type EditColors struct {
	Fg, Bg, SelFg, SelBg, ScrollVis, ScrollBg, HoverScrollVis, HoverScrollBg, CommandBorder, VisualBorder *draw.Image
}

type Edit struct {
	NoScrollbar  bool
	Colors       *EditColors                                `json:"-"`
	Font         *draw.Font                                 `json:"-"`
	Keys         func(k rune, m draw.Mouse) (e Event)       `json:"-"`
	Click        func(m draw.Mouse, offset int64) (e Event) `json:"-"`
	DirtyChanged func(dirty bool)                           `json:"-"`

	text    *text // what we are rendering.  offset & cursors index into this text
	offset  int64 // byte offset of first line we draw
	cursor  int64 // cursor and end of selection
	cursor0 int64 // start of selection

	mode    editMode
	command string // vi command so far
	visual  string // vi visual command so far

	dirty bool

	r,
	barR,
	barActiveR,
	textR image.Rectangle

	textM,
	prevTextB1 draw.Mouse

	lastCursorPoint image.Point
}

func (ui *Edit) colors(dui *DUI) EditColors {
	if ui.Colors != nil {
		return *ui.Colors
	}
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

func NewEdit(f SeekReaderAt) *Edit {
	size, err := f.Seek(0, io.SeekEnd)
	check(err, "seek")
	parts := []textPart{}
	if size > 0 {
		parts = append(parts, &file{f, 0, size})
	}
	return &Edit{
		text: &text{parts},
	}
}

type reverseReader struct {
	src    io.ReaderAt
	offset int64 // and going to 0
}

var _ io.Reader = &reverseReader{}

func (r *reverseReader) Read(buf []byte) (int, error) {
	// log.Printf("reverseReader.Read, len buf %d, offset %d\n", len(buf), r.offset)
	want := int64(len(buf))
	if want > r.offset {
		want = r.offset
	}
	if want == 0 {
		return 0, io.EOF
	}
	have, err := r.src.ReadAt(buf[:want], r.offset-want)
	if have >= 0 {
		buf = buf[:have]
	}

	// reverse the bytes, but keep utf8 valid
	// todo: should probably provide a reader like bufio that has ReadRune and UnReadRune and others and read backwards on demand.
	onbuf := make([]byte, have)
	nbuf := onbuf
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
	copy(obuf[:], onbuf[:have])
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
	check(err, "readrune")
	r.r.UnreadRune()
	return c, false
}

func (r *reader) Get() rune {
	c, size, err := r.r.ReadRune()
	check(err, "readrune")
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

func (r *reader) RevLine() (s string, eof bool) {
	var c rune
	c, eof = r.Peek()
	if eof {
		return
	}
	r.Get()
	for {
		c, eof = r.Peek()
		if eof {
			eof = false
			break
		}
		if c == '\n' {
			break
		}
		r.Get()
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
func (r *reader) Whitespace() (s string, eof bool) {
	return r.gather(func(c rune) (keep bool) {
		return unicode.IsSpace(c)
	})
}

func (r *reader) Nonwhitespace() (s string, eof bool) {
	return r.gather(func(c rune) (keep bool) {
		return !unicode.IsSpace(c)
	})
}

func (r *reader) Whitespacepunct() (s string, eof bool) {
	return r.gather(func(c rune) (keep bool) {
		return unicode.IsSpace(c) || unicode.IsPunct(c)
	})
}

func (r *reader) Nonwhitespacepunct() (s string, eof bool) {
	return r.gather(func(c rune) (keep bool) {
		return !unicode.IsSpace(c) && !unicode.IsPunct(c)
	})
}

func (ui *Edit) ensureInit() {
	if ui.text == nil {
		ui.text = &text{}
	}
}

type EditReader interface {
	Peek() (rune, bool)
	Get() rune
	TryGet() (rune, error)
	Offset() int64
	Whitespace() (s string, eof bool)
	Nonwhitespace() (s string, eof bool)
	Whitespacepunct() (s string, eof bool)
	Nonwhitespacepunct() (s string, eof bool)
}

type ReaderReaderAt interface {
	io.Reader
	io.ReaderAt
}

func (ui *Edit) Text() string {
	buf, err := ioutil.ReadAll(ui.Reader())
	check(err, "read all text")
	return string(buf)
}

// Reader from which contents of edit can be read.
func (ui *Edit) Reader() ReaderReaderAt {
	// xxx should make copy of ui.text
	return io.NewSectionReader(ui.text, 0, ui.text.Size()-0)
}

// Reader from which contents of edit can be read, starting at offset.
func (ui *Edit) EditReader(offset int64) EditReader {
	// xxx should make copy of ui.text
	return ui.reader(offset, ui.text.Size())
}

// Reader from which contents of edit can be read in reverse (whole utf-8 characters), starting at offset, to 0.
func (ui *Edit) ReverseEditReader(offset int64) EditReader {
	// xxx should make copy of ui.text
	return ui.revReader(offset)
}

func (ui *Edit) reader(offset, size int64) *reader {
	return &reader{ui, 0, bufio.NewReader(io.NewSectionReader(ui.text, offset, size-offset)), offset, true}
}

func (ui *Edit) revReader(offset int64) *reader {
	return &reader{ui, 0, bufio.NewReader(&reverseReader{ui.text, offset}), offset, false}
}

func (ui *Edit) orderedCursor() (int64, int64) {
	if ui.cursor > ui.cursor0 {
		return ui.cursor0, ui.cursor
	}
	return ui.cursor, ui.cursor0
}

func (ui *Edit) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Edit) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("Edit", self)

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
	dui.debugDraw("Edit", self)

	ui.ensureInit()
	if ui.r.Empty() {
		return
	}

	colors := ui.colors(dui)
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

	font := ui.font(dui)
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

	c0, c1 := ui.orderedCursor()
	// log.Printf("drawing... c0 %d, c1 %d\n", c0, c1)
	drawLine := func(offsetEnd int64, eof bool) {
		origS := s

		n := len(s)
		offset := offsetEnd - int64(n)
		// log.Printf("drawLine, offset %d, offsetEnd %d, n %d\n", offset, offsetEnd, n)
		p := orig.Add(ui.textR.Min).Add(image.Pt(0, line*font.Height))

		drawCursor := func() {
			// log.Printf("drawCursor, line %d c0 %d, c1 %d, cursor %d, cursor0 %d, offset %d, offsetEnd %d, s %s, n %d\n", line, c0, c1, ui.cursor, ui.cursor0, offset, offsetEnd, s, n)
			p0 := p
			p1 := p0
			p1.Y += font.Height
			thick := 0
			if dui.Scale(1) > 1 {
				thick = 1
			}
			img.Line(p0, p1, 0, 0, thick, dui.Display.Black, image.ZP)
			ui.lastCursorPoint = p1.Sub(orig)
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

		if offset == ui.cursor && ui.cursor == c0 && c0 != c1 && offset < offsetEnd {
			// log.Printf("cursor A, offset %d, ui.cursor %d, c1 %d, offsetEnd %d, size %d\n", offset, ui.cursor, c1, offsetEnd, size)
			drawCursor()
		}

		// then selected text
		if offset >= c0 && offset < c1 && c1-c0 > 0 && offset < offsetEnd {
			nn := minimum64(c1, offsetEnd) - offset
			// log.Printf("drawing %d as selection\n", nn)
			sels := s[:nn]
			toEnd := sels[len(sels)-1] == '\n'
			if toEnd {
				sels = sels[:len(sels)-1]
			}
			seldx := font.StringWidth(sels)
			selR := rect(image.Pt(seldx, font.Height)).Add(p)
			if toEnd {
				selR.Max.X = ui.textR.Max.X + orig.X
			}
			img.Draw(selR, colors.SelBg, nil, image.ZP)
			pp := img.String(p, colors.SelFg, image.ZP, font, sels)
			p.X = pp.X
			s = s[nn:]
			offset += nn
		}
		if offset == ui.cursor && ui.cursor == c1 && (offset < offsetEnd || (offset == size && eof)) {
			// log.Printf("cursor B, offset %d, ui.cursor %d, c1 %d, offsetEnd %d, size %d\n", offset, ui.cursor, c1, offsetEnd, size)
			drawCursor()
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
			_, eof = rd.RevLine()
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
func (ui *Edit) ExpandedText() string {
	br := ui.revReader(ui.cursor)
	br.Nonwhitespace()
	fr := ui.reader(ui.cursor, ui.text.Size())
	fr.Nonwhitespace()
	return ui.readText(br.Offset(), fr.Offset())
}

func (ui *Edit) expand(offset int64, fr, br *reader) (int64, int64) {
	const (
		Starts = "[{(<\"'`"
		Ends   = "]})>\"'`"
	)

	c, eof := br.Peek()
	index := strings.IndexRune(Starts, c)
	if !eof && index >= 0 {
		br.Get()
		n := ui.expandNested(fr, rune(Starts[index]), rune(Ends[index]))
		return offset, offset + n
	}
	c, eof = fr.Peek()
	index = strings.IndexRune(Ends, c)
	if !eof && index >= 0 {
		fr.Get()
		n := ui.expandNested(br, rune(Ends[index]), rune(Starts[index]))
		return offset - n, offset
	}

	c, eof = br.Peek()
	if c == '\n' {
		// at start of line, select to end of line
		fr.Line(true)
		return offset, fr.Offset()
	} else if c, eof = fr.Peek(); c == '\n' || eof {
		// at end of line, select to start of line
		br.Line(false)
		return br.Offset(), offset
	}

	const Space = " \t\r\n\f\r"
	skip := func(isSpace bool) bool {
		return !isSpace
	}

	bc, _ := br.Peek()
	fc, _ := fr.Peek()
	if strings.ContainsAny(string(bc), Space) && strings.ContainsAny(string(fc), Space) {
		skip = func(isSpace bool) bool {
			return isSpace
		}
	}
	for {
		c, eof := br.Peek()
		if !eof && skip(strings.ContainsAny(string(c), Space)) && !strings.ContainsAny(string(c), Starts+Ends) {
			br.Get()
		} else {
			break
		}
	}
	for {
		c, eof := fr.Peek()
		if !eof && skip(strings.ContainsAny(string(c), Space)) && !strings.ContainsAny(string(c), Starts+Ends) {
			fr.Get()
		} else {
			break
		}
	}
	return offset - br.n, offset + fr.n
}

func (ui *Edit) checkDirty(odirty bool) {
	if odirty != ui.dirty && ui.DirtyChanged != nil {
		ui.DirtyChanged(ui.dirty)
	}
}

func (ui *Edit) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	ui.ensureInit()
	font := ui.font(dui)
	scrollLines := func(y int) int {
		lines := ui.textR.Dy() / font.Height
		n := lines * y / ui.textR.Dy()
		if n == 0 {
			return 1
		}
		return n
	}
	if origM.In(ui.barR) {
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
	case Button4:
		ui.scroll(-scrollLines(m.Y/4), self)
	case Button5:
		ui.scroll(scrollLines(m.Y/4), self)
	default:
		mouseOffset := func() int64 {
			rrd := ui.revReader(ui.offset)
			line := m.Y / ui.font(dui).Height
			eof := false
			for ; line < 0 && !eof; line++ {
				_, eof = rrd.RevLine()
			}
			rd := ui.reader(rrd.Offset(), ui.text.Size())
			eof = false
			for ; line > 0 && !eof; line-- {
				_, _, eof = rd.Line(true)
			}
			sdx := 0
			for {
				c, eof := rd.Peek()
				if eof || c == '\n' {
					break
				}
				dx := font.StringWidth(string(c))
				if sdx+dx/2 > m.X {
					break
				}
				sdx += dx
				rd.Get()
			}
			return rd.Offset()
		}
		if m.Buttons == Button1 {
			ui.cursor = mouseOffset()
			ui.ScrollCursor(dui)
			if om.Buttons == 0 {
				if m.Msec-ui.prevTextB1.Msec < 300 {
					ui.cursor, ui.cursor0 = ui.expand(ui.cursor, ui.reader(ui.cursor, ui.text.Size()), ui.revReader(ui.cursor))
				} else {
					ui.cursor0 = ui.cursor
				}
				ui.prevTextB1 = m
			}
			// xxx ensure cursor is visible, can happen when dragging outside UI, or through key commands
			self.Draw = Dirty
			r.Consumed = true
			return
		}
		if m.Buttons == 0 && om.Buttons&(Button1|Button2|Button3) != 0 && ui.Click != nil {
			e := ui.Click(om, mouseOffset())
			propagateEvent(self, &r, e)
		}
		if m.Buttons^om.Buttons != 0 {
			// log.Printf("in text, mouse buttons changed %v ->  %v\n", om, m)
		} else if m.Buttons != 0 && m.Buttons == om.Buttons {
			// log.Printf("in text, mouse drag %v\n", m)
		} else if om.Buttons != 0 && m.Buttons == 0 {
			// log.Printf("in text, button release %v -> %v\n", om, m)
		}
	}
	r.Consumed = true
	return
}

func (ui *Edit) readText(c0, c1 int64) string {
	r := io.NewSectionReader(ui.text, c0, c1-c0)
	buf, err := ioutil.ReadAll(r)
	check(err, "read selection")
	return string(buf)
}

func (ui *Edit) selectionText() string {
	c0, c1 := ui.orderedCursor()
	r := io.NewSectionReader(ui.text, c0, c1-c0)
	buf, err := ioutil.ReadAll(r)
	check(err, "read selection")
	return string(buf)
}

func (ui *Edit) Selection() string {
	return ui.selectionText()
}

func (ui *Edit) Cursor() (current, start int64) {
	return ui.cursor, ui.cursor0
}

// SetCursor sets the new cursor or selection.
// Current is the new cursor. start is the start of the selection.
// If start < 0, it is set to current.
func (ui *Edit) SetCursor(current, start int64) {
	if start < 0 {
		start = current
	}
	ui.cursor = current
	ui.cursor0 = start
}

func (ui *Edit) Append(buf []byte) {
	defer ui.checkDirty(ui.dirty)
	size := ui.text.Size()
	ui.text.Replace(&ui.dirty, size, size, buf)
	ui.cursor = size + int64(len(buf))
	ui.cursor0 = ui.cursor
}

func (ui *Edit) Replace(c0, c1 int64, buf []byte) {
	defer ui.checkDirty(ui.dirty)
	ui.text.Replace(&ui.dirty, c0, c1, buf)
}

// ScrollCursor ensure cursor is visible, scrolling if necessary.
func (ui *Edit) ScrollCursor(dui *DUI) {
	nbr := ui.revReader(ui.cursor)
	if ui.cursor < ui.offset {
		nbr.Line(false)
		ui.offset = nbr.Offset()
		return
	}

	nbr.Line(false)
	for lines := ui.textR.Dy() / ui.font(dui).Height; lines > 1; lines-- {
		if nbr.Offset() <= ui.offset {
			return
		}
		nbr.Line(true)
		nbr.Line(false)
	}
	ui.offset = nbr.Offset()
}

func (ui *Edit) indent(c0, c1 int64) int64 {
	s := ui.readText(c0, c1)
	buf := []byte(s)
	r := &bytes.Buffer{}
	if len(buf) >= 1 && buf[0] != '\n' {
		r.WriteByte('\t')
	}
	for i, c := range buf {
		r.WriteByte(c)
		if c == '\n' {
			if i+1 < len(buf) && buf[i+1] != '\n' {
				r.WriteByte('\t')
			}
		}
	}
	rbuf := r.Bytes()
	ui.text.Replace(&ui.dirty, c0, c1, rbuf)
	return int64(len(rbuf))
}

func (ui *Edit) unindent(c0, c1 int64) int64 {
	s := ui.readText(c0, c1)
	ns := strings.Replace(s, "\n\t", "\n", -1)
	if len(ns) > 0 && ns[0] == '\t' {
		ns = ns[1:]
	}
	ui.text.Replace(&ui.dirty, c0, c1, []byte(ns))
	return int64(len(ns))
}

func (ui *Edit) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
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

	switch ui.mode {
	case modeCommand:
		ui.commandKey(dui, k, &r)
		return
	case modeVisual:
		ui.visualKey(dui, k, false, &r)
		return
	case modeVisualLine:
		ui.visualKey(dui, k, true, &r)
		return
	}

	c0, c1 := ui.orderedCursor()
	fr := ui.reader(c1, ui.text.Size())
	br := ui.revReader(c0)
	font := ui.font(dui)
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
		ui.cursor = br.Offset()
		ui.cursor0 = ui.cursor
		ui.ScrollCursor(dui)
	case draw.KeyRight:
		fr.TryGet()
		ui.cursor = fr.Offset()
		ui.cursor0 = ui.cursor
		ui.ScrollCursor(dui)
	case Ctrl & 'a':
		br.Line(false)
		ui.cursor = br.Offset()
		ui.cursor0 = ui.cursor
		ui.ScrollCursor(dui)
	case Ctrl & 'e':
		fr.Line(false)
		ui.cursor = fr.Offset()
		ui.cursor0 = ui.cursor
		ui.ScrollCursor(dui)
	case Ctrl & 'h':
		br.TryGet()
		ui.text.Replace(&ui.dirty, br.Offset(), fr.Offset(), nil)
		ui.cursor = br.Offset()
		ui.cursor0 = ui.cursor
		ui.ScrollCursor(dui)
	case Ctrl & 'w':
		c, _ := br.Peek()
		if c == '\n' {
			br.Get()
		} else {
			br.Whitespacepunct()
			br.Nonwhitespacepunct()
		}
		ui.text.Replace(&ui.dirty, br.Offset(), fr.Offset(), nil)
		ui.cursor = br.Offset()
		ui.cursor0 = ui.cursor
		ui.ScrollCursor(dui)
	case Ctrl & 'u':
		o := br.Offset()
		br.Line(false)
		if o == br.Offset() && o > 0 {
			br.TryGet()
		}
		ui.text.Replace(&ui.dirty, br.Offset(), fr.Offset(), nil)
		ui.cursor = br.Offset()
		ui.cursor0 = ui.cursor
		ui.ScrollCursor(dui)
	case Ctrl & 'k':
		fr.Line(false)
		ui.text.Replace(&ui.dirty, br.Offset(), fr.Offset(), nil)
		ui.ScrollCursor(dui)
	case draw.KeyDelete:
		fr.TryGet()
		ui.text.Replace(&ui.dirty, br.Offset(), fr.Offset(), nil)
		ui.ScrollCursor(dui)
	case draw.KeyCmd + 'a':
		ui.cursor = 0
		ui.cursor0 = ui.text.Size()
	case draw.KeyCmd + 'n':
		ui.cursor0 = ui.cursor
	case draw.KeyCmd + 'c':
		dui.WriteSnarf([]byte(ui.selectionText()))
	case draw.KeyCmd + 'x':
		dui.WriteSnarf([]byte(ui.selectionText()))
		ui.text.Replace(&ui.dirty, c0, c1, nil)
	case draw.KeyCmd + 'v':
		buf, ok := dui.ReadSnarf()
		if ok {
			ui.text.Replace(&ui.dirty, c0, c1, buf)
		}
	case draw.KeyCmd + 'z':
		// xxx todo: undo
	case draw.KeyCmd + 'Z':
		// xxx todo: redo
	case draw.KeyCmd + '[':
		br.Line(false)
		n := ui.unindent(br.Offset(), fr.Offset())
		ui.cursor = br.Offset()
		ui.cursor0 = ui.cursor + n
	case draw.KeyCmd + ']':
		br.Line(false)
		n := ui.indent(br.Offset(), fr.Offset())
		ui.cursor = br.Offset()
		ui.cursor0 = ui.cursor + n
	case draw.KeyCmd + 'm':
		p := ui.lastCursorPoint.Add(orig)
		r.Warp = &p
	case draw.KeyEscape:
		// oh yeah
		if ui.cursor == ui.cursor0 {
			ui.mode = modeCommand
		} else {
			ui.mode = modeVisual
		}

	default:
		if k >= draw.KeyCmd && k < draw.KeyCmd+128 {
			r.Consumed = false
			return
		}
		ui.text.Replace(&ui.dirty, c0, c1, []byte(string(k)))
		ui.cursor = c0 + int64(len(string(k)))
		ui.cursor0 = ui.cursor
		ui.ScrollCursor(dui)
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

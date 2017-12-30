package duit

import (
	"bufio"
	"fmt"
	"image"
	"io"
	"log"
	"strings"
	"unicode/utf8"

	"9fans.net/go/draw"
)

type EditSource interface {
	io.Seeker
	io.ReaderAt
}

type Edit struct {
	Src  EditSource
	Font *draw.Font

	offset  int64 // byte offset of first line we draw
	cursor  int64 // cursor and end of selection
	cursor0 int64 // start of selection

	r,
	barR,
	barActiveR,
	textR image.Rectangle

	textM,
	prevTextB1 draw.Mouse
}

type reverseReader struct {
	src    io.ReaderAt
	offset int64 // and going to 0
}

var _ io.Reader = &reverseReader{}

func (r *reverseReader) Read(buf []byte) (int, error) {
	//	log.Printf("reverseReader.Read, len buf %d, offset %d\n", len(buf), r.offset)
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
	//	log.Printf("reverseReader.Read, returning n %d, err %s, buf %s\n", have, err, string(buf[:have]))
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

func (r *reader) Line() (s string, eof bool) {
	var c rune
	for {
		c, eof = r.Peek()
		if eof {
			eof = s == ""
			break
		}
		r.Get()
		if c == '\n' {
			break
		}
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

func (ui *Edit) reader(offset, size int64) *reader {
	return &reader{ui, 0, bufio.NewReader(io.NewSectionReader(ui.Src, offset, size-offset)), offset, true}
}

func (ui *Edit) revReader(offset int64) *reader {
	return &reader{ui, 0, bufio.NewReader(&reverseReader{ui.Src, offset}), offset, false}
}

func (ui *Edit) orderedCursor() (int64, int64) {
	if ui.cursor > ui.cursor0 {
		return ui.cursor0, ui.cursor
	}
	return ui.cursor, ui.cursor0
}

// size of source
func (ui *Edit) size() int64 {
	size, err := ui.Src.Seek(0, io.SeekEnd)
	check(err, "seek")
	return size
}

func (ui *Edit) font(env *Env) *draw.Font {
	return env.Font(ui.Font)
}

func (ui *Edit) Layout(env *Env, sizeAvail image.Point) (sizeTaken image.Point) {
	ui.r = rect(sizeAvail)
	ui.barR = ui.r
	ui.barR.Max.X = ui.barR.Min.X + env.Scale(ScrollbarSize)
	ui.barActiveR = ui.barR // Y's are filled in during draw
	ui.textR = ui.r
	ui.textR.Min.X = ui.barR.Max.X
	ui.textR = ui.textR.Inset(env.Scale(4))
	return sizeAvail
}

func (ui *Edit) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	if ui.r.Empty() {
		return
	}

	img.Draw(ui.textR.Add(orig), env.Normal.Background, nil, image.ZP)

	font := ui.font(env)
	s := ""
	sdx := 0
	lineWidth := ui.textR.Dx()
	line := 0

	size := ui.size()
	rd := ui.reader(ui.offset, size)
	lines := ui.textR.Dy() / font.Height

	c0, c1 := ui.orderedCursor()
	drawLine := func(offsetEnd int64) {
		origS := s

		n := len(s)
		offsetStart := offsetEnd - int64(n)
		// we draw text before cursor (selection), then selected text, then text after cursor.
		p := orig.Add(ui.textR.Min).Add(image.Pt(0, line*font.Height))

		drawCursor := func() {
			p0 := p
			p1 := p0
			p1.Y += font.Height
			img.Line(p0, p1, 0, 0, 1, env.Display.Black, image.ZP)
		}

		if offsetStart < c0 {
			nn := minimum64(int64(n), c0-offsetStart)
			pp := img.String(p, env.Normal.Text, image.ZP, font, s[:int(nn)])
			p.X = pp.X
			s = s[nn:]
			offsetStart += nn
		} else if c0 < offsetEnd {
			c0 = offsetStart
		}
		if offsetStart == ui.cursor && ui.cursor == c0 && c0 != c1 {
			drawCursor()
		}
		if offsetStart == c0 && c1-c0 > 0 && offsetEnd > offsetStart {
			nn := minimum64(c1-c0, offsetEnd-offsetStart)
			sels := s[:int(nn)]
			toEnd := sels[len(sels)-1] == '\n'
			if toEnd {
				sels = sels[:len(sels)-1]
			}
			seldx := font.StringWidth(sels)
			selR := rect(image.Pt(seldx, font.Height)).Add(p)
			if toEnd {
				selR.Max.X = ui.textR.Max.X
			}
			img.Draw(selR, env.Inverse.Background, nil, image.ZP)
			pp := img.String(p, env.Inverse.Text, image.ZP, font, sels)
			p.X = pp.X
			s = s[nn:]
			offsetStart += nn
		}
		if offsetStart == ui.cursor && ui.cursor == c1 {
			drawCursor()
		}
		if offsetStart >= c1 && offsetEnd > offsetStart {
			nn := int(offsetEnd - offsetStart)
			pp := img.String(p, env.Normal.Text, image.ZP, font, s)
			p.X = pp.X
			s = s[nn:]
			offsetStart += int64(nn)
		}
		if s != "" || offsetStart != offsetEnd {
			panic(fmt.Sprintf("bug in drawLine, s %v, offsetStart %d, offsetEnd %d, c0 %d c1 %d, line %d, sdx %d, origS %s", s, offsetStart, offsetEnd, c0, c1, line, sdx, origS))
		}

		s = ""
		sdx = 0
		line++
	}

	var lastC rune
	for line < lines {
		c, eof := rd.Peek()
		if eof {
			if s != "" {
				drawLine(rd.Offset())
			} else if ui.cursor == size && (lastC == '\n' || size == 0) {
				drawLine(rd.Offset())
			}
			break
		}
		lastC = c
		if c == '\n' {
			s += string(rd.Get())
			drawLine(rd.Offset())
			continue
		}
		dx := font.StringWidth(string(c))
		if sdx+dx < lineWidth {
			sdx += dx
			s += string(rd.Get())
			continue
		}
		drawLine(rd.Offset())
		s = string(c)
		sdx = dx
	}

	barHover := m.In(ui.barR)
	bg := env.ScrollBGNormal
	vis := env.ScrollVisibleNormal
	if barHover {
		bg = env.ScrollBGHover
		vis = env.ScrollVisibleHover
	}

	ui.barActiveR.Min.Y = int(int64(ui.barR.Dy()) * ui.offset / maximum64(1, size))
	ui.barActiveR.Max.Y = int(int64(ui.barR.Dy()) * rd.Offset() / maximum64(1, size))
	img.Draw(ui.barR.Add(orig), bg, nil, image.ZP)
	img.Draw(ui.barActiveR.Add(orig), vis, nil, image.ZP)
}

func (ui *Edit) scroll(lines int, r *Result) {
	offset := ui.offset
	if lines > 0 {
		rd := ui.reader(ui.offset, ui.size())
		eof := false
		for ; lines > 0 && !eof; lines-- {
			_, eof = rd.Line()
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

	r.Redraw = offset != ui.offset
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

func (ui *Edit) expand(offset int64, fr, br *reader) (int64, int64) {
	const (
		Starts = "[{(<\"'`"
		Ends   = "]})>\"'`"
	)

	c, eof := br.Peek()
	index := strings.IndexRune(Starts, c)
	if !eof && index >= 0 {
		n := ui.expandNested(fr, rune(Starts[index]), rune(Ends[index]))
		return offset, offset + n
	}
	c, eof = fr.Peek()
	index = strings.IndexRune(Ends, c)
	if !eof && index >= 0 {
		n := ui.expandNested(br, rune(Ends[index]), rune(Starts[index]))
		return offset - n, offset
	}

	const Space = " \t\r\n\f"
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

func (ui *Edit) Mouse(env *Env, m draw.Mouse) (r Result) {
	font := ui.font(env)
	scrollLines := func(y int) int {
		lines := ui.textR.Dy() / font.Height
		n := lines * y / ui.textR.Dy()
		if n == 0 {
			return 1
		}
		return n
	}
	r.Hit = ui
	if m.In(ui.barR) {
		switch m.Buttons {
		case Button1:
			ui.scroll(-scrollLines(m.Y), &r)
		case Button2:
			rd := ui.revReader(ui.size() * int64(m.Y) / int64(ui.textR.Dy()))
			for {
				c, eof := rd.Peek()
				if eof || c == '\n' {
					break
				}
				rd.Get()
			}
			r.Redraw = rd.Offset() != ui.offset
			ui.offset = rd.Offset()
		case Button3:
			ui.scroll(scrollLines(m.Y), &r)
		case Button4:
			ui.scroll(-scrollLines(m.Y/4), &r)
		case Button5:
			ui.scroll(scrollLines(m.Y/4), &r)
		}
		return
	}
	if !m.In(ui.textR) {
		return
	}
	m.Point = m.Point.Sub(ui.textR.Min)
	om := ui.textM
	ui.textM = m
	switch m.Buttons {
	case Button4:
		ui.scroll(-scrollLines(m.Y/4), &r)
	case Button5:
		ui.scroll(scrollLines(m.Y/4), &r)
	default:
		if m.Buttons == Button1 {
			rd := ui.reader(ui.offset, ui.size())
			eof := false
			for line := m.Y / ui.font(env).Height; line > 0 && !eof; line-- {
				_, eof = rd.Line()
			}
			startLineOffset := rd.Offset()
			log.Printf("click, startLineOffset %d\n", startLineOffset)
			sdx := 0
			xchars := 0
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
				xchars++
			}
			ui.cursor = rd.Offset()
			if om.Buttons == 0 {
				if m.Msec-ui.prevTextB1.Msec < 300 {
					if xchars == 0 {
						// at start of line, select to end of line
						rd.Line()
						ui.cursor0 = rd.Offset()
					} else {
						c, eof := rd.Peek()
						if eof || c == '\n' {
							// at end of line, select to start of line
							ui.cursor0 = startLineOffset
						} else {
							// somewhere else, try to expand
							ui.cursor, ui.cursor0 = ui.expand(ui.cursor, ui.reader(ui.cursor, ui.size()), ui.revReader(ui.cursor))
						}
					}
				} else {
					ui.cursor0 = ui.cursor
				}
				ui.prevTextB1 = m
			}
			// xxx ensure cursor is visible, can happen when dragging outside UI, or through key commands
			r.Redraw = true
			r.Consumed = true
			return
		}
		if m.Buttons^om.Buttons != 0 {
			log.Printf("in text, mouse buttons changed %v ->  %v\n", om, m)
		} else if m.Buttons != 0 && m.Buttons == om.Buttons {
			log.Printf("in text, mouse drag %v\n", m)
		} else if om.Buttons != 0 && m.Buttons == 0 {
			log.Printf("in text, button release %v -> %v\n", om, m)
		}
	}
	r.Consumed = true
	return
}

func (ui *Edit) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result) {
	r.Hit = ui
	if m.In(ui.barR) {
		log.Printf("key in scrollbar\n")
		return
	}
	if !m.In(ui.textR) {
		return
	}
	log.Printf("key in text\n")
	return
}

func (ui *Edit) FirstFocus(env *Env) (warp *image.Point) {
	return &ui.textR.Min
}

func (ui *Edit) Focus(env *Env, o UI) (warp *image.Point) {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(env)
}

func (ui *Edit) Print(indent int, r image.Rectangle) {
	uiPrint("Edit", indent, r)
}

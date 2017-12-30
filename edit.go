package duit

import (
	"bufio"
	"fmt"
	"image"
	"io"
	"log"
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
	m draw.Mouse
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
	ui *Edit
	n  int64 // number of bytes read, excluding peek
	r  *bufio.Reader
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
	if c != '\n' {
		// xxx could happen due to change on disk. how to handle?
		panic(fmt.Sprintf("RevLine called at offset not preceeded by newline, char %c (%#x)", c, c))
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
	return &reader{ui: ui, n: 0, r: bufio.NewReader(io.NewSectionReader(ui.Src, offset, size-offset))}
}

func (ui *Edit) revReader(offset int64) *reader {
	return &reader{ui: ui, n: 0, r: bufio.NewReader(&reverseReader{ui.Src, offset})}
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

	drawLine := func() {
		img.String(ui.textR.Min.Add(image.Pt(0, line*font.Height)).Add(orig), env.Normal.Text, image.ZP, font, s)
		s = ""
		sdx = 0
		line++
	}

	size := ui.size()
	rd := ui.reader(ui.offset, size)
	lines := ui.textR.Dy() / font.Height
	for line < lines {
		c, eof := rd.Peek()
		if eof {
			break
		}
		if c == '\n' {
			rd.Get()
			drawLine()
			continue
		}
		dx := font.StringWidth(string(c))
		if sdx+dx < lineWidth {
			sdx += dx
			s += string(rd.Get())
			continue
		}
		drawLine()
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

	ui.barActiveR.Min.Y = int(int64(ui.barR.Dy()) * ui.offset / size)
	ui.barActiveR.Max.Y = int(int64(ui.barR.Dy()) * (ui.offset + rd.n) / size)
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
		offset += rd.n
	} else if lines < 0 {
		rd := ui.revReader(ui.offset)
		eof := false
		for ; lines < 0 && !eof; lines++ {
			_, eof = rd.RevLine()
		}
		offset -= rd.n
	}

	r.Redraw = offset != ui.offset
	ui.offset = offset
}

func (ui *Edit) Mouse(env *Env, m draw.Mouse) (r Result) {
	scrollLines := func(y int) int {
		lines := ui.textR.Dy() / ui.font(env).Height
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
			offset := ui.size() * int64(m.Y) / int64(ui.textR.Dy())
			rd := ui.revReader(offset)
			for {
				c, eof := rd.Peek()
				if eof || c == '\n' {
					break
				}
				rd.Get()
			}
			offset -= rd.n
			r.Redraw = offset != ui.offset
			ui.offset = offset
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
	om := ui.m
	ui.m = m
	switch m.Buttons {
	case Button4:
		ui.scroll(-scrollLines(m.Y/4), &r)
	case Button5:
		ui.scroll(scrollLines(m.Y/4), &r)
	default:
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

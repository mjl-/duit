package duit

import (
	"fmt"
	"io"
	"strings"
)

type sizer interface {
	Size() int64
}

type textSource interface {
	sizer
	io.ReaderAt
}

type textPart interface {
	textSource
	Split(offset int64) (textPart, textPart)
	Skip(lead int64) textPart
	Merge(tp textPart) (ntp textPart, merged bool)
	fmt.Stringer
}

type textHist struct {
	clean      bool // not dirty
	c          Cursor
	obuf, nbuf []byte
}

type text struct {
	file    SeekReaderAt
	l       []textPart
	history []textHist
	future  []textHist
	open    bool // whether next replace can be added to the top history
}

var _ textSource = &text{}

func (t *text) saved(ui *Edit) {
	if t.file != nil {
		size, err := t.file.Seek(0, io.SeekEnd)
		if ui.error(err, "seek") {
			return
		}
		if size > 0 {
			t.l = []textPart{&file{t.file, 0, size}}
		}
	} else if t.Size() > 0 {
		buf, err := ui.Text()
		if ui.error(err, "text") {
			return
		}
		t.l = []textPart{stretch(buf)}
	} else {
		t.l = nil
	}
	t.closeHist(ui)
	if len(t.history) > 0 {
		last := len(t.history) - 1
		for i := range t.history {
			t.history[i].clean = i == last
		}
	}
	// xxx modify the offsets for the histories instead of trashing them
	t.history = nil
	t.future = nil
}

func (t *text) undo(ui *Edit) {
	// log.Printf("undo, history %#v\n", t.history)
	if len(t.history) == 0 {
		return
	}
	h := t.history[len(t.history)-1]
	t.history = t.history[:len(t.history)-1]
	t.future = append(t.future, h)
	t.closeHist(ui)
	var dirty bool
	c0, _ := h.c.Ordered()
	c1 := c0 + int64(len(h.nbuf))
	buf := h.obuf
	t.ReplaceHist(&dirty, Cursor{c0, c1}, buf, false)
	t.open = false
	c := c0 + int64(len(buf))
	ui.cursor = Cursor{c, c}
	defer ui.checkDirty(ui.dirty)
	ui.dirty = !h.clean
}

func (t *text) redo(ui *Edit) {
	if len(t.future) == 0 {
		return
	}
	h := t.future[len(t.future)-1]
	t.future = t.future[:len(t.future)-1]
	t.history = append(t.history, h)
	t.closeHist(ui)
	var dirty bool
	t.ReplaceHist(&dirty, h.c, h.nbuf, false)
	t.open = false
	c := h.c.Start + int64(len(h.nbuf))
	ui.cursor = Cursor{c, c}
	defer ui.checkDirty(ui.dirty)
	ui.dirty = !h.clean
}

func (t *text) ReadAt(buf []byte, offset int64) (int, error) {
	// log.Printf("text.ReadAt n %d, offset %d, t %v\n", len(buf), offset, t)
	for _, tp := range t.l {
		size := tp.Size()
		if offset >= size {
			offset -= size
			continue
		}
		// log.Printf("text.readAt, offset %d, size %d, tp %v\n", offset, size, tp)
		n := minimum64(int64(len(buf)), size-offset)
		// xxx read from multiple parts
		return tp.ReadAt(buf[:n], offset)
	}
	return 0, io.EOF
}

func (t *text) Size() (n int64) {
	for _, tt := range t.l {
		n += tt.Size()
	}
	return
}

func (t *text) TryMergeWithBefore(i int) bool {
	if i-1 < 0 || i >= len(t.l) {
		return false
	}
	m, ok := t.l[i-1].Merge(t.l[i])
	if ok {
		// log.Printf("merged i %d with i-1 %d\n", i, i-1)
		t.l[i-1] = m
		copy(t.l[i:], t.l[i+1:])
		t.l = t.l[:len(t.l)-1]
	}
	return ok
}

func (t *text) closeHist(ui *Edit) {
	t.open = false
	if ui.needLastCommandText {
		ui.needLastCommandText = false
		if len(t.history) > 0 {
			buf := t.history[len(t.history)-1].nbuf
			nbuf := make([]byte, len(buf))
			copy(nbuf, buf)
			ui.lastCommandText = nbuf
		}
	}
}

func (t *text) Replace(ui *Edit, dirty *bool, c Cursor, buf []byte, open bool) {
	wasOpen := t.open
	t.open = t.open && open && len(t.future) == 0
	if wasOpen && !t.open {
		t.closeHist(ui)
	}
	t.ReplaceHist(dirty, c, buf, true)
	t.open = open
}

func (t *text) get(c Cursor) (buf []byte, err error) {
	c0, c1 := c.Ordered()
	buf = make([]byte, int(c1-c0))
	n, err := readAtFull(t, buf, c0)
	if n != len(buf) && (err == nil || err == io.EOF) {
		err = fmt.Errorf("short read for history buffer, n %d != len buf %d", n, len(buf))
	} else {
		err = nil
	}
	return
}

func (t *text) ReplaceHist(dirty *bool, c Cursor, buf []byte, recordHist bool) {
	s, e := c.Ordered()
	// log.Printf("replaceHist s %d, e %d, buf %v\n", s, e, buf)

	if s == e && len(buf) == 0 {
		return
	}

	if s > e {
		panic("bad replace")
	}

	*dirty = true

	if recordHist {
		var obuf []byte
		if e > s {
			// read current content, we'll put it in history
			var err error
			obuf, err = t.get(Cursor{e, s})
			if err != nil {
				panic("error reading for history: " + err.Error())
			}
		}
		recorded := false
		if t.open && len(t.history) > 0 {
			i := len(t.history) - 1
			h := t.history[i]
			c0, _ := h.c.Ordered()
			if c0+int64(len(h.nbuf)) == s {
				t.history[i].nbuf = append(h.nbuf, buf...)
				recorded = true
			}
		}
		if !recorded {
			h := textHist{c: c, obuf: obuf}
			h.nbuf = make([]byte, len(buf))
			copy(h.nbuf, buf)
			t.history = append(t.history, h)
		}
		t.future = nil
	}

	// insert at i
	insert := func(i int, tp textPart) {
		if i == len(t.l) {
			t.l = append(t.l, tp)
			return
		}
		tl := append(t.l, nil)
		copy(tl[i+1:], t.l[i:])
		t.l = tl
		t.l[i] = tp
	}

	insertBuf := func(i int) {
		if len(buf) == 0 {
			panic("bad insertBuf call")
		}
		nbuf := make([]byte, len(buf))
		copy(nbuf, buf)
		insert(i, stretch(nbuf))
		buf = nil
	}

	drop := e - s
	i := 0
	for i < len(t.l) && (drop > 0 || len(buf) > 0) {
		// log.Printf("replace, loop, drop %d, i %d, s %d, t %v\n", drop, i, s, t)
		ts := t.l[i]

		size := ts.Size()
		if s >= size {
			s -= size
			i++
			continue
		}

		if s > 0 {
			ts0, ts1 := ts.Split(s)
			t.l[i] = ts0
			insert(i+1, ts1)
			i++
			s = 0
			continue
		}

		if drop >= size {
			copy(t.l[i:], t.l[i+1:])
			t.l = t.l[:len(t.l)-1]
			drop -= size
			continue
		}
		if drop > 0 {
			t.l[i] = ts.Skip(drop)
			drop = 0
			continue
		}

		// log.Printf("inserting buf from loop, i %d\n", i)
		insertBuf(i)
		t.TryMergeWithBefore(i + 1)
		t.TryMergeWithBefore(i)
	}
	if len(buf) > 0 {
		// log.Printf("at end, insert buf, i %d, s %d, t %v\n", i, s, t)
		insertBuf(i)
		t.TryMergeWithBefore(i + 1)
		t.TryMergeWithBefore(i)
	}
	// log.Printf("replace done, t %v\n", t)
}

func (t *text) String() string {
	l := make([]string, len(t.l))
	for i, tp := range t.l {
		l[i] = tp.String()
	}
	return fmt.Sprintf("Text(%s)", strings.Join(l, ", "))
}

type stretch []byte

var _ textPart = stretch(nil)

func (s stretch) ReadAt(buf []byte, offset int64) (int, error) {
	if offset < 0 {
		return -1, fmt.Errorf("read at negative offset")
	}
	if offset >= int64(len(s)) {
		return 0, io.EOF
	}
	start := int(offset)
	n := minimum(len(buf), len(s)-int(offset))
	copy(buf, s[start:start+n])
	return n, nil
}

func (s stretch) Size() int64 {
	return int64(len(s))
}

func (s stretch) Split(offset int64) (textPart, textPart) {
	if offset < 0 || offset >= int64(len(s)) {
		panic("bad call of stretch.Split")
	}
	o := int(offset)
	s0 := s[:o]
	s1 := make([]byte, len(s)-o)
	copy(s1, s[o:])
	return stretch(s0), stretch(s1)
}

func (s stretch) Skip(n int64) textPart {
	if n < 0 || n >= int64(len(s)) {
		panic("bad stretch.Skip call")
	}
	nn := int(n)
	return s[nn:]
}

func (s stretch) Merge(tp textPart) (textPart, bool) {
	os, ok := tp.(stretch)
	if ok {
		return append(s, os...), true
	}
	return nil, false
}

func (s stretch) String() string {
	return fmt.Sprintf("stretch(%d)", len(s))
}

type file struct {
	f      io.ReaderAt
	offset int64 // we read from f starting at offset
	size   int64 // of this part, not of entire file
}

var _ textPart = &file{}

func (f *file) ReadAt(buf []byte, offset int64) (n int, err error) {
	return f.f.ReadAt(buf, f.offset+offset)
}

func (f *file) Size() int64 {
	return f.size
}

func (f *file) Split(offset int64) (textPart, textPart) {
	if offset < 0 || offset >= f.size {
		panic("bad call of file.Split")
	}
	f0 := &file{f.f, f.offset, offset}
	f1 := &file{f.f, f.offset + offset, f.size - offset}
	return f0, f1
}

func (f *file) Skip(n int64) textPart {
	if n < 0 || n >= f.size {
		panic("bad file.Skip call")
	}
	f.offset += n
	f.size -= n
	return f
}

func (f *file) Merge(tp textPart) (nf textPart, merged bool) {
	of, ok := tp.(*file)
	if ok && f.f == of.f && of.offset == f.offset+f.size {
		return &file{f.f, f.offset, f.size + of.size}, true
	}
	return nil, false
}

func (f *file) String() string {
	return fmt.Sprintf("file(o %d, n %d)", f.offset, f.size)
}

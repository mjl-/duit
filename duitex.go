package main

import (
	"fmt"
	"image"
	imagedraw "image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"time"

	"9fans.net/go/draw"
)

var (
	display *draw.Display
	screen  *draw.Image
)

const (
	Margin  = 10
	Padding = 10
	Border  = 1
	Space   = Margin + Border + Padding

	ScrollbarWidth = 15

	WheelUp   = 0xA
	WheelDown = 0xFFFFFFFE

	Fn1 = 0xf001

	ArrowUp   = 0xf00e
	ArrowDown = 0x80
	PageUp    = 0xf00f
	PageDown  = 0xf013
)

type Result struct {
	Hit      UI           // the UI where the event ended up
	Consumed bool         // whether event was consumed, and should not be further handled by upper UI's
	Redraw   bool         // whether event needs a redraw after
	Warp     *image.Point // if set, mouse will warp to location
}

type UI interface {
	Layout(r image.Rectangle, cur image.Point) image.Point
	Draw(img *draw.Image, orig image.Point, m draw.Mouse)
	Mouse(m draw.Mouse) (result Result)
	Key(orig image.Point, m draw.Mouse, k rune) (result Result)

	// FirstFocus returns the top-left corner where the focus should go next when "tab" is hit, if anything.
	FirstFocus() *image.Point
}

type Label struct {
	Text string
}

func (ui *Label) Layout(r image.Rectangle, cur image.Point) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*Margin + 2*Border, 2 * Space})
}
func (ui *Label) Draw(img *draw.Image, orig image.Point, m draw.Mouse) {
	img.String(orig.Add(image.Point{Margin + Border, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Label) Mouse(m draw.Mouse) Result {
	return Result{ui, false, false, nil}
}
func (ui *Label) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{ui, false, false, nil}
}
func (ui *Label) FirstFocus() *image.Point {
	return nil
}

type Field struct {
	Text string

	size image.Point // including space
}

func (ui *Field) Layout(r image.Rectangle, cur image.Point) image.Point {
	ui.size = image.Point{r.Dx(), 2*Space + display.DefaultFont.Height}
	return ui.size
}
func (ui *Field) Draw(img *draw.Image, orig image.Point, m draw.Mouse) {
	hover := m.In(image.Rectangle{image.ZP, ui.size})
	r := image.Rectangle{orig, orig.Add(ui.size)}
	img.Draw(r, display.White, nil, image.ZP)

	color := display.Black
	if hover {
		var err error
		color, err = display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, draw.Blue)
		check(err, "allocimage")
	}
	img.Border(
		image.Rectangle{
			orig.Add(image.Point{Margin, Margin}),
			orig.Add(ui.size).Sub(image.Point{Margin, Margin}),
		},
		1, color, image.ZP)
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Field) Mouse(m draw.Mouse) Result {
	return Result{ui, false, false, nil}
}
func (ui *Field) Key(orig image.Point, m draw.Mouse, c rune) Result {
	switch c {
	case PageUp, PageDown, ArrowUp, ArrowDown:
		return Result{ui, false, false, nil}
	case '\t':
		return Result{ui, false, false, nil}
	case 8:
		if ui.Text != "" {
			ui.Text = ui.Text[:len(ui.Text)-1]
		}
	default:
		ui.Text += string(c)
	}
	return Result{ui, true, true, nil}
}
func (ui *Field) FirstFocus() *image.Point {
	p := image.Pt(Space, Space)
	return &p
}

type Button struct {
	Text  string
	Click func()

	m draw.Mouse
}

func (ui *Button) Layout(r image.Rectangle, cur image.Point) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2 * Space, 2 * Space})
}
func (ui *Button) Draw(img *draw.Image, orig image.Point, m draw.Mouse) {
	size := display.DefaultFont.StringSize(ui.Text)

	grey, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, draw.Palegreygreen)
	check(err, "allocimage grey")

	r := image.Rectangle{
		orig.Add(image.Point{Margin + Border, Margin + Border}),
		orig.Add(size).Add(image.Point{2*Padding + Margin + Border, 2*Padding + Margin + Border}),
	}
	hover := m.In(image.Rectangle{image.ZP, size.Add(image.Pt(2*Space, 2*Space))})
	borderColor := grey
	if hover {
		borderColor = display.Black
	}
	img.Draw(r, grey, nil, image.ZP)
	img.Border(image.Rectangle{orig.Add(image.Point{Margin, Margin}), orig.Add(size).Add(image.Point{Margin + 2*Padding + 2*Border, Margin + 2*Padding + 2*Border})}, 1, borderColor, image.ZP)
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Button) Mouse(m draw.Mouse) Result {
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 && ui.Click != nil {
		ui.Click()
	}
	ui.m = m
	return Result{ui, false, false, nil}
}
func (ui *Button) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{ui, false, false, nil}
}
func (ui *Button) FirstFocus() *image.Point {
	p := image.Pt(Space, Space)
	return &p
}

type Image struct {
	Image *draw.Image
}

func (ui *Image) Layout(r image.Rectangle, cur image.Point) image.Point {
	return ui.Image.R.Size()
}
func (ui *Image) Draw(img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(image.Rectangle{orig, orig.Add(ui.Image.R.Size())}, ui.Image, nil, image.ZP)
}
func (ui *Image) Mouse(m draw.Mouse) Result {
	return Result{ui, false, false, nil}
}
func (ui *Image) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{ui, false, false, nil}
}
func (ui *Image) FirstFocus() *image.Point {
	return nil
}

type Kid struct {
	UI UI
	R  image.Rectangle
}

// box keeps elements on a line as long as they fit
type Box struct {
	Kids []*Kid

	size image.Point
}

func (ui *Box) Layout(r image.Rectangle, ocur image.Point) image.Point {
	xmax := 0
	cur := image.Point{0, 0}
	nx := 0    // number on current line
	liney := 0 // max y of current line
	for _, k := range ui.Kids {
		p := k.UI.Layout(r, cur)
		var kr image.Rectangle
		if nx == 0 || cur.X+p.X <= r.Dx() {
			kr = image.Rectangle{cur, cur.Add(p)}
			cur.X += p.X
			if p.Y > liney {
				liney = p.Y
			}
			nx += 1
		} else {
			cur.X = 0
			cur.Y += liney
			kr = image.Rectangle{cur, cur.Add(p)}
			nx = 1
			cur.X = p.X
			liney = p.Y
		}
		k.R = kr
		if xmax < cur.X {
			xmax = cur.X
		}
	}
	if len(ui.Kids) > 0 {
		cur.Y += liney
	}
	ui.size = image.Point{xmax, cur.Y}
	return ui.size
}
func (ui *Box) Draw(img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(image.Rectangle{orig, orig.Add(ui.size)}, display.White, nil, image.ZP)
	for _, k := range ui.Kids {
		mm := m
		mm.Point = mm.Point.Sub(k.R.Min)
		k.UI.Draw(img, orig.Add(k.R.Min), mm)
	}
}
func (ui *Box) Mouse(m draw.Mouse) Result {
	for _, k := range ui.Kids {
		if m.Point.In(k.R) {
			m.Point = m.Point.Sub(k.R.Min)
			return k.UI.Mouse(m)
		}
	}
	return Result{nil, false, false, nil}
}
func (ui *Box) Key(orig image.Point, m draw.Mouse, c rune) Result {
	for i, k := range ui.Kids {
		if m.Point.In(k.R) {
			m.Point = m.Point.Sub(k.R.Min)
			r := k.UI.Key(orig.Add(k.R.Min), m, c)
			if !r.Consumed && c == '\t' {
				for next := i + 1; next < len(ui.Kids); next++ {
					first := ui.Kids[next].UI.FirstFocus()
					if first != nil {
						kR := ui.Kids[next].R
						p := first.Add(orig).Add(kR.Min)
						r.Warp = &p
						r.Consumed = true
						break
					}
				}
			}
			return r
		}
	}
	return Result{ui, false, false, nil}
}
func (ui *Box) FirstFocus() *image.Point {
	if len(ui.Kids) == 0 {
		return nil
	}
	for _, k := range ui.Kids {
		first := k.UI.FirstFocus()
		if first != nil {
			p := first.Add(k.R.Min)
			return &p
		}
	}
	return nil
}

// Scroll shows a part of its single child, typically a box, and lets you scroll the content.
type Scroll struct {
	Child UI

	r         image.Rectangle // entire ui
	barR      image.Rectangle
	childSize image.Point
	offset    int         // current scroll offset in pixels
	img       *draw.Image // for child to draw on
}

func (ui *Scroll) Layout(r image.Rectangle, cur image.Point) image.Point {
	ui.r = image.Rect(r.Min.X, cur.Y, r.Max.X, r.Max.Y)
	ui.barR = image.Rectangle{ui.r.Min, image.Pt(ui.r.Min.X+ScrollbarWidth, ui.r.Max.Y)}
	ui.childSize = ui.Child.Layout(image.Rectangle{image.ZP, image.Pt(ui.r.Dx()-ui.barR.Dx(), ui.r.Dy())}, image.ZP)
	if ui.r.Dy() > ui.childSize.Y {
		ui.barR.Max.Y = ui.childSize.Y
		ui.r.Max.Y = ui.childSize.Y
	}
	var err error
	ui.img, err = display.AllocImage(image.Rectangle{image.ZP, ui.childSize}, draw.ARGB32, false, draw.White)
	check(err, "allocimage")
	return ui.r.Size()
}
func (ui *Scroll) Draw(img *draw.Image, orig image.Point, m draw.Mouse) {
	// draw scrollbar
	lightGrey, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0xEEEEEEFF)
	check(err, "allowimage lightgrey")
	darkerGrey, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0xAAAAAAFF)
	check(err, "allowimage darkergrey")
	barR := ui.barR.Add(orig)
	img.Draw(barR, lightGrey, nil, image.ZP)
	barRActive := barR
	h := ui.r.Dy()
	uih := ui.childSize.Y
	if uih > h {
		barH := int((float32(h) / float32(uih)) * float32(h))
		barY := int((float32(ui.offset) / float32(uih)) * float32(h))
		barRActive.Min.Y += barY
		barRActive.Max.Y = barRActive.Min.Y + barH
	}
	img.Draw(barRActive, darkerGrey, nil, image.ZP)

	// draw child ui
	ui.img.Draw(ui.img.R, display.White, nil, image.ZP)
	m.Point = m.Point.Add(image.Pt(-ScrollbarWidth, ui.offset))
	ui.Child.Draw(ui.img, image.Pt(0, -ui.offset), m)
	img.Draw(ui.img.R.Add(orig).Add(image.Pt(ScrollbarWidth, 0)), ui.img, nil, image.ZP)
}
func (ui *Scroll) scroll(delta int) bool {
	o := ui.offset
	ui.offset += delta
	if ui.offset < 0 {
		ui.offset = 0
	}
	offsetMax := ui.childSize.Y - ui.r.Dy()
	if ui.offset > offsetMax {
		ui.offset = offsetMax
	}
	return o != ui.offset
}
func (ui *Scroll) scrollKey(c rune) (consumed bool) {
	switch c {
	case ArrowUp:
		return ui.scroll(-50)
	case ArrowDown:
		return ui.scroll(50)
	case PageUp:
		return ui.scroll(-200)
	case PageDown:
		return ui.scroll(200)
	}
	return false
}
func (ui *Scroll) scrollMouse(m draw.Mouse) (consumed bool) {
	switch m.Buttons {
	case WheelUp:
		return ui.scroll(-50)
	case WheelDown:
		return ui.scroll(50)
	}
	return false
}
func (ui *Scroll) Mouse(m draw.Mouse) Result {
	if m.Point.In(ui.barR) {
		consumed := ui.scrollMouse(m)
		redraw := consumed
		return Result{ui, consumed, redraw, nil}
	}
	if m.Point.In(ui.r) {
		m.Point = m.Point.Add(image.Pt(-ScrollbarWidth, ui.offset))
		r := ui.Child.Mouse(m)
		if !r.Consumed {
			r.Consumed = ui.scrollMouse(m)
			r.Redraw = r.Redraw || r.Consumed
		}
		return r
	}
	return Result{nil, false, false, nil}
}
func (ui *Scroll) Key(orig image.Point, m draw.Mouse, c rune) Result {
	if m.Point.In(ui.barR) {
		consumed := ui.scrollKey(c)
		redraw := consumed
		return Result{ui, consumed, redraw, nil}
	}
	if m.Point.In(ui.r) {
		m.Point = m.Point.Add(image.Pt(-ScrollbarWidth, ui.offset))
		r := ui.Child.Key(orig.Add(image.Pt(ScrollbarWidth, -ui.offset)), m, c)
		if !r.Consumed {
			r.Consumed = ui.scrollKey(c)
			r.Redraw = r.Redraw || r.Consumed
		}
		return r
	}
	return Result{nil, false, false, nil}
}
func (ui *Scroll) FirstFocus() *image.Point {
	first := ui.Child.FirstFocus()
	if first == nil {
		return nil
	}
	p := first.Add(image.Pt(ScrollbarWidth, -ui.offset))
	return &p
}

func NewBox(uis ...UI) *Box {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return &Box{Kids: kids}
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	var err error
	display, err = draw.Init(nil, "", "duitex", "600x400")
	check(err, "draw init")
	screen = display.ScreenImage

	mousectl := display.InitMouse()
	kbdctl := display.InitKeyboard()
	//whitemask, _ := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0x7F7F7F7F)

	redraw := make(chan struct{}, 1)

	readImage := func(f io.Reader) *draw.Image {
		img, _, err := image.Decode(f)
		check(err, "decoding image")
		var rgba *image.RGBA
		switch i := img.(type) {
		case *image.RGBA:
			rgba = i
		default:
			b := img.Bounds()
			rgba = image.NewRGBA(image.Rectangle{image.ZP, b.Size()})
			imagedraw.Draw(rgba, rgba.Bounds(), img, b.Min, imagedraw.Src)
		}

		// todo: colors are wrong. it should be RGBA32, but that looks even worse.

		ni, err := display.AllocImage(rgba.Bounds(), draw.ARGB32, false, draw.White)
		check(err, "allocimage")
		_, err = ni.Load(rgba.Bounds(), rgba.Pix)
		check(err, "load image")
		return ni
	}

	readImagePath := func(path string) *draw.Image {
		f, err := os.Open(path)
		check(err, "open image")
		defer f.Close()
		return readImage(f)
	}

	count := 0
	counter := &Label{Text: fmt.Sprintf("%d", count)}

	var top UI = &Scroll{Child: NewBox(
		&Label{Text: "counter:"},
		counter,
		&Button{Text: "button1", Click: func() { log.Printf("button clicked") }},
		&Button{Text: "button2"},
		&Scroll{Child: NewBox(
			&Label{Text: "another label, this one is somewhat longer"},
			&Button{Text: "some other button"},
			&Label{Text: "more labels"},
			&Label{Text: "another"},
			&Field{Text: "A field!!"},
			NewBox(&Image{Image: readImagePath("test.jpg")}),
			&Field{Text: "A field!!"},
			NewBox(&Image{Image: readImagePath("test.jpg")}),
			&Field{Text: "A field!!"},
			NewBox(&Image{Image: readImagePath("test.jpg")}),
		)},
		&Button{Text: "button3"},
		&Field{Text: "field 2"},
		&Field{Text: "field 3"},
		&Field{Text: "field 4"},
		&Field{Text: "field 5"},
		&Field{Text: "field 6"},
		&Field{Text: "field 7"},
		&Label{Text: "this is a label"}),
	}
	top.Layout(screen.R, image.ZP)
	top.Draw(screen, image.ZP, draw.Mouse{})
	display.Flush()

	tick := make(chan struct{}, 0)
	go func() {
		for {
			time.Sleep(1 * time.Second)
			tick <- struct{}{}
		}
	}()

	var mouse draw.Mouse
	logEvents := false
	var lastMouseUI UI
	for {
		select {
		case mouse = <-mousectl.C:
			if logEvents {
				log.Printf("mouse %v, %b\n", mouse, mouse.Buttons)
			}
			r := top.Mouse(mouse)
			if r.Hit != lastMouseUI || r.Redraw {
				top.Draw(screen, image.ZP, mouse)
				display.Flush()
			}
			lastMouseUI = r.Hit

		case <-mousectl.Resize:
			if logEvents {
				log.Printf("resize")
			}
			check(display.Attach(draw.Refmesg), "attach after resize")
			top.Layout(screen.R, image.ZP)
			top.Draw(screen, image.ZP, mouse)
			display.Flush()

		case r := <-kbdctl.C:
			if logEvents {
				log.Printf("kdb %c, %x\n", r, r)
			}
			if r == 0xf001 {
				logEvents = !logEvents
			}
			result := top.Key(image.ZP, mouse, r)
			if !result.Consumed && r == '\t' {
				first := top.FirstFocus()
				if first != nil {
					result.Warp = first
					result.Consumed = true
				}
			}
			if result.Warp != nil {
				err = display.MoveTo(*result.Warp)
				if err != nil {
					log.Printf("move mouse to %v: %v\n", result.Warp, err)
				}
				m := mouse
				m.Point = *result.Warp
				result2 := top.Mouse(m)
				result.Redraw = result.Redraw || result2.Redraw || true
				mouse = m
				lastMouseUI = result2.Hit
			}
			if result.Redraw {
				top.Draw(screen, image.ZP, mouse)
				display.Flush()
			}

		case <-redraw:
			top.Draw(screen, image.ZP, mouse)
			display.Flush()

		case <-tick:
			count++
			counter.Text = fmt.Sprintf("%d", count)
			top.Layout(screen.R, image.ZP)
			top.Draw(screen, image.ZP, mouse)
			display.Flush()
		}
	}
}

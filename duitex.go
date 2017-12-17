package main

import (
	"image"
	imagedraw "image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"

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
)

type UI interface {
	Layout(r image.Rectangle) image.Point
	Draw(img *draw.Image, orig image.Point, m draw.Mouse)
	Mouse(m draw.Mouse) UI
	Key(m draw.Mouse, k rune)
}

type Label struct {
	Text string
}

func (ui *Label) Layout(r image.Rectangle) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*Margin + 2*Border, 2 * Space})
}
func (ui *Label) Draw(img *draw.Image, orig image.Point, m draw.Mouse) {
	img.String(orig.Add(image.Point{Margin + Border, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Label) Mouse(m draw.Mouse) UI {
	return ui
}
func (ui *Label) Key(m draw.Mouse, c rune) {
}

type Field struct {
	Text string

	redraw chan struct{}
	size   image.Point // including space
}

func (ui *Field) Layout(r image.Rectangle) image.Point {
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
func (ui *Field) Mouse(m draw.Mouse) UI {
	return ui
}
func (ui *Field) Key(m draw.Mouse, c rune) {
	if c == 8 {
		if ui.Text != "" {
			ui.Text = ui.Text[:len(ui.Text)-1]
		}
	} else {
		ui.Text += string(c)
	}
	ui.redraw <- struct{}{}
}

type Button struct {
	Text  string
	Click func()

	m draw.Mouse
}

func (ui *Button) Layout(r image.Rectangle) image.Point {
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
func (ui *Button) Mouse(m draw.Mouse) UI {
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 && ui.Click != nil {
		ui.Click()
	}
	ui.m = m
	return ui
}
func (ui *Button) Key(m draw.Mouse, c rune) {
}

type Image struct {
	Image *draw.Image
}

func (ui *Image) Layout(r image.Rectangle) image.Point {
	return ui.Image.R.Size()
}
func (ui *Image) Draw(img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(image.Rectangle{orig, orig.Add(ui.Image.R.Size())}, ui.Image, nil, image.ZP)
}
func (ui *Image) Mouse(m draw.Mouse) UI {
	return ui
}
func (ui *Image) Key(m draw.Mouse, c rune) {
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

func (ui *Box) Layout(r image.Rectangle) image.Point {
	xmax := 0
	cur := image.Point{0, 0}
	nx := 0    // number on current line
	liney := 0 // max y of current line
	for _, k := range ui.Kids {
		p := k.UI.Layout(r)
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
func (ui *Box) Mouse(m draw.Mouse) UI {
	for _, k := range ui.Kids {
		if m.Point.In(k.R) {
			m.Point = m.Point.Sub(k.R.Min)
			return k.UI.Mouse(m)
		}
	}
	return nil
}
func (ui *Box) Key(m draw.Mouse, c rune) {
	for _, k := range ui.Kids {
		if m.Point.In(k.R) {
			m.Point = m.Point.Sub(k.R.Min)
			k.UI.Key(m, c)
			return
		}
	}
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
	display, err = draw.Init(nil, "", "duit-example", "600x400")
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

	var top UI = NewBox(
		&Button{Text: "button1", Click: func() { log.Printf("button clicked") }},
		&Button{Text: "button2"},
		NewBox(
			&Label{Text: "another label, this one is somewhat longer"},
			&Button{Text: "some other button"},
			&Label{Text: "more labels"},
			&Label{Text: "another"},
			&Field{Text: "A field!!", redraw: redraw},
			&Image{Image: readImagePath("test.jpg")}),
		&Button{Text: "button3"},
		&Label{Text: "this is a label"})
	top.Layout(screen.R)
	top.Draw(screen, image.ZP, draw.Mouse{})
	display.Flush()

	var mouse draw.Mouse
	logEvents := false
	var lastMouseUI UI
	for {
		select {
		case mouse = <-mousectl.C:
			if logEvents {
				log.Printf("mouse %v, %b\n", mouse, mouse.Buttons)
			}
			ui := top.Mouse(mouse)
			if ui != lastMouseUI {
				top.Draw(screen, image.ZP, mouse)
				display.Flush()
			}
			lastMouseUI = ui

		case <-mousectl.Resize:
			if logEvents {
				log.Printf("resize")
			}
			check(display.Attach(draw.Refmesg), "attach after resize")
			top.Layout(screen.R)
			top.Draw(screen, image.ZP, mouse)
			display.Flush()

		case r := <-kbdctl.C:
			if logEvents {
				log.Printf("kdb %c, %x\n", r, r)
			}
			if r == 0xf001 {
				logEvents = !logEvents
			}
			top.Key(mouse, r)

		case <-redraw:
			top.Draw(screen, image.ZP, mouse)
			display.Flush()
		}
	}
}

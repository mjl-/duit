package main

import (
	"log"
	"image"

	"9fans.net/go/draw"
)

var (
	display *draw.Display
	screen  *draw.Image
)

const (
	Margin = 10
	Padding = 10
	Border = 1
	Space = Margin + Border + Padding
)

type UI interface {
	Layout(r image.Rectangle) image.Point
	Draw(img *draw.Image, orig image.Point)
	Mouse(m draw.Mouse)
}

type Label struct {
	Text string
}
func (ui *Label) Layout(r image.Rectangle) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*Space, 2*Space})
}
func (ui *Label) Draw(img *draw.Image, orig image.Point) {
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Label) Mouse(m draw.Mouse) {
}

type Button struct {
	Text string
	Click func()

	m draw.Mouse
}
func (ui *Button) Layout(r image.Rectangle) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*Space, 2*Space})
}
func (ui *Button) Draw(img *draw.Image, orig image.Point) {
	size := display.DefaultFont.StringSize(ui.Text)
	img.Border(image.Rectangle{orig.Add(image.Point{Margin, Margin}), orig.Add(size).Add(image.Point{Margin+2*Padding+2*Border,Margin+2*Padding+2*Border})}, 1, display.Black, image.ZP)
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Button) Mouse(m draw.Mouse) {
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 && ui.Click != nil {
		ui.Click()
	}
	ui.m = m
}

type Kid struct {
	UI UI
	R image.Rectangle
}

// box keeps elements on a line as long as they fit
type Box struct {
	Kids []*Kid
}
func (ui *Box) Layout(r image.Rectangle) image.Point {
	xmax := 0
	cur := image.Point{0,0}
	nx := 0 // number on current line
	liney := 0 // max y of current line
	for _, k := range ui.Kids {
		p := k.UI.Layout(r)
		var kr image.Rectangle
		if nx == 0 || cur.X + p.X <= r.Dx() {
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
	return image.Point{xmax, cur.Y}
}
func (ui *Box) Draw(img *draw.Image, orig image.Point) {
	for _, k := range ui.Kids {
		k.UI.Draw(img, orig.Add(k.R.Min))
	}
}
func (ui *Box) Mouse(m draw.Mouse) {
	for _, k := range ui.Kids {
		if m.Point.In(k.R) {
			m.Point = m.Point.Sub(k.R.Min)
			k.UI.Mouse(m)
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
	display, err = draw.Init(nil, "", "qk-ex", "600x400")
	check(err, "draw init")
	screen = display.ScreenImage

	mousectl := display.InitMouse()
	kbdctl := display.InitKeyboard()
	//whitemask, _ := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0x7F7F7F7F)


	log.Printf("display.Image, R %v, Clipr %v\n", display.Image.R, display.Image.Clipr)
	log.Printf("display.ScreenImage, R %v, Clipr %v\n", screen.R, screen.Clipr)
	log.Printf("display.Windows, R %v, Clipr %v\n", display.Windows.R, display.Windows.Clipr)

	var top UI = NewBox(
		&Button{Text: "button1", Click: func() { log.Printf("button clicked")} },
		&Button{Text: "button2"},
		NewBox(
			&Label{Text: "another label, this one is somewhat longer"},
			&Button{Text: "some other button"},
			&Label{Text: "more labels"},
			&Label{Text: "another"}),
		&Button{Text: "button3"},
		&Label{Text: "this is a label"})
	top.Layout(screen.R)
	top.Draw(screen, image.ZP)
	display.Flush()

	var (
		// om draw.Mouse
		mouse draw.Mouse
	)
	for {
		select {
		case mouse = <-mousectl.C:
			log.Printf("mouse %v, %b\n", mouse, mouse.Buttons)
			// mouse.X mouse.Y mouse.Buttons
			// om = mouse
			top.Mouse(mouse)

		case <-mousectl.Resize:
			log.Printf("resize");
			check(display.Attach(draw.Refmesg), "attach after resize")
			top.Layout(screen.R)
			top.Draw(screen, image.ZP)
			display.Flush()

		case r := <-kbdctl.C:
			log.Printf("kdb %c, %x\n", r, r)
		}
	}
}

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
	Size(r image.Rectangle) image.Point
	Draw(img *draw.Image, orig image.Point)
}

type Label struct {
	Text string
}
func (ui *Label) Size(r image.Rectangle) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*Space, 2*Space})
}
func (ui *Label) Draw(img *draw.Image, orig image.Point) {
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}

type Button struct {
	Text string
}
func (ui *Button) Size(r image.Rectangle) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*Space, 2*Space})
}
func (ui *Button) Draw(img *draw.Image, orig image.Point) {
	size := display.DefaultFont.StringSize(ui.Text)
	img.Border(image.Rectangle{orig.Add(image.Point{Margin, Margin}), orig.Add(size).Add(image.Point{Margin+2*Padding+2*Border,Margin+2*Padding+2*Border})}, 1, display.Black, image.ZP)
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}

type Box struct {
	Kids []UI
}
func (b *Box) Size(r image.Rectangle) image.Point {
	// lay elements out one next to the other
	xmax := r.Dx()
	dx := 0
	dy := 0
	nx := 0 // number on current line
	liney := 0 // max y of current line
	for _, k := range b.Kids {
		p := k.Size(r)
		if nx == 0 || dx + p.X <= xmax {
			dx += p.X
			if p.Y > liney {
				liney = p.Y
			}
			nx += 1
		} else {
			nx = 1
			dx = p.X
			liney = p.Y
		}
	}
	return image.Point{dx, dy}
}
func (b *Box) Draw(img *draw.Image, orig image.Point) {
	// lay elements out one next to the other
	xmax := img.R.Dx()
	dx := 0
	dy := 0
	nx := 0 // number on current line
	liney := 0 // max y of current line
	for _, k := range b.Kids {
		p := k.Size(img.R)
		if nx == 0 || dx + p.X <= xmax {
			k.Draw(img, orig.Add(image.Point{dx, dy}))

			dx += p.X
			if p.Y > liney {
				liney = p.Y
			}
			nx += 1
		} else {
			dx = 0
			dy += liney
			k.Draw(img, orig.Add(image.Point{dx, dy}))

			nx = 1
			dx = p.X
			liney = p.Y
		}
	}
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func redraw(ui UI) {
	red, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, draw.Red)
	check(err, "allocimage red")

	screen.Draw(image.Rectangle{image.Point{10, 10}, image.Point{120, 50}}, red, nil, image.ZP)
	screen.String(image.Point{10, 10}, display.White, image.ZP, display.DefaultFont, "hi")
	display.Flush()
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

	var top UI = &Box{
		Kids: []UI{
			&Button{Text: "button1"},
			&Button{Text: "button2"},
			&Button{Text: "button3"},
			&Label{Text: "this is a label"},
		},
	}
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

		case <-mousectl.Resize:
			log.Printf("resize");
			check(display.Attach(draw.Refmesg), "attach after resize")
			top.Draw(screen, image.ZP)
			display.Flush()

		case r := <-kbdctl.C:
			log.Printf("kdb %c, %x\n", r, r)
		}
	}
}

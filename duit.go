package duit

import (
	"fmt"
	"image"
	"log"
	"time"

	"9fans.net/go/draw"
)

const (
	Margin  = 4
	Padding = 6
	Border  = 1

	ScrollbarSize = 8
)
const (
	Button1 = 1 << iota
	Button2
	Button3
	Button4
	Button5
)

type DUI struct {
	Display  *draw.Display
	Mousectl *draw.Mousectl
	Kbdctl   *draw.Keyboardctl
	Top      UI

	env         *Env
	mouse       draw.Mouse
	lastMouseUI UI
	logEvents   bool
	logTiming   bool
}

func check(err error, msg string) {
	if err != nil {
		log.Printf(msg)
		panic(err)
	}
}

func NewDUI(name, dim string) (*DUI, error) {
	dui := &DUI{}
	display, err := draw.Init(nil, "", name, dim)
	if err != nil {
		return nil, err
	}
	dui.Display = display

	dui.Mousectl = display.InitMouse()
	dui.Kbdctl = display.InitKeyboard()

	makeColor := func(v draw.Color) *draw.Image {
		c, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, v)
		check(err, "allocimage")
		return c
	}

	dui.env = &Env{
		Display: display,

		Normal: Colors{
			Text:       makeColor(0x333333ff),
			Background: makeColor(0xf8f8f8ff),
			Border:     makeColor(0xbbbbbbff),
		},
		Hover: Colors{
			Text:       makeColor(0x222222ff),
			Background: makeColor(0xfafafaff),
			Border:     makeColor(0x3272dcff),
		},
		Disabled: Colors{
			Text:       makeColor(0x888888ff),
			Background: makeColor(0xf0f0f0ff),
			Border:     makeColor(0xe0e0e0ff),
		},
		Inverse: Colors{
			Text:       makeColor(0xeeeeeeff),
			Background: makeColor(0x444444ff),
			Border:     makeColor(0x666666ff),
		},
		Primary: Colors{
			Text:       makeColor(0xffffffff),
			Background: makeColor(0x3272dcff),
			Border:     makeColor(0x3272dcff),
		},

		BackgroundColor: draw.Color(0xffffffff),

		ScrollBGNormal:      makeColor(0xf4f4f4ff),
		ScrollBGHover:       makeColor(0xf0f0f0ff),
		ScrollVisibleNormal: makeColor(0xbbbbbbff),
		ScrollVisibleHover:  makeColor(0x999999ff),
	}
	setSize(dui.Display, &dui.env.Size)

	return dui, nil
}

func (d *DUI) Render() {
	var t0 time.Time
	if d.logTiming {
		t0 = time.Now()
	}
	size := image.Pt(d.Display.ScreenImage.R.Dx(), d.Display.ScreenImage.R.Dy())
	d.Top.Layout(d.env, size)
	if d.logTiming {
		log.Printf("time layout: %d µs\n", time.Now().Sub(t0)/time.Microsecond)
	}
	d.Display.ScreenImage.Draw(d.Display.ScreenImage.R, d.Display.White, nil, image.ZP)
	d.Redraw()
}

func (d *DUI) Redraw() {
	var t0, t1 time.Time
	if d.logTiming {
		t0 = time.Now()
	}
	d.Top.Draw(d.env, d.Display.ScreenImage, image.ZP, d.mouse)
	if d.logTiming {
		t1 = time.Now()
	}
	d.Display.Flush()
	if d.logTiming {
		t2 := time.Now()
		log.Printf("time redraw: draw %d µs flush %d µs\n", t1.Sub(t0)/time.Microsecond, t2.Sub(t1)/time.Microsecond)
	}
}

func (d *DUI) Mouse(m draw.Mouse) {
	d.mouse = m
	if d.logEvents {
		log.Printf("mouse %v, %b\n", m, m.Buttons)
	}
	r := d.Top.Mouse(d.env, m)
	if r.Layout {
		d.Render()
	} else if r.Hit != d.lastMouseUI || r.Redraw {
		d.Redraw()
	}
	d.lastMouseUI = r.Hit
}

func (d *DUI) Resize() {
	if d.logEvents {
		log.Printf("resize")
	}
	check(d.Display.Attach(draw.Refmesg), "attach after resize")
	d.Render()
}

func (d *DUI) Key(r rune) {
	if d.logEvents {
		log.Printf("kdb %c, %x\n", r, r)
	}
	if r == draw.KeyFn+1 {
		d.logEvents = !d.logEvents
	}
	if r == draw.KeyFn+2 {
		d.logTiming = !d.logTiming
	}
	if r == draw.KeyFn+3 {
		d.Top.Print(0, d.Display.ScreenImage.R)
	}
	if r == draw.KeyFn+4 {
		d.Display.SetDebug(true)
		log.Println("drawdebug now on")
	}
	result := d.Top.Key(d.env, image.ZP, d.mouse, r)
	if !result.Consumed && r == '\t' {
		first := d.Top.FirstFocus(d.env)
		if first != nil {
			result.Warp = first
			result.Consumed = true
		}
	}
	if result.Warp != nil {
		err := d.Display.MoveTo(*result.Warp)
		if err != nil {
			log.Printf("move mouse to %v: %s\n", result.Warp, err)
		}
		d.mouse.Point = *result.Warp
		result2 := d.Top.Mouse(d.env, d.mouse)
		result.Redraw = result.Redraw || result2.Redraw || true
		result.Layout = result.Layout || result2.Layout
		d.lastMouseUI = result2.Hit
	}
	if result.Layout {
		d.Render()
	} else if result.Redraw {
		d.Redraw()
	}
}

func (d *DUI) Focus(ui UI) {
	p := d.Top.Focus(d.env, ui)
	if p == nil {
		return
	}
	err := d.Display.MoveTo(*p)
	if err != nil {
		log.Printf("move mouse to %v: %v\n", *p, err)
		return
	}
	d.mouse.Point = *p
	r := d.Top.Mouse(d.env, d.mouse)
	d.lastMouseUI = r.Hit
	if r.Layout {
		d.Render()
	} else {
		d.Redraw()
	}
}

func uiPrint(s string, indent int, r image.Rectangle) {
	indentStr := ""
	if indent > 0 {
		indentStr = fmt.Sprintf("%*s", indent*2, " ")
	}
	log.Printf("%s%s r %v\n", indentStr, s, r)
}

func scale(d *draw.Display, n int) int {
	return (d.DPI / 100) * n
}

func (d *DUI) Scale(n int) int {
	return (d.Display.DPI / 100) * n
}

func setSize(d *draw.Display, size *Size) {
	size.Padding = d.Scale(Padding)
	size.Margin = d.Scale(Margin)
	size.Border = Border // slim border is nicer
	size.Space = size.Margin + size.Border + size.Padding
}

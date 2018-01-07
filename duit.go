package duit

import (
	"fmt"
	"image"
	"io"
	"log"
	"time"

	"9fans.net/go/draw"
)

const (
	BorderSize = 1 // regardless of lowDPI/hiDPI

	ScrollbarSize = 10
)

const (
	Button1 = 1 << iota
	Button2
	Button3
	Button4
	Button5
)

type Halign int

const (
	HalignLeft = Halign(iota)
	HalignMiddle
	HalignRight
)

type Valign int

const (
	ValignMiddle = Valign(iota)
	ValignTop
	ValignBottom
)

type Result struct {
	Hit      UI           // the UI where the event ended up
	Consumed bool         // whether event was consumed, and should not be further handled by upper UI's
	Draw     bool         // whether event needs a redraw after
	Layout   bool         // whether event needs a layout after
	Warp     *image.Point // if set, mouse will warp to location
}

type Colors struct {
	Text,
	Background,
	Border *draw.Image
}

type Colorset struct {
	Normal, Hover Colors
}

type EventType byte

const (
	EventMouse = EventType(iota)
	EventKey
	EventFunc
	EventResize
	EventError
)

type Event struct {
	Type  EventType
	Mouse draw.Mouse
	Key   rune
	Func  func()
	Error error
}

type DUI struct {
	Events  chan Event
	Top     UI
	Call    chan func()   // functions sent here will go through DUI.Events and run by DUI.Event() in the main event loop. for code that changes UI state.
	Done    chan struct{} // closed when window is closed
	Display *draw.Display

	// colors
	Disabled,
	Inverse,
	Selection,
	SelectionHover,
	Placeholder,
	Striped Colors

	// colors including hover-variants
	Regular,
	Primary,
	Secondary,
	Success,
	Danger Colorset

	BackgroundColor draw.Color
	Background      *draw.Image

	CommandMode,
	VisualMode,
	ScrollBGNormal,
	ScrollBGHover,
	ScrollVisibleNormal,
	ScrollVisibleHover *draw.Image

	DebugKids   bool // whether to print distinct backgrounds in kids* functions
	debugColors []*draw.Image

	stop        chan struct{}
	mousectl    *draw.Mousectl
	keyctl      *draw.Keyboardctl
	mouse       draw.Mouse
	origMouse   draw.Mouse
	lastMouseUI UI
	logEvents   bool
	logTiming   bool
}

func check(err error, msg string) {
	if err != nil {
		log.Printf("duit: %s: %s\n", msg, err)
		panic(err)
	}
}

func NewDUI(name, dim string) (*DUI, error) {
	errch := make(chan error, 1)
	display, err := draw.Init(errch, "", name, dim)
	if err != nil {
		return nil, err
	}

	makeColor := func(v draw.Color) *draw.Image {
		c, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, v)
		check(err, "allocimage")
		return c
	}

	dui := &DUI{
		mousectl: display.InitMouse(),
		keyctl:   display.InitKeyboard(),
		stop:     make(chan struct{}, 1),
		Events:   make(chan Event, 1),
		Call:     make(chan func(), 1),
		Done:     make(chan struct{}, 1),

		Display: display,

		Disabled: Colors{
			Text:       makeColor(0x888888ff),
			Background: makeColor(0xf0f0f0ff),
			Border:     makeColor(0xe0e0e0ff),
		},
		Inverse: Colors{
			Text:       makeColor(0xeeeeeeff),
			Background: makeColor(0x3272dcff),
			Border:     makeColor(0x666666ff),
		},
		Selection: Colors{
			Text:       makeColor(0xeeeeeeff),
			Background: makeColor(0xbbbbbbff),
			Border:     makeColor(0x666666ff),
		},
		SelectionHover: Colors{
			Text:       makeColor(0xeeeeeeff),
			Background: makeColor(0x3272dcff),
			Border:     makeColor(0x666666ff),
		},
		Placeholder: Colors{
			Text:       makeColor(0xaaaaaaff),
			Background: makeColor(0xf8f8f8ff),
			Border:     makeColor(0xbbbbbbff),
		},
		Striped: Colors{
			Text:       makeColor(0x333333ff),
			Background: makeColor(0xf2f2f2ff),
			Border:     makeColor(0xbbbbbbff),
		},

		Regular: Colorset{
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
		},
		Primary: Colorset{
			Normal: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0x007bffff),
				Border:     makeColor(0x007bffff),
			},
			Hover: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0x0062ccff),
				Border:     makeColor(0x0062ccff),
			},
		},
		Secondary: Colorset{
			Normal: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0x868e96ff),
				Border:     makeColor(0x868e96ff),
			},
			Hover: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0x727b84ff),
				Border:     makeColor(0x6c757dff),
			},
		},
		Success: Colorset{
			Normal: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0x28a745ff),
				Border:     makeColor(0x28a745ff),
			},
			Hover: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0x218838ff),
				Border:     makeColor(0x1e7e34ff),
			},
		},
		Danger: Colorset{
			Normal: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0xdc3545ff),
				Border:     makeColor(0xdc3545ff),
			},
			Hover: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0xc82333ff),
				Border:     makeColor(0xbd2130ff),
			},
		},

		BackgroundColor: draw.Color(0xfcfcfcff),
		Background:      makeColor(0xfcfcfcff),

		CommandMode: makeColor(0x3272dcff),
		VisualMode:  makeColor(0x5cb85cff),

		ScrollBGNormal:      makeColor(0xf4f4f4ff),
		ScrollBGHover:       makeColor(0xf0f0f0ff),
		ScrollVisibleNormal: makeColor(0xbbbbbbff),
		ScrollVisibleHover:  makeColor(0x999999ff),

		debugColors: []*draw.Image{
			makeColor(0x40000040),
			makeColor(0x00400040),
			makeColor(0x00004040),
		},
	}

	go func() {
		for {
			select {
			case m := <-dui.mousectl.C:
				dui.Events <- Event{Type: EventMouse, Mouse: m}
			case k := <-dui.keyctl.C:
				dui.Events <- Event{Type: EventKey, Key: k}
			case <-dui.mousectl.Resize:
				dui.Events <- Event{Type: EventResize}
			case fn := <-dui.Call:
				dui.Events <- Event{Type: EventFunc, Func: fn}
			case <-dui.stop:
				return
			case e := <-errch:
				if e == io.EOF {
					// devdraw disappeared, typically because window was closed (either by user, or by duit)
					close(dui.Done)
					return
				} else {
					dui.Events <- Event{Type: EventError, Error: e}
				}
			}
		}
	}()

	return dui, nil
}

func (d *DUI) Render() {
	var t0 time.Time
	if d.logTiming {
		t0 = time.Now()
	}
	size := image.Pt(d.Display.ScreenImage.R.Dx(), d.Display.ScreenImage.R.Dy())
	d.Top.Layout(d, size)
	if d.logTiming {
		log.Printf("duit: time layout: %d µs\n", time.Now().Sub(t0)/time.Microsecond)
	}
	d.Display.ScreenImage.Draw(d.Display.ScreenImage.R, d.Display.White, nil, image.ZP)
	d.Draw()
}

func (d *DUI) Draw() {
	var t0, t1 time.Time
	if d.logTiming {
		t0 = time.Now()
	}
	d.Display.ScreenImage.Draw(d.Display.ScreenImage.R, d.Background, nil, image.ZP)
	d.Top.Draw(d, d.Display.ScreenImage, image.ZP, d.mouse)
	if d.logTiming {
		t1 = time.Now()
	}
	d.Display.Flush()
	if d.logTiming {
		t2 := time.Now()
		log.Printf("duit: time draw: draw %d µs flush %d µs\n", t1.Sub(t0)/time.Microsecond, t2.Sub(t1)/time.Microsecond)
	}
}

func (d *DUI) Mouse(m draw.Mouse) {
	if m.Buttons == 0 || d.origMouse.Buttons == 0 {
		d.origMouse = m
	}
	d.mouse = m
	if d.logEvents {
		log.Printf("duit: mouse %v, %b\n", m, m.Buttons)
	}
	r := d.Top.Mouse(d, m, d.origMouse)
	if r.Layout {
		d.Render()
	} else if r.Hit != d.lastMouseUI || r.Draw {
		d.Draw()
	}
	d.lastMouseUI = r.Hit
}

func (d *DUI) Resize() {
	if d.logEvents {
		log.Printf("duit: resize")
	}
	check(d.Display.Attach(draw.Refmesg), "attach after resize")
	d.Render()
}

func (d *DUI) Key(k rune) {
	if d.logEvents {
		log.Printf("duit: key %c, %x\n", k, k)
	}
	layout := false
	if k == draw.KeyFn+1 {
		d.logEvents = !d.logEvents
	}
	if k == draw.KeyFn+2 {
		d.logTiming = !d.logTiming
	}
	if k == draw.KeyFn+3 {
		d.Top.Print(0, d.Display.ScreenImage.R)
	}
	if k == draw.KeyFn+4 {
		d.Display.SetDebug(true)
		log.Println("duit: drawdebug now on")
	}
	if k == draw.KeyFn+5 {
		d.DebugKids = !d.DebugKids
		log.Println("duit: debugKids now", d.DebugKids)
		layout = true
	}
	r := d.Top.Key(d, k, d.mouse, image.ZP)
	if !r.Consumed {
		switch k {
		case '\t':
			first := d.Top.FirstFocus(d)
			if first != nil {
				r.Warp = first
				r.Consumed = true
			}
		case draw.KeyCmd + 'w':
			d.Close()
			d.Done <- struct{}{}
			return
		}
	}
	if r.Warp != nil {
		err := d.Display.MoveTo(*r.Warp)
		if err != nil {
			log.Printf("duit: move mouse to %v: %s\n", r.Warp, err)
		}
		d.mouse.Point = *r.Warp
		d.origMouse.Point = *r.Warp
		r2 := d.Top.Mouse(d, d.mouse, d.origMouse)
		r.Draw = r.Draw || r2.Draw || true
		r.Layout = r.Layout || r2.Layout
		d.lastMouseUI = r2.Hit
	}
	if r.Layout || layout {
		d.Render()
	} else if r.Draw {
		d.Draw()
	}
}

func (d *DUI) Focus(ui UI) {
	p := d.Top.Focus(d, ui)
	if p == nil {
		return
	}
	err := d.Display.MoveTo(*p)
	if err != nil {
		log.Printf("duit: move mouse to %v: %v\n", *p, err)
		return
	}
	d.mouse.Point = *p
	d.origMouse.Point = *p
	r := d.Top.Mouse(d, d.mouse, d.origMouse)
	d.lastMouseUI = r.Hit
	if r.Layout {
		d.Render()
	} else {
		d.Draw()
	}
}

func PrintUI(s string, indent int, r image.Rectangle) {
	indentStr := ""
	if indent > 0 {
		indentStr = fmt.Sprintf("%*s", indent*2, " ")
	}
	log.Printf("duit: %s%s r %v\n", indentStr, s, r)
}

func scale(d *draw.Display, n int) int {
	return (d.DPI / 100) * n
}

func scalePt(d *draw.Display, p image.Point) image.Point {
	return p.Mul(d.DPI / 100)
}

func (d *DUI) Scale(n int) int {
	return (d.Display.DPI / 100) * n
}

func (d *DUI) Event(e Event) {
	switch e.Type {
	case EventMouse:
		d.Mouse(e.Mouse)
	case EventKey:
		d.Key(e.Key)
	case EventResize:
		d.Resize()
	case EventFunc:
		e.Func()
	case EventError:
		log.Fatalf("error from devdraw: %s\n", e.Error)
	}
}

func (d *DUI) Close() {
	d.stop <- struct{}{}
	d.Display.Close()
}

func (d *DUI) ScaleSpace(s Space) Space {
	return Space{
		d.Scale(s.Top),
		d.Scale(s.Right),
		d.Scale(s.Bottom),
		d.Scale(s.Left),
	}
}

func (d *DUI) Font(font *draw.Font) *draw.Font {
	if font != nil {
		return font
	}
	return d.Display.DefaultFont
}

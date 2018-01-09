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

type Event struct {
	Consumed   bool // whether event was consumed, and should not be further handled by upper UI's
	NeedLayout bool // whether UI now needs a layout
	NeedDraw   bool // whether UI now needs a draw
}

type Result struct {
	Hit      UI           // the UI where the event ended up
	Consumed bool         // whether event was consumed, and should not be further handled by upper UI's
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

type InputType byte

const (
	InputMouse = InputType(iota)
	InputKey
	InputFunc
	InputResize
	InputError
)

type Input struct {
	Type  InputType
	Mouse draw.Mouse
	Key   rune
	Func  func()
	Error error
}

type State byte

const (
	Dirty    = State(iota) // UI itself needs layout/draw;  kids will also get a layout/draw call, with force set.
	DirtyKid               // UI itself does not need layout/draw, but one of its children does, so pass the call on.
	Clean                  // UI does not need layout/draw.

	// order is important, Clean is highest and means least amount of work
)

type DUI struct {
	Inputs  chan Input
	Top     Kid
	Call    chan func()   // functions sent here will go through DUI.Inputs and run by DUI.Input() in the main event loop. for code that changes UI state.
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

	ScrollBGNormal,
	ScrollBGHover,
	ScrollVisibleNormal,
	ScrollVisibleHover *draw.Image

	DebugDraw   int  // if 1, UIs print each draw they do, if 2, UIs print all calls to their Draw function. Cycle through 0-2 with F7
	DebugLayout int  // if 1, UIs print each Layout they do, if 2, UIs print all calls to their Layout function. Cycle through 0-2 with F8
	DebugKids   bool // whether to print distinct backgrounds in kids* functions
	debugColors []*draw.Image

	// for edit.  we might need a map where other UIs can store images (like colors) for caching purposes in the future...
	commandMode,
	visualMode *draw.Image

	stop        chan struct{}
	mousectl    *draw.Mousectl
	keyctl      *draw.Keyboardctl
	mouse       draw.Mouse
	origMouse   draw.Mouse
	lastMouseUI UI
	logInputs   bool
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
		Inputs:   make(chan Input, 1),
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

		ScrollBGNormal:      makeColor(0xf4f4f4ff),
		ScrollBGHover:       makeColor(0xf0f0f0ff),
		ScrollVisibleNormal: makeColor(0xbbbbbbff),
		ScrollVisibleHover:  makeColor(0x999999ff),

		commandMode: makeColor(0x3272dcff),
		visualMode:  makeColor(0x5cb85cff),

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
				dui.Inputs <- Input{Type: InputMouse, Mouse: m}
			case k := <-dui.keyctl.C:
				dui.Inputs <- Input{Type: InputKey, Key: k}
			case <-dui.mousectl.Resize:
				dui.Inputs <- Input{Type: InputResize}
			case fn := <-dui.Call:
				dui.Inputs <- Input{Type: InputFunc, Func: fn}
			case <-dui.stop:
				return
			case e := <-errch:
				if e == io.EOF {
					// devdraw disappeared, typically because window was closed (either by user, or by duit)
					close(dui.Done)
					return
				} else {
					dui.Inputs <- Input{Type: InputError, Error: e}
				}
			}
		}
	}()

	return dui, nil
}

// Render calls Layout followed by Draw.
func (d *DUI) Render() {
	d.Layout()
	d.Draw()
}

func (d *DUI) Layout() {
	if d.Top.Layout == Clean {
		return
	}
	var t0 time.Time
	if d.logTiming {
		t0 = time.Now()
	}
	d.Top.UI.Layout(d, &d.Top, d.Display.ScreenImage.R.Size(), d.Top.Layout == Dirty)
	d.Top.Layout = Clean
	if d.logTiming {
		log.Printf("duit: time layout: %d µs\n", time.Now().Sub(t0)/time.Microsecond)
	}
}

func (d *DUI) Draw() {
	if d.Top.Draw == Clean {
		return
	}
	var t0, t1 time.Time
	if d.logTiming {
		t0 = time.Now()
	}
	if d.Top.Draw == Dirty {
		d.Display.ScreenImage.Draw(d.Display.ScreenImage.R, d.Background, nil, image.ZP)
	}
	d.Top.UI.Draw(d, &d.Top, d.Display.ScreenImage, image.ZP, d.mouse, d.Top.Draw == Dirty)
	d.Top.Draw = Clean
	if d.logTiming {
		t1 = time.Now()
	}
	d.Display.Flush()
	if d.logTiming {
		t2 := time.Now()
		log.Printf("duit: time draw: draw %d µs flush %d µs\n", t1.Sub(t0)/time.Microsecond, t2.Sub(t1)/time.Microsecond)
	}
}

func (d *DUI) apply(r Result) {
	if r.Warp != nil {
		err := d.Display.MoveTo(*r.Warp)
		if err != nil {
			log.Printf("duit: warp to %v: %s\n", r.Warp, err)
		} else {
			d.mouse.Point = *r.Warp
			d.origMouse.Point = *r.Warp
			r = d.Top.UI.Mouse(d, &d.Top, d.mouse, d.origMouse, image.ZP)
			d.lastMouseUI = r.Hit
		}
	} else {
		if r.Hit != d.lastMouseUI {
			d.Mark(d.lastMouseUI, false, Dirty)
		}
		d.lastMouseUI = r.Hit
	}

	d.Render()
}

func (d *DUI) Mouse(m draw.Mouse) {
	if d.logInputs {
		log.Printf("duit: mouse %v, %b\n", m, m.Buttons)
	}
	if m.Buttons == 0 || d.origMouse.Buttons == 0 {
		d.origMouse = m
	}
	d.mouse = m
	r := d.Top.UI.Mouse(d, &d.Top, m, d.origMouse, image.ZP)
	d.apply(r)
}

func (d *DUI) Resize() {
	if d.logInputs {
		log.Printf("duit: resize")
	}
	check(d.Display.Attach(draw.Refmesg), "attach after resize")
	d.Top.Layout = Dirty
	d.Top.Draw = Dirty
	d.Render()
}

func (d *DUI) Key(k rune) {
	switch k {
	case draw.KeyFn + 1:
		d.logInputs = !d.logInputs
		log.Printf("duit: logInputs now %v\n", d.logInputs)
		return
	case draw.KeyFn + 2:
		d.logTiming = !d.logTiming
		log.Printf("duit: logTiming now %v\n", d.logTiming)
		return
	case draw.KeyFn + 3:
		d.Top.UI.Print(&d.Top, 0)
		return
	case draw.KeyFn + 4:
		d.Display.SetDebug(true)
		log.Println("duit: drawdebug now on")
		return
	case draw.KeyFn + 5:
		d.DebugKids = !d.DebugKids
		log.Println("duit: debugKids now", d.DebugKids)
		return
	case draw.KeyFn + 6:
		log.Println("duit: rendering entire ui")
		d.Top.Layout = Dirty
		d.Top.Draw = Dirty
		d.Render()
		return
	case draw.KeyFn + 7:
		d.DebugDraw = (d.DebugDraw + 1) % 3
		log.Printf("duit: DebugDraw now %d", d.DebugDraw)
		return
	case draw.KeyFn + 8:
		d.DebugLayout = (d.DebugLayout + 1) % 3
		log.Printf("duit: DebugLayout now %d", d.DebugLayout)
		return
	}
	if d.logInputs {
		log.Printf("duit: key %c, %x\n", k, k)
	}
	r := d.Top.UI.Key(d, &d.Top, k, d.mouse, image.ZP)
	if !r.Consumed {
		switch k {
		case '\t':
			first := d.Top.UI.FirstFocus(d)
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
	d.apply(r)
}

func (d *DUI) Focus(ui UI) {
	p := d.Top.UI.Focus(d, ui)
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
	r := d.Top.UI.Mouse(d, &d.Top, d.mouse, d.origMouse, image.ZP)
	d.apply(r)
}

func (d *DUI) debugLayout(what string, self *Kid) {
	if d.DebugLayout > 0 {
		log.Printf("duit: Layout %s %s layout=%d draw=%d\n", what, self.R, self.Layout, self.Draw)
	}
}

func (d *DUI) debugDraw(what string, self *Kid) {
	if d.DebugDraw > 0 {
		log.Printf("duit: Draw %s %s layout=%d draw=%d\n", what, self.R, self.Layout, self.Draw)
	}
}

func PrintUI(s string, self *Kid, indent int) {
	indentStr := ""
	if indent > 0 {
		indentStr = fmt.Sprintf("%*s", indent*2, " ")
	}
	log.Printf("duit: %s%s r %v layout=%d draw=%d\n", indentStr, s, self.R, self.Layout, self.Draw)
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

func (d *DUI) Input(e Input) {
	switch e.Type {
	case InputMouse:
		d.Mouse(e.Mouse)
	case InputKey:
		d.Key(e.Key)
	case InputResize:
		d.Resize()
	case InputFunc:
		e.Func()
	case InputError:
		log.Fatalf("error from devdraw: %s\n", e.Error)
	}
}

func (d *DUI) Mark(o UI, forLayout bool, state State) {
	d.Top.UI.Mark(&d.Top, o, forLayout, state)
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

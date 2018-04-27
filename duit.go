package duit

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"9fans.net/go/draw"
)

const (
	BorderSize = 1 // regardless of lowDPI/hiDPI, won't be scaled

	ScrollbarSize = 10 // in lowDPI pixels
)

// Mouse buttons, see draw.Mouse.Buttons.
const (
	Button1 = 1 << iota
	Button2
	Button3
	Button4 // wheel up
	Button5 // wheel down
)

// Halign represents horizontal align of elements in a Grid.
type Halign byte

const (
	HalignLeft Halign = iota // Align to the left by default, for example in a grid.
	HalignMiddle
	HalignRight
)

// Valign represents vertical align of elements in a Grid, or in a Box.
type Valign byte

const (
	ValignMiddle Valign = iota // Align vertically in the middle by default, for example in a box (line) or grid.
	ValignTop
	ValignBottom
)

// Event is returned by handlers, such as click or key handlers.
type Event struct {
	Consumed   bool // Whether event was consumed, and should not be further handled by upper UI's.  Container UIs can handle some mouse/key events and decide whether they want to pass them on, or first pass them on and only consume them when a child UI hasn't done so yet.
	NeedLayout bool // Whether UI now needs a layout. Only the UI generating the event will be marked. If you another UI needs to be marked, call MarkLayout.
	NeedDraw   bool // Like NeedLayout, but for draw.
}

// Result holds the effects of a mouse/key event, as implement by UIs.
type Result struct {
	Hit      UI           // the UI where the event ended up
	Consumed bool         // whether event was consumed, and should not be further handled by upper UI's
	Warp     *image.Point // if set, mouse will warp to location
}

// Colors represents the style in one state of the UI.
type Colors struct {
	Text       *draw.Image `json:"-"`
	Background *draw.Image `json:"-"`
	Border     *draw.Image `json:"-"`
}

// Colorset is typically used to style buttons. Duit provides some builtin colorsets like Primary, Danger, Success.
type Colorset struct {
	Normal, Hover Colors
}

// InputType presents the type of an input event.
type InputType byte

const (
	InputMouse  InputType = iota // Mouse movement and/or button changes.
	InputKey                     // Key typed.
	InputFunc                    // Call the function.
	InputResize                  // window was resized, reattach; does not have/need a field in Input.
	InputError                   // An error occurred that may be recovered from.
)

// Input is an input event that is typically passed into DUI through Input().
type Input struct {
	Type  InputType
	Mouse draw.Mouse
	Key   rune
	Func  func()
	Error error
}

// State represents the layout/draw state of the UI of a Kid.
type State byte

const (
	Dirty    = State(iota) // UI itself needs layout/draw;  kids will also get a UI.Layout/UI.Draw call, with force set.
	DirtyKid               // UI itself does not need layout/draw, but one of its children does, so pass the call on.
	Clean                  // UI and its children do not need layout/draw.

	// note: order is important, Dirty is the default, Clean is highest and means least amount of work
)

// DUI represents a window and all UI state for that window.
type DUI struct {
	Inputs  chan Input  // Duit sends input events on this channel, needs to be read from the main loop.
	Top     Kid         // Root of the UI hierarchy. Wrapped in a Kid for state management.
	Call    chan func() // Functions sent here will go through DUI.Inputs and run by DUI.Input() in the main event loop. For code that changes UI state.
	Error   chan error  // Receives errors from UIs. For example when memory for an image could not be allocated. Closed when window is closed. Needs to be read from the main loop.
	Display *draw.Display

	// Colors.
	Disabled,
	Inverse,
	Selection,
	SelectionHover,
	Placeholder,
	Striped Colors

	// Colors with hover-variants.
	Regular,
	Primary,
	Secondary,
	Success,
	Danger Colorset

	// Background color UIs should have.
	BackgroundColor draw.Color
	Background      *draw.Image

	// Scrollbar colors.
	ScrollBGNormal,
	ScrollBGHover,
	ScrollVisibleNormal,
	ScrollVisibleHover *draw.Image

	// Gutter color.
	Gutter *draw.Image

	Debug       bool          // Log errors interesting to developers.
	DebugDraw   int           // If 1, UIs print each draw they do. If 2, UIs print all calls to their Draw function. Cycle through 0-2 with F7.
	DebugLayout int           // If 1, UIs print each Layout they do. If 2, UIs print all calls to their Layout function. Cycle through 0-2 with F8.
	DebugKids   bool          // Whether to print distinct backgrounds in kids* functions.
	debugColors []*draw.Image // colors used for DebugKids

	// Border colors for vi modes for Edit.
	CommandMode,
	VisualMode *draw.Image

	//  we might need a map where other UIs can store images (like colors) for caching purposes in the future...

	stop                    chan struct{}
	mousectl                *draw.Mousectl
	keyctl                  *draw.Keyboardctl
	mouse                   draw.Mouse             // Latest mouse event.
	origMouse               draw.Mouse             // Mouse that determines where new mouse events are delivered. Unchanged while button is pressed.
	lastMouseUI             UI                     // Where last mouse was delivered
	logInputs               bool                   // Print all input events. Toggled with F1.
	logTiming               bool                   // Print timings for layout and draw.
	drawDebug               bool                   // For draw.Display.SetDebug.
	dimensionsPath          string                 // For remembering windows dimensions.
	dimensionsDelayedWriter *time.Timer            // Delayed writes of dimensions.
	name                    string                 // Program name, also used for storing dimensions file.
	settings                map[string][]byte      // Indexed by Kid.ID, holds JSON. Helps store per-UI state, such as Split sizes.
	settingsWriters         map[string]*time.Timer // Delayed writes of settings.
}

// DUIOpts exist mostly to make it easier to add changes in the future, and keep the NewDUI function signature sane.

// DUIOpts are options for creating a new DUI.
// Zero values have sane behaviour.
type DUIOpts struct {
	FontName   string // eg "/mnt/font/Lato-Regular/15a/font"
	Dimensions string // eg "800x600", duit has a sane default and remembers size per application name after resize.
}

// AppdataDir returns the directory where the application can store its files, like configuration.
// On unix this is $HOME/lib/<app>. On Windows it is $APPDATA/<app>.
func AppDataDir(app string) string {
	appdata := os.Getenv("APPDATA") // windows, but more helpful than just homedir
	if appdata == "" {
		home := os.Getenv("HOME") // unix
		if home == "" {
			home = os.Getenv("home") // plan 9
		}
		appdata = home + "/lib"
	}
	return appdata + "/" + app
}

func configDir() string {
	return AppDataDir("duit")
}

// NewDUI creates a DUI for an application called name, and optional opts. A DUI is a new window and its UI state.
// Window dimensions and UI settings are automatically written to $APPDATA/duit/<name>, with $APPDATA being $HOME/lib on unix.
func NewDUI(name string, opts *DUIOpts) (dui *DUI, err error) {
	lcheck, handle := errorHandler(func(xerr error) {
		err = xerr
	})
	defer handle()

	if opts == nil {
		opts = &DUIOpts{}
	}
	if opts.Dimensions == "" {
		opts.Dimensions = "800x600"
	}

	var dimensionsPath string
	if name != "" {
		dimensionsPath = fmt.Sprintf("%s/%s/dimensions", configDir(), name)
		buf, err := ioutil.ReadFile(dimensionsPath)
		if err != nil {
			os.MkdirAll(path.Dir(dimensionsPath), os.ModePerm)
			ioutil.WriteFile(dimensionsPath, []byte(opts.Dimensions), os.ModePerm)
		} else {
			opts.Dimensions = strings.TrimSpace(string(buf))
		}
	}

	errch := make(chan error, 1)
	display, err := draw.Init(errch, opts.FontName, name, opts.Dimensions)
	if err != nil {
		return nil, err
	}

	makeColor := func(v draw.Color) *draw.Image {
		c, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, v)
		lcheck(err, "allocimage")
		return c
	}

	dui = &DUI{
		mousectl: display.InitMouse(),
		keyctl:   display.InitKeyboard(),
		stop:     make(chan struct{}, 1),
		Inputs:   make(chan Input, 1),
		Call:     make(chan func(), 1),
		Error:    make(chan error, 1),

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

		Gutter: makeColor(0xbbbbbbff),

		CommandMode: makeColor(0x3272dcff),
		VisualMode:  makeColor(0x5cb85cff),

		debugColors: []*draw.Image{
			makeColor(0x40000040),
			makeColor(0x00400040),
			makeColor(0x00004040),
		},

		dimensionsPath:  dimensionsPath,
		name:            name,
		settings:        map[string][]byte{},
		settingsWriters: map[string]*time.Timer{},

		Debug: true,
	}

	// mousectl sends initial mouse position
	dui.mouse = <-dui.mousectl.C

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
					close(dui.Error)
					return
				}
				dui.Inputs <- Input{Type: InputError, Error: e}
			}
		}
	}()

	return dui, nil
}

// Render calls Layout followed by Draw.
// This only does a layout/draw for UIs marked as needing it. If you want to force a layout/draw, mark the top UI as requiring a layout/draw.
func (d *DUI) Render() {
	d.Layout()
	d.Draw()
}

// Layout the entire UI tree, as necessary.
// Only UIs marked as requiring a layout are actually layed out.
// UIs that receive a layout are marked as requiring a draw.
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

// Draw the entire UI tree, as necessary.
// Only UIs marked as requiring a draw are actually drawn, and their children.
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

// MarkLayout marks ui as requiring a layout.
// If you have access to the Kid that holds this UI, it is more efficient to change the Kid itself. MarkLayout is more convenient. Using it can cut down on bookkeeping.
// If ui is nil, the top UI is marked.
func (d *DUI) MarkLayout(ui UI) {
	if ui == nil {
		d.Top.Layout = Dirty
	} else {
		if !d.Top.UI.Mark(&d.Top, ui, true) {
			log.Printf("duit: marklayout %T: nothing marked\n", ui)
		}
	}
}

// MarkDraw is like MarkLayout, but marks ui as requiring a draw.
func (d *DUI) MarkDraw(ui UI) {
	if ui == nil {
		d.Top.Draw = Dirty
	} else {
		if !d.Top.UI.Mark(&d.Top, ui, false) {
			log.Printf("duit: markdraw %T: nothing marked\n", ui)
		}
	}
}

func (d *DUI) apply(r Result) {
	if r.Warp != nil {
		err := d.Display.MoveTo(*r.Warp)
		if err != nil {
			log.Printf("duit: warp to %v: %s\n", r.Warp, err)
		} else {
			d.mouse.Point = *r.Warp
			d.mouse.Buttons = 0
			d.origMouse = d.mouse
			r = d.Top.UI.Mouse(d, &d.Top, d.mouse, d.origMouse, image.ZP)
		}
	}
	if r.Hit != d.lastMouseUI {
		if r.Hit != nil {
			d.MarkDraw(r.Hit)
		}
		if d.lastMouseUI != nil {
			d.MarkDraw(d.lastMouseUI)
		}
	}
	d.lastMouseUI = r.Hit

	d.Render()
}

// Mouse delivers a mouse event to the UI tree.
// Mouse is typically called by Input.
func (d *DUI) Mouse(m draw.Mouse) {
	if m.Buttons == 0 || d.origMouse.Buttons == 0 {
		d.origMouse = m
	}
	d.mouse = m
	r := d.Top.UI.Mouse(d, &d.Top, m, d.origMouse, image.ZP)
	d.apply(r)
}

// Resize handles a resize of the window. Resize is called automatically through Input when the user resizes a window.
func (d *DUI) Resize() {
	err := d.Display.Attach(draw.Refmesg)
	if d.error(err, "attach after resize") {
		return
	}

	d.Top.Layout = Dirty
	d.Top.Draw = Dirty
	d.Render()
	if d.dimensionsPath != "" {
		if d.dimensionsDelayedWriter != nil {
			d.dimensionsDelayedWriter.Stop()
		}
		size := d.Display.ScreenImage.R.Size()
		d.dimensionsDelayedWriter = time.AfterFunc(2*time.Second, func() {
			x := size.X / d.Scale(1)
			y := size.Y / d.Scale(1)
			err := ioutil.WriteFile(d.dimensionsPath, []byte(fmt.Sprintf("%dx%d", x, y)), os.ModePerm)
			if d.Debug && err != nil {
				log.Printf("duit: write dimensions: %s\n", err)
			}
		})
	}
}

// Key delivers a key press event to the UI tree.
// Key is typically called by Input.
func (d *DUI) Key(k rune) {
	switch k {
	case draw.KeyFn + 1:
		d.logInputs = !d.logInputs
		log.Println("duit: logInputs now", d.logInputs)
		return
	case draw.KeyFn + 2:
		d.logTiming = !d.logTiming
		log.Println("duit: logTiming now", d.logTiming)
		return
	case draw.KeyFn + 3:
		d.Top.UI.Print(&d.Top, 0)
		return
	case draw.KeyFn + 4:
		d.drawDebug = !d.drawDebug
		d.Display.SetDebug(d.drawDebug)
		log.Println("duit: drawDebug now", d.drawDebug)
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
		log.Println("duit: DebugDraw now", d.DebugDraw)
		return
	case draw.KeyFn + 8:
		d.DebugLayout = (d.DebugLayout + 1) % 3
		log.Println("duit: DebugLayout now", d.DebugLayout)
		return
	case draw.KeyFn + 9:
		err := json.NewEncoder(os.Stderr).Encode(&d.Top)
		if err != nil {
			log.Printf("encoding d.Top: %s\n", err)
		}
		return
	}
	r := d.Top.UI.Key(d, &d.Top, k, d.mouse, image.ZP)
	if !r.Consumed {
		switch k {
		case '\t':
			first := d.Top.UI.FirstFocus(d, &d.Top)
			if first != nil {
				r.Warp = first
				r.Consumed = true
			}
		case draw.KeyCmd + 'w':
			close(d.Error)
			d.Close()
			return
		}
	}
	d.apply(r)
}

// Focus renders the UI, then changes focus to ui by warping the mouse pointer to it.
// Container UIs ensure the UI is in place, e.g. scrolling if necessary.
func (d *DUI) Focus(ui UI) {
	d.Render()
	p := d.Top.UI.Focus(d, &d.Top, ui)
	if p == nil {
		log.Printf("duit: focus: no ui found for %T %p\n", ui, ui)
		return
	}
	err := d.Display.MoveTo(*p)
	if err != nil {
		log.Printf("duit: move mouse to %v: %v\n", *p, err)
		return
	}
	d.mouse.Point = *p
	d.mouse.Buttons = 0
	d.origMouse = d.mouse
	r := d.Top.UI.Mouse(d, &d.Top, d.mouse, d.origMouse, image.ZP)
	d.apply(r)
}

func (d *DUI) debugLayout(self *Kid) {
	if d.DebugLayout > 0 {
		log.Printf("duit: Layout %T %s layout=%d draw=%d\n", self.UI, self.R, self.Layout, self.Draw)
	}
}

func (d *DUI) debugDraw(self *Kid) {
	if d.DebugDraw > 0 {
		log.Printf("duit: Draw %T %s layout=%d draw=%d\n", self.UI, self.R, self.Layout, self.Draw)
	}
}

func (d *DUI) error(err error, msg string) bool {
	if err == nil {
		return false
	}
	go func() {
		d.Error <- fmt.Errorf("%s: %s", msg, err)
	}()
	return true
}

// PrintUI is a helper function UIs can use to implement UI.Print. "s" is typically the ui type, possibly with additional properties. Indent should be increased for each child UI that is printed.
func PrintUI(s string, self *Kid, indent int) {
	indentStr := ""
	if indent > 0 {
		indentStr = fmt.Sprintf("%*s", indent*2, " ")
	}
	var id string
	if self.ID != "" {
		id = " " + self.ID
	}
	log.Printf("duit: %s%s r %v size %s layout=%d draw=%d%s %p\n", indentStr, s, self.R, self.R.Size(), self.Layout, self.Draw, id, self.UI)
}

func scalePt(d *draw.Display, p image.Point) image.Point {
	f := d.DPI / 100
	if f <= 1 {
		f = 1
	}
	return p.Mul(f)
}

// Scale turns a low DPI pixel size into a size scaled for the current display.
func (d *DUI) Scale(n int) int {
	if d.Display.DPI <= draw.DefaultDPI {
		return n
	}
	return (d.Display.DPI / 100) * n
}

// Input propagates the input event through the UI tree.
// Mouse and key events are delivered the right UIs.
// Resize is handled by reattaching to devdraw and doing a layout and draw.
// Func calls the function.
// Error implies an error from devdraw and terminates the program.
func (d *DUI) Input(e Input) {
	switch e.Type {
	case InputMouse:
		if d.logInputs {
			log.Printf("duit: mouse %v, %b\n", e.Mouse, e.Mouse.Buttons)
		}
		d.Mouse(e.Mouse)
	case InputKey:
		if d.logInputs {
			log.Printf("duit: key %c, %x\n", e.Key, e.Key)
		}
		d.Key(e.Key)
	case InputResize:
		if d.logInputs {
			log.Printf("duit: resize")
		}
		d.Resize()
	case InputFunc:
		if d.logInputs {
			log.Printf("duit: func")
		}
		e.Func()
		d.Render()
	case InputError:
		if d.logInputs {
			log.Printf("duit: error: %s", e.Error)
		}
		log.Fatalf("error from devdraw: %s\n", e.Error)
	}
}

// Close stops mouse/keyboard event reading and closes the window.
// After closing a DUI you should no longer call functions on it.
func (d *DUI) Close() {
	d.stop <- struct{}{}
	d.Display.Close()
}

// ScaleSpace is like Scale, but for a Space.
func (d *DUI) ScaleSpace(s Space) Space {
	return Space{
		d.Scale(s.Top),
		d.Scale(s.Right),
		d.Scale(s.Bottom),
		d.Scale(s.Left),
	}
}

// Font is a helper function for UI implementations. It returns the passed font. Unless font is nil, then it returns the default font.
func (d *DUI) Font(font *draw.Font) *draw.Font {
	if font != nil {
		return font
	}
	return d.Display.DefaultFont
}

// WriteSnarf writes the snarf buffer and logs an error in case of failure.
func (d *DUI) WriteSnarf(buf []byte) {
	err := d.Display.WriteSnarf(buf)
	if err != nil {
		log.Printf("duit: writesnarf: %s\n", err)
	}
}

// ReadSnarf reads the entire snarf buffer and logs an error in case of failure.
func (d *DUI) ReadSnarf() (buf []byte, success bool) {
	buf = make([]byte, 128)
	have, total, err := d.Display.ReadSnarf(buf)
	if err != nil {
		log.Printf("duit: readsnarf: %s\n", err)
		return nil, false
	}
	if have >= total {
		return buf[:have], true
	}
	buf = make([]byte, total)
	have, _, err = d.Display.ReadSnarf(buf)
	if err != nil {
		log.Printf("duit: readsnarf entire buffer: %s\n", err)
		return nil, false
	}
	return buf[:have], true
}

func (d *DUI) settingsPath(self *Kid) string {
	return fmt.Sprintf("%s/%s/%s.json", configDir(), d.name, self.ID)
}

// ReadSettings reads the settings for self.ID if any into v.
// Settings are stored as JSON, (un)marshalled with encoding/json.
// ReadSettings returns whether reading settings was successful.
func (d *DUI) ReadSettings(self *Kid, v interface{}) bool {
	if self.ID == "" {
		return false
	}
	if buf, ok := d.settings[self.ID]; ok {
		return json.Unmarshal(buf, v) == nil
	}
	buf, err := ioutil.ReadFile(d.settingsPath(self))
	if err != nil {
		d.settings[self.ID] = nil
		return false
	}
	d.settings[self.ID] = buf
	return json.Unmarshal(buf, v) == nil
}

// WriteSettings writes settings v for self.ID as JSON.
// WriteSettings delays a write for an ID for 2 seconds. Delayed writes are canceled by new writes.
func (d *DUI) WriteSettings(self *Kid, v interface{}) bool {
	if self.ID == "" {
		return false
	}
	buf, err := json.Marshal(v)
	if err != nil {
		return false
	}
	d.settings[self.ID] = buf
	w := d.settingsWriters[self.ID]
	if w != nil {
		w.Stop()
	}
	d.settingsWriters[self.ID] = time.AfterFunc(2*time.Second, func() {
		ioutil.WriteFile(d.settingsPath(self), buf, os.ModePerm)
	})
	return true
}

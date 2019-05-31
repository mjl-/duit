package draw

import (
	"encoding/binary"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// Display locking:
// The Exported methods of Display, being entry points for clients, lock the Display structure.
// The unexported ones do not.
// The methods for Font, Image and Screen also lock the associated display by the same rules.

// drawFile exists solely to provide a ReadDraw, which makes Display.Conn compatible with a call in unloadimage, used by the devdraw version.
type drawFile struct {
	*os.File
}

func (f *drawFile) ReadDraw(buf []byte) (int, error) {
	return f.Read(buf)
}

type Display struct {
	mu      sync.Mutex
	debug   bool
	errch   chan<- error
	bufsize int
	buf     []byte
	imageid uint32
	qmask   *Image

	Image       *Image
	Screen      *Screen
	ScreenImage *Image
	Windows     *Image
	DPI         int

	firstfont *Font
	lastfont  *Font

	White       *Image // Pre-allocated color.
	Black       *Image // Pre-allocated color.
	Opaque      *Image // Pre-allocated color.
	Transparent *Image // Pre-allocated color.

	DefaultFont    *Font
	DefaultSubfont *Subfont

	conn        *drawFile
	ctl         *os.File
	mousectl    *Mousectl
	keyboardctl *Keyboardctl

	mtpt string
}

// An Image represents an image on the server, possibly visible on the display.
type Image struct {
	Display *Display
	id      uint32
	Pix     Pix             // The pixel format for the image.
	Depth   int             // The depth of the pixels in bits.
	Repl    bool            // Whether the image is replicated (tiles the rectangle).
	R       image.Rectangle // The extent of the image.
	Clipr   image.Rectangle // The clip region.
	Origin  image.Point     // Of image in screen, for mouse warping.
	next    *Image
	Screen  *Screen // If non-nil, the associated screen; this is a window.
}

// A Screen is a collection of windows that are visible on an image.
type Screen struct {
	Display *Display // Display connected to the server.
	id      uint32
	Fill    *Image // Background image behind the windows.
}

// Refresh algorithms to execute when a window is resized or uncovered.
// Refmesg is almost always the correct one to use.
const (
	Refbackup = 0
	Refnone   = 1
	Refmesg   = 2
)

const deffontname = "*default*"

// Init starts and connects to a server and returns a Display structure through
// which all graphics will be mediated. The arguments are an error channel on
// which to deliver errors (currently unused), the name of the font to use (the
// empty string may be used to represent the default font), the window label,
// and the window size as a string in the form XxY, as in "1000x500"; the units
// are pixels.
func Init(errch chan<- error, fontname, label, winsize string) (*Display, error) {
	if errch == nil {
		errch = make(chan error, 1)
	}

	var err error

	d := &Display{
		errch: errch,
	}

	// Lock Display so we maintain the contract within this library.
	d.mu.Lock()
	defer d.mu.Unlock()

	if dbg := os.Getenv("DRAWDEBUG"); dbg != "" {
		d.debug = true
	}

	width, height := 800, 600
	if winsize != "" {
		t := strings.Split(winsize, "x")
		if len(t) != 2 {
			return nil, fmt.Errorf("bad winsize, must be $widthx$height")
		}
		width, err = strconv.Atoi(t[0])
		if err != nil {
			return nil, fmt.Errorf("bad width in winsize: %s", err)
		}
		height, err = strconv.Atoi(t[1])
		if err != nil {
			return nil, fmt.Errorf("bad height in winsize: %s", err)
		}
	}

	wsys := os.Getenv("wsys")
	if wsys == "" {
		return nil, fmt.Errorf("$wsys not set")
	}
	wsysfd, err := os.OpenFile(wsys, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	// note: must not close wsysfd, or fd's get mixed up...
	d.mtpt = "/n/duit." + path.Base(wsys)
	err = syscall.Mount(int(wsysfd.Fd()), -1, d.mtpt, 0, fmt.Sprintf("new -r 0 0 %d %d", width, height))
	if err != nil {
		return nil, err
	}

	d.ctl, err = os.OpenFile("/dev/draw/new", os.O_RDWR|syscall.O_CLOEXEC, 0666)
	if err != nil {
		return nil, err
	}

	info, err := d.readctl()
	if err != nil {
		return nil, err
	}

	id := atoi(info[:1*12])
	drawDir := fmt.Sprintf("/dev/draw/%d", id)
	fd, err := os.OpenFile(drawDir+"/data", os.O_RDWR|syscall.O_CLOEXEC, 0666)
	if err != nil {
		return nil, err
	}
	d.conn = &drawFile{fd}

	pix, _ := ParsePix(strings.TrimSpace(string(info[2*12 : 3*12])))

	d.Image = &Image{
		Display: d,
		id:      0,
		Pix:     pix,
		Depth:   pix.Depth(),
		Repl:    atoi(info[3*12:]) > 0,
		R:       ator(info[4*12:]),
		Clipr:   ator(info[8*12:]),
	}

	d.bufsize = Iounit(int(d.conn.Fd()))
	if d.bufsize <= 0 {
		d.bufsize = 8000
	}
	if d.bufsize < 512 {
		return nil, fmt.Errorf("iounit too small")
	}
	d.buf = make([]byte, 0, d.bufsize+5)

	d.White, err = d.allocImage(image.Rect(0, 0, 1, 1), GREY1, true, White)
	if err != nil {
		return nil, fmt.Errorf("can't allocate white: %s", err)
	}
	d.Black, err = d.allocImage(image.Rect(0, 0, 1, 1), GREY1, true, Black)
	if err != nil {
		return nil, fmt.Errorf("can't allocate black: %s", err)
	}
	d.Opaque = d.White
	d.Transparent = d.Black

	/*
	 * Set up default font
	 */
	df, err := getdefont(d)
	if err != nil {
		return nil, err
	}
	d.DefaultSubfont = df

	if fontname == "" {
		fontname = os.Getenv("font")
	}

	/*
	 * Build fonts with caches==depth of screen, for speed.
	 * If conversion were faster, we'd use 0 and save memory.
	 */
	var font *Font
	if fontname == "" {
		buf := []byte(fmt.Sprintf("%d %d\n0 %d\t%s\n", df.Height, df.Ascent,
			df.N-1, deffontname))
		//fmt.Printf("%q\n", buf)
		//BUG: Need something better for this	installsubfont("*default*", df);
		font, err = d.buildFont(buf, deffontname)
	} else {
		font, err = d.openFont(fontname) // BUG: grey fonts
	}
	if err != nil {
		return nil, err
	}
	d.DefaultFont = font

	err = ioutil.WriteFile(d.mtpt+"/label", []byte(label), 0600)
	if err != nil {
		return nil, err
	}

	err = gengetwindow(d, d.mtpt+"/winname", Refnone)
	if err != nil {
		d.close()
		return nil, err
	}

	d.mousectl = d.initMouse()
	d.keyboardctl = d.initKeyboard()

	return d, nil
}

// Attach (re-)attaches to a display, typically after a resize, updating the
// display's associated image, screen, and screen image data structures.
func (d *Display) Attach(ref int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.getwindow(ref)
}

func (d *Display) getwindow(ref int) error {
	return gengetwindow(d, d.mtpt+"/winname", ref)
}

// Attach, or possibly reattach, to window.
// If reattaching, maintain value of screen pointer.
func gengetwindow(d *Display, winname string, ref int) error {
	var i *Image

	buf, err := ioutil.ReadFile(winname)
	if err != nil {
		return fmt.Errorf("gengetwindow: %s", err)
	}
	i, err = d.namedimage(buf)
	if err != nil {
		return fmt.Errorf("namedimage %s: %s", buf, err)
	}

	if d.ScreenImage != nil {
		d.ScreenImage.free()
		d.Screen.free()
		d.Screen = nil
	}

	if i == nil {
		d.ScreenImage = nil
		return fmt.Errorf("namedimage returned nil image")
	}

	d.Screen, err = i.allocScreen(d.White, false)
	if err != nil {
		return err
	}

	r := i.R
	const Borderwidth = 4
	r = i.R.Inset(Borderwidth)

	d.ScreenImage = d.Image
	d.ScreenImage, err = allocwindow(nil, d.Screen, r, 0, White)
	if err != nil {
		return err
	}
	err = originwindow(d.ScreenImage, image.Pt(0, 0), r.Min)
	if err != nil {
		return err
	}

	screen := d.ScreenImage
	screen.draw(screen.R, d.White, nil, image.ZP)
	if err := d.flush(true); err != nil {
		return err
	}

	return nil
}

func (d *Display) readctl() ([]byte, error) {
	buf := make([]byte, 12*12)
	n, err := d.ctl.Read(buf)
	if err == nil && n < 143 {
		return nil, fmt.Errorf("bad ctl read, expected 143 bytes, saw %d", n)
	}
	return buf[:n], err
}

/* implements message 'n' */
func (d *Display) namedimage(name []byte) (*Image, error) {
	err := d.flush(false)
	if err != nil {
		return nil, err
	}
	a := d.bufimage(1 + 4 + 1 + len(name))
	d.imageid++
	id := d.imageid

	a[0] = 'n'
	bplong(a[1:], id)
	a[5] = byte(len(name))
	copy(a[6:], name)
	err = d.flush(false)
	if err != nil {
		return nil, fmt.Errorf("namedimage: %s", err)
	}

	ctlbuf, err := d.readctl()
	if err != nil {
		return nil, fmt.Errorf("namedimage: %s", err)
	}

	pix, _ := ParsePix(string(ctlbuf[2*12 : 3*12]))
	image := &Image{
		Display: d,
		id:      id,
		Pix:     pix,
		Depth:   pix.Depth(),
		Repl:    atoi(ctlbuf[3*12:]) > 0,
		R:       ator(ctlbuf[4*12:]),
		Clipr:   ator(ctlbuf[8*12:]),
		next:    nil,
		Screen:  nil,
	}

	return image, nil
}

// Close closes the Display.
func (d *Display) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.close()
}

func (d *Display) close() error {
	d.keyboardctl.fd.Close()
	d.keyboardctl.ctlfd.Close()
	d.mousectl.mfd.Close()
	d.mousectl.cfd.Close()
	ioutil.WriteFile(d.mtpt+"/wctl", []byte("delete"), 0666)
	d.conn.Close()
	d.ctl.Close()
	return nil
}

// TODO: drawerror

func (d *Display) flushBuffer() error {
	if len(d.buf) == 0 {
		return nil
	}
	_, err := d.conn.Write(d.buf)
	d.buf = d.buf[:0]
	if err != nil {
		fmt.Fprintf(os.Stderr, "doflush: %s\n", err)
		return err
	}
	return nil
}

// Flush writes any pending data to the screen.
func (d *Display) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.flush(true)
}

// flush data, maybe make visible
func (d *Display) flush(vis bool) error {
	if vis {
		d.bufsize++
		a := d.bufimage(1)
		d.bufsize--
		a[0] = 'v'
	}

	return d.flushBuffer()
}

func (d *Display) bufimage(n int) []byte {
	if d == nil || n < 0 || n > d.bufsize {
		panic("bad count in bufimage")
	}
	if len(d.buf)+n > d.bufsize {
		if err := d.flushBuffer(); err != nil {
			panic("bufimage flush: " + err.Error())
		}
	}
	i := len(d.buf)
	d.buf = d.buf[:i+n]
	return d.buf[i:]
}

const DefaultDPI = 133

// TODO: Document.
func (d *Display) Scale(n int) int {
	if d == nil || d.DPI <= DefaultDPI {
		return n
	}
	return (n*d.DPI + DefaultDPI/2) / DefaultDPI
}

func atoi(b []byte) int {
	i := 0
	for i < len(b) && b[i] == ' ' {
		i++
	}
	n := 0
	for ; i < len(b) && '0' <= b[i] && b[i] <= '9'; i++ {
		n = n*10 + int(b[i]) - '0'
	}
	return n
}

func atop(b []byte) image.Point {
	return image.Pt(atoi(b), atoi(b[12:]))
}

func ator(b []byte) image.Rectangle {
	return image.Rectangle{atop(b), atop(b[2*12:])}
}

func bplong(b []byte, n uint32) {
	binary.LittleEndian.PutUint32(b, n)
}

func bpshort(b []byte, n uint16) {
	binary.LittleEndian.PutUint16(b, n)
}

func (d *Display) HiDPI() bool {
	return d.DPI >= DefaultDPI*3/2
}

func (d *Display) ScaleSize(n int) int {
	if d == nil || d.DPI <= DefaultDPI {
		return n
	}
	return (n*d.DPI + DefaultDPI/2) / DefaultDPI
}

func originwindow(i *Image, log, scr image.Point) error {
	d := i.Display
	err := d.flush(false)
	if err != nil {
		return err
	}
	b := d.bufimage(1 + 4 + 2*4 + 2*4)
	b[0] = 'o'
	bplong(b[1:], i.id)
	bplong(b[5:], uint32(log.X))
	bplong(b[9:], uint32(log.Y))
	bplong(b[13:], uint32(scr.X))
	bplong(b[17:], uint32(scr.Y))
	err = d.flush(false)
	if err != nil {
		return err
	}
	delta := log.Sub(i.R.Min)
	i.R = i.R.Add(delta)
	i.Clipr = i.Clipr.Add(delta)
	i.Origin = i.Origin.Sub(delta)
	return nil
}

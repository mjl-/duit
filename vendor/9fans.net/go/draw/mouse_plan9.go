package draw

import (
	"fmt"
	"image"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// Mouse is the structure describing the current state of the mouse.
type Mouse struct {
	image.Point        // Location.
	Buttons     int    // Buttons; bit 0 is button 1, bit 1 is button 2, etc.
	Msec        uint32 // Time stamp in milliseconds.
}

// TODO: Mouse field is racy but okay.

// Mousectl holds the interface to receive mouse events.
// The Mousectl's Mouse is updated after send so it doesn't
// have the wrong value if the sending goroutine blocks during send.
// This means that programs should receive into Mousectl.Mouse
//  if they want full synchrony.
type Mousectl struct {
	Mouse                // Store Mouse events here.
	C       <-chan Mouse // Channel of Mouse events.
	Resize  <-chan bool  // Each received value signals a window resize (see the display.Attach method).
	Display *Display     // The associated display.

	mfd *os.File // mouse
	cfd *os.File // cursor
}

func (d *Display) InitMouse() *Mousectl {
	return d.mousectl
}

func (d *Display) initMouse() *Mousectl {
	ch := make(chan Mouse, 0)
	rch := make(chan bool, 2)
	mc := &Mousectl{
		C:       ch,
		Resize:  rch,
		Display: d,
	}
	var err error
	mc.mfd, err = os.OpenFile(d.mtpt+"/mouse", os.O_RDWR|syscall.O_CLOEXEC, 0666)
	if err != nil {
		log.Fatal(err)
	}

	mc.cfd, err = os.OpenFile(d.mtpt+"/cursor", os.O_RDWR|syscall.O_CLOEXEC, 0666)
	if err != nil {
		log.Fatal(err)
	}

	d.mousectl = mc
	go mouseproc(mc, d, ch, rch)
	return mc
}

func mouseproc(mc *Mousectl, d *Display, ch chan Mouse, rch chan bool) {
	buf := make([]byte, 1+5*12)

	for {
		n, err := mc.mfd.Read(buf)
		if n != 1+4*12 {
			log.Fatalf("mouse: bad count %d: %s", n, err)
		}

		switch buf[0] {
		case 'r': // resize
			rch <- true
			fallthrough
		case 'm': // mouse move
			// note: msec can be negative on plan9
			msec, _ := strconv.ParseInt(strings.TrimLeft(string(buf[1+3*12:][:11]), " "), 10, 64)
			mm := Mouse{
				Point: image.Point{
					X: atoi(buf[1+0*12:][:11]),
					Y: atoi(buf[1+1*12:][:11]),
				}.Sub(d.ScreenImage.Origin),
				Buttons: atoi(buf[1+2*12:][:11]),
				Msec:    uint32(msec),
			}
			ch <- mm
			/*
			 * See comment above.
			 */
			mc.Mouse = mm
		}
	}
}

// Read returns the next mouse event.
func (mc *Mousectl) Read() Mouse {
	mc.Display.Flush()
	m := <-mc.C
	mc.Mouse = m
	return m
}

// MoveTo moves the mouse cursor to the specified location.
func (d *Display) MoveTo(pt image.Point) error {
	pt = pt.Add(d.ScreenImage.Origin)
	_, err := fmt.Fprintf(d.mousectl.mfd, "m%d %d", pt.X, pt.Y)
	return err
}

// SetCursor sets the mouse cursor to the specified cursor image.
// SetCursor(nil) changes the cursor to the standard system cursor.
func (d *Display) SetCursor(c *Cursor) {
	mc := d.mousectl
	if c == nil {
		mc.cfd.Write([]byte{0})
	} else {
		buf := make([]byte, 2*4+2*2*16)
		bplong(buf, uint32(c.Point.X))
		bplong(buf[4:], uint32(c.Point.Y))
		copy(buf[8:], c.Clr[:])
		copy(buf[8+2*16:], c.Set[:])
		mc.cfd.Write(buf)
	}
}

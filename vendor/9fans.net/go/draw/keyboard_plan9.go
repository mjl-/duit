package draw

import (
	"io"
	"log"
	"os"
	"syscall"
	"unicode/utf8"
)

const (
	KeyFn = '\uF000'

	KeyHome      = KeyFn | 0x0D
	KeyUp        = KeyFn | 0x0E
	KeyPageUp    = KeyFn | 0xF
	KeyPrint     = KeyFn | 0x10
	KeyLeft      = KeyFn | 0x11
	KeyRight     = KeyFn | 0x12
	KeyDown      = 0x80
	KeyView      = 0x80
	KeyPageDown  = KeyFn | 0x13
	KeyInsert    = KeyFn | 0x14
	KeyEnd       = KeyFn | 0x18
	KeyAlt       = KeyFn | 0x15
	KeyShift     = KeyFn | 0x16
	KeyCtl       = KeyFn | 0x17
	KeyBackspace = 0x08
	KeyDelete    = 0x7F
	KeyEscape    = 0x1b
	KeyEOF       = 0x04
	KeyCmd       = 0xF100
)

type Keyboardctl struct {
	C <-chan rune // Channel on which keyboard characters are delivered.

	fd    *os.File
	ctlfd io.WriteCloser
}

func (d *Display) InitKeyboard() *Keyboardctl {
	return d.keyboardctl
}

func (d *Display) initKeyboard() *Keyboardctl {
	ch := make(chan rune, 20)
	kc := &Keyboardctl{
		C: ch,
	}
	var err error
	kc.fd, err = os.OpenFile(d.mtpt+"/cons", os.O_RDWR|syscall.O_CLOEXEC, 0666)
	if err != nil {
		log.Fatal(err)
	}

	// Must keep reference to ctlfd open so rawon stays in effect.
	kc.ctlfd, err = os.OpenFile(d.mtpt+"/consctl", os.O_WRONLY|syscall.O_CLOEXEC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	_, err = kc.ctlfd.Write([]byte("rawon"))
	if err != nil {
		log.Fatal(err)
	}
	go kbdproc(d, kc.fd, kc.ctlfd, ch)
	return kc
}

func kbdproc(d *Display, fd io.ReadCloser, ctlfd io.Closer, ch chan rune) {
	buf := make([]byte, 32)
	have := 0
	for {
		for have > 0 && utf8.FullRune(buf[:have]) {
			k, n := utf8.DecodeRune(buf[:have])
			copy(buf, buf[n:have])
			have -= n
			ch <- k
		}
		n, err := fd.Read(buf[have:])
		if err != nil {
			select {
			case d.errch <- err:
			default:
			}
			fd.Close()
			ctlfd.Close()
			return
		}
		have += n
	}
}

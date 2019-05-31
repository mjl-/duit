package draw

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
)

func getsubfont(d *Display, name string) (*Subfont, error) {
	scale, fname := parsefontscale(name)
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "getsubfont: %v\n", err)
		return nil, err
	}
	f, err := d.readSubfont(name, bytes.NewReader(data), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "getsubfont: can't read %s: %v\n", fname, err)
	}
	if scale > 1 {
		scalesubfont(f, scale)
	}
	return f, err
}

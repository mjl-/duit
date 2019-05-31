package draw

import (
	"io/ioutil"
)

// ReadSnarf reads the snarf buffer into buf, returning the number of bytes read,
// the total size of the snarf buffer (useful if buf is too short), and any
// error. No error is returned if there is no problem except for buf being too
// short.
func (d *Display) ReadSnarf(buf []byte) (int, int, error) {
	sbuf, err := ioutil.ReadFile(d.mtpt + "/snarf")
	if err != nil {
		return -1, -1, err
	}
	n := len(sbuf)
	if len(buf) < n {
		n = len(buf)
	}
	copy(buf, sbuf[:n])
	return n, len(sbuf), nil
}

// WriteSnarf writes the data to the snarf buffer.
func (d *Display) WriteSnarf(data []byte) error {
	return ioutil.WriteFile(d.mtpt+"/snarf", data, 0666)
}

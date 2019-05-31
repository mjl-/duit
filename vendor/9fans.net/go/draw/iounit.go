package draw

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

// return size of atomic I/O unit for file descriptor
func Iounit(fd int) int {
	buf, err := ioutil.ReadFile(fmt.Sprintf("#d/%dctl", fd))
	if err != nil {
		return 0
	}

	tok := strings.Fields(string(buf))
	if len(tok) != 10 {
		return 0
	}

	iounit, _ := strconv.Atoi(tok[7])

	return iounit
}

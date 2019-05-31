package draw

import "image"

// Cursor describes a single cursor.
type Cursor struct {
	image.Point
	Clr [2 * 16]uint8
	Set [2 * 16]uint8
}

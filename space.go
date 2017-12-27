package duit

import (
	"image"
)

type Space struct {
	Top, Right, Bottom, Left int
}

func (s Space) Dx() int {
	return s.Left + s.Right
}

func (s Space) Dy() int {
	return s.Top + s.Bottom
}

func (s Space) Size() image.Point {
	return image.Pt(s.Dx(), s.Dy())
}

func (s Space) Topleft() image.Point {
	return image.Pt(s.Left, s.Top)
}

func SpaceXY(x, y int) Space {
	return Space{y, x, y, x}
}

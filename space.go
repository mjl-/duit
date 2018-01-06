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

func (s Space) Inset(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X+s.Left, r.Min.Y+s.Top, r.Max.X-s.Right, r.Max.Y-s.Bottom)
}

func SpaceXY(x, y int) Space {
	return Space{y, x, y, x}
}

func SpacePt(p image.Point) Space {
	return Space{p.Y, p.X, p.Y, p.X}
}

// NSpace is a convenience function to create N identical spaces.
func NSpace(n int, space Space) []Space {
	l := make([]Space, n)
	for i := 0; i < n; i++ {
		l[i] = space
	}
	return l
}

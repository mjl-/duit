package duit

import (
	"image"
)

// Space represents the padding or margin on a UI element.
// In duit functions, these are typically in lowDPI pixels.
type Space struct {
	Top, Right, Bottom, Left int
}

// Dx returns the total horizontal space.
func (s Space) Dx() int {
	return s.Left + s.Right
}

// Dy returns the total vertical space.
func (s Space) Dy() int {
	return s.Top + s.Bottom
}

// Size returns the total horizontal and vertical size.
func (s Space) Size() image.Point {
	return image.Pt(s.Dx(), s.Dy())
}

// Mul returns a this space multiplied by n.
func (s Space) Mul(n int) Space {
	s.Top *= n
	s.Right *= n
	s.Bottom *= n
	s.Left *= n
	return s
}

// Topleft returns a point containing the topleft space.
func (s Space) Topleft() image.Point {
	return image.Pt(s.Left, s.Top)
}

// Inset returns a rectangle that is r inset with this space.
func (s Space) Inset(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X+s.Left, r.Min.Y+s.Top, r.Max.X-s.Right, r.Max.Y-s.Bottom)
}

// SpaceXY returns a space with x for left/right and y for top/bottom.
func SpaceXY(x, y int) Space {
	return Space{y, x, y, x}
}

// SpacePt returns a space with p.X for left/right and p.Y for top/bottom.
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

// NSpaceXY is a convenience function to create N identical SpaceXY's.
func NSpaceXY(n, x, y int) []Space {
	l := make([]Space, n)
	for i := 0; i < n; i++ {
		l[i] = SpaceXY(x, y)
	}
	return l
}

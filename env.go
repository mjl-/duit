package duit

import (
	"9fans.net/go/draw"
)

type Env struct {
	Display *draw.Display

	// color for text
	Normal,
	Hover,
	Disabled,
	Inverse,
	Selection,
	SelectionHover,
	Placeholder,
	Primary Colors

	BackgroundColor draw.Color

	Background,
	ScrollBGNormal,
	ScrollBGHover,
	ScrollVisibleNormal,
	ScrollVisibleHover *draw.Image

	// sizes scaled for DPI of screen
	Size Size

	DebugKids   bool // whether to print distinct backgrounds in kids* functions
	debugColors []*draw.Image
}

func (e *Env) Scale(v int) int {
	return scale(e.Display, v)
}

func (e *Env) ScaleSpace(s Space) Space {
	return Space{
		e.Scale(s.Top),
		e.Scale(s.Right),
		e.Scale(s.Bottom),
		e.Scale(s.Left),
	}
}

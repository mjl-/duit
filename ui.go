package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Result struct {
	Hit      UI           // the UI where the event ended up
	Consumed bool         // whether event was consumed, and should not be further handled by upper UI's
	Redraw   bool         // whether event needs a redraw after
	Layout   bool         // whether event needs a layout after
	Warp     *image.Point // if set, mouse will warp to location
}

type Colors struct {
	Text,
	Background,
	Border *draw.Image
}

type Size struct {
	Margin  int
	Border  int
	Padding int
	Space   int
}

type Env struct {
	Display *draw.Display

	// color for text
	Normal,
	Hover,
	Disabled,
	Inverse Colors

	BackgroundColor draw.Color

	ScrollBGNormal,
	ScrollBGHover,
	ScrollVisibleNormal,
	ScrollVisibleHover *draw.Image

	// sizes scaled for DPI of screen
	Size Size
}

type UI interface {
	Layout(env *Env, r image.Rectangle, cur image.Point) image.Point
	Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse)
	Mouse(env *Env, m draw.Mouse) (result Result)
	Key(env *Env, orig image.Point, m draw.Mouse, k rune) (result Result)

	// FirstFocus returns the top-left corner where the focus should go next when "tab" is hit, if anything.
	FirstFocus(env *Env) *image.Point

	// Focus returns the focus-point for `ui`.
	Focus(env *Env, o UI) *image.Point

	// Print line about ui that includes r and is prefixed with indent spaces, following by a Print on each child.
	Print(indent int, r image.Rectangle)
}

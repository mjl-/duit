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

type UI interface {
	Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point
	Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse)
	Mouse(m draw.Mouse) (result Result)
	Key(orig image.Point, m draw.Mouse, k rune) (result Result)

	// FirstFocus returns the top-left corner where the focus should go next when "tab" is hit, if anything.
	FirstFocus() *image.Point

	// Focus returns the focus-point for `ui`.
	Focus(o UI) *image.Point

	// Print line about ui that includes r and is prefixed with indent spaces, following by a Print on each child.
	Print(indent int, r image.Rectangle)
}

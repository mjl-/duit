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
	Inverse,
	Primary Colors

	BackgroundColor draw.Color

	ScrollBGNormal,
	ScrollBGHover,
	ScrollVisibleNormal,
	ScrollVisibleHover *draw.Image

	// sizes scaled for DPI of screen
	Size Size
}

// UI is a user interface widget.
// It is implemented by Button, Label, Field, Image and List.
// And by layout UI's such as Box, Grid, Horizontal, Vertical and Scroll.
// Layout UI's simply contain other UI's, and are in charge of passing layout, draw, mouse, key, etc events on to the right child/children.
type UI interface {
	// Layout asks the UI to lay itself out with a max size of `r`.
	// The UI can lay itself out beyong size.Y, not beyond size.X.
	// size.Y is the amount of screen real estate that will still be visible.
	Layout(env *Env, size image.Point) image.Point

	// Draw asks the UI to draw itself on `img`, with an offset of `orig`.
	Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse)

	// Mouse tells the UI about mouse movement over it.
	// It can also be called when the mouse moved out of the UI. This facilitates redrawing after leaving the UI element, to draw it in non-hovered form. The UI is responsible for determining if the mouse is over the UI or not.
	// Result is used to tell the caller whether the event was consumed, and whether UI's need to be redrawn, etc.
	Mouse(env *Env, m draw.Mouse) (result Result)

	// Key tells the UI about a key press over it.
	Key(env *Env, orig image.Point, m draw.Mouse, k rune) (result Result)

	// FirstFocus returns the top-left corner where the focus should go next when "tab" is hit, if anything.
	FirstFocus(env *Env) *image.Point

	// Focus returns the focus-point for `ui`.
	Focus(env *Env, o UI) *image.Point

	// Print line about ui that includes r and is prefixed with indent spaces, following by a Print on each child.
	Print(indent int, r image.Rectangle)
}

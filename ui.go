package duit

import (
	"image"

	"9fans.net/go/draw"
)

// UI is a user interface widget.
// It is implemented by Button, Label, Field, Image and List.
// And by layout UI's such as Box, Grid, Horizontal, Vertical and Scroll.
// Layout UI's simply contain other UI's, and are in charge of passing layout, draw, mouse, key, etc events on to the right child/children.
type UI interface {
	// Layout asks the UI to lay itself out with a max size of `r`.
	// The UI can lay itself out beyong size.Y, not beyond size.X.
	// size.Y is the amount of screen real estate that will still be visible.
	Layout(env *Env, sizeAvail image.Point) (sizeTaken image.Point)

	// Draw asks the UI to draw itself on `img`, with `orig` as offset.
	Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse)

	// Mouse tells the UI about mouse movement over it.
	// OrigM is the mouse of first button down change, to facilitate tracking dragging. If no button is down, origM is the same as m.
	// It can also be called when the mouse moved out of the UI. This facilitates redrawing after leaving the UI element, to draw it in non-hovered form. The UI is responsible for determining if the mouse is over the UI or not.
	// Result is used to tell the caller whether the event was consumed, and whether UI's need to be redrawn, etc.
	Mouse(env *Env, origM, m draw.Mouse) (r Result)

	// Key tells the UI about a key press over it.
	Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result)

	// FirstFocus returns the top-left corner where the focus should go next when "tab" is hit, if anything.
	FirstFocus(env *Env) (warp *image.Point)

	// Focus returns the focus-point for `ui`.
	Focus(env *Env, o UI) (warp *image.Point)

	// Print line about ui that includes r and is prefixed with indent spaces, following by a Print on each child.
	Print(indent int, r image.Rectangle)
}

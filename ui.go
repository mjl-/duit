package duit

import (
	"image"

	"9fans.net/go/draw"
)

// UI is the interface implemented by a user interface element. For example Button, List, Grid, Scroll.
// UIs must be able to layout themselves, draw themselves, handle mouse events, key presses, deal with focus requests.
// UIs also help with propagating UI state and logging.
// For contain UIs (those that mostly just contain other UIs), many of these functions can be implemented by a single call to the corresponding Kids*-function.
type UI interface {
	// Layout asks the UI to layout itself and its children in `availSize`.
	// Layout must check `self.Layout` and `force`.
	// If force is set, it must layout itself and its kids, and pass on force.
	// Else, if self.Layout is DirtyKid, it only needs to call Layout on its kids (common for layout UIs).
	// The UI can lay itself out beyond size.Y, not beyond size.X.
	// size.Y is the amount of screen real estate that will still be visible.
	// Layout must update self.Draw if it needs to be drawn after.
	// Layout must update self.R with a image.ZP-origin image.Rectangle of the size it allocated.
	Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool)

	// Draw asks the UI to draw itself on `img`, with `orig` as offset and `m` as the current mouse (for hover states)
	// as self.Kid indicates, and pass further Draw calls on to its children as necessary.
	// If `force` is set, the UI must draw itself, overriding self.Draw.
	Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool)

	// Mouse tells the UI about mouse movement over it.
	// Layout UI's are in charge of passing these mouse events to their children.
	// `self.Layout` and `self.Draw` can be updated if the mouse event resulted in UIs needing relayout/redraw.
	// Again it's layout UI's responsibility to propagate requests from self.Layout and self.Draw to its parent.
	// `m` is the current mouse state, relative to this UIs zero point.
	// `origM` is the mouse of first button down change, to facilitate tracking dragging. If no button is down, origM is the same as m.
	// `orig` is the origin location of this UI. If you want to warp the mouse, add the origin to the UI-relative point.
	// Result is used to communicate results of the event back to the top.
	Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result)

	// Key tells the UI about a key press over it.
	// Like in Mouse, `self.Layout` and `self.Draw` can be updated.
	// `k` is the key pressed. There are no key down/up events, only keys typed.
	// See the Key-constants in the draw library for use special keys like the arrow keys,
	// function keys and combinations with the cmd key.
	// `m` is the mouse location at the time of the key, relative to this UIs zero point.
	// `orig` is the origin location of this UI. If you want to warp the mouse, add the origin to the UI-relative point.
	Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result)

	// FirstFocus returns where the focus should go next when "tab" is hit, if anything.
	FirstFocus(dui *DUI, self *Kid) (warp *image.Point)

	// Focus returns the focus-point for `ui`.
	Focus(dui *DUI, self *Kid, o UI) (warp *image.Point)

	// Mark looks for ui (itself or children), marks it as dirty for layout or draw (forLayout),
	// and propagates whether it marked anything back to the caller.
	Mark(self *Kid, o UI, forLayout bool) (marked bool)

	// Print line about ui that includes r and is prefixed with indent spaces, following by a Print on each child.
	Print(self *Kid, indent int)
}

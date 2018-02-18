package duit

import (
	"encoding/json"
	"fmt"
	"image"

	"9fans.net/go/draw"
)

// Kid holds a UI and its layout/draw state.
type Kid struct {
	UI     UI              // UI this state is about.
	R      image.Rectangle // Location and size within this UI.
	Draw   State           // Whether UI or its children need a draw.
	Layout State           // Whether UI or its children need a layout.
	ID     string          // For (re)storing settings with ReadSettings and WriteSettings. If empty, no settings for the UI will be (re)stored.
}

// MarshalJSON writes k with an additional field Type containing the name of the UI type.
func (k *Kid) MarshalJSON() ([]byte, error) {
	type kid struct {
		Kid
		Type string
	}
	return json.Marshal(kid{
		Kid:  *k,
		Type: fmt.Sprintf("%T", k.UI),
	})
}

// Mark checks if o is its UI, and if so marks it as needing a layout or draw (forLayout false).
func (k *Kid) Mark(o UI, forLayout bool) (marked bool) {
	if o != k.UI {
		return false
	}
	if forLayout {
		k.Layout = Dirty
	} else {
		k.Draw = Dirty
	}
	return true
}

// NewKids turns UIs into Kids containing those UIs. Useful for creating UI trees.
func NewKids(uis ...UI) []*Kid {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return kids
}

// KidsLayout is called by layout UIs before they do their own layouts.
// KidsLayout returns whether there is any work left to do, determined by looking at self.Layout.
// Children will be layed out if necessary. KidsLayout updates layout and draw state of self and kids.
func KidsLayout(dui *DUI, self *Kid, kids []*Kid, force bool) (done bool) {
	if force {
		self.Layout = Clean
		self.Draw = Dirty
		return false
	}
	switch self.Layout {
	case Clean:
		return true
	case Dirty:
		self.Layout = Clean
		self.Draw = Dirty
		return false
	}
	for _, k := range kids {
		if k.Layout == Clean {
			continue
		}
		k.UI.Layout(dui, k, k.R.Size(), false)
		switch k.Layout {
		case Dirty:
			self.Layout = Dirty
			self.Draw = Dirty
			return false
		case DirtyKid:
			panic("layout of kid results in kid.Layout = DirtKid")
		case Clean:
		}
	}
	self.Layout = Clean
	self.Draw = Dirty
	return true
}

// KidsDraw draws a UI by drawing all its kids.
// uiSize is the size of the entire UI, used in case it has to be redrawn entirely.
// Bg can override the default duit background color.
// Img is the whether the UI should be drawn on, with orig as origin (offset).
// M is used for passing a mouse position to the kid's UI draw, for possibly drawing hover states.
// KidsDraw only draws if draw state indicates a need for drawing, or if force is set.
func KidsDraw(dui *DUI, self *Kid, kids []*Kid, uiSize image.Point, bg, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)

	force = force || self.Draw == Dirty
	if force {
		self.Draw = Dirty
	}

	if bg == nil {
		bg = dui.Background
	}
	if force {
		img.Draw(rect(uiSize).Add(orig), bg, nil, image.ZP)
	}
	for i, k := range kids {
		if !force && k.Draw == Clean {
			continue
		}
		if dui.DebugKids {
			img.Draw(k.R.Add(orig), dui.debugColors[i%len(dui.debugColors)], nil, image.ZP)
		} else if !force && k.Draw == Dirty {
			img.Draw(k.R.Add(orig), bg, nil, image.ZP)
		}

		mm := m
		mm.Point = mm.Point.Sub(k.R.Min)
		if force {
			k.Draw = Dirty
		}
		k.UI.Draw(dui, k, img, orig.Add(k.R.Min), mm, force)
		k.Draw = Clean
	}
	self.Draw = Clean
}

func propagateResult(dui *DUI, self, k *Kid) {
	// log.Printf("propagateResult, r %#v, dirty %v kid ui %#v, \n", r, *dirty, k.UI)
	if k.Layout != Clean {
		if k.Layout == DirtyKid {
			// panic("kid propagated layout kids")
			k.Layout = Dirty // xxx
		}
		nk := *k
		k.UI.Layout(dui, &nk, k.R.Size(), false)
		if nk.R.Size() != k.R.Size() {
			self.Layout = Dirty
		} else {
			self.Layout = Clean
			k.Layout = Clean
			nk.R = nk.R.Add(k.R.Min)
			k.Draw = Dirty
			self.Draw = DirtyKid
		}
	} else if k.Draw != Clean {
		self.Draw = DirtyKid
	}
	// log.Printf("propagateResult, done %#v, dirty %v\n", r, *dirty)
}

// KidsMouse delivers mouse event m to the UI at origM (often the same, but not in case button is held pressed).
// Mouse positions are always relative to their own origin. Orig is passed so UIs can calculate locations to warp the mouse to.
func KidsMouse(dui *DUI, self *Kid, kids []*Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	for _, k := range kids {
		if !origM.Point.In(k.R) {
			continue
		}
		origM.Point = origM.Point.Sub(k.R.Min)
		m.Point = m.Point.Sub(k.R.Min)
		r = k.UI.Mouse(dui, k, m, origM, orig.Add(k.R.Min))
		if r.Hit == nil {
			r.Hit = k.UI
		}
		propagateResult(dui, self, k)
		return
	}
	return Result{}
}

// KidsKey delivers key event key to the UI at m.
// Orig is passed so UIs can calculate locations to warp the mouse to.
func KidsKey(dui *DUI, self *Kid, kids []*Kid, key rune, m draw.Mouse, orig image.Point) (r Result) {
	for i, k := range kids {
		if !m.Point.In(k.R) {
			continue
		}
		m.Point = m.Point.Sub(k.R.Min)
		r = k.UI.Key(dui, k, key, m, orig.Add(k.R.Min))
		if !r.Consumed && key == '\t' {
			for next := i + 1; next < len(kids); next++ {
				k := kids[next]
				first := k.UI.FirstFocus(dui, k)
				if first != nil {
					p := first.Add(orig).Add(k.R.Min)
					r.Warp = &p
					r.Consumed = true
					r.Hit = k.UI
					break
				}
			}
		}
		if r.Hit == nil {
			r.Hit = self.UI
		}
		propagateResult(dui, self, k)
		return
	}
	return Result{}
}

// KidsFirstFocus delivers the FirstFocus request to the first leaf UI, and returns the location where the mouse should warp to.
func KidsFirstFocus(dui *DUI, self *Kid, kids []*Kid) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		first := k.UI.FirstFocus(dui, k)
		if first != nil {
			p := first.Add(k.R.Min)
			return &p
		}
	}
	return nil
}

// KidsFocus delivers the Focus request to the first leaf UI, and returns the location where the mouse should warp to.
func KidsFocus(dui *DUI, self *Kid, kids []*Kid, ui UI) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		p := k.UI.Focus(dui, k, ui)
		if p != nil {
			pp := p.Add(k.R.Min)
			return &pp
		}
	}
	return nil
}

// KidsMark finds o in this UI subtree (self and kids), marks it as needing layout or draw (forLayout false), and returns whether it found and marked the UI.
func KidsMark(self *Kid, kids []*Kid, o UI, forLayout bool) (marked bool) {
	if self.Mark(o, forLayout) {
		return true
	}
	for _, k := range kids {
		marked = k.UI.Mark(k, o, forLayout)
		if !marked {
			continue
		}
		if forLayout {
			if self.Layout == Clean {
				self.Layout = DirtyKid
			}
		} else {
			if self.Draw == Clean {
				self.Draw = DirtyKid
			}
		}
		return true
	}
	return false
}

// KidsPrint calls Print on each kid UI.
func KidsPrint(kids []*Kid, indent int) {
	for _, k := range kids {
		k.UI.Print(k, indent)
	}
}

func propagateEvent(self *Kid, r *Result, e Event) {
	if e.NeedLayout {
		self.Layout = Dirty
	}
	if e.NeedDraw {
		self.Draw = Dirty
	}
	r.Consumed = e.Consumed || r.Consumed
}

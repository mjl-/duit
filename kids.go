package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Kid struct {
	UI     UI
	R      image.Rectangle
	Draw   State
	Layout State
}

func (k *Kid) Mark(o UI, forLayout bool, state State) (marked bool) {
	if o != k.UI {
		return false
	}
	if forLayout {
		k.Layout = state
	} else {
		k.Draw = state
	}
	return true
}

func NewKids(uis ...UI) []*Kid {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return kids
}

// kidsLayout is called by layout UIs before they do their actual layouts.
// kidsLayout tells them if there is any work to do, by looking at self.Layout.
func kidsLayout(dui *DUI, self *Kid, kids []*Kid, force bool) (done bool) {
	if force {
		self.Layout = StateClean
		self.Draw = StateSelf
		return false
	}
	switch self.Layout {
	case StateClean:
		return true
	case StateSelf:
		self.Layout = StateClean
		self.Draw = StateSelf
		return false
	}
	for _, k := range kids {
		if k.Layout == StateClean {
			continue
		}
		k.UI.Layout(dui, k, k.R.Size(), false)
		switch k.Layout {
		case StateSelf:
			self.Layout = StateSelf
			self.Draw = StateSelf
			return false
		case StateKid:
			panic("layout of kid results in kid.Layout = StateKid")
		case StateClean:
		}
	}
	self.Layout = StateClean
	self.Draw = StateSelf
	return true
}

func kidsDraw(name string, dui *DUI, self *Kid, kids []*Kid, uiSize image.Point, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(name, self)

	force = force || self.Draw == StateSelf
	if force {
		self.Draw = StateSelf
	}

	if force {
		img.Draw(rect(uiSize).Add(orig), dui.Background, nil, image.ZP)
	}
	for i, k := range kids {
		if !force && k.Draw == StateClean {
			continue
		}
		if dui.DebugKids {
			img.Draw(k.R.Add(orig), dui.debugColors[i%len(dui.debugColors)], nil, image.ZP)
		} else if !force && k.Draw == StateSelf {
			img.Draw(k.R.Add(orig), dui.Background, nil, image.ZP)
		}

		mm := m
		mm.Point = mm.Point.Sub(k.R.Min)
		if force {
			k.Draw = StateSelf
		}
		k.UI.Draw(dui, k, img, orig.Add(k.R.Min), mm, force)
		k.Draw = StateClean
	}
	self.Draw = StateClean
}

func propagateResult(dui *DUI, self, k *Kid) {
	// log.Printf("propagateResult, r %#v, dirty %v kid ui %#v, \n", r, *dirty, k.UI)
	if k.Layout != StateClean {
		if k.Layout == StateKid {
			panic("kid propagated layout kids")
		}
		nk := *k
		k.UI.Layout(dui, &nk, k.R.Size(), false)
		if nk.R.Size() != k.R.Size() {
			self.Layout = StateSelf
		} else {
			self.Layout = StateClean
			k.Layout = StateClean
			nk.R = nk.R.Add(k.R.Min)
			k.Draw = StateSelf
			self.Draw = StateKid
		}
	} else if k.Draw != StateClean {
		self.Draw = StateKid
	}
	// log.Printf("propagateResult, done %#v, dirty %v\n", r, *dirty)
}

func kidsMouse(dui *DUI, self *Kid, kids []*Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	for _, k := range kids {
		if !origM.Point.In(k.R) {
			continue
		}
		origM.Point = origM.Point.Sub(k.R.Min)
		m.Point = m.Point.Sub(k.R.Min)
		r = k.UI.Mouse(dui, k, m, origM, orig)
		if r.Hit == nil {
			r.Hit = k.UI
		}
		propagateResult(dui, self, k)
		return
	}
	return Result{}
}

func kidsKey(dui *DUI, self *Kid, kids []*Kid, key rune, m draw.Mouse, orig image.Point) (r Result) {
	for i, k := range kids {
		if !m.Point.In(k.R) {
			continue
		}
		m.Point = m.Point.Sub(k.R.Min)
		r = k.UI.Key(dui, k, key, m, orig.Add(k.R.Min))
		if !r.Consumed && key == '\t' {
			for next := i + 1; next < len(kids); next++ {
				first := kids[next].UI.FirstFocus(dui)
				if first != nil {
					kR := kids[next].R
					p := first.Add(orig).Add(kR.Min)
					r.Warp = &p
					r.Consumed = true
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

func kidsFirstFocus(dui *DUI, kids []*Kid) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		first := k.UI.FirstFocus(dui)
		if first != nil {
			p := first.Add(k.R.Min)
			return &p
		}
	}
	return nil
}

func kidsFocus(dui *DUI, kids []*Kid, ui UI) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		p := k.UI.Focus(dui, ui)
		if p != nil {
			pp := p.Add(k.R.Min)
			return &pp
		}
	}
	return nil
}

func kidsMark(self *Kid, kids []*Kid, o UI, forLayout bool, state State) (marked bool) {
	if self.Mark(o, forLayout, state) {
		return true
	}
	for _, k := range kids {
		marked = k.UI.Mark(k, o, forLayout, state)
		if !marked {
			continue
		}
		if forLayout {
			if self.Layout == StateClean {
				self.Layout = StateKid
			}
		} else {
			if self.Draw == StateClean {
				self.Draw = StateKid
			}
		}
		return true
	}
	return false
}

func kidsPrint(kids []*Kid, indent int) {
	for _, k := range kids {
		k.UI.Print(k, indent)
	}
}

func propagateEvent(self *Kid, r *Result, e Event) {
	if e.NeedLayout {
		self.Layout = StateSelf
	}
	if e.NeedDraw {
		self.Draw = StateSelf
	}
	r.Consumed = e.Consumed || r.Consumed
}

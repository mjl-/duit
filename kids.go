package duit

import (
	"encoding/json"
	"fmt"
	"image"

	"9fans.net/go/draw"
)

type Kid struct {
	UI     UI
	R      image.Rectangle
	Draw   State
	Layout State
	ID     string
}

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

func NewKids(uis ...UI) []*Kid {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return kids
}

// KidsLayout is called by layout UIs before they do their actual layouts.
// KidsLayout tells them if there is any work to do, by looking at self.Layout.
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

func KidsDraw(name string, dui *DUI, self *Kid, kids []*Kid, uiSize image.Point, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(name, self)

	force = force || self.Draw == Dirty
	if force {
		self.Draw = Dirty
	}

	if force {
		img.Draw(rect(uiSize).Add(orig), dui.Background, nil, image.ZP)
	}
	for i, k := range kids {
		if !force && k.Draw == Clean {
			continue
		}
		if dui.DebugKids {
			img.Draw(k.R.Add(orig), dui.debugColors[i%len(dui.debugColors)], nil, image.ZP)
		} else if !force && k.Draw == Dirty {
			img.Draw(k.R.Add(orig), dui.Background, nil, image.ZP)
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

func KidsMouse(dui *DUI, self *Kid, kids []*Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
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

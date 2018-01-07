package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Kid struct {
	UI UI
	R  image.Rectangle
}

func NewKids(uis ...UI) []*Kid {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return kids
}

func kidsDraw(dui *DUI, kids []*Kid, uiSize image.Point, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(rect(uiSize).Add(orig), dui.Background, nil, image.ZP)
	for i, k := range kids {
		if dui.DebugKids {
			img.Draw(k.R.Add(orig), dui.debugColors[i%len(dui.debugColors)], nil, image.ZP)
		}

		mm := m
		mm.Point = mm.Point.Sub(k.R.Min)
		k.UI.Draw(dui, img, orig.Add(k.R.Min), mm)
	}
}

func kidsMouse(dui *DUI, kids []*Kid, m draw.Mouse, origM draw.Mouse) Result {
	for _, k := range kids {
		if origM.Point.In(k.R) {
			origM.Point = origM.Point.Sub(k.R.Min)
			m.Point = m.Point.Sub(k.R.Min)
			return k.UI.Mouse(dui, m, origM)
		}
	}
	return Result{}
}

func kidsKey(dui *DUI, ui UI, kids []*Kid, key rune, m draw.Mouse, orig image.Point) Result {
	for i, k := range kids {
		if m.Point.In(k.R) {
			m.Point = m.Point.Sub(k.R.Min)
			r := k.UI.Key(dui, key, m, orig.Add(k.R.Min))
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
			return r
		}
	}
	return Result{Hit: ui}
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

func kidsPrint(kids []*Kid, indent int) {
	for _, k := range kids {
		k.UI.Print(indent, k.R)
	}
}

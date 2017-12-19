package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Kid struct {
	UI UI

	r image.Rectangle
}

func NewKids(uis ...UI) []*Kid {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return kids
}

func kidsDraw(display *draw.Display, kids []*Kid, uiSize image.Point, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(image.Rectangle{orig, orig.Add(uiSize)}, display.White, nil, image.ZP)
	for _, k := range kids {
		mm := m
		mm.Point = mm.Point.Sub(k.r.Min)
		k.UI.Draw(display, img, orig.Add(k.r.Min), mm)
	}
}

func kidsMouse(kids []*Kid, m draw.Mouse) Result {
	for _, k := range kids {
		if m.Point.In(k.r) {
			m.Point = m.Point.Sub(k.r.Min)
			return k.UI.Mouse(m)
		}
	}
	return Result{}
}

func kidsKey(ui UI, kids []*Kid, orig image.Point, m draw.Mouse, c rune) Result {
	for i, k := range kids {
		if m.Point.In(k.r) {
			m.Point = m.Point.Sub(k.r.Min)
			r := k.UI.Key(orig.Add(k.r.Min), m, c)
			if !r.Consumed && c == '\t' {
				for next := i + 1; next < len(kids); next++ {
					first := kids[next].UI.FirstFocus()
					if first != nil {
						kR := kids[next].r
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

func kidsFirstFocus(kids []*Kid) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		first := k.UI.FirstFocus()
		if first != nil {
			p := first.Add(k.r.Min)
			return &p
		}
	}
	return nil
}

func kidsFocus(kids []*Kid, ui UI) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		p := k.UI.Focus(ui)
		if p != nil {
			pp := p.Add(k.r.Min)
			return &pp
		}
	}
	return nil
}

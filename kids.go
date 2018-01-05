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

func kidsDraw(env *Env, kids []*Kid, uiSize image.Point, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(rect(uiSize).Add(orig), env.Background, nil, image.ZP)
	for i, k := range kids {
		if env.DebugKids {
			img.Draw(k.R.Add(orig), env.debugColors[i%len(env.debugColors)], nil, image.ZP)
		}

		mm := m
		mm.Point = mm.Point.Sub(k.R.Min)
		k.UI.Draw(env, img, orig.Add(k.R.Min), mm)
	}
}

func kidsMouse(env *Env, kids []*Kid, origM, m draw.Mouse) Result {
	for _, k := range kids {
		if origM.Point.In(k.R) {
			origM.Point = origM.Point.Sub(k.R.Min)
			m.Point = m.Point.Sub(k.R.Min)
			return k.UI.Mouse(env, origM, m)
		}
	}
	return Result{}
}

func kidsKey(env *Env, ui UI, kids []*Kid, orig image.Point, m draw.Mouse, c rune) Result {
	for i, k := range kids {
		if m.Point.In(k.R) {
			m.Point = m.Point.Sub(k.R.Min)
			r := k.UI.Key(env, orig.Add(k.R.Min), m, c)
			if !r.Consumed && c == '\t' {
				for next := i + 1; next < len(kids); next++ {
					first := kids[next].UI.FirstFocus(env)
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

func kidsFirstFocus(env *Env, kids []*Kid) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		first := k.UI.FirstFocus(env)
		if first != nil {
			p := first.Add(k.R.Min)
			return &p
		}
	}
	return nil
}

func kidsFocus(env *Env, kids []*Kid, ui UI) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		p := k.UI.Focus(env, ui)
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

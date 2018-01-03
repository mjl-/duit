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

func kidsDraw(env *Env, kids []*Kid, uiSize image.Point, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(rect(uiSize).Add(orig), env.Background, nil, image.ZP)
	for i, k := range kids {
		if env.DebugKids {
			img.Draw(k.r.Add(orig), env.debugColors[i%len(env.debugColors)], nil, image.ZP)
		}

		mm := m
		mm.Point = mm.Point.Sub(k.r.Min)
		k.UI.Draw(env, img, orig.Add(k.r.Min), mm)
	}
}

func kidsMouse(env *Env, kids []*Kid, origM, m draw.Mouse) Result {
	for _, k := range kids {
		if origM.Point.In(k.r) {
			origM.Point = origM.Point.Sub(k.r.Min)
			m.Point = m.Point.Sub(k.r.Min)
			return k.UI.Mouse(env, origM, m)
		}
	}
	return Result{}
}

func kidsKey(env *Env, ui UI, kids []*Kid, orig image.Point, m draw.Mouse, c rune) Result {
	for i, k := range kids {
		if m.Point.In(k.r) {
			m.Point = m.Point.Sub(k.r.Min)
			r := k.UI.Key(env, orig.Add(k.r.Min), m, c)
			if !r.Consumed && c == '\t' {
				for next := i + 1; next < len(kids); next++ {
					first := kids[next].UI.FirstFocus(env)
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

func kidsFirstFocus(env *Env, kids []*Kid) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		first := k.UI.FirstFocus(env)
		if first != nil {
			p := first.Add(k.r.Min)
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
			pp := p.Add(k.r.Min)
			return &pp
		}
	}
	return nil
}

func kidsPrint(kids []*Kid, indent int) {
	for _, k := range kids {
		k.UI.Print(indent, k.r)
	}
}

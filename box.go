package duit

import (
	"image"

	"9fans.net/go/draw"
)

// NewBox returns a box containing all uis in its Kids field.
func NewBox(uis ...UI) *Box {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return &Box{Kids: kids}
}

// NewReverseBox returns a box containing all uis in original order in its Kids field, with the Reverse field set.
func NewReverseBox(uis ...UI) *Box {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return &Box{Kids: kids, Reverse: true}
}

// Box keeps elements on a line as long as they fit, then moves on to the next line.
type Box struct {
	Kids       []*Kid      // Kids and UIs in this box.
	Reverse    bool        // Lay out children from bottom to top. First kid will be at the bottom.
	Margin     image.Point // In lowDPI pixels, will be adjusted for highDPI screens.
	Padding    Space       // Padding inside box, so children don't touch the sides; in lowDPI pixels, also adjusted for highDPI screens.
	Valign     Valign      // How to align children on a line.
	Width      int         // 0 means dynamic (as much as needed), -1 means full width, >0 means that exact amount of lowDPI pixels.
	Height     int         // 0 means dynamic (as much as needed), -1 means full height, >0 means that exact amount of lowDPI pixels.
	MaxWidth   int         // if >0, the max number of lowDPI pixels that will be used.
	Background *draw.Image `json:"-"` // Background for this box, instead of default duit background.

	size image.Point // of entire box, including padding
}

var _ UI = &Box{}

func (ui *Box) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	if KidsLayout(dui, self, ui.Kids, force) {
		return
	}

	if ui.Width < 0 && ui.MaxWidth > 0 {
		panic("combination ui.Width < 0 and ui.MaxWidth > 0 invalid")
	}

	osize := sizeAvail
	if ui.Width > 0 && dui.Scale(ui.Width) < sizeAvail.X {
		sizeAvail.X = dui.Scale(ui.Width)
	} else if ui.MaxWidth > 0 && dui.Scale(ui.MaxWidth) < sizeAvail.X {
		// note: ui.Width is currently the same as MaxWidth, but that might change when we don't mind extending beyong given X, eg with horizontal scroll
		sizeAvail.X = dui.Scale(ui.MaxWidth)
	}
	if ui.Height > 0 {
		sizeAvail.Y = dui.Scale(ui.Height)
	}
	padding := dui.ScaleSpace(ui.Padding)
	margin := scalePt(dui.Display, ui.Margin)
	sizeAvail = sizeAvail.Sub(padding.Size())
	nx := 0 // number on current line

	// variables below are about box contents not offset for padding
	cur := image.ZP
	xmax := 0  // max x seen so far
	lineY := 0 // max y of current line

	fixValign := func(kids []*Kid) {
		if len(kids) < 2 {
			return
		}
		for _, k := range kids {
			switch ui.Valign {
			case ValignTop:
			case ValignMiddle:
				k.R = k.R.Add(image.Pt(0, (lineY-k.R.Dy())/2))
			case ValignBottom:
				k.R = k.R.Add(image.Pt(0, lineY-k.R.Dy()))
			}
		}
	}

	for i, k := range ui.Kids {
		k.UI.Layout(dui, k, sizeAvail.Sub(image.Pt(0, cur.Y+lineY)), true)
		childSize := k.R.Size()
		var kr image.Rectangle
		if nx == 0 || cur.X+childSize.X <= sizeAvail.X {
			kr = rect(childSize).Add(cur).Add(padding.Topleft())
			cur.X += childSize.X + margin.X
			lineY = maximum(lineY, childSize.Y)
			nx += 1
		} else {
			if nx > 0 {
				fixValign(ui.Kids[i-nx : i])
				cur.X = 0
				cur.Y += lineY + margin.Y
			}
			kr = rect(childSize).Add(cur).Add(padding.Topleft())
			nx = 1
			cur.X = childSize.X + margin.X
			lineY = childSize.Y
		}
		k.R = kr
		if xmax < cur.X {
			xmax = cur.X
		}
	}
	fixValign(ui.Kids[len(ui.Kids)-nx : len(ui.Kids)])
	cur.Y += lineY

	if ui.Reverse {
		bottomY := cur.Y + padding.Dy()
		for _, k := range ui.Kids {
			y1 := bottomY - k.R.Min.Y
			y0 := y1 - k.R.Dy()
			k.R = image.Rect(k.R.Min.X, y0, k.R.Max.X, y1)
		}
	}

	ui.size = image.Pt(xmax-margin.X, cur.Y).Add(padding.Size())
	if ui.Width < 0 {
		ui.size.X = osize.X
	}
	if ui.Height < 0 && ui.size.Y < osize.Y {
		ui.size.Y = osize.Y
	}
	self.R = rect(ui.size)
}

func (ui *Box) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	KidsDraw(dui, self, ui.Kids, ui.size, ui.Background, img, orig, m, force)
}

func (ui *Box) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	return KidsMouse(dui, self, ui.Kids, m, origM, orig)
}

func (ui *Box) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return KidsKey(dui, self, ui.orderedKids(), k, m, orig)
}

func (ui *Box) orderedKids() []*Kid {
	if !ui.Reverse {
		return ui.Kids
	}
	n := len(ui.Kids)
	kids := make([]*Kid, n)
	for i := range ui.Kids {
		kids[i] = ui.Kids[n-1-i]
	}
	return kids
}

func (ui *Box) FirstFocus(dui *DUI, self *Kid) *image.Point {
	return KidsFirstFocus(dui, self, ui.orderedKids())
}

func (ui *Box) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	return KidsFocus(dui, self, ui.Kids, o)
}

func (ui *Box) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return KidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Box) Print(self *Kid, indent int) {
	PrintUI("Box", self, indent)
	KidsPrint(ui.Kids, indent+1)
}

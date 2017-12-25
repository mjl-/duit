package duit

import "image"

func pt(v int) image.Point {
	return image.Point{v, v}
}

func rect(p image.Point) image.Rectangle {
	return image.Rectangle{image.ZP, p}
}

func extendY(r image.Rectangle, dy int) image.Rectangle {
	r.Max.Y += dy
	return r
}

func insetPt(r image.Rectangle, pad image.Point) image.Rectangle {
	r.Min = r.Min.Add(pad)
	r.Max = r.Max.Sub(pad)
	return r
}

func outsetPt(r image.Rectangle, pad image.Point) image.Rectangle {
	r.Min = r.Min.Sub(pad)
	r.Max = r.Max.Add(pad)
	return r
}

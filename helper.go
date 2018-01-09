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

func minimum64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maximum64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maximum(a, b int) int {
	if a > b {
		return a
	}
	return b
}

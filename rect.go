package duit

import "image"

func rect(p image.Point) image.Rectangle {
	return image.Rectangle{image.ZP, p}
}

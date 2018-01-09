package duit

import (
	"9fans.net/go/draw"
)

type Icon struct {
	Rune rune
	Font *draw.Font `json:"-"`
}

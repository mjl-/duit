package main

import (
	"image"
	"log"

	"github.com/mjl-/duit"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	dui, err := duit.NewDUI("ex/pick", nil)
	check(err, "new dui")

	b1 := &duit.Button{Text: "b1"}
	b2 := &duit.Button{Text: "b2"}
	horizontal := &duit.Split{
		Gutter: 1,
		Split: func(width int) []int {
			return []int{width / 2, width - width/2}
		},
		Kids: duit.NewKids(b1, b2),
	}
	vertical := &duit.Split{
		Vertical: true,
		Gutter:   1,
		Split: func(height int) []int {
			return []int{height / 2, height - height/2}
		},
		Kids: duit.NewKids(b1, b2),
	}

	dui.Top.UI = &duit.Pick{
		Pick: func(sizeAvail image.Point) duit.UI {
			if sizeAvail.X < dui.Scale(800) {
				return vertical
			}
			return horizontal
		},
	}
	dui.Render()

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)

		case <-dui.Done:
			return
		}
	}
}

package main

import (
	"log"

	"github.com/mjl-/duit"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	dui, err := duit.NewDUI("ex/middle", "800x600")
	check(err, "new dui")

	dui.Top.UI = duit.NewMiddle(duit.SpaceXY(10, 10), &duit.Label{Text: "this label is centered vertically and horizontally"})
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

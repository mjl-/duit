package main

import (
	"image"
	"log"

	"mjl/duit"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	dui, err := duit.NewDUI("page", "800x600")
	check(err, "new dui")

	dui.Top = &duit.Box{
		Padding: duit.SpaceXY(6, 4),
		Margin:  image.Pt(6, 4),
		Valign:  duit.ValignMiddle,
		Kids: duit.NewKids(
			&duit.Button{
				Text: "click me",
			},
		),
	}
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)
		}
	}
}

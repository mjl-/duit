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
	dui, err := duit.NewDUI("ex/horizontal", nil)
	check(err, "new dui")

	dui.Top = duit.Kid{
		ID: "horizontal",
		UI: &duit.Split{
			Gutter: 1,
			Kids: duit.NewKids(
				&duit.Button{Text: "button1"},
				&duit.Button{Text: "button2"},
				&duit.Button{Text: "button3"},
			),
		},
	}
	dui.Render()

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)

		case err := <-dui.Error:
			check(err, "dui")
			return
		}
	}
}

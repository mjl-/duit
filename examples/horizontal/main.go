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
	dui, err := duit.NewDUI("ex/horizontal", "800x600")
	check(err, "new dui")

	dui.Top = duit.Kid{
		ID: "horizontal",
		UI: &duit.Split{
			Gutter: 1,
			Split: func(height int) []int {
				p := height / 4
				return []int{p, p, height - 2*p}
			},
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

		case <-dui.Done:
			return
		}
	}
}

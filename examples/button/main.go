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
	dui, err := duit.NewDUI("page", "800x600")
	check(err, "new dui")

	dui.Top.UI = &duit.Button{
		Text: "click me",
		Click: func(r *duit.Result, draw, layout *duit.State) {
			log.Printf("clicked\n")
		},
	}
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)

		case <-dui.Done:
			return
		}
	}
}

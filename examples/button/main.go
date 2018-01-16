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
	dui, err := duit.NewDUI("ex/button", nil)
	check(err, "new dui")

	dui.Top.UI = &duit.Button{
		Text: "click me",
		Click: func() (e duit.Event) {
			log.Printf("clicked\n")
			return
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

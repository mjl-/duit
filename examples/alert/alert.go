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
	dui, err := duit.NewDUI("ex/alert", &duit.DUIOpts{Dimensions: "400x300"})
	check(err, "new dui")

	field := &duit.Field{
		Text: "type an alert message here",
	}

	dui.Top.UI = &duit.Box{
		Padding: duit.SpaceXY(6, 4),
		Margin:  image.Pt(6, 4),
		Valign:  duit.ValignMiddle,
		Kids: duit.NewKids(
			field,
			&duit.Button{
				Text: "click me to create an alert",
				Click: func() (e duit.Event) {
					go func() {
						err := duit.Alert(field.Text)
						check(err, "alert")
						log.Printf("alert is done\n")
					}()
					return
				},
			},
		),
	}
	dui.Render()

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)

		case err, ok := <-dui.Error:
			if !ok {
				return
			}
			log.Printf("duit: %s\n", err)
		}
	}
}

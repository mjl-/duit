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

	dui.Top.UI = &duit.Grid{
		Columns: 3,
		Valign:  []duit.Valign{duit.ValignTop, duit.ValignMiddle, duit.ValignBottom},
		Halign:  []duit.Halign{duit.HalignLeft, duit.HalignMiddle, duit.HalignRight},
		Padding: duit.NSpace(3, duit.SpaceXY(6, 4)),
		Kids: duit.NewKids(
			&duit.Label{Text: "label1 longer"},
			&duit.Button{Text: "button 2"},
			&duit.Label{Text: "label3 longer"},
			&duit.Button{Text: "button 4"},
			&duit.Label{Text: "label5 longer"},
			&duit.Label{Text: "label6"},
			&duit.Label{Text: "label7"},
			&duit.Label{Text: "label8"},
			&duit.Button{Text: "button 9"},
		),
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

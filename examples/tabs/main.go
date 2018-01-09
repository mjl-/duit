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

	dui.Top.UI = &duit.Tabs{
		Buttongroup: &duit.Buttongroup{
			Texts: []string{
				"tab1",
				"tab2",
				"tab3",
			},
		},
		UIs: []duit.UI{
			&duit.Button{Text: "this is the content of tab1"},
			&duit.Field{Text: "this is the content of tab2"},
			&duit.Label{Text: "this is the content of tab3"},
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

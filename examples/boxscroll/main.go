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
	dui, err := duit.NewDUI("ex/boxscroll", nil)
	check(err, "new dui")

	dui.Top.UI = &duit.Box{
		Kids: duit.NewKids(
			&duit.Field{
				Text: "type in me",
			},
			duit.NewScroll(
				&duit.Label{
					Text: `Lorem ipsum dolor sit amet, graecis habemus dissentiet ei his, legere detraxit insolens mei et. Eum nullam fabellas eleifend ex. Ius possim ceteros te. Dolor eligendi nam cu, iuvaret elaboraret per te.

Est an partiendo prodesset, qui ea incorrupte efficiendi. Ei eam suavitate consectetuer. No est dictas singulis complectitur. Sit eius meliore constituto ea, eruditi percipit suscipiantur mei ex. Eu sea eruditi phaedrum recteque. Quot prompta ius eu, cu nec imperdiet signiferumque. Facete invidunt sed in, ne cum affert dolorem, an nam regione verterem.`,
				},
			),
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

package main

import (
	"fmt"
	"log"

	"mjl/duit"

	"9fans.net/go/draw"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	dui, err := duit.NewDUI("page", "800x600")
	check(err, "new dui")

	var rows []*duit.Gridrow
	for i := 0; i < 100; i++ {
		values := []string{
			fmt.Sprintf("cell 0,%d", i),
			fmt.Sprintf("cell 1,%d", i),
			fmt.Sprintf("cell 2,%d", i),
		}
		row := &duit.Gridrow{
			Selected: i%10 == 0,
			Values:   values,
		}
		rows = append(rows, row)
	}

	dui.Top = &duit.Scroll{
		Child: &duit.Gridlist{
			Header:   duit.Gridrow{Values: []string{"col1", "col2", "col3"}},
			Rows:     rows,
			Multiple: true,
			Striped:  true,
			Padding:  duit.SpaceXY(10, 2),
			Halign: []duit.Halign{
				duit.HalignLeft,
				duit.HalignMiddle,
				duit.HalignRight,
			},
			Changed: func(index int, result *duit.Result) {
				log.Printf("gridlist, index %d changed\n", index)
			},
			Click: func(index, buttons int, r *duit.Result) {
				log.Printf("gridlist, click, index %d, buttons %d\n", index, buttons)
			},
			Keys: func(index int, m draw.Mouse, k rune, r *duit.Result) {
				log.Printf("gridlist, key %c at index %d, mouse %v\n", k, index, m)
			},
		},
	}
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)
		}
	}
}

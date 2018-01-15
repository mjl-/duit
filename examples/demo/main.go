package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"time"

	"9fans.net/go/draw"

	"github.com/mjl-/duit"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	dui, err := duit.NewDUI("ex/demo", "600x400")
	check(err, "new dui")

	readImagePath := func(path string) *draw.Image {
		img, err := duit.ReadImagePath(dui.Display, path)
		check(err, "read image")
		return img
	}

	count := 0
	counter := &duit.Label{Text: fmt.Sprintf("%d", count)}
	tick := make(chan struct{}, 0)
	go func() {
		for {
			time.Sleep(1 * time.Second)
			tick <- struct{}{}
		}
	}()

	radio1 := &duit.Radiobutton{
		Selected: true,
		Value:    1,
		Changed: func(v interface{}, r *duit.Event) {
			log.Printf("radiobutton value changed, now %#v\n", v)
		},
	}
	radio2 := &duit.Radiobutton{
		Value: 2,
		Changed: func(v interface{}, r *duit.Event) {
			log.Printf("radiobutton value changed, now %#v\n", v)
		},
	}
	group := []*duit.Radiobutton{
		radio1,
		radio2,
	}
	radio1.Group = group
	radio2.Group = group

	dui.Top.UI = duit.NewBox(
		&duit.Split{
			Gutter:   1,
			Vertical: true,
			Split: func(height int) []int {
				row1 := height / 4
				row2 := height / 4
				row3 := height - row1 - row2
				return []int{row1, row2, row3}
			},
			Kids: duit.NewKids(
				&duit.Label{Text: "in row 1"},
				&duit.Scroll{
					Kid: duit.Kid{UI: &duit.Grid{
						Columns: 2,
						Padding: duit.NSpace(2, duit.SpaceXY(6, 4)),
						Halign:  []duit.Halign{duit.HalignRight, duit.HalignLeft},
						Valign:  []duit.Valign{duit.ValignMiddle, duit.ValignMiddle},
						Kids: duit.NewKids(
							&duit.Label{Text: "From"},
							&duit.Field{Text: "...from...", Disabled: true},
							&duit.Label{Text: "To"},
							&duit.Field{Text: "...to..."},
							&duit.Label{Text: "Cc"},
							&duit.Field{Text: "...cc..."},
							&duit.Label{Text: "Bcc"},
							&duit.Field{Text: "...bcc..."},
							&duit.Label{Text: "Subject"},
							&duit.Field{Text: "...subject..."},
							&duit.Label{Text: "Checkbox"},
							&duit.Checkbox{
								Checked: true,
								Changed: func(e *duit.Event) {
									log.Println("checkbox value changed")
								},
							},
							&duit.Label{Text: "Radio 1"},
							radio1,
							&duit.Label{Text: "Radio 2"},
							radio2,
						),
					}},
				},
				&duit.Scroll{
					Kid: duit.Kid{UI: &duit.Box{
						Reverse: true,
						Padding: duit.SpaceXY(6, 4),
						Margin:  image.Pt(6, 4),
						Kids: duit.NewKids(
							&duit.Label{Text: "counter:"},
							counter,
							&duit.Button{
								Text:     "button1",
								Colorset: &dui.Primary,
								Click: func(e *duit.Event) {
									log.Printf("button clicked")
								},
							},
							&duit.Button{
								Text:     "button2",
								Disabled: true,
								Click: func(e *duit.Event) {
								},
							},
							&duit.List{
								Multiple: true,
								Values: []*duit.ListValue{
									{Text: "Elem 1", Value: 1},
									{Text: "Elem 2", Value: 2},
									{Text: "Elem 3", Value: 3},
								},
							},
							&duit.Label{Text: "Horizontal split"},
							&duit.Split{
								Gutter: 1,
								Kids: duit.NewKids(
									&duit.Label{Text: "in column 1"},
									&duit.Label{Text: "in column 2"},
									&duit.Label{Text: "in column 3"},
								),
								Split: func(width int) []int {
									col1 := width / 4
									col2 := width / 4
									col3 := width - col1 - col2
									return []int{col1, col2, col3}
								},
							},
							&duit.Label{Text: "Another box with a scrollbar:"},
							&duit.Scroll{
								Kid: duit.Kid{UI: &duit.Box{
									Padding: duit.SpaceXY(6, 4),
									Margin:  image.Pt(6, 4),
									Kids: duit.NewKids(
										&duit.Label{Text: "another label, this one is somewhat longer"},
										&duit.Button{Text: "some other button"},
										&duit.Label{Text: "more labels"},
										&duit.Label{Text: "another"},
										&duit.Field{Text: "A field!!"},
										duit.NewBox(&duit.Image{Image: readImagePath("test.jpg")}),
										&duit.Field{Text: "A field!!"},
										duit.NewBox(&duit.Image{Image: readImagePath("test.jpg")}),
										&duit.Field{Text: "A field!!"},
										duit.NewBox(&duit.Image{Image: readImagePath("test.jpg")}),
									),
								}},
							},
							&duit.Button{Text: "button3"},
							&duit.Field{Text: "field 2"},
							&duit.Field{Text: "field 3"},
							&duit.Field{Text: "field 4"},
							&duit.Field{Text: "field 5"},
							&duit.Field{Text: "field 6"},
							&duit.Field{Text: "field 7"},
							&duit.Label{Text: "this is a label"},
						),
					}},
				},
			),
		},
	)
	dui.Render()

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)

		case <-dui.Done:
			return

		case <-tick:
			count++
			counter.Text = fmt.Sprintf("%d", count)
			dui.Render()
		}
	}
}

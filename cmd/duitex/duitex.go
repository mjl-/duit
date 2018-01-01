package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"time"

	"9fans.net/go/draw"

	"mjl/duit"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	dui, err := duit.NewDUI("duitex", "600x400")
	check(err, "new dui")

	redraw := make(chan struct{}, 1)

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
		Changed: func(v interface{}, r *duit.Result) {
			log.Println("radiobutton value changed, now %#v", v)
		},
	}
	radio2 := &duit.Radiobutton{
		Value: 2,
		Changed: func(v interface{}, r *duit.Result) {
			log.Println("radiobutton value changed, now %#v", v)
		},
	}
	group := []*duit.Radiobutton{
		radio1,
		radio2,
	}
	radio1.Group = group
	radio2.Group = group

	dui.Top = duit.NewBox(
		&duit.Vertical{
			Split: func(height int) []int {
				row1 := height / 4
				row2 := height / 4
				row3 := height - row1 - row2
				return []int{row1, row2, row3}
			},
			Kids: []*duit.Kid{
				{UI: &duit.Label{Text: "in row 1"}},
				{UI: &duit.Scroll{
					Child: &duit.Grid{
						Columns: 2,
						Padding: []duit.Space{
							duit.SpaceXY(6, 4),
							duit.SpaceXY(6, 4),
						},
						Halign: []duit.Halign{duit.HalignRight, duit.HalignLeft},
						Valign: []duit.Valign{duit.ValignMiddle, duit.ValignMiddle},
						Kids: []*duit.Kid{
							{UI: &duit.Label{Text: "From"}},
							{UI: &duit.Field{Text: "...from...", Disabled: true}},
							{UI: &duit.Label{Text: "To"}},
							{UI: &duit.Field{Text: "...to..."}},
							{UI: &duit.Label{Text: "Cc"}},
							{UI: &duit.Field{Text: "...cc..."}},
							{UI: &duit.Label{Text: "Bcc"}},
							{UI: &duit.Field{Text: "...bcc..."}},
							{UI: &duit.Label{Text: "Subject"}},
							{UI: &duit.Field{Text: "...subject..."}},
							{UI: &duit.Label{Text: "Checkbox"}},
							{UI: &duit.Checkbox{
								Checked: true,
								Changed: func(r *duit.Result) {
									log.Println("checkbox value changed")
								},
							}},
							{UI: &duit.Label{Text: "Radio 1"}},
							{UI: radio1},
							{UI: &duit.Label{Text: "Radio 2"}},
							{UI: radio2},
						},
					},
				}},
				{UI: &duit.Scroll{
					Child: &duit.Box{
						Reverse:     true,
						Padding:     duit.SpaceXY(6, 4),
						ChildMargin: image.Pt(6, 4),
						Kids: duit.NewKids(
							&duit.Label{Text: "counter:"},
							counter,
							&duit.Button{
								Text:    "button1",
								Primary: true,
								Click: func(r *duit.Result) {
									log.Printf("button clicked")
								},
							},
							&duit.Button{
								Text:     "button2",
								Disabled: true,
								Click: func(r *duit.Result) {
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
							&duit.Horizontal{
								Kids: []*duit.Kid{
									{UI: &duit.Label{Text: "in column 1"}},
									{UI: &duit.Label{Text: "in column 2"}},
									{UI: &duit.Label{Text: "in column 3"}},
								},
								Split: func(width int) []int {
									col1 := width / 4
									col2 := width / 4
									col3 := width - col1 - col2
									return []int{col1, col2, col3}
								},
							},
							&duit.Label{Text: "Another box with a scrollbar:"},
							&duit.Scroll{Child: &duit.Box{
								Padding:     duit.SpaceXY(6, 4),
								ChildMargin: image.Pt(6, 4),
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
							&duit.Button{Text: "button3"},
							&duit.Field{Text: "field 2"},
							&duit.Field{Text: "field 3"},
							&duit.Field{Text: "field 4"},
							&duit.Field{Text: "field 5"},
							&duit.Field{Text: "field 6"},
							&duit.Field{Text: "field 7"},
							&duit.Label{Text: "this is a label"},
						),
					},
				}},
			},
		})
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)

		case <-redraw:
			dui.Redraw()
		case <-tick:
			count++
			counter.Text = fmt.Sprintf("%d", count)
			dui.Render()
		}
	}
}

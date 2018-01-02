package main

import (
	"bytes"
	"image"
	"log"
	"os/exec"
	"strings"

	"mjl/duit"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	dui, err := duit.NewDUI("font", "800x600")
	check(err, "new dui")

	buf, err := exec.Command("fontsrv", "-p", ".").Output()
	check(err, "listing fonts using fontsrv")
	fonts := strings.Split(string(buf), "\n")

	fontValues := make([]*duit.ListValue, len(fonts))
	for i, s := range fonts {
		fontValues[i] = &duit.ListValue{Text: s}
	}

	src := bytes.NewReader([]byte(`0 1 2 3 4 5 6 7 8 9
a b c d e f g h i j k l m n o p q r s t u v w x y z`))
	edit := duit.NewEdit(src)

	var fontList *duit.List
	fontList = &duit.List{
		Values: fontValues,
		Changed: func(index int, r *duit.Result) {
			lv := fontList.Values[index]
			// xxx todo should free font, but that seems to hang draw
			if lv.Selected {
				font, err := dui.Env.Display.OpenFont("/mnt/font/" + lv.Text + "15a/font")
				check(err, "open font")
				edit.Font = font
			} else {
				edit.Font = nil
			}
			r.Redraw = true
		},
	}

	search := &duit.Field{
		Placeholder: "search...",
		Changed: func(s string, r *duit.Result) {
			s = strings.ToLower(s)
			nl := []*duit.ListValue{}
			for _, lv := range fontValues {
				if lv.Selected || strings.Contains(strings.ToLower(lv.Text), s) {
					nl = append(nl, lv)
				}
			}
			fontList.Values = nl
			r.Redraw = true
		},
	}

	dui.Top = &duit.Horizontal{
		Split: func(width int) []int {
			first := dui.Scale(250)
			return []int{first, width - first}
		},
		Kids: duit.NewKids(
			&duit.Box{
				Padding: duit.SpaceXY(6, 4),
				Margin:  image.Pt(0, 4),
				Kids: duit.NewKids(
					search,
					&duit.Scroll{
						Child: fontList,
					},
				),
			},
			edit,
		),
	}
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)
		}
	}
}

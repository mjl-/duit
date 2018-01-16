package main

import (
	"flag"
	"image"
	"log"
	"os"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

func rect(p image.Point) image.Rectangle {
	return image.Rectangle{image.ZP, p}
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		log.Println("duitplace image-path")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(2)
	}
	dui, err := duit.NewDUI("ex/place", nil)
	check(err, "new dui")

	readImagePath := func(path string) *draw.Image {
		img, err := duit.ReadImagePath(dui.Display, path)
		check(err, "read image")
		return img
	}

	var place *duit.Place
	place = &duit.Place{
		Place: func(self *duit.Kid, sizeAvail image.Point) {
			place.Kids[0].UI.Layout(dui, self, sizeAvail, true)
			place.Kids[1].UI.Layout(dui, place.Kids[1], sizeAvail, true)
			place.Kids[1].R = place.Kids[1].R.Add(sizeAvail.Sub(place.Kids[1].R.Size()).Div(2))
		},
		Kids: duit.NewKids(
			&duit.Image{
				Image: readImagePath(args[0]),
			},
			&duit.Button{Text: "testing"},
		),
	}
	dui.Top.UI = place
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

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
	dui, err := duit.NewDUI("page", "800x600")
	check(err, "new dui")

	readImagePath := func(path string) *draw.Image {
		img, err := duit.ReadImagePath(dui.Display, path)
		check(err, "read image")
		return img
	}

	var place *duit.Place
	place = &duit.Place{
		Place: func(sizeAvail image.Point) image.Point {
			imageSize := place.Kids[0].UI.Layout(dui.Env, sizeAvail)
			place.Kids[0].R = rect(imageSize)
			buttonSize := place.Kids[1].UI.Layout(dui.Env, sizeAvail)
			place.Kids[1].R = rect(buttonSize).Add(sizeAvail.Sub(buttonSize).Div(2))
			return imageSize
		},
		Kids: duit.NewKids(
			&duit.Image{
				Image: readImagePath(args[0]),
			},
			&duit.Button{Text: "testing"},
		),
	}
	dui.Top = place
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)
		}
	}
}

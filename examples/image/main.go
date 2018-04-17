package main

import (
	"flag"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		log.Println("duitimage path")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	dui, err := duit.NewDUI("ex/image", nil)
	check(err, "new dui")

	readImagePath := func(path string) *draw.Image {
		img, err := duit.ReadImagePath(dui.Display, path)
		check(err, "read image")
		return img
	}

	dui.Top.UI = &duit.Image{
		Image: readImagePath(args[0]),
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

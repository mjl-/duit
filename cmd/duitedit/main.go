package main

import (
	"flag"
	"io"
	"log"
	"os"

	"mjl/duit"
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		log.Println("usage: duitedit file")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	f, err := os.Open(args[0])
	check(err, "open")

	dui, err := duit.NewDUI("page", "800x600@0,0")
	check(err, "new dui")

	edit := duit.NewEdit(f)

	print := &duit.Button{
		Text: "print",
		Click: func(r *duit.Result) {
			rd := edit.Reader()
			n, err := io.Copy(os.Stdout, rd)
			if err != nil {
				log.Printf("error copying text: %s\n", err)
			}
			log.Printf("copied %d bytes\n", n)
		},
	}

	dui.Top = &duit.Box{Kids: duit.NewKids(print, edit)}
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)
		}
	}
}

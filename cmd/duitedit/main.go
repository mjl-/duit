package main

import (
	"flag"
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

	dui, err := duit.NewDUI("page", "800x600")
	check(err, "new dui")

	dui.Top = duit.NewEdit(f)
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)
		}
	}
}

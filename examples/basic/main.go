package main

import (
	"image"
	"log"
	"time"

	"github.com/mjl-/duit"
)

func main() {
	// create a new "developer ui"; this opens a new window.
	// "ex/basic" is the window title. nil are options.
	dui, err := duit.NewDUI("ex/basic", nil)
	if err != nil {
		log.Fatalf("new duit: %s\n", err)
	}

	// assign some UIs to variables, so we can reference
	// and modify them in the button click handler.
	status := &duit.Label{Text: "status: not logged in yet"}
	username := &duit.Field{}
	password := &duit.Field{Password: true}
	login := &duit.Button{
		Text:     "Login",
		Colorset: &dui.Primary,
		Click: func() (e duit.Event) {
			// click is called from the "main loop",
			// which is the for-loop at the end of main.
			// it is safe to modify the ui state from here.
			// be sure to call MarkLayout or MarkDraw after changing a UI.
			status.Text = "status: logging in ..."
			dui.MarkLayout(status)

			go func() {
				// pretend to do a slow web api call
				time.Sleep(1 * time.Second)

				// this goroutine is not executed from the main loop.
				// so it's not safe to modify UI state.
				// we send a closure that modifies state on dui.Call,
				// which will pass it on to dui.Input, which our main loop
				// oop picks up and runs.
				dui.Call <- func() {
					status.Text = "status: logged in"
					password.Text = ""
					dui.MarkLayout(nil) // we're lazy, update all of the UI
				}
			}()
			return
		},
	}

	// dui.Top.UI is the top-level UI drawn on the dui's window
	dui.Top.UI = &duit.Box{
		Padding: duit.SpaceXY(6, 4), // inset from the window
		Margin:  image.Pt(6, 4),     // space between kids in this box
		// duit.NewKids is a convenience function turning UIs into Kids
		Kids: duit.NewKids(
			status,
			&duit.Grid{
				Columns: 2,
				Padding: []duit.Space{
					{Right: 6, Top: 4, Bottom: 4},
					{Left: 6, Top: 4, Bottom: 4},
				},
				Valign: []duit.Valign{duit.ValignMiddle, duit.ValignMiddle},
				Kids: duit.NewKids(
					&duit.Label{Text: "username"},
					username,
					&duit.Label{Text: "password"},
					password,
				),
			},
			login,
		),
	}
	// first time the entire UI is drawn
	dui.Render()

	// our main loop
	for {
		// where we listen on two channels
		select {
		case e := <-dui.Inputs:
			// inputs are: mouse events, keyboard events, window resize events,
			// functions to call, recoverable errors
			dui.Input(e)

		case warn, ok := <-dui.Error:
			// on window close (clicking the X in the top corner),
			// the channel is closed and the application should quit.
			// otherwise, err is a warning or recoverable error.
			if !ok {
				return
			}
			log.Printf("duit: %s\n", warn)
		}
	}
}

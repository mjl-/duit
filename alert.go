package duit

import (
	"fmt"
	"image"
)

// Alert creates a new window that show text and a button labeled OK that closes the window.
func Alert(text string) (err error) {
	stop := make(chan struct{}, 1)

	dui, err := NewDUI("alert", &DUIOpts{Dimensions: "300x200"})
	if err != nil {
		return fmt.Errorf("new alert window: %s", err)
	}

	dui.Top.UI = NewMiddle(SpaceXY(20, 10),
		&Box{
			Margin: image.Pt(0, 10),
			Kids: NewKids(
				&Box{
					Width: -1,
					Kids: NewKids(
						&Label{Text: text},
					),
				},
				CenterUI(Space{},
					&Button{
						Colorset: &dui.Primary,
						Text:     "OK",
						Click: func() (e Event) {
							stop <- struct{}{}
							return
						},
					},
				),
			),
		},
	)
	dui.Render()

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)

		case xerr, ok := <-dui.Error:
			if !ok {
				return
			}
			dui.Close()
			return xerr

		case <-stop:
			dui.Close()
			return
		}
	}
}

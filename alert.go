package duit

import (
	"fmt"
	"image"
)

func Alert(s string) (err error) {
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
						&Label{Text: s},
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

		case err := <-dui.Error:
			if err != nil {
				dui.Close()
			}
			return err

		case <-stop:
			dui.Close()
			return
		}
	}
}

package duit

import (
	"image"
)

func Alert(s string) {
	stop := make(chan struct{}, 1)

	dui, err := NewDUI("alert", "300x200")
	check(err, "alert")

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

		case <-dui.Done:
			return

		case <-stop:
			dui.Close()
			return
		}
	}
}

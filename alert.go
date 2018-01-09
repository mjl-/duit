package duit

import ()

func Alert(s string) {
	stop := make(chan struct{}, 1)

	dui, err := NewDUI("alert", "300x200")
	check(err, "alert")

	dui.Top.UI = NewMiddle(
		&Box{
			Kids: NewKids(
				&Box{
					Width:   -1,
					Padding: SpaceXY(20, 10),
					Kids: NewKids(
						&Label{Text: s},
					),
				},
				CenterUI(SpaceXY(20, 10),
					&Button{
						Colorset: &dui.Primary,
						Text:     "OK",
						Click: func(_ *Result, _, _ *State) {
							stop <- struct{}{}
						},
					},
				),
			),
		},
	)
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)

		case <-dui.Done:
			return

		case <-stop:
			dui.Close()
			return
		}
	}
}

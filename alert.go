package duit

import ()

func Alert(s string) {
	stop := make(chan struct{}, 1)

	dui, err := NewDUI("alert", "300x200")
	check(err, "alert")

	dui.Top = &Box{
		Kids: []*Kid{
			{UI: &Label{Text: s}},
			{UI: &Button{
				Text: "OK",
				Click: func(nil *Result) {
					stop <- struct{}{}
				},
			}},
		},
	}
	dui.Render()

	for {
		select {
		case e := <-dui.Events:
			dui.Event(e)

		case <-stop:
			dui.Close()
			return
		}
	}
}

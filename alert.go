package duit

import ()

func Alert(s string) {
	stop := make(chan struct{}, 1)

	dui, err := New("alert", "300x200")
	check(err, "alert")

	dui.Top = &Box{
		Kids: []*Kid{
			{UI: &Label{Text: s}},
			{UI: &Button{
				Text: "OK",
				Click: func() {
					stop <- struct{}{}
				},
			}},
		},
	}
	dui.Render()

	for {
		select {
		case m := <-dui.Mousectl.C:
			dui.Mouse(m)
		case <-dui.Mousectl.Resize:
			dui.Resize()
		case r := <-dui.Kbdctl.C:
			dui.Key(r)

		case <-stop:
			dui.Display.Close()
			return
		}
	}
}

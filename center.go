package duit

func CenterUI(space Space, ui UI) UI {
	return &Grid{
		Columns: 1,
		Padding: []Space{space},
		Halign:  []Halign{HalignMiddle},
		Kids:    NewKids(ui),
		Width:   -1,
	}
}

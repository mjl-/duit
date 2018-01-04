package duit

func CenterUI(ui UI, space Space) *Grid {
	return &Grid{
		Columns: 1,
		Padding: []Space{space},
		Halign:  []Halign{HalignMiddle},
		Kids:    NewKids(ui),
		Width:   -1,
	}
}

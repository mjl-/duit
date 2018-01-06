package duit

import (
	"fmt"
	"image"
)

// tabs just stores the centered buttongroup and active UI in a box and lets that handle all the UI interface calls

// Tabs has a Buttongroup and displays only the active selected UI.
type Tabs struct {
	Buttongroup *Buttongroup
	UIs         []UI
	Box
}

var _ UI = &Tabs{}

// ensure Box is set up properly
func (ui *Tabs) ensure() {
	if ui.Box.Kids == nil {
		if len(ui.UIs) != len(ui.Buttongroup.Texts) {
			panic(fmt.Sprintf("bad Tabs, len(UIs) = %d must be equal to len(ui.Buttongroup.Texts) %d", len(ui.UIs), len(ui.Buttongroup.Texts)))
		}
		ui.Box.Kids = NewKids(CenterUI(SpaceXY(4, 4), ui.Buttongroup), ui.UIs[ui.Buttongroup.Selected])
		ui.Buttongroup.Changed = func(index int, r *Result) {
			ui.Box.Kids[1] = &Kid{UI: ui.UIs[index]}
			r.Consumed = true
			r.Layout = true
		}
	}
}

func (ui *Tabs) Layout(dui *DUI, sizeAvail image.Point) image.Point {
	ui.ensure()
	return ui.Box.Layout(dui, sizeAvail)
}

func (ui *Tabs) Print(indent int, r image.Rectangle) {
	PrintUI("Tabs", indent, r)
	PrintUI("Box", indent+1, r)
	kidsPrint(ui.Box.Kids, indent+2)
}

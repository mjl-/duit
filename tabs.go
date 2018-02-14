package duit

import (
	"fmt"
	"image"
)

// tabs just stores the centered buttongroup and active UI in a box and lets that handle all the UI interface calls

// Tabs has a Buttongroup and displays only the active selected UI.
type Tabs struct {
	Buttongroup *Buttongroup // Shown at top of Tabs.
	UIs         []UI         // UIs selected by Buttongroup, must have same number of elements as buttons in Buttongroup.
	Box
}

var _ UI = &Tabs{}

// ensure Box is set up properly
func (ui *Tabs) ensure(dui *DUI) {
	if ui.Box.Kids == nil {
		if len(ui.UIs) != len(ui.Buttongroup.Texts) {
			panic(fmt.Sprintf("bad Tabs, len(UIs) = %d must be equal to len(ui.Buttongroup.Texts) %d", len(ui.UIs), len(ui.Buttongroup.Texts)))
		}
		ui.Box.Kids = NewKids(CenterUI(SpaceXY(4, 4), ui.Buttongroup), ui.UIs[ui.Buttongroup.Selected])
		ui.Buttongroup.Changed = func(index int) (e Event) {
			k := ui.Box.Kids[1]
			k.UI = ui.UIs[index]
			e.Consumed = true
			e.NeedLayout = true
			dui.MarkLayout(k.UI)
			return
		}
	}
}

func (ui *Tabs) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	ui.ensure(dui)
	ui.Box.Layout(dui, self, sizeAvail, force)
}

func (ui *Tabs) Print(self *Kid, indent int) {
	PrintUI("Tabs", self, indent)
	PrintUI("Box", self, indent+1)
	KidsPrint(ui.Box.Kids, indent+2)
}

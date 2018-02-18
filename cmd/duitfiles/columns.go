package main

import (
	"strings"

	"github.com/mjl-/duit"
)

type columnsUI struct {
	duit.Split
}

// Select "name" in the given column. Name can be empty.
// SelectName can cause columns are removed, a new one opened,
// or an empty selection in the column.
func (ui *columnsUI) selectName(col int, name string) {
	ui.Kids = ui.Kids[:col+1]
	colUI := ui.Kids[col].UI.(*columnUI)
	colUI.name = name
	for _, lv := range colUI.list.Values {
		lv.Selected = lv.Text == name
	}
	path := ui.composePath(col, name)
	pathLabel.Text = path
	dui.MarkLayout(pathLabel)
	if favUI.findFavorite(path) == nil {
		favUI.toggle.Text = "+"
	} else {
		favUI.toggle.Text = "-"
	}
	dui.MarkLayout(favUI.toggle)
	if !strings.HasSuffix(path, "/") || name == "" {
		// not a dir, nothing to do for file selection, or no new column to show
		return
	}
	newColUI := newColumnUI(len(ui.Kids), name, listDir(path))
	ui.Kids = append(ui.Kids, &duit.Kid{UI: newColUI})
	dui.MarkLayout(ui)
	dui.Render()
	dui.Focus(newColUI.field)
}

// Compose path by combining "name" in the column "col".
func (ui *columnsUI) composePath(col int, name string) string {
	path := favUI.active.Value.(string)
	for _, colK := range ui.Kids[:col] {
		path += colK.UI.(*columnUI).name
	}
	path += name
	return path
}

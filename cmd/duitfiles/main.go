package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"os/exec"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

// Favorites on the left, fixed size. Remainder on the right contains one or more listboxes.
// Favorites is populated with some names that point to dirs. Clicking makes the favorite active, and focuses on first column.
// Typing then filters only the matching elements.  We just show text. Names ending in "/" are directories.
// Hitting tab on a directory opens that dir, and moves focus there.
// Hitting "enter" on a file causes it to be plumbed (opened).
var (
	dui        *duit.DUI
	contentUI  duit.UI // contains either the columns with directories/files, or errorUI with an error message
	colsUI     *columnsUI
	errorLabel *duit.Label
	errorClear *duit.Button
	pathLabel  *duit.Label // at top of the window
	favUI      *favoritesUI
	bold       *draw.Font
)

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func uiError(err error, msg string) bool {
	if err == nil {
		return false
	}
	errorLabel.Text = fmt.Sprintf("%s: %s", msg, err)
	dui.MarkLayout(nil)
	dui.Render()
	dui.Focus(errorClear)
	return true
}

func clearError() {
	if errorLabel.Text == "" {
		return
	}
	errorLabel.Text = ""
	dui.MarkLayout(contentUI)
}

func open(path string) {
	// xxx should be per platform, might want to try plumbing first.
	err := exec.Command("open", path).Run()
	uiError(err, "open")
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("files: ")

	var err error
	dui, err = duit.NewDUI("files", nil)
	check(err, "new dui")

	favorites, err := loadFavorites()
	check(err, "loading favorites")

	favUI = newFavoritesUI(favorites)
	pathLabel = &duit.Label{Text: favUI.list.Values[0].Value.(string)}

	errorLabel = &duit.Label{}
	errorClear = &duit.Button{
		Text:     "clear",
		Colorset: &dui.Primary,
		Click: func() (e duit.Event) {
			clearError()
			return
		},
	}
	errorUI := duit.NewMiddle(duit.SpaceXY(duit.ScrollbarSize, duit.ScrollbarSize), &duit.Box{
		Margin: image.Pt(6, 4),
		Kids:   duit.NewKids(errorLabel, errorClear),
	})

	colsUI = &columnsUI{
		Split: duit.Split{
			Background: dui.Gutter,
			Gutter:     1,
			Split: func(width int) []int {
				widths := make([]int, len(colsUI.Kids))
				col := width / len(widths)
				for i := range widths {
					widths[i] = col
				}
				widths[len(widths)-1] = width - col*(len(widths)-1)
				return widths
			},
			Kids: duit.NewKids(newColumnUI(0, "", listDir(pathLabel.Text))),
		},
	}

	bold, _ = dui.Display.OpenFont(os.Getenv("fontbold"))

	contentUI = &duit.Pick{
		Pick: func(_ image.Point) duit.UI {
			if errorLabel.Text != "" {
				return errorUI
			}
			return colsUI
		},
	}

	dui.Top.UI = &duit.Box{
		Kids: duit.NewKids(
			&duit.Split{
				Gutter:     1,
				Background: dui.Gutter,
				Split: func(width int) []int {
					return []int{dui.Scale(200), width - dui.Scale(200)}
				},
				Kids: duit.NewKids(
					favUI,
					&duit.Box{
						Height: -1,
						Valign: duit.ValignMiddle,
						Kids: duit.NewKids(
							&duit.Box{
								Padding: duit.Space{Left: duit.ScrollbarSize, Top: 4, Bottom: 4},
								Kids:    duit.NewKids(pathLabel),
							},
							contentUI,
						),
					},
				),
			},
		),
	}
	dui.Render()
	dui.Focus(colsUI.Kids[0].UI.(*columnUI).field)

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)

		case err, ok := <-dui.Error:
			if !ok {
				return
			}
			log.Printf("duit: %s\n", err)
		}
	}
}

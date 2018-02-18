package main

import (
	"bufio"
	"fmt"
	"image"
	"log"
	"os"
	"path"
	"strings"

	"github.com/mjl-/duit"
)

type favoritesUI struct {
	active *duit.ListValue
	list   *duit.List
	toggle *duit.Button
	duit.Box
}

func newFavoritesUI(favorites []string) (ui *favoritesUI) {
	ui = &favoritesUI{}
	values := make([]*duit.ListValue, len(favorites))
	for i, p := range favorites {
		values[i] = &duit.ListValue{
			Selected: i == 0,
			Text:     path.Base(p),
			Value:    p,
		}
	}
	ui.list = &duit.List{
		Values: values,
		Changed: func(index int) (e duit.Event) {
			clearError()
			ui.active = ui.list.Values[index]
			ui.active.Selected = true
			path := ui.active.Value.(string)
			pathLabel.Text = path
			ui.toggle.Text = "-"
			colsUI.Kids = duit.NewKids(newColumnUI(0, "", listDir(path)))
			e.NeedDraw = true
			dui.MarkLayout(colsUI)
			dui.MarkLayout(pathLabel)
			return
		},
	}
	ui.active = ui.list.Values[0]

	ui.toggle = &duit.Button{
		Text: "-",
		Click: func() (e duit.Event) {
			clearError()
			for _, lv := range ui.list.Values {
				lv.Selected = false
			}
			lv := ui.findFavorite(pathLabel.Text)
			if lv == ui.list.Values[0] || lv == ui.list.Values[1] {
				return
			}
			if lv == nil {
				lv = &duit.ListValue{
					Text:     path.Base(pathLabel.Text),
					Value:    pathLabel.Text,
					Selected: true,
				}
				ui.list.Values = append(ui.list.Values, lv)
				ui.toggle.Text = "-"
			} else {
				var nl []*duit.ListValue
				for _, lv := range ui.list.Values {
					if lv.Value.(string) != pathLabel.Text {
						nl = append(nl, lv)
					}
				}
				ui.list.Values = nl
				ui.toggle.Text = "+"
			}
			favs := make([]string, len(ui.list.Values)-2)
			for i, lv := range ui.list.Values[2:] {
				favs[i] = lv.Value.(string)
			}
			err := saveFavorites(favs)
			if err != nil {
				log.Printf("saving favorites: %s\n", err)
			}
			dui.MarkDraw(ui.list)
			e.NeedLayout = true
			return
		},
	}

	ui.Box = duit.Box{
		Height: -1,
		Kids: duit.NewKids(
			&duit.Box{
				Padding: duit.Space{Left: duit.ScrollbarSize, Top: 4, Bottom: 4},
				Margin:  image.Pt(6, 4),
				Kids: duit.NewKids(
					&duit.Label{
						Text: "Favorites",
						Font: bold,
					},
					ui.toggle,
				),
			},
			&duit.Scroll{
				Height: -1,
				Kid: duit.Kid{
					UI: &duit.Box{
						Height: -1,
						Kids:   duit.NewKids(ui.list),
					},
				},
			},
		),
	}
	return
}

func (ui *favoritesUI) findFavorite(path string) *duit.ListValue {
	for _, lv := range ui.list.Values {
		if lv.Value.(string) == path {
			return lv
		}
	}
	return nil
}

func favoritesPath() string {
	r := duit.AppDataDir("duitfiles") + "/favorites"
	return r
}

func loadFavorites() ([]string, error) {
	l := []string{}
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("home")
	}
	if home != "" {
		home += "/"
		l = append(l, home)
	}
	l = append(l, "/")

	f, err := os.Open(favoritesPath())
	if os.IsNotExist(err) {
		return l, nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		name := scanner.Text()
		l = append(l, name)
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func saveFavorites(l []string) (err error) {
	favPath := favoritesPath()
	os.MkdirAll(path.Dir(favPath), 0777)
	f, err := os.Create(favPath)
	if err != nil {
		return err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()
	_, err = fmt.Fprintln(f, strings.Join(l, "\n"))
	if err != nil {
		log.Printf("writing favorites: %s\n", err)
	}
	err = f.Close()
	if err != nil {
		log.Printf("closing favorites file: %s\n", err)
	}
	f = nil
	return
}

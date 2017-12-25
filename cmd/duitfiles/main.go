package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"9fans.net/go/draw"
	"mjl/duit"
)

type column struct {
	name  string
	names []string
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func open(path string) {
	log.Printf("open %s\n", path)
	// xxx should be per platform, might want to try plumbing first.
	err := exec.Command("open", path).Run()
	if err != nil {
		log.Printf("open: %s\n", err)
	}
}

func favoritesPath() string {
	return os.Getenv("HOME") + "/lib/duit/files/favorites"
}

func loadFavorites() ([]*duit.ListValue, error) {
	home := os.Getenv("HOME") + "/"
	l := []*duit.ListValue{
		{Label: "home", Value: home, Selected: true},
		{Label: "/", Value: "/"},
	}

	f, err := os.Open(favoritesPath())
	if os.IsNotExist(err) {
		return l, nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		name := scanner.Text()
		l = append(l, &duit.ListValue{Label: path.Base(name), Value: name})
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func saveFavorites(l []*duit.ListValue) (err error) {
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
	for _, lv := range l[2:] {
		_, err = fmt.Fprintln(f, lv.Value.(string))
		if err != nil {
			return
		}
	}
	err = f.Close()
	f = nil
	return
}

func listDir(path string) []string {
	files, err := ioutil.ReadDir(path)
	check(err, "readdir")
	names := make([]string, len(files))
	for i, fi := range files {
		names[i] = fi.Name()
		if fi.IsDir() {
			names[i] += "/"
		}
	}
	return names
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("files: ")

	dui, err := duit.NewDUI("files", "1200x700")
	check(err, "new ui")

	redraw := make(chan struct{}, 1)
	layout := make(chan struct{}, 1)

	// layout: favorites on the left, fixed size. remainder on the right contains one or more listboxes.
	// favorites is populated with some names that point to dirs. clicking makes the favorite active, and focuses on first column.
	// typing then filters only the matching elements.  we just show text. names ending in "/" are directories.
	// hitting tab on a directory opens that dir, and moves focus there.
	// hitting "enter" on a file causes it to be plumbed (opened).

	var (
		selectName     func(int, string)
		composePath    func(int, string) string
		columnsUI      *duit.Horizontal
		favoritesUI    *duit.List
		favoriteToggle *duit.Button
		activeFavorite *duit.ListValue
		makeColumnUI   func(colIndex int, c column) duit.UI
	)

	favorites, err := loadFavorites()
	check(err, "loading favorites")

	columns := []column{
		{names: listDir(favorites[0].Value.(string))},
	}
	pathLabel := &duit.Label{Text: favorites[0].Value.(string)}

	favoritesUI = &duit.List{
		Values: favorites,
		Changed: func(index int, r *duit.Result) {
			activeFavorite = favoritesUI.Values[index]
			activeFavorite.Selected = true
			path := activeFavorite.Value.(string)
			pathLabel.Text = path
			favoriteToggle.Text = "-"
			columns = []column{
				{name: "", names: listDir(path)},
			}
			columnsUI.Kids = duit.NewKids(makeColumnUI(0, columns[0]))
			r.Layout = true
		},
	}
	activeFavorite = favoritesUI.Values[0]

	findFavorite := func(path string) *duit.ListValue {
		for _, lv := range favoritesUI.Values {
			if lv.Value.(string) == path {
				return lv
			}
		}
		return nil
	}

	favoriteToggle = &duit.Button{
		Text: "-",
		Click: func(r *duit.Result) {
			log.Printf("toggle favorite\n")
			for _, lv := range favoritesUI.Values {
				lv.Selected = false
			}
			lv := findFavorite(pathLabel.Text)
			if lv == favoritesUI.Values[0] {
				return
			}
			if lv == nil {
				lv = &duit.ListValue{
					Label:    path.Base(pathLabel.Text),
					Value:    pathLabel.Text,
					Selected: true,
				}
				favoritesUI.Values = append(favoritesUI.Values, lv)
			} else {
				var nl []*duit.ListValue
				for _, lv := range favoritesUI.Values {
					if lv.Value.(string) != pathLabel.Text {
						nl = append(nl, lv)
					}
				}
				favoritesUI.Values = nl
			}
			err := saveFavorites(favoritesUI.Values)
			check(err, "saving favorites")
			r.Layout = true
		},
	}

	makeColumnUI = func(colIndex int, c column) duit.UI {
		l := make([]*duit.ListValue, len(c.names))
		for i, name := range c.names {
			l[i] = &duit.ListValue{Label: name, Value: name}
		}
		var list *duit.List
		list = &duit.List{
			Values: l,
			Changed: func(index int, result *duit.Result) {
				if list.Values[index].Selected {
					selectName(colIndex, list.Values[index].Value.(string))
					result.Layout = true
				} else {
					selectName(colIndex, "")
				}
			},
			Click: func(index, buttons int, r *duit.Result) {
				if buttons != 1<<2 {
					return
				}
				path := composePath(colIndex, list.Values[index].Value.(string))
				open(path)
				r.Consumed = true
			},
			Keys: func(index int, m draw.Mouse, k rune, r *duit.Result) {
				log.Printf("list.keys, k %x %c %v\n", k, k, k)
				switch k {
				case '\n':
					r.Consumed = true
					path := composePath(colIndex, list.Values[index].Value.(string))
					open(path)
				case draw.KeyLeft:
					r.Consumed = true
					if colIndex > 0 {
						selectName(colIndex-1, "")
						r.Layout = true
					} else {
						selectName(colIndex, "")
						r.Redraw = true
					}
				case draw.KeyRight:
					elem := list.Values[index].Value.(string)
					log.Printf("arrow right, index %d, elem %s\n", index, elem)
					if strings.HasSuffix(elem, "/") {
						r.Consumed = true
						selectName(colIndex, elem)
						if len(columns[colIndex+1].names) > 0 {
							log.Printf("selecting next first in new column\n")
							selectName(colIndex+1, columns[colIndex+1].names[0])
						}
						dui.Render()
						newList := columnsUI.Kids[len(columnsUI.Kids)-1].UI.(*duit.Box).Kids[1].UI.(*duit.Scroll).Child
						dui.Focus(newList)
						r.Layout = true
					}
				}
			},
		}
		return duit.NewBox(
			&duit.Field{
				Changed: func(newValue string, result *duit.Result) {
					nl := []*duit.ListValue{}
					exactMatch := false
					for _, name := range c.names {
						exactMatch = exactMatch || name == newValue
						if strings.Contains(name, newValue) {
							nl = append(nl, &duit.ListValue{Label: name, Value: name})
						}
					}
					if exactMatch {
						selectName(colIndex, newValue)
						dui.Render()
						field := columnsUI.Kids[len(columnsUI.Kids)-1].UI.(*duit.Box).Kids[0].UI.(*duit.Field)
						dui.Focus(field)
					}
					list.Values = nl
					result.Layout = true
				},
			},
			duit.NewScroll(list),
		)
	}

	columnsUI = &duit.Horizontal{
		Split: func(width int) []int {
			widths := make([]int, len(columns))
			col := width / len(widths)
			for i := range widths {
				widths[i] = col
			}
			widths[len(widths)-1] = width - col*(len(widths)-1)
			// xxx should layout more dynamically.  taking max of what is needed and what is available. and giving more to column of focus.  might need horizontal scroll too.
			return widths
		},
		Kids: duit.NewKids(makeColumnUI(0, columns[0])),
	}

	composePath = func(col int, name string) string {
		path := activeFavorite.Value.(string)
		for _, column := range columns[:col] {
			path += column.name
		}
		path += name
		return path
	}

	selectName = func(col int, name string) {
		log.Printf("selectName col %d, name %s\n", col, name)
		path := activeFavorite.Value.(string)
		columns = columns[:col+1]
		columns[col].name = name
		columnsUI.Kids = columnsUI.Kids[:col+1]
		for _, column := range columns {
			path += column.name
		}
		pathLabel.Text = path
		if findFavorite(path) == nil {
			favoriteToggle.Text = "+"
		} else {
			favoriteToggle.Text = "-"
		}
		if !strings.HasSuffix(path, "/") {
			// not a dir, nothing to do for file selection
			return
		}
		names := listDir(path)
		if name == "" {
			// no new column to show
			return
		}
		newCol := column{name: name, names: names}
		columns = append(columns, newCol)
		columnsUI.Kids = append(columnsUI.Kids, &duit.Kid{UI: makeColumnUI(len(columns)-1, newCol)})
	}

	dui.Top = duit.NewBox(
		favoriteToggle,
		pathLabel,
		&duit.Horizontal{
			Split: func(width int) []int {
				return []int{dui.Scale(200), width - dui.Scale(200)}
			},
			Kids: duit.NewKids(favoritesUI, columnsUI),
		},
	)
	dui.Render()
	dui.Focus(columnsUI.Kids[0].UI.(*duit.Box).Kids[0].UI.(*duit.Field))

	for {
		select {
		case m := <-dui.Mousectl.C:
			dui.Mouse(m)
		case <-dui.Mousectl.Resize:
			dui.Resize()
		case r := <-dui.Kbdctl.C:
			dui.Key(r)

		case <-redraw:
			dui.Redraw()
		case <-layout:
			dui.Render()
		}
	}
}

package main

import (
	"fmt"
	"image"
	imagedraw "image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"time"

	"9fans.net/go/draw"
)

const (
	Margin  = 10
	Padding = 10
	Border  = 1
	Space   = Margin + Border + Padding

	ScrollbarWidth = 15

	WheelUp   = 0xA
	WheelDown = 0xFFFFFFFE

	Fn1 = 0xf001

	ArrowUp   = 0xf00e
	ArrowDown = 0x80
	PageUp    = 0xf00f
	PageDown  = 0xf013
)

type Result struct {
	Hit      UI           // the UI where the event ended up
	Consumed bool         // whether event was consumed, and should not be further handled by upper UI's
	Redraw   bool         // whether event needs a redraw after
	Warp     *image.Point // if set, mouse will warp to location
}

type UI interface {
	Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point
	Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse)
	Mouse(m draw.Mouse) (result Result)
	Key(orig image.Point, m draw.Mouse, k rune) (result Result)

	// FirstFocus returns the top-left corner where the focus should go next when "tab" is hit, if anything.
	FirstFocus() *image.Point
}

type Label struct {
	Text string
}

func (ui *Label) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2*Margin + 2*Border, 2 * Space})
}
func (ui *Label) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.String(orig.Add(image.Point{Margin + Border, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Label) Mouse(m draw.Mouse) Result {
	return Result{ui, false, false, nil}
}
func (ui *Label) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{ui, false, false, nil}
}
func (ui *Label) FirstFocus() *image.Point {
	return nil
}

type Field struct {
	Text string

	size image.Point // including space
}

func (ui *Field) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	ui.size = image.Point{r.Dx(), 2*Space + display.DefaultFont.Height}
	return ui.size
}
func (ui *Field) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	hover := m.In(image.Rectangle{image.ZP, ui.size})
	r := image.Rectangle{orig, orig.Add(ui.size)}
	img.Draw(r, display.White, nil, image.ZP)

	color := display.Black
	if hover {
		var err error
		color, err = display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, draw.Blue)
		check(err, "allocimage")
	}
	img.Border(
		image.Rectangle{
			orig.Add(image.Point{Margin, Margin}),
			orig.Add(ui.size).Sub(image.Point{Margin, Margin}),
		},
		1, color, image.ZP)
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Field) Mouse(m draw.Mouse) Result {
	return Result{ui, false, false, nil}
}
func (ui *Field) Key(orig image.Point, m draw.Mouse, c rune) Result {
	switch c {
	case PageUp, PageDown, ArrowUp, ArrowDown:
		return Result{ui, false, false, nil}
	case '\t':
		return Result{ui, false, false, nil}
	case 8:
		if ui.Text != "" {
			ui.Text = ui.Text[:len(ui.Text)-1]
		}
	default:
		ui.Text += string(c)
	}
	return Result{ui, true, true, nil}
}
func (ui *Field) FirstFocus() *image.Point {
	p := image.Pt(Space, Space)
	return &p
}

type Button struct {
	Text  string
	Click func()

	m draw.Mouse
}

func (ui *Button) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	return display.DefaultFont.StringSize(ui.Text).Add(image.Point{2 * Space, 2 * Space})
}
func (ui *Button) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	size := display.DefaultFont.StringSize(ui.Text)

	grey, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, draw.Palegreygreen)
	check(err, "allocimage grey")

	r := image.Rectangle{
		orig.Add(image.Point{Margin + Border, Margin + Border}),
		orig.Add(size).Add(image.Point{2*Padding + Margin + Border, 2*Padding + Margin + Border}),
	}
	hover := m.In(image.Rectangle{image.ZP, size.Add(image.Pt(2*Space, 2*Space))})
	borderColor := grey
	if hover {
		borderColor = display.Black
	}
	img.Draw(r, grey, nil, image.ZP)
	img.Border(image.Rectangle{orig.Add(image.Point{Margin, Margin}), orig.Add(size).Add(image.Point{Margin + 2*Padding + 2*Border, Margin + 2*Padding + 2*Border})}, 1, borderColor, image.ZP)
	img.String(orig.Add(image.Point{Space, Space}), display.Black, image.ZP, display.DefaultFont, ui.Text)
}
func (ui *Button) Mouse(m draw.Mouse) Result {
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 && ui.Click != nil {
		ui.Click()
	}
	ui.m = m
	return Result{ui, false, false, nil}
}
func (ui *Button) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{ui, false, false, nil}
}
func (ui *Button) FirstFocus() *image.Point {
	p := image.Pt(Space, Space)
	return &p
}

type Image struct {
	Image *draw.Image
}

func (ui *Image) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	return ui.Image.R.Size()
}
func (ui *Image) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(image.Rectangle{orig, orig.Add(ui.Image.R.Size())}, ui.Image, nil, image.ZP)
}
func (ui *Image) Mouse(m draw.Mouse) Result {
	return Result{ui, false, false, nil}
}
func (ui *Image) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return Result{ui, false, false, nil}
}
func (ui *Image) FirstFocus() *image.Point {
	return nil
}

type Kid struct {
	UI UI

	r image.Rectangle
}

// box keeps elements on a line as long as they fit
type Box struct {
	Kids []*Kid

	size image.Point
}

func (ui *Box) Layout(display *draw.Display, r image.Rectangle, ocur image.Point) image.Point {
	xmax := 0
	cur := image.Point{0, 0}
	nx := 0    // number on current line
	liney := 0 // max y of current line
	for _, k := range ui.Kids {
		p := k.UI.Layout(display, r, cur)
		var kr image.Rectangle
		if nx == 0 || cur.X+p.X <= r.Dx() {
			kr = image.Rectangle{cur, cur.Add(p)}
			cur.X += p.X
			if p.Y > liney {
				liney = p.Y
			}
			nx += 1
		} else {
			cur.X = 0
			cur.Y += liney
			kr = image.Rectangle{cur, cur.Add(p)}
			nx = 1
			cur.X = p.X
			liney = p.Y
		}
		k.r = kr
		if xmax < cur.X {
			xmax = cur.X
		}
	}
	if len(ui.Kids) > 0 {
		cur.Y += liney
	}
	ui.size = image.Point{xmax, cur.Y}
	return ui.size
}
func (ui *Box) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(display, ui.Kids, ui.size, img, orig, m)
}
func (ui *Box) Mouse(m draw.Mouse) Result {
	return kidsMouse(ui.Kids, m)
}
func (ui *Box) Key(orig image.Point, m draw.Mouse, c rune) Result {
	return kidsKey(ui, ui.Kids, orig, m, c)
}
func (ui *Box) FirstFocus() *image.Point {
	return kidsFirstFocus(ui.Kids)
}

func kidsDraw(display *draw.Display, kids []*Kid, uiSize image.Point, img *draw.Image, orig image.Point, m draw.Mouse) {
	img.Draw(image.Rectangle{orig, orig.Add(uiSize)}, display.White, nil, image.ZP)
	for _, k := range kids {
		mm := m
		mm.Point = mm.Point.Sub(k.r.Min)
		k.UI.Draw(display, img, orig.Add(k.r.Min), mm)
	}
}

func kidsMouse(kids []*Kid, m draw.Mouse) Result {
	for _, k := range kids {
		if m.Point.In(k.r) {
			m.Point = m.Point.Sub(k.r.Min)
			return k.UI.Mouse(m)
		}
	}
	return Result{nil, false, false, nil}
}

func kidsKey(ui UI, kids []*Kid, orig image.Point, m draw.Mouse, c rune) Result {
	for i, k := range kids {
		if m.Point.In(k.r) {
			m.Point = m.Point.Sub(k.r.Min)
			r := k.UI.Key(orig.Add(k.r.Min), m, c)
			if !r.Consumed && c == '\t' {
				for next := i + 1; next < len(kids); next++ {
					first := kids[next].UI.FirstFocus()
					if first != nil {
						kR := kids[next].r
						p := first.Add(orig).Add(kR.Min)
						r.Warp = &p
						r.Consumed = true
						break
					}
				}
			}
			return r
		}
	}
	return Result{ui, false, false, nil}
}

func kidsFirstFocus(kids []*Kid) *image.Point {
	if len(kids) == 0 {
		return nil
	}
	for _, k := range kids {
		first := k.UI.FirstFocus()
		if first != nil {
			p := first.Add(k.r.Min)
			return &p
		}
	}
	return nil
}

// Scroll shows a part of its single child, typically a box, and lets you scroll the content.
type Scroll struct {
	Child UI

	r         image.Rectangle // entire ui
	barR      image.Rectangle
	childSize image.Point
	offset    int         // current scroll offset in pixels
	img       *draw.Image // for child to draw on
}

func (ui *Scroll) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	ui.r = image.Rect(r.Min.X, cur.Y, r.Max.X, r.Max.Y)
	ui.barR = image.Rectangle{ui.r.Min, image.Pt(ui.r.Min.X+ScrollbarWidth, ui.r.Max.Y)}
	ui.childSize = ui.Child.Layout(display, image.Rectangle{image.ZP, image.Pt(ui.r.Dx()-ui.barR.Dx(), ui.r.Dy())}, image.ZP)
	if ui.r.Dy() > ui.childSize.Y {
		ui.barR.Max.Y = ui.childSize.Y
		ui.r.Max.Y = ui.childSize.Y
	}
	var err error
	ui.img, err = display.AllocImage(image.Rectangle{image.ZP, ui.childSize}, draw.ARGB32, false, draw.White)
	check(err, "allocimage")
	return ui.r.Size()
}
func (ui *Scroll) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	// draw scrollbar
	lightGrey, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0xEEEEEEFF)
	check(err, "allowimage lightgrey")
	darkerGrey, err := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0xAAAAAAFF)
	check(err, "allowimage darkergrey")
	barR := ui.barR.Add(orig)
	img.Draw(barR, lightGrey, nil, image.ZP)
	barRActive := barR
	h := ui.r.Dy()
	uih := ui.childSize.Y
	if uih > h {
		barH := int((float32(h) / float32(uih)) * float32(h))
		barY := int((float32(ui.offset) / float32(uih)) * float32(h))
		barRActive.Min.Y += barY
		barRActive.Max.Y = barRActive.Min.Y + barH
	}
	img.Draw(barRActive, darkerGrey, nil, image.ZP)

	// draw child ui
	ui.img.Draw(ui.img.R, display.White, nil, image.ZP)
	m.Point = m.Point.Add(image.Pt(-ScrollbarWidth, ui.offset))
	ui.Child.Draw(display, ui.img, image.Pt(0, -ui.offset), m)
	img.Draw(ui.img.R.Add(orig).Add(image.Pt(ScrollbarWidth, 0)), ui.img, nil, image.ZP)
}
func (ui *Scroll) scroll(delta int) bool {
	o := ui.offset
	ui.offset += delta
	if ui.offset < 0 {
		ui.offset = 0
	}
	offsetMax := ui.childSize.Y - ui.r.Dy()
	if ui.offset > offsetMax {
		ui.offset = offsetMax
	}
	return o != ui.offset
}
func (ui *Scroll) scrollKey(c rune) (consumed bool) {
	switch c {
	case ArrowUp:
		return ui.scroll(-50)
	case ArrowDown:
		return ui.scroll(50)
	case PageUp:
		return ui.scroll(-200)
	case PageDown:
		return ui.scroll(200)
	}
	return false
}
func (ui *Scroll) scrollMouse(m draw.Mouse) (consumed bool) {
	switch m.Buttons {
	case WheelUp:
		return ui.scroll(-50)
	case WheelDown:
		return ui.scroll(50)
	}
	return false
}
func (ui *Scroll) Mouse(m draw.Mouse) Result {
	if m.Point.In(ui.barR) {
		consumed := ui.scrollMouse(m)
		redraw := consumed
		return Result{ui, consumed, redraw, nil}
	}
	if m.Point.In(ui.r) {
		m.Point = m.Point.Add(image.Pt(-ScrollbarWidth, ui.offset))
		r := ui.Child.Mouse(m)
		if !r.Consumed {
			r.Consumed = ui.scrollMouse(m)
			r.Redraw = r.Redraw || r.Consumed
		}
		return r
	}
	return Result{nil, false, false, nil}
}
func (ui *Scroll) Key(orig image.Point, m draw.Mouse, c rune) Result {
	if m.Point.In(ui.barR) {
		consumed := ui.scrollKey(c)
		redraw := consumed
		return Result{ui, consumed, redraw, nil}
	}
	if m.Point.In(ui.r) {
		m.Point = m.Point.Add(image.Pt(-ScrollbarWidth, ui.offset))
		r := ui.Child.Key(orig.Add(image.Pt(ScrollbarWidth, -ui.offset)), m, c)
		if !r.Consumed {
			r.Consumed = ui.scrollKey(c)
			r.Redraw = r.Redraw || r.Consumed
		}
		return r
	}
	return Result{nil, false, false, nil}
}
func (ui *Scroll) FirstFocus() *image.Point {
	first := ui.Child.FirstFocus()
	if first == nil {
		return nil
	}
	p := first.Add(image.Pt(ScrollbarWidth, -ui.offset))
	return &p
}

type ListValue struct {
	Label    string
	Value    interface{}
	Selected bool
}

type List struct {
	Values   []*ListValue
	Multiple bool

	lineHeight int
	size       image.Point
}

func (ui *List) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	font := display.DefaultFont
	ui.lineHeight = font.Height
	ui.size = image.Pt(r.Dx(), len(ui.Values)*ui.lineHeight)
	return ui.size
}
func (ui *List) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	font := display.DefaultFont
	r := image.Rectangle{orig, orig.Add(ui.size)}
	img.Draw(r, display.White, nil, image.ZP)
	cur := orig
	for _, v := range ui.Values {
		color := display.Black
		if v.Selected {
			img.Draw(image.Rectangle{cur, cur.Add(image.Pt(ui.size.X, font.Height))}, display.Black, nil, image.ZP)
			color = display.White
		}
		img.String(cur, color, image.ZP, font, v.Label)
		cur.Y += ui.lineHeight
	}
}
func (ui *List) Mouse(m draw.Mouse) (result Result) {
	result.Hit = ui
	if m.In(image.Rectangle{image.ZP, ui.size}) {
		index := m.Y / ui.lineHeight
		if m.Buttons == 1 {
			v := ui.Values[index]
			v.Selected = !v.Selected
			if v.Selected && !ui.Multiple {
				for _, vv := range ui.Values {
					if vv != v {
						vv.Selected = false
					}
				}
			}
			result.Redraw = true
			result.Consumed = true
		}
	}
	return
}
func (ui *List) Key(orig image.Point, m draw.Mouse, k rune) (result Result) {
	result.Hit = ui
	return
}
func (ui *List) FirstFocus() *image.Point {
	return &image.Point{Space, Space}
}

type Horizontal struct {
	Kids  []*Kid
	Split func(r image.Rectangle) (widths []int)

	size   image.Point
	widths []int
}

func (ui *Horizontal) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	r.Min = image.Pt(0, cur.Y)
	widths := ui.Split(r)
	if len(widths) != len(ui.Kids) {
		panic("bad number of widths from split")
	}
	ui.widths = widths
	ui.size = image.ZP
	for i, k := range ui.Kids {
		p := image.Pt(ui.size.X, 0)
		size := k.UI.Layout(display, image.Rectangle{p, image.Pt(widths[i], r.Dy())}, image.ZP)
		k.r = image.Rectangle{p, p.Add(size)}
		ui.size.X += widths[i]
		if k.r.Dy() > ui.size.Y {
			ui.size.Y = k.r.Dy()
		}
	}
	return ui.size
}
func (ui *Horizontal) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(display, ui.Kids, ui.size, img, orig, m)
}
func (ui *Horizontal) Mouse(m draw.Mouse) (result Result) {
	return kidsMouse(ui.Kids, m)
}
func (ui *Horizontal) Key(orig image.Point, m draw.Mouse, k rune) (result Result) {
	return kidsKey(ui, ui.Kids, orig, m, k)
}
func (ui *Horizontal) FirstFocus() *image.Point {
	return kidsFirstFocus(ui.Kids)
}

type Vertical struct {
	Kids  []*Kid
	Split func(r image.Rectangle) (heights []int)

	size    image.Point
	heights []int
}

func (ui *Vertical) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	r.Min = image.Pt(0, cur.Y)
	heights := ui.Split(r)
	if len(heights) != len(ui.Kids) {
		panic("bad number of heights from split")
	}
	ui.heights = heights
	ui.size = image.ZP
	for i, k := range ui.Kids {
		p := image.Pt(0, ui.size.Y)
		size := k.UI.Layout(display, image.Rectangle{p, image.Pt(r.Dx(), heights[i])}, image.ZP)
		k.r = image.Rectangle{p, p.Add(size)}
		ui.size.Y += heights[i]
	}
	ui.size.X = r.Dx()
	return ui.size
}
func (ui *Vertical) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(display, ui.Kids, ui.size, img, orig, m)
}
func (ui *Vertical) Mouse(m draw.Mouse) (result Result) {
	return kidsMouse(ui.Kids, m)
}
func (ui *Vertical) Key(orig image.Point, m draw.Mouse, k rune) (result Result) {
	return kidsKey(ui, ui.Kids, orig, m, k)
}
func (ui *Vertical) FirstFocus() *image.Point {
	return kidsFirstFocus(ui.Kids)
}

type Grid struct {
	Kids    []*Kid
	Columns int

	widths  []int
	heights []int
	size    image.Point
}

func (ui *Grid) Layout(display *draw.Display, r image.Rectangle, cur image.Point) image.Point {
	r.Min = image.Pt(0, cur.Y)

	ui.widths = make([]int, ui.Columns)
	width := 0
	for col := 0; col < ui.Columns; col++ {
		ui.widths[col] = 0
		for i := col; i < len(ui.Kids); i += ui.Columns {
			k := ui.Kids[i]
			kr := image.Rectangle{image.ZP, image.Pt(r.Dx()-width, r.Dy())}
			size := k.UI.Layout(display, kr, image.ZP)
			if size.X > ui.widths[col] {
				ui.widths[col] = size.X
			}
		}
		width += ui.widths[col]
	}

	ui.heights = make([]int, (len(ui.Kids)+ui.Columns-1)/ui.Columns)
	height := 0
	for i := 0; i < len(ui.Kids); i += ui.Columns {
		row := i / ui.Columns
		ui.heights[row] = 0
		for col := 0; col < ui.Columns; col++ {
			k := ui.Kids[i+col]
			kr := image.Rectangle{image.ZP, image.Pt(ui.widths[col], r.Dy())}
			size := k.UI.Layout(display, kr, image.ZP)
			if size.Y > ui.heights[row] {
				ui.heights[row] = size.Y
			}
		}
		height += ui.heights[row]
	}

	x := make([]int, len(ui.widths))
	for col := range x {
		if col > 0 {
			x[col] = x[col-1] + ui.widths[col-1]
		}
	}
	y := make([]int, len(ui.heights))
	for row := range y {
		if row > 0 {
			y[row] = y[row-1] + ui.heights[row-1]
		}
	}

	for i, k := range ui.Kids {
		row := i / ui.Columns
		col := i % ui.Columns
		p := image.Pt(x[col], y[row])
		k.r = image.Rectangle{p, p.Add(image.Pt(ui.widths[col], ui.heights[row]))}
	}

	ui.size = image.Pt(width, height)
	return ui.size
}
func (ui *Grid) Draw(display *draw.Display, img *draw.Image, orig image.Point, m draw.Mouse) {
	kidsDraw(display, ui.Kids, ui.size, img, orig, m)
}
func (ui *Grid) Mouse(m draw.Mouse) (result Result) {
	return kidsMouse(ui.Kids, m)
}
func (ui *Grid) Key(orig image.Point, m draw.Mouse, k rune) (result Result) {
	return kidsKey(ui, ui.Kids, orig, m, k)
}
func (ui *Grid) FirstFocus() *image.Point {
	return kidsFirstFocus(ui.Kids)
}

func NewBox(uis ...UI) *Box {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return &Box{Kids: kids}
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

func main() {
	display, err := draw.Init(nil, "", "duitex", "600x400")
	check(err, "draw init")
	screen := display.ScreenImage

	mousectl := display.InitMouse()
	kbdctl := display.InitKeyboard()
	//whitemask, _ := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0x7F7F7F7F)

	redraw := make(chan struct{}, 1)

	readImage := func(f io.Reader) *draw.Image {
		img, _, err := image.Decode(f)
		check(err, "decoding image")
		var rgba *image.RGBA
		switch i := img.(type) {
		case *image.RGBA:
			rgba = i
		default:
			b := img.Bounds()
			rgba = image.NewRGBA(image.Rectangle{image.ZP, b.Size()})
			imagedraw.Draw(rgba, rgba.Bounds(), img, b.Min, imagedraw.Src)
		}

		// todo: colors are wrong. it should be RGBA32, but that looks even worse.

		ni, err := display.AllocImage(rgba.Bounds(), draw.ARGB32, false, draw.White)
		check(err, "allocimage")
		_, err = ni.Load(rgba.Bounds(), rgba.Pix)
		check(err, "load image")
		return ni
	}

	readImagePath := func(path string) *draw.Image {
		f, err := os.Open(path)
		check(err, "open image")
		defer f.Close()
		return readImage(f)
	}

	count := 0
	counter := &Label{Text: fmt.Sprintf("%d", count)}

	var top UI = NewBox(
		&Vertical{
			Split: func(r image.Rectangle) []int {
				height := r.Dy()
				row1 := height / 4
				row2 := height / 4
				row3 := height - row1 - row2
				return []int{row1, row2, row3}
			},
			Kids: []*Kid{
				{UI: &Label{Text: "in row 1"}},
				{UI: &Scroll{
					Child: &Grid{
						Columns: 2,
						Kids: []*Kid{
							{UI: &Label{Text: "From"}},
							{UI: &Field{Text: "...from..."}},
							{UI: &Label{Text: "To"}},
							{UI: &Field{Text: "...to..."}},
							{UI: &Label{Text: "Cc"}},
							{UI: &Field{Text: "...cc..."}},
							{UI: &Label{Text: "Bcc"}},
							{UI: &Field{Text: "...bcc..."}},
							{UI: &Label{Text: "Subject"}},
							{UI: &Field{Text: "...subject..."}},
						},
					},
				}},
				{UI: &Scroll{
					Child: NewBox(
						&Label{Text: "counter:"},
						counter,
						&Button{Text: "button1", Click: func() { log.Printf("button clicked") }},
						&Button{Text: "button2"},
						&List{
							Multiple: true,
							Values: []*ListValue{
								{Label: "Elem 1", Value: 1},
								{Label: "Elem 2", Value: 2},
								{Label: "Elem 3", Value: 3},
							},
						},
						&Label{Text: "Horizontal split"},
						&Horizontal{
							Kids: []*Kid{
								{UI: &Label{Text: "in column 1"}},
								{UI: &Label{Text: "in column 2"}},
								{UI: &Label{Text: "in column 3"}},
							},
							Split: func(r image.Rectangle) []int {
								width := r.Dx()
								col1 := width / 4
								col2 := width / 4
								col3 := width - col1 - col2
								return []int{col1, col2, col3}
							},
						},
						&Label{Text: "Another box with a scrollbar:"},
						&Scroll{Child: NewBox(
							&Label{Text: "another label, this one is somewhat longer"},
							&Button{Text: "some other button"},
							&Label{Text: "more labels"},
							&Label{Text: "another"},
							&Field{Text: "A field!!"},
							NewBox(&Image{Image: readImagePath("test.jpg")}),
							&Field{Text: "A field!!"},
							NewBox(&Image{Image: readImagePath("test.jpg")}),
							&Field{Text: "A field!!"},
							NewBox(&Image{Image: readImagePath("test.jpg")}),
						)},
						&Button{Text: "button3"},
						&Field{Text: "field 2"},
						&Field{Text: "field 3"},
						&Field{Text: "field 4"},
						&Field{Text: "field 5"},
						&Field{Text: "field 6"},
						&Field{Text: "field 7"},
						&Label{Text: "this is a label"},
					),
				}},
			},
		})
	top.Layout(display, screen.R, image.ZP)
	top.Draw(display, screen, image.ZP, draw.Mouse{})
	display.Flush()

	tick := make(chan struct{}, 0)
	go func() {
		for {
			time.Sleep(1 * time.Second)
			tick <- struct{}{}
		}
	}()

	var mouse draw.Mouse
	logEvents := false
	var lastMouseUI UI
	for {
		select {
		case mouse = <-mousectl.C:
			if logEvents {
				log.Printf("mouse %v, %b\n", mouse, mouse.Buttons)
			}
			r := top.Mouse(mouse)
			if r.Hit != lastMouseUI || r.Redraw {
				top.Draw(display, screen, image.ZP, mouse)
				display.Flush()
			}
			lastMouseUI = r.Hit

		case <-mousectl.Resize:
			if logEvents {
				log.Printf("resize")
			}
			check(display.Attach(draw.Refmesg), "attach after resize")
			top.Layout(display, screen.R, image.ZP)
			top.Draw(display, screen, image.ZP, mouse)
			display.Flush()

		case r := <-kbdctl.C:
			if logEvents {
				log.Printf("kdb %c, %x\n", r, r)
			}
			if r == 0xf001 {
				logEvents = !logEvents
			}
			result := top.Key(image.ZP, mouse, r)
			if !result.Consumed && r == '\t' {
				first := top.FirstFocus()
				if first != nil {
					result.Warp = first
					result.Consumed = true
				}
			}
			if result.Warp != nil {
				err = display.MoveTo(*result.Warp)
				if err != nil {
					log.Printf("move mouse to %v: %v\n", result.Warp, err)
				}
				m := mouse
				m.Point = *result.Warp
				result2 := top.Mouse(m)
				result.Redraw = result.Redraw || result2.Redraw || true
				mouse = m
				lastMouseUI = result2.Hit
			}
			if result.Redraw {
				top.Draw(display, screen, image.ZP, mouse)
				display.Flush()
			}

		case <-redraw:
			top.Draw(display, screen, image.ZP, mouse)
			display.Flush()

		case <-tick:
			count++
			counter.Text = fmt.Sprintf("%d", count)
			top.Layout(display, screen.R, image.ZP)
			top.Draw(display, screen, image.ZP, mouse)
			display.Flush()
		}
	}
}

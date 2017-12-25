package duit

import (
	"image"
	"strings"

	"9fans.net/go/draw"
)

type Field struct {
	Text     string
	Disabled bool
	Cursor   int                                   // index in string of cursor, 0 is before first char
	Changed  func(string, *Result)                 // called after contents of field have changed
	Keys     func(m draw.Mouse, k rune, r *Result) // called before handling key. if you consume the event, Changed will not be called

	size image.Point // including space
	m    draw.Mouse
}

var _ UI = &Field{}

func (ui *Field) Layout(env *Env, size image.Point) image.Point {
	ui.size = image.Point{size.X, 2*env.Size.Space + env.Display.DefaultFont.Height}
	return ui.size
}

func (ui *Field) Draw(env *Env, img *draw.Image, orig image.Point, m draw.Mouse) {
	r := rect(ui.size)
	hover := m.In(r)
	r = r.Add(orig)

	colors := env.Normal
	if ui.Disabled {
		colors = env.Disabled
	} else if hover {
		colors = env.Hover
	}
	img.Draw(r, colors.Background, nil, image.ZP)
	drawRoundedBorder(img, r, colors.Border)
	img.String(orig.Add(pt(env.Size.Space)), colors.Text, image.ZP, env.Display.DefaultFont, ui.Text)

	if hover && !ui.Disabled {
		ui.fixCursor()
		f := env.Display.DefaultFont
		p0 := r.Min.Add(pt(env.Size.Space))
		p0.X += f.StringWidth(ui.Text[:ui.Cursor])
		p1 := p0
		p1.Y += f.Height
		img.Line(p0, p1, 1, 1, 0, env.Hover.Border, image.ZP)
	}
}

func (ui *Field) Mouse(env *Env, m draw.Mouse) (r Result) {
	if !m.In(rect(ui.size)) {
		return
	}
	r.Hit = ui
	if ui.m.Buttons&1 == 1 && m.Buttons&1 == 0 {
		f := env.Display.DefaultFont
		n := len(ui.Text)
		mX := m.X - env.Size.Space
		found := false
		for i := 0; i < n; i++ {
			x := f.StringWidth(ui.Text[:i])
			if mX <= x {
				ui.Cursor = i
				found = true
				break
			}
		}
		if !found {
			ui.Cursor = len(ui.Text)
		}
		r.Consumed = true
		r.Redraw = true
	}
	ui.m = m
	return
}

func (ui *Field) fixCursor() {
	if ui.Cursor < 0 {
		ui.Cursor = 0
	}
	if ui.Cursor > len(ui.Text) {
		ui.Cursor = len(ui.Text)
	}
}

func (ui *Field) Key(env *Env, orig image.Point, m draw.Mouse, k rune) (r Result) {
	if !m.In(rect(ui.size)) {
		return
	}
	r.Hit = ui
	if ui.Disabled {
		return
	}

	if ui.Keys != nil {
		ui.Keys(m, k, &r)
		if r.Consumed {
			return
		}
	}

	const Ctrl = 0x1f
	ui.fixCursor()
	switch k {
	case draw.KeyPageUp, draw.KeyPageDown, draw.KeyUp, draw.KeyDown, '\t':
		return Result{Hit: ui}
	case draw.KeyLeft:
		ui.Cursor--
	case draw.KeyRight:
		ui.Cursor++
	case Ctrl & 'a':
		ui.Cursor = 0
	case Ctrl & 'e':
		ui.Cursor = len(ui.Text)

	case Ctrl & 'h':
		// remove char before cursor
		if ui.Cursor > 0 {
			ui.Text = ui.Text[:ui.Cursor-1] + ui.Text[ui.Cursor:]
			ui.Cursor--
		}
	case Ctrl & 'w':
		// remove to start of space+word
		for ui.Cursor > 0 && strings.ContainsAny(ui.Text[ui.Cursor-1:ui.Cursor], " \t\r\n") {
			ui.Cursor--
		}
		for ui.Cursor > 0 && !strings.ContainsAny(ui.Text[ui.Cursor-1:ui.Cursor], " \t\r\n") {
			ui.Cursor--
		}
		ui.Text = ui.Text[:ui.Cursor]
	case Ctrl & 'u':
		// remove entire line
		ui.Text = ""
		ui.Cursor = 0
	case Ctrl & 'k':
		// remove to end of line
		ui.Text = ui.Text[ui.Cursor:]

	case '\n':
		return

	default:
		if ui.Cursor > len(ui.Text) {
			ui.Text += string(k)
		} else {
			ui.Text = ui.Text[:ui.Cursor] + string(k) + ui.Text[ui.Cursor:]
		}
		ui.Cursor += 1
	}
	ui.fixCursor()
	r.Consumed = true
	r.Redraw = true
	if ui.Changed != nil {
		ui.Changed(ui.Text, &r)
	}
	return
}

func (ui *Field) FirstFocus(env *Env) *image.Point {
	return &image.ZP
}

func (ui *Field) Focus(env *Env, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(env)
}

func (ui *Field) Print(indent int, r image.Rectangle) {
	uiPrint("Field", indent, r)
}

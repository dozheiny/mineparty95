package ui

import (
	"image/color"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	glyphW = 7
	glyphH = 13
)

func (g *Game) fill(s *ebiten.Image, x, y, w, h int, c color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	s.DrawImage(g.pixel, op)
}

func (g *Game) blit(s, img *ebiten.Image, x, y, scale int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	op.GeoM.Translate(float64(x), float64(y))
	s.DrawImage(img, op)
}

func (g *Game) text(s *ebiten.Image, str string, x, y int, c color.Color) {
	g.textScaled(s, str, x, y, 1, c)
}

func (g *Game) textScaled(s *ebiten.Image, str string, x, y, scale int, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(s, str, g.font, op)
}

func textWidth(str string, scale int) int { return utf8.RuneCountInString(str) * glyphW * scale }

func (g *Game) textCentered(s *ebiten.Image, str string, cx, y, scale int, c color.Color) {
	g.textScaled(s, str, cx-textWidth(str, scale)/2, y, scale, c)
}

// raised draws a Win95-style bevel that sticks out of the surface.
func (g *Game) raised(s *ebiten.Image, x, y, w, h int) {
	t := g.theme
	g.fill(s, x, y, w, h, t.Face)
	g.fill(s, x, y, w-1, 1, t.Hi)
	g.fill(s, x, y, 1, h-1, t.Hi)
	g.fill(s, x, y+h-1, w, 1, t.Dark)
	g.fill(s, x+w-1, y, 1, h, t.Dark)
	g.fill(s, x+1, y+h-2, w-2, 1, t.Lo)
	g.fill(s, x+w-2, y+1, 1, h-2, t.Lo)
}

// sunken draws the same bevel pushed into the surface.
func (g *Game) sunken(s *ebiten.Image, x, y, w, h int, inside color.Color) {
	t := g.theme
	g.fill(s, x, y, w, h, inside)
	g.fill(s, x, y, w-1, 1, t.Lo)
	g.fill(s, x, y, 1, h-1, t.Lo)
	g.fill(s, x, y+h-1, w, 1, t.Hi)
	g.fill(s, x+w-1, y, 1, h, t.Hi)
}

// window draws the desktop and a titled window filling most of the screen.
func (g *Game) window(s *ebiten.Image, title string) {
	t := g.theme
	s.Fill(t.Desk)
	g.raised(s, 8, 6, ScreenW-16, ScreenH-12)
	g.fill(s, 11, 9, ScreenW-22, 18, t.Title)
	g.text(s, title, 17, 11, t.TitleText)
}

// Button is a clickable control. Scenes hold a slice of them.
type Button struct {
	X, Y, W, H int
	Label      string
	On         func()
	Disabled   bool
	Selected   bool
	Badge      string
	armed      bool
}

func (b *Button) hit(in *Input) bool {
	return in.X >= b.X && in.X < b.X+b.W && in.Y >= b.Y && in.Y < b.Y+b.H
}

func (b *Button) update(g *Game, in *Input) {
	if b.Disabled {
		b.armed = false
		return
	}
	if in.LeftPress && b.hit(in) {
		b.armed = true
	}
	if in.LeftRelease {
		if b.armed && b.hit(in) && b.On != nil {
			g.sfx.Play(sndClick)
			b.On()
		}
		b.armed = false
	}
}

func (b *Button) draw(g *Game, s *ebiten.Image, in *Input) {
	t := g.theme
	pushed := b.armed && b.hit(in) || b.Selected
	ty := b.Y + (b.H-glyphH)/2
	tx := b.X + (b.W-textWidth(b.Label, 1))/2
	if pushed {
		g.sunken(s, b.X, b.Y, b.W, b.H, t.Face)
		tx++
		ty++
	} else {
		g.raised(s, b.X, b.Y, b.W, b.H)
	}
	c := color.Color(t.Text)
	if b.Disabled {
		c = t.Lo
	}
	g.text(s, b.Label, tx, ty, c)
	if b.Selected {
		g.fill(s, b.X+3, b.Y+b.H-4, b.W-6, 1, t.Accent)
	}
	if b.Badge != "" {
		w := textWidth(b.Badge, 1) + 6
		g.fill(s, b.X+b.W-w-3, b.Y+3, w, b.H-6, t.Accent)
		g.text(s, b.Badge, b.X+b.W-w, ty, t.TitleText)
	}
}

func updateButtons(g *Game, in *Input, bs []*Button) {
	// A callback can replace the slice, so walk over a snapshot.
	for _, b := range append([]*Button(nil), bs...) {
		b.update(g, in)
	}
}

func drawButtons(g *Game, s *ebiten.Image, in *Input, bs []*Button) {
	for _, b := range bs {
		b.draw(g, s, in)
	}
}

func closeButton(on func()) *Button {
	return &Button{X: ScreenW - 30, Y: 11, W: 16, H: 14, Label: "x", On: on}
}

// TextField is a single-line text input.
type TextField struct {
	X, Y, W int
	Value   string
	Max     int
}

func (f *TextField) update(in *Input) {
	for _, r := range in.Chars {
		if r >= 32 && r < 127 && len(f.Value) < f.Max {
			f.Value += string(r)
		}
	}
	if in.Key(ebiten.KeyBackspace) && f.Value != "" {
		f.Value = f.Value[:len(f.Value)-1]
	}
	if in.Ctrl && in.Key(ebiten.KeyV) {
		for _, r := range pasteText() {
			if r >= 32 && r < 127 && len(f.Value) < f.Max {
				f.Value += string(r)
			}
		}
	}
}

func (f *TextField) draw(g *Game, s *ebiten.Image) {
	g.sunken(s, f.X, f.Y, f.W, 20, g.theme.Open)
	str := f.Value
	if (g.frame/30)%2 == 0 {
		str += "_"
	}
	g.text(s, str, f.X+5, f.Y+4, g.theme.Text)
}

// led draws a sunken seven-segment counter.
func (g *Game) led(s *ebiten.Image, x, y, value, digits int) {
	const w = 13
	g.sunken(s, x, y, digits*w+4, 27, g.theme.LEDBack)
	limit := 1
	for range digits {
		limit *= 10
	}
	value = min(value, limit-1)
	neg := value < 0
	if neg {
		value = min(-value, limit/10-1)
	}
	for i := digits - 1; i >= 0; i-- {
		g.blit(s, g.art.LED[value%10], x+2+i*w, y+2, 1)
		value /= 10
	}
	if neg {
		g.blit(s, g.art.LED[10], x+2, y+2, 1)
	}
}

// bar draws a sunken progress bar.
func (g *Game) bar(s *ebiten.Image, x, y, w, h int, frac float64, c color.Color) {
	g.sunken(s, x, y, w, h, g.theme.Open)
	frac = max(0, min(1, frac))
	g.fill(s, x+1, y+1, int(float64(w-2)*frac), h-2, c)
}

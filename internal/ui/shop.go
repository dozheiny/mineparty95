package ui

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/profile"
)

var shopShelves = []struct {
	name string
	kind profile.ItemKind
}{
	{"Themes", profile.KindTheme},
	{"Flags", profile.KindFlag},
	{"Mines", profile.KindMine},
	{"Faces", profile.KindFace},
	{"Titles", profile.KindTitle},
	{"Extras", profile.KindFreeze},
}

const (
	shopTop  = 66
	shopRowH = 39
)

type shop struct {
	shelf    int
	message  string
	tabs     []*Button
	rows     []*Button
	fixed    []*Button
	previews map[string]*ebiten.Image
}

// canBuySomething reports whether the wallet covers anything not yet owned.
func canBuySomething(p *profile.Profile) bool {
	for _, shelf := range shopShelves {
		if shelf.kind == profile.KindFreeze {
			continue
		}
		for _, it := range profile.Catalog(shelf.kind) {
			if it.Price > 0 && it.Price <= p.Coins && !p.Owns(it.ID) {
				return true
			}
		}
	}
	return false
}

func newShop(g *Game) *shop {
	back := func() { g.goTo(newMenu(g)) }
	sc := &shop{previews: map[string]*ebiten.Image{}}
	sc.fixed = []*Button{closeButton(back), {X: 24, Y: 430, W: 120, H: 30, Label: "Back", On: back}}
	for i, shelf := range shopShelves {
		sc.tabs = append(sc.tabs, &Button{X: 24 + i*100, Y: 34, W: 96, H: 24, Label: shelf.name, On: func() {
			sc.shelf = i
			sc.message = ""
			sc.stock(g)
		}})
	}
	sc.drawPreviews(g)
	sc.stock(g)
	return sc
}

func (sc *shop) drawPreviews(g *Game) {
	t := g.theme
	for id, rows := range flagSkins {
		sc.previews[id] = glyph(rows, 2, 3, flagInk(t, t.Flag))
	}
	for id, rows := range mineSkins {
		sc.previews[id] = glyph(rows, 2, 2, mineInk(t))
	}
	for id, c := range faceColors {
		sc.previews[id] = ebiten.NewImageFromImage(drawFace(faceHappy, c))
	}
}

// stock lays out the buttons of the current shelf for what the player owns.
func (sc *shop) stock(g *Game) {
	p := g.Profile
	kind := shopShelves[sc.shelf].kind
	sc.rows = nil
	for i, it := range profile.Catalog(kind) {
		b := &Button{X: 486, Y: shopTop + i*shopRowH + 4, W: 130, H: 28}
		switch {
		case it.Kind == profile.KindFreeze:
			b.Label = fmt.Sprintf("Buy  %d", it.Price)
			b.On = func() { sc.buy(g, it) }
		case p.Equipped(kind) == it.ID:
			b.Label = "In use"
			b.Selected = true
		case p.Owns(it.ID):
			b.Label = "Use"
			b.On = func() { sc.use(g, it) }
		case it.Price > 0:
			b.Label = fmt.Sprintf("Buy  %d", it.Price)
			b.On = func() { sc.buy(g, it) }
		default:
			b.Label = "Locked"
			b.Disabled = true
		}
		sc.rows = append(sc.rows, b)
	}
}

func capitalize(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

func (sc *shop) buy(g *Game, it profile.Item) {
	if err := g.Profile.Buy(it.ID); err != nil {
		sc.message = capitalize(err.Error()) + "."
		return
	}
	g.save()
	g.sfx.Play(sndAward)
	sc.message = "Bought " + it.Name + "!"
	sc.stock(g)
}

func (sc *shop) use(g *Game, it profile.Item) {
	if err := g.Profile.Equip(it.ID); err != nil {
		sc.message = capitalize(err.Error()) + "."
		return
	}
	g.applyLook()
	g.save()
	sc.message = ""
	sc.drawPreviews(g)
	sc.stock(g)
}

func (sc *shop) Update(g *Game, in *Input) {
	for i, b := range sc.tabs {
		b.Selected = i == sc.shelf
	}
	updateButtons(g, in, sc.fixed)
	updateButtons(g, in, sc.tabs)
	updateButtons(g, in, sc.rows)
}

func (sc *shop) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	p := g.Profile
	g.window(s, "Shop")
	coins := fmt.Sprintf("Coins: %d", p.Coins)
	g.text(s, coins, ScreenW-40-textWidth(coins, 1), 11, t.TitleText)
	drawButtons(g, s, &g.in, sc.tabs)
	for i, it := range profile.Catalog(shopShelves[sc.shelf].kind) {
		y := shopTop + i*shopRowH
		sc.drawItem(g, s, it, y)
		if i+1 < len(sc.rows) {
			g.fill(s, 24, y+shopRowH-1, 592, 1, t.Lo)
		}
	}
	drawButtons(g, s, &g.in, sc.rows)
	drawButtons(g, s, &g.in, sc.fixed)
	if sc.message != "" {
		g.text(s, sc.message, 160, 438, t.Accent)
	}
}

func (sc *shop) drawItem(g *Game, s *ebiten.Image, it profile.Item, y int) {
	t := g.theme
	p := g.Profile
	x := 200
	note := ""
	switch it.Kind {
	case profile.KindTheme:
		pal := themeFor(it.ID)
		for j, c := range []color.RGBA{pal.Desk, pal.Face, pal.Open, pal.Num[1], pal.Num[3], pal.Flag, pal.Accent} {
			g.fill(s, 28+j*22, y+8, 20, 20, c)
		}
		if !p.Owns(it.ID) && it.Price == 0 {
			note = profile.ThemeByID(it.ID).Unlocks()
		}
	case profile.KindFlag, profile.KindMine:
		g.blit(s, g.art.Hidden, 28, y+3, 2)
		g.blit(s, sc.previews[it.ID], 28, y+3, 2)
		x = 76
	case profile.KindFace:
		g.blit(s, sc.previews[it.ID], 30, y+7, 1)
		x = 76
	case profile.KindTitle:
		x = 28
	case profile.KindFreeze:
		x = 28
		note = fmt.Sprintf("Covers one missed daily. You hold %d of %d.", p.Freezes, profile.MaxFreezes)
	}
	g.text(s, it.Name, x, y+5, t.Text)
	if note != "" {
		g.text(s, note, x, y+20, t.Lo)
	}
}

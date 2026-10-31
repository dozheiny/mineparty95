package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// developers is the list shown on the About page.
var developers = []struct {
	Name, Role string
}{
	{"Iliya Aghamajidi", "Developer"},
}

var aboutText = "Mine Party 95 is a retro Minesweeper with direct online multiplayer for you and your friends. " +
	"There is no account, no server and no subscription: one player's game hosts the match, " +
	"and everything else lives on your own machine."

var builtWith = []string{"Ebitengine, the game engine", "goupnp, for opening router ports", "Go, and a lot of square waves"}

type about struct {
	bs []*Button
}

func newAbout(g *Game) *about {
	back := func() { g.goTo(newMenu(g)) }
	return &about{bs: []*Button{closeButton(back), {X: 24, Y: 430, W: 120, H: 30, Label: "Back", On: back}}}
}

func (a *about) Update(g *Game, in *Input) { updateButtons(g, in, a.bs) }

func (a *about) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	g.window(s, "About")
	g.textScaled(s, "MINE PARTY", 40, 40, 4, t.Text)
	g.textScaled(s, "95", 330, 40, 4, t.Accent)
	g.blit(s, g.art.Face[faceCool], 394, 46, 2)
	g.text(s, "Version "+Version, 42, 96, t.Text)

	y := 122
	for _, line := range wrap(aboutText, 78) {
		g.text(s, line, 42, y, t.Text)
		y += 16
	}

	y += 16
	g.text(s, "Developers", 42, y, t.Accent)
	y += 20
	for _, d := range developers {
		g.sunken(s, 42, y, 556, 26, t.Open)
		g.text(s, d.Name, 52, y+6, t.Text)
		g.text(s, d.Role, 588-textWidth(d.Role, 1), y+6, t.Text)
		y += 30
	}

	y += 12
	g.text(s, "Built with", 42, y, t.Accent)
	y += 18
	for _, line := range builtWith {
		g.text(s, line, 42, y, t.Text)
		y += 15
	}
	drawButtons(g, s, &g.in, a.bs)
}

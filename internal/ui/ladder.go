package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/profile"
)

type ladder struct {
	bs []*Button
}

func newLadder(g *Game) *ladder {
	back := func() { g.goTo(newMenu(g)) }
	return &ladder{bs: []*Button{closeButton(back), {X: 24, Y: 430, W: 120, H: 30, Label: "Back", On: back}}}
}

func (l *ladder) Update(g *Game, in *Input) { updateButtons(g, in, l.bs) }

func (l *ladder) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	p := g.Profile
	g.window(s, "Ladder")
	rank := profile.RankOf(p.RP)
	g.textCentered(s, rank.Name, ScreenW/2, 38, 3, t.Accent)
	progress := fmt.Sprintf("%d RP", p.RP)
	frac := 1.0
	if rank.Need > 0 {
		frac = float64(rank.Into) / float64(rank.Need)
		progress = fmt.Sprintf("%d RP, %d to the next division", p.RP, rank.Need-rank.Into)
	}
	g.bar(s, 120, 84, 400, 14, frac, t.Accent)
	g.textCentered(s, progress, ScreenW/2, 102, 1, t.Text)

	g.text(s, "Tiers", 40, 128, t.Text)
	for i := len(profile.Tiers) - 1; i >= 0; i-- {
		tier := profile.Tiers[i]
		y := 148 + (len(profile.Tiers)-1-i)*42
		fill := t.Open
		if i == rank.Tier {
			fill = t.Hi
		}
		g.sunken(s, 40, y, 270, 38, fill)
		g.text(s, tier.Name, 52, y+4, t.Text)
		g.text(s, fmt.Sprintf("from %d RP", tier.MinRP), 52, y+20, t.Text)
		reward := "Paid"
		switch {
		case i > p.BestTier:
			reward = fmt.Sprintf("+%d coins", tier.Coins)
		case i == 0:
			reward = "Start"
		}
		g.text(s, reward, 298-textWidth(reward, 1), y+12, t.Accent)
	}

	g.text(s, "Level rewards", 340, 128, t.Text)
	level := p.Level()
	for i := range 6 {
		lv := level + 1 + i
		y := 148 + i*42
		g.sunken(s, 340, y, 260, 38, t.Open)
		g.text(s, fmt.Sprintf("Level %d", lv), 352, y+4, t.Text)
		g.text(s, profile.Title(lv), 352, y+20, t.Text)
		reward := fmt.Sprintf("+%d coins", profile.LevelCoins(lv))
		g.text(s, reward, 588-textWidth(reward, 1), y+12, t.Accent)
	}
	g.text(s, "Wins climb, losses slip, but you never drop a tier.", 160, 438, t.Accent)
	drawButtons(g, s, &g.in, l.bs)
}

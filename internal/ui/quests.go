package ui

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/profile"
)

type questsScene struct {
	bs []*Button
}

func newQuests(g *Game) *questsScene {
	back := func() { g.goTo(newMenu(g)) }
	return &questsScene{bs: []*Button{closeButton(back), {X: 24, Y: 430, W: 120, H: 30, Label: "Back", On: back}}}
}

func (sc *questsScene) Update(g *Game, in *Input) { updateButtons(g, in, sc.bs) }

func (sc *questsScene) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	p := g.Profile
	date := profile.Today(time.Now())
	g.window(s, "Quests")
	g.textCentered(s, "Today's quests", ScreenW/2, 40, 2, t.Text)
	y := 76
	for _, q := range p.Quests(date) {
		g.sunken(s, 40, y, 560, 66, t.Open)
		g.text(s, q.Desc, 52, y+8, t.Text)
		reward := fmt.Sprintf("+%d coins  +%d XP", q.Coins, q.XP)
		g.text(s, reward, 588-textWidth(reward, 1), y+8, t.Accent)
		g.bar(s, 52, y+30, 460, 14, float64(q.Progress)/float64(q.Goal), t.Accent)
		status := fmt.Sprintf("%d / %d", q.Progress, q.Goal)
		if q.Done() {
			status = "DONE!"
		}
		g.text(s, status, 522, y+31, t.Text)
		y += 76
	}
	g.textCentered(s, fmt.Sprintf("Finish all three for a %d coin bonus. New quests every day!", profile.QuestBonusPay), ScreenW/2, y+2, 1, t.Accent)
	y += 30
	g.text(s, "Login bonus", 150, y, t.Text)
	g.loginStrip(s, 150, y+18, p.LoginDay)
	drawButtons(g, s, &g.in, sc.bs)
}

package ui

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/match"
	"mineparty/internal/profile"
)

type stats struct {
	bs []*Button
}

func newStats(g *Game) *stats {
	back := func() { g.goTo(newMenu(g)) }
	return &stats{bs: []*Button{closeButton(back), {X: 24, Y: 430, W: 120, H: 30, Label: "Back", On: back}}}
}

func (st *stats) Update(g *Game, in *Input) { updateButtons(g, in, st.bs) }

func (st *stats) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	p := g.Profile
	g.window(s, "Stats & Awards")
	y := 36
	line := func(str string) {
		g.text(s, str, 26, y, t.Text)
		y += 15
	}
	level := p.Level()
	line(fmt.Sprintf("Level %d %s, %d XP", level, profile.Title(level), p.XP))
	y += 6
	line("Games and wins")
	for _, mode := range []match.Mode{match.Classic, match.Blitz, match.Survival, match.Race, match.Flags, match.Coop, match.Territory, match.LastStand} {
		line(fmt.Sprintf("  %-11s %4d played  %4d won", mode.String(), p.Played[mode.String()], p.Wins[mode.String()]))
	}
	y += 6
	line("Best times")
	for _, key := range []string{"beginner", "intermediate", "expert", "daily", "survival"} {
		best := "--"
		if ms, ok := p.Best[key]; ok {
			best = formatTime(ms)
		}
		line(fmt.Sprintf("  %-13s %s", key, best))
	}
	y += 6
	line(fmt.Sprintf("Most Blitz boards: %d", p.BestBoards))
	line(fmt.Sprintf("Best daily streak: %d", p.BestDaily))
	line(fmt.Sprintf("Best win streak:   %d", p.BestWinStreak))
	y += 6
	line("Rivals")
	names := make([]string, 0, len(p.Rivals))
	for name := range p.Rivals {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		line("  Beat a friend to start a rivalry!")
	}
	for _, name := range names[:min(len(names), 2)] {
		rec := p.Rivals[name]
		line(fmt.Sprintf("  %-14s %d-%d", name, rec.Wins, rec.Losses))
	}

	const ax = 342
	g.text(s, "Awards", ax, 36, t.Text)
	in := &g.in
	desc := ""
	for i, a := range profile.Achievements() {
		ay := 54 + i*16
		_, have := p.Achievements[a.ID]
		if have {
			g.fill(s, ax, ay+2, 11, 11, t.Accent)
		} else {
			g.sunken(s, ax, ay+2, 11, 11, t.Open)
		}
		c := t.Lo
		if have {
			c = t.Text
		}
		g.text(s, a.Name, ax+18, ay+1, c)
		if in.X >= ax && in.X < ScreenW-16 && in.Y >= ay && in.Y < ay+16 {
			desc = a.Desc
		}
	}
	if desc == "" {
		desc = "Hover over an award to see how to earn it."
	}
	g.text(s, desc, 160, 438, t.Accent)
	drawButtons(g, s, &g.in, st.bs)
}

type settings struct {
	name TextField
	bs   []*Button
	nog  *Button
	full *Button
	tune *Button
}

func newSettings(g *Game) *settings {
	sc := &settings{name: TextField{X: 150, Y: 60, W: 200, Value: g.Profile.Name, Max: 14}}
	back := func() {
		if sc.name.Value != "" {
			g.Profile.Name = sc.name.Value
		}
		g.save()
		g.goTo(newMenu(g))
	}
	sc.nog = &Button{X: 150, Y: 136, W: 200, H: 26, On: func() {
		g.Profile.PureRandom = !g.Profile.PureRandom
	}}
	sc.full = &Button{X: 150, Y: 174, W: 200, H: 26, On: func() {
		g.Profile.Fullscreen = !g.Profile.Fullscreen
		ebiten.SetFullscreen(g.Profile.Fullscreen)
	}}
	sc.tune = &Button{X: 280, Y: 98, W: 120, H: 26, On: func() {
		g.Profile.MusicOff = !g.Profile.MusicOff
		g.sfx.setMusic(!g.Profile.MusicOff)
	}}
	volume := func(d int) {
		g.Profile.Volume = max(0, min(10, g.Profile.Volume+d))
		g.sfx.setVolume(g.Profile.Volume)
		g.sfx.Play(sndFlag)
	}
	sc.bs = []*Button{
		closeButton(back), sc.nog, sc.full, sc.tune,
		{X: 150, Y: 98, W: 26, H: 26, Label: "-", On: func() { volume(-1) }},
		{X: 240, Y: 98, W: 26, H: 26, Label: "+", On: func() { volume(1) }},
		{X: 24, Y: 430, W: 120, H: 30, Label: "Save", On: back},
	}
	return sc
}

func (sc *settings) Update(g *Game, in *Input) {
	sc.nog.Label = "No-guess boards: ON"
	if g.Profile.PureRandom {
		sc.nog.Label = "No-guess boards: OFF"
	}
	sc.full.Label = "Fullscreen: OFF"
	if g.Profile.Fullscreen {
		sc.full.Label = "Fullscreen: ON"
	}
	sc.tune.Label = "Music: ON"
	if g.Profile.MusicOff {
		sc.tune.Label = "Music: OFF"
	}
	sc.name.update(in)
	updateButtons(g, in, sc.bs)
}

func (sc *settings) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	g.window(s, "Settings")
	g.text(s, "Your name", 28, 64, t.Text)
	sc.name.draw(g, s)
	g.text(s, "Sound", 28, 104, t.Text)
	g.textCentered(s, fmt.Sprint(g.Profile.Volume), 208, 104, 1, t.Text)
	g.text(s, "Solo boards", 28, 142, t.Text)
	g.text(s, "Display", 28, 180, t.Text)
	g.text(s, "Controls", 28, 224, t.Text)
	for i, line := range []string{
		"Left click        open a tile, or chord on a number",
		"Right click       flag (Ctrl+click also works)",
		"Space or F2       new game",
		"P or middle click ping a tile in multiplayer",
		"B                 arm your bomb in Mine Flags",
		"F11               fullscreen",
		"Esc               leave a solo or practice game",
	} {
		g.text(s, line, 28, 242+i*16, t.Text)
	}
	drawButtons(g, s, &g.in, sc.bs)
}

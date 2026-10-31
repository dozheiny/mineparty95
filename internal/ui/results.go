package ui

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/match"
	"mineparty/internal/netplay"
	"mineparty/internal/profile"
)

const (
	panelX = 150
	panelW = 340

	// maxResultLines keeps the panel on the screen after a big game.
	maxResultLines = 20
)

// results is the end-of-game panel. It is the moment that sells one more
// game, so the XP bar fills visibly and every reward gets its own line.
type results struct {
	title, sub string
	lines      []string
	rw         profile.Rewards
	xpFrom     int
	age        int
	hidden     bool
	share      string
	bs         []*Button
	reopen     *Button
	note       string
	y, h       int
	settle     int // frames to leave the board visible first
}

func newResults(g *Game, p *play, v *match.View) *results {
	you := p.you()
	won, draw := outcome(v)
	r := &results{xpFrom: g.Profile.XP}

	stale := g.lastSession == p.sess && g.lastRound == v.Round
	if !stale && you != nil {
		g.lastSession, g.lastRound = p.sess, v.Round
		date := p.opt.Date
		if date == "" {
			date = profile.Today(time.Now())
		}
		r.rw = g.Profile.Record(profile.GameResult{
			Mode: v.Mode, Board: p.opt.BoardKey, Daily: p.opt.Daily, Date: date,
			Won: won, Draw: draw, ElapsedMs: v.ElapsedMs, BV3: v.BV3, Clicks: you.Clicks,
			Progress: you.Progress, Score: you.Score, Flagged: you.Flagged, UsedBomb: you.BombUsed,
			Rivals: rivals(p.sess, v), Boards: v.Boards, Roll: rand.IntN(100),
		})
		g.save()
	}

	switch {
	case won:
		g.sfx.Play(sndWin)
		g.confetti()
	case v.Mode == match.Blitz:
		g.sfx.Play(sndLose)
	case v.Mode.Solo() || v.Mode == match.Coop:
		r.settle = 55
		g.sfx.Play(sndBoom)
		g.shake = 18
		for i, t := range v.Tiles {
			if t == match.TileExploded {
				p.explode(g, i, 70)
			}
		}
	default:
		g.sfx.Play(sndLose)
	}
	r.describe(g, p, v, won, draw)
	r.layout(g, p)
	return r
}

func outcome(v *match.View) (won, draw bool) {
	if v.Mode.Solo() || v.Mode == match.Coop {
		return v.Won, false
	}
	return v.Winner == v.You, (v.Mode == match.Territory || v.Mode == match.LastStand) && v.Winner < 0
}

func rivals(sess netplay.Session, v *match.View) []string {
	if v.Mode.Solo() || v.Mode == match.Coop {
		return nil
	}
	var names []string
	room := sess.Room()
	for i, pl := range room.Players {
		if i != v.You && !pl.Bot {
			names = append(names, pl.Name)
		}
	}
	return names
}

func (r *results) describe(g *Game, p *play, v *match.View, won, draw bool) {
	you := p.you()
	switch {
	case v.Mode == match.Blitz:
		r.title = "TIME UP!"
	case won && v.Mode == match.Coop:
		r.title = "TEAM CLEAR!"
	case won && v.Mode.Solo():
		r.title = "CLEARED!"
	case won:
		r.title = "YOU WIN!"
	case draw:
		r.title = "A DRAW!"
	case v.Mode.Solo() || v.Mode == match.Coop:
		r.title = "BOOM!"
	case v.Winner >= 0:
		r.title = v.Players[v.Winner].Name + " wins"
	default:
		r.title = "GAME OVER"
	}
	if !won && you != nil && you.Progress >= 75 && (v.Mode == match.Classic || v.Mode == match.Survival) {
		r.sub = fmt.Sprintf("So close! %d%% cleared", you.Progress)
	}
	if r.rw.NewBest {
		r.sub = "NEW PERSONAL BEST!"
	}
	r.lines = append(r.lines, fmt.Sprintf("Time %s", formatTime(v.ElapsedMs)))
	if won && v.ElapsedMs > 0 && v.BV3 > 0 && you != nil && v.Mode == match.Classic {
		r.lines = append(r.lines, fmt.Sprintf("3BV %d   %.2f/s   %d clicks", v.BV3, float64(v.BV3)*1000/float64(v.ElapsedMs), you.Clicks))
	}
	switch {
	case v.Mode == match.Blitz:
		r.lines = append(r.lines, fmt.Sprintf("Boards cleared: %d", v.Boards))
	case v.Mode == match.Survival && you != nil:
		r.lines = append(r.lines, fmt.Sprintf("Lives left: %d", you.Lives))
	case !v.Mode.Solo():
		r.lines = append(r.lines, standings(v)...)
	}
	if r.rw.DailyStreak > 0 && p.opt.Daily {
		r.lines = append(r.lines, fmt.Sprintf("Daily streak: %d", r.rw.DailyStreak))
	}
	if r.rw.LevelAfter > r.rw.LevelBefore {
		r.lines = append(r.lines, fmt.Sprintf("LEVEL UP! You're a %s now.", profile.Title(r.rw.LevelAfter)))
	}
	for _, a := range r.rw.Achievements {
		r.lines = append(r.lines, "* Award: "+a.Name)
	}
	for _, th := range r.rw.Themes {
		r.lines = append(r.lines, "* New theme: "+th.Name)
	}
	r.lines = append(r.lines, r.economyLines(g)...)
	r.lines = capLines(r.lines)
	if won && p.opt.Daily {
		r.share = fmt.Sprintf("Mine Party 95 daily %s: %s, 3BV %d, streak %d. Beat that!", p.opt.Date, formatTime(v.ElapsedMs), v.BV3, g.Profile.DailyStreak)
	}
}

// economyLines reports coins, rank and quests. They're the last thing on the
// panel, so that the next goal is what the player reads before clicking Again.
func (r *results) economyLines(g *Game) []string {
	rw := r.rw
	if rw.RankAfter == "" {
		return nil
	}
	var lines []string
	for _, q := range rw.Quests {
		lines = append(lines, "* Quest done: "+q.Desc)
	}
	if rw.QuestBonus {
		lines = append(lines, fmt.Sprintf("* All quests done! +%d coins", profile.QuestBonusPay))
	}
	for _, t := range rw.Promotions {
		lines = append(lines, fmt.Sprintf("* PROMOTED to %s! +%d coins", t.Name, t.Coins))
	}
	coins := fmt.Sprintf("+%d coins", rw.Coins)
	switch rw.Lucky {
	case 2:
		coins += "  LUCKY x2!"
	case 5:
		coins += "  JACKPOT x5!"
	}
	lines = append(lines, coins)
	rank := profile.RankOf(g.Profile.RP)
	line := fmt.Sprintf("Rank %s  %+d RP", rank.Name, rw.RP)
	if rank.Need > 0 {
		line += fmt.Sprintf("  (%d to go)", rank.Need-rank.Into)
	}
	return append(lines, line)
}

func capLines(lines []string) []string {
	if len(lines) <= maxResultLines {
		return lines
	}
	return append(lines[:maxResultLines-1:maxResultLines-1], "...and more rewards!")
}

func standings(v *match.View) []string {
	order := make([]int, len(v.Players))
	for i := range order {
		order[i] = i
	}
	score := func(i int) int {
		if v.Mode == match.Race {
			return v.Players[i].Progress
		}
		return v.Players[i].Score
	}
	sort.SliceStable(order, func(a, b int) bool {
		if v.Mode == match.LastStand && (order[a] == v.Winner || order[b] == v.Winner) {
			return order[a] == v.Winner && order[b] != v.Winner
		}
		return score(order[a]) > score(order[b])
	})
	unit := map[match.Mode]string{match.Race: "%", match.Flags: " mines", match.Coop: " tiles", match.Territory: " tiles", match.LastStand: " tiles"}[v.Mode]
	var out []string
	for rank, i := range order {
		out = append(out, fmt.Sprintf("%d. %s  %d%s", rank+1, v.Players[i].Name, score(i), unit))
	}
	return out
}

func (r *results) layout(g *Game, p *play) {
	r.h = 96 + len(r.lines)*15 + 44
	r.y = (ScreenH - r.h) / 2
	by := r.y + r.h - 34
	add := func(label string, on func()) *Button {
		b := &Button{Y: by, H: 26, Label: label, On: on}
		r.bs = append(r.bs, b)
		return b
	}
	switch {
	case p.opt.Restart != nil:
		add("Again", func() { p.restart(g) })
	case p.opt.Host != nil:
		add("Rematch", func() { p.restart(g) })
		add("Lobby", func() { g.goTo(p.opt.Lobby) })
	default:
		r.note = "Waiting for the host..."
	}
	if r.share != "" {
		add("Share", func() {
			if copyText(r.share) {
				r.note = "Copied! Paste it to your friends."
			} else {
				r.note = "Couldn't reach the clipboard here."
			}
		})
	}
	add("View", func() { r.hidden = true })
	add("Menu", func() { p.leave(g) })
	w := (panelW - 16) / len(r.bs)
	for i, b := range r.bs {
		b.X, b.W = panelX+8+i*w, w-4
	}
	r.reopen = &Button{X: 12, Y: 62, W: 90, H: 26, Label: "Results", On: func() { r.hidden = false }}
}

func (r *results) update(g *Game, in *Input) {
	if r.settle > 0 {
		r.settle--
		return
	}
	r.age++
	if r.age == 50 {
		switch {
		case r.rw.LevelAfter > r.rw.LevelBefore:
			g.sfx.Play(sndLevel)
		case len(r.rw.Achievements) > 0:
			g.sfx.Play(sndAward)
		}
	}
	if r.hidden {
		r.reopen.update(g, in)
		return
	}
	updateButtons(g, in, r.bs)
}

func (r *results) draw(g *Game, s *ebiten.Image) {
	if r.settle > 0 {
		return
	}
	if r.hidden {
		r.reopen.draw(g, s, &g.in)
		return
	}
	t := g.theme
	x, y := panelX, r.y
	g.raised(s, x, y, panelW, r.h)
	g.fill(s, x+3, y+3, panelW-6, 16, t.Title)
	g.text(s, "Results", x+8, y+4, t.TitleText)
	g.textCentered(s, r.title, x+panelW/2, y+24, 2, t.Text)
	line := y + 52
	if r.sub != "" {
		g.textCentered(s, r.sub, x+panelW/2, line, 1, t.Accent)
	}
	line += 18
	for _, l := range r.lines {
		g.text(s, l, x+14, line, t.Text)
		line += 15
	}
	r.drawXP(g, s, x+14, line+4)
	if r.note != "" {
		g.textCentered(s, r.note, x+panelW/2, r.y+r.h-50, 1, t.Accent)
	}
	drawButtons(g, s, &g.in, r.bs)
}

func (r *results) drawXP(g *Game, s *ebiten.Image, x, y int) {
	t := g.theme
	frac := min(float64(r.age)/45, 1)
	xp := r.xpFrom + int(float64(r.rw.XP)*frac)
	level, into, need := profile.Level(xp)
	g.bar(s, x, y, panelW-28-70, 11, float64(into)/float64(need), t.Accent)
	g.text(s, fmt.Sprintf("Lv%d +%d", level, r.rw.XP), x+panelW-28-64, y-1, t.Text)
}

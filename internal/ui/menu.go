package ui

import (
	"fmt"
	"hash/fnv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/engine"
	"mineparty/internal/match"
	"mineparty/internal/netplay"
	"mineparty/internal/profile"
)

// dailyBoard is bigger than Intermediate, so a daily is worth sitting down for.
var dailyBoard = engine.Config{W: 24, H: 16, Mines: 64}

func dailySeed(date string) uint64 {
	h := fnv.New64a()
	h.Write([]byte("mine party daily " + date))
	return h.Sum64() | 1
}

type menu struct {
	bs   []*Button
	gift *loginGift
}

// loginGift is the popup for the day's login bonus.
type loginGift struct {
	coins, day int
	ok         *Button
}

func newMenu(g *Game) *menu {
	m := &menu{}
	date := profile.Today(time.Now())
	if coins, day := g.Profile.ClaimLogin(date); coins > 0 {
		g.save()
		g.sfx.Play(sndAward)
		m.gift = &loginGift{coins: coins, day: day}
		m.gift.ok = &Button{X: 270, Y: 300, W: 100, H: 28, Label: "Collect", On: func() { m.gift = nil }}
	}
	daily := &Button{Label: "Daily Challenge", On: func() { startDaily(g, date) }}
	if !g.Profile.DailyDone(date) {
		daily.Badge = "NEW"
	}
	quests := &Button{Label: "Quests", On: func() { g.goTo(newQuests(g)) }}
	if done := questsDone(g.Profile, date); done < profile.QuestsPerDay {
		quests.Badge = fmt.Sprintf("%d/%d", done, profile.QuestsPerDay)
	}
	shop := &Button{Label: "Shop", On: func() { g.goTo(newShop(g)) }}
	if canBuySomething(g.Profile) {
		shop.Badge = "BUY"
	}
	entries := []*Button{
		daily,
		{Label: "Solo", On: func() { g.goTo(newSolo(g)) }},
		{Label: "Practice vs CPU", On: func() { startPractice(g) }},
		{Label: "Host Game", On: func() { startHosting(g) }},
		{Label: "Join Game", On: func() { g.goTo(newJoin(g)) }},
		quests,
		{Label: "Ladder", On: func() { g.goTo(newLadder(g)) }},
		shop,
		{Label: "Stats & Awards", On: func() { g.goTo(newStats(g)) }},
		{Label: "Settings", On: func() { g.goTo(newSettings(g)) }},
		{Label: "About", On: func() { g.goTo(newAbout(g)) }},
		{Label: "Quit", On: func() { g.quit = true }},
	}
	for i, b := range entries {
		b.X, b.Y, b.W, b.H = 28, 112+i*28, 260, 25
	}
	m.bs = entries
	return m
}

func questsDone(p *profile.Profile, date string) int {
	done := 0
	for _, q := range p.Quests(date) {
		if q.Done() {
			done++
		}
	}
	return done
}

func (m *menu) Update(g *Game, in *Input) {
	if m.gift != nil {
		m.gift.ok.update(g, in)
		return
	}
	updateButtons(g, in, m.bs)
}

func (m *menu) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	g.window(s, "Mine Party 95")
	g.textScaled(s, "MINE PARTY", 29, 38, 4, t.Lo)
	g.textScaled(s, "MINE PARTY", 27, 36, 4, t.Text)
	g.textScaled(s, "95", 318, 36, 4, t.Accent)
	g.blit(s, g.art.Face[faceHappy], 382, 42, 2)
	g.text(s, "Retro sweeping. Real friends. Zero guessing.", 29, 92, t.Text)
	drawButtons(g, s, &g.in, m.bs)
	m.drawCard(g, s)
	if m.gift != nil {
		m.gift.draw(g, s)
	}
}

func (lg *loginGift) draw(g *Game, s *ebiten.Image) {
	t := g.theme
	const x, y, w, h = 150, 160, 340, 180
	g.raised(s, x, y, w, h)
	g.fill(s, x+3, y+3, w-6, 16, t.Title)
	g.text(s, "Welcome back!", x+8, y+4, t.TitleText)
	g.textCentered(s, fmt.Sprintf("+%d COINS", lg.coins), ScreenW/2, y+28, 3, t.Accent)
	g.textCentered(s, fmt.Sprintf("Day %d of your login bonus", lg.day), ScreenW/2, y+74, 1, t.Text)
	g.loginStrip(s, x+24, y+96, lg.day)
	lg.ok.draw(g, s, &g.in)
}

// loginStrip draws the week of the login bonus, with claimed days filled in.
func (g *Game) loginStrip(s *ebiten.Image, x, y, day int) {
	t := g.theme
	for i, coins := range profile.LoginCoins() {
		bx := x + i*42
		if i < day {
			g.fill(s, bx, y, 38, 24, t.Accent)
			g.textCentered(s, fmt.Sprint(coins), bx+19, y+6, 1, t.TitleText)
			continue
		}
		g.sunken(s, bx, y, 38, 24, t.Open)
		g.textCentered(s, fmt.Sprint(coins), bx+19, y+6, 1, t.Text)
	}
}

func (m *menu) drawCard(g *Game, s *ebiten.Image) {
	t := g.theme
	p := g.Profile
	const x, w = 306, 306
	g.sunken(s, x, 112, w, 342, t.Open)
	y := 122
	level, into, need := profile.Level(p.XP)
	g.textScaled(s, p.Name, x+12, y, 2, t.Text)
	y += 30
	g.text(s, fmt.Sprintf("Level %d  %s", level, p.DisplayTitle()), x+12, y, t.Text)
	y += 18
	g.bar(s, x+12, y, w-24, 12, float64(into)/float64(need), t.Accent)
	y += 16
	g.text(s, fmt.Sprintf("%d / %d XP", into, need), x+12, y, t.Text)
	y += 22
	g.text(s, fmt.Sprintf("Coins: %d", p.Coins), x+12, y, t.Accent)
	if p.Freezes > 0 {
		g.text(s, fmt.Sprintf("Freezes: %d", p.Freezes), x+170, y, t.Text)
	}
	y += 18
	rank := profile.RankOf(p.RP)
	g.text(s, "Rank: "+rank.Name, x+12, y, t.Text)
	y += 16
	frac := 1.0
	if rank.Need > 0 {
		frac = float64(rank.Into) / float64(rank.Need)
	}
	g.bar(s, x+12, y, w-24, 8, frac, t.Accent)
	y += 16

	date := profile.Today(time.Now())
	streak := p.DailyStreakAlive(date)
	g.text(s, fmt.Sprintf("Daily streak: %d", streak), x+12, y, t.Text)
	y += 16
	switch {
	case p.DailyDone(date):
		g.text(s, "Today's daily is done. Back tomorrow!", x+12, y, t.Text)
	case streak > 0:
		g.text(s, "Play today to keep your streak alive!", x+12, y, t.Accent)
	default:
		g.text(s, "Today's board is waiting.", x+12, y, t.Text)
	}
	y += 22
	g.text(s, fmt.Sprintf("Quests today: %d of %d done", questsDone(p, date), profile.QuestsPerDay), x+12, y, t.Text)
	y += 16
	g.text(s, fmt.Sprintf("Win streak: %d (best %d)", p.WinStreak, p.BestWinStreak), x+12, y, t.Text)
	y += 16
	g.text(s, fmt.Sprintf("Awards: %d of %d", len(p.Achievements), len(profile.Achievements())), x+12, y, t.Text)
	y += 22
	if next := nextUnlock(p); next != "" {
		g.text(s, "Next unlock:", x+12, y, t.Text)
		y += 16
		g.text(s, next, x+12, y, t.Accent)
	}
	if g.notice != "" {
		g.text(s, g.notice, x+12, 418, t.Accent)
	}
}

func nextUnlock(p *profile.Profile) string {
	for _, th := range profile.Themes {
		if !p.ThemeUnlocked(th.ID) {
			return fmt.Sprintf("%s (%s)", th.Name, th.Unlocks())
		}
	}
	return ""
}

// startOffline opens a room with no networking, the base of every solo game.
func startOffline(g *Game) (*netplay.Host, bool) {
	h, err := netplay.NewHost(netplay.HostOptions{Name: g.Profile.Name, Offline: true})
	if err != nil {
		g.notice = err.Error()
		return nil, false
	}
	g.session = h
	return h, true
}

func startClassic(g *Game, key string, board engine.Config) {
	h, ok := startOffline(g)
	if !ok {
		return
	}
	restart := func() {
		h.StartWith(match.Config{Mode: match.Classic, Board: board, AllowGuess: g.Profile.PureRandom})
	}
	restart()
	g.goTo(newPlay(g, h, PlayOptions{Title: "Solo", BoardKey: key, Restart: restart}))
}

// startSoloMode begins a solo mode that brings its own board.
func startSoloMode(g *Game, key, title string, mode match.Mode) {
	h, ok := startOffline(g)
	if !ok {
		return
	}
	restart := func() { h.StartWith(match.Config{Mode: mode}) }
	restart()
	g.goTo(newPlay(g, h, PlayOptions{Title: title, BoardKey: key, Restart: restart}))
}

func startDaily(g *Game, date string) {
	h, ok := startOffline(g)
	if !ok {
		return
	}
	start := dailyBoard.H/2*dailyBoard.W + dailyBoard.W/2
	restart := func() {
		h.StartWith(match.Config{Mode: match.Classic, Board: dailyBoard, Seed: dailySeed(date), Start: []int{start}})
	}
	restart()
	g.goTo(newPlay(g, h, PlayOptions{Title: "Daily " + date, Daily: true, Date: date, Restart: restart}))
}

func startPractice(g *Game) {
	h, ok := startOffline(g)
	if !ok {
		return
	}
	h.AddBot()
	g.goTo(newLobby(g, h, h))
}

func startHosting(g *Game) {
	h, err := netplay.NewHost(netplay.HostOptions{Name: g.Profile.Name, Port: netplay.DefaultPort})
	if err != nil {
		g.notice = err.Error()
		return
	}
	// A second copy of the game may hold this port; codes still work then.
	h.ServeDiscovery(netplay.DiscoveryPort)
	if !g.opts.NoRouter {
		h.OpenInternet()
	}
	g.session = h
	g.goTo(newLobby(g, h, h))
}

type solo struct {
	bs []*Button
}

var classicBoards = []struct {
	key, name string
	board     engine.Config
}{
	{"beginner", "Beginner", engine.Beginner},
	{"intermediate", "Intermediate", engine.Intermediate},
	{"expert", "Expert", engine.Expert},
}

func formatTime(ms int) string {
	tenths := ms / 100
	return fmt.Sprintf("%d:%02d.%d", tenths/600, tenths/10%60, tenths%10)
}

func newSolo(g *Game) *solo {
	back := func() { g.goTo(newMenu(g)) }
	sc := &solo{bs: []*Button{closeButton(back)}}
	row := func(i int, label string, on func()) {
		sc.bs = append(sc.bs, &Button{X: 150, Y: 84 + i*62, W: 340, H: 36, Label: label, On: on})
	}
	for i, c := range classicBoards {
		row(i, fmt.Sprintf("%s  (%dx%d, %d mines)", c.name, c.board.W, c.board.H, c.board.Mines), func() { startClassic(g, c.key, c.board) })
	}
	row(3, "Blitz  (clear boards against the clock)", func() { startSoloMode(g, "blitz", "Blitz", match.Blitz) })
	row(4, "Survival  (Expert board, 3 lives)", func() { startSoloMode(g, "survival", "Survival", match.Survival) })
	sc.bs = append(sc.bs, &Button{X: 150, Y: 400, W: 340, H: 32, Label: "Back", On: back})
	return sc
}

func (sc *solo) Update(g *Game, in *Input) { updateButtons(g, in, sc.bs) }

func (sc *solo) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	g.window(s, "Solo")
	g.textCentered(s, "Pick your minefield", ScreenW/2, 44, 2, t.Text)
	best := func(i int, key string) {
		text := "No record yet"
		if ms, ok := g.Profile.Best[key]; ok {
			text = "Best: " + formatTime(ms)
		}
		g.textCentered(s, text, ScreenW/2, 123+i*62, 1, t.Text)
	}
	for i, c := range classicBoards {
		best(i, c.key)
	}
	g.textCentered(s, fmt.Sprintf("Most boards: %d", g.Profile.BestBoards), ScreenW/2, 123+3*62, 1, t.Text)
	best(4, "survival")
	drawButtons(g, s, &g.in, sc.bs)
}

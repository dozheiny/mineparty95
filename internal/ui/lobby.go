package ui

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/match"
	"mineparty/internal/netplay"
)

const (
	blurbWidth    = 41
	blurbTop      = 120
	difficultyTop = 184
)

var modeBlurbs = map[match.Mode]string{
	match.Race:      "Everyone gets the same board. First to clear it wins. Hit a mine and you're stunned for 5 seconds while your rivals pull ahead.",
	match.Flags:     "A 2-player duel. Take turns hunting MINES: find one to score and go again. Safe tile? Turn over. If behind, use your bomb once!",
	match.Coop:      "One board, live cursors, one timer. A single mine ends the run for the whole team, so talk it out. Press P to ping a tile.",
	match.Territory: "Everyone plays one huge board at the same time. Tiles you open are yours; mines stun you and cost tiles. Most land in 3 minutes wins.",
	match.LastStand: "Take turns opening tiles on one board, with 10 seconds a turn. Hit a mine or run out of time and you're out. Last one in wins.",
}

var botSpeeds = []struct {
	name  string
	delay time.Duration
}{{"Relaxed", 900 * time.Millisecond}, {"Normal", 500 * time.Millisecond}, {"Fast", 250 * time.Millisecond}, {"Brutal", 120 * time.Millisecond}}

var lobbyModes = []match.Mode{match.Race, match.Flags, match.Coop, match.Territory, match.LastStand}

type lobby struct {
	sess    netplay.Session
	host    *netplay.Host // nil when joined to someone else's room
	offline bool

	mode   match.Mode
	diff   int
	speed  int
	code   int
	status string
	hint   string
	copied int

	modeBtns []*Button
	diffBtns []*Button
	kicks    []*Button
	bs       []*Button
	start    *Button
	addBot   *Button
	speedBtn *Button
	copyBtn  *Button
	nextBtn  *Button
	ipField  TextField
	ipBtn    *Button
}

func newLobby(g *Game, sess netplay.Session, h *netplay.Host) *lobby {
	l := &lobby{sess: sess, host: h, mode: match.Race, diff: 1, speed: 1}
	leave := func() {
		g.leaveSession()
		g.goTo(newMenu(g))
	}
	l.bs = []*Button{closeButton(leave), {X: 24, Y: 430, W: 120, H: 30, Label: "Leave", On: leave}}
	if h == nil {
		return l
	}
	l.offline = h.Offline()
	for i, mode := range lobbyModes {
		b := &Button{X: 326 + i%3*96, Y: 56 + i/3*30, W: 92, H: 26, Label: mode.String()}
		b.On = func() {
			l.mode = mode
			l.configure()
		}
		l.modeBtns = append(l.modeBtns, b)
	}
	for i, c := range classicBoards {
		b := &Button{X: 326 + i*96, Y: difficultyTop, W: 92, H: 24, Label: c.name[:min(len(c.name), 12)]}
		b.On = func() {
			l.diff = i
			l.configure()
		}
		l.diffBtns = append(l.diffBtns, b)
	}
	for i := 1; i < netplay.MaxPlayers; i++ {
		l.kicks = append(l.kicks, &Button{X: 282, Y: 58 + i*26, W: 18, H: 18, Label: "x", On: func() { h.Kick(i) }})
	}
	l.addBot = &Button{X: 24, Y: 166, W: 134, H: 26, Label: "Add CPU", On: h.AddBot}
	l.speedBtn = &Button{X: 164, Y: 166, W: 136, H: 26, Label: "", On: func() {
		l.speed = (l.speed + 1) % len(botSpeeds)
		h.SetBotDelay(botSpeeds[l.speed].delay)
	}}
	l.start = &Button{X: 456, Y: 424, W: 160, H: 36, Label: "Start Match", On: func() { l.begin(g) }}
	l.copyBtn = &Button{X: 460, Y: 240, W: 70, H: 24, Label: "Copy", On: func() {
		codes := h.Codes()
		l.copied = 120
		if !copyText(codes[l.code%len(codes)].Code) {
			l.copied = -120
		}
	}}
	l.nextBtn = &Button{X: 536, Y: 240, W: 70, H: 24, Label: "Next", On: func() { l.code++ }}
	l.ipField = TextField{X: 34, Y: 382, W: 200, Max: 15}
	l.ipBtn = &Button{X: 240, Y: 380, W: 90, H: 24, Label: "Use this IP", On: func() { l.applyIP() }}
	l.bs = append(l.bs, l.addBot, l.speedBtn, l.start)
	l.bs = append(l.bs, l.modeBtns...)
	l.bs = append(l.bs, l.diffBtns...)
	l.bs = append(l.bs, l.kicks...)
	if !l.offline {
		l.bs = append(l.bs, l.copyBtn, l.nextBtn)
	}
	h.SetBotDelay(botSpeeds[l.speed].delay)
	l.configure()
	return l
}

func (l *lobby) applyIP() {
	if err := l.host.SetPublicIP(l.ipField.Value); err != nil {
		l.status = err.Error()
		return
	}
	l.status = ""
	l.code = 0
}

func (l *lobby) configure() {
	l.host.Configure(l.mode, classicBoards[l.diff].board)
}

func (l *lobby) begin(g *Game) {
	if err := l.host.Start(); err != nil {
		l.status = err.Error()
		return
	}
	l.status = ""
	g.goTo(l.playScene(g))
}

func (l *lobby) playScene(g *Game) *play {
	return newPlay(g, l.sess, PlayOptions{Title: "Online", Lobby: l, Host: l.host})
}

func (l *lobby) Update(g *Game, in *Input) {
	if err := l.sess.Err(); err != nil {
		g.notice = err.Error()
		g.leaveSession()
		g.goTo(newMenu(g))
		return
	}
	room := l.sess.Room()
	if l.host == nil && room.InMatch {
		g.goTo(l.playScene(g))
		return
	}
	if l.copied > 0 {
		l.copied--
	} else if l.copied < 0 {
		l.copied++
	}
	if l.host != nil {
		for i, b := range l.modeBtns {
			b.Selected = lobbyModes[i] == l.mode
		}
		fixedBoard := l.mode.FixedBoard()
		for i, b := range l.diffBtns {
			b.Selected = i == l.diff && !fixedBoard
			b.Disabled = fixedBoard
		}
		for i, b := range l.kicks {
			b.Disabled = i+1 >= len(room.Players)
		}
		l.addBot.Disabled = len(room.Players) >= netplay.MaxPlayers
		l.speedBtn.Label = "CPU: " + botSpeeds[l.speed].name
		lo, hi := l.mode.PlayerLimits()
		n := len(room.Players)
		l.start.Disabled = n < lo || n > hi
		l.hint = ""
		switch {
		case n < lo:
			l.hint = fmt.Sprintf("%v needs at least %d players.", l.mode, lo)
		case n > hi:
			l.hint = fmt.Sprintf("%v is for %d players. Remove someone.", l.mode, hi)
		}
	}
	updateButtons(g, in, l.bs)
	manual := l.manualIP()
	if manual {
		l.ipField.update(in)
		l.ipBtn.update(g, in)
	}
	switch {
	case !in.Key(ebiten.KeyEnter) || l.host == nil:
	case manual && l.ipField.Value != "":
		l.applyIP()
	case !l.start.Disabled:
		l.begin(g)
	}
}

func (l *lobby) manualIP() bool {
	return l.host != nil && !l.offline && l.host.NeedsManualIP()
}

func (l *lobby) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	title := "Game Lobby"
	if l.offline {
		title = "Practice vs CPU"
	}
	g.window(s, title)
	room := l.sess.Room()
	g.text(s, "Players", 26, 36, t.Text)
	for i, p := range room.Players {
		y := 58 + i*26
		g.fill(s, 26, y+2, 14, 14, playerColors[i%4])
		name := p.Name
		if i == room.You {
			name += " (you)"
		}
		g.text(s, name, 46, y+2, t.Text)
		if !p.Bot && i != room.You {
			if rec, ok := g.Profile.Rivals[p.Name]; ok {
				g.text(s, fmt.Sprintf("%d-%d", rec.Wins, rec.Losses), 214, y+2, t.Accent)
			}
		}
		if p.Wins > 0 {
			g.text(s, fmt.Sprintf("x%d", p.Wins), 250, y+2, t.Text)
		}
	}
	if l.host != nil {
		g.text(s, "Game mode", 326, 36, t.Text)
		y := blurbTop
		for _, line := range wrap(modeBlurbs[l.mode], blurbWidth) {
			g.text(s, line, 326, y, t.Text)
			y += 15
		}
		l.drawInvite(g, s)
	} else {
		g.text(s, "Mode: "+room.Mode.String(), 326, 36, t.Text)
		y := 60
		for _, line := range wrap(modeBlurbs[room.Mode], blurbWidth) {
			g.text(s, line, 326, y, t.Text)
			y += 15
		}
		if g.frame/30%2 == 0 {
			g.textCentered(s, "Waiting for the host to start...", 320, 300, 1, t.Accent)
		}
	}
	message := l.status
	if message == "" {
		message = l.hint
	}
	if message != "" {
		g.text(s, message, 150, 438, t.Accent)
	}
	drawButtons(g, s, &g.in, l.bs)
}

func (l *lobby) drawInvite(g *Game, s *ebiten.Image) {
	t := g.theme
	if l.offline {
		g.textCentered(s, "Add CPU players, pick a mode and hit Start.", 320, 260, 1, t.Text)
		g.textCentered(s, "Use the speed button to make them tougher.", 320, 276, 1, t.Text)
		return
	}
	g.sunken(s, 24, 212, 592, 196, t.Open)
	g.text(s, "Invite friends", 34, 220, t.Text)
	codes := l.host.Codes()
	info := codes[l.code%len(codes)]
	g.text(s, info.Label+" code:", 34, 240, t.Text)
	g.textScaled(s, info.Code, 34, 258, 2, t.Accent)
	y := 296
	switch {
	case l.copied > 0:
		g.text(s, "Copied to clipboard!", 34, y, t.Accent)
	case l.copied < 0:
		g.text(s, "Couldn't copy here; read the code out instead.", 34, y, t.Accent)
	default:
		g.text(s, "Friends enter this under Join Game. Sharing it", 34, y, t.Text)
		y += 15
		g.text(s, "shows them your IP address, so keep it private.", 34, y, t.Text)
	}
	if status := l.host.InternetStatus(); status != "" {
		for i, line := range wrap(status, 80) {
			g.text(s, line, 34, 340+i*15, t.Text)
		}
	}
	if l.manualIP() {
		l.ipField.draw(g, s)
		l.ipBtn.draw(g, s, &g.in)
	}
}

// wrap splits text into lines of at most width characters.
func wrap(text string, width int) []string {
	var lines []string
	line := ""
	word := ""
	flush := func() {
		if line == "" {
			line = word
		} else if len(line)+1+len(word) <= width {
			line += " " + word
		} else {
			lines = append(lines, line)
			line = word
		}
		word = ""
	}
	for _, r := range text {
		if r == ' ' {
			flush()
			continue
		}
		word += string(r)
	}
	flush()
	return append(lines, line)
}

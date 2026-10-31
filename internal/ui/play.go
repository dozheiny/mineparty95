package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/match"
	"mineparty/internal/netplay"
)

// Screen layout of the game board.
const (
	boardTop    = 92
	boardBottom = 468
	boardMaxW   = 620
	maxWave     = 26 // frames a flood-open animation may take
)

// PlayOptions says what kind of game a play screen is showing.
type PlayOptions struct {
	Title    string
	BoardKey string
	Daily    bool
	Date     string
	Restart  func() // instant replay, for solo games
	Host     *netplay.Host
	Lobby    *lobby
}

type play struct {
	sess netplay.Session
	opt  PlayOptions
	v    *match.View

	visibleAt []int // frame at which each tile's opening animation arrives
	lastTile  int
	press     int
	hover     int
	bomb      bool
	lastCount int
	wasStun   bool
	res       *results

	ts, bx, by int
	smiley     *Button
	quit       *Button
	bombBtn    *Button
	bs         []*Button
}

func newPlay(g *Game, sess netplay.Session, opt PlayOptions) *play {
	p := &play{sess: sess, opt: opt, press: -1, hover: -1, lastTile: -1, lastCount: -1}
	p.quit = closeButton(func() { p.leave(g) })
	p.smiley = &Button{X: ScreenW/2 - 13, Y: 30, W: 26, H: 26, On: func() { p.restart(g) }}
	p.bombBtn = &Button{X: 16, Y: 64, W: 96, H: 22, Label: "Bomb (B)", On: func() { p.bomb = !p.bomb }}
	p.bs = []*Button{p.quit, p.smiley}
	return p
}

func (p *play) leave(g *Game) {
	g.leaveSession()
	g.goTo(newMenu(g))
}

func (p *play) restart(g *Game) {
	switch {
	case p.opt.Restart != nil:
		p.opt.Restart()
	case p.opt.Host != nil && p.v != nil && p.v.Phase == match.Over:
		if err := p.opt.Host.Start(); err != nil {
			g.goTo(p.opt.Lobby)
			p.opt.Lobby.status = err.Error()
		}
	}
}

// online reports whether other people are relying on this session. Leaving
// a hosted game ends it for everyone, so that takes a deliberate click.
func (p *play) online() bool {
	return p.opt.Lobby != nil && (p.opt.Host == nil || !p.opt.Host.Offline())
}

func (p *play) multi() bool { return p.v != nil && p.v.Mode != match.Classic }

func (p *play) you() *match.PlayerView {
	if p.v == nil || p.v.You < 0 || p.v.You >= len(p.v.Players) {
		return nil
	}
	return &p.v.Players[p.v.You]
}

func (p *play) Update(g *Game, in *Input) {
	if err := p.sess.Err(); err != nil {
		g.notice = err.Error()
		p.leave(g)
		return
	}
	if v := p.sess.View(); v != nil {
		p.sync(g, v)
	}
	if in.Key(ebiten.KeyEscape) && !p.online() {
		p.leave(g)
		return
	}
	if p.v == nil {
		updateButtons(g, in, p.bs[:1])
		return
	}
	updateButtons(g, in, p.bs)
	if in.Key(ebiten.KeySpace) || in.Key(ebiten.KeyF2) {
		p.restart(g)
		return
	}
	if p.res != nil {
		p.res.update(g, in)
		return
	}
	if p.v.Phase == match.Playing {
		p.play(g, in)
	}
}

// sync absorbs the newest view and fires the feedback for what changed.
func (p *play) sync(g *Game, v *match.View) {
	if p.v == nil || v.Round != p.v.Round {
		p.v = nil
		p.res = nil
		p.bomb = false
		p.press = -1
		p.wasStun = false
		p.lastCount = -1
		p.visibleAt = make([]int, len(v.Tiles))
		p.layout(v)
	}
	old := p.v
	p.v = v

	if v.Phase == match.Countdown {
		if n := v.Countdown/1000 + 1; n != p.lastCount {
			p.lastCount = n
			g.sfx.Play(sndTick)
		}
	}
	if old != nil {
		if old.Phase == match.Countdown && v.Phase == match.Playing {
			g.sfx.Play(sndGo)
		}
		p.animate(g, old, v)
		if v.Mode == match.Flags && v.Phase == match.Playing && old.Turn != v.Turn && v.Turn == v.You {
			g.sfx.Play(sndTurn)
		}
	}
	if you := p.you(); you != nil {
		stunned := you.StunMs > 0
		if stunned && !p.wasStun {
			g.sfx.Play(sndBoom)
			g.shake = 14
			p.explode(g, p.lastTile, 50)
		}
		p.wasStun = stunned
	}
	if v.Phase == match.Over && p.res == nil {
		p.res = newResults(g, p, v)
	}
}

func (p *play) tileCenter(tile int) (float64, float64) {
	return float64(p.bx + tile%p.v.W*p.ts + p.ts/2), float64(p.by + tile/p.v.W*p.ts + p.ts/2)
}

func (p *play) explode(g *Game, tile, n int) {
	if tile < 0 || p.v == nil {
		return
	}
	x, y := p.tileCenter(tile)
	g.burst(x, y, n, 3, g.theme.Boom, g.theme.Flag, rgb(255, 200, 0))
}

func (p *play) animate(g *Game, old, v *match.View) {
	if len(old.Tiles) != len(v.Tiles) {
		return
	}
	opened := 0
	for i, t := range v.Tiles {
		was := old.Tiles[i]
		if t <= 8 && was > 8 {
			opened++
			p.visibleAt[i] = g.frame + p.wave(i, v.W)
		}
		if v.Phase != match.Over && (t == match.TileMine) && was != match.TileMine && v.Owners != nil && int(v.Owners[i]) > 0 {
			owner := int(v.Owners[i]) - 1
			g.sfx.Play(sndClaim)
			x, y := p.tileCenter(i)
			g.burst(x, y, 24, 2, playerColors[owner%4], rgb(255, 225, 0))
		}
	}
	if opened > 0 {
		g.sfx.PlayOpen(opened)
	}
}

// wave delays a tile by its distance from the click, so a big opening
// ripples outward instead of appearing all at once.
func (p *play) wave(tile, w int) int {
	if p.lastTile < 0 {
		return 0
	}
	dx := float64(tile%w - p.lastTile%w)
	dy := float64(tile/w - p.lastTile/w)
	return min(int(math.Hypot(dx, dy)), maxWave)
}

func (p *play) layout(v *match.View) {
	p.ts = 32
	if v.W*32 > boardMaxW || v.H*32 > boardBottom-boardTop {
		p.ts = 16
	}
	top := boardTop
	if v.Mode == match.Classic {
		top = 64
	}
	p.bx = (ScreenW - v.W*p.ts) / 2
	p.by = top + (boardBottom-top-v.H*p.ts)/2
}

func (p *play) tileAt(x, y int) int {
	v := p.v
	tx, ty := x-p.bx, y-p.by
	if tx < 0 || ty < 0 || tx >= v.W*p.ts || ty >= v.H*p.ts {
		return -1
	}
	return ty/p.ts*v.W + tx/p.ts
}

func (p *play) bombReady() bool {
	v := p.v
	you := p.you()
	if v.Mode != match.Flags || you == nil || you.BombUsed || v.Turn != v.You || len(v.Players) != 2 {
		return false
	}
	return you.Score < v.Players[1-v.You].Score
}

func (p *play) play(g *Game, in *Input) {
	v := p.v
	you := p.you()
	if you == nil {
		return
	}
	ready := p.bombReady()
	if ready {
		p.bombBtn.Selected = p.bomb
		p.bombBtn.update(g, in)
		if in.Key(ebiten.KeyB) {
			p.bomb = !p.bomb
		}
	} else {
		p.bomb = false
	}

	tile := p.tileAt(in.X, in.Y)
	if tile != p.hover {
		p.hover = tile
		if p.multi() {
			p.sess.Send(match.Action{Kind: match.Cursor, Tile: tile})
		}
	}
	if you.StunMs > 0 {
		p.press = -1
		return
	}
	send := func(kind match.ActionKind) {
		p.lastTile = tile
		p.sess.Send(match.Action{Kind: kind, Tile: tile})
	}
	flagging := in.RightPress || in.LeftPress && in.Ctrl
	switch {
	case tile < 0:
	case flagging && v.Mode != match.Flags:
		if v.Tiles[tile] <= 8 {
			send(match.Chord)
		} else {
			g.sfx.Play(sndFlag)
			send(match.Flag)
		}
	case in.MiddlePress || in.Key(ebiten.KeyP):
		if p.multi() {
			g.sfx.Play(sndClick)
			send(match.Ping)
		} else {
			send(match.Chord)
		}
	case in.LeftPress:
		p.press = tile
	}
	if in.LeftRelease {
		if p.press >= 0 && p.press == tile {
			switch {
			case p.bomb:
				p.bomb = false
				send(match.Bomb)
				g.sfx.Play(sndBoom)
				g.shake = 8
			case v.Tiles[tile] == match.TileHidden:
				send(match.Reveal)
			case v.Tiles[tile] <= 8:
				send(match.Chord)
			}
		}
		p.press = -1
	}
	if !in.LeftDown && !in.LeftRelease {
		p.press = -1
	}
}

func (p *play) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	title := p.opt.Title
	if p.v != nil && p.multi() {
		title = p.v.Mode.String()
	}
	g.window(s, "Mine Party 95 - "+title)
	if p.v == nil {
		g.textCentered(s, "Waiting for the match to begin...", ScreenW/2, 220, 1, t.Text)
		p.quit.draw(g, s, &g.in)
		return
	}
	p.drawHeader(g, s)
	if p.multi() {
		p.drawCards(g, s)
	}
	p.drawBoard(g, s)
	p.drawOverlays(g, s)
	if p.res != nil {
		p.res.draw(g, s)
	}
}

func (p *play) face() int {
	v := p.v
	you := p.you()
	switch {
	case v.Phase == match.Over && (v.Won && (v.Mode == match.Classic || v.Mode == match.Coop) || v.Winner == v.You):
		return faceCool
	case v.Phase == match.Over, you != nil && you.StunMs > 0:
		return faceDead
	case p.press >= 0:
		return faceScared
	}
	return faceHappy
}

func (p *play) drawHeader(g *Game, s *ebiten.Image) {
	v := p.v
	g.sunken(s, 12, 28, ScreenW-24, 32, g.theme.Face)
	left := v.Mines
	for _, t := range v.Tiles {
		// A lost board shows every mine, and those weren't found by anyone.
		if t == match.TileFlag || t == match.TileWrongFlag || t == match.TileMine && v.Phase != match.Over {
			left--
		}
	}
	g.led(s, 18, 30, left, 3)
	seconds := v.ElapsedMs / 1000
	switch {
	case v.Mode == match.Flags && v.Phase == match.Playing:
		seconds = (v.TurnMs + 999) / 1000
	case v.Mode == match.Territory && v.Phase == match.Playing:
		seconds = (v.LimitMs + 999) / 1000
	}
	g.led(s, ScreenW-61, 30, seconds, 3)
	p.smiley.draw(g, s, &g.in)
	g.blit(s, g.art.Face[p.face()], p.smiley.X+3, p.smiley.Y+3, 1)
	p.quit.draw(g, s, &g.in)
}

func (p *play) drawCards(g *Game, s *ebiten.Image) {
	t := g.theme
	v := p.v
	n := len(v.Players)
	x := 12
	if p.bombReady() {
		p.bombBtn.draw(g, s, &g.in)
		x = 116
	}
	w := (ScreenW - 12 - x) / n
	room := p.sess.Room()
	for i, pl := range v.Players {
		cx := x + i*w
		fill := t.Face
		if v.Mode == match.Flags && v.Phase == match.Playing && v.Turn == i {
			fill = t.Hi
		}
		g.sunken(s, cx, 62, w-4, 26, fill)
		g.fill(s, cx+3, 65, 6, 20, playerColors[i%4])
		name := pl.Name
		if i == v.You {
			name = "> " + name
		}
		if pl.Left {
			name += " (left)"
		}
		maxLen := (w - 16) / glyphW
		if len(name) > maxLen {
			name = name[:maxLen]
		}
		g.text(s, name, cx+12, 63, t.Text)
		if i < len(room.Players) && room.Players[i].Wins > 0 {
			wins := fmt.Sprintf("x%d", room.Players[i].Wins)
			g.text(s, wins, cx+w-8-textWidth(wins, 1), 63, t.Text)
		}
		switch {
		case pl.StunMs > 0:
			g.text(s, fmt.Sprintf("STUNNED %.1f", float64(pl.StunMs)/1000), cx+12, 74, t.Accent)
		case v.Mode == match.Race:
			g.bar(s, cx+12, 76, w-24, 9, float64(pl.Progress)/100, playerColors[i%4])
		case v.Mode == match.Flags:
			g.text(s, fmt.Sprintf("%d mines", pl.Score), cx+12, 74, t.Text)
		default:
			g.text(s, fmt.Sprintf("%d tiles", pl.Score), cx+12, 74, t.Text)
		}
	}
}

func (p *play) drawBoard(g *Game, s *ebiten.Image) {
	v := p.v
	scale := p.ts / 16
	g.sunken(s, p.bx-3, p.by-3, v.W*p.ts+6, v.H*p.ts+6, g.theme.Lo)
	for i, tile := range v.Tiles {
		x, y := p.bx+i%v.W*p.ts, p.by+i/v.W*p.ts
		if tile <= 8 && g.frame < p.visibleAt[i] {
			tile = match.TileHidden
		}
		p.drawTile(g, s, i, tile, x, y, scale)
	}
	for _, pg := range v.Pings {
		x, y := p.tileCenter(pg.Tile)
		r := 8 + (g.frame*2)%14
		c := playerColors[pg.Player%4]
		g.fill(s, int(x)-r, int(y)-r, 2*r, 2, c)
		g.fill(s, int(x)-r, int(y)+r, 2*r, 2, c)
		g.fill(s, int(x)-r, int(y)-r, 2, 2*r, c)
		g.fill(s, int(x)+r, int(y)-r, 2, 2*r, c)
	}
	if v.Mode != match.Race {
		for i, pl := range v.Players {
			if i == v.You || pl.Cursor < 0 || pl.Cursor >= len(v.Tiles) || pl.Left {
				continue
			}
			x, y := p.bx+pl.Cursor%v.W*p.ts, p.by+pl.Cursor/v.W*p.ts
			c := playerColors[i%4]
			g.fill(s, x, y, p.ts, 2, c)
			g.fill(s, x, y+p.ts-2, p.ts, 2, c)
			g.fill(s, x, y, 2, p.ts, c)
			g.fill(s, x+p.ts-2, y, 2, p.ts, c)
		}
	}
	if p.bomb && p.hover >= 0 {
		cx, cy := p.hover%v.W, p.hover/v.W
		g.fill(s, p.bx+(cx-2)*p.ts, p.by+(cy-2)*p.ts, 5*p.ts, 5*p.ts, color.RGBA{255, 0, 0, 70})
	}
}

func (p *play) drawTile(g *Game, s *ebiten.Image, i int, tile byte, x, y, scale int) {
	a := g.art
	v := p.v
	switch {
	case tile <= 8:
		g.blit(s, a.Open, x, y, scale)
		if tile > 0 {
			g.blit(s, a.Num[tile], x, y, scale)
		}
		if v.Owners != nil && v.Owners[i] > 0 {
			c := playerColors[(v.Owners[i]-1)%4]
			g.fill(s, x, y, p.ts, p.ts, color.RGBA{c.R / 3, c.G / 3, c.B / 3, 85})
		}
	case tile == match.TileHidden:
		if i == p.press && !p.bomb {
			g.blit(s, a.Open, x, y, scale)
		} else {
			g.blit(s, a.Hidden, x, y, scale)
		}
	case tile == match.TileFlag:
		g.blit(s, a.Hidden, x, y, scale)
		g.blit(s, a.Flag, x, y, scale)
	case tile == match.TileMine:
		if v.Owners != nil && v.Owners[i] > 0 && v.Mode == match.Flags {
			g.blit(s, a.Hidden, x, y, scale)
			g.blit(s, a.PlayerFlag[(v.Owners[i]-1)%4], x, y, scale)
			return
		}
		g.blit(s, a.Open, x, y, scale)
		g.blit(s, a.Mine, x, y, scale)
	case tile == match.TileExploded:
		g.blit(s, a.Boom, x, y, scale)
		g.blit(s, a.Mine, x, y, scale)
	case tile == match.TileWrongFlag:
		g.blit(s, a.Open, x, y, scale)
		g.blit(s, a.Mine, x, y, scale)
		g.blit(s, a.Wrong, x, y, scale)
	}
}

func (p *play) drawOverlays(g *Game, s *ebiten.Image) {
	t := g.theme
	v := p.v
	cx := p.bx + v.W*p.ts/2
	cy := p.by + v.H*p.ts/2
	banner := func(str string, scale int, c color.Color) {
		w := textWidth(str, scale) + 16
		h := glyphH*scale + 8
		g.fill(s, cx-w/2, cy-h/2, w, h, color.RGBA{0, 0, 0, 170})
		g.textCentered(s, str, cx, cy-h/2+4, scale, c)
	}
	switch {
	case v.Phase == match.Countdown:
		banner(fmt.Sprint(v.Countdown/1000+1), 6, rgb(255, 255, 255))
	case v.Phase == match.Playing:
		if you := p.you(); you != nil && you.StunMs > 0 {
			banner(fmt.Sprintf("STUNNED %.1f", float64(you.StunMs)/1000), 3, rgb(255, 80, 60))
		}
		if v.Mode == match.Flags && v.Turn == v.You {
			g.text(s, "Your turn: find a mine!", 345, 37, t.Accent)
		}
	}
}

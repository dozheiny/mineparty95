package match

import (
	"slices"
	"time"

	"mineparty/internal/engine"
)

// Values of a tile in a View, alongside the numbers 0 to 8.
const (
	TileHidden    byte = 9
	TileFlag      byte = 10
	TileMine      byte = 11
	TileExploded  byte = 12
	TileWrongFlag byte = 13
	TileQuestion  byte = 14
)

// PlayerView is what everyone may know about a player.
type PlayerView struct {
	Name     string
	Score    int
	Progress int // percent of safe tiles opened
	StunMs   int
	Hits     int
	Clicks   int
	Flagged  bool
	Cursor   int
	Left     bool
	BombUsed bool
	Lives    int  // left in Survival
	Out      bool // knocked out of Last Stand
}

// PingView is a tile a player is pointing at.
type PingView struct {
	Tile, Player int
}

// View is everything one player may see. It is built only from opened
// tiles, so a client can never read a mine it hasn't uncovered. It is also
// the whole network protocol for game state.
type View struct {
	Round     int
	Mode      Mode
	W, H      int
	Mines     int
	Phase     Phase
	You       int
	Countdown int // ms until play starts
	ElapsedMs int
	LimitMs   int // ms remaining, for timed modes
	Tiles     []byte
	Owners    []byte
	Players   []PlayerView
	Turn      int
	TurnMs    int
	Winner    int
	Won       bool
	BV3       int
	Boards    int // cleared in Blitz
	Pings     []PingView
}

// View builds the view for one player at time now.
func (m *Match) View(player int, now time.Duration) View {
	v := View{
		Round:  m.cfg.Round,
		Mode:   m.cfg.Mode,
		W:      m.cfg.Board.W,
		H:      m.cfg.Board.H,
		Mines:  m.cfg.Board.Mines,
		Phase:  m.phase,
		You:    player,
		Turn:   m.turn,
		Winner: m.winner,
		Won:    m.won,
		Boards: m.boards,
		Owners: slices.Clone(m.owners),
	}
	if m.board != nil {
		v.W, v.H, v.Mines = m.board.W, m.board.H, m.board.Mines
	}
	end := now
	if m.phase == Over {
		end = m.stopAt
		if m.board != nil {
			v.BV3 = m.board.ThreeBV()
		}
	}
	switch {
	case m.phase == Countdown:
		v.Countdown = max(int((m.startAt - now).Milliseconds()), 0)
	case m.ticking:
		v.ElapsedMs = max(int((end - m.startAt).Milliseconds()), 0)
	}
	if m.cfg.Mode == Territory && m.phase == Playing {
		v.LimitMs = max(int((m.startAt + TerritoryTime - now).Milliseconds()), 0)
	}
	if m.cfg.Mode == Blitz && m.phase == Playing {
		v.LimitMs = max(int((m.deadline - now).Milliseconds()), 0)
	}
	if m.cfg.Mode.TurnBased() && m.phase == Playing {
		v.TurnMs = max(int((m.turnEnds - now).Milliseconds()), 0)
	}
	for _, p := range m.players {
		v.Players = append(v.Players, m.playerView(p, now))
	}
	for _, g := range m.pings {
		if now < g.at+PingTime {
			v.Pings = append(v.Pings, PingView{Tile: g.tile, Player: g.player})
		}
	}
	v.Tiles = m.tiles(player, v.W*v.H)
	return v
}

func (m *Match) playerView(p *player, now time.Duration) PlayerView {
	pv := PlayerView{
		Name:     p.name,
		Score:    p.score,
		Hits:     p.hits,
		Clicks:   p.clicks,
		Flagged:  p.flagged,
		Cursor:   p.cursor,
		Left:     p.left,
		BombUsed: p.bombUsed,
		Lives:    p.lives,
		Out:      p.out,
	}
	if now < p.stunUntil && m.phase == Playing {
		pv.StunMs = int((p.stunUntil - now).Milliseconds())
	}
	if p.field != nil {
		safe := m.board.Size() - m.board.Mines
		if safe > 0 {
			pv.Progress = 100 * (safe - p.field.Left()) / safe
		}
	}
	return pv
}

func (m *Match) tiles(player, size int) []byte {
	out := make([]byte, size)
	for i := range out {
		out[i] = TileHidden
	}
	if m.board == nil || player < 0 || player >= len(m.players) {
		return out
	}
	p := m.players[player]
	f := p.field
	over := m.phase == Over
	for i := range out {
		mark := f.Mark(i)
		if p.marks != nil {
			mark = p.marks[i]
		}
		flagged := mark == engine.FlagMark
		mine := m.board.IsMine(i)
		switch {
		case f.Open(i):
			out[i] = byte(m.board.Adj(i))
		case f.Exposed(i):
			out[i] = TileMine
		case i == m.boom:
			out[i] = TileExploded
		case over && flagged && !mine:
			out[i] = TileWrongFlag
		case flagged:
			out[i] = TileFlag
		case over && mine && f.Cleared():
			out[i] = TileFlag
		case over && mine:
			out[i] = TileMine
		case mark == engine.QuestionMark:
			out[i] = TileQuestion
		}
	}
	return out
}

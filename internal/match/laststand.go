package match

import "time"

// applyLastStand runs Last Stand: players take turns opening tiles on a shared
// board, and a mine puts them out. Flags are private and need no turn, so that
// nobody can trap a rival with a misleading flag.
func (m *Match) applyLastStand(now time.Duration, index int, p *player, a Action) {
	f := p.field
	if a.Kind == Flag {
		if f.Hidden(a.Tile) {
			p.flags[a.Tile] = !p.flags[a.Tile]
			p.flagged = true
		}
		return
	}
	if index != m.turn {
		return
	}
	var opened []int
	mine := -1
	switch a.Kind {
	case Reveal:
		if p.flags[a.Tile] {
			return
		}
		var hit bool
		if opened, hit = f.Reveal(a.Tile); hit {
			mine = a.Tile
		}
	case Chord:
		opened, mine = f.ChordWith(a.Tile, p.flags)
	}
	// A move that does nothing doesn't use up the turn.
	if len(opened) == 0 && mine < 0 {
		return
	}
	p.clicks++
	p.score += len(opened)
	if mine >= 0 {
		f.Expose(mine)
		p.hits++
		m.eliminate(now, index)
		return
	}
	if f.Cleared() {
		m.finishByScore(now)
		return
	}
	m.advanceTurn(now)
}

// eliminate puts a player out, which can end the match.
func (m *Match) eliminate(now time.Duration, index int) {
	m.players[index].out = true
	last, count := -1, 0
	for i, p := range m.players {
		if !p.out && !p.left {
			last = i
			count++
		}
	}
	switch {
	case count == 0:
		m.finish(now, -1, false)
	case count == 1:
		m.finish(now, last, true)
	case m.players[index].field.Cleared():
		m.finishByScore(now)
	case m.turn == index:
		m.advanceTurn(now)
	}
}

// advanceTurn moves to the next player still in the match.
func (m *Match) advanceTurn(now time.Duration) {
	m.turnEnds = now + LastStandTurn
	for range m.players {
		m.turn = (m.turn + 1) % len(m.players)
		if p := m.players[m.turn]; !p.out && !p.left {
			return
		}
	}
}

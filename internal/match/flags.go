package match

import "time"

// applyFlags runs Mine Flags: players take turns hunting mines on a shared
// board. A found mine scores and keeps the turn; anything else passes it.
func (m *Match) applyFlags(now time.Duration, index int, a Action) {
	if index != m.turn {
		return
	}
	switch a.Kind {
	case Reveal:
		m.flagsReveal(now, index, a.Tile)
	case Bomb:
		m.flagsBomb(now, index, a.Tile)
	}
}

func (m *Match) flagsReveal(now time.Duration, index, tile int) {
	p := m.players[index]
	f := p.field
	if !f.Hidden(tile) {
		return
	}
	p.clicks++
	if m.board.IsMine(tile) {
		m.claimMine(index, tile)
		m.turnEnds = now + TurnTime
		m.checkFlagsWin(now)
		return
	}
	f.Reveal(tile)
	m.passTurn(now)
}

// flagsBomb is the comeback mechanic: once per match, a player who is behind
// may blast a 5x5 area, claiming every mine inside it.
func (m *Match) flagsBomb(now time.Duration, index, tile int) {
	p := m.players[index]
	if p.bombUsed || p.score >= m.players[1-index].score {
		return
	}
	p.bombUsed = true
	p.clicks++
	f := p.field
	cx, cy := tile%m.board.W, tile/m.board.W
	for y := max(cy-bombRadius, 0); y <= min(cy+bombRadius, m.board.H-1); y++ {
		for x := max(cx-bombRadius, 0); x <= min(cx+bombRadius, m.board.W-1); x++ {
			i := y*m.board.W + x
			switch {
			case !f.Hidden(i):
			case m.board.IsMine(i):
				m.claimMine(index, i)
			default:
				f.Reveal(i)
			}
		}
	}
	m.passTurn(now)
	m.checkFlagsWin(now)
}

func (m *Match) claimMine(index, tile int) {
	p := m.players[index]
	p.field.Expose(tile)
	p.score++
	m.owners[tile] = byte(index + 1)
}

func (m *Match) checkFlagsWin(now time.Duration) {
	for i, p := range m.players {
		if p.score > m.board.Mines/2 {
			m.finish(now, i, true)
			return
		}
	}
}

func (m *Match) passTurn(now time.Duration) {
	m.turnEnds = now + TurnTime
	if len(m.players) > 0 {
		m.turn = (m.turn + 1) % len(m.players)
	}
}

package match

import "time"

// applyBlitz runs Blitz: clear as many boards as the clock allows. Each clear
// deals a new board and buys time, and each mine stuns and burns time.
func (m *Match) applyBlitz(now time.Duration, p *player, a Action) {
	f := p.field
	switch a.Kind {
	case Flag:
		f.CycleMark(a.Tile)
		p.flagged = true
	case Reveal, Chord:
		p.clicks++
		opened, mine := perform(f, a)
		p.score += len(opened)
		if mine >= 0 {
			f.Expose(mine)
			p.hits++
			p.stunUntil = now + BlitzStun
			m.deadline -= BlitzPenalty
		}
		if f.Cleared() {
			m.deadline += blitzBonus(m.boards)
			m.boards++
			m.nextBoard()
		}
	}
}

// blitzBonus is the time earned for clearing a board. It shrinks as a run goes
// on, or a fast enough player would bank time forever.
func blitzBonus(cleared int) time.Duration {
	return max(BlitzBonus-time.Duration(cleared)*2*time.Second, BlitzMinBonus)
}

// nextBoard deals the following board of a run.
func (m *Match) nextBoard() {
	m.cfg.Seed++
	if m.cfg.Layout != nil {
		m.setBoard(m.cfg.Layout)
		return
	}
	m.setBoard(m.generate(m.cfg.Start))
}

// applySurvival runs Survival: one big board and a few lives to clear it with.
func (m *Match) applySurvival(now time.Duration, p *player, a Action) {
	f := p.field
	switch a.Kind {
	case Flag:
		f.CycleMark(a.Tile)
		p.flagged = true
	case Reveal, Chord:
		p.clicks++
		opened, mine := perform(f, a)
		p.score += len(opened)
		if mine >= 0 {
			p.hits++
			p.lives--
			if p.lives <= 0 {
				m.boom = mine
				m.finish(now, -1, false)
				return
			}
			f.Expose(mine)
		}
		if f.Cleared() {
			m.finish(now, -1, true)
		}
	}
}

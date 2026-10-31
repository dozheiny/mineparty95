package match

import (
	"slices"
	"time"
)

// applyTerritory runs Territory: everyone plays one huge board at once, each
// opened tile is claimed for its opener, and a mine stuns and costs tiles.
// Flags are private so that nobody can block a rival by flagging safe tiles.
func (m *Match) applyTerritory(now time.Duration, index int, p *player, a Action) {
	f := p.field
	switch a.Kind {
	case Flag:
		if f.Hidden(a.Tile) {
			p.flags[a.Tile] = !p.flags[a.Tile]
			p.flagged = true
		}
		return
	case Reveal:
		if p.flags[a.Tile] {
			return
		}
		p.clicks++
		opened, hit := f.Reveal(a.Tile)
		m.claimAll(index, opened)
		if hit {
			m.territoryHit(now, p, a.Tile)
		}
	case Chord:
		p.clicks++
		opened, mine := f.ChordWith(a.Tile, p.flags)
		m.claimAll(index, opened)
		if mine >= 0 {
			m.territoryHit(now, p, mine)
		}
	}
	if f.Cleared() {
		m.finishByScore(now)
	}
}

func (m *Match) claim(index, tile int) {
	opened, _ := m.players[index].field.Reveal(tile)
	m.claimAll(index, opened)
}

func (m *Match) claimAll(index int, tiles []int) {
	p := m.players[index]
	for _, t := range tiles {
		m.owners[t] = byte(index + 1)
	}
	p.claims = append(p.claims, tiles...)
	p.score += len(tiles)
}

// territoryHit exposes the mine and hands back the player's newest tiles.
// Those tiles stay open but belong to nobody, and can't be reclaimed.
func (m *Match) territoryHit(now time.Duration, p *player, tile int) {
	p.field.Expose(tile)
	p.hits++
	p.stunUntil = now + TerritoryStun
	loss := min(max(p.score/10, TerritoryMinLoss), len(p.claims))
	cut := len(p.claims) - loss
	for _, t := range p.claims[cut:] {
		m.owners[t] = 0
	}
	p.claims = slices.Delete(p.claims, cut, len(p.claims))
	p.score -= loss
}

// finishByScore ends the match in favour of the highest score still in play.
// A tie has no winner.
func (m *Match) finishByScore(now time.Duration) {
	best, winner := -1, -1
	for i, p := range m.players {
		switch {
		case p.left, p.out:
		case p.score > best:
			best, winner = p.score, i
		case p.score == best:
			winner = -1
		}
	}
	m.finish(now, winner, winner >= 0)
}

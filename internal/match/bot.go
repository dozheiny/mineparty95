package match

import (
	"math/rand/v2"

	"mineparty/internal/engine"
)

// Bot is a CPU player. It decides from a View alone, exactly as a person
// must, so it has no access to hidden mines.
type Bot struct {
	player int
	rng    *rand.Rand
}

// NewBot returns a bot playing as the given player.
func NewBot(player int, seed uint64) *Bot {
	return &Bot{player: player, rng: rand.New(rand.NewPCG(seed, 0x626f74))}
}

// SetPlayer moves the bot to another seat.
func (b *Bot) SetPlayer(player int) { b.player = player }

// Next picks the bot's move. It reports false if the bot should wait.
func (b *Bot) Next(v View) (Action, bool) {
	if v.Phase != Playing || b.player >= len(v.Players) || v.Players[b.player].StunMs > 0 {
		return Action{}, false
	}
	if v.Mode == Flags && v.Turn != b.player {
		return Action{}, false
	}
	visible := make([]int8, len(v.Tiles))
	minesLeft := v.Mines
	for i, t := range v.Tiles {
		switch {
		case t <= 8:
			visible[i] = int8(t)
		case t == TileMine || t == TileExploded:
			visible[i] = engine.KnownMine
			minesLeft--
		default:
			visible[i] = engine.Hidden
		}
	}
	safe, mines := engine.Deduce(v.W, v.H, visible, minesLeft)

	reveal := func(tile int) (Action, bool) {
		return Action{Player: b.player, Kind: Reveal, Tile: tile}, true
	}
	if v.Mode == Flags {
		// The aim here is to find mines, and opening a safe tile only
		// helps the rival, so safe tiles are the last resort.
		if len(mines) > 0 {
			return reveal(b.pick(mines))
		}
		if tile := b.guess(visible, safe); tile >= 0 {
			return reveal(tile)
		}
		return reveal(b.pick(safe))
	}
	if len(safe) > 0 {
		return reveal(b.pick(safe))
	}
	if tile := b.guess(visible, mines); tile >= 0 {
		return reveal(tile)
	}
	return Action{}, false
}

func (b *Bot) pick(tiles []int) int { return tiles[b.rng.IntN(len(tiles))] }

// guess picks a hidden tile outside avoid, or -1 if none remains.
func (b *Bot) guess(visible []int8, avoid []int) int {
	skip := make(map[int]bool, len(avoid))
	for _, t := range avoid {
		skip[t] = true
	}
	var options []int
	for i, t := range visible {
		if t == engine.Hidden && !skip[i] {
			options = append(options, i)
		}
	}
	if len(options) == 0 {
		return -1
	}
	return b.pick(options)
}

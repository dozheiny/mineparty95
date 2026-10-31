package engine

import "slices"

// Values of a visible tile, alongside the numbers 0 to 8.
const (
	Hidden    int8 = -1 // nothing is known about the tile
	KnownMine int8 = -2 // the tile is established to be a mine
	knownSafe int8 = -3 // solver-internal: proven safe, number not yet seen
)

const maxNoGuessAttempts = 50000

type constraint struct {
	cells []int // hidden tiles, ascending
	need  int   // mines among them
}

// Deduce works out what logic alone proves from a visible board. minesLeft
// is how many mines are not yet KnownMine, or -1 if that is unknown. The
// results are sorted ascending.
func Deduce(w, h int, visible []int8, minesLeft int) (safe, mines []int) {
	v := slices.Clone(visible)
	for progress := true; progress; {
		progress = false
		found := func(cells []int, value int8) {
			for _, c := range cells {
				if v[c] != Hidden {
					continue
				}
				v[c] = value
				progress = true
				if value == KnownMine {
					mines = append(mines, c)
					if minesLeft > 0 {
						minesLeft--
					}
				} else {
					safe = append(safe, c)
				}
			}
		}

		cons := constraints(w, h, v)
		for _, c := range cons {
			switch c.need {
			case 0:
				found(c.cells, knownSafe)
			case len(c.cells):
				found(c.cells, KnownMine)
			}
		}
		if progress {
			continue
		}
		comparePairs(w*h, cons, found)
		if progress {
			continue
		}
		applyMineCount(v, minesLeft, found)
	}
	slices.Sort(safe)
	slices.Sort(mines)
	return safe, mines
}

func constraints(w, h int, v []int8) []constraint {
	var cons []constraint
	var buf [8]int
	for i, value := range v {
		if value < 0 {
			continue
		}
		c := constraint{need: int(value)}
		for _, n := range neighbors(w, h, i, buf[:0]) {
			switch v[n] {
			case Hidden:
				c.cells = append(c.cells, n)
			case KnownMine:
				c.need--
			}
		}
		if len(c.cells) > 0 {
			cons = append(cons, c)
		}
	}
	return cons
}

// comparePairs compares overlapping constraints. If A needs as many more
// mines than B as A has tiles outside B, then all of those tiles are mines
// and every tile only B has is safe.
func comparePairs(size int, cons []constraint, found func([]int, int8)) {
	touching := make([][]int, size)
	for ci, c := range cons {
		for _, cell := range c.cells {
			touching[cell] = append(touching[cell], ci)
		}
	}
	for ai, a := range cons {
		for _, cell := range a.cells {
			for _, bi := range touching[cell] {
				if bi != ai {
					compare(a, cons[bi], found)
				}
			}
		}
	}
}

func compare(a, b constraint, found func([]int, int8)) {
	onlyA := without(a.cells, b.cells)
	if len(onlyA) == 0 || a.need-b.need != len(onlyA) {
		return
	}
	found(onlyA, KnownMine)
	found(without(b.cells, a.cells), knownSafe)
}

// without returns the tiles of a that are not in b. Both are ascending.
func without(a, b []int) []int {
	var out []int
	j := 0
	for _, x := range a {
		for j < len(b) && b[j] < x {
			j++
		}
		if j == len(b) || b[j] != x {
			out = append(out, x)
		}
	}
	return out
}

func applyMineCount(v []int8, minesLeft int, found func([]int, int8)) {
	if minesLeft < 0 {
		return
	}
	var hidden []int
	for i, value := range v {
		if value == Hidden {
			hidden = append(hidden, i)
		}
	}
	switch {
	case minesLeft == 0:
		found(hidden, knownSafe)
	case minesLeft == len(hidden):
		found(hidden, KnownMine)
	}
}

// Solvable reports whether the board can be cleared from start with logic
// alone, never needing a guess.
func Solvable(b *Board, start int) bool {
	f := NewField(b)
	f.Reveal(start)
	v := make([]int8, b.Size())
	for i := range v {
		v[i] = Hidden
	}
	minesLeft := b.Mines
	for !f.Cleared() {
		for i := range v {
			if f.Open(i) {
				v[i] = b.Number[i]
			}
		}
		safe, mines := Deduce(b.W, b.H, v, minesLeft)
		if len(safe) == 0 && len(mines) == 0 {
			return false
		}
		for _, m := range mines {
			v[m] = KnownMine
			minesLeft--
		}
		for _, s := range safe {
			f.Reveal(s)
		}
	}
	return true
}

// GenerateNoGuess returns a board that Solvable accepts from safe. If no
// such board turns up within the attempt limit it returns an ordinary board
// and false. The limit is a count, not a deadline, so results stay
// reproducible regardless of how fast the machine is.
func GenerateNoGuess(cfg Config, seed uint64, safe int) (*Board, bool) {
	rng := newRNG(seed)
	var b *Board
	for range maxNoGuessAttempts {
		b = generate(cfg, rng, []int{safe})
		if Solvable(b, safe) {
			return b, true
		}
	}
	return b, false
}

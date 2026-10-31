// Package engine holds the pure Minesweeper logic: boards, play state and
// the logic solver. It knows nothing about graphics, networking or modes.
package engine

import "math/rand/v2"

// Config describes a board to generate.
type Config struct {
	W, H, Mines int
}

// The classic difficulties.
var (
	Beginner     = Config{W: 9, H: 9, Mines: 10}
	Intermediate = Config{W: 16, H: 16, Mines: 40}
	Expert       = Config{W: 30, H: 16, Mines: 99}
)

// Board is an immutable mine layout. Tiles are addressed by index y*W+x.
type Board struct {
	W, H, Mines int
	Mine        []bool
	Number      []int8
}

// NewBoard builds a board from an explicit list of mine tiles.
func NewBoard(w, h int, mines []int) *Board {
	b := &Board{W: w, H: h, Mines: len(mines), Mine: make([]bool, w*h), Number: make([]int8, w*h)}
	for _, m := range mines {
		b.Mine[m] = true
	}
	var buf [8]int
	for _, m := range mines {
		for _, n := range b.Neighbors(m, buf[:0]) {
			b.Number[n]++
		}
	}
	return b
}

// Size returns the number of tiles.
func (b *Board) Size() int { return b.W * b.H }

// IsMine reports whether tile i holds a mine.
func (b *Board) IsMine(i int) bool { return b.Mine[i] }

// Adj returns how many mines touch tile i.
func (b *Board) Adj(i int) int { return int(b.Number[i]) }

// Neighbors appends the tiles touching i to buf.
func (b *Board) Neighbors(i int, buf []int) []int {
	return neighbors(b.W, b.H, i, buf)
}

func neighbors(w, h, i int, buf []int) []int {
	x, y := i%w, i/w
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if (dx != 0 || dy != 0) && nx >= 0 && ny >= 0 && nx < w && ny < h {
				buf = append(buf, ny*w+nx)
			}
		}
	}
	return buf
}

// Generate places cfg.Mines mines at random, keeping each safe tile and its
// neighbours clear so that clicking it opens a region. Boards are identical
// across platforms for a given seed, which the daily challenge relies on.
func Generate(cfg Config, seed uint64, safe ...int) *Board {
	return generate(cfg, newRNG(seed), safe)
}

func newRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, 0x6d696e6570617274))
}

func generate(cfg Config, rng *rand.Rand, safe []int) *Board {
	reserved := make([]bool, cfg.W*cfg.H)
	var buf [8]int
	for _, s := range safe {
		reserved[s] = true
		for _, n := range neighbors(cfg.W, cfg.H, s, buf[:0]) {
			reserved[n] = true
		}
	}
	free := make([]int, 0, cfg.W*cfg.H)
	for i := range reserved {
		if !reserved[i] {
			free = append(free, i)
		}
	}
	mines := min(cfg.Mines, len(free))
	// A hand-written shuffle: rand.Shuffle's algorithm is not a stable API.
	for i := 0; i < mines; i++ {
		j := i + int(rng.Uint64()%uint64(len(free)-i))
		free[i], free[j] = free[j], free[i]
	}
	return NewBoard(cfg.W, cfg.H, free[:mines])
}

// ThreeBV returns the Bechtel Board Benchmark Value: the fewest left clicks
// that can clear the board. It lets runs on different boards be compared.
func (b *Board) ThreeBV() int {
	seen := make([]bool, b.Size())
	count := 0
	var buf [8]int
	var stack []int
	for i := range seen {
		if seen[i] || b.Mine[i] || b.Number[i] != 0 {
			continue
		}
		count++
		seen[i] = true
		stack = append(stack[:0], i)
		for len(stack) > 0 {
			t := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, n := range b.Neighbors(t, buf[:0]) {
				if !seen[n] {
					seen[n] = true
					if b.Number[n] == 0 {
						stack = append(stack, n)
					}
				}
			}
		}
	}
	for i := range seen {
		if !seen[i] && !b.Mine[i] {
			count++
		}
	}
	return count
}

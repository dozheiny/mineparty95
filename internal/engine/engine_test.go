package engine

import (
	"reflect"
	"slices"
	"testing"
)

// idx converts x,y to a tile index for a board of width w.
func idx(w, x, y int) int { return y*w + x }

func TestAdjacencyCounts(t *testing.T) {
	// M 1 .
	// 1 1 .
	// . . .
	b := NewBoard(3, 3, []int{0})
	want := []int{0, 1, 0, 1, 1, 0, 0, 0, 0}
	for i, w := range want {
		if i == 0 {
			continue
		}
		if got := b.Adj(i); got != w {
			t.Errorf("Adj(%d) = %d, want %d", i, got, w)
		}
	}
	if !b.IsMine(0) || b.IsMine(1) {
		t.Fatal("IsMine disagrees with the mine list")
	}
}

func TestGeneratePlacesExactMinesOutsideSafeZone(t *testing.T) {
	safe := idx(Expert.W, 10, 8)
	b := Generate(Expert, 42, safe)
	n := 0
	for i := 0; i < b.Size(); i++ {
		if b.IsMine(i) {
			n++
		}
	}
	if n != Expert.Mines {
		t.Fatalf("placed %d mines, want %d", n, Expert.Mines)
	}
	if b.IsMine(safe) || b.Adj(safe) != 0 {
		t.Fatalf("safe tile must open a region: mine=%v adj=%d", b.IsMine(safe), b.Adj(safe))
	}
}

func TestGenerateIsDeterministicForSeed(t *testing.T) {
	a := Generate(Intermediate, 7, 0)
	b := Generate(Intermediate, 7, 0)
	c := Generate(Intermediate, 8, 0)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed produced different boards")
	}
	if reflect.DeepEqual(a, c) {
		t.Fatal("different seeds produced identical boards")
	}
}

func TestGenerateCapsImpossibleMineCounts(t *testing.T) {
	b := Generate(Config{W: 4, H: 4, Mines: 999}, 1, 0)
	n := 0
	for i := 0; i < b.Size(); i++ {
		if b.IsMine(i) {
			n++
		}
	}
	// A corner safe zone reserves 4 of the 16 tiles.
	if n != 12 || b.Mines != 12 {
		t.Fatalf("mines = %d, Board.Mines = %d, want 12", n, b.Mines)
	}
}

func TestThreeBV(t *testing.T) {
	// One mine in a corner leaves a single opening covering every safe tile.
	if got := NewBoard(5, 5, []int{0}).ThreeBV(); got != 1 {
		t.Fatalf("one-opening board: 3BV = %d, want 1", got)
	}
	// With no openings every safe tile needs its own click.
	if got := NewBoard(3, 1, []int{1}).ThreeBV(); got != 2 {
		t.Fatalf("no-opening strip: 3BV = %d, want 2", got)
	}
}

func TestRevealFloodsZeroRegionIncludingBorderNumbers(t *testing.T) {
	b := NewBoard(5, 5, []int{0})
	f := NewField(b)
	opened, hit := f.Reveal(idx(5, 4, 4))
	if hit {
		t.Fatal("reported a mine on a safe tile")
	}
	if len(opened) != 24 {
		t.Fatalf("opened %d tiles, want 24", len(opened))
	}
	if !f.Cleared() {
		t.Fatal("board with every safe tile open should be cleared")
	}
}

func TestRevealStopsAtNumbers(t *testing.T) {
	b := NewBoard(3, 1, []int{1})
	f := NewField(b)
	opened, _ := f.Reveal(0)
	if !slices.Equal(opened, []int{0}) {
		t.Fatalf("opened = %v, want just tile 0", opened)
	}
}

func TestRevealSkipsFlaggedAndOpenTiles(t *testing.T) {
	b := NewBoard(5, 5, []int{0})
	f := NewField(b)
	f.ToggleFlag(1)
	if opened, _ := f.Reveal(1); len(opened) != 0 {
		t.Fatalf("flagged tile opened: %v", opened)
	}
	opened, _ := f.Reveal(idx(5, 4, 4))
	if slices.Contains(opened, 1) {
		t.Fatal("flood opened a flagged tile")
	}
	if again, _ := f.Reveal(idx(5, 4, 4)); len(again) != 0 {
		t.Fatalf("reopening returned %v", again)
	}
}

func TestRevealMineReportsHitWithoutOpeningSafeTiles(t *testing.T) {
	f := NewField(NewBoard(3, 1, []int{1}))
	opened, hit := f.Reveal(1)
	if !hit || len(opened) != 0 {
		t.Fatalf("opened=%v hit=%v, want no tiles and a hit", opened, hit)
	}
	if f.Cleared() {
		t.Fatal("hitting a mine must not count as clearing")
	}
}

func TestToggleFlagIgnoresOpenTiles(t *testing.T) {
	f := NewField(NewBoard(3, 1, []int{1}))
	f.Reveal(0)
	f.ToggleFlag(0)
	if f.Flagged(0) {
		t.Fatal("open tile accepted a flag")
	}
}

func TestChordOpensNeighboursWhenFlagsMatch(t *testing.T) {
	// Tile 1 sees one mine; flagging it lets a chord open everything else.
	f := NewField(NewBoard(3, 3, []int{0}))
	f.Reveal(1)
	f.ToggleFlag(0)
	opened, mine := f.Chord(1)
	if mine != -1 || len(opened) == 0 {
		t.Fatalf("opened=%v mine=%d", opened, mine)
	}
	if !f.Cleared() {
		t.Fatalf("chord should clear this board, opened %v", opened)
	}
}

func TestChordRefusesWhenFlagCountDiffers(t *testing.T) {
	f := NewField(NewBoard(3, 3, []int{0}))
	f.Reveal(1)
	if opened, mine := f.Chord(1); mine != -1 || len(opened) != 0 {
		t.Fatalf("unflagged chord acted: opened=%v mine=%d", opened, mine)
	}
}

func TestChordOnWrongFlagReportsTheMineItHit(t *testing.T) {
	f := NewField(NewBoard(3, 3, []int{0}))
	f.Reveal(1)
	f.ToggleFlag(2)
	if _, mine := f.Chord(1); mine != 0 {
		t.Fatalf("mine = %d, want the real mine at 0", mine)
	}
}

func TestChordCountsExposedMinesAsFlags(t *testing.T) {
	f := NewField(NewBoard(3, 3, []int{0}))
	f.Reveal(1)
	f.Expose(0)
	if opened, mine := f.Chord(1); mine != -1 || len(opened) == 0 {
		t.Fatalf("opened=%v mine=%d", opened, mine)
	}
}

func TestDeduceProvesTheOnlyHiddenNeighbourIsAMine(t *testing.T) {
	safe, mines := Deduce(2, 1, []int8{1, Hidden}, -1)
	if len(safe) != 0 || !slices.Equal(mines, []int{1}) {
		t.Fatalf("safe=%v mines=%v, want mine at 1", safe, mines)
	}
}

func TestDeduceTreatsSatisfiedNumbersAsSafe(t *testing.T) {
	safe, mines := Deduce(3, 1, []int8{KnownMine, 1, Hidden}, -1)
	if !slices.Equal(safe, []int{2}) || len(mines) != 0 {
		t.Fatalf("safe=%v mines=%v, want safe tile 2", safe, mines)
	}
}

func TestDeduceSolvesOneTwoOnePattern(t *testing.T) {
	// No single number decides this; only comparing the 1 and 2 does.
	v := []int8{
		Hidden, Hidden, Hidden,
		1, 2, 1,
	}
	safe, mines := Deduce(3, 2, v, -1)
	if !slices.Equal(safe, []int{1}) || !slices.Equal(mines, []int{0, 2}) {
		t.Fatalf("safe=%v mines=%v, want safe [1] mines [0 2]", safe, mines)
	}
}

func TestDeduceUsesGlobalMineCount(t *testing.T) {
	// The hidden tile touches no number, so only the count can decide it.
	v := []int8{KnownMine, Hidden}
	if safe, _ := Deduce(2, 1, v, 0); !slices.Equal(safe, []int{1}) {
		t.Fatalf("zero left: safe=%v", safe)
	}
	if _, mines := Deduce(2, 1, v, 1); !slices.Equal(mines, []int{1}) {
		t.Fatalf("one left: mines=%v", mines)
	}
}

func TestSolvableRejectsForcedFiftyFifty(t *testing.T) {
	// . . M .
	// . . . M   The two right columns cannot be told apart from the left.
	if Solvable(NewBoard(4, 2, []int{2, 7}), 0) {
		t.Fatal("a board with a forced guess was reported solvable")
	}
}

func TestSolvableAcceptsLogicBoard(t *testing.T) {
	if !Solvable(NewBoard(5, 5, []int{0}), idx(5, 4, 4)) {
		t.Fatal("trivial single-opening board should be solvable")
	}
}

func TestGenerateNoGuessProducesSolvableBoards(t *testing.T) {
	for name, cfg := range map[string]Config{"beginner": Beginner, "intermediate": Intermediate, "expert": Expert} {
		t.Run(name, func(t *testing.T) {
			for seed := uint64(1); seed <= 5; seed++ {
				start := idx(cfg.W, cfg.W/2, cfg.H/2)
				b, ok := GenerateNoGuess(cfg, seed, start)
				if !ok {
					t.Fatalf("seed %d: generator gave up", seed)
				}
				if !Solvable(b, start) {
					t.Fatalf("seed %d: returned an unsolvable board", seed)
				}
			}
		})
	}
}

func TestGenerateNoGuessIsDeterministic(t *testing.T) {
	a, _ := GenerateNoGuess(Expert, 99, 100)
	b, _ := GenerateNoGuess(Expert, 99, 100)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("no-guess generation must be reproducible for the daily challenge")
	}
}

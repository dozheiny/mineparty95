package engine

// Mark is what a player has put on a hidden tile.
type Mark byte

const (
	NoMark Mark = iota
	FlagMark
	QuestionMark // unsure: counts as no mine, but chords and floods leave it shut
)

// Next is the mark a right click turns m into: a flag, then a question mark,
// then nothing again.
func (m Mark) Next() Mark { return (m + 1) % 3 }

// Field is the play state of one board: which tiles are open or marked.
// Several players can share a Field, or each can have their own.
type Field struct {
	B       *Board
	open    []bool
	mark    []Mark
	exposed []bool
	left    int
}

// NewField returns a fully hidden field.
func NewField(b *Board) *Field {
	return &Field{
		B:       b,
		open:    make([]bool, b.Size()),
		mark:    make([]Mark, b.Size()),
		exposed: make([]bool, b.Size()),
		left:    b.Size() - b.Mines,
	}
}

// Open reports whether tile i has been opened.
func (f *Field) Open(i int) bool { return f.open[i] }

// Flagged reports whether tile i carries a flag.
func (f *Field) Flagged(i int) bool { return f.mark[i] == FlagMark }

// Mark returns the mark on tile i.
func (f *Field) Mark(i int) Mark { return f.mark[i] }

// Exposed reports whether tile i is a mine that has been uncovered.
func (f *Field) Exposed(i int) bool { return f.exposed[i] }

// Cleared reports whether every safe tile is open.
func (f *Field) Cleared() bool { return f.left == 0 }

// Left returns how many safe tiles remain closed.
func (f *Field) Left() int { return f.left }

// Flags returns how many flags are on the board.
func (f *Field) Flags() int {
	n := 0
	for _, m := range f.mark {
		if m == FlagMark {
			n++
		}
	}
	return n
}

// Hidden reports whether tile i has neither been opened nor exposed.
func (f *Field) Hidden(i int) bool { return !f.open[i] && !f.exposed[i] }

// CycleMark moves a hidden tile on to its next mark.
func (f *Field) CycleMark(i int) {
	if f.Hidden(i) {
		f.mark[i] = f.mark[i].Next()
	}
}

// Expose uncovers a mine so that it stays visible and counts as flagged.
func (f *Field) Expose(i int) {
	f.exposed[i] = true
	f.mark[i] = NoMark
}

// Reveal opens tile i, flooding outward from empty tiles. A flag stops it,
// and the flood leaves marked tiles closed. It reports the tiles opened and
// whether tile i was a mine.
func (f *Field) Reveal(i int) (opened []int, hit bool) {
	if !f.Hidden(i) || f.mark[i] == FlagMark {
		return nil, false
	}
	if f.B.Mine[i] {
		return nil, true
	}
	return f.flood(i, nil), false
}

func (f *Field) flood(start int, opened []int) []int {
	stack := []int{start}
	f.open[start] = true
	f.left--
	opened = append(opened, start)
	var buf [8]int
	for len(stack) > 0 {
		t := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if f.B.Number[t] != 0 {
			continue
		}
		for _, n := range f.B.Neighbors(t, buf[:0]) {
			if f.Hidden(n) && f.mark[n] == NoMark {
				f.open[n] = true
				f.left--
				opened = append(opened, n)
				stack = append(stack, n)
			}
		}
	}
	return opened
}

// Chord opens the unmarked neighbours of an open number once enough of them
// are flagged. Question marks don't count as flags and stay closed. A
// misplaced flag makes it hit a mine, whose tile it reports; mine is -1
// otherwise.
func (f *Field) Chord(i int) (opened []int, mine int) {
	return f.ChordWith(i, f.mark)
}

// ChordWith is Chord for callers that track marks themselves, such as modes
// where each player has private marks on a shared field.
func (f *Field) ChordWith(i int, marks []Mark) (opened []int, mine int) {
	mine = -1
	if !f.open[i] || f.B.Number[i] == 0 {
		return nil, mine
	}
	var buf [8]int
	near := f.B.Neighbors(i, buf[:0])
	marked := 0
	for _, n := range near {
		if marks[n] == FlagMark || f.exposed[n] {
			marked++
		}
	}
	if marked != int(f.B.Number[i]) {
		return nil, mine
	}
	for _, n := range near {
		if !f.Hidden(n) || marks[n] != NoMark {
			continue
		}
		if f.B.Mine[n] {
			if mine < 0 {
				mine = n
			}
			continue
		}
		opened = f.flood(n, opened)
	}
	return opened, mine
}

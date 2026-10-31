// Package match implements the rules of every game mode as a pure state
// machine. Actions go in and per-player views come out; the clock is passed
// in, so a match never touches the wall clock, the network or the screen.
package match

import (
	"time"

	"mineparty/internal/engine"
)

// Mode selects the rules.
type Mode int

// The game modes.
const (
	Classic Mode = iota
	Race
	Flags
	Coop
	Territory
	Blitz
	Survival
	LastStand
)

// ModeNames holds a display name for each Mode.
var ModeNames = [...]string{"Classic", "Race", "Mine Flags", "Co-op", "Territory", "Blitz", "Survival", "Last Stand"}

func (m Mode) String() string { return ModeNames[m] }

// Solo reports whether a mode is played alone.
func (m Mode) Solo() bool { return m == Classic || m == Blitz || m == Survival }

// TurnBased reports whether players move one at a time against a turn clock.
func (m Mode) TurnBased() bool { return m == Flags || m == LastStand }

// FixedBoard reports whether a mode always plays on its own board.
func (m Mode) FixedBoard() bool {
	return m == Flags || m == Territory || m == Blitz || m == Survival || m == LastStand
}

// PlayerLimits returns the fewest and most players a mode allows.
func (m Mode) PlayerLimits() (lo, hi int) {
	switch {
	case m.Solo():
		return 1, 1
	case m == Flags:
		return 2, 2
	}
	return 2, 4
}

// Phase is the stage a match has reached.
type Phase int

// The match phases.
const (
	Countdown Phase = iota
	Playing
	Over
)

// ActionKind is what a player is trying to do.
type ActionKind int

// The player actions.
const (
	Reveal ActionKind = iota
	Flag
	Chord
	Bomb
	Cursor
	Ping
)

// Action is one player's intent. The match decides whether it is legal.
type Action struct {
	Player int
	Kind   ActionKind
	Tile   int
}

// Rule timings and sizes.
const (
	CountdownTime    = 3 * time.Second
	RaceStun         = 5 * time.Second
	TurnTime         = 20 * time.Second
	TerritoryTime    = 3 * time.Minute
	TerritoryStun    = 4 * time.Second
	TerritoryMinLoss = 5
	PingTime         = 2 * time.Second
	BlitzTime        = 2 * time.Minute
	BlitzBonus       = 20 * time.Second
	BlitzMinBonus    = 2 * time.Second
	BlitzPenalty     = 15 * time.Second
	BlitzStun        = time.Second
	SurvivalLives    = 3
	LastStandTurn    = 10 * time.Second
	bombRadius       = 2
)

// Fixed boards for the modes that need a particular shape.
var (
	FlagsBoard     = engine.Config{W: 16, H: 16, Mines: 51}
	TerritoryBoard = engine.Config{W: 38, H: 24, Mines: 150}
	BlitzBoard     = engine.Intermediate
	SurvivalBoard  = engine.Expert
	LastStandBoard = engine.Config{W: 16, H: 16, Mines: 51}
)

// Config describes a match to create.
type Config struct {
	Mode       Mode
	Board      engine.Config
	Seed       uint64
	Round      int // identifies this match within a session
	Players    []string
	AllowGuess bool // skip no-guess generation

	// Start lists tiles opened before play begins: one shared opening, or
	// one per player in Territory. With neither Start nor Layout set, the
	// mode picks its own.
	Start []int

	// Layout replaces generation with a fixed board.
	Layout *engine.Board
}

type player struct {
	name      string
	left      bool
	field     *engine.Field // own field in Race and Classic, otherwise shared
	flags     []bool        // private flags in Territory and Last Stand
	score     int
	hits      int
	clicks    int
	flagged   bool
	stunUntil time.Duration
	cursor    int
	bombUsed  bool
	claims    []int // owned tiles, oldest first
	lives     int   // in Survival
	out       bool  // knocked out of Last Stand
}

type ping struct {
	tile, player int
	at           time.Duration
}

// Match is one game in progress.
type Match struct {
	cfg     Config
	board   *engine.Board
	players []*player
	owners  []byte // player+1 for each tile, on boards with claims
	pings   []ping

	phase    Phase
	startAt  time.Duration // when the clock started
	stopAt   time.Duration // when the match ended
	ticking  bool
	boom     int // tile of the fatal mine, or -1
	winner   int
	won      bool
	turn     int
	turnEnds time.Duration
	deadline time.Duration // when Blitz runs out of time
	boards   int           // boards cleared in Blitz
}

// New creates a match. It starts counting down at now, except for Classic
// which waits for the first click.
func New(cfg Config, now time.Duration) *Match {
	m := &Match{cfg: cfg, boom: -1, winner: -1}
	for _, name := range cfg.Players {
		pl := &player{name: name, cursor: -1}
		if cfg.Mode == Survival {
			pl.lives = SurvivalLives
		}
		m.players = append(m.players, pl)
	}
	if cfg.Layout == nil {
		switch cfg.Mode {
		case Flags:
			m.cfg.Board = FlagsBoard
			m.cfg.AllowGuess = true
		case Territory:
			m.cfg.Board = TerritoryBoard
			m.cfg.AllowGuess = true
			if m.cfg.Start == nil {
				m.cfg.Start = territoryStarts(TerritoryBoard, len(cfg.Players))
			}
		case Blitz:
			m.cfg.Board = BlitzBoard
		case Survival:
			m.cfg.Board = SurvivalBoard
		case LastStand:
			m.cfg.Board = LastStandBoard
			m.cfg.AllowGuess = true
		}
		switch cfg.Mode {
		case Race, Coop, Blitz, Survival, LastStand:
			if m.cfg.Start == nil {
				m.cfg.Start = []int{openingTile(m.cfg.Board, cfg.Seed)}
			}
		}
	}
	if cfg.Layout != nil {
		m.setBoard(cfg.Layout)
	} else if cfg.Mode != Classic || m.cfg.Start != nil {
		m.setBoard(m.generate(m.cfg.Start))
	}

	if cfg.Mode == Classic {
		m.phase = Playing
		return m
	}
	m.startAt = now + CountdownTime
	m.turn = int(cfg.Seed % uint64(max(len(m.players), 1)))
	return m
}

func (m *Match) generate(safe []int) *engine.Board {
	if !m.cfg.AllowGuess && len(safe) == 1 {
		b, _ := engine.GenerateNoGuess(m.cfg.Board, m.cfg.Seed, safe[0])
		return b
	}
	return engine.Generate(m.cfg.Board, m.cfg.Seed, safe...)
}

func (m *Match) setBoard(b *engine.Board) {
	m.board = b
	shared := engine.NewField(b)
	for _, p := range m.players {
		switch m.cfg.Mode {
		case Classic, Race:
			p.field = engine.NewField(b)
		case Territory:
			p.field = shared
			p.flags = make([]bool, b.Size())
		case LastStand:
			p.field = shared
			p.flags = make([]bool, b.Size())
		default:
			p.field = shared
		}
	}
	if m.claiming() {
		m.owners = make([]byte, b.Size())
	}
	for i, tile := range m.cfg.Start {
		switch m.cfg.Mode {
		case Territory:
			if i < len(m.players) {
				m.claim(i, tile)
			}
		case Race:
			for _, p := range m.players {
				p.field.Reveal(tile)
			}
		default:
			for _, p := range m.players[:1] {
				p.field.Reveal(tile)
			}
		}
	}
}

func (m *Match) claiming() bool { return m.cfg.Mode == Flags || m.cfg.Mode == Territory }

// openingTile picks a pre-opened tile near the middle of the board.
func openingTile(b engine.Config, seed uint64) int {
	x := b.W/4 + int(seed%uint64(b.W/2))
	y := b.H/4 + int((seed>>16)%uint64(b.H/2))
	return y*b.W + x
}

func territoryStarts(b engine.Config, players int) []int {
	const in = 4
	corners := []int{
		in*b.W + in,
		(b.H-1-in)*b.W + b.W - 1 - in,
		in*b.W + b.W - 1 - in,
		(b.H-1-in)*b.W + in,
	}
	return corners[:min(players, len(corners))]
}

// Phase returns the current stage of the match.
func (m *Match) Phase() Phase { return m.phase }

// Tick advances anything driven by time.
func (m *Match) Tick(now time.Duration) {
	if m.phase == Countdown && now >= m.startAt {
		m.phase = Playing
		m.ticking = true
		m.turnEnds = now + TurnTime
		switch m.cfg.Mode {
		case LastStand:
			m.turnEnds = now + LastStandTurn
		case Blitz:
			m.deadline = now + BlitzTime
		}
	}
	if m.phase != Playing {
		return
	}
	switch m.cfg.Mode {
	case Flags:
		if now >= m.turnEnds {
			m.passTurn(now)
		}
	case Territory:
		if now >= m.startAt+TerritoryTime {
			m.finishByScore(now)
		}
	case Blitz:
		if now >= m.deadline {
			m.finish(now, -1, m.boards > 0)
		}
	case LastStand:
		if now >= m.turnEnds {
			m.eliminate(now, m.turn)
		}
	}
}

// Apply performs an action if it is legal; anything else is ignored.
func (m *Match) Apply(now time.Duration, a Action) {
	m.Tick(now)
	if a.Player < 0 || a.Player >= len(m.players) {
		return
	}
	p := m.players[a.Player]
	if p.left || p.out || m.phase != Playing {
		return
	}
	if a.Kind != Cursor && (a.Tile < 0 || (m.board != nil && a.Tile >= m.board.Size())) {
		return
	}
	switch a.Kind {
	case Cursor:
		if a.Tile >= -1 && (m.board == nil || a.Tile < m.board.Size()) {
			p.cursor = a.Tile
		}
		return
	case Ping:
		m.pings = append(m.pings, ping{tile: a.Tile, player: a.Player, at: now})
		return
	}
	if m.board == nil {
		// Classic builds its board around the first click.
		if a.Kind != Reveal {
			return
		}
		m.setBoard(m.generate([]int{a.Tile}))
	}
	if now < p.stunUntil {
		return
	}
	switch m.cfg.Mode {
	case Classic, Coop:
		m.applyCommon(now, p, a)
	case Race:
		m.applyRace(now, a.Player, p, a)
	case Flags:
		m.applyFlags(now, a.Player, a)
	case Territory:
		m.applyTerritory(now, a.Player, p, a)
	case Blitz:
		m.applyBlitz(now, p, a)
	case Survival:
		m.applySurvival(now, p, a)
	case LastStand:
		m.applyLastStand(now, a.Player, p, a)
	}
}

func (m *Match) startClock(now time.Duration) {
	if !m.ticking {
		m.ticking = true
		m.startAt = now
	}
}

// Leave removes a player who disconnected.
func (m *Match) Leave(now time.Duration, player int) {
	if player < 0 || player >= len(m.players) || m.players[player].left {
		return
	}
	m.players[player].left = true
	if m.phase == Over {
		return
	}
	if m.cfg.Mode == LastStand {
		m.eliminate(now, player)
		return
	}
	present := -1
	count := 0
	for i, p := range m.players {
		if !p.left {
			present = i
			count++
		}
	}
	switch {
	case count == 0:
		m.finish(now, -1, false)
	case count == 1 && m.cfg.Mode != Coop && !m.cfg.Mode.Solo():
		m.finish(now, present, true)
	case m.cfg.Mode == Flags && m.turn == player:
		m.passTurn(now)
	}
}

func (m *Match) finish(now time.Duration, winner int, won bool) {
	m.phase = Over
	m.stopAt = now
	m.winner = winner
	m.won = won
}

// applyCommon is the shared-fate logic of Classic and Co-op: one mine ends
// the run, and clearing the board wins it.
func (m *Match) applyCommon(now time.Duration, p *player, a Action) {
	f := p.field
	switch a.Kind {
	case Flag:
		f.ToggleFlag(a.Tile)
		p.flagged = true
	case Reveal, Chord:
		m.startClock(now)
		p.clicks++
		opened, mine := perform(f, a)
		p.score += len(opened)
		if mine >= 0 {
			m.boom = mine
			m.finish(now, -1, false)
		} else if f.Cleared() {
			m.finish(now, -1, true)
		}
	}
}

// perform runs a reveal or chord and reports any mine it struck.
func perform(f *engine.Field, a Action) (opened []int, mine int) {
	if a.Kind == Chord {
		return f.Chord(a.Tile)
	}
	opened, hit := f.Reveal(a.Tile)
	if hit {
		return nil, a.Tile
	}
	return opened, -1
}

func (m *Match) applyRace(now time.Duration, index int, p *player, a Action) {
	f := p.field
	switch a.Kind {
	case Flag:
		f.ToggleFlag(a.Tile)
		p.flagged = true
	case Reveal, Chord:
		p.clicks++
		opened, mine := perform(f, a)
		p.score += len(opened)
		if mine >= 0 {
			f.Expose(mine)
			p.hits++
			p.stunUntil = now + RaceStun
		}
		if f.Cleared() {
			m.finish(now, index, true)
		}
	}
}

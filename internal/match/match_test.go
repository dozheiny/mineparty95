package match

import (
	"slices"
	"testing"
	"time"

	"mineparty/internal/engine"
)

const sec = time.Second

// strip is a 6x1 board: tile 1 and tile 4 are mines, the rest are numbers.
func strip() *engine.Board { return engine.NewBoard(6, 1, []int{1, 4}) }

// open5 is a 5x5 board with a single corner mine, cleared by one click.
func open5() *engine.Board { return engine.NewBoard(5, 5, []int{0}) }

func started(t *testing.T, cfg Config) *Match {
	t.Helper()
	m := New(cfg, 0)
	m.Tick(CountdownTime)
	if m.Phase() != Playing {
		t.Fatalf("phase = %v after the countdown, want Playing", m.Phase())
	}
	return m
}

func act(m *Match, now time.Duration, player int, kind ActionKind, tile int) {
	m.Apply(now, Action{Player: player, Kind: kind, Tile: tile})
}

func TestClassicFirstClickIsAlwaysSafe(t *testing.T) {
	for seed := uint64(0); seed < 50; seed++ {
		m := New(Config{Mode: Classic, Board: engine.Beginner, Seed: seed, Players: []string{"a"}}, 0)
		act(m, 0, 0, Reveal, 40)
		if m.Phase() == Over && !m.View(0, 0).Won {
			t.Fatalf("seed %d: first click lost the game", seed)
		}
		if got := m.View(0, 0).Tiles[40]; got != 0 {
			t.Fatalf("seed %d: first click tile = %d, want an opening", seed, got)
		}
	}
}

func TestClassicTimerStartsOnFirstClick(t *testing.T) {
	m := New(Config{Mode: Classic, Board: engine.Expert, Seed: 1, Players: []string{"a"}}, 0)
	if got := m.View(0, 5*sec).ElapsedMs; got != 0 {
		t.Fatalf("elapsed before first click = %d", got)
	}
	act(m, 5*sec, 0, Reveal, 100)
	if got := m.View(0, 8*sec).ElapsedMs; got != 3000 {
		t.Fatalf("elapsed = %d, want 3000", got)
	}
}

func TestClassicLosingShowsMinesAndExplodedTile(t *testing.T) {
	m := New(Config{Mode: Classic, Layout: strip(), Players: []string{"a"}}, 0)
	act(m, 0, 0, Flag, 2)
	act(m, 0, 0, Reveal, 1)
	v := m.View(0, 0)
	if m.Phase() != Over || v.Won {
		t.Fatalf("phase=%v won=%v, want a loss", m.Phase(), v.Won)
	}
	if v.Tiles[1] != TileExploded || v.Tiles[4] != TileMine || v.Tiles[2] != TileWrongFlag {
		t.Fatalf("tiles = %v", v.Tiles)
	}
}

func TestQuestionMarksShowUntilTheEnd(t *testing.T) {
	m := New(Config{Mode: Classic, Layout: strip(), Players: []string{"a"}}, 0)
	for _, tile := range []int{4, 4, 5, 5} {
		act(m, 0, 0, Flag, tile)
	}
	if v := m.View(0, 0); v.Tiles[4] != TileQuestion || v.Tiles[5] != TileQuestion {
		t.Fatalf("tiles = %v, want question marks on 4 and 5", v.Tiles)
	}
	act(m, 0, 0, Reveal, 1)
	if v := m.View(0, 0); v.Tiles[4] != TileMine || v.Tiles[5] != TileQuestion {
		t.Fatalf("tiles = %v after losing", v.Tiles)
	}
}

func TestClassicClearingWinsAndCountsClicks(t *testing.T) {
	m := New(Config{Mode: Classic, Layout: open5(), Players: []string{"a"}}, 0)
	act(m, 2*sec, 0, Reveal, 24)
	v := m.View(0, 2*sec)
	if m.Phase() != Over || !v.Won || v.Players[0].Clicks != 1 || v.BV3 != 1 {
		t.Fatalf("phase=%v won=%v clicks=%d bv=%d", m.Phase(), v.Won, v.Players[0].Clicks, v.BV3)
	}
	if v.Tiles[0] != TileFlag {
		t.Fatalf("a won board should auto-flag its mines, got %d", v.Tiles[0])
	}
}

func TestActionsAreIgnoredDuringCountdown(t *testing.T) {
	m := New(Config{Mode: Race, Layout: open5(), Players: []string{"a", "b"}}, 0)
	act(m, sec, 0, Reveal, 24)
	if m.View(0, sec).Tiles[24] != TileHidden {
		t.Fatal("a tile opened before the countdown ended")
	}
}

func TestOutOfRangeActionsAreRejected(t *testing.T) {
	m := started(t, Config{Mode: Race, Layout: open5(), Players: []string{"a", "b"}})
	for _, a := range []Action{{Player: 0, Kind: Reveal, Tile: -1}, {Player: 0, Kind: Reveal, Tile: 25}, {Player: 9, Kind: Reveal, Tile: 1}, {Player: -1, Kind: Flag, Tile: 1}} {
		m.Apply(CountdownTime, a) // must neither panic nor change anything
	}
	if m.Phase() != Playing {
		t.Fatal("invalid actions changed the game")
	}
}

func TestHiddenMinesNeverLeakIntoAnyView(t *testing.T) {
	for _, mode := range []Mode{Race, Flags, Coop, Territory} {
		m := started(t, Config{Mode: mode, Seed: 5, Players: []string{"a", "b"}, Board: engine.Intermediate})
		// Anything still unopened must read exactly the same whether or
		// not it holds a mine.
		for _, p := range []int{0, 1} {
			v := m.View(p, CountdownTime)
			for i, tile := range v.Tiles {
				if m.board.IsMine(i) && tile != TileHidden {
					t.Fatalf("%v: mine tile %d visible as %d", mode, i, tile)
				}
			}
		}
	}
}

func TestRacePlayersShareALayoutButNotProgress(t *testing.T) {
	m := started(t, Config{Mode: Race, Seed: 3, Board: engine.Intermediate, Players: []string{"a", "b"}})
	a, b := m.View(0, CountdownTime), m.View(1, CountdownTime)
	if !slices.Equal(a.Tiles, b.Tiles) {
		t.Fatal("players must begin on the same opened board")
	}
	opened := a.Players[0].Progress
	if opened == 0 {
		t.Fatal("the race must begin with a guaranteed opening")
	}

	var safe int
	for i, tile := range a.Tiles {
		if tile == TileHidden && !m.board.IsMine(i) {
			safe = i
			break
		}
	}
	act(m, CountdownTime, 0, Reveal, safe)
	a, b = m.View(0, CountdownTime), m.View(1, CountdownTime)
	if a.Tiles[safe] == TileHidden {
		t.Fatal("reveal did not open the tile")
	}
	if b.Tiles[safe] != TileHidden {
		t.Fatal("an opponent's move leaked into the other player's board")
	}
	if b.Players[0].Progress == 0 {
		t.Fatal("rival progress should be visible")
	}
}

func TestRaceMineStunsThenRecovers(t *testing.T) {
	m := started(t, Config{Mode: Race, Layout: strip(), Players: []string{"a", "b"}})
	now := CountdownTime
	act(m, now, 0, Reveal, 1)
	v := m.View(0, now)
	if v.Players[0].StunMs != int(RaceStun.Milliseconds()) || v.Players[0].Hits != 1 {
		t.Fatalf("stun=%d hits=%d", v.Players[0].StunMs, v.Players[0].Hits)
	}
	act(m, now+sec, 0, Reveal, 0)
	if m.View(0, now+sec).Tiles[0] != TileHidden {
		t.Fatal("a stunned player was able to act")
	}
	if m.View(0, now).Tiles[1] != TileMine {
		t.Fatal("a struck mine should stay exposed so it can't be hit twice")
	}
	act(m, now+RaceStun, 0, Reveal, 0)
	if m.View(0, now+RaceStun).Tiles[0] == TileHidden {
		t.Fatal("player should recover once the stun expires")
	}
}

func TestRaceFirstToClearWins(t *testing.T) {
	m := started(t, Config{Mode: Race, Layout: open5(), Players: []string{"a", "b"}})
	act(m, CountdownTime+sec, 1, Reveal, 24)
	v := m.View(0, CountdownTime+sec)
	if m.Phase() != Over || v.Winner != 1 {
		t.Fatalf("phase=%v winner=%d", m.Phase(), v.Winner)
	}
}

func TestRaceLastPlayerStandingWinsWhenOthersLeave(t *testing.T) {
	m := started(t, Config{Mode: Race, Layout: open5(), Players: []string{"a", "b"}})
	m.Leave(CountdownTime, 1)
	if v := m.View(0, CountdownTime); m.Phase() != Over || v.Winner != 0 {
		t.Fatalf("phase=%v winner=%d", m.Phase(), v.Winner)
	}
}

func flagsMatch(t *testing.T, b *engine.Board) *Match {
	t.Helper()
	return started(t, Config{Mode: Flags, Layout: b, Players: []string{"a", "b"}})
}

func TestFlagsClaimingAMineScoresAndKeepsTheTurn(t *testing.T) {
	m := flagsMatch(t, engine.NewBoard(6, 1, []int{1, 3, 5}))
	act(m, CountdownTime, 0, Reveal, 1)
	v := m.View(1, CountdownTime)
	if v.Players[0].Score != 1 || v.Turn != 0 {
		t.Fatalf("score=%d turn=%d", v.Players[0].Score, v.Turn)
	}
	if v.Tiles[1] != TileMine || v.Owners[1] != 1 {
		t.Fatalf("tile=%d owner=%d", v.Tiles[1], v.Owners[1])
	}
}

func TestFlagsSafeRevealPassesTheTurn(t *testing.T) {
	m := flagsMatch(t, engine.NewBoard(6, 1, []int{1, 3, 5}))
	act(m, CountdownTime, 0, Reveal, 0)
	if got := m.View(0, CountdownTime).Turn; got != 1 {
		t.Fatalf("turn = %d, want 1", got)
	}
}

func TestFlagsIgnoresTheWaitingPlayer(t *testing.T) {
	m := flagsMatch(t, engine.NewBoard(6, 1, []int{1, 3, 5}))
	act(m, CountdownTime, 1, Reveal, 0)
	if m.View(1, CountdownTime).Tiles[0] != TileHidden {
		t.Fatal("the waiting player moved")
	}
}

func TestFlagsTurnTimesOut(t *testing.T) {
	m := flagsMatch(t, engine.NewBoard(6, 1, []int{1, 3, 5}))
	m.Tick(CountdownTime + TurnTime)
	if got := m.View(0, CountdownTime+TurnTime).Turn; got != 1 {
		t.Fatalf("turn = %d after a timeout, want 1", got)
	}
}

func TestFlagsMajorityEndsTheMatch(t *testing.T) {
	m := flagsMatch(t, engine.NewBoard(6, 1, []int{1, 3, 5}))
	act(m, CountdownTime, 0, Reveal, 1)
	if m.Phase() == Over {
		t.Fatal("one of three mines is not a majority")
	}
	act(m, CountdownTime, 0, Reveal, 3)
	if v := m.View(0, CountdownTime); m.Phase() != Over || v.Winner != 0 {
		t.Fatalf("phase=%v winner=%d", m.Phase(), v.Winner)
	}
}

func TestFlagsBombRequiresBeingBehindAndWorksOnce(t *testing.T) {
	// 7x7 with mines on the left for the leader and a cluster on the right.
	b := engine.NewBoard(7, 7, []int{0, 3, 4, 5, 10, 11, 12, 17, 18, 19, 48})
	m := flagsMatch(t, b)
	now := CountdownTime

	act(m, now, 0, Bomb, 11) // level on 0-0: refused, and still player 0's turn
	if v := m.View(0, now); v.Players[0].BombUsed || v.Turn != 0 {
		t.Fatal("a player who isn't behind must not be able to bomb")
	}
	act(m, now, 0, Reveal, 0) // claims a mine
	act(m, now, 0, Reveal, 7) // safe tile, turn passes

	act(m, now, 1, Bomb, 11)
	v := m.View(0, now)
	if !v.Players[1].BombUsed || v.Players[1].Score < 6 {
		t.Fatalf("bomb used=%v score=%d", v.Players[1].BombUsed, v.Players[1].Score)
	}
	if v.Turn != 0 {
		t.Fatalf("a bomb should end the turn, turn=%d", v.Turn)
	}
	if v.Owners[11] != 2 {
		t.Fatalf("owner = %d, want player 2's mark", v.Owners[11])
	}
}

func TestCoopBoardIsShared(t *testing.T) {
	m := started(t, Config{Mode: Coop, Layout: strip(), Players: []string{"a", "b"}})
	act(m, CountdownTime, 0, Flag, 1)
	act(m, CountdownTime, 1, Reveal, 0)
	v := m.View(0, CountdownTime)
	if v.Tiles[1] != TileFlag || v.Tiles[0] == TileHidden {
		t.Fatalf("tiles = %v", v.Tiles)
	}
}

func TestCoopMineLosesForEveryone(t *testing.T) {
	m := started(t, Config{Mode: Coop, Layout: strip(), Players: []string{"a", "b"}})
	act(m, CountdownTime, 1, Reveal, 4)
	if v := m.View(0, CountdownTime); m.Phase() != Over || v.Won {
		t.Fatalf("phase=%v won=%v", m.Phase(), v.Won)
	}
}

func TestCoopClearWinsTogether(t *testing.T) {
	m := started(t, Config{Mode: Coop, Layout: open5(), Players: []string{"a", "b"}})
	act(m, CountdownTime, 1, Reveal, 24)
	if v := m.View(0, CountdownTime); m.Phase() != Over || !v.Won {
		t.Fatalf("phase=%v won=%v", m.Phase(), v.Won)
	}
}

// walled is a 20x20 territory match split by a row of mines, so opening the
// top half claims 100 tiles without ending the game.
func walled(t *testing.T) *Match {
	t.Helper()
	var mines []int
	for x := 0; x < 20; x++ {
		mines = append(mines, 20*5+x)
	}
	return started(t, Config{Mode: Territory, Layout: engine.NewBoard(20, 20, mines), Players: []string{"a", "b"}})
}

// territoryMatch is a 10x10 match with two corner mines and no opening.
func territoryMatch(t *testing.T) *Match {
	t.Helper()
	return started(t, Config{Mode: Territory, Layout: engine.NewBoard(10, 10, []int{0, 9}), Players: []string{"a", "b"}})
}

func TestTerritoryRevealClaimsTiles(t *testing.T) {
	m := territoryMatch(t)
	act(m, CountdownTime, 0, Reveal, 55)
	v := m.View(1, CountdownTime)
	if v.Players[0].Score != 98 || v.Owners[55] != 1 {
		t.Fatalf("score=%d owner=%d", v.Players[0].Score, v.Owners[55])
	}
}

func TestTerritoryFlagsArePrivate(t *testing.T) {
	m := territoryMatch(t)
	act(m, CountdownTime, 0, Flag, 0)
	if m.View(0, CountdownTime).Tiles[0] != TileFlag {
		t.Fatal("flag missing for its owner")
	}
	if m.View(1, CountdownTime).Tiles[0] != TileHidden {
		t.Fatal("private flag leaked to the rival")
	}
}

func TestTerritoryMineStunsAndCostsTiles(t *testing.T) {
	m := walled(t)
	act(m, CountdownTime, 0, Reveal, 0)
	before := m.View(0, CountdownTime).Players[0].Score
	if before == 0 {
		t.Fatal("reveal claimed nothing")
	}
	act(m, CountdownTime, 0, Reveal, 105)
	v := m.View(0, CountdownTime)
	if v.Players[0].StunMs != int(TerritoryStun.Milliseconds()) {
		t.Fatalf("stun = %d", v.Players[0].StunMs)
	}
	lost := before - v.Players[0].Score
	if want := max(before/10, TerritoryMinLoss); lost != want {
		t.Fatalf("lost %d tiles, want %d", lost, want)
	}
}

func TestTerritoryTimeLimitCrownsHighestScore(t *testing.T) {
	m := walled(t)
	act(m, CountdownTime, 1, Reveal, 0)
	if m.Phase() == Over {
		t.Fatal("half a board should not end the match")
	}
	end := CountdownTime + TerritoryTime
	m.Tick(end)
	if v := m.View(0, end); m.Phase() != Over || v.Winner != 1 {
		t.Fatalf("phase=%v winner=%d", m.Phase(), v.Winner)
	}
}

func TestTerritoryDrawHasNoWinner(t *testing.T) {
	m := walled(t)
	end := CountdownTime + TerritoryTime
	m.Tick(end)
	if v := m.View(0, end); v.Winner != -1 {
		t.Fatalf("winner = %d, want a draw", v.Winner)
	}
}

func TestTerritoryStartsOpenAtSeparateCorners(t *testing.T) {
	m := started(t, Config{Mode: Territory, Seed: 9, Players: []string{"a", "b", "c", "d"}})
	v := m.View(0, CountdownTime)
	for p := range 4 {
		if v.Players[p].Score == 0 {
			t.Fatalf("player %d started with no territory", p)
		}
	}
}

func TestPingsAreVisibleToTeammatesThenFade(t *testing.T) {
	m := started(t, Config{Mode: Coop, Layout: strip(), Players: []string{"a", "b"}})
	act(m, CountdownTime, 0, Ping, 3)
	if ps := m.View(1, CountdownTime).Pings; len(ps) != 1 || ps[0].Tile != 3 || ps[0].Player != 0 {
		t.Fatalf("pings = %+v", ps)
	}
	if ps := m.View(1, CountdownTime+PingTime).Pings; len(ps) != 0 {
		t.Fatalf("pings should fade, got %+v", ps)
	}
}

func TestCursorsAreShared(t *testing.T) {
	m := started(t, Config{Mode: Coop, Layout: strip(), Players: []string{"a", "b"}})
	act(m, CountdownTime, 0, Cursor, 3)
	if got := m.View(1, CountdownTime).Players[0].Cursor; got != 3 {
		t.Fatalf("cursor = %d", got)
	}
}

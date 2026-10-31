package match

import (
	"testing"
	"time"

	"mineparty/internal/engine"
)

func TestSoloModesAreForOnePlayer(t *testing.T) {
	for _, mode := range []Mode{Classic, Blitz, Survival} {
		if lo, hi := mode.PlayerLimits(); !mode.Solo() || lo != 1 || hi != 1 {
			t.Errorf("%v: solo=%v limits=%d-%d", mode, mode.Solo(), lo, hi)
		}
	}
	for _, mode := range []Mode{Race, Flags, Coop, Territory, LastStand} {
		if lo, _ := mode.PlayerLimits(); mode.Solo() || lo != 2 {
			t.Errorf("%v: solo=%v lowest=%d", mode, mode.Solo(), lo)
		}
	}
}

func TestEveryModeHasAName(t *testing.T) {
	if len(ModeNames) != int(LastStand)+1 {
		t.Fatalf("%d names for %d modes", len(ModeNames), int(LastStand)+1)
	}
}

func survival(t *testing.T, b *engine.Board) *Match {
	return started(t, Config{Mode: Survival, Layout: b, Players: []string{"a"}})
}

func TestSurvivalBuildsAnExpertBoardWithAnOpening(t *testing.T) {
	m := New(Config{Mode: Survival, Seed: 7, Players: []string{"a"}}, 0)
	v := m.View(0, 0)
	if v.W != engine.Expert.W || v.H != engine.Expert.H || v.Mines != engine.Expert.Mines {
		t.Fatalf("board %dx%d with %d mines", v.W, v.H, v.Mines)
	}
	if v.Phase != Countdown {
		t.Fatalf("phase = %v, want a countdown", v.Phase)
	}
	if v.Players[0].Lives != SurvivalLives {
		t.Fatalf("lives = %d", v.Players[0].Lives)
	}
	opened := 0
	for _, tile := range v.Tiles {
		if tile <= 8 {
			opened++
		}
	}
	if opened == 0 {
		t.Fatal("the run should start with an opening")
	}
}

func TestSurvivalMineCostsALifeAndTheRunContinues(t *testing.T) {
	m := survival(t, strip())
	act(m, 4*sec, 0, Reveal, 1)
	v := m.View(0, 4*sec)
	if v.Phase != Playing || v.Players[0].Lives != SurvivalLives-1 {
		t.Fatalf("phase=%v lives=%d", v.Phase, v.Players[0].Lives)
	}
	if v.Tiles[1] != TileMine {
		t.Fatalf("the mine should stay exposed, tile = %d", v.Tiles[1])
	}
}

func TestSurvivalLastLifeEndsTheRunOnThatMine(t *testing.T) {
	m := survival(t, engine.NewBoard(6, 1, []int{1, 3, 5}))
	for _, tile := range []int{1, 3, 5} {
		act(m, 4*sec, 0, Reveal, tile)
	}
	v := m.View(0, 4*sec)
	if v.Phase != Over || v.Won || v.Players[0].Lives != 0 {
		t.Fatalf("phase=%v won=%v lives=%d", v.Phase, v.Won, v.Players[0].Lives)
	}
	if v.Tiles[5] != TileExploded {
		t.Fatalf("fatal tile = %d, want exploded", v.Tiles[5])
	}
}

func TestSurvivalClearingWinsDespiteLostLives(t *testing.T) {
	m := survival(t, engine.NewBoard(5, 1, []int{0, 4}))
	act(m, 4*sec, 0, Reveal, 0)
	act(m, 4*sec, 0, Reveal, 2)
	v := m.View(0, 4*sec)
	if v.Phase != Over || !v.Won {
		t.Fatalf("phase=%v won=%v", v.Phase, v.Won)
	}
}

func blitz(t *testing.T, b *engine.Board) *Match {
	return started(t, Config{Mode: Blitz, Layout: b, Players: []string{"a"}})
}

func TestBlitzClearingDealsANewBoardAndAddsTime(t *testing.T) {
	m := blitz(t, open5())
	now := CountdownTime + 10*sec
	act(m, now, 0, Reveal, 24)
	v := m.View(0, now)
	if v.Phase != Playing || v.Boards != 1 {
		t.Fatalf("phase=%v boards=%d", v.Phase, v.Boards)
	}
	if v.Tiles[24] != TileHidden {
		t.Fatalf("the new board should start hidden, tile = %d", v.Tiles[24])
	}
	want := int((BlitzTime + BlitzBonus - 10*sec).Milliseconds())
	if v.LimitMs != want {
		t.Fatalf("time left = %d, want %d", v.LimitMs, want)
	}
}

func TestBlitzBonusShrinksSoARunAlwaysEnds(t *testing.T) {
	m := blitz(t, open5())
	last := m.View(0, CountdownTime).LimitMs
	previous := int(BlitzBonus.Milliseconds()) + 1
	gain := 0
	for range 20 {
		act(m, CountdownTime, 0, Reveal, 24)
		left := m.View(0, CountdownTime).LimitMs
		if gain = left - last; gain >= previous {
			t.Fatalf("bonus went from %d to %d", previous, gain)
		}
		if gain < int(BlitzMinBonus.Milliseconds()) {
			t.Fatalf("bonus fell to %d", gain)
		}
		previous, last = gain+1, left
	}
	if gain != int(BlitzMinBonus.Milliseconds()) {
		t.Fatalf("bonus settled at %d, want the floor", gain)
	}
}

func TestBlitzMineCostsTimeAndStuns(t *testing.T) {
	m := blitz(t, strip())
	now := CountdownTime + sec
	act(m, now, 0, Reveal, 1)
	v := m.View(0, now)
	if want := int((BlitzTime - BlitzPenalty - sec).Milliseconds()); v.LimitMs != want {
		t.Fatalf("time left = %d, want %d", v.LimitMs, want)
	}
	if v.Players[0].StunMs <= 0 {
		t.Fatal("a mine should stun")
	}
	act(m, now, 0, Reveal, 0)
	if m.View(0, now).Tiles[0] != TileHidden {
		t.Fatal("a stunned player can't open tiles")
	}
}

func TestBlitzTimeoutWinsOnlyWithABoardCleared(t *testing.T) {
	for _, tc := range []struct {
		clear bool
		won   bool
	}{{false, false}, {true, true}} {
		m := blitz(t, open5())
		if tc.clear {
			act(m, CountdownTime, 0, Reveal, 24)
		}
		end := CountdownTime + BlitzTime + BlitzBonus
		m.Tick(end)
		v := m.View(0, end)
		if v.Phase != Over || v.Won != tc.won {
			t.Fatalf("clear=%v: phase=%v won=%v", tc.clear, v.Phase, v.Won)
		}
	}
}

func TestBlitzPenaltiesCanRunTheClockOut(t *testing.T) {
	m := blitz(t, strip())
	now := CountdownTime + BlitzTime - BlitzPenalty + sec
	act(m, now, 0, Reveal, 1)
	m.Tick(now)
	if m.Phase() != Over {
		t.Fatalf("phase = %v with no time left", m.Phase())
	}
}

func TestBlitzBoardsDiffer(t *testing.T) {
	m := started(t, Config{Mode: Blitz, Seed: 3, Players: []string{"a"}})
	first := m.board
	m.nextBoard()
	if m.board == first {
		t.Fatal("no new board")
	}
	same := true
	for i := range first.Mine {
		same = same && first.Mine[i] == m.board.Mine[i]
	}
	if same {
		t.Fatal("the next board repeats the last layout")
	}
}

func stand(t *testing.T, b *engine.Board, players ...string) *Match {
	return started(t, Config{Mode: LastStand, Layout: b, Players: players})
}

func TestLastStandSafeTilePassesTheTurn(t *testing.T) {
	m := stand(t, strip(), "a", "b")
	act(m, 4*sec, 0, Reveal, 0)
	if v := m.View(0, 4*sec); v.Turn != 1 || v.Players[0].Score != 1 {
		t.Fatalf("turn=%d score=%d", v.Turn, v.Players[0].Score)
	}
}

func TestLastStandIgnoresTheWaitingPlayer(t *testing.T) {
	m := stand(t, strip(), "a", "b")
	act(m, 4*sec, 1, Reveal, 0)
	if v := m.View(0, 4*sec); v.Turn != 0 || v.Tiles[0] != TileHidden {
		t.Fatalf("turn=%d tile=%d", v.Turn, v.Tiles[0])
	}
}

func TestLastStandKeepsTheTurnWhenNothingHappens(t *testing.T) {
	m := stand(t, strip(), "a", "b")
	act(m, 4*sec, 0, Flag, 0)
	act(m, 4*sec, 0, Reveal, 0)
	act(m, 4*sec, 0, Chord, 2)
	if v := m.View(0, 4*sec); v.Turn != 0 {
		t.Fatalf("turn = %d after moves that opened nothing", v.Turn)
	}
}

func TestLastStandFlagsArePrivateAndNeedNoTurn(t *testing.T) {
	m := stand(t, strip(), "a", "b")
	act(m, 4*sec, 1, Flag, 1)
	if m.View(1, 4*sec).Tiles[1] != TileFlag {
		t.Fatal("the waiting player should be able to flag")
	}
	if m.View(0, 4*sec).Tiles[1] != TileHidden {
		t.Fatal("a flag leaked to the other player")
	}
}

func TestLastStandMineLeavesTheLastPlayerWinning(t *testing.T) {
	m := stand(t, strip(), "a", "b")
	act(m, 4*sec, 0, Reveal, 1)
	v := m.View(0, 4*sec)
	if v.Phase != Over || v.Winner != 1 || !v.Won || !v.Players[0].Out {
		t.Fatalf("phase=%v winner=%d won=%v out=%v", v.Phase, v.Winner, v.Won, v.Players[0].Out)
	}
}

func TestLastStandSkipsPlayersWhoAreOut(t *testing.T) {
	m := stand(t, strip(), "a", "b", "c")
	act(m, 4*sec, 0, Reveal, 1)
	if v := m.View(0, 4*sec); v.Phase != Playing || v.Turn != 1 {
		t.Fatalf("phase=%v turn=%d", v.Phase, v.Turn)
	}
	act(m, 4*sec, 1, Reveal, 0)
	act(m, 4*sec, 2, Reveal, 2)
	if got := m.View(0, 4*sec).Turn; got != 1 {
		t.Fatalf("turn = %d, want 1: player 0 is out", got)
	}
	act(m, 4*sec, 0, Reveal, 3)
	if m.View(0, 4*sec).Tiles[3] != TileHidden {
		t.Fatal("a player who is out can't move")
	}
}

func TestLastStandTimeoutPutsThePlayerOut(t *testing.T) {
	m := stand(t, strip(), "a", "b", "c")
	end := CountdownTime + LastStandTurn
	m.Tick(end)
	v := m.View(0, end)
	if !v.Players[0].Out || v.Turn != 1 {
		t.Fatalf("out=%v turn=%d", v.Players[0].Out, v.Turn)
	}
	if want := int(LastStandTurn.Milliseconds()); v.TurnMs != want {
		t.Fatalf("turn time = %d, want a fresh %d", v.TurnMs, want)
	}
}

func TestLastStandClearedBoardGoesToTheMostTiles(t *testing.T) {
	m := stand(t, engine.NewBoard(5, 1, []int{1}), "a", "b")
	act(m, 4*sec, 0, Reveal, 3)
	act(m, 4*sec, 1, Reveal, 0)
	v := m.View(0, 4*sec)
	if v.Phase != Over || v.Winner != 0 || !v.Won {
		t.Fatalf("phase=%v winner=%d won=%v", v.Phase, v.Winner, v.Won)
	}
}

func TestLastStandClearedBoardCanBeADraw(t *testing.T) {
	m := stand(t, strip(), "a", "b")
	for i, tile := range []int{0, 2, 3, 5} {
		act(m, 4*sec, i%2, Reveal, tile)
	}
	v := m.View(0, 4*sec)
	if v.Phase != Over || v.Winner != -1 || v.Won {
		t.Fatalf("phase=%v winner=%d won=%v", v.Phase, v.Winner, v.Won)
	}
}

func TestLastStandClearedBoardIgnoresPlayersWhoAreOut(t *testing.T) {
	m := stand(t, engine.NewBoard(9, 1, []int{1, 5}), "a", "b", "c")
	for _, a := range []Action{{0, Reveal, 3}, {1, Reveal, 6}, {2, Reveal, 0}, {0, Reveal, 1}, {1, Reveal, 7}} {
		m.Apply(4*sec, a)
	}
	v := m.View(0, 4*sec)
	if v.Phase != Over || v.Winner != 1 || !v.Players[0].Out || v.Players[0].Score != v.Players[1].Score {
		t.Fatalf("phase=%v winner=%d: a player who is out shouldn't tie for the win", v.Phase, v.Winner)
	}
}

func TestLastStandLeavingOnYourTurnPassesIt(t *testing.T) {
	m := stand(t, strip(), "a", "b", "c")
	m.Leave(4*sec, 0)
	if v := m.View(1, 4*sec); v.Phase != Playing || v.Turn != 1 {
		t.Fatalf("phase=%v turn=%d", v.Phase, v.Turn)
	}
}

func TestLastStandLastPlayerInWinsWhenAnotherLeaves(t *testing.T) {
	m := stand(t, strip(), "a", "b", "c")
	act(m, 4*sec, 0, Reveal, 1)
	m.Leave(4*sec, 2)
	if v := m.View(1, 4*sec); v.Phase != Over || v.Winner != 1 {
		t.Fatalf("phase=%v winner=%d", v.Phase, v.Winner)
	}
}

func TestBotWaitsItsTurnInLastStand(t *testing.T) {
	m := stand(t, strip(), "a", "b")
	if _, ok := NewBot(1, 1).Next(m.View(1, 4*sec)); ok {
		t.Fatal("the bot moved out of turn")
	}
}

func TestBotThatIsOutStopsPlaying(t *testing.T) {
	m := stand(t, strip(), "a", "b", "c")
	act(m, 4*sec, 0, Reveal, 1)
	v := m.View(0, 4*sec)
	v.Turn = 0
	if _, ok := NewBot(0, 1).Next(v); ok {
		t.Fatal("a bot that is out shouldn't move")
	}
}

func TestBotsFinishTheNewModes(t *testing.T) {
	names := []string{"a", "b", "c", "d"}
	for _, tc := range []struct {
		mode    Mode
		players int
	}{{Blitz, 1}, {Survival, 1}, {LastStand, 2}, {LastStand, 4}} {
		m := New(Config{Mode: tc.mode, Seed: 11, Players: names[:tc.players]}, 0)
		play(t, m, tc.players, 15*time.Minute)
	}
}

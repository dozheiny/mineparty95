package match

import (
	"testing"
	"time"

	"mineparty/internal/engine"
)

// play drives a match with one bot per player until it ends.
func play(t *testing.T, m *Match, players int, limit time.Duration) View {
	t.Helper()
	bots := make([]*Bot, players)
	for i := range bots {
		bots[i] = NewBot(i, uint64(i+1))
	}
	step := 50 * time.Millisecond
	for now := CountdownTime; now < limit; now += step {
		m.Tick(now)
		if m.Phase() == Over {
			return m.View(0, now)
		}
		for i, b := range bots {
			if a, ok := b.Next(m.View(i, now)); ok {
				m.Apply(now, a)
			}
		}
	}
	t.Fatalf("match still running after %v", limit)
	return View{}
}

func TestBotsFinishEveryMode(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cfg     Config
		players int
	}{
		{"race", Config{Mode: Race, Board: engine.Intermediate}, 3},
		{"coop", Config{Mode: Coop, Board: engine.Expert}, 4},
		{"flags", Config{Mode: Flags}, 2},
		{"territory", Config{Mode: Territory}, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.cfg.Seed = 11
			tc.cfg.Players = []string{"a", "b", "c", "d"}[:tc.players]
			v := play(t, New(tc.cfg, 0), tc.players, 30*time.Minute)
			if v.Phase != Over {
				t.Fatalf("phase = %v", v.Phase)
			}
		})
	}
}

func TestBotClearsNoGuessBoardWithoutHittingMines(t *testing.T) {
	m := New(Config{Mode: Coop, Board: engine.Expert, Seed: 21, Players: []string{"a", "b"}}, 0)
	if v := play(t, m, 2, 30*time.Minute); !v.Won {
		t.Fatal("logic alone should clear a no-guess board")
	}
}

func TestBotOnlyPlaysOnItsTurnInFlags(t *testing.T) {
	m := started(t, Config{Mode: Flags, Seed: 2, Players: []string{"a", "b"}})
	v := m.View(1-m.turn, CountdownTime)
	if a, ok := NewBot(v.You, 1).Next(v); ok {
		t.Fatalf("bot acted out of turn: %+v", a)
	}
}

func TestBotDoesNothingWhileStunned(t *testing.T) {
	m := started(t, Config{Mode: Race, Layout: strip(), Players: []string{"a", "b"}})
	act(m, CountdownTime, 0, Reveal, 1)
	if _, ok := NewBot(0, 1).Next(m.View(0, CountdownTime)); ok {
		t.Fatal("a stunned bot should wait")
	}
}

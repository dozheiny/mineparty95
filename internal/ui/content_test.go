package ui

import (
	"math"
	"strings"
	"testing"

	"mineparty/internal/match"
	"mineparty/internal/profile"
)

func TestEveryThemeHasAPalette(t *testing.T) {
	for _, th := range profile.Themes {
		if _, ok := themes[th.ID]; !ok {
			t.Errorf("theme %q would silently fall back to classic", th.ID)
		}
	}
}

func TestEverySkinHasArt(t *testing.T) {
	for _, it := range profile.Catalog(profile.KindFlag) {
		if _, ok := flagSkins[it.ID]; !ok {
			t.Errorf("no bitmap for flag %q", it.ID)
		}
	}
	for _, it := range profile.Catalog(profile.KindMine) {
		if _, ok := mineSkins[it.ID]; !ok {
			t.Errorf("no bitmap for mine %q", it.ID)
		}
	}
	for _, it := range profile.Catalog(profile.KindFace) {
		if _, ok := faceColors[it.ID]; !ok {
			t.Errorf("no colour for face %q", it.ID)
		}
	}
}

func TestSkinBitmapsFitATile(t *testing.T) {
	check := func(id string, rows []string, maxW, maxH int) {
		if len(rows) > maxH {
			t.Errorf("%s has %d rows", id, len(rows))
		}
		for _, row := range rows {
			if len(row) != len(rows[0]) || len(row) > maxW {
				t.Errorf("%s has a row of %d", id, len(row))
			}
		}
	}
	for id, rows := range flagSkins {
		check(id, rows, 12, 10)
	}
	for id, rows := range mineSkins {
		check(id, rows, 12, 12)
	}
}

func TestEveryLobbyModeHasABlurb(t *testing.T) {
	for _, mode := range lobbyModes {
		if modeBlurbs[mode] == "" {
			t.Errorf("no blurb for %v", mode)
		}
	}
}

func TestModeBlurbsClearTheDifficultyButtons(t *testing.T) {
	for mode, blurb := range modeBlurbs {
		lines := len(wrap(blurb, blurbWidth))
		if bottom := blurbTop + lines*15; bottom > difficultyTop {
			t.Errorf("%v: blurb ends at %d, buttons start at %d", mode, bottom, difficultyTop)
		}
	}
}

func TestDevelopersAreListed(t *testing.T) {
	if len(developers) == 0 {
		t.Fatal("the About page needs at least one developer")
	}
	for _, d := range developers {
		if d.Name == "" || d.Role == "" {
			t.Fatalf("incomplete entry: %+v", d)
		}
	}
}

func TestOutcomeOfTheNewModes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		v         match.View
		won, draw bool
	}{
		{"blitz win", match.View{Mode: match.Blitz, Won: true, Winner: -1}, true, false},
		{"blitz loss", match.View{Mode: match.Blitz, Winner: -1, You: -1}, false, false},
		{"survival win", match.View{Mode: match.Survival, Won: true, Winner: -1}, true, false},
		{"last stand win", match.View{Mode: match.LastStand, Winner: 1, You: 1}, true, false},
		{"last stand loss", match.View{Mode: match.LastStand, Winner: 0, You: 1}, false, false},
		{"last stand draw", match.View{Mode: match.LastStand, Winner: -1, You: 1}, false, true},
	} {
		if won, draw := outcome(&tc.v); won != tc.won || draw != tc.draw {
			t.Errorf("%s: won=%v draw=%v", tc.name, won, draw)
		}
	}
}

func TestLastStandStandingsPutTheWinnerFirst(t *testing.T) {
	v := &match.View{Mode: match.LastStand, Winner: 1, Players: []match.PlayerView{
		{Name: "out", Score: 40, Out: true}, {Name: "champ", Score: 5},
	}}
	lines := standings(v)
	if !strings.Contains(lines[0], "champ") {
		t.Fatalf("standings = %q", lines)
	}
}

func TestResultLinesStayOnScreen(t *testing.T) {
	lines := make([]string, 40)
	got := capLines(lines)
	if len(got) != maxResultLines {
		t.Fatalf("%d lines", len(got))
	}
	if h := 96 + len(got)*15 + 44; h > ScreenH {
		t.Fatalf("panel is %d tall on a %d screen", h, ScreenH)
	}
	if short := capLines([]string{"a", "b"}); len(short) != 2 {
		t.Fatalf("a short list was changed: %q", short)
	}
}

func TestNoteFrequencies(t *testing.T) {
	for name, want := range map[string]float64{"A4": 440, "A5": 880, "C4": 261.63, "G#4": 415.30} {
		if got := noteFreq(name); math.Abs(got-want) > .02 {
			t.Errorf("%s = %.2f, want %.2f", name, got, want)
		}
	}
}

func TestEveryBarOfTheSongIsFull(t *testing.T) {
	if len(leadBars) != len(bassRoots) {
		t.Fatalf("%d lead bars, %d bass roots", len(leadBars), len(bassRoots))
	}
	for i, bar := range leadBars {
		if n := len(strings.Fields(bar)); n != stepsPerBar {
			t.Errorf("bar %d has %d steps", i+1, n)
		}
	}
}

func TestMusicLoopsWithoutAClick(t *testing.T) {
	song := composeMusic()
	if want := len(leadBars) * stepsPerBar * stepSamples; len(song) != want {
		t.Fatalf("%d samples, want %d", len(song), want)
	}
	loudest := 0.0
	for _, s := range song {
		loudest = max(loudest, math.Abs(s))
	}
	if loudest < .1 || loudest > 1 {
		t.Fatalf("peak = %.2f", loudest)
	}
	if edge := math.Abs(song[0]) + math.Abs(song[len(song)-1]); edge > .02 {
		t.Fatalf("the loop point jumps by %.3f", edge)
	}
}

func TestMusicSitsUnderTheSoundEffects(t *testing.T) {
	if musicGain >= 1 {
		t.Fatalf("music gain = %.2f", musicGain)
	}
}

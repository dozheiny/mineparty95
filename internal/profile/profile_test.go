package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"mineparty/internal/match"
)

func has(rw Rewards, id string) bool {
	return slices.ContainsFunc(rw.Achievements, func(a Achievement) bool { return a.ID == id })
}

func win(key string, ms int) GameResult {
	return GameResult{Mode: match.Classic, Board: key, Won: true, ElapsedMs: ms, BV3: 20, Clicks: 25, Progress: 100, Flagged: true}
}

func TestEvenALossEarnsXP(t *testing.T) {
	p := New()
	rw := p.Record(GameResult{Mode: match.Classic, Board: "beginner", BV3: 20, Progress: 60})
	if rw.XP <= 0 {
		t.Fatal("every game must move the bar, or a bad run feels wasted")
	}
}

func TestWinningPaysMoreThanLosing(t *testing.T) {
	a, b := New(), New()
	game := GameResult{Mode: match.Classic, Board: "expert", BV3: 120, Progress: 100}
	lost := a.Record(game)
	game.Won = true
	won := b.Record(game)
	if won.XP <= lost.XP {
		t.Fatalf("won=%d lost=%d", won.XP, lost.XP)
	}
}

func TestIdlingThroughSharedBoardEarnsAlmostNothing(t *testing.T) {
	p := New()
	rw := p.Record(GameResult{Mode: match.Coop, Won: true, BV3: 300, Progress: 100, Score: 0})
	if rw.XP > 10 {
		t.Fatalf("a player who opened nothing earned %d XP", rw.XP)
	}
	if has(rw, "coop_win") {
		t.Fatal("watching CPUs win shouldn't earn the award")
	}
}

func TestSharedBoardXPFollowsPersonalContribution(t *testing.T) {
	few := New().Record(GameResult{Mode: match.Coop, Won: true, BV3: 300, Progress: 100, Score: 20})
	many := New().Record(GameResult{Mode: match.Coop, Won: true, BV3: 300, Progress: 100, Score: 200})
	if many.XP <= few.XP {
		t.Fatalf("few=%d many=%d", few.XP, many.XP)
	}
}

func TestLevelCurveStartsAtOneAndClimbs(t *testing.T) {
	if lv, into, need := Level(0); lv != 1 || into != 0 || need <= 0 {
		t.Fatalf("level=%d into=%d need=%d", lv, into, need)
	}
	_, _, first := Level(0)
	if lv, _, _ := Level(first); lv != 2 {
		t.Fatalf("level after first threshold = %d", lv)
	}
	_, _, second := Level(first)
	if second <= first {
		t.Fatal("each level should cost more than the last")
	}
}

func TestLevelUpIsReported(t *testing.T) {
	p := New()
	_, _, need := Level(0)
	p.XP = need - 1
	if rw := p.Record(win("beginner", 30000)); rw.LevelAfter != rw.LevelBefore+1 {
		t.Fatalf("before=%d after=%d", rw.LevelBefore, rw.LevelAfter)
	}
}

func TestPersonalBestTracksFastestWinOnly(t *testing.T) {
	p := New()
	if rw := p.Record(win("beginner", 30000)); !rw.NewBest {
		t.Fatal("first win is a record")
	}
	if rw := p.Record(win("beginner", 40000)); rw.NewBest {
		t.Fatal("a slower win is not a record")
	}
	if rw := p.Record(win("beginner", 20000)); !rw.NewBest || p.Best["beginner"] != 20000 {
		t.Fatalf("best=%d", p.Best["beginner"])
	}
	lost := win("beginner", 1)
	lost.Won = false
	if rw := p.Record(lost); rw.NewBest {
		t.Fatal("a loss can't set a record")
	}
}

func TestWinStreakResetsOnLoss(t *testing.T) {
	p := New()
	p.Record(win("beginner", 30000))
	p.Record(win("beginner", 30000))
	if p.WinStreak != 2 {
		t.Fatalf("streak = %d", p.WinStreak)
	}
	p.Record(GameResult{Mode: match.Classic, Board: "beginner"})
	if p.WinStreak != 0 || p.BestWinStreak != 2 {
		t.Fatalf("streak=%d best=%d", p.WinStreak, p.BestWinStreak)
	}
}

func daily(date string, won bool) GameResult {
	return GameResult{Mode: match.Classic, Daily: true, Date: date, Won: won, ElapsedMs: 60000, BV3: 50, Progress: 100}
}

func TestDailyStreakGrowsOnConsecutiveDays(t *testing.T) {
	p := New()
	p.Record(daily("2026-10-30", true))
	rw := p.Record(daily("2026-10-31", true))
	if p.DailyStreak != 2 || rw.DailyStreak != 2 {
		t.Fatalf("streak=%d", p.DailyStreak)
	}
}

func TestDailyStreakSurvivesMonthBoundary(t *testing.T) {
	p := New()
	p.Record(daily("2026-10-31", true))
	p.Record(daily("2026-11-01", true))
	if p.DailyStreak != 2 {
		t.Fatalf("streak = %d", p.DailyStreak)
	}
}

func TestDailyStreakBreaksAfterMissedDay(t *testing.T) {
	p := New()
	p.Record(daily("2026-10-28", true))
	p.Record(daily("2026-10-31", true))
	if p.DailyStreak != 1 {
		t.Fatalf("streak = %d", p.DailyStreak)
	}
}

func TestDailyPaysOutOnlyOncePerDay(t *testing.T) {
	p := New()
	first := p.Record(daily("2026-10-31", true))
	second := p.Record(daily("2026-10-31", true))
	if second.XP >= first.XP || p.DailyStreak != 1 {
		t.Fatalf("first=%d second=%d streak=%d", first.XP, second.XP, p.DailyStreak)
	}
}

func TestDailyLossAllowsRetry(t *testing.T) {
	p := New()
	p.Record(daily("2026-10-31", false))
	if p.DailyDone("2026-10-31") {
		t.Fatal("a failed attempt shouldn't complete the day")
	}
	p.Record(daily("2026-10-31", true))
	if !p.DailyDone("2026-10-31") || p.DailyStreak != 1 {
		t.Fatalf("done=%v streak=%d", p.DailyDone("2026-10-31"), p.DailyStreak)
	}
}

func TestRivalRecordsCountHumansOnly(t *testing.T) {
	p := New()
	p.Record(GameResult{Mode: match.Race, Won: true, Rivals: []string{"sam"}})
	p.Record(GameResult{Mode: match.Race, Rivals: []string{"sam"}})
	p.Record(GameResult{Mode: match.Race, Won: true, Rivals: []string{"sam"}})
	if r := p.Rivals["sam"]; r.Wins != 2 || r.Losses != 1 {
		t.Fatalf("record = %+v", r)
	}
}

func TestDrawChangesNoRivalRecord(t *testing.T) {
	p := New()
	p.Record(GameResult{Mode: match.Territory, Draw: true, Rivals: []string{"sam"}})
	if r := p.Rivals["sam"]; r.Wins != 0 || r.Losses != 0 {
		t.Fatalf("record = %+v", r)
	}
}

func TestAchievementsAwardOnce(t *testing.T) {
	p := New()
	if rw := p.Record(win("beginner", 30000)); !has(rw, "first_win") {
		t.Fatalf("achievements = %+v", rw.Achievements)
	}
	if rw := p.Record(win("beginner", 30000)); has(rw, "first_win") {
		t.Fatal("awarded twice")
	}
}

func TestFlaglessWinAchievement(t *testing.T) {
	p := New()
	r := win("intermediate", 90000)
	r.Flagged = false
	if rw := p.Record(r); !has(rw, "no_flags") {
		t.Fatal("expected the no-flags achievement")
	}
}

func TestEachMultiplayerModeHasAWinAchievement(t *testing.T) {
	for mode, id := range map[match.Mode]string{match.Race: "race_win", match.Flags: "flags_win", match.Coop: "coop_win", match.Territory: "territory_win"} {
		if rw := New().Record(GameResult{Mode: mode, Won: true, Score: 10}); !has(rw, id) {
			t.Errorf("%v: missing %s", mode, id)
		}
	}
}

func TestThemeUnlocksByLevel(t *testing.T) {
	p := New()
	if !p.ThemeUnlocked("classic") || p.ThemeUnlocked("gameboy") {
		t.Fatal("only the default theme starts unlocked")
	}
	for !p.ThemeUnlocked("gameboy") {
		p.XP += 50
	}
	lv, _, _ := Level(p.XP)
	if lv != ThemeByID("gameboy").Level {
		t.Fatalf("unlocked at level %d", lv)
	}
}

func TestLevellingReportsNewlyUnlockedThemes(t *testing.T) {
	p := New()
	for p.Level() < ThemeByID("gameboy").Level-1 {
		p.XP += 50
	}
	_, into, need := Level(p.XP)
	p.XP += need - into - 1
	rw := p.Record(win("beginner", 30000))
	if len(rw.Themes) != 1 || rw.Themes[0].ID != "gameboy" {
		t.Fatalf("themes = %+v", rw.Themes)
	}
}

func TestDailyStreakUnlocksVaporwave(t *testing.T) {
	p := New()
	for d := 1; d <= 7; d++ {
		p.Record(daily("2026-11-0"+string(rune('0'+d)), true))
	}
	if !p.ThemeUnlocked("vaporwave") {
		t.Fatalf("streak %d should unlock the theme", p.DailyStreak)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "profile.json")
	p := New()
	p.Name = "ada"
	p.Record(win("expert", 100000))
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "ada" || got.XP != p.XP || got.Best["expert"] != 100000 {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadingMissingFileGivesFreshProfile(t *testing.T) {
	p, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil || p.XP != 0 || p.Theme != "classic" {
		t.Fatalf("p=%+v err=%v", p, err)
	}
}

func TestCorruptSaveIsSetAsideNotDestroyed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.json")
	os.WriteFile(path, []byte("{not json"), 0o644)
	p, err := Load(path)
	if err == nil {
		t.Fatal("corruption should be reported")
	}
	if p == nil || p.Theme != "classic" {
		t.Fatalf("still need a usable profile, got %+v", p)
	}
	if data, _ := os.ReadFile(path + ".corrupt"); string(data) != "{not json" {
		t.Fatalf("backup = %q", data)
	}
}

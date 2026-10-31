package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"mineparty/internal/match"
)

const today = "2026-03-10"

func dateOf(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC)
}

// dateWith finds a date whose daily quests include the given quest.
func dateWith(t *testing.T, id string) string {
	t.Helper()
	for day := 1; day <= 28; day++ {
		for month := 1; month <= 12; month++ {
			date := Today(dateOf(2026, month, day))
			if slices.ContainsFunc(DailyQuests(date), func(q Quest) bool { return q.ID == id }) {
				return date
			}
		}
	}
	t.Fatalf("no date offers quest %q", id)
	return ""
}

func solo(key string, won bool) GameResult {
	r := win(key, 60000)
	r.Won = won
	r.Date = today
	return r
}

func TestEveryFinishedGamePaysCoins(t *testing.T) {
	if rw := New().Record(solo("beginner", false)); rw.Coins <= 0 {
		t.Fatalf("a loss paid %d coins", rw.Coins)
	}
}

func TestWinningPaysMoreCoinsThanLosing(t *testing.T) {
	lost := New().Record(solo("expert", false))
	won := New().Record(solo("expert", true))
	if won.Coins <= lost.Coins {
		t.Fatalf("won=%d lost=%d", won.Coins, lost.Coins)
	}
}

func TestCoinsLandInTheWallet(t *testing.T) {
	p := New()
	rw := p.Record(solo("expert", true))
	if p.Coins != rw.Coins || p.CoinsEarned != rw.Coins {
		t.Fatalf("wallet=%d earned=%d reward=%d", p.Coins, p.CoinsEarned, rw.Coins)
	}
}

func TestAWinStreakBoostsCoins(t *testing.T) {
	p := New()
	first := p.Record(solo("expert", true)).Lucky
	if first != 1 {
		t.Fatalf("lucky = %d on a plain game", first)
	}
	a, b := New(), New()
	b.WinStreak = 4
	plain := a.Record(GameResult{Mode: match.Classic, Won: true, BV3: 100, Progress: 100, Date: today})
	boosted := b.Record(GameResult{Mode: match.Classic, Won: true, BV3: 100, Progress: 100, Date: today})
	if boosted.Coins <= plain.Coins {
		t.Fatalf("boosted=%d plain=%d", boosted.Coins, plain.Coins)
	}
}

func TestLuckyRollsMultiplyCoins(t *testing.T) {
	game := GameResult{Mode: match.Classic, BV3: 100, Progress: 100, Date: "2020-01-01"}
	for _, tc := range []struct{ roll, want int }{{0, 1}, {87, 1}, {88, 2}, {97, 2}, {98, 5}, {99, 5}} {
		game.Roll = tc.roll
		if got := New().Record(game).Lucky; got != tc.want {
			t.Errorf("roll %d: lucky x%d, want x%d", tc.roll, got, tc.want)
		}
	}
	game.Roll = 0
	plain := New().Record(game).Coins
	game.Roll = 99
	if lucky := New().Record(game).Coins; lucky <= plain {
		t.Fatalf("lucky=%d plain=%d", lucky, plain)
	}
}

func TestReplayingTheDailyCantBeFarmed(t *testing.T) {
	p := New()
	game := GameResult{Mode: match.Classic, Daily: true, Date: today, Won: true, BV3: 100, Progress: 100, Roll: 99}
	p.Record(game)
	coins, rp := p.Coins, p.RP
	rw := p.Record(game)
	if rw.Coins > 1 || rw.RP != 0 || rw.Lucky != 1 {
		t.Fatalf("replay paid %d coins and %d RP at x%d", rw.Coins, rw.RP, rw.Lucky)
	}
	if p.Coins-coins != rw.Coins || p.RP != rp {
		t.Fatalf("wallet moved by %d, RP by %d", p.Coins-coins, p.RP-rp)
	}
}

func TestReplayingTheDailyDoesntAdvanceQuests(t *testing.T) {
	date := dateWith(t, "play_games")
	p := New()
	game := GameResult{Mode: match.Classic, Daily: true, Date: date, Won: true, BV3: 10, Progress: 100}
	p.Record(game)
	p.Record(game)
	for _, q := range p.Quests(date) {
		if q.ID == "play_games" && q.Progress != 1 {
			t.Fatalf("progress = %d after one real game", q.Progress)
		}
	}
}

func TestIdlingOnASharedBoardPaysNothingExtra(t *testing.T) {
	rw := New().Record(GameResult{Mode: match.Coop, Won: true, BV3: 300, Progress: 100, Score: 0, Date: today, Roll: 99})
	if rw.Coins > 1 || rw.RP != 0 {
		t.Fatalf("an idle player earned %d coins and %d RP", rw.Coins, rw.RP)
	}
}

func TestLevelUpsPayCoins(t *testing.T) {
	a, b := New(), New()
	b.XP = 95
	game := GameResult{Mode: match.Classic, BV3: 20, Progress: 100, Date: today}
	plain, up := a.Record(game), b.Record(game)
	if up.LevelAfter <= up.LevelBefore {
		t.Fatal("expected a level up")
	}
	if up.Coins < plain.Coins+LevelCoins(2) {
		t.Fatalf("plain=%d levelled=%d", plain.Coins, up.Coins)
	}
}

func TestAwardsPayCoins(t *testing.T) {
	a, b := New(), New()
	a.Achievements["first_win"] = today
	game := GameResult{Mode: match.Classic, Won: true, BV3: 20, Progress: 100, Date: today, Flagged: true}
	plain, award := a.Record(game), b.Record(game)
	if !has(award, "first_win") || award.Coins <= plain.Coins {
		t.Fatalf("plain=%d award=%d", plain.Coins, award.Coins)
	}
}

func TestBlitzRewardsBoardsCleared(t *testing.T) {
	few := New().Record(GameResult{Mode: match.Blitz, Won: true, Boards: 1, Date: today})
	many := New().Record(GameResult{Mode: match.Blitz, Won: true, Boards: 6, Date: today})
	if many.XP <= few.XP || many.RP <= few.RP {
		t.Fatalf("few=%+v many=%+v", few, many)
	}
}

func TestBlitzKeepsTheMostBoards(t *testing.T) {
	p := New()
	if !p.Record(GameResult{Mode: match.Blitz, Board: "blitz", Won: true, Boards: 3, Date: today}).NewBest {
		t.Fatal("the first run is a record")
	}
	if p.Record(GameResult{Mode: match.Blitz, Board: "blitz", Won: true, Boards: 2, Date: today}).NewBest {
		t.Fatal("fewer boards isn't a record")
	}
	if p.BestBoards != 3 {
		t.Fatalf("best = %d", p.BestBoards)
	}
}

func TestLastStandIdlerEarnsAlmostNothing(t *testing.T) {
	rw := New().Record(GameResult{Mode: match.LastStand, Won: true, Score: 0, Date: today})
	if rw.XP > 10 || rw.RP != 0 {
		t.Fatalf("xp=%d rp=%d", rw.XP, rw.RP)
	}
}

func TestOldSavesKeepTheirProgress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.json")
	old := `{"Name":"Zero","XP":476,"Theme":"gameboy","Volume":7,"Best":{},"Played":{"Classic":5},
		"Wins":{},"Rivals":{"zero":{"Wins":0,"Losses":2}},"Achievements":{}}`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.XP != 476 || p.Theme != "gameboy" || p.Coins != 0 {
		t.Fatalf("loaded %+v", p)
	}
	if !p.Owns("gameboy") {
		t.Fatal("a theme earned before the shop existed must stay owned")
	}
	if p.Equipped(KindFlag) != "flag_classic" {
		t.Fatalf("flag = %q", p.Equipped(KindFlag))
	}
	p.Coins = 1000
	if err := p.Buy("flag_star"); err != nil {
		t.Fatalf("buying on an old save: %v", err)
	}
	p.Record(solo("beginner", true))
}

func TestNewFieldsSurviveASave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.json")
	p := New()
	p.Coins, p.RP, p.Freezes = 500, 700, 1
	if err := p.Buy("flag_star"); err != nil {
		t.Fatal(err)
	}
	if err := p.Equip("flag_star"); err != nil {
		t.Fatal(err)
	}
	p.Record(solo("beginner", true))
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Coins != p.Coins || got.RP != p.RP || got.Freezes != 1 || got.Equipped(KindFlag) != "flag_star" || !got.Owns("flag_star") {
		t.Fatalf("loaded %+v", got)
	}
	if got.QuestProgress == nil || len(got.Quests(today)) != QuestsPerDay {
		t.Fatalf("quests lost: %+v", got.QuestProgress)
	}
}

func TestBuyingSpendsCoinsAndKeepsTheItem(t *testing.T) {
	p := New()
	price := mustItem(t, "flag_star").Price
	p.Coins = price + 7
	if err := p.Buy("flag_star"); err != nil {
		t.Fatal(err)
	}
	if p.Coins != 7 || !p.Owns("flag_star") {
		t.Fatalf("coins=%d owns=%v", p.Coins, p.Owns("flag_star"))
	}
}

func mustItem(t *testing.T, id string) Item {
	t.Helper()
	it, ok := ItemByID(id)
	if !ok {
		t.Fatalf("no item %q", id)
	}
	return it
}

func TestBuyingRefusesWithoutChargingAnything(t *testing.T) {
	p := New()
	p.Coins = mustItem(t, "flag_star").Price - 1
	before := p.Coins
	if p.Buy("flag_star") == nil || p.Owns("flag_star") || p.Coins != before {
		t.Fatal("bought without enough coins")
	}
	p.Coins = 100000
	if err := p.Buy("flag_star"); err != nil {
		t.Fatal(err)
	}
	before = p.Coins
	if p.Buy("flag_star") == nil || p.Coins != before {
		t.Fatal("bought the same item twice")
	}
	if p.Buy("no_such_item") == nil {
		t.Fatal("bought an item that doesn't exist")
	}
	if p.Buy("gameboy") == nil {
		t.Fatal("a theme earned by levelling isn't for sale")
	}
}

func TestPurchasedThemeIsLockedUntilBought(t *testing.T) {
	p := New()
	if p.ThemeUnlocked("midnight") {
		t.Fatal("midnight should start locked")
	}
	p.Coins = mustItem(t, "midnight").Price
	if err := p.Buy("midnight"); err != nil {
		t.Fatal(err)
	}
	if !p.ThemeUnlocked("midnight") {
		t.Fatal("a bought theme should unlock")
	}
	if err := p.Equip("midnight"); err != nil || p.Theme != "midnight" {
		t.Fatalf("equip: %v, theme %q", err, p.Theme)
	}
}

func TestEquippingNeedsOwnership(t *testing.T) {
	p := New()
	if p.Equip("flag_star") == nil || p.Equipped(KindFlag) != "flag_classic" {
		t.Fatal("equipped an item the player doesn't own")
	}
	if p.Equip("midnight") == nil || p.Theme != "classic" {
		t.Fatal("equipped a locked theme")
	}
	if p.Equip("flag_classic") != nil {
		t.Fatal("the free default should always equip")
	}
}

func TestEquippedTitleReplacesTheLevelTitle(t *testing.T) {
	p := New()
	if p.DisplayTitle() != Title(1) {
		t.Fatalf("title = %q", p.DisplayTitle())
	}
	it := Catalog(KindTitle)[1]
	p.Coins = it.Price
	if err := p.Buy(it.ID); err != nil {
		t.Fatal(err)
	}
	if err := p.Equip(it.ID); err != nil {
		t.Fatal(err)
	}
	if p.DisplayTitle() != it.Name {
		t.Fatalf("title = %q, want %q", p.DisplayTitle(), it.Name)
	}
}

func TestCatalogIDsAreUniqueAndEveryKindSells(t *testing.T) {
	seen := map[string]bool{}
	for _, kind := range []ItemKind{KindTheme, KindFlag, KindMine, KindFace, KindTitle, KindFreeze} {
		forSale := 0
		for _, it := range Catalog(kind) {
			if seen[it.ID] {
				t.Errorf("duplicate id %q", it.ID)
			}
			seen[it.ID] = true
			if it.Price > 0 {
				forSale++
			}
		}
		if forSale == 0 {
			t.Errorf("kind %d sells nothing", kind)
		}
	}
}

func TestFreezesAreCapped(t *testing.T) {
	p := New()
	p.Coins = 100000
	for range MaxFreezes {
		if err := p.Buy("freeze"); err != nil {
			t.Fatal(err)
		}
	}
	if p.Buy("freeze") == nil || p.Freezes != MaxFreezes {
		t.Fatalf("freezes = %d", p.Freezes)
	}
}

func dailyOn(date string) GameResult {
	return GameResult{Mode: match.Classic, Daily: true, Date: date, Won: true, BV3: 10, Progress: 100}
}

func TestFreezeCoversOneMissedDaily(t *testing.T) {
	p := New()
	p.Record(dailyOn("2026-03-01"))
	p.Record(dailyOn("2026-03-02"))
	p.Freezes = 1
	if got := p.DailyStreakAlive("2026-03-04"); got != 2 {
		t.Fatalf("alive = %d, the freeze should hold the streak", got)
	}
	rw := p.Record(dailyOn("2026-03-04"))
	if rw.DailyStreak != 3 || p.Freezes != 0 {
		t.Fatalf("streak=%d freezes=%d", rw.DailyStreak, p.Freezes)
	}
}

func TestFreezeDoesntCoverTwoMissedDailies(t *testing.T) {
	p := New()
	p.Record(dailyOn("2026-03-01"))
	p.Freezes = 1
	if got := p.DailyStreakAlive("2026-03-04"); got != 0 {
		t.Fatalf("alive = %d", got)
	}
	if rw := p.Record(dailyOn("2026-03-04")); rw.DailyStreak != 1 || p.Freezes != 1 {
		t.Fatalf("streak=%d freezes=%d", rw.DailyStreak, p.Freezes)
	}
}

func TestFreezeIsntSpentOnAnUnbrokenStreak(t *testing.T) {
	p := New()
	p.Freezes = 1
	p.Record(dailyOn("2026-03-01"))
	p.Record(dailyOn("2026-03-02"))
	if p.Freezes != 1 || p.DailyStreak != 2 {
		t.Fatalf("freezes=%d streak=%d", p.Freezes, p.DailyStreak)
	}
}

func TestLoginBonusPaysOncePerDay(t *testing.T) {
	p := New()
	coins, day := p.ClaimLogin("2026-03-01")
	if coins <= 0 || day != 1 || p.Coins != coins {
		t.Fatalf("coins=%d day=%d wallet=%d", coins, day, p.Coins)
	}
	if coins, day := p.ClaimLogin("2026-03-01"); coins != 0 || day != 0 {
		t.Fatalf("paid twice: coins=%d day=%d", coins, day)
	}
}

func TestLoginBonusClimbsToADaySevenPrizeThenCycles(t *testing.T) {
	p := New()
	var paid []int
	for d := 1; d <= 8; d++ {
		coins, _ := p.ClaimLogin(Today(dateOf(2026, 3, d)))
		paid = append(paid, coins)
	}
	for i := 1; i < 7; i++ {
		if paid[i] <= paid[i-1] {
			t.Fatalf("day %d paid %d after %d", i+1, paid[i], paid[i-1])
		}
	}
	if paid[7] != paid[0] || p.LoginDay != 1 {
		t.Fatalf("day 8 paid %d on cycle day %d", paid[7], p.LoginDay)
	}
}

func TestMissingADayRestartsTheLoginCycle(t *testing.T) {
	p := New()
	p.ClaimLogin("2026-03-01")
	p.ClaimLogin("2026-03-02")
	if _, day := p.ClaimLogin("2026-03-04"); day != 1 {
		t.Fatalf("day = %d, want 1", day)
	}
}

func TestDailyQuestsAreStableForADateAndDistinct(t *testing.T) {
	a, b := DailyQuests(today), DailyQuests(today)
	if len(a) != QuestsPerDay {
		t.Fatalf("%d quests", len(a))
	}
	ids := map[string]bool{}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatal("the same date gave different quests")
		}
		ids[a[i].ID] = true
	}
	if len(ids) != QuestsPerDay {
		t.Fatalf("repeated quests: %v", ids)
	}
}

func TestDailyQuestsChangeFromDayToDay(t *testing.T) {
	first := DailyQuests("2026-03-01")
	for d := 2; d <= 28; d++ {
		next := DailyQuests(Today(dateOf(2026, 3, d)))
		if next[0].ID != first[0].ID || next[1].ID != first[1].ID {
			return
		}
	}
	t.Fatal("quests never change")
}

func TestQuestPaysOnceWhenFinished(t *testing.T) {
	date := dateWith(t, "play_games")
	p := New()
	game := GameResult{Mode: match.Classic, Date: date, BV3: 1, Progress: 1}
	var done []Quest
	for range 6 {
		done = append(done, p.Record(game).Quests...)
	}
	count := 0
	for _, q := range done {
		if q.ID == "play_games" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("paid %d times", count)
	}
}

func TestQuestProgressStartsOverEachDay(t *testing.T) {
	date := dateWith(t, "play_games")
	p := New()
	p.Record(GameResult{Mode: match.Classic, Date: date, BV3: 1, Progress: 1})
	next := Today(dateOf(2026, 12, 31))
	for _, q := range p.Quests(next) {
		if q.Progress != 0 {
			t.Fatalf("%s carried over %d", q.ID, q.Progress)
		}
	}
}

func TestFinishingEveryQuestPaysABonusAndAnAward(t *testing.T) {
	date := dateWith(t, "play_games")
	p := New()
	p.QuestDate = date
	p.QuestProgress = map[string]int{}
	quests := DailyQuests(date)
	for _, q := range quests {
		p.QuestProgress[q.ID] = q.Goal
	}
	p.QuestProgress["play_games"] = 0
	for _, q := range quests {
		if q.ID == "play_games" {
			p.QuestProgress[q.ID] = q.Goal - 1
		}
	}
	rw := p.Record(GameResult{Mode: match.Classic, Date: date, BV3: 1, Progress: 1})
	if !rw.QuestBonus || !has(rw, "quest_all") {
		t.Fatalf("bonus=%v awards=%v", rw.QuestBonus, rw.Achievements)
	}
	if rw2 := p.Record(GameResult{Mode: match.Classic, Date: date, BV3: 1, Progress: 1}); rw2.QuestBonus {
		t.Fatal("the bonus paid twice")
	}
}

func TestRankNamesFollowTiersAndDivisions(t *testing.T) {
	for _, tc := range []struct {
		rp   int
		want string
	}{{0, "Bronze III"}, {Tiers[1].MinRP - 1, "Bronze I"}, {Tiers[1].MinRP, "Silver III"}, {Tiers[len(Tiers)-1].MinRP + 5000, "Master"}} {
		if got := RankOf(tc.rp).Name; got != tc.want {
			t.Errorf("rank at %d RP = %q, want %q", tc.rp, got, tc.want)
		}
	}
}

func TestRankProgressMeasuresTheCurrentDivision(t *testing.T) {
	r := RankOf(Tiers[1].MinRP - 1)
	if r.Need <= 0 || r.Into != r.Need-1 {
		t.Fatalf("%+v", r)
	}
	if top := RankOf(Tiers[len(Tiers)-1].MinRP); top.Need != 0 {
		t.Fatalf("the top rank has nowhere to go: %+v", top)
	}
}

func TestWinsClimbTheLadderAndLossesFall(t *testing.T) {
	p := New()
	p.RP = 100
	if rw := p.Record(solo("expert", true)); rw.RP <= 0 || p.RP != 100+rw.RP {
		t.Fatalf("win moved %d, RP %d", rw.RP, p.RP)
	}
	before := p.RP
	if rw := p.Record(solo("expert", false)); rw.RP >= 0 || p.RP != before+rw.RP {
		t.Fatalf("loss moved %d, RP %d", rw.RP, p.RP)
	}
}

func TestHarderGamesClimbFaster(t *testing.T) {
	easy := New().Record(solo("beginner", true))
	hard := New().Record(solo("expert", true))
	if hard.RP <= easy.RP {
		t.Fatalf("easy=%d hard=%d", easy.RP, hard.RP)
	}
}

func TestBeatingFriendsClimbsFasterThanBeatingCPUs(t *testing.T) {
	cpu := New().Record(GameResult{Mode: match.Race, Won: true, Score: 50, Date: today})
	friend := New().Record(GameResult{Mode: match.Race, Won: true, Score: 50, Date: today, Rivals: []string{"zero"}})
	if friend.RP <= cpu.RP {
		t.Fatalf("cpu=%d friend=%d", cpu.RP, friend.RP)
	}
}

func TestLossesNeverDropBelowTheCurrentTier(t *testing.T) {
	p := New()
	p.RP = Tiers[1].MinRP + 1
	p.BestTier = 1
	for range 5 {
		p.Record(solo("expert", false))
	}
	if p.RP != Tiers[1].MinRP {
		t.Fatalf("RP = %d, floor is %d", p.RP, Tiers[1].MinRP)
	}
}

func TestReachingATierPaysOnce(t *testing.T) {
	p := New()
	p.RP = Tiers[1].MinRP - 1
	rw := p.Record(solo("expert", true))
	if len(rw.Promotions) != 1 || rw.Promotions[0].Name != Tiers[1].Name {
		t.Fatalf("promotions = %v", rw.Promotions)
	}
	if p.BestTier != 1 {
		t.Fatalf("best tier = %d", p.BestTier)
	}
	if rw := p.Record(solo("expert", true)); len(rw.Promotions) != 0 {
		t.Fatalf("paid again: %v", rw.Promotions)
	}
}

func TestNewAwards(t *testing.T) {
	p := New()
	if !has(p.Record(GameResult{Mode: match.Blitz, Board: "blitz", Won: true, Boards: 5, Date: today}), "blitz5") {
		t.Fatal("blitz5")
	}
	if !has(p.Record(GameResult{Mode: match.Survival, Board: "survival", Won: true, BV3: 100, Progress: 100, Date: today}), "survivor") {
		t.Fatal("survivor")
	}
	if !has(p.Record(GameResult{Mode: match.LastStand, Won: true, Score: 20, Date: today}), "last_stand") {
		t.Fatal("last_stand")
	}
	q := New()
	q.RP = Tiers[2].MinRP - 1
	if !has(q.Record(solo("expert", true)), "gold_rank") {
		t.Fatal("gold_rank")
	}
	c := New()
	c.CoinsEarned = 2000
	if !has(c.Record(solo("beginner", true)), "coins2k") {
		t.Fatal("coins2k")
	}
}

// Package profile is the player's local save: progress, records, rivalries
// and unlocks. With no server, everything that keeps people coming back
// lives here, on their own machine.
package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"mineparty/internal/match"
)

// Rival is a head-to-head record against one friend.
type Rival struct {
	Wins, Losses int
}

// Profile is everything saved between sessions.
type Profile struct {
	Name          string
	XP            int
	Theme         string
	Volume        int // 0 to 10
	PureRandom    bool
	Fullscreen    bool
	Best          map[string]int // fastest win in ms, by board key
	Played        map[string]int // games by mode name
	Wins          map[string]int // wins by mode name
	WinStreak     int
	BestWinStreak int
	DailyStreak   int
	BestDaily     int
	LastDaily     string
	Rivals        map[string]Rival
	Achievements  map[string]string // id to date earned
}

// defaultName uses the OS account name, so that friends' games don't all
// start out with the same name.
func defaultName() string {
	u, err := user.Current()
	if err != nil {
		return "Player"
	}
	name := u.Username[strings.LastIndexAny(u.Username, `\/`)+1:]
	if name == "" {
		return "Player"
	}
	return name[:min(len(name), 14)]
}

// New returns an empty profile.
func New() *Profile {
	return &Profile{
		Name:         defaultName(),
		Theme:        "classic",
		Volume:       7,
		Best:         map[string]int{},
		Played:       map[string]int{},
		Wins:         map[string]int{},
		Rivals:       map[string]Rival{},
		Achievements: map[string]string{},
	}
}

// DefaultPath returns where the save lives for this user.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mineparty", "profile.json"), nil
}

// Load reads a save. A missing file gives a fresh profile. A corrupt file is
// moved aside to path.corrupt, and a fresh profile comes back with the error.
func Load(path string) (*Profile, error) {
	p := New()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(data, p); err != nil {
		backup := path + ".corrupt"
		if moveErr := os.Rename(path, backup); moveErr != nil {
			return New(), moveErr
		}
		return New(), fmt.Errorf("save file was damaged and moved to %s: %w", backup, err)
	}
	p.fillMaps()
	return p, nil
}

func (p *Profile) fillMaps() {
	for _, m := range []*map[string]int{&p.Best, &p.Played, &p.Wins} {
		if *m == nil {
			*m = map[string]int{}
		}
	}
	if p.Rivals == nil {
		p.Rivals = map[string]Rival{}
	}
	if p.Achievements == nil {
		p.Achievements = map[string]string{}
	}
}

// Save writes the profile atomically, so a crash can't truncate it.
func (p *Profile) Save(path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Level turns total XP into a level, plus the progress within that level.
func Level(xp int) (level, into, need int) {
	level, need = 1, levelCost(1)
	for xp >= need {
		xp -= need
		level++
		need = levelCost(level)
	}
	return level, xp, need
}

func levelCost(level int) int { return 50 + 50*level }

// Level returns the player's current level.
func (p *Profile) Level() int {
	lv, _, _ := Level(p.XP)
	return lv
}

var titles = []string{
	"Recruit", "Sapper", "Corporal", "Sergeant", "Lieutenant", "Captain",
	"Major", "Colonel", "Brigadier", "General", "Field Marshal", "Legend",
}

// Title names a level.
func Title(level int) string {
	return titles[min(level-1, len(titles)-1)]
}

// Theme is a cosmetic that can be earned.
type Theme struct {
	ID, Name string
	Level    int    // level that unlocks it, or 0
	Award    string // achievement that unlocks it, or empty
}

// Themes lists every theme in unlock order.
var Themes = []Theme{
	{ID: "classic", Name: "Desktop '95"},
	{ID: "gameboy", Name: "Pocket Green", Level: 3},
	{ID: "amber", Name: "Amber Terminal", Level: 5},
	{ID: "arcade", Name: "Neon Arcade", Level: 8},
	{ID: "vaporwave", Name: "Vaporwave", Award: "daily7"},
}

// ThemeByID finds a theme, falling back to the default.
func ThemeByID(id string) Theme {
	for _, t := range Themes {
		if t.ID == id {
			return t
		}
	}
	return Themes[0]
}

// Unlocks describes what earns a theme.
func (t Theme) Unlocks() string {
	switch {
	case t.Level > 0:
		return fmt.Sprintf("Reach level %d", t.Level)
	case t.Award != "":
		return "Earn: " + AchievementByID(t.Award).Name
	}
	return "Unlocked"
}

// ThemeUnlocked reports whether the player has earned a theme.
func (p *Profile) ThemeUnlocked(id string) bool {
	t := ThemeByID(id)
	if t.Award != "" {
		_, ok := p.Achievements[t.Award]
		return ok
	}
	return p.Level() >= t.Level
}

func (p *Profile) unlockedThemes() []string {
	var ids []string
	for _, t := range Themes {
		if p.ThemeUnlocked(t.ID) {
			ids = append(ids, t.ID)
		}
	}
	return ids
}

// Achievement is a one-time goal.
type Achievement struct {
	ID, Name, Desc string
}

// GameResult summarises a finished game for the local player.
type GameResult struct {
	Mode      match.Mode
	Board     string // beginner, intermediate or expert; empty otherwise
	Daily     bool
	Date      string
	Won       bool
	Draw      bool
	ElapsedMs int
	BV3       int
	Clicks    int
	Progress  int
	Score     int // what this player personally earned on a shared board
	Flagged   bool
	UsedBomb  bool
	Rivals    []string // human opponents only
}

// Rewards is what a game earned, for the results screen.
type Rewards struct {
	XP           int
	LevelBefore  int
	LevelAfter   int
	NewBest      bool
	DailyStreak  int
	Achievements []Achievement
	Themes       []Theme
}

// Today returns a date in the form results and streaks use.
func Today(t time.Time) string { return t.Format(dateLayout) }

const dateLayout = "2006-01-02"

// DailyDone reports whether the day's challenge has been completed.
func (p *Profile) DailyDone(date string) bool { return p.LastDaily == date }

// DailyStreakAlive returns the streak still in reach on the given date.
func (p *Profile) DailyStreakAlive(date string) int {
	if p.LastDaily == date || p.LastDaily == dayBefore(date) {
		return p.DailyStreak
	}
	return 0
}

func dayBefore(date string) string {
	t, err := time.Parse(dateLayout, date)
	if err != nil {
		return ""
	}
	return t.AddDate(0, 0, -1).Format(dateLayout)
}

// Record applies a finished game and reports what it earned.
func (p *Profile) Record(r GameResult) Rewards {
	rw := Rewards{LevelBefore: p.Level()}
	themesBefore := p.unlockedThemes()
	key := r.Mode.String()
	p.Played[key]++

	replayedDaily := r.Daily && p.DailyDone(r.Date)
	rw.XP = xpFor(r)
	if r.Won {
		p.Wins[key]++
		if !replayedDaily {
			p.WinStreak++
			p.BestWinStreak = max(p.BestWinStreak, p.WinStreak)
		}
		rw.NewBest = p.updateBest(r)
	} else if !r.Draw {
		p.WinStreak = 0
	}

	if r.Daily {
		if replayedDaily {
			rw.XP = 5
		} else if r.Won {
			if p.LastDaily == dayBefore(r.Date) {
				p.DailyStreak++
			} else {
				p.DailyStreak = 1
			}
			p.LastDaily = r.Date
			p.BestDaily = max(p.BestDaily, p.DailyStreak)
			rw.XP += 100 + 20*min(p.DailyStreak, 7)
		}
		rw.DailyStreak = p.DailyStreak
	}
	if !r.Draw {
		for _, name := range r.Rivals {
			rec := p.Rivals[name]
			if r.Won {
				rec.Wins++
			} else {
				rec.Losses++
			}
			p.Rivals[name] = rec
		}
	}

	p.XP += rw.XP
	rw.Achievements = p.award(r)
	rw.LevelAfter = p.Level()
	for _, id := range p.unlockedThemes() {
		if !slices.Contains(themesBefore, id) {
			rw.Themes = append(rw.Themes, ThemeByID(id))
		}
	}
	return rw
}

func (p *Profile) updateBest(r GameResult) bool {
	key := r.Board
	if r.Daily {
		key = "daily"
	}
	if key == "" {
		return false
	}
	if best, ok := p.Best[key]; ok && r.ElapsedMs >= best {
		return false
	}
	p.Best[key] = r.ElapsedMs
	return true
}

func xpFor(r GameResult) int {
	var xp int
	switch r.Mode {
	case match.Coop, match.Territory:
		xp = 10 + r.Score/3
	case match.Flags:
		xp = 10 + r.Score*5
	default:
		xp = 10 + r.BV3*r.Progress/100
	}
	// Board-wide progress can't be the measure on a shared board, or a
	// player could idle while CPUs do the work.
	if sharedBoard(r.Mode) && r.Score == 0 {
		return 10
	}
	if r.Won {
		xp *= 2
	}
	if r.Mode != match.Classic {
		xp += 20
		if r.Won {
			xp += 30
		}
	}
	return xp
}

func sharedBoard(m match.Mode) bool {
	return m == match.Coop || m == match.Flags || m == match.Territory
}

var achievements = []Achievement{
	{"first_win", "First Steps", "Win a solo game"},
	{"quick", "Quick Fingers", "Win Beginner in under 15 seconds"},
	{"inter_win", "Getting Serious", "Win Intermediate"},
	{"expert_win", "Bomb Squad", "Win Expert"},
	{"no_flags", "Flagless", "Win Intermediate or Expert with no flags"},
	{"efficient", "Efficient", "Win a large board within 10% of its 3BV clicks"},
	{"streak5", "Hot Streak", "Win 5 games in a row"},
	{"daily3", "Regular", "Keep a 3-day daily streak"},
	{"daily7", "Devoted", "Keep a 7-day daily streak"},
	{"race_win", "Photo Finish", "Win a Race"},
	{"flags_win", "Flag Captain", "Win a Mine Flags duel"},
	{"coop_win", "Teamwork", "Clear a Co-op board"},
	{"territory_win", "Land Grab", "Win a Territory battle"},
	{"comeback", "Comeback Kid", "Win Mine Flags after using your bomb"},
	{"social", "Party Animal", "Play 25 games against friends"},
	{"level10", "Veteran", "Reach level 10"},
}

// Achievements lists every achievement.
func Achievements() []Achievement { return achievements }

// AchievementByID finds an achievement by its ID.
func AchievementByID(id string) Achievement {
	for _, a := range achievements {
		if a.ID == id {
			return a
		}
	}
	return Achievement{ID: id, Name: id}
}

func (p *Profile) award(r GameResult) []Achievement {
	large := r.Board == "intermediate" || r.Board == "expert"
	solo := r.Mode == match.Classic
	social := 0
	for _, rec := range p.Rivals {
		social += rec.Wins + rec.Losses
	}
	earned := map[string]bool{
		"first_win":     solo && r.Won,
		"quick":         solo && r.Won && r.Board == "beginner" && r.ElapsedMs < 15000,
		"inter_win":     solo && r.Won && r.Board == "intermediate",
		"expert_win":    solo && r.Won && r.Board == "expert",
		"no_flags":      solo && r.Won && large && !r.Flagged,
		"efficient":     solo && r.Won && large && r.Clicks > 0 && r.Clicks*10 <= r.BV3*11,
		"streak5":       p.WinStreak >= 5,
		"daily3":        p.DailyStreak >= 3,
		"daily7":        p.DailyStreak >= 7,
		"race_win":      r.Mode == match.Race && r.Won,
		"flags_win":     r.Mode == match.Flags && r.Won && r.Score > 0,
		"coop_win":      r.Mode == match.Coop && r.Won && r.Score > 0,
		"territory_win": r.Mode == match.Territory && r.Won && r.Score > 0,
		"comeback":      r.Mode == match.Flags && r.Won && r.UsedBomb && r.Score > 0,
		"social":        social >= 25,
		"level10":       p.Level() >= 10,
	}
	var out []Achievement
	for _, a := range achievements {
		if _, have := p.Achievements[a.ID]; earned[a.ID] && !have {
			p.Achievements[a.ID] = r.Date
			out = append(out, a)
		}
	}
	return out
}

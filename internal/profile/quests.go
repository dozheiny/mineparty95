package profile

import (
	"hash/fnv"
	"math/rand/v2"

	"mineparty/internal/match"
)

// Quest rewards.
const (
	QuestsPerDay  = 3
	QuestBonusPay = 150
)

// Quest is a goal for one day. count says how far a game moves it along.
type Quest struct {
	ID, Desc string
	Goal     int
	Coins    int
	XP       int
	count    func(GameResult) int
}

func one(ok bool) int {
	if ok {
		return 1
	}
	return 0
}

var questPool = []Quest{
	{"play_games", "Play 5 games", 5, 40, 40, func(GameResult) int { return 1 }},
	{"win_games", "Win 3 games", 3, 60, 60, func(r GameResult) int { return one(r.Won) }},
	{"open_tiles", "Open 300 tiles", 300, 50, 50, func(r GameResult) int {
		if r.Mode == match.Flags {
			return 0
		}
		return r.Score
	}},
	{"no_flags", "Win a game without flags", 1, 50, 50, func(r GameResult) int { return one(r.Won && r.Mode.Solo() && !r.Flagged) }},
	{"daily", "Finish the Daily Challenge", 1, 60, 60, func(r GameResult) int { return one(r.Daily && r.Won) }},
	{"versus", "Play 2 matches against anyone", 2, 50, 50, func(r GameResult) int { return one(!r.Mode.Solo()) }},
	{"blitz_boards", "Clear 3 boards in Blitz", 3, 70, 70, func(r GameResult) int { return r.Boards }},
	{"survive", "Win a Survival run", 1, 90, 90, func(r GameResult) int { return one(r.Won && r.Mode == match.Survival) }},
	{"stand", "Win a Last Stand", 1, 70, 70, func(r GameResult) int { return one(r.Won && r.Mode == match.LastStand) }},
	{"big_board", "Win Intermediate or Expert", 1, 60, 60, func(r GameResult) int {
		return one(r.Won && (r.Board == "intermediate" || r.Board == "expert"))
	}},
	{"speedy", "Win Beginner in under 30 seconds", 1, 50, 50, func(r GameResult) int {
		return one(r.Won && r.Board == "beginner" && r.ElapsedMs < 30000)
	}},
}

// DailyQuests picks the day's quests. Everyone gets the same ones on a date.
func DailyQuests(date string) []Quest {
	h := fnv.New64a()
	h.Write([]byte("mine party quests " + date))
	rng := rand.New(rand.NewPCG(h.Sum64(), 0x717565))
	out := make([]Quest, 0, QuestsPerDay)
	for _, i := range rng.Perm(len(questPool))[:QuestsPerDay] {
		out = append(out, questPool[i])
	}
	return out
}

// QuestStatus is a quest and how far the player has got with it.
type QuestStatus struct {
	Quest
	Progress int
}

// Done reports whether the quest is finished.
func (q QuestStatus) Done() bool { return q.Progress >= q.Goal }

// Quests returns the state of a date's quests.
func (p *Profile) Quests(date string) []QuestStatus {
	var out []QuestStatus
	for _, q := range DailyQuests(date) {
		st := QuestStatus{Quest: q}
		if p.QuestDate == date {
			st.Progress = p.QuestProgress[q.ID]
		}
		out = append(out, st)
	}
	return out
}

func (p *Profile) questsDone(date string) bool {
	for _, q := range p.Quests(date) {
		if !q.Done() {
			return false
		}
	}
	return true
}

func (p *Profile) advanceQuests(r GameResult, rw *Rewards, pay func(int)) {
	if p.QuestDate != r.Date {
		p.QuestDate = r.Date
		p.QuestProgress = map[string]int{}
	}
	finished := false
	for _, q := range DailyQuests(r.Date) {
		have := p.QuestProgress[q.ID]
		if have >= q.Goal {
			continue
		}
		now := min(have+q.count(r), q.Goal)
		p.QuestProgress[q.ID] = now
		if now == q.Goal {
			finished = true
			rw.Quests = append(rw.Quests, q)
			rw.XP += q.XP
			pay(q.Coins)
		}
	}
	if finished && p.questsDone(r.Date) {
		rw.QuestBonus = true
		pay(QuestBonusPay)
	}
}

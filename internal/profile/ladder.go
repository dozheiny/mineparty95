package profile

import (
	"fmt"

	"mineparty/internal/match"
)

// Tier is a step of the ladder. Reaching it for the first time pays Coins.
type Tier struct {
	Name  string
	MinRP int
	Coins int
}

// Tiers lists the ladder from the bottom.
var Tiers = []Tier{
	{"Bronze", 0, 0},
	{"Silver", 200, 100},
	{"Gold", 600, 250},
	{"Platinum", 1200, 500},
	{"Diamond", 2200, 900},
	{"Master", 3600, 1500},
}

const divisions = 3

// Rank is a place on the ladder. Each tier but the last has three divisions,
// counting down from III. Need is 0 at the top.
type Rank struct {
	Tier     int
	Division int
	Name     string
	Into     int // RP earned within the division
	Need     int // RP the division spans
}

func tierOf(rp int) int {
	tier := 0
	for i, t := range Tiers {
		if rp >= t.MinRP {
			tier = i
		}
	}
	return tier
}

// RankOf places a score on the ladder.
func RankOf(rp int) Rank {
	tier := tierOf(rp)
	t := Tiers[tier]
	if tier == len(Tiers)-1 {
		return Rank{Tier: tier, Name: t.Name}
	}
	span := (Tiers[tier+1].MinRP - t.MinRP) / divisions
	step := min((rp-t.MinRP)/span, divisions-1)
	need := span
	if step == divisions-1 {
		need = Tiers[tier+1].MinRP - t.MinRP - span*step
	}
	return Rank{
		Tier:     tier,
		Division: divisions - step,
		Name:     fmt.Sprintf("%s %s", t.Name, []string{"I", "II", "III"}[divisions-step-1]),
		Into:     rp - t.MinRP - span*step,
		Need:     need,
	}
}

// climb moves the player on the ladder and returns the change. Losing can
// never drop a player out of their current tier.
func (p *Profile) climb(r GameResult) int {
	floor := Tiers[tierOf(p.RP)].MinRP
	before := p.RP
	p.RP = max(p.RP+rankChange(r, p.WinStreak), floor)
	return p.RP - before
}

func rankChange(r GameResult, streak int) int {
	switch {
	case r.Draw:
		return 0
	case r.Won:
		return rankGain(r) + 2*min(streak, 5)
	case len(r.Rivals) > 0:
		return -12
	case r.Board == "beginner":
		return -2
	}
	return -5
}

func rankGain(r GameResult) int {
	switch {
	case r.Daily:
		return 30
	case r.Mode == match.Blitz:
		return 5 + r.Boards*5
	case r.Mode == match.Survival:
		return 30
	case r.Mode.Solo():
		switch r.Board {
		case "intermediate":
			return 15
		case "expert":
			return 30
		}
		return 8
	case len(r.Rivals) > 0:
		return 30
	}
	return 12
}

// promote records any tiers reached for the first time.
func (p *Profile) promote() []Tier {
	var out []Tier
	tier := tierOf(p.RP)
	for p.BestTier < tier {
		p.BestTier++
		out = append(out, Tiers[p.BestTier])
	}
	return out
}

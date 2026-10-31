package profile

import (
	"errors"
	"fmt"
)

// Coin rewards.
const (
	awardCoins = 25
	MaxFreezes = 2
)

// LevelCoins is what reaching a level pays. Every fifth level pays triple.
func LevelCoins(level int) int {
	coins := 40 + 10*level
	if level%5 == 0 {
		coins *= 3
	}
	return coins
}

// gameCoins turns a game's XP into coins, with a bonus for a winning streak.
func gameCoins(xp int, won bool, streak int) int {
	coins := 2 + xp/6
	if won {
		coins += coins * min(streak, 5) / 10
	}
	return coins
}

// luckyMultiplier turns a roll from 0 to 99 into the occasional big payout.
func luckyMultiplier(roll int) int {
	switch {
	case roll >= 98:
		return 5
	case roll >= 88:
		return 2
	}
	return 1
}

// ItemKind is a shelf in the shop.
type ItemKind int

// The kinds of item.
const (
	KindTheme ItemKind = iota
	KindFlag
	KindMine
	KindFace
	KindTitle
	KindFreeze
)

// Item is something in the shop. A price of 0 means it can't be bought: it is
// either a free default or, for a theme, earned by playing.
type Item struct {
	ID, Name string
	Kind     ItemKind
	Price    int
}

// Everything but themes, which are listed in Themes. The first item of each
// kind is the free default.
var items = []Item{
	{"flag_classic", "Classic Flag", KindFlag, 0},
	{"flag_star", "Star", KindFlag, 150},
	{"flag_heart", "Heart", KindFlag, 200},
	{"flag_pirate", "Pirate Flag", KindFlag, 300},
	{"flag_crown", "Crown", KindFlag, 450},
	{"mine_classic", "Classic Mine", KindMine, 0},
	{"mine_skull", "Skull", KindMine, 250},
	{"mine_bug", "Bug", KindMine, 300},
	{"mine_ghost", "Ghost", KindMine, 400},
	{"face_yellow", "Sunny", KindFace, 0},
	{"face_mint", "Mint", KindFace, 150},
	{"face_rose", "Rose", KindFace, 150},
	{"face_sky", "Sky", KindFace, 200},
	{"face_gold", "Solid Gold", KindFace, 750},
	{"title_rank", "Earned rank", KindTitle, 0},
	{"title_sweeper", "Street Sweeper", KindTitle, 200},
	{"title_lucky", "Lucky Guesser", KindTitle, 300},
	{"title_whisper", "Bomb Whisperer", KindTitle, 500},
	{"title_king", "Mine King", KindTitle, 1000},
	{"title_95", "Legend of '95", KindTitle, 2000},
	{"freeze", "Streak Freeze", KindFreeze, 200},
}

// Catalog lists one shelf of the shop.
func Catalog(kind ItemKind) []Item {
	var out []Item
	if kind == KindTheme {
		for _, t := range Themes {
			out = append(out, Item{t.ID, t.Name, KindTheme, t.Price})
		}
		return out
	}
	for _, it := range items {
		if it.Kind == kind {
			out = append(out, it)
		}
	}
	return out
}

// ItemByID finds an item on any shelf.
func ItemByID(id string) (Item, bool) {
	for _, kind := range []ItemKind{KindTheme, KindFlag, KindMine, KindFace, KindTitle, KindFreeze} {
		for _, it := range Catalog(kind) {
			if it.ID == id {
				return it, true
			}
		}
	}
	return Item{}, false
}

// Owns reports whether the player can use an item.
func (p *Profile) Owns(id string) bool {
	it, ok := ItemByID(id)
	switch {
	case !ok:
		return false
	case it.Kind == KindTheme:
		return p.ThemeUnlocked(id)
	}
	return it.Price == 0 || p.Owned[id]
}

// Buy spends coins on an item. A refused purchase changes nothing, and its
// error says why in words the player can read.
func (p *Profile) Buy(id string) error {
	it, ok := ItemByID(id)
	switch {
	case !ok || it.Price == 0:
		return errors.New("that isn't for sale")
	case it.Kind == KindFreeze && p.Freezes >= MaxFreezes:
		return fmt.Errorf("you can only hold %d freezes", MaxFreezes)
	case it.Kind != KindFreeze && p.Owned[id]:
		return errors.New("you already own that")
	case p.Coins < it.Price:
		return fmt.Errorf("you need %d more coins", it.Price-p.Coins)
	}
	p.Coins -= it.Price
	if it.Kind == KindFreeze {
		p.Freezes++
		return nil
	}
	p.Owned[id] = true
	return nil
}

// Equip puts an owned item to use.
func (p *Profile) Equip(id string) error {
	if !p.Owns(id) {
		return errors.New("you don't own that yet")
	}
	it, _ := ItemByID(id)
	switch it.Kind {
	case KindTheme:
		p.Theme = id
	case KindFlag:
		p.FlagSkin = id
	case KindMine:
		p.MineSkin = id
	case KindFace:
		p.FaceSkin = id
	case KindTitle:
		p.TitleSkin = id
	}
	return nil
}

// Equipped returns the item in use on a shelf.
func (p *Profile) Equipped(kind ItemKind) string {
	var id string
	switch kind {
	case KindTheme:
		return p.Theme
	case KindFlag:
		id = p.FlagSkin
	case KindMine:
		id = p.MineSkin
	case KindFace:
		id = p.FaceSkin
	case KindTitle:
		id = p.TitleSkin
	}
	if id == "" {
		return Catalog(kind)[0].ID
	}
	return id
}

// DisplayTitle is the title shown beside the player's name.
func (p *Profile) DisplayTitle() string {
	if p.TitleSkin != "" && p.TitleSkin != Catalog(KindTitle)[0].ID {
		if it, ok := ItemByID(p.TitleSkin); ok {
			return it.Name
		}
	}
	return Title(p.Level())
}

var loginCoins = [7]int{20, 30, 40, 50, 60, 80, 200}

// LoginCoins lists the seven-day cycle of the login bonus.
func LoginCoins() [7]int { return loginCoins }

// ClaimLogin pays the day's login bonus and reports which day of the cycle it
// was. It pays nothing if today's bonus has been claimed. Missing a day
// restarts the cycle.
func (p *Profile) ClaimLogin(date string) (coins, day int) {
	if p.LastLogin == date {
		return 0, 0
	}
	day = 1
	if p.LastLogin == dayBefore(date) {
		day = p.LoginDay%len(loginCoins) + 1
	}
	p.LoginDay, p.LastLogin = day, date
	coins = loginCoins[day-1]
	p.Coins += coins
	p.CoinsEarned += coins
	return coins, day
}

package ui

import "image/color"

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

// Theme is the palette a skin draws with.
type Theme struct {
	Desk, Face, Hi, Lo, Dark color.RGBA
	Title, TitleText, Text   color.RGBA
	Open, Grid               color.RGBA
	Num                      [9]color.RGBA
	Flag, Mine, Boom         color.RGBA
	LEDBack, LEDOn, LEDOff   color.RGBA
	Accent                   color.RGBA
	Scanlines                bool
}

// Player colours stay the same in every theme, so that a player can always
// be told apart whichever skin each friend picks.
var playerColors = [4]color.RGBA{rgb(50, 110, 240), rgb(235, 80, 40), rgb(30, 165, 80), rgb(190, 60, 205)}

var themes = map[string]*Theme{
	"classic": {
		Desk: rgb(0, 128, 128), Face: rgb(192, 192, 192), Hi: rgb(255, 255, 255), Lo: rgb(128, 128, 128), Dark: rgb(0, 0, 0),
		Title: rgb(0, 0, 128), TitleText: rgb(255, 255, 255), Text: rgb(0, 0, 0),
		Open: rgb(192, 192, 192), Grid: rgb(128, 128, 128),
		Num:  [9]color.RGBA{{}, rgb(0, 0, 255), rgb(0, 128, 0), rgb(255, 0, 0), rgb(0, 0, 128), rgb(128, 0, 0), rgb(0, 128, 128), rgb(0, 0, 0), rgb(96, 96, 96)},
		Flag: rgb(255, 0, 0), Mine: rgb(0, 0, 0), Boom: rgb(255, 0, 0),
		LEDBack: rgb(0, 0, 0), LEDOn: rgb(255, 0, 0), LEDOff: rgb(70, 0, 0), Accent: rgb(0, 0, 128),
	},
	"gameboy": {
		Desk: rgb(15, 56, 15), Face: rgb(155, 188, 15), Hi: rgb(205, 225, 90), Lo: rgb(48, 98, 48), Dark: rgb(15, 56, 15),
		Title: rgb(15, 56, 15), TitleText: rgb(155, 188, 15), Text: rgb(15, 56, 15),
		Open: rgb(139, 172, 15), Grid: rgb(48, 98, 48),
		Num:  [9]color.RGBA{{}, rgb(15, 56, 15), rgb(15, 56, 15), rgb(15, 56, 15), rgb(15, 56, 15), rgb(15, 56, 15), rgb(15, 56, 15), rgb(15, 56, 15), rgb(15, 56, 15)},
		Flag: rgb(15, 56, 15), Mine: rgb(15, 56, 15), Boom: rgb(48, 98, 48),
		LEDBack: rgb(15, 56, 15), LEDOn: rgb(155, 188, 15), LEDOff: rgb(48, 98, 48), Accent: rgb(48, 98, 48),
	},
	"amber": {
		Desk: rgb(18, 10, 0), Face: rgb(44, 26, 4), Hi: rgb(120, 76, 14), Lo: rgb(20, 12, 2), Dark: rgb(0, 0, 0),
		Title: rgb(255, 176, 0), TitleText: rgb(24, 14, 0), Text: rgb(255, 176, 0),
		Open: rgb(26, 15, 2), Grid: rgb(70, 44, 8),
		Num:  [9]color.RGBA{{}, rgb(255, 196, 60), rgb(255, 176, 0), rgb(255, 140, 0), rgb(255, 214, 120), rgb(240, 120, 0), rgb(255, 196, 60), rgb(255, 230, 170), rgb(190, 130, 30)},
		Flag: rgb(255, 214, 120), Mine: rgb(255, 176, 0), Boom: rgb(160, 60, 0),
		LEDBack: rgb(0, 0, 0), LEDOn: rgb(255, 176, 0), LEDOff: rgb(58, 36, 0), Accent: rgb(255, 176, 0), Scanlines: true,
	},
	"arcade": {
		Desk: rgb(10, 4, 34), Face: rgb(32, 18, 80), Hi: rgb(0, 230, 255), Lo: rgb(120, 20, 190), Dark: rgb(4, 0, 18),
		Title: rgb(255, 0, 140), TitleText: rgb(255, 255, 255), Text: rgb(190, 245, 255),
		Open: rgb(16, 8, 48), Grid: rgb(60, 34, 130),
		Num:  [9]color.RGBA{{}, rgb(0, 230, 255), rgb(60, 255, 130), rgb(255, 70, 120), rgb(255, 230, 60), rgb(255, 140, 40), rgb(190, 130, 255), rgb(255, 255, 255), rgb(150, 150, 190)},
		Flag: rgb(255, 0, 140), Mine: rgb(255, 230, 60), Boom: rgb(200, 0, 90),
		LEDBack: rgb(4, 0, 18), LEDOn: rgb(60, 255, 130), LEDOff: rgb(10, 50, 34), Accent: rgb(255, 0, 140), Scanlines: true,
	},
	"vaporwave": {
		Desk: rgb(255, 113, 206), Face: rgb(1, 205, 254), Hi: rgb(185, 240, 255), Lo: rgb(5, 120, 175), Dark: rgb(55, 20, 105),
		Title: rgb(185, 103, 255), TitleText: rgb(255, 251, 150), Text: rgb(55, 20, 105),
		Open: rgb(150, 232, 255), Grid: rgb(5, 150, 200),
		Num:  [9]color.RGBA{{}, rgb(120, 0, 200), rgb(0, 110, 90), rgb(225, 0, 120), rgb(55, 20, 105), rgb(170, 30, 0), rgb(0, 90, 150), rgb(30, 10, 60), rgb(90, 90, 120)},
		Flag: rgb(225, 0, 120), Mine: rgb(55, 20, 105), Boom: rgb(255, 113, 206),
		LEDBack: rgb(55, 20, 105), LEDOn: rgb(255, 251, 150), LEDOff: rgb(90, 50, 130), Accent: rgb(185, 103, 255),
	},
}

func themeFor(id string) *Theme {
	if t, ok := themes[id]; ok {
		return t
	}
	return themes["classic"]
}

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
	"midnight": {
		Desk: rgb(12, 16, 40), Face: rgb(38, 46, 84), Hi: rgb(90, 104, 170), Lo: rgb(20, 24, 54), Dark: rgb(4, 6, 20),
		Title: rgb(70, 100, 230), TitleText: rgb(255, 255, 255), Text: rgb(220, 228, 255),
		Open: rgb(26, 32, 66), Grid: rgb(56, 66, 118),
		Num:  [9]color.RGBA{{}, rgb(120, 170, 255), rgb(110, 225, 170), rgb(255, 120, 130), rgb(190, 160, 255), rgb(255, 180, 90), rgb(110, 220, 230), rgb(240, 240, 255), rgb(160, 170, 200)},
		Flag: rgb(255, 90, 110), Mine: rgb(230, 235, 255), Boom: rgb(170, 40, 70),
		LEDBack: rgb(4, 6, 20), LEDOn: rgb(120, 170, 255), LEDOff: rgb(20, 34, 70), Accent: rgb(140, 180, 255),
	},
	"sakura": {
		Desk: rgb(250, 214, 226), Face: rgb(255, 236, 242), Hi: rgb(255, 255, 255), Lo: rgb(214, 150, 172), Dark: rgb(120, 50, 80),
		Title: rgb(214, 84, 130), TitleText: rgb(255, 255, 255), Text: rgb(90, 30, 56),
		Open: rgb(255, 244, 247), Grid: rgb(230, 190, 205),
		Num:  [9]color.RGBA{{}, rgb(40, 90, 200), rgb(30, 130, 90), rgb(214, 30, 90), rgb(110, 40, 160), rgb(160, 60, 20), rgb(0, 120, 130), rgb(70, 20, 50), rgb(130, 90, 110)},
		Flag: rgb(200, 30, 90), Mine: rgb(90, 30, 56), Boom: rgb(255, 150, 180),
		LEDBack: rgb(90, 30, 56), LEDOn: rgb(255, 180, 205), LEDOff: rgb(120, 60, 85), Accent: rgb(200, 50, 110),
	},
	"glacier": {
		Desk: rgb(190, 225, 240), Face: rgb(225, 242, 250), Hi: rgb(255, 255, 255), Lo: rgb(130, 175, 200), Dark: rgb(30, 70, 100),
		Title: rgb(30, 110, 160), TitleText: rgb(255, 255, 255), Text: rgb(15, 50, 75),
		Open: rgb(238, 248, 252), Grid: rgb(170, 205, 222),
		Num:  [9]color.RGBA{{}, rgb(0, 80, 200), rgb(0, 120, 110), rgb(200, 30, 70), rgb(60, 40, 170), rgb(150, 70, 20), rgb(0, 120, 160), rgb(15, 40, 60), rgb(90, 120, 140)},
		Flag: rgb(0, 100, 180), Mine: rgb(15, 50, 75), Boom: rgb(120, 200, 240),
		LEDBack: rgb(15, 50, 75), LEDOn: rgb(130, 225, 255), LEDOff: rgb(30, 75, 105), Accent: rgb(0, 105, 170),
	},
	"phosphor": {
		Desk: rgb(0, 12, 4), Face: rgb(6, 36, 16), Hi: rgb(30, 130, 60), Lo: rgb(2, 20, 8), Dark: rgb(0, 0, 0),
		Title: rgb(50, 255, 120), TitleText: rgb(0, 24, 8), Text: rgb(70, 255, 130),
		Open: rgb(3, 24, 10), Grid: rgb(14, 70, 34),
		Num:  [9]color.RGBA{{}, rgb(120, 255, 160), rgb(70, 255, 130), rgb(200, 255, 120), rgb(160, 255, 200), rgb(230, 255, 150), rgb(90, 230, 150), rgb(220, 255, 230), rgb(60, 170, 100)},
		Flag: rgb(190, 255, 190), Mine: rgb(70, 255, 130), Boom: rgb(0, 120, 50),
		LEDBack: rgb(0, 0, 0), LEDOn: rgb(70, 255, 130), LEDOff: rgb(4, 44, 20), Accent: rgb(50, 255, 120), Scanlines: true,
	},
}

func themeFor(id string) *Theme {
	if t, ok := themes[id]; ok {
		return t
	}
	return themes["classic"]
}

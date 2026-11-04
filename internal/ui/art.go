package ui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// All of the game's art is drawn here at start-up, from original bitmaps and
// the active theme, so that every skin recolours everything automatically.

// Smiley expressions.
const (
	faceHappy = iota
	faceScared
	faceCool
	faceDead
	faceCount
)

// Art is the sprite set for one theme.
type Art struct {
	Hidden, Open, Boom *ebiten.Image
	Num                [9]*ebiten.Image
	Flag, Mine, Wrong  *ebiten.Image
	Question           *ebiten.Image
	PlayerFlag         [4]*ebiten.Image
	Face               [faceCount]*ebiten.Image
	LED                [11]*ebiten.Image // 0-9, then a minus sign
	Scan               *ebiten.Image
}

var digitBitmaps = [9][]string{
	1: {"...##...", "..###...", ".####...", "...##...", "...##...", "...##...", "...##...", "...##...", ".######."},
	2: {".######.", "##....##", "......##", ".....##.", "...##...", ".##.....", "##......", "##......", "########"},
	3: {".######.", "##....##", "......##", "......##", "..#####.", "......##", "......##", "##....##", ".######."},
	4: {"....###.", "...####.", "..##.##.", ".##..##.", "##...##.", "########", ".....##.", ".....##.", ".....##."},
	5: {"########", "##......", "##......", "#######.", "......##", "......##", "......##", "##....##", ".######."},
	6: {"..#####.", ".##.....", "##......", "##......", "#######.", "##....##", "##....##", "##....##", ".######."},
	7: {"########", "......##", ".....##.", ".....##.", "....##..", "....##..", "...##...", "...##...", "...##..."},
	8: {".######.", "##....##", "##....##", "##....##", ".######.", "##....##", "##....##", "##....##", ".######."},
}

var questionBitmap = []string{".######.", "##....##", "......##", ".....##.", "...###..", "...##...", "........", "...##...", "...##..."}

var flagBitmap = []string{
	"............",
	".....RRK....",
	"...RRRRK....",
	".RRRRRRK....",
	"...RRRRK....",
	".....RRK....",
	".......K....",
	".......K....",
	".....KKKK...",
	"...KKKKKKKK.",
}

var mineBitmap = []string{
	".....KK.....",
	".K..KKKK..K.",
	"..KKKKKKKK..",
	"..KWWKKKKK..",
	".KKWWKKKKKK.",
	"KKKKKKKKKKKK",
	"KKKKKKKKKKKK",
	".KKKKKKKKKK.",
	"..KKKKKKKK..",
	"..KKKKKKKK..",
	".K..KKKK..K.",
	".....KK.....",
}

var crossBitmap = []string{
	"X..........X",
	".X........X.",
	"..X......X..",
	"...X....X...",
	"....X..X....",
	".....XX.....",
	".....XX.....",
	"....X..X....",
	"...X....X...",
	"..X......X..",
	".X........X.",
	"X..........X",
}

// Skins sold in the shop, by item ID.
var flagSkins = map[string][]string{
	"flag_classic": flagBitmap,
	"flag_star": {
		".....RR.....",
		".....RR.....",
		"....RRRR....",
		"RRRRRRRRRRRR",
		".RRRRRRRRRR.",
		"...RRRRRR...",
		"...RRRRRR...",
		"..RRR..RRR..",
		"..RR....RR..",
		"............",
	},
	"flag_heart": {
		"..RRR..RRR..",
		".RRRRRRRRRR.",
		".RRRRRRRRRR.",
		".RRRRRRRRRR.",
		"..RRRRRRRR..",
		"...RRRRRR...",
		"....RRRR....",
		".....RR.....",
		"............",
		"............",
	},
	"flag_pirate": {
		".KKKKKKKKK.K",
		".KKRKKRKKK.K",
		".KKKKKKKKK.K",
		".KKKRRKKKK.K",
		".KKRKKRKKK.K",
		".KKKKKKKKK.K",
		"...........K",
		"...........K",
		".........KKK",
		"........KKKK",
	},
	"flag_crown": {
		"............",
		"R....RR....R",
		"RR...RR...RR",
		"RR..RRRR..RR",
		"RRR.RRRR.RRR",
		"RRRRRRRRRRRR",
		"RRRRRRRRRRRR",
		"RRRRRRRRRRRR",
		"KKKKKKKKKKKK",
		"............",
	},
}

var mineSkins = map[string][]string{
	"mine_classic": mineBitmap,
	"mine_skull": {
		"...KKKKKK...",
		"..KKKKKKKK..",
		".KKKKKKKKKK.",
		".KWWKKKKWWK.",
		".KWWKKKKWWK.",
		".KKKKKKKKKK.",
		"..KKKWWKKK..",
		"...KKKKKK...",
		"...KWKKWK...",
		"...KKKKKK...",
		"...KWKKWK...",
		"............",
	},
	"mine_bug": {
		"K..........K",
		".K..KKKK..K.",
		"..KKKKKKKK..",
		"K.KWKKKKWK.K",
		".KKKKKKKKKK.",
		"..KKKKKKKK..",
		"K.KKKKKKKK.K",
		".KKKKKKKKKK.",
		"..KKKKKKKK..",
		"K.KKKKKKKK.K",
		".K..KKKK..K.",
		"K..........K",
	},
	"mine_ghost": {
		"....KKKK....",
		"..KKKKKKKK..",
		".KKKKKKKKKK.",
		".KWWKKKWWKK.",
		".KWWKKKWWKK.",
		"KKKKKKKKKKKK",
		"KKKKKKKKKKKK",
		"KKKKKKKKKKKK",
		"KKKKKKKKKKKK",
		"KKKKKKKKKKKK",
		"KK.KKK.KKK.K",
		"K...K...K..K",
	},
}

var faceColors = map[string]color.RGBA{
	"face_yellow": rgb(255, 225, 0),
	"face_mint":   rgb(110, 235, 170),
	"face_rose":   rgb(255, 140, 170),
	"face_sky":    rgb(120, 200, 255),
	"face_gold":   rgb(240, 180, 30),
}

func stamp(img *image.RGBA, x, y int, rows []string, ink map[byte]color.RGBA) {
	for dy, row := range rows {
		for dx := 0; dx < len(row); dx++ {
			if c, ok := ink[row[dx]]; ok {
				img.SetRGBA(x+dx, y+dy, c)
			}
		}
	}
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			img.SetRGBA(i, j, c)
		}
	}
}

func bevelTile(t *Theme) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	fillRect(img, 0, 0, 16, 16, t.Face)
	fillRect(img, 0, 0, 15, 2, t.Hi)
	fillRect(img, 0, 0, 2, 15, t.Hi)
	fillRect(img, 1, 14, 15, 2, t.Lo)
	fillRect(img, 14, 1, 2, 15, t.Lo)
	return img
}

func openTile(t *Theme, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	fillRect(img, 0, 0, 16, 16, c)
	fillRect(img, 0, 0, 16, 1, t.Grid)
	fillRect(img, 0, 0, 1, 16, t.Grid)
	return img
}

func glyph(rows []string, x, y int, ink map[byte]color.RGBA) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	stamp(img, x, y, rows, ink)
	return ebiten.NewImageFromImage(img)
}

func flagInk(t *Theme, c color.RGBA) map[byte]color.RGBA {
	return map[byte]color.RGBA{'R': c, 'K': t.Dark}
}

func mineInk(t *Theme) map[byte]color.RGBA {
	return map[byte]color.RGBA{'K': t.Mine, 'W': t.Hi}
}

func newArt(t *Theme, flag, mine, face string) *Art {
	a := &Art{
		Hidden: ebiten.NewImageFromImage(bevelTile(t)),
		Open:   ebiten.NewImageFromImage(openTile(t, t.Open)),
		Boom:   ebiten.NewImageFromImage(openTile(t, t.Boom)),
	}
	for n := 1; n <= 8; n++ {
		a.Num[n] = glyph(digitBitmaps[n], 4, 4, map[byte]color.RGBA{'#': t.Num[n]})
	}
	a.Flag = glyph(flagSkins[flag], 2, 3, flagInk(t, t.Flag))
	a.Question = glyph(questionBitmap, 4, 4, map[byte]color.RGBA{'#': t.Dark})
	for i, c := range playerColors {
		a.PlayerFlag[i] = glyph(flagBitmap, 2, 3, flagInk(t, c))
	}
	a.Mine = glyph(mineSkins[mine], 2, 2, mineInk(t))
	a.Wrong = glyph(crossBitmap, 2, 2, map[byte]color.RGBA{'X': t.Boom})
	for i := range a.Face {
		a.Face[i] = ebiten.NewImageFromImage(drawFace(i, faceColors[face]))
	}
	for d := range a.LED {
		a.LED[d] = ebiten.NewImageFromImage(drawLED(t, d))
	}
	if t.Scanlines {
		a.Scan = scanlines()
	}
	return a
}

func drawFace(kind int, skin color.RGBA) *image.RGBA {
	const size = 20
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	ink := rgb(0, 0, 0)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			d := math.Hypot(float64(x)-9.5, float64(y)-9.5)
			switch {
			case d <= 8.2:
				img.SetRGBA(x, y, skin)
			case d <= 9.4:
				img.SetRGBA(x, y, ink)
			}
		}
	}
	var rows []string
	switch kind {
	case faceHappy:
		rows = []string{
			"...##....##...",
			"...##....##...",
			"..............",
			"..............",
			".#..........#.",
			"..#........#..",
			"...##....##...",
			".....####.....",
		}
	case faceScared:
		rows = []string{
			"...##....##...",
			"...##....##...",
			"..............",
			"..............",
			"......##......",
			".....#..#.....",
			".....#..#.....",
			"......##......",
		}
	case faceCool:
		rows = []string{
			".############.",
			"#.####..####.#",
			"..####..####..",
			"...##....##...",
			".#..........#.",
			"..#........#..",
			"...##....##...",
			".....####.....",
		}
	case faceDead:
		rows = []string{
			"..#.#....#.#..",
			"...#......#...",
			"..#.#....#.#..",
			"..............",
			".....####.....",
			"...##....##...",
			"..#........#..",
			"..............",
		}
	}
	stamp(img, 3, 6, rows, map[byte]color.RGBA{'#': ink})
	return img
}

// drawLED paints one seven-segment digit, unlit segments included.
func drawLED(t *Theme, digit int) *image.RGBA {
	const w, h = 13, 23
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fillRect(img, 0, 0, w, h, t.LEDBack)
	// Segments a to g: top, upper right, lower right, bottom, lower left,
	// upper left, middle.
	segments := [7][4]int{{2, 1, 9, 2}, {10, 2, 2, 9}, {10, 12, 2, 9}, {2, 20, 9, 2}, {1, 12, 2, 9}, {1, 2, 2, 9}, {2, 10, 9, 2}}
	lit := [11]string{"abcdef", "bc", "abdeg", "abcdg", "bcfg", "acdfg", "acdefg", "abc", "abcdefg", "abcdfg", "g"}[digit]
	for s, r := range segments {
		c := t.LEDOff
		for _, l := range lit {
			if int(l-'a') == s {
				c = t.LEDOn
			}
		}
		fillRect(img, r[0], r[1], r[2], r[3], c)
	}
	return img
}

func scanlines() *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, ScreenW, ScreenH))
	shade := color.RGBA{0, 0, 0, 56}
	for y := 0; y < ScreenH; y += 2 {
		fillRect(img, 0, y, ScreenW, 1, shade)
	}
	return ebiten.NewImageFromImage(img)
}

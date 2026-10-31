package ui

import (
	"bufio"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/match"
)

// A script feeds synthetic input through the real game loop and takes
// screenshots, so that whole UI flows can be exercised on a real window.
//
//	wait 30          let 30 frames pass
//	move 100 200     move the pointer
//	click 100 200    left click
//	rclick 100 200   right click
//	key Enter        press a key
//	type hello       type text
//	autoplay         let a CPU play the local seat, through the real screen
//	shot out.png     save a screenshot
//	quit             exit
type script struct {
	lines []string
	wait  int
	x, y  int
	path  string
	phase int // frames left of a click in progress
	right bool
	bot   *match.Bot
}

func loadScript(path string) (*script, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := &script{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" && !strings.HasPrefix(line, "#") {
			sc.lines = append(sc.lines, line)
		}
	}
	return sc, scanner.Err()
}

func (sc *script) capture(screen *ebiten.Image) {
	if sc.path == "" {
		return
	}
	path := sc.path
	sc.path = ""
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, img)
}

func (sc *script) autoplay(g *Game) {
	p, ok := g.scene.(*play)
	if !ok || p.v == nil || g.frame%4 != 0 {
		return
	}
	sc.bot.SetPlayer(p.v.You)
	if a, ok := sc.bot.Next(*p.v); ok {
		p.lastTile = a.Tile
		p.sess.Send(a)
	}
}

// step applies the next instruction and reports when the script has quit.
func (sc *script) step(g *Game, in *Input) bool {
	if sc.bot != nil {
		sc.autoplay(g)
	}
	in.X, in.Y = sc.x, sc.y
	switch sc.phase {
	case 2:
		in.LeftDown = !sc.right
		in.LeftPress = !sc.right
		in.RightDown = sc.right
		in.RightPress = sc.right
		sc.phase = 1
		return false
	case 1:
		in.LeftRelease = !sc.right
		sc.phase = 0
		return false
	}
	if sc.wait > 0 {
		sc.wait--
		return false
	}
	if len(sc.lines) == 0 {
		return false
	}
	fields := strings.SplitN(sc.lines[0], " ", 2)
	sc.lines = sc.lines[1:]
	arg := ""
	if len(fields) > 1 {
		arg = fields[1]
	}
	switch fields[0] {
	case "wait":
		sc.wait, _ = strconv.Atoi(arg)
	case "move", "click", "rclick":
		fmt.Sscanf(arg, "%d %d", &sc.x, &sc.y)
		in.X, in.Y = sc.x, sc.y
		if fields[0] != "move" {
			sc.phase = 2
			sc.right = fields[0] == "rclick"
		}
	case "key":
		var k ebiten.Key
		if err := k.UnmarshalText([]byte(arg)); err == nil {
			in.Keys = append(in.Keys, k)
		}
	case "type":
		in.Chars = append(in.Chars, []rune(arg)...)
	case "autoplay":
		sc.bot = match.NewBot(0, 1)
	case "shot":
		sc.path = arg
	case "quit":
		return true
	}
	return false
}

package ui

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/netplay"
)

const maxListedGames = 4

type joinResult struct {
	client *netplay.Client
	err    error
}

type join struct {
	field  TextField
	games  atomic.Pointer[[]netplay.LANGame]
	stop   chan struct{}
	result chan joinResult
	busy   bool
	status string
	bs     []*Button
	listed []*Button
	go2    *Button
}

func newJoin(g *Game) *join {
	j := &join{
		field:  TextField{X: 28, Y: 284, W: 440, Max: 24},
		stop:   make(chan struct{}),
		result: make(chan joinResult, 1),
	}
	back := func() {
		close(j.stop)
		g.goTo(newMenu(g))
	}
	j.go2 = &Button{X: 476, Y: 282, W: 136, H: 24, Label: "Join", On: func() { j.connect(g, j.field.Value) }}
	j.bs = []*Button{closeButton(back), j.go2, {X: 24, Y: 430, W: 120, H: 30, Label: "Back", On: back}}
	for i := range maxListedGames {
		b := &Button{X: 28, Y: 76 + i*36, W: 584, H: 30, Disabled: true}
		b.On = func() {
			if games := j.games.Load(); games != nil && i < len(*games) {
				j.connect(g, (*games)[i].Code)
			}
		}
		j.listed = append(j.listed, b)
		j.bs = append(j.bs, b)
	}
	go j.search()
	return j
}

func (j *join) search() {
	for {
		games := netplay.Discover(netplay.DiscoveryPort, 700*time.Millisecond)
		j.games.Store(&games)
		select {
		case <-j.stop:
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (j *join) connect(g *Game, code string) {
	if j.busy {
		return
	}
	j.busy = true
	j.status = "Connecting..."
	name := g.Profile.Name
	go func() {
		c, err := netplay.Join(code, name)
		j.result <- joinResult{c, err}
	}()
}

func (j *join) Update(g *Game, in *Input) {
	select {
	case r := <-j.result:
		j.busy = false
		if r.err != nil {
			j.status = r.err.Error()
			break
		}
		close(j.stop)
		g.session = r.client
		g.goTo(newLobby(g, r.client, nil))
		return
	default:
	}
	var games []netplay.LANGame
	if p := j.games.Load(); p != nil {
		games = *p
	}
	for i, b := range j.listed {
		b.Disabled = i >= len(games)
		b.Label = ""
		if i < len(games) {
			b.Label = fmt.Sprintf("%s's room  (%d/%d players)", games[i].Name, games[i].Players, netplay.MaxPlayers)
		}
	}
	j.go2.Disabled = j.busy || j.field.Value == ""
	j.go2.Label = "Join"
	j.field.update(in)
	updateButtons(g, in, j.bs)
	if in.Key(ebiten.KeyEnter) && !j.go2.Disabled {
		j.connect(g, j.field.Value)
	}
}

func (j *join) Draw(g *Game, s *ebiten.Image) {
	t := g.theme
	g.window(s, "Join Game")
	g.text(s, "Games on your network", 28, 40, t.Text)
	if p := j.games.Load(); p == nil || len(*p) == 0 {
		g.text(s, "Searching...", 28, 58, t.Text)
	}
	g.text(s, "Or enter a room code (Ctrl+V pastes)", 28, 264, t.Text)
	j.field.draw(g, s)
	if j.status != "" {
		for i, line := range wrap(j.status, 80) {
			g.text(s, line, 28, 320+i*15, t.Accent)
		}
	}
	for _, b := range j.bs {
		if !b.Disabled || b.Label != "" {
			b.draw(g, s, &g.in)
		}
	}
}

package ui

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

type particle struct {
	x, y, vx, vy float64
	life         int
	c            color.Color
}

func (g *Game) burst(x, y float64, n int, speed float64, palette ...color.Color) {
	for range n {
		a := rand.Float64() * 2 * math.Pi
		v := speed * (0.3 + rand.Float64())
		g.parts = append(g.parts, particle{
			x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a) * v,
			life: 30 + rand.IntN(40), c: palette[rand.IntN(len(palette))],
		})
	}
}

func (g *Game) confetti() {
	colors := []color.Color{playerColors[0], playerColors[1], playerColors[2], playerColors[3], rgb(255, 225, 0)}
	for range 140 {
		g.parts = append(g.parts, particle{
			x: rand.Float64() * ScreenW, y: -rand.Float64() * 160,
			vx: rand.Float64() - .5, vy: 1 + rand.Float64()*2,
			life: 150 + rand.IntN(70), c: colors[rand.IntN(len(colors))],
		})
	}
}

func (g *Game) updateParticles() {
	alive := g.parts[:0]
	for _, p := range g.parts {
		p.x += p.vx
		p.y += p.vy
		p.vy += .06
		if p.life--; p.life > 0 && p.y < ScreenH {
			alive = append(alive, p)
		}
	}
	g.parts = alive
}

func (g *Game) drawParticles(s *ebiten.Image) {
	for _, p := range g.parts {
		g.fill(s, int(p.x), int(p.y), 3, 3, p.c)
	}
}

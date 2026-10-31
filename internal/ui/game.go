// Package ui is the game's window: screens, drawing, sound and input.
package ui

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"

	"mineparty/internal/netplay"
	"mineparty/internal/profile"
)

// The virtual screen. Everything is drawn at VGA resolution and scaled up
// with nearest-neighbour filtering to keep the pixels crisp.
const (
	ScreenW = 640
	ScreenH = 480
)

// Scene is one screen of the game.
type Scene interface {
	Update(g *Game, in *Input)
	Draw(g *Game, s *ebiten.Image)
}

// Options configures the game.
type Options struct {
	ProfilePath string
	Silent      bool
	ScriptPath  string
	NoRouter    bool // never ask the router to open a port
}

// Game implements ebiten.Game and holds everything shared between screens.
type Game struct {
	Profile *profile.Profile
	path    string
	opts    Options
	theme   *Theme
	art     *Art
	sfx     *Audio
	font    *text.GoXFace
	pixel   *ebiten.Image
	canvas  *ebiten.Image
	scene   Scene
	session netplay.Session
	in      Input
	script  *script
	quit    bool
	frame   int
	shake   int
	parts   []particle
	notice  string // shown once on the menu
	// lastRound stops a stale result being counted twice.
	lastSession netplay.Session
	lastRound   int
}

// New creates the game at the main menu.
func New(opts Options) (*Game, error) {
	path := opts.ProfilePath
	if path == "" {
		var err error
		if path, err = profile.DefaultPath(); err != nil {
			return nil, err
		}
	}
	prof, err := profile.Load(path)
	g := &Game{
		Profile: prof,
		path:    path,
		opts:    opts,
		font:    text.NewGoXFace(basicfont.Face7x13),
		pixel:   ebiten.NewImage(1, 1),
		canvas:  ebiten.NewImage(ScreenW, ScreenH),
		sfx:     newAudio(!opts.Silent),
	}
	if err != nil {
		g.notice = "Your save couldn't be read, so a new one was started."
		log.Print(err)
	}
	g.pixel.Fill(color.White)
	g.sfx.setVolume(prof.Volume)
	g.applyTheme(prof.Theme)
	ebiten.SetFullscreen(prof.Fullscreen)
	if opts.ScriptPath != "" {
		if g.script, err = loadScript(opts.ScriptPath); err != nil {
			return nil, err
		}
	}
	g.scene = newMenu(g)
	return g, nil
}

func (g *Game) applyTheme(id string) {
	g.theme = themeFor(id)
	g.art = newArt(g.theme)
}

// save writes the profile, logging rather than interrupting play on failure.
func (g *Game) save() {
	if err := g.Profile.Save(g.path); err != nil {
		log.Printf("saving profile: %v", err)
	}
}

// goTo switches screens.
func (g *Game) goTo(s Scene) { g.scene = s }

// leaveSession closes any active game connection.
func (g *Game) leaveSession() {
	if g.session != nil {
		g.session.Close()
		g.session = nil
	}
}

// Shutdown closes the network session after the window has gone.
func (g *Game) Shutdown() { g.leaveSession() }

// Update implements ebiten.Game.
func (g *Game) Update() error {
	g.frame++
	g.in.capture()
	if g.script != nil {
		if g.script.step(g, &g.in) {
			g.quit = true
		}
	}
	if g.in.Key(ebiten.KeyF11) {
		g.Profile.Fullscreen = !ebiten.IsFullscreen()
		ebiten.SetFullscreen(g.Profile.Fullscreen)
		g.save()
	}
	g.scene.Update(g, &g.in)
	g.updateParticles()
	if g.shake > 0 {
		g.shake--
	}
	if g.quit {
		return ebiten.Termination
	}
	return nil
}

// Draw implements ebiten.Game.
func (g *Game) Draw(screen *ebiten.Image) {
	g.canvas.Clear()
	g.scene.Draw(g, g.canvas)
	g.drawParticles(g.canvas)
	screen.Fill(g.theme.Dark)
	op := &ebiten.DrawImageOptions{}
	if g.shake > 0 {
		dx := float64((g.frame*7)%(g.shake+3)) - float64(g.shake+3)/2
		dy := float64((g.frame*11)%(g.shake+3)) - float64(g.shake+3)/2
		op.GeoM.Translate(dx, dy)
	}
	screen.DrawImage(g.canvas, op)
	if g.art.Scan != nil {
		screen.DrawImage(g.art.Scan, nil)
	}
	if g.script != nil {
		g.script.capture(screen)
	}
}

// DrawFinalScreen keeps pixels square and sharp at any window size.
func (g *Game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	screen.Fill(color.Black)
	screen.DrawImage(offscreen, &ebiten.DrawImageOptions{GeoM: geoM, Filter: ebiten.FilterNearest})
}

// Layout implements ebiten.Game.
func (g *Game) Layout(int, int) (int, int) { return ScreenW, ScreenH }

// Input is one frame of player input.
type Input struct {
	X, Y                   int
	LeftDown, RightDown    bool
	LeftPress, LeftRelease bool
	RightPress             bool
	MiddlePress            bool
	Keys                   []ebiten.Key
	Chars                  []rune
	Ctrl                   bool
}

func (in *Input) capture() {
	in.X, in.Y = ebiten.CursorPosition()
	in.LeftDown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	in.RightDown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	in.LeftPress = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	in.LeftRelease = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	in.RightPress = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	in.MiddlePress = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle)
	in.Keys = inpututil.AppendJustPressedKeys(in.Keys[:0])
	in.Chars = ebiten.AppendInputChars(in.Chars[:0])
	in.Ctrl = ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
}

// Key reports whether a key was pressed this frame.
func (in *Input) Key(k ebiten.Key) bool {
	for _, have := range in.Keys {
		if have == k {
			return true
		}
	}
	return false
}

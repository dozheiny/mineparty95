// Command mineparty is a retro Minesweeper with direct online multiplayer.
package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"mineparty/internal/ui"
)

func main() {
	profile := flag.String("profile", "", "use this save file instead of the default")
	silent := flag.Bool("silent", false, "run without sound")
	script := flag.String("script", "", "drive the game from a UI test script")
	lan := flag.Bool("lan", false, "host for the local network only, without asking the router to open a port")
	flag.Parse()

	game, err := ui.New(ui.Options{ProfilePath: *profile, Silent: *silent, ScriptPath: *script, NoRouter: *lan})
	if err != nil {
		log.Fatal(err)
	}
	w, h := ebiten.Monitor().Size()
	scale := max(1, min((w-40)/ui.ScreenW, (h-100)/ui.ScreenH))
	ebiten.SetWindowSize(ui.ScreenW*scale, ui.ScreenH*scale)
	ebiten.SetWindowTitle("Mine Party 95")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)

	err = ebiten.RunGame(game)
	game.Shutdown()
	if err == nil {
		return
	}
	// A missing sound device is fatal to the game loop, and a game can only
	// be run once per process, so start over without sound.
	if !*silent && strings.Contains(err.Error(), "audio") {
		restartSilently()
		return
	}
	log.Fatal(err)
}

func restartSilently() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command(exe, append(os.Args[1:], "-silent")...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

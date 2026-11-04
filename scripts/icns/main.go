// Command icns writes the app icon as a macOS .icns file, for the build
// script to put in the app bundle.
package main

import (
	"log"
	"os"

	"mineparty/internal/ui"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: icns out.icns")
	}
	data, err := ui.ICNS()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(os.Args[1], data, 0o644); err != nil {
		log.Fatal(err)
	}
}

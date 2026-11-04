package ui

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
)

// iconArt is the app icon at its native 16x16: a classic mine on a raised
// tile.
func iconArt() *image.RGBA {
	t := themes["classic"]
	img := bevelTile(t)
	stamp(img, 2, 2, mineBitmap, mineInk(t))
	return img
}

// Icon returns the app icon scaled up by whole pixels to size, which should
// be a multiple of 16.
func Icon(size int) image.Image {
	src := iconArt()
	k := max(size/16, 1)
	out := image.NewRGBA(image.Rect(0, 0, 16*k, 16*k))
	for y := range out.Rect.Dy() {
		for x := range out.Rect.Dx() {
			out.SetRGBA(x, y, src.RGBAAt(x/k, y/k))
		}
	}
	return out
}

// WindowIcons are the sizes desktops pick from for a window's icon.
func WindowIcons() []image.Image {
	return []image.Image{Icon(16), Icon(32), Icon(48)}
}

// ICNS encodes the icon as a macOS .icns file, for the app bundle.
func ICNS() ([]byte, error) {
	var body bytes.Buffer
	for _, e := range []struct {
		kind string
		size int
	}{{"ic07", 128}, {"ic08", 256}, {"ic09", 512}, {"ic10", 1024}} {
		var p bytes.Buffer
		if err := png.Encode(&p, Icon(e.size)); err != nil {
			return nil, err
		}
		body.WriteString(e.kind)
		binary.Write(&body, binary.BigEndian, uint32(8+p.Len()))
		body.Write(p.Bytes())
	}
	var out bytes.Buffer
	out.WriteString("icns")
	binary.Write(&out, binary.BigEndian, uint32(8+body.Len()))
	out.Write(body.Bytes())
	return out.Bytes(), nil
}

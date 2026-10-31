package ui

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// All sound is synthesised at start-up: square waves and noise, like the
// PC speaker and sound chips of the era.

const sampleRate = 44100

// Sound names.
const (
	sndClick = "click"
	sndFlag  = "flag"
	sndClaim = "claim"
	sndBoom  = "boom"
	sndWin   = "win"
	sndLose  = "lose"
	sndTick  = "tick"
	sndGo    = "go"
	sndLevel = "level"
	sndAward = "award"
	sndTurn  = "turn"
)

// Audio plays the sound bank. A nil context means silent mode.
type Audio struct {
	ctx    *audio.Context
	sounds map[string][]byte
	opens  [6][]byte
	volume float64
	music  *audio.Player
	wanted bool // whether the player wants music on
}

func newAudio(enabled bool) *Audio {
	a := &Audio{sounds: map[string][]byte{}}
	if !enabled {
		return a
	}
	a.ctx = audio.NewContext(sampleRate)
	a.sounds = map[string][]byte{
		sndClick: pcm(square(900, .012, .25)),
		sndFlag:  pcm(append(square(660, .03, .3), square(990, .04, .3)...)),
		sndClaim: pcm(arpeggio(.05, .35, 784, 1175, 1568)),
		sndBoom:  pcm(explosion()),
		sndWin:   pcm(arpeggio(.09, .3, 523, 659, 784, 1047, 1319, 1568)),
		sndLose:  pcm(arpeggio(.13, .3, 392, 330, 262, 196)),
		sndTick:  pcm(square(880, .05, .3)),
		sndGo:    pcm(square(1320, .22, .35)),
		sndLevel: pcm(arpeggio(.07, .3, 523, 587, 659, 784, 880, 1047, 1319)),
		sndAward: pcm(arpeggio(.08, .25, 1047, 1568, 2093)),
		sndTurn:  pcm(arpeggio(.06, .3, 659, 880)),
	}
	for i := range a.opens {
		a.opens[i] = pcm(square(420*math.Pow(1.19, float64(i)), .03, .22))
	}
	song := pcm(composeMusic())
	if p, err := a.ctx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(song), int64(len(song)))); err == nil {
		a.music = p
	}
	return a
}

func (a *Audio) setVolume(level int) {
	a.volume = float64(level) / 10
	a.applyMusic()
}

// setMusic turns the background music on or off.
func (a *Audio) setMusic(on bool) {
	a.wanted = on
	a.applyMusic()
}

func (a *Audio) applyMusic() {
	if a.music == nil {
		return
	}
	a.music.SetVolume(a.volume * musicGain)
	if a.wanted && a.volume > 0 {
		a.music.Play()
	} else {
		a.music.Pause()
	}
}

func (a *Audio) play(data []byte) {
	if a.ctx == nil || data == nil || a.volume == 0 {
		return
	}
	p := a.ctx.NewPlayerFromBytes(data)
	p.SetVolume(a.volume)
	p.Play()
}

// Play plays a named sound.
func (a *Audio) Play(name string) { a.play(a.sounds[name]) }

// PlayOpen plays a tile opening, rising in pitch as the opening gets bigger.
func (a *Audio) PlayOpen(tiles int) {
	step := min(int(math.Log2(float64(max(tiles, 1)))), len(a.opens)-1)
	a.play(a.opens[step])
}

func square(freq, dur, vol float64) []float64 {
	n := int(dur * sampleRate)
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / sampleRate
		v := vol
		if math.Mod(t*freq, 1) >= 0.5 {
			v = -vol
		}
		out[i] = v * (1 - float64(i)/float64(n))
	}
	return out
}

func arpeggio(step, vol float64, notes ...float64) []float64 {
	var out []float64
	for i, f := range notes {
		dur := step
		if i == len(notes)-1 {
			dur *= 2.5
		}
		out = append(out, square(f, dur, vol)...)
	}
	return out
}

func explosion() []float64 {
	n := int(.7 * sampleRate)
	out := make([]float64, n)
	rng := rand.New(rand.NewPCG(1, 2))
	lp := 0.0
	for i := range out {
		env := math.Pow(1-float64(i)/float64(n), 2.5)
		lp += (rng.Float64()*2 - 1 - lp) * (0.28*env + 0.02)
		out[i] = (lp*1.6 + 0.4*math.Sin(float64(i)*2*math.Pi*55/sampleRate)) * env
	}
	return out
}

// pcm encodes samples as 16-bit stereo.
func pcm(samples []float64) []byte {
	out := make([]byte, len(samples)*4)
	for i, s := range samples {
		v := int16(math.Max(-1, math.Min(1, s)) * 32000)
		binary.LittleEndian.PutUint16(out[i*4:], uint16(v))
		binary.LittleEndian.PutUint16(out[i*4+2:], uint16(v))
	}
	return out
}

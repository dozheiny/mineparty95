package ui

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
)

// The background music is an original chiptune loop, synthesised at start-up
// like everything else: a pulse wave lead, triangle wave bass and noise drums.

const (
	stepsPerBar = 8
	noteAttack  = 60                    // samples, so that no note starts with a click
	stepSamples = sampleRate * 30 / 132 // one eighth note at 132 BPM
	musicGain   = 0.55                  // keeps the music under the sound effects
)

// leadBars scores the melody in eighth notes. A dash holds the previous note.
var leadBars = []string{
	"A4 C5 E5 C5 A4 E5 D5 C5", "F4 A4 C5 A4 F5 E5 D5 C5", "G4 C5 E5 G5 E5 C5 D5 E5", "D5 B4 G4 B4 D5 - D5 -",
	"A4 C5 E5 C5 A5 G5 E5 C5", "F5 E5 C5 A4 F5 G5 A5 -", "G5 F5 E5 D5 B4 D5 G5 -", "A5 E5 C5 E5 A5 - - -",
	"C6 - A5 F5 C5 F5 A5 C6", "B5 - G5 D5 B4 D5 G5 B5", "G5 - E5 B4 G4 B4 E5 G5", "A5 G5 E5 C5 A4 C5 E5 -",
	"A5 - F5 D5 A4 D5 F5 A5", "G5 - D5 B4 G4 B4 D5 G5", "E5 G5 C6 G5 E5 C5 E5 G5", "G#5 - E5 B4 G#4 B4 E5 -",
}

var bassRoots = []string{
	"A2", "F2", "C3", "G2",
	"A2", "F2", "G2", "A2",
	"F2", "G2", "E2", "A2",
	"D3", "G2", "C3", "E2",
}

// bassPattern gives the semitones above the root on each step of a bar.
var bassPattern = [stepsPerBar]int{0, 0, 12, 0, 0, 12, 0, 12}

var noteSteps = map[string]int{"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}

func noteNumber(name string) int {
	octave, _ := strconv.Atoi(name[len(name)-1:])
	n := noteSteps[name[:1]] + 12*(octave+1)
	if strings.Contains(name, "#") {
		n++
	}
	return n
}

func midiFreq(n int) float64 { return 440 * math.Pow(2, float64(n-69)/12) }

func noteFreq(name string) float64 { return midiFreq(noteNumber(name)) }

func pulseWave(phase float64) float64 {
	if phase < .25 {
		return 1
	}
	return -1
}

func triangleWave(phase float64) float64 { return 4*math.Abs(phase-.5) - 1 }

// addNote mixes one note in. It fades out completely by its last sample, which
// also keeps the loop point silent.
func addNote(out []float64, start, length int, freq, vol float64, wave func(float64) float64) {
	for i := range length {
		env := math.Exp(-2.2*float64(i)/float64(length)) * (1 - float64(i)/float64(length))
		if i < noteAttack {
			env *= float64(i) / noteAttack
		}
		out[start+i] += vol * env * wave(math.Mod(float64(i)*freq/sampleRate, 1))
	}
}

func addNoise(out []float64, start, length int, vol float64, rng *rand.Rand) {
	for i := range length {
		fade := min(float64(i)/noteAttack, 1)
		out[start+i] += vol * fade * (rng.Float64()*2 - 1) * math.Pow(1-float64(i)/float64(length), 3)
	}
}

// composeMusic renders the whole loop.
func composeMusic() []float64 {
	steps := len(leadBars) * stepsPerBar
	out := make([]float64, steps*stepSamples)
	rng := rand.New(rand.NewPCG(95, 1995))

	var melody []string
	for _, bar := range leadBars {
		melody = append(melody, strings.Fields(bar)...)
	}
	for i := 0; i < len(melody); {
		held := 1
		for i+held < len(melody) && melody[i+held] == "-" {
			held++
		}
		addNote(out, i*stepSamples, held*stepSamples, noteFreq(melody[i]), .2, pulseWave)
		i += held
	}
	for i := range steps {
		bar, step := i/stepsPerBar, i%stepsPerBar
		root := noteNumber(bassRoots[bar]) + bassPattern[step]
		addNote(out, i*stepSamples, stepSamples, midiFreq(root), .28, triangleWave)
		switch step {
		case 2, 6:
			addNoise(out, i*stepSamples, stepSamples*3/4, .16, rng)
		default:
			addNoise(out, i*stepSamples, stepSamples/4, .05, rng)
		}
	}
	return out
}

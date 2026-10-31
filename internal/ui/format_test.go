package ui

import (
	"strings"
	"testing"
)

func TestFormatTimeNeverShowsSixtySeconds(t *testing.T) {
	for ms, want := range map[int]string{0: "0:00.0", 1600: "0:01.6", 59990: "0:59.9", 60000: "1:00.0", 188400: "3:08.4"} {
		if got := formatTime(ms); got != want {
			t.Errorf("formatTime(%d) = %q, want %q", ms, got, want)
		}
	}
}

func TestWrapKeepsEveryWordWithinWidth(t *testing.T) {
	text := "Everyone gets the same board. First to clear it wins."
	lines := wrap(text, 20)
	if strings.Join(lines, " ") != text {
		t.Fatalf("lost words: %q", lines)
	}
	for _, l := range lines {
		if len(l) > 20 {
			t.Fatalf("line too long: %q", l)
		}
	}
}

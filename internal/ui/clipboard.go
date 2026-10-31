package ui

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// The clipboard goes through each OS's own tools, which avoids a C
// dependency that would stop the game cross-compiling.

func clipboardCommands() (copyCmd, pasteCmd []string) {
	switch runtime.GOOS {
	case "darwin":
		return []string{"pbcopy"}, []string{"pbpaste"}
	case "windows":
		return []string{"clip"}, []string{"powershell", "-NoProfile", "-Command", "Get-Clipboard"}
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return []string{"wl-copy"}, []string{"wl-paste", "-n"}
		}
	}
	return []string{"xclip", "-selection", "clipboard"}, []string{"xclip", "-selection", "clipboard", "-o"}
}

func command(ctx context.Context, argv []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	hideWindow(cmd)
	return cmd
}

// copyText puts text on the clipboard and reports whether that worked.
func copyText(text string) bool {
	argv, _ := clipboardCommands()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := command(ctx, argv)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run() == nil
}

func pasteText() string {
	_, argv := clipboardCommands()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := command(ctx, argv).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

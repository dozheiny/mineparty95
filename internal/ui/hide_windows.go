package ui

import (
	"os/exec"
	"syscall"
)

// hideWindow stops helper commands flashing a console window.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

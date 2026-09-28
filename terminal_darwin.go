//go:build darwin

package main

import (
	"fmt"
	"os/exec"
)

func openTerminal(command string) error {
	script := fmt.Sprintf(
		`tell application "Terminal"
			activate
			do script %q
		end tell`,
		command,
	)

	cmd := exec.Command("osascript", "-e", script)

	return cmd.Start()
}

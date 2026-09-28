//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

func openTerminal(command string) error {
	terminals := []struct {
		name string
		args []string
	}{
		{
			"gnome-terminal",
			[]string{"--", "bash", "-c", command + "; exec bash"},
		},
		{
			"konsole",
			[]string{"-e", "bash", "-c", command + "; exec bash"},
		},
		{
			"xfce4-terminal",
			[]string{"--command", "bash -c " + shellQuote(command) + "; exec bash"},
		},
		{
			"xterm",
			[]string{"-e", "bash", "-c", command + "; exec bash"},
		},
	}

	for _, terminal := range terminals {
		if _, err := exec.LookPath(terminal.name); err != nil {
			continue
		}

		cmd := exec.Command(terminal.name, terminal.args...)
		return cmd.Start()
	}

	return fmt.Errorf("no supported terminal emulator found")
}

func shellQuote(s string) string {
	return "'" + replaceSingleQuotes(s) + "'"
}

func replaceSingleQuotes(s string) string {
	result := ""

	for _, part := range splitSingleQuotes(s) {
		if result != "" {
			result += "'\\''"
		}

		result += part
	}

	return result
}

func splitSingleQuotes(s string) []string {
	var result []string
	start := 0

	for i, c := range s {
		if c == '\'' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}

	result = append(result, s[start:])

	return result
}

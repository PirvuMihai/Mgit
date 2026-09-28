//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

const createNewConsole = 0x00000010

func openTerminal(command string) error {
	cmd := exec.Command("cmd.exe", command)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewConsole,
	}

	return cmd.Start()
}

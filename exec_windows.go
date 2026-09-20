//go:build windows

package main

import (
	"os"
	"os/exec"
)

func createShellCmd(command string) *exec.Cmd {
	// On Windows, use cmd /c or powershell -Command.
	// Since the user has PowerShell 5.1, we use that for better compatibility with modern CLI tools.
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", command)
}

func killProcessSigterm(p *os.Process) error {
	// Windows doesn't have SIGTERM. Process.Kill() is the closest equivalent
	// for an immediate stop, though it's more like SIGKILL.
	return p.Kill()
}

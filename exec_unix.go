//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func createShellCmd(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}

func killProcessSigterm(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}

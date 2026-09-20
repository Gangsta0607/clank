//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func sessionID() string {
	if !haveTTY() {
		return "headless"
	}
	// Since we don't have rdev on Windows, we can use os.Getppid()
	return fmt.Sprintf("win-%d", os.Getppid())
}

func processAlive(pid int) bool {
	// On Windows, os.FindProcess doesn't check if the process is actually running.
	// We need to try to open the process with limited access to see if it exists and is accessible.
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	h, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(h)
	return true
}

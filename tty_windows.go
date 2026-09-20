//go:build windows

package main

import (
	"bufio"
	"os"
)

func openPlatformTTY() (*os.File, *bufio.Reader, error) {
	// On Windows console, CONIN$ is for reading, CONOUT$ is for writing.
	// Or we can open CON for reading/writing.
	f, err := os.OpenFile("CON", os.O_RDWR, 0)
	if err != nil {
		// Fallback to CONIN$ if CON fails
		f, err = os.OpenFile("CONIN$", os.O_RDWR, 0)
	}
	if err != nil {
		return nil, nil, err
	}
	return f, bufio.NewReader(f), nil
}

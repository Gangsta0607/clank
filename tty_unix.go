//go:build !windows

package main

import (
	"bufio"
	"os"
)

func openPlatformTTY() (*os.File, *bufio.Reader, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	return f, bufio.NewReader(f), nil
}

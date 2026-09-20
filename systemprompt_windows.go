//go:build windows

package main

import (
	"os"
	"path/filepath"
)

func detectShell() string {
	if comspec := os.Getenv("COMSPEC"); comspec != "" {
		return filepath.Base(comspec)
	}
	return "powershell"
}

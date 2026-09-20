//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// detectShell пытается понять, из какого шелла реально запущен clank —
// через имя процесса-родителя (ps -p <ppid>), а не через $SHELL (тот
// показывает логин-шелл юзера, не обязательно текущий). ps есть и на
// Linux, и на macOS "из коробки", так что это кроссплатформенно без
// костылей с /proc.
func detectShell() string {
	out, err := exec.Command("ps", "-p", fmt.Sprint(os.Getppid()), "-o", "comm=").Output()
	if err == nil {
		s := strings.TrimSpace(string(out))
		if s != "" {
			return filepath.Base(s)
		}
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		return filepath.Base(sh) + " (по $SHELL, не точно)"
	}
	return "неизвестно"
}

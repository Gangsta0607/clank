//go:build windows

package main

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

type coord struct {
	X, Y int16
}

type smallRect struct {
	Left, Top, Right, Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

func termWidth() int {
	if s := os.Getenv("COLUMNS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 1 {
			return n
		}
	}

	var info consoleScreenBufferInfo
	// Try stdout, stderr
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		r, _, _ := procGetConsoleInfo.Call(f.Fd(), uintptr(unsafe.Pointer(&info)))
		if r != 0 {
			return int(info.Window.Right - info.Window.Left + 1)
		}
	}
	return 80
}

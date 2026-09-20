//go:build !windows

package main

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// termWidth — ширина терминала. Порядок попыток: ioctl на stdout/stderr/tty,
// затем $COLUMNS, затем 80. Windows не поддерживается, поэтому ioctl можно
// звать напрямую без обёрток.
func termWidth() int {
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		if w := ioctlWidth(f); w > 1 {
			return w
		}
	}
	if f, _, err := ttyIO(); err == nil {
		if w := ioctlWidth(f); w > 1 {
			return w
		}
	}
	if s := os.Getenv("COLUMNS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 1 {
			return n
		}
	}
	return 80
}

func ioctlWidth(f *os.File) int {
	var ws struct {
		Row, Col, Xpixel, Ypixel uint16
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0
	}
	return int(ws.Col)
}

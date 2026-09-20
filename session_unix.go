//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func sessionID() string {
	f, _, err := ttyIO()
	if err != nil {
		// cron, CI, пайплайн без терминала — общий файл на всех
		return "headless"
	}
	var st syscall.Stat_t
	if err := syscall.Fstat(int(f.Fd()), &st); err != nil {
		return "headless"
	}
	return fmt.Sprintf("%x-%d", uint64(st.Rdev), os.Getppid())
}

// processAlive — сигнал 0 не доставляется, но проверяет существование
// процесса: ESRCH означает, что терминала с таким шеллом больше нет.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

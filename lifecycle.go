package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func getExecPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(execPath)
}

func removeBinary(execPath string) error {
	if runtime.GOOS == "windows" {
		oldPath := execPath + ".old"
		_ = os.Remove(oldPath)
		if err := os.Rename(execPath, oldPath); err != nil {
			return err
		}
		return nil
	}
	return os.Remove(execPath)
}

func cmdPurge(args []string) int {
	var autoYes bool
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			autoYes = true
		}
	}

	dir, err := clankDir()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println("конфигурационный каталог и данные уже пусты")
		return exitOK
	}

	if !autoYes && !confirmYN(fmt.Sprintf("очистить все настройки, сессии и данные (%s)? [y/N]: ", dir), false) {
		fmt.Println("отменено")
		return exitOK
	}

	if err := os.RemoveAll(dir); err != nil {
		fail("ошибка при очистке данных (%s): %v", dir, err)
		return exitConfig
	}

	fmt.Println("все пользовательские данные и настройки clank очищены")
	return exitOK
}

func cmdUninstall(args []string) int {
	var autoYes bool
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			autoYes = true
		}
	}

	execPath, err := getExecPath()
	if err != nil {
		fail("не удалось определить путь к бинарнику: %v", err)
		return exitConfig
	}

	if !autoYes && !confirmYN(fmt.Sprintf("удалить исполняемый файл clank (%s)? [y/N]: ", execPath), false) {
		fmt.Println("отменено")
		return exitOK
	}

	if err := removeBinary(execPath); err != nil {
		fail("ошибка при удалении %s: %v\n  (если нет прав, выполните: sudo rm %s)", execPath, err, execPath)
		return exitConfig
	}

	fmt.Printf("исполняемый файл clank (%s) удалён с машины\n", execPath)
	return exitOK
}

func cmdNuke(args []string) int {
	var autoYes bool
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			autoYes = true
		}
	}

	execPath, err := getExecPath()
	if err != nil {
		fail("не удалось определить путь к бинарнику: %v", err)
		return exitConfig
	}

	dir, _ := clankDir()

	prompt := fmt.Sprintf("ВНИМАНИЕ: это полностью удалит бинарник (%s) и ВСЕ данные (%s). Продолжить? [y/N]: ", execPath, dir)
	if !autoYes && !confirmYN(prompt, false) {
		fmt.Println("отменено")
		return exitOK
	}

	var errs []string

	if dir != "" {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("данные (%v)", err))
		}
	}

	if err := removeBinary(execPath); err != nil {
		errs = append(errs, fmt.Sprintf("бинарник (%v)", err))
	}

	if len(errs) > 0 {
		fail("при полном удалении возникли ошибки: %s", errs)
		return exitConfig
	}

	fmt.Println("clank полностью удалён с машины (бинарник и все данные)")
	return exitOK
}

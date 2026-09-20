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
		fmt.Println(M.PurgeEmpty)
		return exitOK
	}

	if !autoYes && !confirmYN(fmt.Sprintf(M.PurgeAsk, dir), false) {
		fmt.Println(M.Aborted)
		return exitOK
	}

	if err := os.RemoveAll(dir); err != nil {
		fail(M.PurgeFail, dir, err)
		return exitConfig
	}

	fmt.Println(M.PurgeDone)
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
		fail(M.BinPathFail, err)
		return exitConfig
	}

	if !autoYes && !confirmYN(fmt.Sprintf(M.UninstAsk, execPath), false) {
		fmt.Println(M.Aborted)
		return exitOK
	}

	if cf, err := loadConfigFile(); err == nil {
		removeCompletions(cf)
	}

	if err := removeBinary(execPath); err != nil {
		fail(M.UninstFail, execPath, err, execPath)
		return exitConfig
	}

	fmt.Printf(M.UninstDone, execPath)
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
		fail(M.BinPathFail, err)
		return exitConfig
	}

	dir, _ := clankDir()

	prompt := fmt.Sprintf(M.NukeAsk, execPath, dir)
	if !autoYes && !confirmYN(prompt, false) {
		fmt.Println(M.Aborted)
		return exitOK
	}

	if cf, err := loadConfigFile(); err == nil {
		removeCompletions(cf)
	}

	var errs []string

	if dir != "" {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf(M.NukeDataErr, err))
		}
	}

	if err := removeBinary(execPath); err != nil {
		errs = append(errs, fmt.Sprintf(M.NukeBinErr, err))
	}

	if len(errs) > 0 {
		fail(M.NukePartFail, errs)
		return exitConfig
	}

	fmt.Println(M.NukeDone)
	return exitOK
}

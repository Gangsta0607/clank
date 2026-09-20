package main

import (
	"fmt"
	"os"
)

// подставляется линкером: go build -ldflags "-X main.version=..."
var version = "dev"

func usage() {
	fmt.Fprintln(os.Stderr, M.Usage)
}

func main() {
	initLang()
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitConfig)
	}

	cmd := os.Args[1]
	rest := os.Args[2:]

	var code int
	switch cmd {
	case "ask":
		code = cmdAsk(rest)
	case "models":
		code = cmdModels(rest)
	case "config":
		code = cmdConfig(rest)
	case "session":
		code = cmdSession(rest)
	case "yolo":
		code = cmdYolo(rest)
	case "completion":
		code = cmdCompletion(rest)
	case "language":
		code = cmdLanguage(rest)
	case "update":
		code = cmdUpdate(rest)
	case "purge":
		code = cmdPurge(rest)
	case "uninstall":
		code = cmdUninstall(rest)
	case "nuke":
		code = cmdNuke(rest)
	case "version", "--version":
		fmt.Println("clank", version)
		code = exitOK
	case "-h", "--help", "help":
		code = cmdHelp(rest)
	default:
		// Всё, что не подкоманда, — вопрос. Спрашивать приходится чаще,
		// чем настраивать, так что `clank "..."` работает без слова ask.
		// Единственный конфликт — вопрос, чьё первое слово совпадает с
		// именем подкоманды; для него остался явный `clank ask ...`.
		code = cmdAsk(os.Args[1:])
	}
	os.Exit(code)
}

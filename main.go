package main

import (
	"fmt"
	"os"
)

// подставляется линкером: go build -ldflags "-X main.version=..."
var version = "dev"

func usage() {
	fmt.Fprintln(os.Stderr, `clank — минималистичный CLI к OpenAI-совместимому API

использование:
  clank <вопрос>                        задать вопрос (stdin читается сам, если это пайп)
  clank -c <вопрос>                     форс чтения stdin, если автодетект не сработал
  clank -r <вопрос>                     продолжить сессию этого терминала (--resume)
  clank -c -r <вопрос>                  и пайп, и продолжение — работают вместе
  clank -i <файл> <вопрос>              прикрепить изображение (PNG, JPG, WebP, GIF) к вопросу
  clank --yolo=on|off <вопрос>          задать YOLO режим на сессию (on — без подтверждения команд)
  clank --reasoning=on|off <вопрос>     принудительно включить/выключить размышления модели
  clank -v <вопрос>                     подробный лог хода работы (-q — наоборот, молча)
  clank -- <вопрос>                     всё дальше — текст вопроса, даже если похоже на флаг

  clank ask <вопрос>                    то же самое явно (нужно, если вопрос начинается
                                        со слова models/config/session/help)

  clank models [фильтр]                 модели активного профиля; выбор цепочки: 1,4,7

  clank config init                     быстрая настройка профиля "default"
  clank config add <имя>                именованный профиль (для нескольких провайдеров)
  clank config use <имя>                переключить активный профиль
  clank config list                     список профилей
  clank config rm <имя>
  clank config show                     показать активный профиль
  clank config set-url/set-key/set-model/set-proxy/set-tools/set-reasoning/set-exec-timeout <значение>
  clank config test-tools               проверить поддержку tool calls моделью
  clank config test-reasoning           проверить и откалибровать поддержку reasoning моделью
  clank config test-vision              проверить поддержку изображений (vision) моделью
  clank config allow-rm/allow-clear     список команд, выполняемых без подтверждения

  clank session show                    транскрипт сессии этого терминала
  clank session clear [--all]           стереть сессию (--all — во всех терминалах)
  clank session list                    все сессии на машине

  clank yolo [on|off]                   включить/выключить глобальный режим YOLO (без подтверждения команд)
  clank update [-y]                     проверить и установить новую версию с GitHub
  clank purge [-y]                      стереть все пользовательские данные (~/.config/clank/)
  clank uninstall [-y]                  удалить исполняемый файл clank
  clank nuke [-y]                       полностью удалить clank и все его данные

  clank version

примеры:
  clank "почему grub не видит второй диск"
  dmesg | tail -50 | clank "разбери ошибку"
  clank -r "а если так: ..."

вопрос лучше брать в кавычки: без них ?, *, >, ! и обратные кавычки перехватит шелл.`)
}

func main() {
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
		usage()
		code = exitOK
	default:
		// Всё, что не подкоманда, — вопрос. Спрашивать приходится чаще,
		// чем настраивать, так что `clank "..."` работает без слова ask.
		// Единственный конфликт — вопрос, чьё первое слово совпадает с
		// именем подкоманды; для него остался явный `clank ask ...`.
		code = cmdAsk(os.Args[1:])
	}
	os.Exit(code)
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cmdCompletion — управление автодополнением:
//
//	clank completion              поставить для текущего шелла
//	clank completion install [sh] поставить (шелл по умолчанию — текущий)
//	clank completion uninstall    убрать всё, что ставили мы
//	clank completion print [sh]   напечатать скрипт (отладка, свой путь)
func cmdCompletion(args []string) int {
	if len(args) == 0 {
		return cmdCompletionInstall("")
	}
	switch strings.ToLower(args[0]) {
	case "install":
		shell := ""
		if len(args) > 1 {
			shell = strings.ToLower(args[1])
		}
		return cmdCompletionInstall(shell)
	case "uninstall":
		return cmdCompletionUninstall()
	case "print":
		shell := completionShell()
		if len(args) > 1 {
			shell = strings.ToLower(args[1])
		}
		return printCompletionScript(shell)
	default:
		fail(M.CompletionUsage)
		return exitConfig
	}
}

// cmdCompletionUninstall убирает все установленные нами скрипты
// и чистит запись в конфиге.
func cmdCompletionUninstall() int {
	cf, err := loadConfigFile()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	if len(cf.Completions) == 0 {
		fmt.Println(M.CompletionNone)
		return exitOK
	}
	removeCompletions(cf)
	cf.Completions = nil
	if err := saveConfigFile(cf); err != nil {
		fail(M.SaveFail, err)
		return exitConfig
	}
	return exitOK
}

func printCompletionScript(shell string) int {
	script, ok := completionScript(shell)
	if !ok {
		fail(M.CompletionBadShell, shell)
		return exitConfig
	}
	fmt.Println(script)
	return exitOK
}

// completionShell — текущий шелл одним словом. Сначала смотрим реального
// родителя (detectShell), если там мусор — откатываемся на basename $SHELL.
func completionShell() string {
	if s := normalizeShellToken(detectShell()); supportedShell(s) {
		return s
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		if s := normalizeShellToken(filepath.Base(sh)); supportedShell(s) {
			return s
		}
	}
	return "unknown"
}

// normalizeShellToken чистит имя процесса шелла: первое слово, нижний
// регистр, без дефиса логин-шелла ("-zsh" → "zsh"). detectShell может
// вернуть "zsh (via $SHELL, approximate)" — берём первое слово.
func normalizeShellToken(s string) string {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.TrimPrefix(strings.ToLower(parts[0]), "-")
}

func supportedShell(s string) bool {
	_, ok := completionScript(s)
	return ok
}

func completionScript(shell string) (string, bool) {
	ru := curLang == LangRU
	switch shell {
	case "zsh":
		if ru {
			return zshCompletionRU, true
		}
		return zshCompletionEN, true
	case "bash":
		return bashCompletion, true
	case "fish":
		if ru {
			return fishCompletionRU, true
		}
		return fishCompletionEN, true
	}
	return "", false
}

// reinstallCompletions перезаписывает установленные скрипты под текущий
// язык (зовётся из `clank language`). Шелл выводим из пути: других записей
// у нас не бывает.
func reinstallCompletions(cf *ConfigFile) {
	for _, path := range cf.Completions {
		shell := ""
		switch {
		case filepath.Base(path) == "_clank":
			shell = "zsh"
		case strings.HasSuffix(path, ".fish"):
			shell = "fish"
		case filepath.Base(path) == "clank":
			shell = "bash"
		default:
			continue
		}
		script, ok := completionScript(shell)
		if !ok {
			continue
		}
		if err := os.WriteFile(path, []byte(script+"\n"), 0644); err != nil {
			warn(M.CompletionWriteFail, err)
			continue
		}
		info(M.CompletionDone, path)
	}
}

// completionTarget — куда класть скрипт для шелла. Только пользовательские
// каталоги, никакого sudo.
func completionTarget(shell string) (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	switch shell {
	case "zsh":
		// Всегда ~/.zfunc: детерминировано работает и с oh-my-zsh, и без.
		// Гадать по подпапкам omz не стали — у него нет гарантированного
		// места под сторонние дополнения, а fpath юзер правит один раз.
		return filepath.Join(home, ".zfunc", "_clank"), true
	case "bash":
		return filepath.Join(home, ".local", "share", "bash-completion", "completions", "clank"), true
	case "fish":
		return filepath.Join(home, ".config", "fish", "completions", "clank.fish"), true
	}
	return "", false
}

// installCompletion пишет скрипт на место и запоминает путь в конфиге.
func installCompletion(cf *ConfigFile, shell string) (string, error) {
	script, ok := completionScript(shell)
	if !ok {
		return "", fmt.Errorf(M.CompletionBadShell, shell)
	}
	path, ok := completionTarget(shell)
	if !ok {
		return "", fmt.Errorf(M.CompletionBadShell, shell)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf(M.CompletionWriteFail, err)
	}
	if err := os.WriteFile(path, []byte(script+"\n"), 0644); err != nil {
		return "", fmt.Errorf(M.CompletionWriteFail, err)
	}
	for _, p := range cf.Completions {
		if p == path {
			return path, saveConfigFile(*cf)
		}
	}
	cf.Completions = append(cf.Completions, path)
	return path, saveConfigFile(*cf)
}

func cmdCompletionInstall(shell string) int {
	if shell == "" {
		shell = completionShell()
	}
	cf, err := loadConfigFile()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	path, err := installCompletion(&cf, shell)
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	info(M.CompletionDone, path)
	if shell == "zsh" && strings.Contains(path, ".zfunc") {
		fmt.Println(M.CompletionZshHint)
	}
	return exitOK
}

// offerCompletion предлагает поставить автодополнение. Зовётся из мастера
// настройки при создании нового профиля — человек как раз всё ставит.
// Молчит, если дополнения уже записаны в конфиге или файл на месте.
func offerCompletion(cf *ConfigFile) {
	if !haveTTY() || len(cf.Completions) > 0 {
		return
	}
	shell := completionShell()
	if _, ok := completionScript(shell); !ok {
		return
	}
	if path, ok := completionTarget(shell); ok {
		if _, err := os.Stat(path); err == nil {
			return
		}
	}
	target, _ := completionTarget(shell)
	if !confirmYN(fmt.Sprintf(M.AskCompletion, shell, target), true) {
		return
	}
	path, err := installCompletion(cf, shell)
	if err != nil {
		warn("%v", err)
		return
	}
	info(M.CompletionDone, path)
	if shell == "zsh" && strings.Contains(path, ".zfunc") {
		fmt.Println(M.CompletionZshHint)
	}
}

// removeCompletions убирает установленные нами скрипты (uninstall/nuke).
func removeCompletions(cf ConfigFile) {
	for _, path := range cf.Completions {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err == nil {
			info(M.CompletionRemoved, path)
		}
	}
}

const zshCompletionEN = `#compdef clank
# zsh completion for clank — generated by: clank completion zsh
_clank() {
  local context state line
  typeset -A opt_args

  _arguments -C \
    '(-c --context)'{-c,--context}'[force stdin read]' \
    '(-r --resume)'{-r,--resume}'[resume this terminal session]' \
    '(-v --verbose)'{-v,--verbose}'[verbose log]' \
    '(-q --quiet)'{-q,--quiet}'[quiet mode]' \
    '(-i --image)'{-i,--image}'[attach image]:file:_files' \
    '--yolo=[YOLO mode for the reply]:mode:(on off)' \
    '--reasoning=[force reasoning]:mode:(on off)' \
    '(-h --help)'{-h,--help}'[show help]' \
    '1: :->cmds' \
    '*:: :->args'

  case $state in
    cmds)
      local -a commands
      commands=(
        'ask:ask the model'
        'models:list or pick models'
        'config:manage profiles'
        'session:manage sessions'
        'yolo:toggle YOLO mode'
        'language:set interface language'
        'completion:print completion script'
        'update:update from GitHub'
        'purge:wipe user data'
        'uninstall:remove the binary'
        'nuke:remove everything'
        'help:show help'
        'version:print version'
      )
      _describe 'command' commands
      ;;
    args)
      case $line[1] in
        config)
          local -a subs
          subs=(
            'init:setup wizard'
            'add:create profile'
            'use:switch profile'
            'list:list profiles'
            'rm:remove profile'
            'show:show active profile'
            'set-url:set base url'
            'set-key:set api key'
            'set-model:set model chain'
            'set-proxy:set proxy'
            'set-tools:override use_tools'
            'set-reasoning:reasoning on/off/reset'
            'set-exec-timeout:set exec timeout'
            'test-tools:check tool calls'
            'test-reasoning:check reasoning'
            'test-vision:check vision'
            'test-model:check all at once'
            'allow-rm:unallow a command'
            'allow-clear:clear allowed list'
          )
          _describe 'config subcommand' subs
          case $line[2] in
            use|rm)
              local -a profiles
              profiles=(${(f)"$(clank config list 2>/dev/null | sed 's/^[* ] //;s/ .*//')"})
              _describe 'profile' profiles
              ;;
          esac
          ;;
        session)
          _describe 'session subcommand' \
            'show:show transcript' \
            'clear:clear session' \
            'list:list sessions'
          ;;
        yolo)
          _describe 'mode' 'on:enable' 'off:disable'
          ;;
        language)
          _describe 'language' 'en:English' 'ru:Russian' 'auto:autodetect'
          ;;
        completion)
          _describe 'action' 'install:install' 'uninstall:remove' 'print:print script'
          case $line[2] in
            print|install)
              _describe 'shell' 'zsh:zsh' 'bash:bash' 'fish:fish'
              ;;
          esac
          ;;
        help)
          _describe 'topic' \
            'ask:ask' 'models:models' 'config:config' 'session:session' \
            'yolo:yolo' 'language:language' 'completion:completion' \
            'update:update' 'purge:purge' 'uninstall:uninstall' 'nuke:nuke'
          ;;
      esac
      ;;
  esac
}
_clank "$@"
`

const bashCompletion = `# bash completion for clank — generated by: clank completion bash
_clank() {
  local cur prev words cword
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  local cmds="ask models config session yolo language completion update purge uninstall nuke help version"

  if [[ $COMP_CWORD -eq 1 ]]; then
    if [[ "$cur" == -* ]]; then
      COMPREPLY=($(compgen -W "-c -r -v -q -i -h --context --resume --verbose --quiet --image --help --yolo= --reasoning=" -- "$cur"))
    else
      COMPREPLY=($(compgen -W "$cmds" -- "$cur"))
    fi
    return 0
  fi

  case "${COMP_WORDS[1]}" in
    config)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=($(compgen -W "init add use list rm show set-url set-key set-model set-proxy set-tools set-reasoning set-exec-timeout test-tools test-reasoning test-vision test-model allow-rm allow-clear" -- "$cur"))
      elif [[ $COMP_CWORD -eq 3 ]]; then
        case "$prev" in
          use|rm)
            local profiles
            profiles=$(clank config list 2>/dev/null | sed 's/^[* ] //;s/ .*//')
            COMPREPLY=($(compgen -W "$profiles" -- "$cur"))
            ;;
        esac
      fi
      ;;
    session)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=($(compgen -W "show clear list" -- "$cur"))
      fi
      ;;
    yolo)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=($(compgen -W "on off" -- "$cur"))
      fi
      ;;
    language)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=($(compgen -W "en ru auto" -- "$cur"))
      fi
      ;;
    completion)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=($(compgen -W "install uninstall print" -- "$cur"))
      elif [[ $COMP_CWORD -eq 3 ]]; then
        case "$prev" in
          install|print)
            COMPREPLY=($(compgen -W "zsh bash fish" -- "$cur"))
            ;;
        esac
      fi
      ;;
    help)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=($(compgen -W "ask models config session yolo language completion update purge uninstall nuke" -- "$cur"))
      fi
      ;;
  esac
}
complete -F _clank clank
`

const fishCompletionEN = `# fish completion for clank — generated by: clank completion fish
# flags: -c -r -v -q -i -h --context --resume --verbose --quiet --image --yolo --reasoning --help
set -l cmds ask models config session yolo language completion update purge uninstall nuke help version
complete -c clank -f -n __fish_use_subcommand -a "$cmds"

set -l config_subs init add use list rm show set-url set-key set-model set-proxy set-tools set-reasoning set-exec-timeout test-tools test-reasoning test-vision test-model allow-rm allow-clear
complete -c clank -f -n '__fish_seen_subcommand_from config' -a "$config_subs"

complete -c clank -f -n '__fish_seen_subcommand_from use rm; and __fish_seen_subcommand_from config' -a "(clank config list 2>/dev/null | sed 's/^[* ] //;s/ .*//')"

complete -c clank -f -n '__fish_seen_subcommand_from session' -a "show clear list"
complete -c clank -f -n '__fish_seen_subcommand_from yolo' -a "on off"
complete -c clank -f -n '__fish_seen_subcommand_from language' -a "en ru auto"
complete -c clank -f -n '__fish_seen_subcommand_from completion' -a "install uninstall print"
complete -c clank -f -n '__fish_seen_subcommand_from print install; and __fish_seen_subcommand_from completion' -a "zsh bash fish"
complete -c clank -f -n '__fish_seen_subcommand_from help' -a "ask models config session yolo language completion update purge uninstall nuke"

complete -c clank -s c -l context -d 'force stdin read'
complete -c clank -s r -l resume -d 'resume session'
complete -c clank -s v -l verbose -d 'verbose log'
complete -c clank -s q -l quiet -d 'quiet mode'
complete -c clank -s i -l image -r -d 'attach image'
complete -c clank -s h -l help -d 'show help'`

const zshCompletionRU = `#compdef clank
# zsh completion for clank — generated by: clank completion print zsh
_clank() {
  local context state line
  typeset -A opt_args

  _arguments -C \
    '(-c --context)'{-c,--context}'[принудительно читать stdin]' \
    '(-r --resume)'{-r,--resume}'[продолжить сессию вкладки]' \
    '(-v --verbose)'{-v,--verbose}'[подробный лог]' \
    '(-q --quiet)'{-q,--quiet}'[тихий режим]' \
    '(-i --image)'{-i,--image}'[прикрепить картинку]:file:_files' \
    '--yolo=[YOLO на одну реплику]:режим:(on off)' \
    '--reasoning=[мышление модели]:режим:(on off)' \
    '(-h --help)'{-h,--help}'[показать справку]' \
    '1: :->cmds' \
    '*:: :->args'

  case $state in
    cmds)
      local -a commands
      commands=(
        'ask:спросить модель'
        'models:список и выбор моделей'
        'config:управление профилями'
        'session:управление сессиями'
        'yolo:режим YOLO'
        'language:язык интерфейса'
        'completion:автодополнение'
        'update:обновление с GitHub'
        'purge:стереть данные'
        'uninstall:удалить бинарник'
        'nuke:удалить всё'
        'help:справка'
        'version:версия'
      )
      _describe 'command' commands
      ;;
    args)
      case $line[1] in
        config)
          local -a subs
          subs=(
            'init:мастер настройки'
            'add:создать профиль'
            'use:переключить профиль'
            'list:список профилей'
            'rm:удалить профиль'
            'show:показать активный'
            'set-url:задать url'
            'set-key:задать ключ'
            'set-model:цепочка моделей'
            'set-proxy:задать прокси'
            'set-tools:оверрайд use_tools'
            'set-reasoning:мышление on/off/сброс'
            'set-exec-timeout:потолок команды'
            'test-tools:проверить tool calls'
            'test-reasoning:проверить reasoning'
            'test-vision:проверить vision'
            'test-model:все проверки разом'
            'allow-rm:убрать из разрешённых'
            'allow-clear:очистить разрешённые'
          )
          _describe 'config subcommand' subs
          case $line[2] in
            use|rm)
              local -a profiles
              profiles=(${(f)"$(clank config list 2>/dev/null | sed 's/^[* ] //;s/ .*//')"})
              _describe 'profile' profiles
              ;;
          esac
          ;;
        session)
          _describe 'session subcommand' \
            'show:транскрипт' \
            'clear:стереть сессию' \
            'list:список сессий'
          ;;
        yolo)
          _describe 'mode' 'on:включить' 'off:выключить'
          ;;
        language)
          _describe 'language' 'en:английский' 'ru:русский' 'auto:авто'
          ;;
        completion)
          _describe 'action' 'install:установить' 'uninstall:убрать' 'print:напечатать'
          case $line[2] in
            print|install)
              _describe 'shell' 'zsh:zsh' 'bash:bash' 'fish:fish'
              ;;
          esac
          ;;
        help)
          _describe 'topic' \
            'ask:вопрос' 'models:модели' 'config:профили' 'session:сессии' \
            'yolo:yolo' 'language:язык' 'completion:дополнение' \
            'update:обновление' 'purge:чистка' 'uninstall:удаление' 'nuke:снос'
          ;;
      esac
      ;;
  esac
}
_clank "$@"
`

const fishCompletionRU = `# fish completion for clank — generated by: clank completion print fish
# flags: -c -r -v -q -i -h --context --resume --verbose --quiet --image --yolo --reasoning --help
set -l cmds ask models config session yolo language completion update purge uninstall nuke help version
complete -c clank -f -n __fish_use_subcommand -a "$cmds"

set -l config_subs init add use list rm show set-url set-key set-model set-proxy set-tools set-reasoning set-exec-timeout test-tools test-reasoning test-vision test-model allow-rm allow-clear
complete -c clank -f -n '__fish_seen_subcommand_from config' -a "$config_subs"

complete -c clank -f -n '__fish_seen_subcommand_from use rm; and __fish_seen_subcommand_from config' -a "(clank config list 2>/dev/null | sed 's/^[* ] //;s/ .*//')"

complete -c clank -f -n '__fish_seen_subcommand_from session' -a "show clear list"
complete -c clank -f -n '__fish_seen_subcommand_from yolo' -a "on off"
complete -c clank -f -n '__fish_seen_subcommand_from language' -a "en ru auto"
complete -c clank -f -n '__fish_seen_subcommand_from completion' -a "install uninstall print"
complete -c clank -f -n '__fish_seen_subcommand_from print install; and __fish_seen_subcommand_from completion' -a "zsh bash fish"
complete -c clank -f -n '__fish_seen_subcommand_from help' -a "ask models config session yolo language completion update purge uninstall nuke"

complete -c clank -s c -l context -d 'принудительно читать stdin'
complete -c clank -s r -l resume -d 'продолжить сессию'
complete -c clank -s v -l verbose -d 'подробный лог'
complete -c clank -s q -l quiet -d 'тихий режим'
complete -c clank -s i -l image -r -d 'прикрепить картинку'
complete -c clank -s h -l help -d 'показать справку'
`

// Конец скриптов автодополнения.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Весь интерактивный ввод идёт через управляющий терминал, а не через
// os.Stdin: stdin может быть занят пайпом с контекстом. Ридер строго один
// на процесс — bufio.Reader тянет данные кусками, и второй ридер поверх
// того же fd потерял бы то, что первый уже забрал в буфер.
var (
	ttyOnce   sync.Once
	ttyFile   *os.File
	ttyReader *bufio.Reader
	ttyErr    error
)

func ttyIO() (*os.File, *bufio.Reader, error) {
	ttyOnce.Do(func() {
		f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			ttyErr = err
			return
		}
		ttyFile = f
		ttyReader = bufio.NewReader(f)
	})
	return ttyFile, ttyReader, ttyErr
}

func haveTTY() bool {
	_, _, err := ttyIO()
	return err == nil
}

// askLine задаёт вопрос и читает строку с терминала. ok=false означает,
// что терминала нет или пришёл EOF — вызывающий решает, что делать.
func askLine(prompt string) (line string, ok bool) {
	f, rd, err := ttyIO()
	if err != nil {
		return "", false
	}
	fmt.Fprint(f, prompt)
	s, err := rd.ReadString('\n')
	if err != nil {
		fmt.Fprintln(f)
		return "", false
	}
	return strings.TrimSpace(s), true
}

// askLineDefault — вопрос с показом текущего значения; пустой ввод
// оставляет как было.
func askLineDefault(label, current, display string) string {
	prompt := label + ": "
	if current != "" {
		prompt = fmt.Sprintf("%s [%s]: ", label, display)
	}
	line, ok := askLine(prompt)
	if !ok || line == "" {
		return current
	}
	return line
}

// confirmYN перезапрашивает при непонятном ответе, а не считает всё
// подряд отказом: раньше опечатка или лишний символ молча означали "нет".
// Пустая строка = defaultYes. EOF (терминал закрыли) = отказ.
func confirmYN(prompt string, defaultYes bool) bool {
	f, rd, err := ttyIO()
	if err != nil {
		// Терминала нет вообще (cron, CI) — уважаем заявленный дефолт.
		return defaultYes
	}
	for {
		fmt.Fprint(f, prompt)
		line, err := rd.ReadString('\n')
		if err != nil {
			fmt.Fprintln(f)
			return false
		}
		switch strings.TrimSpace(strings.ToLower(line)) {
		case "":
			return defaultYes
		case "y", "yes", "д", "да":
			return true
		case "n", "no", "н", "нет":
			return false
		default:
			fmt.Fprintln(f, "не понял: y — да, n — нет")
		}
	}
}

// answer — исход подтверждения команды.
type answer int

const (
	ansNo answer = iota
	ansYes
	ansAlways
)

// confirmCommand печатает команду и спрашивает подтверждение в один и тот
// же fd — управляющий терминал. Раньше команда уходила в stderr, а вопрос
// в /dev/tty, и при `clank ... 2>log` пользователь видел только "выполнить?"
// без текста команды, то есть подтверждал вслепую.
func confirmCommand(cmdStr, name string, level TrustLevel) answer {
	f, rd, err := ttyIO()
	if err != nil {
		warn("нет управляющего терминала — подтвердить выполнение невозможно, отказываю")
		return ansNo
	}

	fmt.Fprintf(f, "\nмодель хочет выполнить:\n%s\n", indentBlock(sanitizeForDisplay(cmdStr), "    "))
	var prompt string
	if level == TrustSimple {
		fmt.Fprintf(f, "  («%s» разрешена только для простых команд, но эта команда содержит спецсимволы шелла — подтверди вручную)\n", name)
		prompt = fmt.Sprintf("выполнить? [y/N/a] (a — разрешать «%s» всегда, включая сложные): ", name)
	} else {
		prompt = fmt.Sprintf("выполнить? [y/N/a] (a — разрешать «%s» всегда): ", name)
	}

	for {
		fmt.Fprint(f, prompt)
		line, err := rd.ReadString('\n')
		if err != nil {
			fmt.Fprintln(f)
			return ansNo
		}
		switch strings.TrimSpace(strings.ToLower(line)) {
		case "":
			return ansNo
		case "y", "yes", "д", "да":
			return ansYes
		case "n", "no", "н", "нет":
			return ansNo
		case "a", "always", "в", "всегда":
			return ansAlways
		default:
			fmt.Fprintln(f, "не понял: y — выполнить, n — отказать, a — разрешать эту команду всегда")
		}
	}
}

// promptQuestion задаёт вопрос пользователю, отображает нумерованные
// варианты (если переданы) и возвращает выбранный вариант либо кастомный ввод.
func promptQuestion(q string, options []string, defVal string) (string, bool) {
	f, rd, err := ttyIO()
	if err != nil {
		if defVal != "" {
			info("нет управляющего терминала — выбираю значение по умолчанию: %s", defVal)
			if len(options) > 0 {
				if n, err := strconv.Atoi(defVal); err == nil && n >= 1 && n <= len(options) {
					return options[n-1], true
				}
			}
			return defVal, true
		}
		warn("нет управляющего терминала — ответить на вопрос невозможно")
		return "", false
	}

	fmt.Fprintf(f, "\n[?] %s\n", sanitizeForDisplay(q))
	for i, opt := range options {
		fmt.Fprintf(f, "    %d) %s\n", i+1, sanitizeForDisplay(opt))
	}

	defDisplay := defVal
	if len(options) > 0 && defVal != "" {
		if n, err := strconv.Atoi(defVal); err == nil && n >= 1 && n <= len(options) {
			defDisplay = fmt.Sprintf("%d (%s)", n, options[n-1])
		}
	}

	var prompt string
	switch {
	case len(options) > 0 && defDisplay != "":
		prompt = fmt.Sprintf("выбери 1-%d или напиши свой ответ [%s]: ", len(options), defDisplay)
	case len(options) > 0:
		prompt = fmt.Sprintf("выбери 1-%d или напиши свой ответ: ", len(options))
	case defDisplay != "":
		prompt = fmt.Sprintf("ответ [%s]: ", defDisplay)
	default:
		prompt = "ответ: "
	}

	for {
		fmt.Fprint(f, prompt)
		line, err := rd.ReadString('\n')
		if err != nil {
			fmt.Fprintln(f)
			return "", false
		}
		text := strings.TrimSpace(line)
		if text == "" {
			if defVal != "" {
				if len(options) > 0 {
					if n, err := strconv.Atoi(defVal); err == nil && n >= 1 && n <= len(options) {
						return options[n-1], true
					}
				}
				return defVal, true
			}
			if len(options) == 0 {
				return "", true
			}
			fmt.Fprintln(f, "введи номер варианта или свой ответ")
			continue
		}

		if len(options) > 0 {
			if n, err := strconv.Atoi(text); err == nil && n >= 1 && n <= len(options) {
				return options[n-1], true
			}
		}
		return text, true
	}
}

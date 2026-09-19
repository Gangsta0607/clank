package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type outcome int

const (
	outcomeAnswer outcome = iota
	outcomeDeclined
	outcomeInterrupted
	outcomeEmpty
)

func readStdin() string {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		warn("не смог дочитать stdin: %v — контекст может быть неполным", err)
	}
	return strings.TrimRight(string(data), "\n")
}

func cmdAsk(args []string) int {
	var (
		wantContext bool
		resume      bool
		rest        []string
		literal     bool
	)
	for _, a := range args {
		if literal {
			rest = append(rest, a)
			continue
		}
		switch a {
		case "--":
			literal = true
		case "-c", "--context":
			wantContext = true
		case "-r", "--resume":
			resume = true
		case "-v", "--verbose":
			verboseLevel = vVerbose
		case "-q", "--quiet":
			verboseLevel = vQuiet
		case "-h", "--help":
			usage()
			return exitOK
		default:
			// Раньше неизвестный флаг молча уезжал в текст вопроса, и
			// опечатка в `--resmue` никак себя не проявляла.
			if strings.HasPrefix(a, "-") && len(a) > 1 {
				fail("неизвестный флаг: %s (если это часть вопроса — поставь перед ним --)", a)
				return exitConfig
			}
			rest = append(rest, a)
		}
	}

	question := strings.Join(rest, " ")
	if strings.TrimSpace(question) == "" {
		fail("нечего спрашивать")
		fmt.Fprintln(os.Stderr, `использование: clank [-c] [-r] [-v|-q] <вопрос>`)
		return exitConfig
	}

	cf, err := loadConfigFile()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	profileName, profile, err := activeProfile(&cf)
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	if err := profile.validate(); err != nil {
		fail("%v", err)
		return exitConfig
	}
	detail("профиль %s, модели: %s", profileName, strings.Join(profile.Models, " → "))

	gcSessions()

	var context string
	if wantContext && isTTY(os.Stdin) {
		// Иначе выглядит как зависание: программа молча ждёт EOF.
		info("читаю stdin, Ctrl-D когда закончишь")
	}
	if wantContext || !isTTY(os.Stdin) {
		context = readStdin()
		detail("контекст из stdin: %d байт", len(context))
	}

	userContent := question
	if context != "" {
		userContent = fmt.Sprintf("Контекст:\n%s\n\nВопрос: %s", context, question)
	}

	var history []chatMessage
	if resume {
		history = loadSession(profileName)
	}

	sysMsg := chatMessage{Role: "system", Content: buildSystemPrompt(profile.UseTools)}
	userMsg := chatMessage{Role: "user", Content: userContent}

	messages := make([]chatMessage, 0, len(history)+2)
	messages = append(messages, sysMsg)
	messages = append(messages, history...)
	messages = append(messages, userMsg)

	// turnMessages — то, что уйдёт на диск как новая сессия: старая
	// история (если resume) + всё, что произошло в этом ходе, без
	// system-сообщения (оно всегда пересобирается заново на след. раз,
	// т.к. cwd/дата могли измениться).
	turnMessages := make([]chatMessage, 0, len(history)+4)
	turnMessages = append(turnMessages, history...)
	turnMessages = append(turnMessages, userMsg)

	// Сессию сохраняем на любом выходе, в том числе аварийном: иначе
	// упавшая на пятом шаге сеть уносит с собой журнал уже выполненных
	// команд, и -r восстанавливает состояние «до начала».
	finish := func(code int) int {
		if err := saveSession(profileName, turnMessages); err != nil {
			warn("не смог сохранить сессию: %v", err)
		}
		return code
	}

	var (
		final        string
		result       = outcomeAnswer
		finishReason string
	)

loop:
	for i := 0; ; i++ {
		sp := startSpinner("думаю")
		res, err := chatWithFallback(profile, messages, profile.UseTools)
		sp.stopSpinner()
		if err != nil {
			fail("%v", err)
			return finish(exitAPI)
		}
		detail("шаг %d: ответила %s, finish_reason=%s", i+1, res.model, valueOr(res.finishReason, "-"))

		messages = append(messages, res.msg)
		turnMessages = append(turnMessages, res.msg)
		finishReason = res.finishReason

		if len(res.msg.ToolCalls) == 0 {
			final = res.msg.Content
			if strings.TrimSpace(final) == "" {
				result = outcomeEmpty
			}
			break
		}

		for _, tc := range res.msg.ToolCalls {
			toolMsg, verdict := runToolCall(&cf, profile, tc)
			messages = append(messages, toolMsg)
			turnMessages = append(turnMessages, toolMsg)

			switch verdict {
			case verdictDeclined:
				final = res.msg.Content
				result = outcomeDeclined
				break loop
			case verdictInterrupted:
				final = res.msg.Content
				result = outcomeInterrupted
				break loop
			}
		}
	}

	if finishReason == "length" {
		warn("ответ обрезан сервером по лимиту токенов — переспроси короче или задай вопрос по частям")
	}

	code := exitOK
	switch result {
	case outcomeDeclined:
		info("выполнение отменено")
		if strings.TrimSpace(final) == "" {
			final = "(команда не подтверждена, ход прерван)"
		}
		code = exitDeclined
	case outcomeInterrupted:
		info("прервано (Ctrl-C)")
		if strings.TrimSpace(final) == "" {
			final = "(выполнение прервано)"
		}
		code = exitDeclined
	case outcomeEmpty:
		final = "(модель вернула пустой ответ)"
		code = exitNoAnswer
	}

	fmt.Println(final)
	return finish(code)
}

type toolVerdict int

const (
	verdictOK toolVerdict = iota
	verdictInvalid
	verdictDeclined
	verdictInterrupted
)

// runToolCall проверяет запрос модели и, если пользователь согласен,
// выполняет команду. Возвращает сообщение с ролью tool — то, что уйдёт
// обратно модели.
func runToolCall(cf *ConfigFile, p Profile, tc toolCall) (chatMessage, toolVerdict) {
	mk := func(content string) chatMessage {
		return chatMessage{Role: "tool", ToolCallID: tc.ID, Content: content}
	}

	// Инструмент объявлен ровно один. Раньше имя не проверялось вообще:
	// что бы модель ни назвала, поле command уходило в sh -c.
	if tc.Function.Name != shellToolName {
		warn("модель просит неизвестный инструмент %q — не выполняю",
			sanitizeForDisplay(tc.Function.Name))
		return mk(fmt.Sprintf("ошибка: инструмента %q не существует, доступен только %s",
			tc.Function.Name, shellToolName)), verdictInvalid
	}

	var parsedArgs struct {
		Command string `json:"command"`
	}
	// Ошибку разбора раньше выбрасывали (_ =), и после неудачи в шелл
	// уходила пустая строка: sh -c "" отрабатывает с exit 0, и модель
	// считала шаг успешно выполненным.
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &parsedArgs); err != nil {
		warn("аргументы инструмента не разобрать: %v", err)
		return mk(fmt.Sprintf("ошибка: аргументы должны быть JSON вида {\"command\": \"...\"}, разбор не удался: %v", err)), verdictInvalid
	}

	cmdStr := strings.TrimSpace(parsedArgs.Command)
	if cmdStr == "" {
		warn("модель прислала пустую команду — не выполняю")
		return mk("ошибка: поле command пустое, команда не выполнялась"), verdictInvalid
	}

	name := commandName(cmdStr)
	simple := isSimpleCommand(cmdStr)
	knownButComplex := !simple && cf.isAllowed(name)

	if simple && cf.isAllowed(name) {
		info("выполняю без вопроса (%s в списке разрешённых): %s", name, sanitizeForDisplay(cmdStr))
	} else {
		switch confirmCommand(cmdStr, name, knownButComplex) {
		case ansNo:
			return mk("пользователь отказался выполнять эту команду"), verdictDeclined
		case ansAlways:
			cf.allow(name)
			if err := saveConfigFile(*cf); err != nil {
				warn("не смог сохранить список разрешённых: %v", err)
			} else {
				info("%s добавлена в разрешённые (убрать: clank config allow-rm %s)", name, name)
			}
		}
	}

	// Маркеры в stderr, сам вывод команды — в stdout: при `clank ... > файл`
	// вывод выполненных команд считается такой же частью результата, как
	// и ответ модели.
	fmt.Fprintln(os.Stderr, "--- вывод ---")
	res := execShell(cmdStr, p.execTimeout())
	fmt.Fprintf(os.Stderr, "--- exit %d ---\n", res.exitCode)

	var b strings.Builder
	switch {
	case res.interrupted:
		b.WriteString("команда прервана пользователем (Ctrl-C)\n")
	case res.timedOut:
		b.WriteString("команда снята по таймауту\n")
	}
	fmt.Fprintf(&b, "exit code: %d\n%s", res.exitCode, res.output)

	verdict := verdictOK
	if res.interrupted {
		verdict = verdictInterrupted
	}
	return mk(b.String()), verdict
}

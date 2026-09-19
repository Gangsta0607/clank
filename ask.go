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
		yoloFlag    *bool
	)
	for _, a := range args {
		if literal {
			rest = append(rest, a)
			continue
		}
		switch {
		case a == "--":
			literal = true
		case a == "-c" || a == "--context":
			wantContext = true
		case a == "-r" || a == "--resume":
			resume = true
		case a == "-v" || a == "--verbose":
			verboseLevel = vVerbose
		case a == "-q" || a == "--quiet":
			verboseLevel = vQuiet
		case a == "-h" || a == "--help":
			usage()
			return exitOK
		case a == "--yolo" || a == "--yolo=on" || a == "--yolo=true":
			v := true
			yoloFlag = &v
		case a == "--yolo=off" || a == "--yolo=false":
			v := false
			yoloFlag = &v
		default:
			if strings.HasPrefix(a, "--yolo=") {
				fail("неверное значение флага --yolo: %s (ожидается on или off)", a)
				return exitConfig
			}
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

	var (
		history     []chatMessage
		sessionYolo *bool
	)
	if resume {
		history, sessionYolo = loadSession(profileName)
	}

	effectiveYolo := cf.Yolo
	if sessionYolo != nil {
		effectiveYolo = *sessionYolo
	}
	if yoloFlag != nil {
		effectiveYolo = *yoloFlag
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
		if err := saveSession(profileName, turnMessages, &effectiveYolo); err != nil {
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
			toolMsg, verdict := runToolCall(&cf, profile, tc, effectiveYolo)
			messages = append(messages, toolMsg)
			turnMessages = append(turnMessages, toolMsg)

			if verdict == verdictInterrupted {
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
func runToolCall(cf *ConfigFile, p Profile, tc toolCall, yolo bool) (chatMessage, toolVerdict) {
	mk := func(content string) chatMessage {
		return chatMessage{Role: "tool", ToolCallID: tc.ID, Content: content}
	}

	switch tc.Function.Name {
	case shellToolName:
		return runShellToolCall(cf, p, tc, yolo, mk)
	case questionToolName, "input":
		return runQuestionToolCall(tc, mk)
	default:
		warn("модель просит неизвестный инструмент %q — не выполняю",
			sanitizeForDisplay(tc.Function.Name))
		return mk(fmt.Sprintf("ошибка: инструмента %q не существует, доступны: %s, %s",
			tc.Function.Name, shellToolName, questionToolName)), verdictInvalid
	}
}

func runQuestionToolCall(tc toolCall, mk func(string) chatMessage) (chatMessage, toolVerdict) {
	var qArgs struct {
		Question string   `json:"question"`
		Prompt   string   `json:"prompt"`
		Text     string   `json:"text"`
		Options  []string `json:"options"`
		Default  string   `json:"default"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &qArgs); err != nil {
		warn("аргументы вопроса не разобрать: %v", err)
		return mk(fmt.Sprintf("ошибка: аргументы вопроса должны быть JSON вида {\"question\": \"...\"}, разбор не удался: %v", err)), verdictInvalid
	}

	qText := strings.TrimSpace(qArgs.Question)
	if qText == "" {
		qText = strings.TrimSpace(qArgs.Prompt)
	}
	if qText == "" {
		qText = strings.TrimSpace(qArgs.Text)
	}
	if qText == "" {
		warn("модель прислала пустой вопрос")
		return mk("ошибка: поле question пустое"), verdictInvalid
	}

	ans, ok := promptQuestion(qText, qArgs.Options, strings.TrimSpace(qArgs.Default))
	if !ok {
		if !haveTTY() && qArgs.Default == "" {
			return mk("ошибка: нет управляющего терминала для ответа на вопрос"), verdictInvalid
		}
		return mk("пользователь отменил ввод (Ctrl-C / EOF)"), verdictInterrupted
	}

	return mk(ans), verdictOK
}

func runShellToolCall(cf *ConfigFile, p Profile, tc toolCall, yolo bool, mk func(string) chatMessage) (chatMessage, toolVerdict) {
	var parsedArgs struct {
		Command string `json:"command"`
	}
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
	trust := cf.trustLevel(name)

	if yolo {
		info("выполняю без вопроса (режим YOLO): %s", sanitizeForDisplay(cmdStr))
	} else if trust == TrustAll {
		info("выполняю без вопроса (%s разрешена всегда): %s", name, sanitizeForDisplay(cmdStr))
	} else if trust == TrustSimple && simple {
		info("выполняю без вопроса (%s разрешена для простых команд): %s", name, sanitizeForDisplay(cmdStr))
	} else {
		switch confirmCommand(cmdStr, name, trust) {
		case ansNo:
			return mk("пользователь отказался выполнять эту команду"), verdictDeclined
		case ansAlways:
			level := TrustSimple
			if !simple {
				level = TrustAll
			}
			cf.allow(name, level)
			if err := saveConfigFile(*cf); err != nil {
				warn("не смог сохранить список разрешённых: %v", err)
			} else {
				if level == TrustAll {
					info("%s теперь разрешена всегда, включая сложные команды (убрать: clank config allow-rm %s)", name, name)
				} else {
					info("%s добавлена в разрешённые для простых команд (убрать: clank config allow-rm %s)", name, name)
				}
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

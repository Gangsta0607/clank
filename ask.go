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
		warn(M.StdinReadFail, err)
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
		reasonFlag  *bool
		imagePaths  []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
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
		case a == "-i" || a == "--image":
			if i+1 >= len(args) {
				fail(M.ImageFlagNeeds, a)
				return exitConfig
			}
			i++
			imagePaths = append(imagePaths, args[i])
		case strings.HasPrefix(a, "--image="):
			imagePaths = append(imagePaths, strings.TrimPrefix(a, "--image="))
		case a == "--yolo" || a == "--yolo=on" || a == "--yolo=true":
			v := true
			yoloFlag = &v
		case a == "--yolo=off" || a == "--yolo=false":
			v := false
			yoloFlag = &v
		case a == "--reasoning" || a == "--reasoning=on" || a == "--reasoning=true" || a == "--reasoning=1":
			v := true
			reasonFlag = &v
		case a == "--reasoning=off" || a == "--reasoning=false" || a == "--reasoning=0":
			v := false
			reasonFlag = &v
		default:
			if strings.HasPrefix(a, "--yolo=") {
				fail(M.BadYoloFlag, a)
				return exitConfig
			}
			if strings.HasPrefix(a, "--reasoning=") {
				fail(M.BadReasonFlag, a)
				return exitConfig
			}
			if strings.HasPrefix(a, "-") && len(a) > 1 {
				fail(M.UnknownFlag, a)
				return exitConfig
			}
			rest = append(rest, a)
		}
	}

	question := strings.Join(rest, " ")
	if strings.TrimSpace(question) == "" {
		fail(M.NothingToAsk)
		fmt.Fprintln(os.Stderr, M.AskUsage)
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
	if reasonFlag != nil {
		profile.Reasoning = reasonFlag
	}
	detail(M.ProfileDetail, profileName, strings.Join(profile.Models, " → "))

	gcSessions()

	var context string
	if wantContext && isTTY(os.Stdin) {
		// Иначе выглядит как зависание: программа молча ждёт EOF.
		info(M.ReadingStdin)
	}
	if wantContext || !isTTY(os.Stdin) {
		context = readStdin()
		detail(M.StdinBytes, len(context))
	}

	userContent := question
	if context != "" {
		userContent = fmt.Sprintf(M.ContextWrap, context, question)
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

	sysMsg := chatMessage{Role: "system", Content: buildSystemPrompt(profile.UseTools, profile.Vision == nil || *profile.Vision)}
	var userMsg chatMessage
	if len(imagePaths) > 0 {
		parts := []contentPart{
			{Type: "text", Text: userContent},
		}
		for _, imgPath := range imagePaths {
			dataURL, err := encodeImageFile(imgPath)
			if err != nil {
				fail("%v", err)
				return exitConfig
			}
			parts = append(parts, contentPart{
				Type:     "image_url",
				ImageURL: &imageURL{URL: dataURL},
			})
			detail(M.ImageAttached, imgPath, len(dataURL))
		}
		userMsg = chatMessage{Role: "user", Content: parts}
	} else {
		userMsg = chatMessage{Role: "user", Content: userContent}
	}

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
			warn(M.SessionSaveErr, err)
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
		sp := startSpinner(M.Thinking)
		res, err := chatWithFallback(profile, messages, profile.UseTools)
		sp.stopSpinner()
		if err != nil {
			fail("%v", err)
			return finish(exitAPI)
		}
		detail(M.StepDetail, i+1, res.model, valueOr(res.finishReason, "-"))

		messages = append(messages, res.msg)
		turnMessages = append(turnMessages, res.msg)
		finishReason = res.finishReason

		if len(res.msg.ToolCalls) == 0 {
			final = res.msg.Text()
			if strings.TrimSpace(final) == "" {
				result = outcomeEmpty
			}
			break
		}

		for _, tc := range res.msg.ToolCalls {
			toolMsg, followUp, verdict := runToolCall(&cf, profile, tc, effectiveYolo)
			messages = append(messages, toolMsg)
			turnMessages = append(turnMessages, toolMsg)
			if followUp != nil {
				messages = append(messages, *followUp)
				turnMessages = append(turnMessages, *followUp)
			}

			if verdict == verdictInterrupted {
				final = res.msg.Text()
				result = outcomeInterrupted
				break loop
			}
		}
	}

	if finishReason == "length" {
		warn(M.AnswerCut)
	}

	code := exitOK
	switch result {
	case outcomeInterrupted:
		info(M.Interrupted)
		if strings.TrimSpace(final) == "" {
			final = M.OutcomeIntr
		}
		code = exitDeclined
	case outcomeEmpty:
		final = M.OutcomeEmpty
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
func runToolCall(cf *ConfigFile, p Profile, tc toolCall, yolo bool) (chatMessage, *chatMessage, toolVerdict) {
	mk := func(content string) chatMessage {
		return chatMessage{Role: "tool", ToolCallID: tc.ID, Content: content}
	}

	switch tc.Function.Name {
	case shellToolName:
		msg, verd := runShellToolCall(cf, p, tc, yolo, mk)
		return msg, nil, verd
	case questionToolName, "input":
		msg, verd := runQuestionToolCall(tc, mk)
		return msg, nil, verd
	case viewImageToolName:
		return runViewImageToolCall(tc, mk)
	default:
		warn(M.UnknownTool,
			sanitizeForDisplay(tc.Function.Name))
		return mk(fmt.Sprintf(M.NoSuchTool,
			tc.Function.Name, shellToolName, questionToolName, viewImageToolName)), nil, verdictInvalid
	}
}

func runViewImageToolCall(tc toolCall, mk func(string) chatMessage) (chatMessage, *chatMessage, toolVerdict) {
	var parsedArgs struct {
		Path string `json:"path"`
		File string `json:"file"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &parsedArgs); err != nil {
		warn(M.ViewArgsBad, err)
		return mk(fmt.Sprintf(M.ViewArgsNeed, err)), nil, verdictInvalid
	}

	imgPath := strings.TrimSpace(parsedArgs.Path)
	if imgPath == "" {
		imgPath = strings.TrimSpace(parsedArgs.File)
	}
	if imgPath == "" {
		warn(M.ViewEmptyPath)
		return mk(M.ViewPathEmpty), nil, verdictInvalid
	}

	info(M.ViewWatch, sanitizeForDisplay(imgPath))
	dataURL, err := encodeImageFile(imgPath)
	if err != nil {
		warn(M.ViewOpenFail, imgPath, err)
		return mk(fmt.Sprintf(M.ViewReadFail, imgPath, err)), nil, verdictInvalid
	}

	toolMsg := mk(fmt.Sprintf(M.ViewLoaded, imgPath))
	followUp := &chatMessage{
		Role: "user",
		Content: []contentPart{
			{Type: "text", Text: fmt.Sprintf(M.ViewContent, imgPath)},
			{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
		},
	}
	return toolMsg, followUp, verdictOK
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
		warn(M.AskArgsBad, err)
		return mk(fmt.Sprintf(M.AskArgsNeed, err)), verdictInvalid
	}

	qText := strings.TrimSpace(qArgs.Question)
	if qText == "" {
		qText = strings.TrimSpace(qArgs.Prompt)
	}
	if qText == "" {
		qText = strings.TrimSpace(qArgs.Text)
	}
	if qText == "" {
		warn(M.AskEmptyQ)
		return mk(M.AskQEmpty), verdictInvalid
	}

	ans, ok := promptQuestion(qText, qArgs.Options, strings.TrimSpace(qArgs.Default))
	if !ok {
		if !haveTTY() && qArgs.Default == "" {
			return mk(M.AskNoTTY), verdictInvalid
		}
		return mk(M.AskCancelled), verdictInterrupted
	}

	return mk(ans), verdictOK
}

func runShellToolCall(cf *ConfigFile, p Profile, tc toolCall, yolo bool, mk func(string) chatMessage) (chatMessage, toolVerdict) {
	var parsedArgs struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &parsedArgs); err != nil {
		warn(M.ShellArgsBad, err)
		return mk(fmt.Sprintf(M.ShellArgsNeed, err)), verdictInvalid
	}

	cmdStr := strings.TrimSpace(parsedArgs.Command)
	if cmdStr == "" {
		warn(M.ShellEmpty)
		return mk(M.ShellCmdEmpty), verdictInvalid
	}

	name := commandName(cmdStr)
	simple := isSimpleCommand(cmdStr)
	trust := cf.trustLevel(name)

	if yolo {
		info(M.RunYolo, sanitizeForDisplay(cmdStr))
	} else if trust == TrustAll {
		info(M.RunAlways, name, sanitizeForDisplay(cmdStr))
	} else if trust == TrustSimple && simple {
		info(M.RunSimple, name, sanitizeForDisplay(cmdStr))
	} else {
		switch confirmCommand(cmdStr, name, trust) {
		case ansNo:
			return mk(M.UserDeclined), verdictDeclined
		case ansAlways:
			level := TrustSimple
			if !simple {
				level = TrustAll
			}
			cf.allow(name, level)
			if err := saveConfigFile(*cf); err != nil {
				warn(M.AllowSaveFail, err)
			} else {
				if level == TrustAll {
					info(M.AllowAll, name, name)
				} else {
					info(M.AllowSimple, name, name)
				}
			}
		}
	}

	// Маркеры в stderr, сам вывод команды — в stdout: при `clank ... > файл`
	// вывод выполненных команд считается такой же частью результата, как
	// и ответ модели.
	fmt.Fprintln(os.Stderr, M.OutputMark)
	res := execShell(cmdStr, p.execTimeout())
	fmt.Fprintf(os.Stderr, "--- exit %d ---\n", res.exitCode)

	var b strings.Builder
	switch {
	case res.interrupted:
		b.WriteString(M.CmdInterrupted)
	case res.timedOut:
		b.WriteString(M.CmdTimedOut)
	}
	fmt.Fprintf(&b, "exit code: %d\n%s", res.exitCode, res.output)

	verdict := verdictOK
	if res.interrupted {
		verdict = verdictInterrupted
	}
	return mk(b.String()), verdict
}

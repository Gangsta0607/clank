package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func cmdConfig(args []string) int {
	if len(args) == 0 {
		printConfigUsage()
		return exitConfig
	}

	cf, err := loadConfigFile()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}

	switch args[0] {
	case "-h", "--help", "help":
		printConfigUsage()
		return exitOK

	case "init":
		return cmdConfigAdd(&cf, "default")

	case "add":
		if len(args) < 2 {
			fail(M.NeedProfileName, "add")
			return exitConfig
		}
		return cmdConfigAdd(&cf, args[1])

	case "use":
		if len(args) < 2 {
			fail(M.NeedProfileName, "use")
			return exitConfig
		}
		if _, ok := cf.Profiles[args[1]]; !ok {
			fail(M.NoSuchProfileList, args[1], strings.Join(profileNames(cf), ", "))
			return exitConfig
		}
		cf.Active = args[1]
		if err := saveConfigFile(cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.ActiveProfileIs, args[1])
		return exitOK

	case "list":
		if len(cf.Profiles) == 0 {
			fmt.Println(M.NoProfiles)
			return exitOK
		}
		for _, n := range profileNames(cf) {
			mark := "  "
			if n == cf.Active {
				mark = "* "
			}
			p := cf.Profiles[n]
			fmt.Printf("%s%-15s tools=%-5v url=%-40s models=%s\n",
				mark, n, p.UseTools, valueOr(p.BaseURL, "-"), valueOr(strings.Join(p.Models, ", "), "-"))
		}
		return exitOK

	case "rm":
		if len(args) < 2 {
			fail(M.NeedProfileName, "rm")
			return exitConfig
		}
		if _, ok := cf.Profiles[args[1]]; !ok {
			fail(M.NoSuchProfile, args[1])
			return exitConfig
		}
		delete(cf.Profiles, args[1])
		if cf.Active == args[1] {
			cf.Active = ""
			if len(cf.Profiles) > 0 {
				cf.Active = newestProfileName(cf)
				info(M.ActiveRemoved, cf.Active)
			}
		}
		if err := saveConfigFile(cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.RemovedIs, args[1])
		return exitOK

	case "show":
		name, p, err := activeProfile(&cf)
		if err != nil {
			fail("%v", err)
			return exitConfig
		}
		path, _ := configPath()
		fmt.Println(M.ShowFile, path)
		fmt.Println(M.ShowProfile, name)
		fmt.Println("base_url:  ", valueOr(p.BaseURL, M.ShowNotSet))
		fmt.Println("api_key:   ", maskKey(p.APIKey))
		if len(p.Models) == 0 {
			fmt.Println(M.ShowModelsNone)
		} else {
			fmt.Println(M.ShowModelsHead)
			for i, m := range p.Models {
				fmt.Printf("             %d) %s\n", i+1, m)
			}
		}
		fmt.Println("proxy:     ", valueOr(p.Proxy, M.ShowNoEnvProxy))
		fmt.Println("use_tools: ", p.UseTools)
		if p.Reasoning != nil {
			state := M.ShowThinkOff
			if *p.Reasoning {
				state = M.ShowThinkOn
			}
			var details []string
			if p.ReasoningEffort != "" {
				details = append(details, "effort: "+p.ReasoningEffort)
			}
			if p.ReasoningBudget > 0 {
				details = append(details, fmt.Sprintf("budget: %d", p.ReasoningBudget))
			}
			if len(details) > 0 {
				fmt.Printf(M.ShowThinkCal, state, strings.Join(details, ", "))
			} else {
				fmt.Println(M.ShowThinkPlain, state)
			}
		} else {
			fmt.Println(M.ShowThinkAuto)
		}
		if p.Vision != nil {
			if *p.Vision {
				fmt.Println(M.ShowVisionOn)
			} else {
				fmt.Println(M.ShowVisionOff)
			}
		} else {
			fmt.Println(M.ShowVisionAuto)
		}
		fmt.Println(M.ShowTimeout, p.execTimeout())
		if cf.Yolo {
			fmt.Println(M.ShowYoloOn)
		}
		fmt.Println(M.ShowLanguage, showLanguage(cf.Language))
		if len(cf.Allowed) > 0 {
			var items []string
			for k, lvl := range cf.Allowed {
				items = append(items, fmt.Sprintf("%s (%s)", k, lvl))
			}
			sort.Strings(items)
			fmt.Println(M.ShowAllowed, strings.Join(items, ", "))
		}
		return exitOK

	case "set-url", "set-key", "set-model", "set-models", "set-proxy", "set-tools", "set-reasoning", "set-exec-timeout":
		if len(args) < 2 {
			fail(M.NeedValue, args[0])
			return exitConfig
		}
		name, p, err := activeProfile(&cf)
		if err != nil {
			fail("%v", err)
			return exitConfig
		}
		val := args[1]
		switch args[0] {
		case "set-url":
			p.BaseURL = normalizeBaseURL(val)
		case "set-key":
			p.APIKey = val
		case "set-model", "set-models":
			// список через запятую = цепочка фоллбэка
			var models []string
			for _, m := range strings.Split(val, ",") {
				if m = strings.TrimSpace(m); m != "" {
					models = append(models, m)
				}
			}
			if len(models) == 0 {
				fail(M.ModelsEmpty)
				return exitConfig
			}
			p.Models = models
		case "set-proxy":
			if val == "-" {
				p.Proxy = ""
			} else {
				p.Proxy = val
			}
		case "set-tools":
			p.UseTools = val == "true" || val == "1" || val == "y" || val == "yes"
		case "set-reasoning":
			switch strings.ToLower(val) {
			case "on", "1", "true", "yes", "вкл":
				v := true
				p.Reasoning = &v
			case "off", "0", "false", "no", "выкл":
				v := false
				p.Reasoning = &v
			case "-", "none", "auto", "default", "сброс":
				p.Reasoning = nil
				p.ReasoningEffort = ""
				p.ReasoningBudget = 0
			default:
				fail(M.BadSetReason, val)
				return exitConfig
			}
		case "set-exec-timeout":
			secs, err := strconv.Atoi(val)
			if err != nil || secs < 0 {
				fail(M.BadTimeout)
				return exitConfig
			}
			p.ExecTimeoutSec = secs
		}
		cf.Profiles[name] = p
		if err := saveConfigFile(cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.SavedOK)
		return exitOK

	case "test-tools":
		name, _, err := activeProfile(&cf)
		if err != nil {
			fail("%v", err)
			return exitConfig
		}
		return cmdTestTools(&cf, name)

	case "test-reasoning":
		name, _, err := activeProfile(&cf)
		if err != nil {
			fail("%v", err)
			return exitConfig
		}
		return cmdTestReasoning(&cf, name)

	case "test-vision":
		name, _, err := activeProfile(&cf)
		if err != nil {
			fail("%v", err)
			return exitConfig
		}
		return cmdTestVision(&cf, name)

	case "test-model":
		name, _, err := activeProfile(&cf)
		if err != nil {
			fail("%v", err)
			return exitConfig
		}
		return cmdTestModel(&cf, name)

	case "allow-rm":
		if len(args) < 2 {
			fail(M.AllowNeedName)
			return exitConfig
		}
		if !cf.disallow(args[1]) {
			fail(M.AllowNotListed, args[1])
			return exitConfig
		}
		if err := saveConfigFile(cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.AllowRemoved, args[1])
		return exitOK

	case "allow-clear":
		cf.Allowed = nil
		if err := saveConfigFile(cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.AllowCleared)
		return exitOK

	default:
		fail(M.UnknownSubcommand, args[0])
		printConfigUsage()
		return exitConfig
	}
}

// cmdConfigAdd — мастер настройки профиля. Работает именно с тем профилем,
// который создаёт: раньше он звал cmdModels/cmdTestTools, а те брали
// активный профиль, и `config add work` при активном default молча
// перенастраивал default.
func cmdConfigAdd(cf *ConfigFile, name string) int {
	if !haveTTY() {
		fail(M.WizardNeedsTTY)
		return exitConfig
	}

	existing, exists := cf.Profiles[name]
	if !exists {
		existing.Created = time.Now().Format(time.RFC3339)
	}

	fmt.Printf(M.WizardTitle+"\n", name)
	existing.BaseURL = normalizeBaseURL(askLineDefault("base_url", existing.BaseURL, existing.BaseURL))
	// Ключ показываем замаскированным: иначе он остаётся в скроллбэке
	// терминала и в любой записи экрана.
	existing.APIKey = askLineDefault("api_key", existing.APIKey, maskKey(existing.APIKey))

	if cf.Profiles == nil {
		cf.Profiles = map[string]Profile{}
	}
	cf.Profiles[name] = existing
	if cf.Active == "" {
		cf.Active = name
	}
	if err := saveConfigFile(*cf); err != nil {
		fail(M.ConfigSaveFail, err)
		return exitConfig
	}
	fmt.Println(M.ProfileSaved, name)

	if confirmYN(M.AskFetchModels, true) {
		if code := selectModels(cf, name, ""); code != exitOK {
			warn(M.ModelsFetchFail)
		}
	}

	// Раньше при отказе или сетевой ошибке профиль оставался без модели,
	// и первый же вопрос падал на проверке конфига.
	if len(cf.Profiles[name].Models) == 0 {
		line, ok := askLine(M.AskModelLine)
		if ok && strings.TrimSpace(line) != "" {
			p := cf.Profiles[name]
			for _, m := range strings.Split(line, ",") {
				if m = strings.TrimSpace(m); m != "" {
					p.Models = append(p.Models, m)
				}
			}
			cf.Profiles[name] = p
			if err := saveConfigFile(*cf); err != nil {
				fail(M.ConfigSaveFail, err)
				return exitConfig
			}
		}
	}

	if len(cf.Profiles[name].Models) == 0 {
		warn(M.ProfileNoModel)
		return exitOK
	}

	if confirmYN(M.AskTestTools, true) {
		return cmdTestTools(cf, name)
	}

	fmt.Println(M.ToolsSkipLater)
	return exitOK
}

func cmdModels(args []string) int {
	var filter string
	for _, a := range args {
		switch a {
		case "-v", "--verbose":
			verboseLevel = vVerbose
		case "-q", "--quiet":
			verboseLevel = vQuiet
		case "-h", "--help":
			fmt.Fprintln(os.Stderr, M.ModelsHelp)
			return exitOK
		default:
			filter = a
		}
	}

	cf, err := loadConfigFile()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	name, _, err := activeProfile(&cf)
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	return selectModels(&cf, name, filter)
}

// selectModels показывает модели конкретного профиля и, если есть куда
// спросить, даёт выбрать цепочку фоллбэка.
func selectModels(cf *ConfigFile, name, filter string) int {
	p := cf.Profiles[name]
	if p.BaseURL == "" || p.APIKey == "" {
		fail(M.ModelsNeedCreds, name)
		return exitConfig
	}

	sp := startSpinner(M.FetchingModels)
	ids, err := listModels(p)
	sp.stopSpinner()
	if err != nil {
		fail("%v", err)
		return exitAPI
	}

	if filter != "" {
		var kept []string
		for _, id := range ids {
			if strings.Contains(strings.ToLower(id), strings.ToLower(filter)) {
				kept = append(kept, id)
			}
		}
		ids = kept
	}
	if len(ids) == 0 {
		if filter != "" {
			fail(M.FilterNoMatch, filter)
		} else {
			fail(M.ModelsEmptyAPI)
		}
		return exitAPI
	}
	sort.Strings(ids)

	// В пайпе интерактив бессмысленен: раньше чтение номера уходило в
	// тот же пайп, откуда читался контекст.
	if !isTTY(os.Stdout) || !haveTTY() {
		for _, id := range ids {
			fmt.Println(id)
		}
		return exitOK
	}

	current := map[string]int{}
	for i, m := range p.Models {
		current[m] = i + 1
	}

	cells := make([]string, len(ids))
	for i, id := range ids {
		mark := " "
		if n, ok := current[id]; ok {
			mark = strconv.Itoa(n) // текущая позиция в цепочке
		}
		cells[i] = fmt.Sprintf("%s%3d) %s", mark, i+1, id)
	}
	for _, line := range layoutColumns(cells, termWidth()) {
		fmt.Println(line)
	}
	if len(current) > 0 {
		fmt.Println(M.ChainHint)
	}

	line, ok := askLine(M.AskChainNumbers)
	if !ok || line == "" {
		fmt.Println(M.Cancelled)
		return exitOK
	}

	var chosen []string
	seen := map[string]bool{}
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(ids) {
			fail(M.BadNumber, part, len(ids))
			return exitConfig
		}
		if !seen[ids[n-1]] {
			seen[ids[n-1]] = true
			chosen = append(chosen, ids[n-1])
		}
	}
	if len(chosen) == 0 {
		fmt.Println(M.Cancelled)
		return exitOK
	}

	p.Models = chosen
	cf.Profiles[name] = p
	if err := saveConfigFile(*cf); err != nil {
		fail(M.ConfigSaveFail, err)
		return exitConfig
	}
	if len(chosen) == 1 {
		fmt.Println(M.ModelSet, chosen[0])
	} else {
		fmt.Println(M.ChainSet, strings.Join(chosen, " → "))
	}
	return exitOK
}

func cmdTestTools(cf *ConfigFile, name string) int {
	p := cf.Profiles[name]
	if err := p.validate(); err != nil {
		fail("%v", err)
		return exitConfig
	}

	model := p.Models[0]
	fmt.Println(M.TestToolsCheck, model, "...")
	sp := startSpinner(M.WaitingAnswer)
	ok, detailText, err := testToolSupport(p, model)
	sp.stopSpinner()
	if err != nil {
		fail("%v", err)
		return exitAPI
	}
	if ok {
		fmt.Println(M.ToolCalled, sanitizeForDisplay(truncateRunes(detailText, 300)))
	} else {
		fmt.Println(M.ToolNotCalled, truncateRunes(detailText, 300))
	}
	if len(p.Models) > 1 {
		info(M.ChainFirstOnly)
	}

	if confirmYN(fmt.Sprintf(M.AskSaveTools, ok, name), true) {
		p.UseTools = ok
		cf.Profiles[name] = p
		if err := saveConfigFile(*cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.SavedOK)
	} else {
		fmt.Println(M.SavedNot)
	}
	return exitOK
}

func cmdTestReasoning(cf *ConfigFile, name string) int {
	p := cf.Profiles[name]
	if err := p.validate(); err != nil {
		fail("%v", err)
		return exitConfig
	}

	model := p.Models[0]
	fmt.Println(M.TestReasonCheck, model, "...")
	sp := startSpinner(M.TestingParams)
	res, err := testReasoningSupport(p, model)
	sp.stopSpinner()
	if err != nil {
		fail("%v", err)
		return exitAPI
	}

	fmt.Println(M.ReasonResults)
	printReasoningResult(res)

	supported := res.EffortOn != "" || res.EffortOff != "" || res.UsesBudget || res.HasOutput
	if !supported {
		info(M.ReasonUnsupported)
		return exitOK
	}

	if confirmYN(fmt.Sprintf(M.AskReasonOn, name), true) {
		v := true
		p.Reasoning = &v
		if res.EffortOn != "" {
			p.ReasoningEffort = res.EffortOn
		}
		if res.UsesBudget {
			p.ReasoningBudget = res.BudgetOn
		}
		cf.Profiles[name] = p
		if err := saveConfigFile(*cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.SavedReasonOn)
	} else {
		fmt.Println(M.SettingsKept)
	}
	return exitOK
}

// printReasoningResult печатает итог проверки reasoning нейтрально:
// что поддерживается и найден ли блок — без verdict'ов.
func printReasoningResult(res reasoningTestResult) {
	if res.EffortOn != "" || res.EffortOff != "" {
		fmt.Printf(M.ReasonEffortOK,
			valueOr(res.EffortOn, M.ReasonNone), valueOr(res.EffortOff, M.ReasonNone))
	} else if res.UsesBudget {
		fmt.Println(M.ReasonBudgetOK)
	} else {
		fmt.Println(M.ReasonNoParams)
	}

	if res.HasOutput {
		fmt.Println(M.ReasonHasOutput)
	} else {
		fmt.Println(M.ReasonNoOutput)
	}
}

func cmdTestVision(cf *ConfigFile, name string) int {
	p := cf.Profiles[name]
	if err := p.validate(); err != nil {
		fail("%v", err)
		return exitConfig
	}

	model := p.Models[0]
	fmt.Println(M.TestVisionCheck, model, "...")
	sp := startSpinner(M.WaitingAnswer)
	ok, detailText, err := testVisionSupport(p, model)
	sp.stopSpinner()
	if err != nil {
		fail("%v", err)
		return exitAPI
	}
	if ok {
		fmt.Println(M.VisionOK, sanitizeForDisplay(truncateRunes(detailText, 300)))
		fmt.Println(M.VisionJudge)
	} else {
		fmt.Println(M.VisionFail)
	}
	if confirmYN(fmt.Sprintf(M.AskSaveVision, ok, name), true) {
		p.Vision = &ok
		cf.Profiles[name] = p
		if err := saveConfigFile(*cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.SavedOK)
	} else {
		fmt.Println(M.SavedNot)
	}
	return exitOK
}

// cmdTestModel гоняет все три проверки разом (tools, reasoning, vision)
// и сохраняет всё одним вопросом. Отчёты нейтральные: факты плюс ответ
// модели, выводы — на пользователе.
func cmdTestModel(cf *ConfigFile, name string) int {
	p := cf.Profiles[name]
	if err := p.validate(); err != nil {
		fail("%v", err)
		return exitConfig
	}

	model := p.Models[0]
	fmt.Printf(M.TestModelCheck+"\n", model)

	sp := startSpinner(M.WaitingAnswer)
	toolsOK, toolsDetail, err := testToolSupport(p, model)
	sp.stopSpinner()
	if err != nil {
		fail("%v", err)
		return exitAPI
	}
	if toolsOK {
		fmt.Println(M.ToolCalled, sanitizeForDisplay(truncateRunes(toolsDetail, 300)))
	} else {
		fmt.Println(M.ToolNotCalled, truncateRunes(toolsDetail, 300))
	}

	sp = startSpinner(M.TestingParams)
	rres, err := testReasoningSupport(p, model)
	sp.stopSpinner()
	if err != nil {
		fail("%v", err)
		return exitAPI
	}
	printReasoningResult(rres)
	reasonOK := rres.EffortOn != "" || rres.EffortOff != "" || rres.UsesBudget || rres.HasOutput

	sp = startSpinner(M.WaitingAnswer)
	visOK, visDetail, err := testVisionSupport(p, model)
	sp.stopSpinner()
	if err != nil {
		fail("%v", err)
		return exitAPI
	}
	if visOK {
		fmt.Println(M.VisionOK, sanitizeForDisplay(truncateRunes(visDetail, 300)))
		fmt.Println(M.VisionJudge)
	} else {
		fmt.Println(M.VisionFail)
	}

	if len(p.Models) > 1 {
		info(M.ChainFirstOnly)
	}

	if confirmYN(fmt.Sprintf(M.AskSaveModel, name), true) {
		p.UseTools = toolsOK
		p.Vision = &visOK
		if reasonOK {
			v := true
			p.Reasoning = &v
			if rres.EffortOn != "" {
				p.ReasoningEffort = rres.EffortOn
			}
			if rres.UsesBudget {
				p.ReasoningBudget = rres.BudgetOn
			}
		}
		cf.Profiles[name] = p
		if err := saveConfigFile(*cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.SavedOK)
	} else {
		fmt.Println(M.SavedNot)
	}
	return exitOK
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, M.ConfigUsage)
}

func cmdYolo(args []string) int {
	cf, err := loadConfigFile()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}

	if len(args) == 0 {
		if cf.Yolo {
			fmt.Println(M.YoloIsOn)
		} else {
			fmt.Println(M.YoloIsOff)
		}
		return exitOK
	}

	switch strings.ToLower(args[0]) {
	case "on", "1", "true", "enable":
		cf.Yolo = true
		if err := saveConfigFile(cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.YoloGlobalOn)
		return exitOK
	case "off", "0", "false", "disable":
		cf.Yolo = false
		if err := saveConfigFile(cf); err != nil {
			fail(M.SaveFail, err)
			return exitConfig
		}
		fmt.Println(M.YoloGlobalOff)
		return exitOK
	default:
		fail(M.YoloBadArg, args[0])
		return exitConfig
	}
}

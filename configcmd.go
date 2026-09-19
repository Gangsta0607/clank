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
			fail("нужно имя профиля: clank config add <имя>")
			return exitConfig
		}
		return cmdConfigAdd(&cf, args[1])

	case "use":
		if len(args) < 2 {
			fail("нужно имя профиля: clank config use <имя>")
			return exitConfig
		}
		if _, ok := cf.Profiles[args[1]]; !ok {
			fail("нет такого профиля: %s (есть: %s)", args[1], strings.Join(profileNames(cf), ", "))
			return exitConfig
		}
		cf.Active = args[1]
		if err := saveConfigFile(cf); err != nil {
			fail("не смог сохранить: %v", err)
			return exitConfig
		}
		fmt.Println("активный профиль:", args[1])
		return exitOK

	case "list":
		if len(cf.Profiles) == 0 {
			fmt.Println("профилей нет: clank config init")
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
			fail("нужно имя профиля: clank config rm <имя>")
			return exitConfig
		}
		if _, ok := cf.Profiles[args[1]]; !ok {
			fail("нет такого профиля: %s", args[1])
			return exitConfig
		}
		delete(cf.Profiles, args[1])
		if cf.Active == args[1] {
			cf.Active = ""
			if len(cf.Profiles) > 0 {
				cf.Active = newestProfileName(cf)
				info("удалён активный профиль, переключаюсь на %s", cf.Active)
			}
		}
		if err := saveConfigFile(cf); err != nil {
			fail("не смог сохранить: %v", err)
			return exitConfig
		}
		fmt.Println("удалён:", args[1])
		return exitOK

	case "show":
		name, p, err := activeProfile(&cf)
		if err != nil {
			fail("%v", err)
			return exitConfig
		}
		path, _ := configPath()
		fmt.Println("файл:      ", path)
		fmt.Println("профиль:   ", name)
		fmt.Println("base_url:  ", valueOr(p.BaseURL, "<не задан>"))
		fmt.Println("api_key:   ", maskKey(p.APIKey))
		if len(p.Models) == 0 {
			fmt.Println("модели:     <не заданы>")
		} else {
			fmt.Println("модели:    ", "(по порядку фоллбэка)")
			for i, m := range p.Models {
				fmt.Printf("             %d) %s\n", i+1, m)
			}
		}
		fmt.Println("proxy:     ", valueOr(p.Proxy, "<не задан, берётся из env>"))
		fmt.Println("use_tools: ", p.UseTools)
		fmt.Println("таймаут:   ", p.execTimeout())
		if len(cf.Allowed) > 0 {
			fmt.Println("разрешены без подтверждения:", strings.Join(cf.Allowed, ", "))
		}
		return exitOK

	case "set-url", "set-key", "set-model", "set-models", "set-proxy", "set-tools", "set-exec-timeout":
		if len(args) < 2 {
			fail("нужно значение: clank config %s <значение>", args[0])
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
				fail("список моделей пуст")
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
		case "set-exec-timeout":
			secs, err := strconv.Atoi(val)
			if err != nil || secs < 0 {
				fail("таймаут задаётся целым числом секунд (0 — вернуть дефолт)")
				return exitConfig
			}
			p.ExecTimeoutSec = secs
		}
		cf.Profiles[name] = p
		if err := saveConfigFile(cf); err != nil {
			fail("не смог сохранить: %v", err)
			return exitConfig
		}
		fmt.Println("ок")
		return exitOK

	case "test-tools":
		name, _, err := activeProfile(&cf)
		if err != nil {
			fail("%v", err)
			return exitConfig
		}
		return cmdTestTools(&cf, name)

	case "allow-rm":
		if len(args) < 2 {
			fail("нужно имя команды: clank config allow-rm <имя>")
			return exitConfig
		}
		if !cf.disallow(args[1]) {
			fail("%s нет в списке разрешённых", args[1])
			return exitConfig
		}
		if err := saveConfigFile(cf); err != nil {
			fail("не смог сохранить: %v", err)
			return exitConfig
		}
		fmt.Println("убрано из разрешённых:", args[1])
		return exitOK

	case "allow-clear":
		cf.Allowed = nil
		if err := saveConfigFile(cf); err != nil {
			fail("не смог сохранить: %v", err)
			return exitConfig
		}
		fmt.Println("список разрешённых команд очищен")
		return exitOK

	default:
		fail("неизвестная подкоманда: %s", args[0])
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
		fail("мастеру настройки нужен терминал; в скрипте используй clank config set-url/set-key/set-model")
		return exitConfig
	}

	existing, exists := cf.Profiles[name]
	if !exists {
		existing.Created = time.Now().Format(time.RFC3339)
	}

	fmt.Printf("настройка профиля %q (Enter — оставить как есть)\n", name)
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
		fail("не смог сохранить конфиг: %v", err)
		return exitConfig
	}
	fmt.Println("профиль сохранён:", name)

	if confirmYN("подтянуть список моделей и выбрать сейчас? [Y/n]: ", true) {
		if code := selectModels(cf, name, ""); code != exitOK {
			warn("список моделей получить не удалось")
		}
	}

	// Раньше при отказе или сетевой ошибке профиль оставался без модели,
	// и первый же вопрос падал на проверке конфига.
	if len(cf.Profiles[name].Models) == 0 {
		line, ok := askLine("модель (через запятую — цепочка фоллбэка): ")
		if ok && strings.TrimSpace(line) != "" {
			p := cf.Profiles[name]
			for _, m := range strings.Split(line, ",") {
				if m = strings.TrimSpace(m); m != "" {
					p.Models = append(p.Models, m)
				}
			}
			cf.Profiles[name] = p
			if err := saveConfigFile(*cf); err != nil {
				fail("не смог сохранить конфиг: %v", err)
				return exitConfig
			}
		}
	}

	if len(cf.Profiles[name].Models) == 0 {
		warn("профиль без модели — задай позже: clank models  или  clank config set-model <имя>")
		return exitOK
	}

	if confirmYN("проверить поддержку tool calls для этого профиля? [Y/n]: ", true) {
		return cmdTestTools(cf, name)
	}

	fmt.Println("готово. use_tools можно проверить позже: clank config test-tools")
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
			fmt.Fprintln(os.Stderr, `использование: clank models [фильтр]
  показывает модели активного профиля и позволяет выбрать цепочку
  номерами через запятую: 1,4,7 — первая основная, остальные запасные`)
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
		fail("сначала задай base_url и api_key: clank config add %s", name)
		return exitConfig
	}

	sp := startSpinner("получаю список моделей")
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
			fail("под фильтр %q ничего не подошло", filter)
		} else {
			fail("API вернул пустой список моделей")
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
		fmt.Println("\nслева от номера — текущая позиция модели в цепочке фоллбэка")
	}

	line, ok := askLine("\nномера через запятую, первый — основной (Enter — отмена): ")
	if !ok || line == "" {
		fmt.Println("отменено")
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
			fail("некорректный номер: %s (нужно от 1 до %d)", part, len(ids))
			return exitConfig
		}
		if !seen[ids[n-1]] {
			seen[ids[n-1]] = true
			chosen = append(chosen, ids[n-1])
		}
	}
	if len(chosen) == 0 {
		fmt.Println("отменено")
		return exitOK
	}

	p.Models = chosen
	cf.Profiles[name] = p
	if err := saveConfigFile(*cf); err != nil {
		fail("не смог сохранить конфиг: %v", err)
		return exitConfig
	}
	if len(chosen) == 1 {
		fmt.Println("модель установлена:", chosen[0])
	} else {
		fmt.Println("цепочка установлена:", strings.Join(chosen, " → "))
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
	fmt.Println("проверяю tool calls на модели", model, "...")
	sp := startSpinner("жду ответ")
	ok, detailText, err := testToolSupport(p, model)
	sp.stopSpinner()
	if err != nil {
		fail("%v", err)
		return exitAPI
	}
	if ok {
		fmt.Println("модель вызвала tool_call:", sanitizeForDisplay(truncateRunes(detailText, 300)))
	} else {
		fmt.Println("модель НЕ вызвала tool_call, ответила текстом:", truncateRunes(detailText, 300))
	}
	if len(p.Models) > 1 {
		info("проверена только первая модель цепочки; use_tools общий на профиль")
	}

	if confirmYN(fmt.Sprintf("сохранить use_tools=%v для профиля %s? [Y/n]: ", ok, name), true) {
		p.UseTools = ok
		cf.Profiles[name] = p
		if err := saveConfigFile(*cf); err != nil {
			fail("не смог сохранить: %v", err)
			return exitConfig
		}
		fmt.Println("сохранено")
	} else {
		fmt.Println("не сохранено")
	}
	return exitOK
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, `использование: clank config <команда>
  init                       быстрая настройка профиля "default"
  add <имя>                  создать/отредактировать именованный профиль
  use <имя>                  сделать профиль активным
  list                       список профилей
  rm <имя>                   удалить профиль
  show                       показать активный профиль
  set-url <url>              /v1 в конце дописывать не нужно, снимется сам
  set-key <key>
  set-model <a[,b,c]>        цепочка моделей: не ответила первая — идёт вторая
  set-proxy <url|->          '-' — сброс на env-прокси
  set-tools <true|false>     ручной оверрайд use_tools
  set-exec-timeout <сек>     потолок на выполнение команды (0 — дефолт)
  test-tools                 проверить и (с подтверждением) сохранить use_tools
  allow-rm <команда>         убрать команду из разрешённых без подтверждения
  allow-clear                очистить список разрешённых команд`)
}

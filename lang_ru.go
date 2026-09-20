package main

// RU — русские строки интерфейса.
var RU = Msgs{
	// ask.go
	StdinReadFail:  "не смог дочитать stdin: %v — контекст может быть неполным",
	ImageFlagNeeds: "флагу %s нужен путь к файлу изображения",
	BadYoloFlag:    "неверное значение флага --yolo: %s (ожидается on или off)",
	BadReasonFlag:  "неверное значение флага --reasoning: %s (ожидается on или off)",
	UnknownFlag:    "неизвестный флаг: %s (если это часть вопроса — поставь перед ним --)",
	NothingToAsk:   "нечего спрашивать",
	AskUsage:       "использование: clank [-c] [-r] [-v|-q] <вопрос>",
	ProfileDetail:  "профиль %s, модели: %s",
	ReadingStdin:   "читаю stdin, Ctrl-D когда закончишь",
	StdinBytes:     "контекст из stdin: %d байт",
	ContextWrap:    "Контекст:\n%s\n\nВопрос: %s",
	ImageAttached:  "прикреплено изображение %s (%d байт data-uri)",
	SessionSaveErr: "не смог сохранить сессию: %v",
	Thinking:       "думаю",
	StepDetail:     "шаг %d: ответила %s, finish_reason=%s",
	AnswerCut:      "ответ обрезан сервером по лимиту токенов — переспроси короче или задай вопрос по частям",
	Interrupted:    "прервано (Ctrl-C)",
	OutcomeIntr:    "(выполнение прервано)",
	OutcomeEmpty:   "(модель вернула пустой ответ)",
	UnknownTool:    "модель просит неизвестный инструмент %q — не выполняю",
	NoSuchTool:     "ошибка: инструмента %q не существует, доступны: %s, %s, %s",
	ViewArgsBad:    "аргументы просмотра изображения не разобрать: %v",
	ViewArgsNeed:   "ошибка: аргументы должны быть JSON вида {\"path\": \"...\"}, разбор не удался: %v",
	ViewEmptyPath:  "модель прислала пустой путь к изображению",
	ViewPathEmpty:  "ошибка: поле path пустое",
	ViewWatch:      "· смотрю %s",
	ViewOpenFail:   "не удалось открыть изображение %s: %v",
	ViewReadFail:   "ошибка при чтении изображения %s: %v",
	ViewLoaded:     "Изображение %s успешно загружено.",
	ViewContent:    "Содержимое изображения %s:",
	AskArgsBad:     "аргументы вопроса не разобрать: %v",
	AskArgsNeed:    "ошибка: аргументы вопроса должны быть JSON вида {\"question\": \"...\"}, разбор не удался: %v",
	AskEmptyQ:      "модель прислала пустой вопрос",
	AskQEmpty:      "ошибка: поле question пустое",
	AskNoTTY:       "ошибка: нет управляющего терминала для ответа на вопрос",
	AskCancelled:   "пользователь отменил ввод (Ctrl-C / EOF)",
	ShellArgsBad:   "аргументы инструмента не разобрать: %v",
	ShellArgsNeed:  "ошибка: аргументы должны быть JSON вида {\"command\": \"...\"}, разбор не удался: %v",
	ShellEmpty:     "модель прислала пустую команду — не выполняю",
	ShellCmdEmpty:  "ошибка: поле command пустое, команда не выполнялась",
	RunYolo:        "выполняю без вопроса (режим YOLO): %s",
	RunAlways:      "выполняю без вопроса (%s разрешена всегда): %s",
	RunSimple:      "выполняю без вопроса (%s разрешена для простых команд): %s",
	UserDeclined:   "пользователь отказался выполнять эту команду",
	AllowSaveFail:  "не смог сохранить список разрешённых: %v",
	AllowAll:       "%s теперь разрешена всегда, включая сложные команды (убрать: clank config allow-rm %s)",
	AllowSimple:    "%s добавлена в разрешённые для простых команд (убрать: clank config allow-rm %s)",
	OutputMark:     "--- вывод ---",
	CmdInterrupted: "команда прервана пользователем (Ctrl-C)\n",
	CmdTimedOut:    "команда снята по таймауту\n",
	OutputCut:      "%s\n...(пропущено примерно %d байт середины вывода)...\n%s",
	RunStartFail:   "не удалось запустить: ",
	ExecTooLong:    "команда идёт дольше %s — снимаю",
	// config.go
	ConfigCorrupt:  "конфиг повреждён (%s): %w",
	ActiveNotFound: "активный профиль %q не найден в конфиге",
	ActiveSaveFail: "не смог сохранить выбор активного профиля: %v",
	ActiveSwitch:   "активный профиль не задан, переключаюсь на %s (сменить: clank config use <имя>)",
	ProfileInvalid: "профиль не настроен, не хватает: %s — заполни: clank config add <имя>",
	KeyMissing:     "<не задан>",

	// configcmd.go
	NeedProfileName:   "нужно имя профиля: clank config %s <имя>",
	NoSuchProfile:     "нет такого профиля: %s",
	NoSuchProfileList: "нет такого профиля: %s (есть: %s)",
	SaveFail:          "не смог сохранить: %v",
	ActiveProfileIs:   "активный профиль:",
	NoProfiles:        "профилей нет: clank config init",
	ActiveRemoved:     "удалён активный профиль, переключаюсь на %s",
	RemovedIs:         "удалён:",
	ShowFile:          "файл:       ",
	ShowProfile:       "профиль:    ",
	ShowNotSet:        "<не задан>",
	ShowModelsNone:    "модели:     <не заданы>",
	ShowModelsHead:    "модели:     (по порядку фоллбэка)",
	ShowNoEnvProxy:    "<не задан, берётся из env>",
	ShowThinkOn:       "включено",
	ShowThinkOff:      "выключено",
	ShowThinkCal:      "мышление:   %s (%s)\n",
	ShowThinkPlain:    "мышление:   %s",
	ShowThinkAuto:     "мышление:   <авто/по умолчанию провайдера>",
	ShowVisionOn:      "vision:      поддерживается (проверено)",
	ShowVisionOff:     "vision:      не поддерживается",
	ShowVisionAuto:    "vision:      <не проверяли>",
	ShowTimeout:       "таймаут:    ",
	ShowYoloOn:        "yolo:       включён",
	ShowAllowed:       "разрешены:  ",
	ShowLanguage:      "язык:       ",
	ShowLangAuto:      "авто",
	NeedValue:         "нужно значение: clank config %s <значение>",
	ModelsEmpty:       "список моделей пуст",
	BadSetReason:      "неверное значение: %s (ожидается on, off или -)",
	BadTimeout:        "таймаут задаётся целым числом секунд (0 — вернуть дефолт)",
	SavedOK:           "ок",
	AllowNeedName:     "нужно имя команды: clank config allow-rm <имя>",
	AllowNotListed:    "%s нет в списке разрешённых",
	AllowRemoved:      "убрано из разрешённых:",
	AllowCleared:      "список разрешённых команд очищен",
	UnknownSubcommand: "неизвестная подкоманда: %s",
	WizardNeedsTTY:    "мастеру настройки нужен терминал; в скрипте используй clank config set-url/set-key/set-model",
	WizardTitle:       "настройка профиля %q (Enter — оставить как есть)",
	ConfigSaveFail:    "не смог сохранить конфиг: %v",
	ProfileSaved:      "профиль сохранён:",
	AskFetchModels:    "подтянуть список моделей и выбрать сейчас? [Y/n]: ",
	ModelsFetchFail:   "список моделей получить не удалось",
	AskModelLine:      "модель (через запятую — цепочка фоллбэка): ",
	ProfileNoModel:    "профиль без модели — задай позже: clank models  или  clank config set-model <имя>",
	AskTestTools:      "проверить поддержку tool calls для этого профиля? [Y/n]: ",
	ToolsSkipLater:    "готово. use_tools можно проверить позже: clank config test-tools",
	ModelsNeedCreds:   "сначала задай base_url и api_key: clank config add %s",
	FetchingModels:    "получаю список моделей",
	FilterNoMatch:     "под фильтр %q ничего не подошло",
	ModelsEmptyAPI:    "API вернул пустой список моделей",
	ChainHint:         "\nслева от номера — текущая позиция модели в цепочке фоллбэка",
	AskChainNumbers:   "\nномера через запятую, первый — основной (Enter — отмена): ",
	Cancelled:         "отменено",
	BadNumber:         "некорректный номер: %s (нужно от 1 до %d)",
	ModelSet:          "модель установлена:",
	ChainSet:          "цепочка установлена:",
	TestToolsCheck:    "проверяю tool calls на модели",
	WaitingAnswer:     "жду ответ",
	ToolCalled:        "модель вызвала tool_call:",
	ToolNotCalled:     "модель НЕ вызвала tool_call, ответила текстом:",
	ChainFirstOnly:    "проверена только первая модель цепочки; use_tools общий на профиль",
	AskSaveTools:      "сохранить use_tools=%v для профиля %s? [Y/n]: ",
	SavedNot:          "не сохранено",
	TestReasonCheck:   "проверяю поддержку reasoning на модели",
	TestingParams:     "тестирую параметры",
	ReasonResults:     "\nрезультаты проверки:",
	ReasonNone:        "нет",
	ReasonEffortOK:    "  управление reasoning_effort: поддерживается (включение: %s, выключение: %s)\n",
	ReasonBudgetOK:    "  управление бюджетом токенов: поддерживается (reasoning_budget)",
	ReasonNoParams:    "  параметры управления reasoning: не поддерживаются сервером",
	ReasonHasOutput:   "  блок reasoning: найден",
	ReasonNoOutput:    "  блок reasoning: не найден",
	ReasonUnsupported: "эта модель не использует и не настраивает reasoning",
	AskReasonOn:       "\nвключить reasoning для профиля %s? [Y/n]: ",
	SavedReasonOn:     "сохранено (reasoning: включён)",
	SettingsKept:      "настройки не изменены",
	TestVisionCheck:   "проверяю поддержку vision (изображений) на модели",
	VisionOK:          "модель ответила на запрос с изображением:",
	VisionJudge:       "реши сам, увидела ли она картинку, прежде чем сохранять",
	VisionFail:        "модель НЕ смогла обработать изображение",
	AskSaveVision:     "сохранить vision=%v для профиля %s? [Y/n]: ",
	TestModelCheck:    "проверяю модель %s: tools, reasoning, vision...",
	AskSaveModel:      "сохранить use_tools, reasoning и vision в профиль %s? [Y/n]: ",
	YoloIsOn:          "режим YOLO: включён (команды выполняются без подтверждения)",
	YoloIsOff:         "режим YOLO: выключен (требуется подтверждение команд)",
	YoloGlobalOn:      "режим YOLO включён глобально",
	YoloGlobalOff:     "режим YOLO выключен глобально",
	YoloBadArg:        "неизвестный параметр: %s (используй clank yolo on|off)",
	// client.go
	ErrSetup:   "настройка профиля",
	ErrNoConn:  "не удалось соединиться",
	ModelParen: " (модель %s)",
	ErrShort:   "ошибка настройки",
	NoConn:     "нет связи",
	HintAuth:   "ключ не принят — проверь: clank config show",
	HintURL:    "проверь base_url (clank ask сам дописывает /v1/chat/completions): clank config show",
	HintModel:  "такой модели у провайдера нет — выбери заново: clank models",
	HintCtx:    "контекст переполнен — очисти историю (clank session clear) или подавай меньше данных в пайп",
	HintRate:   "лимит запросов у провайдера",
	HintNet:    "сеть, прокси или адрес — проверь: clank config show",
	ProxyParse: "не разобрать proxy %q: %v — поправь: clank config set-proxy <url|->",
	ProxyBlind: "proxy %q без схемы или хоста (нужно вида http://host:port) — поправь: clank config set-proxy <url|->",
	RespUnpars: "ответ не разобрать: %v (первые байты: %s)",
	EmptyChoi:  "пустой список choices в ответе",
	ReqDetail:  "запрос: модель %s, сообщений %d, %d байт",
	RespDetail: "ответ %d за %.1fс, %d байт",
	NoModels:   "в профиле нет ни одной модели — выбери: clank models",
	Answering:  "отвечает %s",
	ModelFail:  "%s не ответил (%s), пробую %s",
	RetryIn:    "%s: %s, повтор через %s (%d/%d)",
	NoAnswer:   "ни одна модель не ответила:\n  %s\nпоследняя ошибка: %v",
	ModelsUnp:  "список моделей не разобрать: %v",
	APIError:   "API вернул ошибку: %s",
	ToolProbeS: "У тебя есть инструмент run_shell_command. Если для точного ответа нужно посмотреть реальные данные окружения — используй его, а не гадай.",
	ToolProbeU: "Сколько файлов и папок лежит прямо в текущей директории? Если можешь узнать точно через доступный инструмент — узнай.",
	VisionProb: "Какого цвета это изображение? Ответь кратко.",
	ImageBig:   "файл слишком большой (>20MB)",
	ImageType:  "файл %s не является поддерживаемым изображением (тип %s)",

	// lifecycle.go
	PurgeEmpty:   "конфигурационный каталог и данные уже пусты",
	PurgeAsk:     "очистить все настройки, сессии и данные (%s)? [y/N]: ",
	Aborted:      "отменено",
	PurgeFail:    "ошибка при очистке данных (%s): %v",
	PurgeDone:    "все пользовательские данные и настройки clank очищены",
	BinPathFail:  "не удалось определить путь к бинарнику: %v",
	UninstAsk:    "удалить исполняемый файл clank (%s)? [y/N]: ",
	UninstFail:   "ошибка при удалении %s: %v\n  (если нет прав, выполните: sudo rm %s)",
	UninstDone:   "исполняемый файл clank (%s) удалён с машины\n",
	NukeAsk:      "ВНИМАНИЕ: это полностью удалит бинарник (%s) и ВСЕ данные (%s). Продолжить? [y/N]: ",
	NukePartFail: "при полном удалении возникли ошибки: %s",
	NukeDataErr:  "данные (%v)",
	NukeBinErr:   "бинарник (%v)",
	NukeDone:     "clank полностью удалён с машины (бинарник и все данные)",

	// session.go
	SessPathFail:  "не смог определить путь сессии: %v",
	SessFresh:     "сессии в этом терминале ещё нет — начинаю с чистого листа",
	SessReadFail:  "не смог прочитать сессию (%s): %v — начинаю с чистого листа",
	SessCorrupt:   "файл сессии повреждён (%s): %v — начинаю с чистого листа",
	SessSwitch:    "сессия в этом терминале от профиля %s, сейчас активен %s — начинаю заново",
	SessDetail:    "сессия: %d сообщений, обновлена %s",
	SessEmpty:     "сессия этого терминала пуста",
	SessReadErr:   "не смог прочитать сессию: %v",
	SessFileBad:   "файл сессии повреждён (%s): %v",
	SessHeader:    "профиль: %s, сообщений: %d, обновлена: %s%s\n\n",
	MarkDone:      "[выполнено]",
	MarkAnswer:    "[ответ пользователя]",
	MarkImage:     "[изображение загружено]",
	MarkAsks:      "[assistant спрашивает] %s\n",
	MarkViews:     "[assistant смотрит изображение] %s\n\n",
	MarkRuns:      "[assistant просит выполнить] %s(%s)\n\n",
	MarkAsst:      "[assistant] %s\n\n",
	SessRemoved:   "удалено сессий: %d\n",
	NoSessions:    "сессий нет",
	SessClearDone: "сессия этого терминала очищена",
	SessClearFail: "не смог удалить сессию: %v",
	SessClearOne:  "сессия этого терминала и так пуста",
	StClosed:      "терминал закрыт",
	StShellAlive:  "шелл %d жив",
	StHeadless:    "без терминала",
	SessRow:       "%s%-24s %s  %6d байт  %s\n",

	// tty.go
	ConfirmHint:   "не понял: y — да, n — нет",
	NoTTYDecline:  "нет управляющего терминала — подтвердить выполнение невозможно, отказываю",
	ModelWantsRun: "\nмодель хочет выполнить:\n%s\n",
	ComplexWarn:   "  («%s» разрешена только для простых команд, но эта команда содержит спецсимволы шелла — подтверди вручную)\n",
	RunPromptAll:  "выполнить? [y/N/a] (a — разрешать «%s» всегда, включая сложные): ",
	RunPrompt:     "выполнить? [y/N/a] (a — разрешать «%s» всегда): ",
	ConfirmCMD:    "не понял: y — выполнить, n — отказать, a — разрешать эту команду всегда",
	NoTTYDefault:  "нет управляющего терминала — выбираю значение по умолчанию: %s",
	NoTTYNoAnswer: "нет управляющего терминала — ответить на вопрос невозможно",
	ChooseOrType:  "выбери 1-%d или напиши свой ответ: ",
	ChooseOrDef:   "выбери 1-%d или напиши свой ответ [%s]: ",
	AnswerOrDef:   "ответ [%s]: ",
	AnswerIs:      "ответ: ",
	EnterNum:      "введи номер варианта или свой ответ",

	// update.go
	UpdBinFail:  "не удалось определить путь к текущему бинарнику: %v",
	UpdLinkFail: "не удалось разрешить симлинк бинарника: %v",
	UpdChecking: "проверяю обновления на GitHub",
	UpdCheckErr: "не удалось проверить обновления: %v",
	UpdLatest:   "clank %s уже последней версии\n",
	UpdAvail:    "Доступна новая версия: %s (текущая: %s)\n",
	UpdNotes:    "\nИзменения в релизе:\n%s\n\n",
	UpdCached:   "найден ранее скачанный бинарник в кэше: %s",
	UpdAsk:      "обновить %s до %s? [Y/n]: ",
	UpdAbort:    "обновление отменено",
	UpdDown:     "скачиваю %s",
	UpdDownErr:  "не удалось скачать релиз: %v",
	UpdUnpErr:   "не удалось распаковать бинарник: %v",
	UpdKept:     "сохранён в кэш: %s",
	UpdCacheErr: "не удалось прочитать кэш: %v, скачиваю заново",
	UpdApplyErr: "ошибка при установке обновления в %s: %v",
	UpdSudoHint: "\nБинарник сохранён в кэше. Если проблема в правах доступа, запустите:\n  sudo clank update\n",
	UpdDone:     "clank успешно обновлён до %s!\n",
	UpdNoAsset:  "не найден подходящий архив для платформы %s/%s в релизе %s",
	UpdNoExeZip: "исполняемый файл clank.exe не найден внутри zip архива",
	UpdNoExeTar: "исполняемый файл clank не найден внутри tar.gz архива",

	// systemprompt_unix.go
	ShellApprox:  " (по $SHELL, не точно)",
	ShellUnknown: "неизвестно",

	LangNameRU: "русский",
	LangNameEN: "английский",
	FailPrefix: "ошибка: ",
	LangShow:   "язык интерфейса:",
	LangSaved:  "язык сохранён:",
	LangBadArg: "неизвестный язык: %s (используй clank language en|ru|auto)",

	HelpAsk: `clank <вопрос> — спросить модель. stdin из пайпа подхватывается сам.
Флаги: -c (форс чтения stdin), -r (продолжить сессию своей вкладки),
-i <файл> (прикрепить картинку), --yolo=on|off, --reasoning=on|off,
-v (подробно) / -q (молча), -- (дальше — текст вопроса).
Вопрос, начинающийся со слова-команды (models, config…), задавай явно:
clank ask models …`,
	HelpConfigInit: `clank config init — мастер настройки профиля "default".
По шагам спросит base_url, api_key, предложит выбрать модель и проверить
tool calls. В скриптах вместо мастера — set-url/set-key/set-model.`,
	HelpConfigAdd: `clank config add <имя> — создать или перенастроить именованный профиль.
Удобно держать по профилю на провайдера и переключаться через config use.`,
	HelpConfigUse: `clank config use <имя> — сделать профиль активным.
Все вопросы и тесты идут через активный профиль.`,
	HelpConfigList: `clank config list — профили, активный помечен *. Показывает
url, модели и включённые tool calls.`,
	HelpConfigRm: `clank config rm <имя> — удалить профиль. Если удалён активный,
clank сам переключится на самый свежий из оставшихся.`,
	HelpConfigShow: `clank config show — активный профиль целиком: url, ключ
(замаскирован), модели, reasoning, vision, yolo, язык, разрешённые команды.`,
	HelpConfigSet: `clank config set-* — точечная правка активного профиля:
set-url, set-key, set-model <a[,b,c]> (цепочка фоллбэка), set-proxy <url|->,
set-tools <true|false>, set-reasoning <on|off|->, set-exec-timeout <сек>.`,
	HelpTestTools: `clank config test-tools — проверяет, вызывает ли модель tool_call
на задаче, где без инструмента не ответить точно. Результат показывают,
с твоего подтверждения пишут use_tools в профиль.`,
	HelpTestReasoning: `clank config test-reasoning — подбирает параметры управления
мышлением (effort или бюджет токенов) и смотрит, thinking-блок в ответах.
Печатает что нашлось; про блок reasoning пишет "найден / не найден".
С твоего подтверждения включает reasoning в профиле.`,
	HelpTestVision: `clank config test-vision — шлёт модели зелёный квадрат с вопросом
о цвете и показывает её ответ. Видит ли она картинку — решаешь сам по ответу;
с твоего подтверждения результат пишут в профиль (нужен для промпта).`,
	HelpTestModel: `clank config test-model — все три проверки разом (tools,
reasoning, vision) с общим итогом. Один вопрос — сохранить всё в профиль.`,
	HelpConfigAllow: `clank config allow-rm <команда> — убрать команду из списка
выполняемых без подтверждения. allow-clear — очистить список целиком.
Команды попадают в список, когда на вопрос "выполнить?" отвечаешь "a".`,
	HelpSessionShow: `clank session show — транскрипт сессии текущей вкладки:
вопросы, ответы, вызовы инструментов.`,
	HelpSessionClear: `clank session clear — стереть сессию текущей вкладки.
С --all — стереть сессии всех терминалов.`,
	HelpSessionList: `clank session list — все сессии на машине. * — текущая
вкладка; видно, чей шелл ещё жив, а чей терминал уже закрыт.`,
	HelpYolo: `clank yolo [on|off] — глобальный режим без подтверждений команд.
Без аргумента показывает состояние. На одну реплику — флаг --yolo=on|off.`,
	HelpLanguage: `clank language [en|ru|auto] — язык интерфейса.
Без аргумента показывает текущий. auto — определять по LANG в терминале.`,
	HelpUpdate: `clank update [-y] — свериться с GitHub-релизами и поставить новую
версию поверх текущей. Скачанное кешируется: при ошибке прав повтор делают
через sudo clank update без перекачки.`,
	HelpPurge: `clank purge [-y] — стереть ~/.config/clank/ (профили, сессии, кэш).
Бинарник остаётся. Это заводской сброс настроек.`,
	HelpUninstall: `clank uninstall [-y] — удалить исполняемый файл clank.
Настройки и сессии остаются.`,
	HelpNuke: `clank nuke [-y] — удалить и бинарник, и все данные.
Точка невозврата; спросит подтверждение.`,
	HelpUnknown: "нет такой справки: %s",
	HelpTopics: `разделы: ask models config session yolo language completion update purge uninstall nuke
у config и session есть вложенные: например clank help config test-model`,

	CompletionUsage:     "использование: clank completion [zsh|bash|fish|install [шелл]]",
	CompletionBadShell:  "неизвестный шелл: %s (умею: zsh, bash, fish)",
	CompletionDone:      "автодополнение установлено: %s",
	CompletionWriteFail: "не смог записать автодополнение: %v",
	CompletionZshHint:   "добавь в .zshrc, если ещё нет: fpath=(~/.zfunc $fpath)",
	CompletionRemoved:   "убрано автодополнение: %s",
	AskCompletion:       "поставить автодополнение команд для %s (%s)? [Y/n]: ",
	HelpCompletion: `clank completion [zsh|bash|fish] — напечатать скрипт автодополнения.
clank completion install — поставить его для текущего шелла.
Предложить установку могут и сами при первом config init.`,
	Usage: `clank — минималистичный CLI к OpenAI-совместимому API

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
  clank config test-vision              проверить и сохранить поддержку изображений (vision) моделью
  clank config test-model               все три проверки разом с одним сохранением
  clank config allow-rm/allow-clear     список команд, выполняемых без подтверждения

  clank session show                    транскрипт сессии этого терминала
  clank session clear [--all]           стереть сессию (--all — во всех терминалах)
  clank session list                    все сессии на машине

  clank language [en|ru|auto]           язык интерфейса (без аргумента — показать)
  clank completion [шелл|install]        скрипт автодополнения или его установка
  clank yolo [on|off]                   включить/выключить глобальный режим YOLO (без подтверждения команд)
  clank update [-y]                     проверить и установить новую версию с GitHub
  clank purge [-y]                      стереть все пользовательские данные (~/.config/clank/)
  clank uninstall [-y]                  удалить исполняемый файл clank
  clank nuke [-y]                       полностью удалить clank и все его данные

  clank help <команда>                справка по разделу, вложенные тоже: help config test-model
  clank version

примеры:
  clank "почему grub не видит второй диск"
  dmesg | tail -50 | clank "разбери ошибку"
  clank -r "а если так: ..."

вопрос лучше брать в кавычки: без них ?, *, >, ! и обратные кавычки перехватит шелл.`,
	SessionUsage: `использование: clank session <команда>
  show              транскрипт сессии текущего терминала
  clear             стереть сессию текущего терминала
  clear --all       стереть сессии всех терминалов
  list              все сессии на машине

у каждой вкладки терминала своя история; -r продолжает историю своей вкладки`,
	ConfigUsage: `использование: clank config <команда>
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
  set-reasoning <on|off|->   включить/выключить/сбросить мышление модели
  set-exec-timeout <сек>     потолок на выполнение команды (0 — дефолт)
  test-tools                 проверить и (с подтверждением) сохранить use_tools
  test-reasoning             проверить и откалибровать поддержку reasoning
  test-vision                проверить и (с подтверждением) сохранить vision
  test-model                 все три проверки разом (tools, reasoning, vision)
  allow-rm <команда>         убрать команду из разрешённых без подтверждения
  allow-clear                очистить список разрешённых команд`,
	ModelsHelp: `использование: clank models [фильтр]
  показывает модели активного профиля и позволяет выбрать цепочку
  номерами через запятую: 1,4,7 — первая основная, остальные запасные`,
}

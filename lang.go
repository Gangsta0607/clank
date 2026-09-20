package main

import (
	"fmt"
	"os"
	"strings"
)

// Lang — код языка интерфейса. Сейчас два: ru и en, всё остальное
// откатывается на en. Новый язык добавляется так: таблица MsgsXX в
// lang_xx.go и одна ветка в applyLang — компилятор сам подсветит
// незаполненные поля (структура одна на всех).
type Lang string

const (
	LangEN Lang = "en"
	LangRU Lang = "ru"
)

// Msgs — все строки интерфейса на одном языке. Форматные глаголы (%s, %d)
// обязаны совпадать в обеих таблицах.
type Msgs struct {
	Usage        string
	SessionUsage string
	ConfigUsage  string
	ModelsHelp   string

	LangNameRU string
	LangNameEN string
	FailPrefix string
	LangShow   string
	LangSaved  string
	LangBadArg string

	// ask.go
	StdinReadFail  string
	ImageFlagNeeds string
	BadYoloFlag    string
	BadReasonFlag  string
	UnknownFlag    string
	NothingToAsk   string
	AskUsage       string
	ProfileDetail  string
	ReadingStdin   string
	StdinBytes     string
	ContextWrap    string
	ImageAttached  string
	SessionSaveErr string
	Thinking       string
	StepDetail     string
	AnswerCut      string
	Interrupted    string
	OutcomeIntr    string
	OutcomeEmpty   string
	UnknownTool    string
	NoSuchTool     string
	ViewArgsBad    string
	ViewArgsNeed   string
	ViewEmptyPath  string
	ViewPathEmpty  string
	ViewWatch      string
	ViewOpenFail   string
	ViewReadFail   string
	ViewLoaded     string
	ViewContent    string
	AskArgsBad     string
	AskArgsNeed    string
	AskEmptyQ      string
	AskQEmpty      string
	AskNoTTY       string
	AskCancelled   string
	ShellArgsBad   string
	ShellArgsNeed  string
	ShellEmpty     string
	ShellCmdEmpty  string
	RunYolo        string
	RunAlways      string
	RunSimple      string
	UserDeclined   string
	AllowSaveFail  string
	AllowAll       string
	AllowSimple    string
	OutputMark     string
	CmdInterrupted string
	CmdTimedOut    string
	OutputCut      string
	RunStartFail   string
	ExecTooLong    string

	// config.go
	ConfigCorrupt  string
	ActiveNotFound string
	ActiveSaveFail string
	ActiveSwitch   string
	ProfileInvalid string
	KeyMissing     string

	// configcmd.go
	NeedProfileName   string
	NoSuchProfile     string
	NoSuchProfileList string
	SaveFail          string
	ActiveProfileIs   string
	NoProfiles        string
	ActiveRemoved     string
	RemovedIs         string
	ShowFile          string
	ShowProfile       string
	ShowNotSet        string
	ShowModelsNone    string
	ShowModelsHead    string
	ShowNoEnvProxy    string
	ShowThinkOn       string
	ShowThinkOff      string
	ShowThinkCal      string
	ShowThinkPlain    string
	ShowThinkAuto     string
	ShowVisionOn      string
	ShowVisionOff     string
	ShowVisionAuto    string
	ShowTimeout       string
	ShowYoloOn        string
	ShowAllowed       string
	ShowLanguage      string
	ShowLangAuto      string
	NeedValue         string
	ModelsEmpty       string
	BadSetReason      string
	BadTimeout        string
	SavedOK           string
	AllowNeedName     string
	AllowNotListed    string
	AllowRemoved      string
	AllowCleared      string
	UnknownSubcommand string
	WizardNeedsTTY    string
	WizardTitle       string
	ConfigSaveFail    string
	ProfileSaved      string
	AskFetchModels    string
	ModelsFetchFail   string
	AskModelLine      string
	ProfileNoModel    string
	AskTestTools      string
	ToolsSkipLater    string
	ModelsNeedCreds   string
	FetchingModels    string
	FilterNoMatch     string
	ModelsEmptyAPI    string
	ChainHint         string
	AskChainNumbers   string
	Cancelled         string
	BadNumber         string
	ModelSet          string
	ChainSet          string
	TestToolsCheck    string
	WaitingAnswer     string
	ToolCalled        string
	ToolNotCalled     string
	ChainFirstOnly    string
	AskSaveTools      string
	SavedNot          string
	TestReasonCheck   string
	TestingParams     string
	ReasonResults     string
	ReasonNone        string
	ReasonEffortOK    string
	ReasonBudgetOK    string
	ReasonNoParams    string
	ReasonHasOutput   string
	ReasonNoOutput    string
	ReasonUnsupported string
	AskReasonOn       string
	SavedReasonOn     string
	AskReasonOff      string
	SavedReasonOff    string
	SettingsKept      string
	TestVisionCheck   string
	VisionOK          string
	VisionConfirmed   string
	VisionFail        string
	AskSaveVision     string
	YoloIsOn          string
	YoloIsOff         string
	YoloGlobalOn      string
	YoloGlobalOff     string
	YoloBadArg        string

	// client.go
	ErrSetup   string
	ErrNoConn  string
	ModelParen string
	ErrShort   string
	NoConn     string
	HintAuth   string
	HintURL    string
	HintModel  string
	HintCtx    string
	HintRate   string
	HintNet    string
	ProxyParse string
	ProxyBlind string
	RespUnpars string
	EmptyChoi  string
	ReqDetail  string
	RespDetail string
	NoModels   string
	Answering  string
	ModelFail  string
	RetryIn    string
	NoAnswer   string
	ModelsUnp  string
	APIError   string
	ToolProbeS string
	ToolProbeU string
	VisionProb string
	ImageBig   string
	ImageType  string

	// lifecycle.go
	PurgeEmpty   string
	PurgeAsk     string
	Aborted      string
	PurgeFail    string
	PurgeDone    string
	BinPathFail  string
	UninstAsk    string
	UninstFail   string
	UninstDone   string
	NukeAsk      string
	NukePartFail string
	NukeDataErr  string
	NukeBinErr   string
	NukeDone     string

	// session.go
	SessPathFail  string
	SessFresh     string
	SessReadFail  string
	SessCorrupt   string
	SessSwitch    string
	SessDetail    string
	SessEmpty     string
	SessReadErr   string
	SessFileBad   string
	SessHeader    string
	MarkDone      string
	MarkAnswer    string
	MarkImage     string
	MarkAsks      string
	MarkViews     string
	MarkRuns      string
	MarkAsst      string
	SessRemoved   string
	NoSessions    string
	SessClearDone string
	SessClearFail string
	SessClearOne  string
	StClosed      string
	StShellAlive  string
	StHeadless    string
	SessRow       string

	// tty.go
	ConfirmHint   string
	NoTTYDecline  string
	ModelWantsRun string
	ComplexWarn   string
	RunPromptAll  string
	RunPrompt     string
	ConfirmCMD    string
	NoTTYDefault  string
	NoTTYNoAnswer string
	ChooseOrType  string
	ChooseOrDef   string
	AnswerOrDef   string
	AnswerIs      string
	EnterNum      string

	// update.go
	UpdBinFail  string
	UpdLinkFail string
	UpdChecking string
	UpdCheckErr string
	UpdLatest   string
	UpdAvail    string
	UpdNotes    string
	UpdCached   string
	UpdAsk      string
	UpdAbort    string
	UpdDown     string
	UpdDownErr  string
	UpdUnpErr   string
	UpdKept     string
	UpdCacheErr string
	UpdApplyErr string
	UpdSudoHint string
	UpdDone     string
	UpdNoAsset  string
	UpdNoExeZip string
	UpdNoExeTar string

	// systemprompt_unix.go
	ShellApprox  string
	ShellUnknown string
}

// M — активная таблица строк. Переключается один раз на старте.
var M = &EN

// curLang — активный язык (нужен systemprompt.go).
var curLang = LangEN

// detectLang определяет язык по переменным окружения. Порядок — как
// принято: LC_ALL главнее всего, дальше LC_MESSAGES, дальше LANG.
// Первое непустое значение решает: начинается с ru — русский, иначе
// английский (включая C/POSIX и неизвестные локали).
func detectLang() Lang {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			if strings.HasPrefix(strings.ToLower(v), "ru") {
				return LangRU
			}
			return LangEN
		}
	}
	return LangEN
}

// resolveLang: явная настройка из конфига важнее автоопределения.
func resolveLang(setting string) Lang {
	switch strings.ToLower(strings.TrimSpace(setting)) {
	case "ru":
		return LangRU
	case "en":
		return LangEN
	default:
		return detectLang()
	}
}

// applyLang включает таблицу нужного языка.
func applyLang(l Lang) {
	curLang = l
	if l == LangRU {
		M = &RU
	} else {
		M = &EN
	}
}

// showLanguage — значение для строки языка в config show.
func showLanguage(setting string) string {
	if strings.TrimSpace(setting) == "" {
		return M.ShowLangAuto + " (" + string(resolveLang("")) + ")"
	}
	return setting
}

// initLang читает настройку языка из конфига (без побочных эффектов —
// loadConfigFile ничего не пишет) и включает нужную таблицу.
func initLang() {
	cf, err := loadConfigFile()
	if err != nil {
		applyLang(detectLang())
		return
	}
	applyLang(resolveLang(cf.Language))
}

// cmdLanguage — `clank language [en|ru|auto]`: без аргумента показывает
// текущий язык, с аргументом — сохраняет в конфиг.
func cmdLanguage(args []string) int {
	cf, err := loadConfigFile()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}

	if len(args) == 0 {
		l := resolveLang(cf.Language)
		name := M.LangNameEN
		if l == LangRU {
			name = M.LangNameRU
		}
		fmt.Println(M.LangShow, name)
		return exitOK
	}

	switch strings.ToLower(args[0]) {
	case "ru", "русский":
		cf.Language = "ru"
	case "en", "английский", "english":
		cf.Language = "en"
	case "auto", "авто", "-", "default":
		cf.Language = ""
	default:
		fail(M.LangBadArg, args[0])
		return exitConfig
	}
	if err := saveConfigFile(cf); err != nil {
		fail(M.SaveFail, err)
		return exitConfig
	}
	applyLang(resolveLang(cf.Language))
	name := M.LangNameEN
	if curLang == LangRU {
		name = M.LangNameRU
	}
	fmt.Println(M.LangSaved, name)
	return exitOK
}

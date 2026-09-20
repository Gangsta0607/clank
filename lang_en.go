package main

// EN — English UI strings.
var EN = Msgs{
	// ask.go
	StdinReadFail:  "could not read stdin to the end: %v — context may be incomplete",
	ImageFlagNeeds: "flag %s needs an image file path",
	BadYoloFlag:    "bad --yolo value: %s (expected on or off)",
	BadReasonFlag:  "bad --reasoning value: %s (expected on or off)",
	UnknownFlag:    "unknown flag: %s (if it is part of the question — put -- before it)",
	NothingToAsk:   "nothing to ask",
	AskUsage:       "usage: clank [-c] [-r] [-v|-q] <question>",
	ProfileDetail:  "profile %s, models: %s",
	ReadingStdin:   "reading stdin, Ctrl-D when done",
	StdinBytes:     "stdin context: %d bytes",
	ContextWrap:    "Context:\n%s\n\nQuestion: %s",
	ImageAttached:  "attached image %s (%d bytes of data-uri)",
	SessionSaveErr: "could not save session: %v",
	Thinking:       "thinking",
	StepDetail:     "step %d: answered by %s, finish_reason=%s",
	AnswerCut:      "answer was cut by the server token limit — ask shorter or split the question",
	Interrupted:    "interrupted (Ctrl-C)",
	OutcomeIntr:    "(execution interrupted)",
	OutcomeEmpty:   "(model returned an empty answer)",
	UnknownTool:    "model asks for unknown tool %q — not running it",
	NoSuchTool:     "error: no such tool %q, available: %s, %s, %s",
	ViewArgsBad:    "cannot parse image-view arguments: %v",
	ViewArgsNeed:   "error: arguments must be JSON like {\"path\": \"...\"}, parse failed: %v",
	ViewEmptyPath:  "model sent an empty image path",
	ViewPathEmpty:  "error: path field is empty",
	ViewWatch:      "· viewing %s",
	ViewOpenFail:   "could not open image %s: %v",
	ViewReadFail:   "error reading image %s: %v",
	ViewLoaded:     "Image %s loaded successfully.",
	ViewContent:    "Contents of image %s:",
	AskArgsBad:     "cannot parse question arguments: %v",
	AskArgsNeed:    "error: question arguments must be JSON like {\"question\": \"...\"}, parse failed: %v",
	AskEmptyQ:      "model sent an empty question",
	AskQEmpty:      "error: question field is empty",
	AskNoTTY:       "error: no controlling terminal to answer the question",
	AskCancelled:   "user cancelled the input (Ctrl-C / EOF)",
	ShellArgsBad:   "cannot parse tool arguments: %v",
	ShellArgsNeed:  "error: arguments must be JSON like {\"command\": \"...\"}, parse failed: %v",
	ShellEmpty:     "model sent an empty command — not running it",
	ShellCmdEmpty:  "error: command field is empty, nothing was run",
	RunYolo:        "running without asking (YOLO mode): %s",
	RunAlways:      "running without asking (%s is always allowed): %s",
	RunSimple:      "running without asking (%s is allowed for simple commands): %s",
	UserDeclined:   "user declined to run this command",
	AllowSaveFail:  "could not save the allowed list: %v",
	AllowAll:       "%s is now always allowed, including complex commands (remove: clank config allow-rm %s)",
	AllowSimple:    "%s added to allowed simple commands (remove: clank config allow-rm %s)",
	OutputMark:     "--- output ---",
	CmdInterrupted: "command interrupted by user (Ctrl-C)\n",
	CmdTimedOut:    "command killed on timeout\n",
	OutputCut:      "%s\n...(about %d bytes of middle output skipped)...\n%s",
	RunStartFail:   "could not start: ",
	ExecTooLong:    "command runs longer than %s — killing it",
	// config.go
	ConfigCorrupt:  "config is corrupt (%s): %w",
	ActiveNotFound: "active profile %q not found in config",
	ActiveSaveFail: "could not save the active profile choice: %v",
	ActiveSwitch:   "no active profile set, switching to %s (change: clank config use <name>)",
	ProfileInvalid: "profile is not set up, missing: %s — fill in: clank config add <name>",
	KeyMissing:     "<not set>",

	// configcmd.go
	NeedProfileName:   "need a profile name: clank config %s <name>",
	NoSuchProfile:     "no such profile: %s",
	NoSuchProfileList: "no such profile: %s (have: %s)",
	SaveFail:          "could not save: %v",
	ActiveProfileIs:   "active profile:",
	NoProfiles:        "no profiles: clank config init",
	ActiveRemoved:     "removed the active profile, switching to %s",
	RemovedIs:         "removed:",
	ShowFile:          "file:       ",
	ShowProfile:       "profile:    ",
	ShowNotSet:        "<not set>",
	ShowModelsNone:    "models:     <none>",
	ShowModelsHead:    "models:     (in fallback order)",
	ShowNoEnvProxy:    "<not set, taken from env>",
	ShowThinkOn:       "on",
	ShowThinkOff:      "off",
	ShowThinkCal:      "reasoning:  %s (%s)\n",
	ShowThinkPlain:    "reasoning:  %s",
	ShowThinkAuto:     "reasoning:  <auto/provider default>",
	ShowVisionOn:      "vision:      supported (tested)",
	ShowVisionOff:     "vision:      not supported",
	ShowVisionAuto:    "vision:      <not tested>",
	ShowTimeout:       "timeout:    ",
	ShowYoloOn:        "yolo:       on",
	ShowAllowed:       "allowed:    ",
	ShowLanguage:      "language:   ",
	ShowLangAuto:      "auto",
	NeedValue:         "need a value: clank config %s <value>",
	ModelsEmpty:       "model list is empty",
	BadSetReason:      "bad value: %s (expected on, off or -)",
	BadTimeout:        "timeout is whole seconds (0 — back to default)",
	SavedOK:           "ok",
	AllowNeedName:     "need a command name: clank config allow-rm <name>",
	AllowNotListed:    "%s is not in the allowed list",
	AllowRemoved:      "removed from allowed:",
	AllowCleared:      "allowed commands list cleared",
	UnknownSubcommand: "unknown subcommand: %s",
	WizardNeedsTTY:    "the setup wizard needs a terminal; in scripts use clank config set-url/set-key/set-model",
	WizardTitle:       "setting up profile %q (Enter — keep as is)",
	ConfigSaveFail:    "could not save config: %v",
	ProfileSaved:      "profile saved:",
	AskFetchModels:    "fetch the model list and choose now? [Y/n]: ",
	ModelsFetchFail:   "could not get the model list",
	AskModelLine:      "model (comma-separated — fallback chain): ",
	ProfileNoModel:    "profile has no model — set later: clank models  or  clank config set-model <name>",
	AskTestTools:      "check tool calls support for this profile? [Y/n]: ",
	ToolsSkipLater:    "done. you can check use_tools later: clank config test-tools",
	ModelsNeedCreds:   "set base_url and api_key first: clank config add %s",
	FetchingModels:    "fetching the model list",
	FilterNoMatch:     "nothing matches filter %q",
	ModelsEmptyAPI:    "API returned an empty model list",
	ChainHint:         "\nleft of the number — current position in the fallback chain",
	AskChainNumbers:   "\ncomma-separated numbers, first is primary (Enter — cancel): ",
	Cancelled:         "cancelled",
	BadNumber:         "bad number: %s (need 1 to %d)",
	ModelSet:          "model set:",
	ChainSet:          "chain set:",
	TestToolsCheck:    "checking tool calls on model",
	WaitingAnswer:     "waiting for answer",
	ToolCalled:        "model made a tool_call:",
	ToolNotCalled:     "model did NOT make a tool_call, answered with text:",
	ChainFirstOnly:    "only the first model of the chain was checked; use_tools is per profile",
	AskSaveTools:      "save use_tools=%v for profile %s? [Y/n]: ",
	SavedNot:          "not saved",
	TestReasonCheck:   "checking reasoning support on model",
	TestingParams:     "testing parameters",
	ReasonResults:     "\ncheck results:",
	ReasonNone:        "none",
	ReasonEffortOK:    "  reasoning_effort control: supported (on: %s, off: %s)\n",
	ReasonBudgetOK:    "  token budget control: supported (reasoning_budget)",
	ReasonNoParams:    "  reasoning control parameters: not supported by server",
	ReasonHasOutput:   "  reasoning block: found",
	ReasonNoOutput:    "  reasoning block: not found",
	ReasonUnsupported: "this model neither uses nor tunes reasoning",
	AskReasonOn:       "\nturn reasoning on for profile %s? [Y/n]: ",
	SavedReasonOn:     "saved (reasoning: on)",
	SettingsKept:      "settings unchanged",
	TestVisionCheck:   "checking vision (images) support on model",
	VisionOK:          "model answered the image request:",
	VisionJudge:       "judge yourself whether it really saw the picture before saving",
	VisionFail:        "model could NOT process the image",
	AskSaveVision:     "save vision=%v for profile %s? [Y/n]: ",
	TestModelCheck:    "checking model %s: tools, reasoning, vision...",
	AskSaveModel:      "save use_tools, reasoning and vision into profile %s? [Y/n]: ",
	YoloIsOn:          "YOLO mode: on (commands run without confirmation)",
	YoloIsOff:         "YOLO mode: off (commands need confirmation)",
	YoloGlobalOn:      "YOLO mode enabled globally",
	YoloGlobalOff:     "YOLO mode disabled globally",
	YoloBadArg:        "unknown argument: %s (use clank yolo on|off)",
	// client.go
	ErrSetup:   "profile setup",
	ErrNoConn:  "could not connect",
	ModelParen: " (model %s)",
	ErrShort:   "setup error",
	NoConn:     "no connection",
	HintAuth:   "key was rejected — check: clank config show",
	HintURL:    "check base_url (clank ask appends /v1/chat/completions itself): clank config show",
	HintModel:  "provider has no such model — pick again: clank models",
	HintCtx:    "context overflow — clear history (clank session clear) or pipe less data",
	HintRate:   "provider request limit",
	HintNet:    "network, proxy or address — check: clank config show",
	ProxyParse: "cannot parse proxy %q: %v — fix: clank config set-proxy <url|->",
	ProxyBlind: "proxy %q has no scheme or host (need http://host:port) — fix: clank config set-proxy <url|->",
	RespUnpars: "cannot parse answer: %v (first bytes: %s)",
	EmptyChoi:  "empty choices list in answer",
	ReqDetail:  "request: model %s, %d messages, %d bytes",
	RespDetail: "response %d in %.1fs, %d bytes",
	NoModels:   "profile has no models — pick: clank models",
	Answering:  "answering: %s",
	ModelFail:  "%s failed (%s), trying %s",
	RetryIn:    "%s: %s, retry in %s (%d/%d)",
	NoAnswer:   "no model answered:\n  %s\nlast error: %v",
	ModelsUnp:  "cannot parse model list: %v",
	APIError:   "API returned an error: %s",
	ToolProbeS: "You have a run_shell_command tool. If an exact answer needs real environment data — use it, don't guess.",
	ToolProbeU: "How many files and folders are directly in the current directory? If you can find out exactly with an available tool — do it.",
	VisionProb: "What color is this image? Answer briefly.",
	ImageBig:   "file too big (>20MB)",
	ImageType:  "file %s is not a supported image (type %s)",

	// lifecycle.go
	PurgeEmpty:   "config directory and data are already empty",
	PurgeAsk:     "wipe all settings, sessions and data (%s)? [y/N]: ",
	Aborted:      "aborted",
	PurgeFail:    "error wiping data (%s): %v",
	PurgeDone:    "all clank user data and settings wiped",
	BinPathFail:  "could not locate the binary: %v",
	UninstAsk:    "remove the clank executable (%s)? [y/N]: ",
	UninstFail:   "error removing %s: %v\n  (if permission is missing, run: sudo rm %s)",
	UninstDone:   "clank executable (%s) removed from the machine\n",
	NukeAsk:      "WARNING: this fully removes the binary (%s) and ALL data (%s). Continue? [y/N]: ",
	NukePartFail: "errors during full removal: %s",
	NukeDataErr:  "data (%v)",
	NukeBinErr:   "binary (%v)",
	NukeDone:     "clank fully removed from the machine (binary and all data)",

	// session.go
	SessPathFail:  "could not determine the session path: %v",
	SessFresh:     "no session in this terminal yet — starting fresh",
	SessReadFail:  "could not read session (%s): %v — starting fresh",
	SessCorrupt:   "session file is corrupt (%s): %v — starting fresh",
	SessSwitch:    "this terminal's session is from profile %s, active is %s — starting over",
	SessDetail:    "session: %d messages, updated %s",
	SessEmpty:     "this terminal's session is empty",
	SessReadErr:   "could not read session: %v",
	SessFileBad:   "session file is corrupt (%s): %v",
	SessHeader:    "profile: %s, messages: %d, updated: %s%s\n\n",
	MarkDone:      "[done]",
	MarkAnswer:    "[user answer]",
	MarkImage:     "[image loaded]",
	MarkAsks:      "[assistant asks] %s\n",
	MarkViews:     "[assistant views image] %s\n\n",
	MarkRuns:      "[assistant asks to run] %s(%s)\n\n",
	MarkAsst:      "[assistant] %s\n\n",
	SessRemoved:   "sessions removed: %d\n",
	NoSessions:    "no sessions",
	SessClearDone: "this terminal's session cleared",
	SessClearFail: "could not delete session: %v",
	SessClearOne:  "this terminal's session is already empty",
	StClosed:      "terminal closed",
	StShellAlive:  "shell %d alive",
	StHeadless:    "headless",
	SessRow:       "%s%-24s %s  %6d bytes  %s\n",

	// tty.go
	ConfirmHint:   "didn't get it: y — yes, n — no",
	NoTTYDecline:  "no controlling terminal — cannot confirm, declining",
	ModelWantsRun: "\nmodel wants to run:\n%s\n",
	ComplexWarn:   "  («%s» is allowed only for simple commands, but this one has shell specials — confirm manually)\n",
	RunPromptAll:  "run? [y/N/a] (a — always allow «%s», including complex): ",
	RunPrompt:     "run? [y/N/a] (a — always allow «%s»): ",
	ConfirmCMD:    "didn't get it: y — run, n — decline, a — always allow this command",
	NoTTYDefault:  "no controlling terminal — taking the default: %s",
	NoTTYNoAnswer: "no controlling terminal — cannot answer the question",
	ChooseOrType:  "pick 1-%d or type your answer: ",
	ChooseOrDef:   "pick 1-%d or type your answer [%s]: ",
	AnswerOrDef:   "answer [%s]: ",
	AnswerIs:      "answer: ",
	EnterNum:      "enter an option number or your answer",

	// update.go
	UpdBinFail:  "could not locate the current binary: %v",
	UpdLinkFail: "could not resolve the binary symlink: %v",
	UpdChecking: "checking GitHub for updates",
	UpdCheckErr: "could not check for updates: %v",
	UpdLatest:   "clank %s is already the latest\n",
	UpdAvail:    "New version available: %s (current: %s)\n",
	UpdNotes:    "\nRelease notes:\n%s\n\n",
	UpdCached:   "found a previously downloaded binary in cache: %s",
	UpdAsk:      "update %s to %s? [Y/n]: ",
	UpdAbort:    "update cancelled",
	UpdDown:     "downloading %s",
	UpdDownErr:  "could not download the release: %v",
	UpdUnpErr:   "could not unpack the binary: %v",
	UpdKept:     "kept in cache: %s",
	UpdCacheErr: "could not read cache: %v, downloading again",
	UpdApplyErr: "error installing the update into %s: %v",
	UpdSudoHint: "\nBinary kept in cache. If it is a permissions issue, run:\n  sudo clank update\n",
	UpdDone:     "clank successfully updated to %s!\n",
	UpdNoAsset:  "no suitable archive for %s/%s in release %s",
	UpdNoExeZip: "clank.exe not found inside the zip archive",
	UpdNoExeTar: "clank executable not found inside the tar.gz archive",

	// systemprompt_unix.go
	ShellApprox:  " (via $SHELL, approximate)",
	ShellUnknown: "unknown",

	LangNameRU: "Russian",
	LangNameEN: "English",
	FailPrefix: "error: ",
	LangShow:   "interface language:",
	LangSaved:  "language saved:",
	LangBadArg: "unknown language: %s (use clank language en|ru|auto)",

	HelpAsk: `clank <question> — ask the model. Piped stdin is picked up automatically.
Flags: -c (force stdin read), -r (resume your tab's session),
-i <file> (attach an image), --yolo=on|off, --reasoning=on|off,
-v (verbose) / -q (quiet), -- (the rest is question text).
A question starting with a command word (models, config…) needs the
explicit form: clank ask models …`,
	HelpConfigInit: `clank config init — setup wizard for the "default" profile.
Asks base_url and api_key step by step, offers model pick and a tool calls
check. In scripts use set-url/set-key/set-model instead of the wizard.`,
	HelpConfigAdd: `clank config add <name> — create or reconfigure a named profile.
Handy for one profile per provider, switch with config use.`,
	HelpConfigUse: `clank config use <name> — make a profile active.
All questions and tests go through the active profile.`,
	HelpConfigList: `clank config list — profiles, active one marked with *.
Shows url, models and tool calls state.`,
	HelpConfigRm: `clank config rm <name> — remove a profile. If it was active,
clank switches to the freshest remaining one by itself.`,
	HelpConfigShow: `clank config show — the active profile in full: url, key
(masked), models, reasoning, vision, yolo, language, allowed commands.`,
	HelpConfigSet: `clank config set-* — tweak the active profile:
set-url, set-key, set-model <a[,b,c]> (fallback chain), set-proxy <url|->,
set-tools <true|false>, set-reasoning <on|off|->, set-exec-timeout <sec>.`,
	HelpTestTools: `clank config test-tools — checks whether the model makes
a tool_call on a task that needs a tool for an exact answer. Shows the result;
with your confirmation writes use_tools into the profile.`,
	HelpTestReasoning: `clank config test-reasoning — finds thinking controls
(effort or token budget) and looks for thinking blocks in answers.
Prints what it found ("reasoning block: found / not found").
With your confirmation turns reasoning on in the profile.`,
	HelpTestVision: `clank config test-vision — sends the model a green square
asking its color and shows the answer. Whether it really saw the picture
is your call from the answer; with your confirmation the result is saved
into the profile (drives the prompt).`,
	HelpTestModel: `clank config test-model — all three checks at once (tools,
reasoning, vision) with a combined summary. One question — save all.`,
	HelpConfigAllow: `clank config allow-rm <command> — stop running a command
without confirmation. allow-clear — wipe the whole list.
Commands land there when you answer "a" to the "run?" prompt.`,
	HelpSessionShow: `clank session show — transcript of the current tab's session:
questions, answers, tool calls.`,
	HelpSessionClear: `clank session clear — wipe the current tab's session.
With --all — wipe every terminal's sessions.`,
	HelpSessionList: `clank session list — all sessions on the machine. * is the
current tab; shows whose shell is still alive and whose terminal is closed.`,
	HelpYolo: `clank yolo [on|off] — global no-confirmation mode for commands.
No args shows the state. For one reply — the --yolo=on|off flag.`,
	HelpLanguage: `clank language [en|ru|auto] — interface language.
No args shows the current one. auto — detect from the terminal LANG.`,
	HelpUpdate: `clank update [-y] — check GitHub releases and install the new
version over the current one. Downloads are cached: on permission errors
retry with sudo clank update, no re-download.`,
	HelpPurge: `clank purge [-y] — wipe ~/.config/clank/ (profiles, sessions, cache).
The binary stays. Factory reset for the settings.`,
	HelpUninstall: `clank uninstall [-y] — remove the clank executable.
Settings and sessions stay.`,
	HelpNuke: `clank nuke [-y] — remove both the binary and all data.
Point of no return; asks for confirmation.`,
	HelpUnknown: "no such help topic: %s",
	HelpTopics: `topics: ask models config session yolo language completion update purge uninstall nuke
config and session have nested topics: e.g. clank help config test-model`,

	CompletionUsage:     "usage: clank completion [zsh|bash|fish|install [shell]]",
	CompletionBadShell:  "unknown shell: %s (supported: zsh, bash, fish)",
	CompletionDone:      "completion installed: %s",
	CompletionWriteFail: "could not write completion: %v",
	CompletionZshHint:   "add to .zshrc if missing: fpath=(~/.zfunc $fpath)",
	CompletionRemoved:   "completion removed: %s",
	AskCompletion:       "install command completion for %s (%s)? [Y/n]: ",
	HelpCompletion: `clank completion [zsh|bash|fish] — print the completion script.
clank completion install — install it for the current shell.
Install may also be offered automatically on the first config init.`,
	Usage: `clank — a minimal CLI for OpenAI-compatible APIs

usage:
  clank <question>                      ask (stdin is picked up automatically if piped)
  clank -c <question>                   force stdin read if autodetect misses
  clank -r <question>                   continue this terminal's session (--resume)
  clank -c -r <question>                both pipe and resume work together
  clank -i <file> <question>            attach an image (PNG, JPG, WebP, GIF) to the question
  clank --yolo=on|off <question>        set YOLO mode for the session (on — no command confirmations)
  clank --reasoning=on|off <question>   force model reasoning on/off
  clank -v <question>                   verbose progress log (-q — quiet instead)
  clank -- <question>                   everything after is the question text, even if flag-like

  clank ask <question>                  same thing explicitly (needed if the question starts
                                        with models/config/session/help)

  clank models [filter]                 active profile models; chain pick: 1,4,7

  clank config init                     quick setup of the "default" profile
  clank config add <name>               named profile (for multiple providers)
  clank config use <name>               switch the active profile
  clank config list                     list profiles
  clank config rm <name>
  clank config show                     show the active profile
  clank config set-url/set-key/set-model/set-proxy/set-tools/set-reasoning/set-exec-timeout <value>
  clank config test-tools               check tool calls support on the model
  clank config test-reasoning           check and calibrate reasoning support on the model
  clank config test-vision              check and save image (vision) support on the model
  clank config test-model               all three checks at once with one save
  clank config allow-rm/allow-clear     commands that run without confirmation

  clank session show                    transcript of this terminal's session
  clank session clear [--all]           wipe the session (--all — every terminal)
  clank session list                    all sessions on the machine

  clank language [en|ru|auto]           interface language (no args — show)
  clank completion [shell|install]      completion script or its install
  clank yolo [on|off]                   global YOLO mode on/off (no command confirmations)
  clank update [-y]                     check for and install a new version from GitHub
  clank purge [-y]                      wipe all user data (~/.config/clank/)
  clank uninstall [-y]                  remove the clank executable
  clank nuke [-y]                       fully remove clank and all its data

  clank help <command>                help on a topic, nested too: help config test-model
  clank version

examples:
  clank "why doesn't grub see the second disk"
  dmesg | tail -50 | clank "figure out the error"
  clank -r "what about this: ..."

quote the question: without quotes ?, *, >, ! and backticks get eaten by the shell.`,
	SessionUsage: `usage: clank session <command>
  show              transcript of the current terminal's session
  clear             wipe the current terminal's session
  clear --all       wipe every terminal's sessions
  list              all sessions on the machine

each terminal tab has its own history; -r continues your own tab's`,
	ConfigUsage: `usage: clank config <command>
  init                       quick setup of the "default" profile
  add <name>                 create/edit a named profile
  use <name>                 make a profile active
  list                       list profiles
  rm <name>                  remove a profile
  show                       show the active profile
  set-url <url>              trailing /v1 is unneeded, stripped automatically
  set-key <key>
  set-model <a[,b,c]>        model chain: first fails — second goes
  set-proxy <url|->          '-' — back to env proxies
  set-tools <true|false>     manual use_tools override
  set-reasoning <on|off|->   reasoning on/off/reset
  set-exec-timeout <sec>     command ceiling (0 — default)
  test-tools                 check and (with confirmation) save use_tools
  test-reasoning             check and calibrate reasoning support
  test-vision                check and (with confirmation) save vision
  test-model                 all three checks at once (tools, reasoning, vision)
  allow-rm <command>         stop running a command without confirmation
  allow-clear                clear the allowed commands list`,
	ModelsHelp: `usage: clank models [filter]
  shows the active profile's models and picks a chain
  by comma-separated numbers: 1,4,7 — first is primary, rest are fallback`,
}

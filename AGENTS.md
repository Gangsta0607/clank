# AGENTS.md — clank codebase guide

> For AI agents working on this repo. Read this before touching any file.

---

## What is clank

A minimal CLI that wraps any OpenAI-compatible API. Core use case: pipe terminal output into it, ask a question, get an answer — without opening a browser. Secondary use case: let the model run real shell commands and ask interactive questions via registered tools (`run_shell_command`, `question`).

No dependencies outside the Go stdlib. Single binary. Config in `~/.config/clank/`.

---

## Build & install

```
make          # → bin/clank
make install  # → /usr/local/bin/clank (PREFIX overridable)
make check    # fmt + vet + test
```

Version is injected at link time from `date +%Y.%m.%d`. VCS is Fossil, not git — don't add git-based versioning.

Module: `clank` (go 1.22.2, zero external deps).

---

## Directory layout

All code is in package `main` (single package, flat directory). No subdirectories.

| File            | Responsibility                                              |
|-----------------|-------------------------------------------------------------|
| `main.go`       | Entry point, top-level command dispatch                     |
| `ask.go`        | Core ask loop, tool-call orchestration                      |
| `client.go`     | HTTP client, OpenAI-compatible API types, retry/fallback    |
| `config.go`     | Config file load/save, profile management, allowed-commands |
| `configcmd.go`  | `clank config *` and `clank models` subcommands             |
| `exec.go`       | Shell execution, output capping, command vetting            |
| `exec_*.go`     | Platform-specific shell execution (Unix/Windows)            |
| `session.go`    | Per-TTY session persistence, GC                             |
| `session_*.go`  | Platform-specific session identity (Unix/Windows)           |
| `systemprompt.go` | Builds the system prompt dynamically                      |
| `systemprompt_*.go` | Platform-specific shell detection (Unix/Windows)        |
| `term.go`       | Terminal column layout                                      |
| `term_*.go`     | Platform-specific terminal width (Unix/Windows)             |
| `tty.go`        | All interactive I/O (single reader), prompts                |
| `tty_*.go`      | Platform-specific TTY handle (Unix/Windows)                 |
| `ui.go`         | Logging levels, spinner, exit codes, string helpers         |

---

## Command dispatch (`main.go`)

```
os.Args[1] → switch:
  "ask"     → cmdAsk(rest)
  "models"  → cmdModels(rest)
  "config"  → cmdConfig(rest)
  "session" → cmdSession(rest)
  "version" → print version
  default   → cmdAsk(os.Args[1:])   ← the common path
```

The `default` branch means `clank "why is grub broken"` works without the `ask` subword. Conflict only when the question's first word is itself a subcommand name — for that, `clank ask ...` exists.

**Exit codes** (defined in `ui.go`):

| Constant      | Value | Meaning                                     |
|---------------|-------|---------------------------------------------|
| `exitOK`      | 0     | success                                     |
| `exitConfig`  | 1     | bad config / usage error                    |
| `exitAPI`     | 2     | API / network failure                       |
| `exitDeclined`| 3     | user interrupted execution (Ctrl-C)         |
| `exitNoAnswer`| 4     | model returned empty answer                 |

---

## The ask loop (`ask.go`)

### `cmdAsk(args []string) int`

Flags parsed manually (no `flag` package):
- `-c / --context` — force-read stdin even if it's a TTY
- `-r / --resume` — load previous session for this terminal
- `-v / --verbose` — set `verboseLevel = vVerbose`
- `-q / --quiet` — set `verboseLevel = vQuiet`
- `--` — everything after is question text (escape hatch for args starting with `-`)

**Flow:**

```
1. Parse flags
2. Load config + active profile, validate
3. gcSessions() — clean up dead terminal sessions
4. Read stdin if -c or stdin is not a TTY
5. Build userContent = question [+ context block if stdin present]
6. Load session history if --resume
7. Assemble messages = [system] + [history] + [user]
8. Enter tool loop:
     a. chatWithFallback() → chatResult
     b. Append assistant message to messages + turnMessages
     c. If no tool_calls → break (final answer)
     d. For each tool_call → runToolCall()
        - verdictInterrupted → break loop
     e. Append tool result message → continue
9. Print final answer to stdout
10. saveSession() on all exits (including error paths via finish())
```

**`turnMessages`** is what gets saved to disk: it starts from history (if resume), appends everything that happened this turn, but **never** includes the system message — the system prompt is rebuilt fresh every run (cwd and date change).

**`messages`** is what gets sent to the API: it includes the system message prepended, because the API requires it.

### `runToolCall(cf *ConfigFile, p Profile, tc toolCall, yolo bool) (chatMessage, toolVerdict)`

Validates and routes the tool call:

#### 1. `run_shell_command`
1. JSON-unmarshal `tc.Function.Arguments` → `{command: string}`
2. Reject empty command
3. Check if command is in `cf.Allowed` AND `isSimpleCommand()` (or if YOLO mode is active) → auto-execute
4. Otherwise call `confirmCommand()` → `ansYes / ansNo / ansAlways`
   - `ansAlways` → `cf.allow(name, level)` + `saveConfigFile()`
   - `ansNo` → return `verdictDeclined`
5. `execShell()` → `execResult`
6. Return tool message with stdout/stderr/exit code

#### 2. `question`
1. JSON-unmarshal `tc.Function.Arguments` → `{question: string, options: []string, default: string}`
2. Call `promptQuestion(qText, options, default)` to interactively ask the user on `/dev/tty`
3. Return the selected/custom option or default, with `verdictOK` (or `verdictInterrupted` on Ctrl-C/EOF)

**Verdicts:**

| Constant            | Meaning                                     |
|---------------------|---------------------------------------------|
| `verdictOK`         | executed, continue loop                     |
| `verdictInvalid`    | bad tool request, model gets error message  |
| `verdictDeclined`   | user said no, tool refusal message sent to model, continue loop |
| `verdictInterrupted`| Ctrl-C during execution, break loop         |

---

## API client (`client.go`)

### Key types

```go
chatMessage  {Role, Content, ToolCalls, ToolCallID}
toolCall     {ID, Type, Function{Name, Arguments}}
toolDef      {Type, Function{Name, Description, Parameters}}
chatRequest  {Model, Messages, Stream=false, Tools}
chatResponse {Choices[]{Message{Role, Content, ToolCalls, ReasoningContent}, FinishReason}, Error}
chatResult   {msg chatMessage, model string, finishReason string}
```

`Content` has `omitempty` because some strict servers reject an empty string where they expect `null` on assistant messages that only have `tool_calls`.

### Call stack

```
cmdAsk
  └─ chatWithFallback(profile, messages, useTools)
       └─ for each model in profile.Models:
            chatCompleteRetry(profile, model, messages, useTools)
              └─ [up to maxRetries=3 with exponential backoff]
                   chatComplete(profile, model, messages, useTools)
                     └─ apiRequest(profile, "POST", "/v1/chat/completions", body, model)
                          └─ httpClient(profile) → http.Client [proxy aware]
```

### Fallback logic

`chatWithFallback` walks `profile.Models` in order. A model is retried (same model) for:
- `errTransport` (network)
- `errRateLimit` (429)
- `errServer` (5xx)

A model triggers advancement to the next model for the above **plus** `errModelMissing` (model doesn't exist at provider).

Auth errors (`errAuth`), context overflow (`errContextOverflow`), and bad requests (`errBadRequest`) **do not** trigger fallback — they'll fail on any model in the same profile.

### Error classification (`classify`)

`classify(statusCode int, bodyText string) errKind` — matches HTTP status + body text patterns against known error kinds. Body-text matching is needed because providers use status 400 for wildly different things.

### Reasoning model support

Reasoning blocks (`<think>`, `<thought>`, `<reasoning>`, and `reasoning_content`) are stripped and never exposed to the user.
Reasoning can be toggled via `clank config set-reasoning on|off` or `--reasoning=on|off`.
`clank config test-reasoning` auto-detects and calibrates model parameters (`max` → `ultra` → `xhigh` → `high` for ON; `none` → `minimal` → `low` for OFF; or token budgets).

### Tool definition

Two tools are registered: `run_shell_command` and `question`. Their JSON schemas are hardcoded. The system prompt instructs the model on how to use them.

---

## Config (`config.go`)

### File location

`~/.config/clank/config.json` (hardcoded — intentionally consistent across macOS and Linux, avoids `~/Library/Application Support` on macOS).

### `ConfigFile` structure

```go
ConfigFile {
    Active   string                // name of currently active profile
    Profiles map[string]Profile
    Allowed  map[string]TrustLevel // command names with trust levels ("simple" or "all")
    Yolo     bool                  // global YOLO mode (bypass all prompts)
}
```

### `Profile` structure

```go
Profile {
    BaseURL        string    // without trailing /v1 — normalizeBaseURL() strips it
    APIKey         string    // stored plaintext, file is 0600
    Models         []string  // fallback chain; [0] is primary
    Model          string    // legacy compat mirror of Models[0]
    Proxy          string    // explicit proxy; empty = use HTTP_PROXY/HTTPS_PROXY env
    UseTools       bool      // whether to include tool definition in API requests
    ExecTimeoutSec int       // 0 = defaultExecTimeout (10 min)
    Created        string    // RFC3339, used for auto-selection when Active is stale
}
```

### `normalizeBaseURL`

Strips trailing `/v1` (and any repetitions of it). The client always appends `/v1/chat/completions` or `/v1/models` itself. Without this, configs copied from provider docs (which include `/v1`) would produce `/v1/v1/...`.

### `activeProfile`

If `cf.Active` is set and exists → return it. If not (deleted profile, empty config), auto-selects by `newestProfileName` (most recent `Created` timestamp, ties broken alphabetically), saves the selection, and logs a notice. Prevents the "no active profile" error after `config rm`.

### Allowed commands

`cf.Allowed` is a sorted list of command names (first word only). When the model requests a command whose name is in this list **and** the command passes `isSimpleCommand()`, it runs without prompting. Managed via `clank config allow-rm` / `allow-clear`.

---

## Shell execution (`exec.go`)

### `execShell(command string, timeout time.Duration) execResult`

- Runs `sh -c <command>`
- Stdout and stderr go to **both** the user's stdout (live) and a `cappedWriter` buffer
- Stdin of the child is `/dev/tty` (not inherited stdin) — allows `sudo` password prompts even when clank's own stdin is a pipe
- Signal handling: `SIGINT` (Ctrl-C) is caught; child is in the **same process group** so it also receives SIGINT naturally. clank waits up to 3s for child to die, then `SIGKILL`. This is intentional — putting child in its own group would make it background and break `sudo` / `read` prompts via SIGTTIN.
- Timeout: `SIGTERM` first, then `SIGKILL` after 5s
- Returns `execResult{output, exitCode, interrupted, timedOut}`

### `cappedWriter`

Caps output at `maxToolOutputBytes = 10MB`. Keeps **head** and **tail** (each `limit/2`), drops the middle with a note. Cuts on UTF-8 rune boundaries so the model doesn't see partial Cyrillic characters. `Write` always reports success to avoid breaking `io.MultiWriter`.

### `isSimpleCommand(command string) bool`

Heuristic, not a shell parser. Returns false if command contains any of: `; & | < > \n \r ( ) { }` or `$( ${ `` $((`. Commands that pass are safe to auto-execute by name — complex ones require manual confirmation even if the name is in the allowed list.

### `commandName(command string) string`

Returns the binary name from the first non-assignment token, stripping leading path components. `FOO=bar /usr/bin/ls -la` → `ls`.

---

## Session management (`session.go`)

### Session identity

Sessions are per-TTY, not per-user. ID is `<tty_rdev_hex>-<ppid>`:
- `rdev` of `/dev/tty` — identifies the terminal device
- `ppid` — the shell PID of the terminal tab

This distinguishes multiple tabs sharing the same tty device (different ppid) and survives terminal device name reuse after close (different rdev after reopen).

Headless environments (no `/dev/tty`) use the fixed ID `"headless"`.

### Storage

`~/.config/clank/sessions/<id>.json`

```go
sessionFile {
    Profile  string
    Updated  time.Time
    Messages []chatMessage  // no system message; rebuilt fresh each run
    Yolo     *bool          // session YOLO mode
}
```

File permissions: `0600`.

### `loadSession(profile string) []chatMessage`

- Profile mismatch check: if session was written by a different profile, discards it. Reason: session may contain `role:"tool"` messages; a profile without `use_tools` will get HTTP 400 on those.
- Logs verbose detail on what it loaded, warns loudly on corruption.

### `saveSession`

Called via `finish()` closure in `cmdAsk` — runs **on all exit paths** including API errors. A session interrupted halfway still records what happened, so `-r` can resume from the actual state.

### `gcSessions`

Runs at the start of every `cmdAsk`. Removes session files older than `sessionMaxAge = 7 days` where the parent shell process (from the filename's pid component) is no longer alive (`kill(pid, 0)` returns ESRCH).

---

## System prompt (`systemprompt.go`)

### `buildSystemPrompt(useTools bool) string`

Rebuilt every invocation. Contains:
- Current working directory
- Hostname
- Current date (minute precision)
- Detected shell (via `ps -p <ppid> -o comm=`, falls back to `$SHELL`)
- OS/arch (`runtime.GOOS/GOARCH`)
- Explicit "no markdown" instruction (no `**`, `###`, triple backticks, bullet `*`)
- Tool usage instructions if `useTools == true`, else "suggest a one-liner pipe to clank -r"

The no-markdown rule is critical: the terminal doesn't render markdown, it just prints the asterisks.

### `detectShell`

Uses `ps -p <ppid>` to find the actual current shell, not `$SHELL` (which reflects the login shell, not the currently running one). Works on both Linux and macOS without `/proc`.

---

## TTY and interactive I/O (`tty.go`)

**Single global `/dev/tty` reader** (`ttyOnce sync.Once`). This is non-negotiable: `bufio.Reader` buffers ahead; two readers on the same fd would lose data. All interactive prompts use `ttyIO()`.

stdin is deliberately **not** used for prompts — it may be carrying a context pipe.

### Functions

| Function                                   | Purpose                                          |
|--------------------------------------------|--------------------------------------------------|
| `ttyIO() (*os.File, *bufio.Reader, error)` | Open/return the single tty handle                |
| `haveTTY() bool`                           | Whether we have a terminal at all                |
| `askLine(prompt) (string, bool)`           | Read one line from tty; ok=false on EOF/no-tty   |
| `askLineDefault(label, current, display)`  | Like askLine but returns current on empty input  |
| `confirmYN(prompt, defaultYes) bool`       | Y/N prompt with re-ask on garbage input          |
| `confirmCommand(cmdStr, name, known) answer` | Show command, ask y/N/a                        |

`confirmYN` loops on unrecognized input instead of treating it as "no" (prevents typos silently declining).

`confirmCommand` writes both the command text and the prompt to the **same fd** (`/dev/tty`). Previously they went to different fds; with `2>log` the user would see "execute?" with no command text.

---

## Terminal utilities (`term.go`)

### `termWidth() int`

Order: ioctl on stdout → ioctl on stderr → ioctl on /dev/tty → `$COLUMNS` env → 80.

### `layoutColumns(cells []string, width int) []string`

Column-major layout (like `ls`): numbers read top-to-bottom within each column. Used for model selection display. Pads with spaces, trims trailing spaces per line.

---

## Logging (`ui.go`)

All diagnostic output → **stderr**. All model output and command stdout → **stdout**. This means `clank "..." > file` captures both the model answer and command output in the file.

| Function         | Level condition     | Prefix   | Notes                              |
|------------------|---------------------|----------|------------------------------------|
| `detail()`       | `vVerbose` only     | `  · `   | API timing, message counts         |
| `info()`         | `vNormal` and above | none     | profile selection, spinner, etc.   |
| `warn()`         | always              | `! `     | data loss risk, degraded operation |
| `fail()`         | always              | `ошибка:` | fatal errors before exit          |

### Spinner

`startSpinner(msg) *spinner` — starts a goroutine writing animated `- \ | /` frames to stderr. Only runs when `verboseLevel != vQuiet` and stderr is a TTY. `stopSpinner()` closes the goroutine and clears the line with `\r\033[K`.

### `sanitizeForDisplay(s string)`

Replaces control characters (except `\n` and `\t`) with `\xNN` / `\uNNNN` escape sequences. Prevents ANSI escape sequences or `\r` from overwriting already-printed text in the confirmation prompt — otherwise a model response like `"echo ok\r\033[2Krm -rf ~"` could display "rm -rf ~" in the confirmation while actually running "echo ok".

---

## Data flow: full request lifecycle

```
User: clank -c -r "what's wrong with my nginx config"
  │
  ├─ stdin → readStdin() → context string
  ├─ loadSession(profile) → []chatMessage (history)
  ├─ buildSystemPrompt(useTools) → system message
  │
  ├─ messages = [system, ...history, user("Context:\n<stdin>\n\nQuestion: ...")]
  │
  └─ loop (i < 1000):
       ├─ chatWithFallback()
       │    ├─ chatCompleteRetry(model[0])  ← try up to 3× with backoff
       │    │    └─ chatComplete()
       │    │         └─ apiRequest() → POST /v1/chat/completions
       │    └─ on failure → try model[1], model[2], ...
       │
       ├─ res.msg has ToolCalls? NO → print res.msg.Content → done
       │
       └─ YES → for each toolCall:
                  ├─ validate tool name
                  ├─ parse JSON args → cmdStr
                  ├─ isSimpleCommand && isAllowed? → auto-run
                  │   else confirmCommand() → y/N/a
                  └─ execShell(cmdStr, timeout)
                       ├─ stdout → user's stdout (live) + cappedWriter
                       └─ return execResult → tool message → back to loop
```

---

## Key invariants / design rules

1. **No external deps.** Stdlib only. Don't add modules.

2. **`content omitempty` on chatMessage.** Some servers reject `"content": ""` on assistant messages that have `tool_calls`. Do not change this.

3. **System message excluded from saved session.** `turnMessages` never gets the system prompt. It's always rebuilt from current env at startup.

4. **Save session on all exit paths.** The `finish()` closure in `cmdAsk` is called with `defer`-equivalent pattern at every `return`. Don't add bare `return exitFoo` without going through `finish()`.

5. **Single tty reader.** Never open a second `bufio.Reader` on `/dev/tty`. All prompts must go through `ttyIO()`.

6. **Child process in same process group.** `execShell` does NOT call `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}`. This is intentional: sudo password prompts need tty access and would get SIGTTIN if backgrounded.

7. **Fallback triggers only on technical failures.** "Model gave a bad answer" does not trigger model fallback. Only transport/rate/server/missing-model errors do.

8. **`normalizeBaseURL` is idempotent.** Strips all trailing `/v1` suffixes in a loop. Safe to call multiple times.

9. **All user-visible command text goes through `sanitizeForDisplay`.** Never print raw model output as a command without sanitizing.

10. **`cappedWriter.Write` always returns `(n, nil)`.** Non-negotiable: returning an error from Write breaks `io.MultiWriter` and kills command output mid-run.

---

## Config file format (reference)

```json
{
  "active": "home",
  "profiles": {
    "home": {
      "base_url": "https://api.openai.com",
      "api_key": "sk-...",
      "models": ["gpt-4o", "gpt-4o-mini"],
      "model": "gpt-4o",
      "proxy": "",
      "use_tools": true,
      "exec_timeout_sec": 0,
      "created": "2024-11-01T10:00:00Z"
    }
  },
  "allowed_commands": ["git", "ls", "cat"]
}
```

`model` (singular) is always kept in sync with `models[0]` by `saveConfigFile`. It exists solely for backward compatibility with older clank versions.

---

## Things to watch out for

- **`clank config add <name>`** works on the named profile, not the active one. Before this was fixed, the wizard silently modified the active profile instead.
- **`clank config use <name>`** after `config rm <active>` auto-switches to newest remaining profile. After the fix, explicit `use` persists correctly.
- **`-r` with a different profile** in session file: history is discarded with a notice. Don't suppress that notice.
- **Reasoning models** (`reasoning_content` field, `<think>` blocks): handled in `chatComplete`. If adding a new model family with different output conventions, extend `chatComplete`, not the callers.
- **`confimYN` with no TTY** returns `defaultYes`. In CI/cron this means setup wizards auto-accept defaults — which is the intended behavior.
- **Spinner** is nil-safe: `startSpinner` returns `nil` when quiet/non-TTY, and `stopSpinner`/`setMessage` check `if s == nil` first.

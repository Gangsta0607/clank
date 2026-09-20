package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Сессия привязана к терминалу, а не к пользователю: в разных вкладках,
// окнах и панелях tmux своя история, и `-r` подхватывает историю именно
// той вкладки, где его набрали. Раньше файл был один на всех, и два
// параллельных clank затирали друг друга.
//
// Идентификатор — номер устройства /dev/tty плюс PID процесса-родителя
// (шелла вкладки). Одного tty мало: имена устройств переиспользуются, и
// свежая вкладка подхватила бы чужую историю. Одного PPID мало: он не
// различает панели и не переживает запуск из скрипта.
type sessionFile struct {
	Profile  string        `json:"profile"`
	Updated  time.Time     `json:"updated"`
	Messages []chatMessage `json:"messages"`
	Yolo     *bool         `json:"yolo,omitempty"`
}

const sessionMaxAge = 7 * 24 * time.Hour

func sessionsDir() (string, error) {
	dir, err := clankDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sessions"), nil
}

func sessionPath() (string, error) {
	dir, err := sessionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sessionID()+".json"), nil
}

// loadSession возвращает историю текущего терминала. Про любую проблему
// сообщает вслух: раньше повреждённый файл молча давал пустую историю, и
// пользователь не понимал, почему модель забыла контекст.
func loadSession(profile string) ([]chatMessage, *bool) {
	path, err := sessionPath()
	if err != nil {
		warn(M.SessPathFail, err)
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		info(M.SessFresh)
		return nil, nil
	}
	if err != nil {
		warn(M.SessReadFail, path, err)
		return nil, nil
	}

	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		warn(M.SessCorrupt, path, err)
		return nil, nil
	}

	// В истории могут лежать tool_calls и role:"tool". Если новый профиль
	// не умеет инструменты, сервер ответит 400 на такие сообщения — так
	// что чужую историю не тащим.
	if sf.Profile != "" && sf.Profile != profile {
		info(M.SessSwitch,
			sf.Profile, profile)
		return nil, nil
	}

	detail(M.SessDetail, len(sf.Messages), sf.Updated.Format("15:04:05"))
	return sf.Messages, sf.Yolo
}

func saveSession(profile string, messages []chatMessage, yolo *bool) error {
	dir, err := sessionsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path, err := sessionPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(sessionFile{
		Profile:  profile,
		Updated:  time.Now(),
		Messages: messages,
		Yolo:     yolo,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// gcSessions подчищает файлы мёртвых терминалов: процесс-родитель уже не
// существует и файл давно не трогали.
func gcSessions() {
	dir, err := sessionsDir()
	if err != nil {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		if e.IsDir() || name == e.Name() || name == "headless" {
			continue
		}
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < sessionMaxAge {
			continue
		}
		if pid := pidFromSessionName(name); pid > 0 && processAlive(pid) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

func pidFromSessionName(name string) int {
	i := strings.LastIndex(name, "-")
	if i < 0 {
		return 0
	}
	pid, err := strconv.Atoi(name[i+1:])
	if err != nil {
		return 0
	}
	return pid
}

// --- команда session ---

func cmdSession(args []string) int {
	if len(args) == 0 {
		printSessionUsage()
		return exitConfig
	}

	switch args[0] {
	case "show":
		return sessionShow()
	case "clear":
		return sessionClear(len(args) > 1 && (args[1] == "--all" || args[1] == "-a"))
	case "list":
		return sessionList()
	case "-h", "--help", "help":
		printSessionUsage()
		return exitOK
	default:
		fail(M.UnknownSubcommand, args[0])
		printSessionUsage()
		return exitConfig
	}
}

func sessionShow() int {
	path, err := sessionPath()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		fmt.Println(M.SessEmpty)
		return exitOK
	}
	if err != nil {
		fail(M.SessReadErr, err)
		return exitConfig
	}
	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		fail(M.SessFileBad, path, err)
		return exitConfig
	}
	if len(sf.Messages) == 0 {
		fmt.Println(M.SessEmpty)
		return exitOK
	}

	yoloStr := ""
	if sf.Yolo != nil {
		if *sf.Yolo {
			yoloStr = ", yolo: on"
		} else {
			yoloStr = ", yolo: off"
		}
	}
	fmt.Printf(M.SessHeader,
		valueOr(sf.Profile, "?"), len(sf.Messages), sf.Updated.Format("2006-01-02 15:04:05"), yoloStr)

	toolCallNames := make(map[string]string)
	for _, m := range sf.Messages {
		if m.Role == "assistant" {
			for _, tc := range m.ToolCalls {
				toolCallNames[tc.ID] = tc.Function.Name
			}
		}

		switch m.Role {
		case "tool":
			prefix := M.MarkDone
			if toolCallNames[m.ToolCallID] == questionToolName {
				prefix = M.MarkAnswer
			} else if toolCallNames[m.ToolCallID] == viewImageToolName {
				prefix = M.MarkImage
			}
			fmt.Printf("%s\n%s\n\n", prefix, indentBlock(truncateRunes(m.Text(), 2000), "    "))
		case "assistant":
			if len(m.ToolCalls) > 0 {
				for _, tc := range m.ToolCalls {
					if tc.Function.Name == questionToolName {
						var qArgs struct {
							Question string   `json:"question"`
							Options  []string `json:"options"`
						}
						_ = json.Unmarshal([]byte(tc.Function.Arguments), &qArgs)
						fmt.Printf(M.MarkAsks, sanitizeForDisplay(qArgs.Question))
						for i, opt := range qArgs.Options {
							fmt.Printf("    %d) %s\n", i+1, sanitizeForDisplay(opt))
						}
						fmt.Println()
					} else if tc.Function.Name == viewImageToolName {
						var imgArgs struct {
							Path string `json:"path"`
						}
						_ = json.Unmarshal([]byte(tc.Function.Arguments), &imgArgs)
						fmt.Printf(M.MarkViews, sanitizeForDisplay(imgArgs.Path))
					} else {
						fmt.Printf(M.MarkRuns,
							tc.Function.Name, sanitizeForDisplay(tc.Function.Arguments))
					}
				}
			}
			if strings.TrimSpace(m.Text()) != "" {
				fmt.Printf(M.MarkAsst, m.Text())
			}
		default:
			fmt.Printf("[%s] %s\n\n", m.Role, m.Text())
		}
	}
	return exitOK
}

func sessionClear(all bool) int {
	dir, err := sessionsDir()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}

	if all {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Println(M.NoSessions)
			return exitOK
		}
		n := 0
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".json") {
				if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
					n++
				}
			}
		}
		fmt.Printf(M.SessRemoved, n)
		return exitOK
	}

	path, err := sessionPath()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	err = os.Remove(path)
	switch {
	case os.IsNotExist(err):
		fmt.Println(M.SessClearOne)
	case err != nil:
		fail(M.SessClearFail, err)
		return exitConfig
	default:
		fmt.Println(M.SessClearDone)
	}
	return exitOK
}

func sessionList() int {
	dir, err := sessionsDir()
	if err != nil {
		fail("%v", err)
		return exitConfig
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		fmt.Println(M.NoSessions)
		return exitOK
	}

	current := sessionID()
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		if name == e.Name() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		mark := "  "
		if name == current {
			mark = "* "
		}
		state := M.StClosed
		if pid := pidFromSessionName(name); pid > 0 && processAlive(pid) {
			state = fmt.Sprintf(M.StShellAlive, pid)
		} else if name == "headless" {
			state = M.StHeadless
		}
		fmt.Printf(M.SessRow,
			mark, name, fi.ModTime().Format("2006-01-02 15:04"), fi.Size(), state)
	}
	return exitOK
}

func printSessionUsage() {
	fmt.Fprintln(os.Stderr, M.SessionUsage)
}

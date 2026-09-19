package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
}

const sessionMaxAge = 7 * 24 * time.Hour

func sessionsDir() (string, error) {
	dir, err := clankDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sessions"), nil
}

func sessionID() string {
	f, _, err := ttyIO()
	if err != nil {
		// cron, CI, пайплайн без терминала — общий файл на всех
		return "headless"
	}
	var st syscall.Stat_t
	if err := syscall.Fstat(int(f.Fd()), &st); err != nil {
		return "headless"
	}
	return fmt.Sprintf("%x-%d", uint64(st.Rdev), os.Getppid())
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
func loadSession(profile string) []chatMessage {
	path, err := sessionPath()
	if err != nil {
		warn("не смог определить путь сессии: %v", err)
		return nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		info("сессии в этом терминале ещё нет — начинаю с чистого листа")
		return nil
	}
	if err != nil {
		warn("не смог прочитать сессию (%s): %v — начинаю с чистого листа", path, err)
		return nil
	}

	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		warn("файл сессии повреждён (%s): %v — начинаю с чистого листа", path, err)
		return nil
	}

	// В истории могут лежать tool_calls и role:"tool". Если новый профиль
	// не умеет инструменты, сервер ответит 400 на такие сообщения — так
	// что чужую историю не тащим.
	if sf.Profile != "" && sf.Profile != profile {
		info("сессия в этом терминале от профиля %s, сейчас активен %s — начинаю заново",
			sf.Profile, profile)
		return nil
	}

	detail("сессия: %d сообщений, обновлена %s", len(sf.Messages), sf.Updated.Format("15:04:05"))
	return sf.Messages
}

func saveSession(profile string, messages []chatMessage) error {
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

// processAlive — сигнал 0 не доставляется, но проверяет существование
// процесса: ESRCH означает, что терминала с таким шеллом больше нет.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
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
		fail("неизвестная подкоманда: %s", args[0])
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
		fmt.Println("сессия этого терминала пуста")
		return exitOK
	}
	if err != nil {
		fail("не смог прочитать сессию: %v", err)
		return exitConfig
	}
	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		fail("файл сессии повреждён (%s): %v", path, err)
		return exitConfig
	}
	if len(sf.Messages) == 0 {
		fmt.Println("сессия этого терминала пуста")
		return exitOK
	}

	fmt.Printf("профиль: %s, сообщений: %d, обновлена: %s\n\n",
		valueOr(sf.Profile, "?"), len(sf.Messages), sf.Updated.Format("2006-01-02 15:04:05"))

	for _, m := range sf.Messages {
		switch m.Role {
		case "tool":
			fmt.Printf("[выполнено]\n%s\n\n", indentBlock(truncateRunes(m.Content, 2000), "    "))
		case "assistant":
			if len(m.ToolCalls) > 0 {
				for _, tc := range m.ToolCalls {
					fmt.Printf("[assistant просит выполнить] %s(%s)\n\n",
						tc.Function.Name, sanitizeForDisplay(tc.Function.Arguments))
				}
			}
			if strings.TrimSpace(m.Content) != "" {
				fmt.Printf("[assistant] %s\n\n", m.Content)
			}
		default:
			fmt.Printf("[%s] %s\n\n", m.Role, m.Content)
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
			fmt.Println("сессий нет")
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
		fmt.Printf("удалено сессий: %d\n", n)
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
		fmt.Println("сессия этого терминала и так пуста")
	case err != nil:
		fail("не смог удалить сессию: %v", err)
		return exitConfig
	default:
		fmt.Println("сессия этого терминала очищена")
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
		fmt.Println("сессий нет")
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
		state := "терминал закрыт"
		if pid := pidFromSessionName(name); pid > 0 && processAlive(pid) {
			state = fmt.Sprintf("шелл %d жив", pid)
		} else if name == "headless" {
			state = "без терминала"
		}
		fmt.Printf("%s%-24s %s  %6d байт  %s\n",
			mark, name, fi.ModTime().Format("2006-01-02 15:04"), fi.Size(), state)
	}
	return exitOK
}

func printSessionUsage() {
	fmt.Fprintln(os.Stderr, `использование: clank session <команда>
  show              транскрипт сессии текущего терминала
  clear             стереть сессию текущего терминала
  clear --all       стереть сессии всех терминалов
  list              все сессии на машине

у каждой вкладки терминала своя история; -r продолжает историю своей вкладки`)
}

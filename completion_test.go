package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// caseLabels вытаскивает case "xxx": из switch в исходнике — чтобы тест
// замечал новые команды/подкоманды, забытые в скриптах автодополнения.
func caseLabels(t *testing.T, file, fn string) []string {
	t.Helper()
	src, err := os.ReadFile(fn)
	if err != nil {
		t.Fatal(err)
	}
	_ = file
	re := regexp.MustCompile(`(?m)^\tcase "([a-z-]+)":`)
	var out []string
	seen := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

func TestCompletionCoversCommands(t *testing.T) {
	scripts := map[string]string{
		"zsh-ru":  zshCompletionRU,
		"zsh-en":  zshCompletionEN,
		"fish-ru": fishCompletionRU,
		"fish-en": fishCompletionEN,
		"bash":    bashCompletion,
	}
	for _, cmd := range caseLabels(t, "main", "main.go") {
		for shell, s := range scripts {
			if !containsWord(s, cmd) {
				t.Errorf("%s script misses top command %q", shell, cmd)
			}
		}
	}
	for _, sub := range caseLabels(t, "config", "configcmd.go") {
		if sub == "-h" || sub == "--help" || sub == "help" {
			continue
		}
		for shell, s := range scripts {
			if !containsWord(s, sub) {
				t.Errorf("%s script misses config subcommand %q", shell, sub)
			}
		}
	}
	for _, flag := range []string{"-c", "-r", "-v", "-q", "-i", "--yolo", "--reasoning", "--image"} {
		for shell, s := range scripts {
			if !containsWord(s, flag) {
				t.Errorf("%s script misses ask flag %q", shell, flag)
			}
		}
	}
}

func containsWord(s, w string) bool {
	re := regexp.MustCompile(`(^|[^a-z-])` + regexp.QuoteMeta(w) + `([^a-z-]|$)`)
	return re.MatchString(s)
}

func TestCompletionInstallRoundtrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cf := ConfigFile{Profiles: map[string]Profile{}}
	path, err := installCompletion(&cf, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("script not written: %v", err)
	}
	if len(cf.Completions) != 1 || cf.Completions[0] != path {
		t.Fatalf("path not recorded: %v", cf.Completions)
	}
	if filepath.Base(path) != "_clank" {
		t.Fatalf("unexpected zsh target: %s", path)
	}
	removeCompletions(cf)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("script not removed")
	}
}

func TestCompletionUnknownShell(t *testing.T) {
	applyLang(LangEN)
	if code := printCompletionScript("powershell"); code != exitConfig {
		t.Fatalf("expected exitConfig, got %d", code)
	}
}

// reinstallCompletions ставит вариант под текущий язык: был EN — после
// переключения на RU в файле должны быть русские описания.
func TestCompletionReinstallFollowsLanguage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	applyLang(LangEN)
	cf := ConfigFile{Profiles: map[string]Profile{}}
	path, err := installCompletion(&cf, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "ask the model") {
		t.Fatalf("expected EN script, got:\n%.200s", data)
	}
	applyLang(LangRU)
	reinstallCompletions(&cf)
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "спросить модель") {
		t.Fatalf("expected RU script after reinstall, got:\n%.200s", data)
	}
	applyLang(LangEN)
}

func TestNormalizeShellToken(t *testing.T) {
	for in, want := range map[string]string{
		"-zsh":                          "zsh",
		"-bash":                         "bash",
		"zsh":                           "zsh",
		"zsh (via $SHELL, approximate)": "zsh",
		"Fish":                          "fish",
		"":                              "unknown",
	} {
		if got := normalizeShellToken(in); got != want {
			t.Errorf("normalizeShellToken(%q) = %q, want %q", in, got, want)
		}
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Профиль — один набор доступа к API (можно держать несколько:
// домашний сервер, VPS, платный провайдер и т.п.)
type Profile struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`

	// Models — цепочка моделей по приоритету: не ответила первая, идёт
	// вторая и так далее. Model оставлено для обратной совместимости со
	// старым конфигом; при записи дублирует Models[0].
	Models []string `json:"models"`
	Model  string   `json:"model,omitempty"`

	Proxy    string `json:"proxy"`     // пусто = брать из HTTP_PROXY/HTTPS_PROXY/NO_PROXY
	UseTools bool   `json:"use_tools"` // выставляется через `clank config test-tools`

	ExecTimeoutSec int    `json:"exec_timeout_sec,omitempty"` // 0 = дефолт
	Created        string `json:"created,omitempty"`
}

// ConfigFile — весь файл конфига: набор профилей, какой активен и список
// команд, которые пользователь разрешил выполнять без вопросов.
type ConfigFile struct {
	Active   string             `json:"active"`
	Profiles map[string]Profile `json:"profiles"`
	Allowed  []string           `json:"allowed_commands,omitempty"`
}

// Путь фиксирован явно (не os.UserConfigDir()) — на macOS это дало бы
// ~/Library/Application Support, а на Linux ~/.config. Раз тулза живёт
// на обеих платформах, путь один и тот же везде.
func clankDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "clank"), nil
}

func configPath() (string, error) {
	dir, err := clankDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// normalizeBaseURL снимает хвостовой /v1: канонический base_url у OpenAI и
// большинства совместимых провайдеров уже с ним (https://api.openai.com/v1),
// а clank сам дописывает /v1/chat/completions. Без нормализации выходило
// /v1/v1/... и глухой 404.
func normalizeBaseURL(u string) string {
	u = strings.TrimRight(strings.TrimSpace(u), "/")
	for strings.HasSuffix(u, "/v1") {
		u = strings.TrimSuffix(u, "/v1")
		u = strings.TrimRight(u, "/")
	}
	return u
}

func loadConfigFile() (ConfigFile, error) {
	cf := ConfigFile{Profiles: map[string]Profile{}}
	path, err := configPath()
	if err != nil {
		return cf, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cf, nil
	}
	if err != nil {
		return cf, err
	}
	if err := json.Unmarshal(data, &cf); err != nil {
		return cf, fmt.Errorf("конфиг повреждён (%s): %w", path, err)
	}
	if cf.Profiles == nil {
		cf.Profiles = map[string]Profile{}
	}
	// Миграция со старого формата с единственной моделью.
	for name, p := range cf.Profiles {
		if len(p.Models) == 0 && p.Model != "" {
			p.Models = []string{p.Model}
			cf.Profiles[name] = p
		}
	}
	return cf, nil
}

func saveConfigFile(cf ConfigFile) error {
	dir, err := clankDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// Model держим синхронным с Models[0] — чтобы старая версия clank
	// (или чужой скрипт) не увидела профиль без модели.
	for name, p := range cf.Profiles {
		if len(p.Models) > 0 {
			p.Model = p.Models[0]
		} else {
			p.Model = ""
		}
		cf.Profiles[name] = p
	}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	// 0600 — в профилях api_key плейнтекстом
	return os.WriteFile(path, data, 0600)
}

// activeProfile возвращает активный профиль. Если активный не задан или
// указывает в пустоту, но профили есть — переключается сам (на самый
// свежий, при равенстве — первый по алфавиту) и сохраняет выбор. Раньше
// после `config rm` активного профиля любая команда отвечала «нет
// активного профиля — создай», хотя другие профили никуда не делись.
func activeProfile(cf *ConfigFile) (string, Profile, error) {
	if cf.Active != "" {
		if p, ok := cf.Profiles[cf.Active]; ok {
			return cf.Active, p, nil
		}
		warn("активный профиль %q не найден в конфиге", cf.Active)
	}
	if len(cf.Profiles) == 0 {
		return "", Profile{}, fmt.Errorf("профилей нет — создай: clank config init")
	}

	name := newestProfileName(*cf)
	cf.Active = name
	if err := saveConfigFile(*cf); err != nil {
		warn("не смог сохранить выбор активного профиля: %v", err)
	}
	info("активный профиль не задан, переключаюсь на %s (сменить: clank config use <имя>)", name)
	return name, cf.Profiles[name], nil
}

func newestProfileName(cf ConfigFile) string {
	names := profileNames(cf)
	best := names[0]
	var bestTime time.Time
	for _, n := range names {
		t, err := time.Parse(time.RFC3339, cf.Profiles[n].Created)
		if err != nil {
			continue
		}
		if t.After(bestTime) {
			bestTime, best = t, n
		}
	}
	return best
}

func profileNames(cf ConfigFile) []string {
	names := make([]string, 0, len(cf.Profiles))
	for n := range cf.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (p Profile) validate() error {
	var missing []string
	if p.BaseURL == "" {
		missing = append(missing, "base_url")
	}
	if p.APIKey == "" {
		missing = append(missing, "api_key")
	}
	if len(p.Models) == 0 {
		missing = append(missing, "model")
	}
	if len(missing) > 0 {
		return fmt.Errorf("профиль не настроен, не хватает: %s — заполни: clank config add <имя>",
			strings.Join(missing, ", "))
	}
	return nil
}

func (p Profile) execTimeout() time.Duration {
	if p.ExecTimeoutSec > 0 {
		return time.Duration(p.ExecTimeoutSec) * time.Second
	}
	return defaultExecTimeout
}

func maskKey(key string) string {
	if key == "" {
		return "<не задан>"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// --- список разрешённых команд ---

func (cf *ConfigFile) isAllowed(name string) bool {
	for _, a := range cf.Allowed {
		if a == name {
			return true
		}
	}
	return false
}

func (cf *ConfigFile) allow(name string) {
	if name == "" || cf.isAllowed(name) {
		return
	}
	cf.Allowed = append(cf.Allowed, name)
	sort.Strings(cf.Allowed)
}

func (cf *ConfigFile) disallow(name string) bool {
	for i, a := range cf.Allowed {
		if a == name {
			cf.Allowed = append(cf.Allowed[:i], cf.Allowed[i+1:]...)
			return true
		}
	}
	return false
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

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

	Reasoning       *bool  `json:"reasoning,omitempty"`        // nil = авто/дефолт, true = on, false = off
	ReasoningEffort string `json:"reasoning_effort,omitempty"` // калиброванное значение effort
	ReasoningBudget int    `json:"reasoning_budget,omitempty"` // калиброванный токеновый бюджет (если модель использует budget)

	// Vision — умеет ли модель разбирать изображения. nil = не проверяли
	// (считаем, что умеет), выставляется через `clank config test-vision`.
	Vision *bool `json:"vision,omitempty"`

	ExecTimeoutSec int    `json:"exec_timeout_sec,omitempty"` // 0 = дефолт
	Created        string `json:"created,omitempty"`
}

// TrustLevel задаёт уровень доверия к выполнению команды без подтверждения:
// - TrustNone: не разрешена, всегда запрашивать
// - TrustSimple: разрешены простые команды (без спецсимволов шелла)
// - TrustAll: разрешены любые команды, включая конвейеры и перенаправления
type TrustLevel string

const (
	TrustNone   TrustLevel = ""
	TrustSimple TrustLevel = "simple"
	TrustAll    TrustLevel = "all"
)

// ConfigFile — весь файл конфига: набор профилей, какой активен, список
// команд с их уровнями доверия и глобальный флаг YOLO.
type ConfigFile struct {
	Active   string                `json:"active"`
	Profiles map[string]Profile    `json:"profiles"`
	Allowed  map[string]TrustLevel `json:"allowed_commands,omitempty"`
	Yolo     bool                  `json:"yolo,omitempty"`
	// Language — язык интерфейса: "ru", "en", пусто = автоопределение
	// по LANG/LC_ALL/LC_MESSAGES. Меняется через `clank language`.
	Language string `json:"language,omitempty"`
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

func (cf *ConfigFile) UnmarshalJSON(data []byte) error {
	var raw struct {
		Active          string             `json:"active"`
		Profiles        map[string]Profile `json:"profiles"`
		AllowedCommands json.RawMessage    `json:"allowed_commands"`
		Yolo            bool               `json:"yolo"`
		Language        string             `json:"language"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	cf.Active = raw.Active
	cf.Profiles = raw.Profiles
	cf.Yolo = raw.Yolo
	cf.Language = raw.Language
	cf.Allowed = make(map[string]TrustLevel)

	if len(raw.AllowedCommands) > 0 {
		var m map[string]TrustLevel
		if err := json.Unmarshal(raw.AllowedCommands, &m); err == nil {
			cf.Allowed = m
		} else {
			var s []string
			if err := json.Unmarshal(raw.AllowedCommands, &s); err == nil {
				for _, cmd := range s {
					cf.Allowed[cmd] = TrustSimple
				}
			}
		}
	}
	return nil
}

func loadConfigFile() (ConfigFile, error) {
	cf := ConfigFile{Profiles: map[string]Profile{}, Allowed: make(map[string]TrustLevel)}
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
		return cf, fmt.Errorf(M.ConfigCorrupt, path, err)
	}
	if cf.Profiles == nil {
		cf.Profiles = map[string]Profile{}
	}
	if cf.Allowed == nil {
		cf.Allowed = make(map[string]TrustLevel)
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
		warn(M.ActiveNotFound, cf.Active)
	}
	if len(cf.Profiles) == 0 {
		if cf.Profiles == nil {
			cf.Profiles = make(map[string]Profile)
		}
		cf.Profiles["default"] = Profile{}
		cf.Active = "default"
		return "default", cf.Profiles["default"], nil
	}

	name := newestProfileName(*cf)
	cf.Active = name
	if err := saveConfigFile(*cf); err != nil {
		warn(M.ActiveSaveFail, err)
	}
	info(M.ActiveSwitch, name)
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
		return fmt.Errorf(M.ProfileInvalid,
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
		return M.KeyMissing
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// --- список разрешённых команд ---

func (cf *ConfigFile) trustLevel(name string) TrustLevel {
	if cf.Allowed == nil {
		return TrustNone
	}
	return cf.Allowed[name]
}

func (cf *ConfigFile) isAllowed(name string) bool {
	return cf.trustLevel(name) != TrustNone
}

func (cf *ConfigFile) allow(name string, level TrustLevel) {
	if name == "" {
		return
	}
	if cf.Allowed == nil {
		cf.Allowed = make(map[string]TrustLevel)
	}
	cf.Allowed[name] = level
}

func (cf *ConfigFile) disallow(name string) bool {
	if cf.Allowed == nil {
		return false
	}
	if _, ok := cf.Allowed[name]; ok {
		delete(cf.Allowed, name)
		return true
	}
	return false
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

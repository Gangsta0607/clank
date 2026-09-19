package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	httpTimeout = 120 * time.Second
	maxRetries  = 3 // на одну модель, до перехода к следующей в цепочке
)

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Content с omitempty: у assistant-сообщения с tool_calls текста нет, и
// часть строгих OpenAI-совместимых серверов отвергает пустую строку там,
// где ждёт отсутствующее поле или null.
type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolDef struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

const shellToolName = "run_shell_command"

func runShellCommandTool() toolDef {
	var t toolDef
	t.Type = "function"
	t.Function.Name = shellToolName
	t.Function.Description = "Выполнить shell-команду в POSIX-совместимом окружении пользователя и получить её stdout/stderr/exit code."
	t.Function.Parameters = json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "Команда для sh -c, одна строка"
			}
		},
		"required": ["command"]
	}`)
	return t
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Tools    []toolDef     `json:"tools,omitempty"`
}

type errorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []toolCall `json:"tool_calls"`
			// reasoning-модели (deepseek-r1 и родня) кладут текст сюда,
			// а content оставляют пустым
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *errorBody `json:"error,omitempty"`
}

type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Error *errorBody `json:"error,omitempty"`
}

// --- классификация ошибок ---

type errKind int

const (
	errOther errKind = iota
	errTransport
	errAuth
	errRateLimit
	errServer
	errModelMissing
	errContextOverflow
	errBadRequest
	errNotFound
	errConfig // кривой конфиг: повторять и переключать модель бессмысленно
)

type apiError struct {
	kind   errKind
	status int
	msg    string
	url    string
	model  string
}

func (e *apiError) Error() string {
	var b strings.Builder
	switch {
	case e.status > 0:
		fmt.Fprintf(&b, "HTTP %d", e.status)
	case e.kind == errConfig:
		b.WriteString("настройка профиля")
	default:
		b.WriteString("не удалось соединиться")
	}
	if e.model != "" {
		fmt.Fprintf(&b, " (модель %s)", e.model)
	}
	if e.msg != "" {
		fmt.Fprintf(&b, ": %s", truncateRunes(e.msg, 400))
	}
	if hint := e.hint(); hint != "" {
		fmt.Fprintf(&b, "\n  %s", hint)
	}
	return b.String()
}

// short — однострочная причина для сообщений о переключении модели.
func (e *apiError) short() string {
	switch {
	case e.status > 0:
		return fmt.Sprintf("HTTP %d", e.status)
	case e.kind == errConfig:
		return "ошибка настройки"
	}
	return "нет связи"
}

func (e *apiError) hint() string {
	switch e.kind {
	case errAuth:
		return "ключ не принят — проверь: clank config show"
	case errNotFound:
		return "проверь base_url (clank ask сам дописывает /v1/chat/completions): clank config show"
	case errModelMissing:
		return "такой модели у провайдера нет — выбери заново: clank models"
	case errContextOverflow:
		return "контекст переполнен — очисти историю (clank session clear) или подавай меньше данных в пайп"
	case errRateLimit:
		return "лимит запросов у провайдера"
	case errTransport:
		return "сеть, прокси или адрес — проверь: clank config show"
	}
	return ""
}

// retryable — есть смысл повторить тот же запрос к той же модели.
func (e *apiError) retryable() bool {
	switch e.kind {
	case errTransport, errRateLimit, errServer:
		return true
	}
	return false
}

// tryNextModel — есть смысл перейти к следующей модели цепочки.
// Ошибка ключа или переполнение контекста воспроизведутся на любой
// модели того же профиля, так что цепочку они не запускают.
func (e *apiError) tryNextModel() bool {
	return e.retryable() || e.kind == errModelMissing
}

func classify(status int, body string) errKind {
	low := strings.ToLower(body)
	switch {
	case status == 401 || status == 403:
		return errAuth
	case status == 429:
		return errRateLimit
	case status >= 500:
		return errServer
	}
	if containsAny(low, "context length", "context_length", "maximum context", "too many tokens",
		"reduce the length", "context window", "prompt is too long") {
		return errContextOverflow
	}
	if containsAny(low, "model not found", "model_not_found", "unknown model", "no such model",
		"does not exist", "invalid model",
		// mlx_lm и другие локальные серверы при неизвестном имени модели
		// пытаются скачать её с HuggingFace и отдают его 404
		"repository not found") {
		return errModelMissing
	}
	if status == 404 {
		return errNotFound
	}
	if status >= 400 {
		return errBadRequest
	}
	return errOther
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// --- транспорт ---

func httpClient(p Profile) (*http.Client, error) {
	transport := &http.Transport{}
	if p.Proxy != "" {
		u, err := url.Parse(p.Proxy)
		if err != nil {
			// Раньше кривой прокси молча приводил к прямому соединению:
			// пользователь думал, что трафик идёт через прокси, а он не шёл.
			return nil, fmt.Errorf("не разобрать proxy %q: %w — поправь: clank config set-proxy <url|->", p.Proxy, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("proxy %q без схемы или хоста (нужно вида http://host:port) — поправь: clank config set-proxy <url|->", p.Proxy)
		}
		transport.Proxy = http.ProxyURL(u)
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{Transport: transport, Timeout: httpTimeout}, nil
}

func apiRequest(p Profile, method, path string, body []byte, model string) ([]byte, error) {
	endpoint := normalizeBaseURL(p.BaseURL) + path
	client, err := httpClient(p)
	if err != nil {
		// Профиль настроен неправильно — ни повтор, ни другая модель
		// этого не исправят.
		return nil, &apiError{kind: errConfig, msg: err.Error(), url: endpoint, model: model}
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, &apiError{kind: errTransport, msg: err.Error(), url: endpoint, model: model}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("User-Agent", "clank/"+version)

	detail("%s %s", method, endpoint)
	started := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, &apiError{kind: errTransport, msg: err.Error(), url: endpoint, model: model}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &apiError{kind: errTransport, msg: err.Error(), url: endpoint, model: model}
	}
	detail("ответ %d за %.1fс, %d байт", resp.StatusCode, time.Since(started).Seconds(), len(respBody))

	// Любой 2xx считаем успехом: некоторые прокси отвечают 201.
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := extractErrorMessage(respBody)
		return nil, &apiError{
			kind:   classify(resp.StatusCode, msg+" "+string(respBody)),
			status: resp.StatusCode,
			msg:    msg,
			url:    endpoint,
			model:  model,
		}
	}
	return respBody, nil
}

// extractErrorMessage вытаскивает человеческий текст из тела ошибки.
// Раньше наружу вываливался сырой JSON целиком.
func extractErrorMessage(body []byte) string {
	var wrapped struct {
		Error   *errorBody `json:"error"`
		Message string     `json:"message"`
		Detail  string     `json:"detail"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		switch {
		case wrapped.Error != nil && wrapped.Error.Message != "":
			return wrapped.Error.Message
		case wrapped.Message != "":
			return wrapped.Message
		case wrapped.Detail != "":
			return wrapped.Detail
		}
	}
	return strings.TrimSpace(string(body))
}

// --- запросы ---

type chatResult struct {
	msg          chatMessage
	model        string
	finishReason string
}

// chatComplete — один запрос к одной модели.
func chatComplete(p Profile, model string, messages []chatMessage, useTools bool) (chatResult, error) {
	reqStruct := chatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}
	if useTools {
		reqStruct.Tools = []toolDef{runShellCommandTool()}
	}
	reqBody, err := json.Marshal(reqStruct)
	if err != nil {
		return chatResult{}, err
	}
	detail("запрос: модель %s, сообщений %d, %d байт", model, len(messages), len(reqBody))

	respBody, err := apiRequest(p, http.MethodPost, "/v1/chat/completions", reqBody, model)
	if err != nil {
		return chatResult{}, err
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return chatResult{}, &apiError{kind: errOther, model: model,
			msg: fmt.Sprintf("ответ не разобрать: %v (первые байты: %s)", err, truncateRunes(string(respBody), 200))}
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return chatResult{}, &apiError{kind: classify(0, parsed.Error.Message), model: model, msg: parsed.Error.Message}
	}
	if len(parsed.Choices) == 0 {
		return chatResult{}, &apiError{kind: errServer, model: model, msg: "пустой список choices в ответе"}
	}

	c := parsed.Choices[0]
	content := c.Message.Content
	if strings.TrimSpace(content) == "" && c.Message.ReasoningContent != "" {
		content = c.Message.ReasoningContent
		detail("content пуст, беру reasoning_content")
	}
	content = stripThinkBlocks(content)

	return chatResult{
		msg: chatMessage{
			Role:      "assistant",
			Content:   content,
			ToolCalls: c.Message.ToolCalls,
		},
		model:        model,
		finishReason: c.FinishReason,
	}, nil
}

// stripThinkBlocks убирает <think>…</think>: reasoning-модели иногда
// отдают рассуждения прямо в content, и пользователю они не нужны.
func stripThinkBlocks(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], "</think>")
		if end < 0 {
			// открывающий тег без закрывающего — обрезали по лимиту,
			// оставляем как есть, чтобы не потерять весь ответ
			break
		}
		s = s[:start] + s[start+end+len("</think>"):]
	}
	return strings.TrimSpace(s)
}

// chatWithFallback идёт по цепочке моделей профиля. Переключается только
// на технических отказах (сеть, 5xx, 429, нет такой модели) — «плохой по
// смыслу» ответ фоллбэк не запускает, иначе каждый неудачный ответ молча
// прожигал бы всю цепочку.
func chatWithFallback(p Profile, messages []chatMessage, useTools bool) (chatResult, error) {
	if len(p.Models) == 0 {
		return chatResult{}, fmt.Errorf("в профиле нет ни одной модели — выбери: clank models")
	}

	var (
		attempts []string
		lastErr  error
	)
	for i, model := range p.Models {
		res, err := chatCompleteRetry(p, model, messages, useTools)
		if err == nil {
			if i > 0 {
				info("отвечает %s", model)
			}
			return res, nil
		}
		lastErr = err

		ae, ok := err.(*apiError)
		if !ok {
			return chatResult{}, err
		}
		attempts = append(attempts, fmt.Sprintf("%s — %s", model, ae.short()))
		if !ae.tryNextModel() {
			return chatResult{}, err
		}
		if i < len(p.Models)-1 {
			info("%s не ответил (%s), пробую %s", model, ae.short(), p.Models[i+1])
		}
	}

	// При единственной модели список попыток ничего не добавляет, а
	// подсказку из исходной ошибки терять жалко.
	if len(attempts) == 1 {
		return chatResult{}, lastErr
	}
	return chatResult{}, fmt.Errorf("ни одна модель не ответила:\n  %s\nпоследняя ошибка: %v",
		strings.Join(attempts, "\n  "), lastErr)
}

// chatCompleteRetry повторяет запрос при временных отказах: 429 и 5xx у
// публичных провайдеров — штатная ситуация, ронять из-за них весь ход глупо.
func chatCompleteRetry(p Profile, model string, messages []chatMessage, useTools bool) (chatResult, error) {
	var lastErr error
	backoff := time.Second
	for attempt := 1; attempt <= maxRetries; attempt++ {
		res, err := chatComplete(p, model, messages, useTools)
		if err == nil {
			return res, nil
		}
		lastErr = err

		ae, ok := err.(*apiError)
		if !ok || !ae.retryable() || attempt == maxRetries {
			return chatResult{}, err
		}
		info("%s: %s, повтор через %s (%d/%d)", model, ae.short(), backoff, attempt, maxRetries-1)
		time.Sleep(backoff)
		backoff *= 2
	}
	return chatResult{}, lastErr
}

func listModels(p Profile) ([]string, error) {
	respBody, err := apiRequest(p, http.MethodGet, "/v1/models", nil, "")
	if err != nil {
		return nil, err
	}
	var parsed modelsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("список моделей не разобрать: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("API вернул ошибку: %s", parsed.Error.Message)
	}
	ids := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// testToolSupport — эвристическая проверка: просим модель что-то, для
// чего "честный" ответ требует вызова инструмента, и смотрим, вызвала
// ли она tool_call. Не 100% надёжно (мелкие модели ведут себя по-разному),
// поэтому результат перед сохранением подтверждается пользователем.
func testToolSupport(p Profile, model string) (ok bool, detailText string, err error) {
	sys := chatMessage{
		Role:    "system",
		Content: "У тебя есть инструмент run_shell_command. Если для точного ответа нужно посмотреть реальные данные окружения — используй его, а не гадай.",
	}
	user := chatMessage{
		Role:    "user",
		Content: "Сколько файлов и папок лежит прямо в текущей директории? Если можешь узнать точно через доступный инструмент — узнай.",
	}
	res, err := chatComplete(p, model, []chatMessage{sys, user}, true)
	if err != nil {
		return false, "", err
	}
	if len(res.msg.ToolCalls) > 0 {
		tc := res.msg.ToolCalls[0]
		return true, fmt.Sprintf("%s(%s)", tc.Function.Name, tc.Function.Arguments), nil
	}
	return false, res.msg.Content, nil
}

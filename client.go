package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

type imageURL struct {
	URL string `json:"url"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

// Content с omitempty: у assistant-сообщения с tool_calls текста нет, и
// часть строгих OpenAI-совместимых серверов отвергает пустую строку там,
// где ждёт отсутствующее поле или null. Content может быть string или []contentPart.
type chatMessage struct {
	Role       string     `json:"role"`
	Content    any        `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

func (m chatMessage) Text() string {
	if s, ok := m.Content.(string); ok {
		return s
	}
	if parts, ok := m.Content.([]contentPart); ok {
		var sb strings.Builder
		for _, p := range parts {
			if p.Type == "text" {
				sb.WriteString(p.Text)
			}
		}
		return sb.String()
	}
	if parts, ok := m.Content.([]any); ok {
		var sb strings.Builder
		for _, p := range parts {
			if pm, ok := p.(map[string]any); ok {
				if pm["type"] == "text" {
					if t, ok := pm["text"].(string); ok {
						sb.WriteString(t)
					}
				}
			}
		}
		return sb.String()
	}
	if m.Content != nil {
		return fmt.Sprint(m.Content)
	}
	return ""
}

type toolDef struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

const (
	shellToolName     = "run_shell_command"
	questionToolName  = "question"
	viewImageToolName = "view_image"
)

func runShellCommandTool() toolDef {
	var t toolDef
	t.Type = "function"
	t.Function.Name = shellToolName
	t.Function.Description = "Run a shell command in the user's POSIX environment via sh -c and get its stdout/stderr/exit code."
	t.Function.Parameters = json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "Single-line command for sh -c"
			}
		},
		"required": ["command"]
	}`)
	return t
}

func questionTool() toolDef {
	var t toolDef
	t.Type = "function"
	t.Function.Name = questionToolName
	t.Function.Description = "Ask the user a question, offer a choice of options, or request clarification/input."
	t.Function.Parameters = json.RawMessage(`{
		"type": "object",
		"properties": {
			"question": {
				"type": "string",
				"description": "Question text for the user"
			},
			"options": {
				"type": "array",
				"items": {
					"type": "string"
				},
				"description": "List of options for the user to choose from (optional)"
			},
			"default": {
				"type": "string",
				"description": "Default option or value on Enter (optional)"
			}
		},
		"required": ["question"]
	}`)
	return t
}

func viewImageTool() toolDef {
	var t toolDef
	t.Type = "function"
	t.Function.Name = viewImageToolName
	t.Function.Description = "Look at a local image file (PNG, JPEG, WebP, GIF) to analyze the picture, read text (OCR), or inspect graphics."
	t.Function.Parameters = json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the image file (relative or absolute)"
			}
		},
		"required": ["path"]
	}`)
	return t
}

// requestTools собирает набор инструментов для запроса. view_image отдаём
// только если поддержка vision не опровергнута (Profile.Vision == false):
// слепой модели схема ни к чему, она всё равно получит ошибку.
func requestTools(p Profile, force bool) []toolDef {
	if !p.UseTools && !force {
		return nil
	}
	tools := []toolDef{runShellCommandTool(), questionTool()}
	if p.Vision == nil || *p.Vision {
		tools = append(tools, viewImageTool())
	}
	return tools
}

func encodeImageFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) > 20<<20 {
		return "", fmt.Errorf(M.ImageBig)
	}
	mime := http.DetectContentType(data)
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		mime = "image/png"
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	}
	if !strings.HasPrefix(mime, "image/") {
		return "", fmt.Errorf(M.ImageType, path, mime)
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mime, b64), nil
}

type chatRequest struct {
	Model             string        `json:"model"`
	Messages          []chatMessage `json:"messages"`
	Stream            bool          `json:"stream"`
	Tools             []toolDef     `json:"tools,omitempty"`
	ReasoningEffort   string        `json:"reasoning_effort,omitempty"`
	ReasoningBudget   *int          `json:"reasoning_budget,omitempty"`
	MaxThinkingTokens *int          `json:"max_thinking_tokens,omitempty"`
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
		b.WriteString(M.ErrSetup)
	default:
		b.WriteString(M.ErrNoConn)
	}
	if e.model != "" {
		fmt.Fprintf(&b, M.ModelParen, e.model)
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
		return M.ErrShort
	}
	return M.NoConn
}

func (e *apiError) hint() string {
	switch e.kind {
	case errAuth:
		return M.HintAuth
	case errNotFound:
		return M.HintURL
	case errModelMissing:
		return M.HintModel
	case errContextOverflow:
		return M.HintCtx
	case errRateLimit:
		return M.HintRate
	case errTransport:
		return M.HintNet
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
			return nil, fmt.Errorf(M.ProxyParse, p.Proxy, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf(M.ProxyBlind, p.Proxy)
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
	detail(M.RespDetail, resp.StatusCode, time.Since(started).Seconds(), len(respBody))

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
	if useTools := requestTools(p, useTools); useTools != nil {
		reqStruct.Tools = useTools
	}
	if p.Reasoning != nil {
		if *p.Reasoning {
			if p.ReasoningBudget > 0 {
				b := p.ReasoningBudget
				reqStruct.ReasoningBudget = &b
				reqStruct.MaxThinkingTokens = &b
			} else if p.ReasoningEffort != "" {
				reqStruct.ReasoningEffort = p.ReasoningEffort
			} else {
				reqStruct.ReasoningEffort = "max"
			}
		} else {
			if p.ReasoningBudget > 0 {
				b := 0
				reqStruct.ReasoningBudget = &b
				reqStruct.MaxThinkingTokens = &b
			} else if p.ReasoningEffort != "" {
				reqStruct.ReasoningEffort = p.ReasoningEffort
			} else {
				reqStruct.ReasoningEffort = "none"
			}
		}
	}
	reqBody, err := json.Marshal(reqStruct)
	if err != nil {
		return chatResult{}, err
	}
	detail(M.ReqDetail, model, len(messages), len(reqBody))

	respBody, err := apiRequest(p, http.MethodPost, "/v1/chat/completions", reqBody, model)
	if err != nil {
		return chatResult{}, err
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return chatResult{}, &apiError{kind: errOther, model: model,
			msg: fmt.Sprintf(M.RespUnpars, err, truncateRunes(string(respBody), 200))}
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return chatResult{}, &apiError{kind: classify(0, parsed.Error.Message), model: model, msg: parsed.Error.Message}
	}
	if len(parsed.Choices) == 0 {
		return chatResult{}, &apiError{kind: errServer, model: model, msg: M.EmptyChoi}
	}

	c := parsed.Choices[0]
	// Размышления (reasoning_content и теги <think>) никогда не подставляются
	// в ответ пользователю.
	content := stripThinkBlocks(c.Message.Content)

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

// stripThinkBlocks убирает <think>…</think>, <thought>…</thought>,
// <reasoning>…</reasoning>: размышления не должны засорять вывод пользователю.
func stripThinkBlocks(s string) string {
	tags := []string{"think", "thought", "reasoning"}
	for _, tag := range tags {
		open := "<" + tag + ">"
		close := "</" + tag + ">"
		for {
			start := strings.Index(s, open)
			if start < 0 {
				break
			}
			end := strings.Index(s[start:], close)
			if end < 0 {
				// открывающий тег без закрывающего (обрыв по лимиту токенов) —
				// вырезаем всё от открывающего тега до конца, чтобы мысли не утекли
				s = s[:start]
				break
			}
			s = s[:start] + s[start+end+len(close):]
		}
	}
	return strings.TrimSpace(s)
}

// chatWithFallback идёт по цепочке моделей профиля. Переключается только
// на технических отказах (сеть, 5xx, 429, нет такой модели) — «плохой по
// смыслу» ответ фоллбэк не запускает, иначе каждый неудачный ответ молча
// прожигал бы всю цепочку.
func chatWithFallback(p Profile, messages []chatMessage, useTools bool) (chatResult, error) {
	if len(p.Models) == 0 {
		return chatResult{}, fmt.Errorf(M.NoModels)
	}

	var (
		attempts []string
		lastErr  error
	)
	for i, model := range p.Models {
		res, err := chatCompleteRetry(p, model, messages, useTools)
		if err == nil {
			if i > 0 {
				info(M.Answering, model)
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
			info(M.ModelFail, model, ae.short(), p.Models[i+1])
		}
	}

	// При единственной модели список попыток ничего не добавляет, а
	// подсказку из исходной ошибки терять жалко.
	if len(attempts) == 1 {
		return chatResult{}, lastErr
	}
	return chatResult{}, fmt.Errorf(M.NoAnswer,
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
		info(M.RetryIn, model, ae.short(), backoff, attempt, maxRetries-1)
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
		return nil, fmt.Errorf(M.ModelsUnp, err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf(M.APIError, parsed.Error.Message)
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
		Content: M.ToolProbeS,
	}
	user := chatMessage{
		Role:    "user",
		Content: M.ToolProbeU,
	}
	res, err := chatComplete(p, model, []chatMessage{sys, user}, true)
	if err != nil {
		return false, "", err
	}
	if len(res.msg.ToolCalls) > 0 {
		tc := res.msg.ToolCalls[0]
		return true, fmt.Sprintf("%s(%s)", tc.Function.Name, tc.Function.Arguments), nil
	}
	return false, res.msg.Text(), nil
}

type reasoningTestResult struct {
	EffortOn   string
	EffortOff  string
	BudgetOn   int
	BudgetOff  int
	UsesBudget bool
	HasOutput  bool
}

func testReasoningSupport(p Profile, model string) (reasoningTestResult, error) {
	var res reasoningTestResult

	sendTest := func(reqStruct chatRequest) (chatResponse, error) {
		reqBody, err := json.Marshal(reqStruct)
		if err != nil {
			return chatResponse{}, err
		}
		respBody, err := apiRequest(p, http.MethodPost, "/v1/chat/completions", reqBody, model)
		if err != nil {
			return chatResponse{}, err
		}
		var parsed chatResponse
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return chatResponse{}, err
		}
		return parsed, nil
	}

	baseMsg := []chatMessage{{Role: "user", Content: "2+2=? Reply with the digit only / Ответь только цифрой"}}

	// 1. Тестируем effort для ON в порядке: max, ultra, xhigh, high
	for _, val := range []string{"max", "ultra", "xhigh", "high"} {
		resp, err := sendTest(chatRequest{Model: model, Messages: baseMsg, ReasoningEffort: val})
		if err == nil {
			res.EffortOn = val
			if len(resp.Choices) > 0 {
				c := resp.Choices[0].Message
				if c.ReasoningContent != "" || strings.Contains(c.Content, "<think>") || strings.Contains(c.Content, "<thought>") {
					res.HasOutput = true
				}
			}
			break
		}
	}

	// 2. Тестируем effort для OFF в порядке: none, minimal, low
	for _, val := range []string{"none", "minimal", "low"} {
		_, err := sendTest(chatRequest{Model: model, Messages: baseMsg, ReasoningEffort: val})
		if err == nil {
			res.EffortOff = val
			break
		}
	}

	// 3. Если effort не поддержан, пробуем reasoning_budget / max_thinking_tokens
	if res.EffortOn == "" && res.EffortOff == "" {
		bOn := 1024
		resp, err := sendTest(chatRequest{Model: model, Messages: baseMsg, ReasoningBudget: &bOn})
		if err == nil {
			res.UsesBudget = true
			res.BudgetOn = 32768
			res.BudgetOff = 0
			if len(resp.Choices) > 0 {
				c := resp.Choices[0].Message
				if c.ReasoningContent != "" || strings.Contains(c.Content, "<think>") || strings.Contains(c.Content, "<thought>") {
					res.HasOutput = true
				}
			}
		} else {
			resp, err := sendTest(chatRequest{Model: model, Messages: baseMsg, MaxThinkingTokens: &bOn})
			if err == nil {
				res.UsesBudget = true
				res.BudgetOn = 32768
				res.BudgetOff = 0
				if len(resp.Choices) > 0 {
					c := resp.Choices[0].Message
					if c.ReasoningContent != "" || strings.Contains(c.Content, "<think>") || strings.Contains(c.Content, "<thought>") {
						res.HasOutput = true
					}
				}
			}
		}
	}

	// 4. Проверяем без параметров, генерирует ли модель размышления сама по себе
	if !res.HasOutput {
		resp, err := sendTest(chatRequest{Model: model, Messages: baseMsg})
		if err == nil && len(resp.Choices) > 0 {
			c := resp.Choices[0].Message
			if c.ReasoningContent != "" || strings.Contains(c.Content, "<think>") || strings.Contains(c.Content, "<thought>") {
				res.HasOutput = true
			}
		}
	}

	return res, nil
}

const testPNG32x32Base64 = "iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAIAAAD8GO2jAAAAJklEQVR4nO3NsQkAAAjAsP7/tF7hIASyp6ZbAoFAIBAIBAKB4EuwNof8LkGrxSIAAAAASUVORK5CYII="

func testVisionSupport(p Profile, model string) (ok bool, detailText string, err error) {
	dataURL := "data:image/png;base64," + testPNG32x32Base64
	userMsg := chatMessage{
		Role: "user",
		Content: []contentPart{
			{Type: "text", Text: M.VisionProb},
			{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
		},
	}
	res, err := chatComplete(p, model, []chatMessage{userMsg}, false)
	if err != nil {
		return false, "", err
	}
	ans := strings.TrimSpace(res.msg.Text())
	return true, ans, nil
}

// Package llm – подключение языковой модели. Провайдер выбирается переменной LLM_PROVIDER:
// mock (по умолчанию, работает без ключа и без сети) или openai_compat – любой OpenAI-совместимый
// API: YandexGPT (AI Studio), Cloud.ru Foundation Models, Avito A-Vibe через Ollama.
package llm

import (
	"context"
	"encoding/json"
	"errors"
)

// Message – сообщение диалога в формате chat completions.
type Message struct {
	Role       string     `json:"role"` // system, user, assistant, tool
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Tool – инструмент, который модель может вызвать. Parameters – JSON Schema аргументов.
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// ToolCall – решение модели вызвать инструмент.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Request – запрос к модели.
type Request struct {
	Messages []Message
	Tools    []Tool
}

// Response – ответ модели: текст или вызовы инструментов.
type Response struct {
	Content   string
	ToolCalls []ToolCall
}

// Provider – языковая модель.
type Provider interface {
	Name() string
	Chat(ctx context.Context, req Request) (Response, error)
}

// ErrNotConfigured – выбран openai_compat, но не задан адрес или модель.
var ErrNotConfigured = errors.New("llm: LLM_BASE_URL and LLM_MODEL are required for openai_compat")

// Config – настройки провайдера из окружения.
type Config struct {
	Provider string // mock | openai_compat
	BaseURL  string // например https://api.openrouter.ai/api/v1
	APIKey   string
	Model    string
	Project  string // Yandex AI Studio: ID каталога, уходит заголовком OpenAI-Project
}

// New создает провайдера по настройкам.
func New(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case "", "mock":
		return Mock{}, nil
	case "openai_compat":
		if cfg.BaseURL == "" || cfg.Model == "" {
			return nil, ErrNotConfigured
		}
		o := NewOpenAI(cfg.BaseURL, cfg.APIKey, cfg.Model, nil)
		o.Project = cfg.Project
		return o, nil
	}
	return nil, errors.New("llm: unknown LLM_PROVIDER (use mock or openai_compat)")
}

// LastUser – текст последнего сообщения пользователя.
func LastUser(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

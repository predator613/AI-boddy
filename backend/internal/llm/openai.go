package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAI – клиент OpenAI-совместимого API (POST {base}/chat/completions с function calling).
type OpenAI struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
	Project string // необязательный заголовок OpenAI-Project (ID каталога в Yandex AI Studio)
}

// NewOpenAI создает клиента. client == nil – клиент с таймаутом 30 секунд.
func NewOpenAI(baseURL, apiKey, model string, client *http.Client) *OpenAI {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &OpenAI{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: model, http: client}
}

// Name – имя провайдера для журнала.
func (o *OpenAI) Name() string { return "openai_compat:" + o.model }

type oaTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type oaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

// Chat отправляет диалог модели.
func (o *OpenAI) Chat(ctx context.Context, req Request) (Response, error) {
	body := map[string]any{"model": o.model, "messages": toOA(req.Messages), "temperature": 0.5} // 0.5 – ответы живее, факты держит промпт и проверка в агенте
	if len(req.Tools) > 0 {
		tools := make([]oaTool, len(req.Tools))
		for i, t := range req.Tools {
			tools[i].Type = "function"
			tools[i].Function.Name = t.Name
			tools[i].Function.Description = t.Description
			tools[i].Function.Parameters = t.Parameters
		}
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	}
	if o.Project != "" {
		httpReq.Header.Set("OpenAI-Project", o.Project)
	}
	resp, err := o.http.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Response{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("llm: status %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	var out struct {
		Choices []struct {
			Message oaMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return Response{}, fmt.Errorf("llm: bad response: %w", err)
	}
	if len(out.Choices) == 0 {
		return Response{}, fmt.Errorf("llm: empty choices")
	}
	m := out.Choices[0].Message
	res := Response{Content: strings.TrimSpace(m.Content)}
	for _, c := range m.ToolCalls {
		args := json.RawMessage(c.Function.Arguments)
		if !json.Valid(args) {
			args = json.RawMessage(`{}`)
		}
		res.ToolCalls = append(res.ToolCalls, ToolCall{ID: c.ID, Name: c.Function.Name, Arguments: args})
	}
	return res, nil
}

func toOA(msgs []Message) []oaMessage {
	out := make([]oaMessage, len(msgs))
	for i, m := range msgs {
		out[i] = oaMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for _, c := range m.ToolCalls {
			var tc oaToolCall
			tc.ID, tc.Type = c.ID, "function"
			tc.Function.Name, tc.Function.Arguments = c.Name, string(c.Arguments)
			out[i].ToolCalls = append(out[i].ToolCalls, tc)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

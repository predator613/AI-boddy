package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	p, err := New(Config{})
	require.NoError(t, err)
	assert.Equal(t, "mock", p.Name())
	_, err = New(Config{Provider: "openai_compat"})
	assert.ErrorIs(t, err, ErrNotConfigured)
	p, err = New(Config{Provider: "openai_compat", BaseURL: "http://x/v1/", Model: "a-vibe", Project: "b1g"})
	require.NoError(t, err)
	assert.Equal(t, "openai_compat:a-vibe", p.Name())
	assert.Equal(t, "b1g", p.(*OpenAI).Project)
	_, err = New(Config{Provider: "gpt"})
	assert.Error(t, err)
}

func TestMockIntents(t *testing.T) {
	tools := []Tool{{Name: "prepare_access_request"}, {Name: "prepare_service_request"}}
	ask := func(text string) ToolCall {
		r, err := Mock{}.Chat(context.Background(), Request{Messages: []Message{{Role: "user", Content: text}}, Tools: tools})
		require.NoError(t, err)
		require.Len(t, r.ToolCalls, 1, text)
		return r.ToolCalls[0]
	}
	assert.Equal(t, "prepare_access_request", ask("Запроси доступ к GitLab").Name)
	assert.Equal(t, "prepare_leave_request", ask("хочу отгул в пятницу").Name)
	assert.Equal(t, "list_requests", ask("что с заявкой №1038").Name)
	assert.JSONEq(t, `{"number":"1038"}`, string(ask("что с заявкой №1038").Arguments))
	assert.Equal(t, "handoff_to_mentor", ask("передай наставнику вопрос").Name)
	assert.Equal(t, "search_kb", ask("что такое код-ревью").Name)
	assert.Equal(t, "prepare_service_request", ask("нужен ноутбук").Name)
	assert.Equal(t, "prepare_service_request", ask("хочу оформить отпуск").Name)
	assert.Equal(t, "prepare_service_request", ask("нужен доступ к VPN").Name)
	assert.Equal(t, "search_kb", ask("сколько стоит компьютер").Name, "без намерения – вопрос, а не заявка")

	// «Что делать сейчас?» – без инструмента: ответ по состоянию чата собирает агент.
	r0, _ := Mock{}.Chat(context.Background(), Request{Messages: []Message{{Role: "user", Content: "что делать сейчас?"}}, Tools: tools})
	assert.Empty(t, r0.ToolCalls)
	assert.Empty(t, r0.Content)

	// Без инструмента заявки вопрос идет в поиск.
	r, _ := Mock{}.Chat(context.Background(), Request{Messages: []Message{{Role: "user", Content: "дай доступ"}}})
	assert.Equal(t, "search_kb", r.ToolCalls[0].Name)

	// Ответ по результату поиска – первые два предложения.
	r, _ = Mock{}.Chat(context.Background(), Request{Messages: []Message{
		{Role: "user", Content: "q"},
		{Role: "tool", Content: `{"results":[{"snippet":"Раз. Два! Три?"}]}`},
	}})
	assert.Equal(t, "Раз. Два!", r.Content)
	r, _ = Mock{}.Chat(context.Background(), Request{Messages: []Message{{Role: "tool", Content: `{"results":[]}`}}})
	assert.Empty(t, r.Content)
	assert.Empty(t, LastUser(nil))
}

func TestOpenAI(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer key", r.Header.Get("Authorization"))
		assert.Equal(t, "folder", r.Header.Get("OpenAI-Project"))
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":" ок ","tool_calls":[{"id":"1","type":"function","function":{"name":"search_kb","arguments":"not json"}}]}}]}`))
	}))
	defer srv.Close()
	o := NewOpenAI(srv.URL+"/v1/", "key", "m", nil)
	o.Project = "folder"
	resp, err := o.Chat(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "привет"}, {Role: "assistant", ToolCalls: []ToolCall{{ID: "x", Name: "search_kb", Arguments: json.RawMessage(`{}`)}}}},
		Tools:    []Tool{{Name: "search_kb", Description: "d", Parameters: json.RawMessage(`{"type":"object"}`)}},
	})
	require.NoError(t, err)
	assert.Equal(t, "ок", resp.Content)
	assert.JSONEq(t, `{}`, string(resp.ToolCalls[0].Arguments), "битые аргументы заменяются пустым объектом")
	assert.Equal(t, "m", got["model"])
	assert.Equal(t, "auto", got["tool_choice"])

	for name, h := range map[string]http.HandlerFunc{
		"status": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) },
		"bad":    func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{`)) },
		"empty":  func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"choices":[]}`)) },
	} {
		s := httptest.NewServer(h)
		_, err := NewOpenAI(s.URL, "", "m", nil).Chat(context.Background(), Request{})
		assert.Error(t, err, name)
		s.Close()
	}
	_, err = NewOpenAI("http://127.0.0.1:1", "", "m", nil).Chat(context.Background(), Request{})
	assert.Error(t, err)
	assert.Equal(t, "аб…", truncate("абв", 2))
}

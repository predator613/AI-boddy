package llm

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

// Mock – офлайн-режим без ключа: намерение определяется по ключевым словам, ответ собирается
// из найденного фрагмента справочника. Вызывает те же инструменты, что и настоящая модель,
// поэтому весь путь агента (права, подтверждение, заявка) работает одинаково.
type Mock struct{}

// Name – имя провайдера для журнала.
func (Mock) Name() string { return "mock" }

var (
	reHandoff = regexp.MustCompile(`передай|позови|спроси наставник|напиши наставник|наставнику|человек`)
	reLeave   = regexp.MustCompile(`отгул|отпуск|выходн|учебн\S* день|не выйду`)
	reStatus  = regexp.MustCompile(`статус|мои заявк|моя заявк|моей заявк|что с заявк|заявка №|заявку №`)
	reAccess  = regexp.MustCompile(`доступ|запрос|выда|получить|оформ|подключ|права|нужен|нужна|дай|открой`)
	reNumber  = regexp.MustCompile(`\d{4}`)
)

// Chat имитирует модель с function calling.
func (Mock) Chat(_ context.Context, req Request) (Response, error) {
	if n := len(req.Messages); n > 0 && req.Messages[n-1].Role == "tool" {
		return Response{Content: answerFromTool(req.Messages[n-1].Content)}, nil
	}
	text := strings.ToLower(LastUser(req.Messages))
	call := func(name string, args map[string]string) (Response, error) {
		raw, _ := json.Marshal(args)
		return Response{ToolCalls: []ToolCall{{ID: "mock-" + name, Name: name, Arguments: raw}}}, nil
	}
	switch {
	case reHandoff.MatchString(text):
		return call("handoff_to_mentor", map[string]string{"question": LastUser(req.Messages)})
	case reStatus.MatchString(text):
		return call("list_requests", map[string]string{"number": reNumber.FindString(text)})
	case reLeave.MatchString(text):
		return call("prepare_leave_request", map[string]string{"date": text})
	case reAccess.MatchString(text) && hasTool(req.Tools, "prepare_access_request"):
		return call("prepare_access_request", map[string]string{"resource": text})
	}
	return call("search_kb", map[string]string{"query": LastUser(req.Messages)})
}

// answerFromTool – ответ по результату поиска: первые одно-два предложения лучшего фрагмента.
func answerFromTool(content string) string {
	var res struct {
		Results []struct {
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	if json.Unmarshal([]byte(content), &res) != nil || len(res.Results) == 0 {
		return ""
	}
	return firstSentences(res.Results[0].Snippet, 2)
}

func firstSentences(s string, n int) string {
	s = strings.TrimSpace(s)
	out, count := []rune{}, 0
	for _, r := range s {
		out = append(out, r)
		if r == '.' || r == '!' || r == '?' {
			count++
			if count == n {
				break
			}
		}
	}
	return strings.TrimSpace(string(out))
}

func hasTool(tools []Tool, name string) bool {
	for _, t := range tools {
		if t.Name == name {
			return true
		}
	}
	return false
}

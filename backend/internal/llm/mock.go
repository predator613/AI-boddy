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
	reHandoff = regexp.MustCompile(`передай|позови|спроси наставник|напиши наставник|наставнику|человек|(поговорить|связаться|созвониться|обсудить)\s.{0,20}наставник`)
	reLeave   = regexp.MustCompile(`отгул|выходн|учебн\S* день|не выйду`)
	// Заявки в отделы: предмет (ноутбук, VPN, справка) и намерение его получить.
	reItem   = regexp.MustCompile(`ноут|макбук|компьютер|монитор|гарнитур|наушник|клавиатур|мыш|пропуск|бейдж|vpn|впн|почт|парол|двухфактор|2fa|отпуск|справк|ндфл|практик|характеристик|выплат|стипенди|зарплат|личные данные`)
	reWant   = regexp.MustCompile(`нуж|выда|получ|оформ|закаж|заказ|хочу|дай|сломал|не работа|не выдал|сброс|восстанов|подключ|забыл|потерял|не приш|запрос`)
	reStatus = regexp.MustCompile(`статус|(какие|мои|моя|моей|моих|все|список|покажи)\s.{0,12}заявк|у меня\s.{0,10}заявк|что с заявк|заявк\S* №`)
	reAccess = regexp.MustCompile(`доступ|запрос|выда|получить|оформ|подключ|права|нужен|нужна|дай|открой`)
	reNumber = regexp.MustCompile(`\d{4}`)
	// «что делать дальше?» без предмета: ответ зависит от состояния чата, его собирает агент.
	reNext = regexp.MustCompile(`что (мне )?(теперь |сейчас |дальше )?(делать|дальше|теперь|нажать|сейчас)|а дальше|и дальше|что потом|следующий шаг|куда нажать|как подтвердить`)
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
	case reNext.MatchString(text):
		return Response{}, nil
	case reItem.MatchString(text) && reWant.MatchString(text) && hasTool(req.Tools, "prepare_service_request"):
		return call("prepare_service_request", map[string]string{"item": text})
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

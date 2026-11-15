package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/database"
	"ai-buddy/backend/internal/mockservices"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/testfix"
	"ai-buddy/backend/migrations"
)

// Чат с помощником и заявки: главный сценарий «доступ» от вопроса до выдачи и закрытого пункта пути.

const gitlabTask = "b7000000-0000-4000-8000-000000000204"

type chatEnv struct {
	*env
	store *mockservices.Store
	svc   Services
}

// newChatEnv поднимает backend с отдельным контейнером-имитацией (httptest), чтобы тест
// мог принять решение за согласующего. Все, что тест создал, удаляется после него.
func newChatEnv(t *testing.T, llmURL string) *chatEnv {
	t.Helper()
	cfg := testConfig(t, true)
	store := mockservices.NewStore(0)
	mock := httptest.NewServer(mockservices.Handler(store))
	t.Cleanup(mock.Close)
	cfg.MockServicesURL = mock.URL
	cfg.LLMTimeout = 2 * time.Second
	if llmURL != "" {
		cfg.LLMProvider, cfg.LLMBaseURL, cfg.LLMModel = "openai_compat", llmURL, "test-model"
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL, 10*time.Second)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = database.Migrate(ctx, pool, migrations.FS)
	require.NoError(t, err)
	require.NoError(t, testfix.OtherTeam(ctx, pool))

	// Снимок прогресса Марии и метка времени: после теста все возвращаем как было.
	start := time.Now()
	var snapshot []byte
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COALESCE(json_agg(p), '[]') FROM intern_task_progress p WHERE intern_id = $1`, annaID).Scan(&snapshot))
	t.Cleanup(func() {
		bg := context.Background()
		for _, q := range []string{
			`DELETE FROM messages WHERE created_at >= $1`,
			`DELETE FROM chat_dialogs WHERE started_at >= $1`,
			`UPDATE chat_dialogs SET closed_at = NULL WHERE closed_at >= $1`,
			`DELETE FROM notifications WHERE created_at >= $1`,
			`DELETE FROM agent_tool_calls WHERE created_at >= $1`,
			`DELETE FROM requests WHERE created_at >= $1`,
		} {
			_, err := pool.Exec(bg, q, start)
			assert.NoError(t, err, q)
		}
		_, err := pool.Exec(bg, `DELETE FROM intern_task_progress WHERE intern_id = $1`, annaID)
		assert.NoError(t, err)
		_, err = pool.Exec(bg, `
			INSERT INTO intern_task_progress SELECT * FROM json_populate_recordset(NULL::intern_task_progress, $1)`, snapshot)
		assert.NoError(t, err)
	})

	svc, err := NewServices(cfg, pool, quietLog)
	require.NoError(t, err)
	srv := httptest.NewServer(NewRouter(cfg, svc, quietLog))
	t.Cleanup(srv.Close)
	e := &env{t: t, base: srv.URL, tok: map[string]string{}}
	for _, u := range []string{"intern_maria", "intern_pavel", "mentor_igor"} {
		code, token, _ := login(t, srv.URL, u, "demo123")
		require.Equal(t, http.StatusOK, code)
		e.tok[u] = token
	}
	return &chatEnv{env: e, store: store, svc: svc}
}

func (e *chatEnv) say(user string, body map[string]any) []models.ChatMessage {
	e.t.Helper()
	var r models.ChatReply
	require.Equal(e.t, http.StatusOK, e.call(user, http.MethodPost, "/api/chat/messages", body, &r))
	require.NotEmpty(e.t, r.Messages)
	return r.Messages
}

func confirmOf(t *testing.T, msgs []models.ChatMessage) *models.ConfirmCard {
	t.Helper()
	for _, m := range msgs {
		if m.Confirm != nil {
			return m.Confirm
		}
	}
	t.Fatalf("no confirm card in %+v", msgs)
	return nil
}

func fieldsMap(f []models.Field) map[string]string {
	out := map[string]string{}
	for _, x := range f {
		out[x.Label] = x.Value
	}
	return out
}

func taskStatus(t *testing.T, e *chatEnv, taskID string) models.TaskView {
	t.Helper()
	for _, s := range e.path("intern_maria").Stages {
		for _, task := range s.Tasks {
			if task.ID == taskID {
				return task
			}
		}
	}
	t.Fatalf("task %s not found", taskID)
	return models.TaskView{}
}

func TestAccessScenarioEndToEnd(t *testing.T) {
	e := newChatEnv(t, "")

	// 1. Вопрос текстом: помощник подбирает проект, уровень прав, согласующего и задачу пути.
	msgs := e.say("intern_maria", map[string]any{"text": "Запроси доступ к GitLab"})
	require.Len(t, msgs, 2)
	assert.Contains(t, msgs[0].Text, "владелец проекта")
	require.NotNil(t, msgs[0].Source)
	assert.Equal(t, "Доступы и сервисы → Как запросить доступ к сервису", msgs[0].Source.Title)
	card := confirmOf(t, msgs)
	assert.Equal(t, "pending", card.State)
	assert.Equal(t, map[string]string{
		"Ресурс":       "GitLab",
		"Проект":       "backend / auth-service",
		"Уровень прав": "редактирование (Developer)",
		"Обоснование":  "Нужен для задачи «Получить доступ к GitLab»",
		"Согласует":    "Артемий, владелец проекта",
	}, fieldsMap(card.Fields))

	// Черновик не виден на странице «Заявки» и ничего не меняет в пути.
	var list []models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/requests", nil, &list))
	for _, r := range list {
		assert.NotEqual(t, card.ActionID, r.ID)
	}

	// Чужой черновик подтвердить нельзя.
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil))

	// 2. Подтверждение: заявка уходит во внешнюю систему, задача пути – «на согласовании».
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	require.Len(t, r.Messages, 1)
	res := r.Messages[0].Result
	require.NotNil(t, res)
	assert.Equal(t, "waiting", res.Status)
	assert.Equal(t, "Доступ к GitLab", res.Title)
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil), "повторно не отправляется")

	task := taskStatus(t, e, gitlabTask)
	assert.Equal(t, "waiting", task.Status)
	assert.Contains(t, task.Note, "№"+res.Number)

	// Повторная просьба не создает дубль.
	again := e.say("intern_maria", map[string]any{"text": "дай доступ к гитлабу"})
	assert.Contains(t, again[0].Text, "уже на согласовании")

	// Пока не решено – синхронизация ничего не меняет.
	n, err := e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)
	assert.Zero(t, n)

	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/requests", nil, &list))
	require.Equal(t, res.Number, list[0].Number)
	assert.Equal(t, "waiting", list[0].Status)
	assert.Equal(t, "now", list[0].History[len(list[0].History)-1].State)
	assert.Contains(t, list[0].History[len(list[0].History)-1].Title, "Артемий, владелец проекта")

	// 3. Согласующий одобрил: доступ выдан, пункт пути закрыт, стажер получил сообщение.
	ref := externalRef(t, e, res.RequestID)
	_, err = e.store.Resolve(ref, true, "")
	require.NoError(t, err)
	n, err = e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/requests", nil, &list))
	assert.Equal(t, "done", list[0].Status)
	assert.Equal(t, "Доступ выдан", list[0].History[len(list[0].History)-1].Title)
	assert.Equal(t, "Артемий, владелец проекта", fieldsMap(list[0].Fields)["Согласовал"])
	assert.Equal(t, "done", taskStatus(t, e, gitlabTask).Status)

	var hist models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/chat/messages", nil, &hist))
	last := hist.Messages[len(hist.Messages)-1]
	assert.Equal(t, "result", last.Kind)
	assert.Contains(t, last.Text, "доступ к GitLab выдан")
	require.NotNil(t, last.Result)
	assert.Equal(t, "done", last.Result.Status)
	// Карточка подтверждения в истории помнит, что ее подтвердили.
	for _, m := range hist.Messages {
		if m.Confirm != nil && m.Confirm.ActionID == card.ActionID {
			assert.Equal(t, "confirmed", m.Confirm.State)
		}
	}
	// Доступ уже есть – помощник так и говорит.
	assert.Contains(t, e.say("intern_maria", map[string]any{"text": "нужен доступ к gitlab"})[0].Text, "уже выдан")
}

func TestAccessRejectedAndAuto(t *testing.T) {
	e := newChatEnv(t, "")

	// Кнопка «Сделать с AI-buddy» у задачи: действие известно, проект в старом формате (group).
	msgs := e.say("intern_maria", map[string]any{
		"text":   "Получить доступ к GitLab",
		"action": map[string]any{"kind": "access_request", "payload": map[string]string{"service": "gitlab", "group": "backend", "role": "developer"}},
		"taskId": gitlabTask,
	})
	card := confirmOf(t, msgs)
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	_, err := e.store.Resolve(externalRef(t, e, r.Messages[0].Result.RequestID), false, "Сначала пройди инструктаж по безопасности")
	require.NoError(t, err)
	_, err = e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)

	var list []models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/requests", nil, &list))
	assert.Equal(t, "rejected", list[0].Status)
	require.NotNil(t, list[0].RejectReason)
	assert.Equal(t, "Сначала пройди инструктаж по безопасности", *list[0].RejectReason)
	assert.Equal(t, "rejected", list[0].History[len(list[0].History)-1].State)
	task := taskStatus(t, e, gitlabTask)
	assert.Equal(t, "in_progress", task.Status, "после отказа задача снова в работе")
	assert.Contains(t, task.Note, "инструктаж")

	// Трекер прогресса выдается сразу: задача 2-й недели закрывается без согласования.
	msgs = e.say("intern_maria", map[string]any{"text": "Запроси трекер прогресса и внеси мои цели"})
	card = confirmOf(t, msgs)
	assert.Equal(t, "выдается автоматически", fieldsMap(card.Fields)["Согласование"])
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	assert.Equal(t, "done", r.Messages[0].Result.Status)
	assert.Contains(t, r.Messages[0].Text, "Пункт на «Моем пути» закрыт")
	assert.Equal(t, "done", taskStatus(t, e, "b7000000-0000-4000-8000-000000000205").Status)
}

func TestAssistantAnswersAndLimits(t *testing.T) {
	e := newChatEnv(t, "")

	// Ответ из справочника со ссылкой на статью.
	msgs := e.say("intern_maria", map[string]any{"text": "что такое код-ревью?"})
	assert.Equal(t, "text", msgs[0].Kind)
	require.NotNil(t, msgs[0].Source)
	assert.Contains(t, msgs[0].Source.Title, "код-ревью")
	require.NotNil(t, msgs[0].Source.ArticleID)

	// Ответа нет – помощник не выдумывает, а предлагает передать вопрос.
	msgs = e.say("intern_maria", map[string]any{"text": "сколько стоит квантовый компьютер"})
	assert.Nil(t, msgs[0].Source)
	assert.Contains(t, msgs[0].Text, "Передать Игорю вопрос в таком виде")
	// «нет» на предложение – вопрос не уходит.
	msgs = e.say("intern_maria", map[string]any{"text": "нет"})
	assert.Contains(t, msgs[0].Text, "Хорошо, не передаю")

	// Доступ, запрещенный роли, не оформляется.
	msgs = e.say("intern_maria", map[string]any{"text": "дай доступ к логам продакшена"})
	assert.Equal(t, "error", msgs[0].Kind)
	assert.Equal(t, "sad", msgs[0].Mood)
	assert.Contains(t, msgs[0].Text, "Стажерам этот доступ не выдается")

	// Администрирование запрещено, но помощник предлагает разрешенный уровень.
	msgs = e.say("intern_maria", map[string]any{
		"text":   "Maintainer в GitLab",
		"action": map[string]any{"kind": "access_request", "payload": map[string]string{"service": "gitlab", "role": "maintainer"}},
	})
	assert.Equal(t, "error", msgs[0].Kind)
	assert.Contains(t, msgs[0].Text, "редактирование (Developer)")

	// Заявка на стенд уже есть (демо-данные) – дубль не создается.
	msgs = e.say("intern_maria", map[string]any{"text": "получить доступ к тестовому стенду"})
	assert.Contains(t, msgs[0].Text, "№1038 уже на согласовании: Jira Service Desk, команда доступов")

	// Непонятный сервис.
	msgs = e.say("intern_maria", map[string]any{"text": "дай доступ к холодильнику"})
	assert.Contains(t, msgs[0].Text, "Не понял, к какому сервису")

	// Статус заявок.
	msgs = e.say("intern_maria", map[string]any{"text": "какой статус моих заявок?"})
	assert.Contains(t, msgs[0].Text, "№1038 Доступ к тестовому стенду – на согласовании (Jira Service Desk, команда доступов)")
	msgs = e.say("intern_maria", map[string]any{"text": "что с заявкой №1027"})
	assert.Contains(t, msgs[0].Text, "отклонена")
	assert.NotContains(t, msgs[0].Text, "1038")

	// Передать вопрос наставнику.
	msgs = e.say("intern_maria", map[string]any{"text": "передай наставнику: можно брать задачи из чужого спринта?"})
	assert.Contains(t, msgs[0].Text, "Передать Игорю вопрос в таком виде?\n«можно брать задачи из чужого спринта?»", "сначала показываем, что уйдет")
	msgs = e.say("intern_maria", map[string]any{"text": "да"})
	assert.Contains(t, msgs[0].Text, "Передал Игорю вопрос: «можно брать задачи из чужого спринта?»")
	pool := testPool(t)
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT count(*) FROM handoffs h JOIN messages m ON m.id = h.forwarded_id
		WHERE m.text = 'можно брать задачи из чужого спринта?'`).Scan(&n))
	assert.Equal(t, 1, n, "наставнику уходит сам вопрос, без «передай наставнику:»")

	// Ответ по статье и действие «открыть статью».
	msgs = e.say("intern_maria", map[string]any{
		"text":   "Открой памятку",
		"action": map[string]any{"kind": "open_article", "payload": map[string]string{"articleId": "c2000000-0000-4000-8000-000000000001"}},
	})
	require.NotNil(t, msgs[0].Source)

	// Вопрос-действие ask_buddy идет через модель.
	msgs = e.say("intern_maria", map[string]any{
		"text":   "Как устроено ревью?",
		"action": map[string]any{"kind": "ask_buddy", "payload": map[string]string{"question": "как устроено код-ревью"}},
	})
	require.NotNil(t, msgs[0].Source)

	// Права и ввод.
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodPost, "/api/chat/messages", map[string]any{"text": "привет"}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodGet, "/api/chat/messages", nil, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("intern_maria", http.MethodPost, "/api/chat/messages", map[string]any{"text": "  "}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("intern_maria", http.MethodPost, "/api/chat/messages", map[string]any{"text": strings.Repeat("я", 2001)}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodPost, "/api/chat/actions/nope/confirm", nil, nil))
	assert.Equal(t, http.StatusUnauthorized, e.call("nobody", http.MethodGet, "/api/requests", nil, nil))
}

func TestLeaveRequestAndCancel(t *testing.T) {
	e := newChatEnv(t, "")

	// Даты считаются от сегодняшнего дня: рабочий день через две недели и ближайшая суббота после него.
	workday := time.Now().AddDate(0, 0, 14)
	for workday.Weekday() == time.Saturday || workday.Weekday() == time.Sunday {
		workday = workday.AddDate(0, 0, 1)
	}
	saturday := workday
	for saturday.Weekday() != time.Saturday {
		saturday = saturday.AddDate(0, 0, 1)
	}
	ddmm := func(d time.Time) string { return d.Format("02.01") }

	msgs := e.say("intern_maria", map[string]any{"text": "хочу отгул " + ddmm(workday) + " за свой счет"})
	card := confirmOf(t, msgs)
	f := fieldsMap(card.Fields)
	assert.True(t, strings.HasPrefix(f["Дата"], fmt.Sprintf("%d ", workday.Day())), f["Дата"])
	assert.Equal(t, "отгул за свой счет", f["Тип"])
	assert.Equal(t, "Артемий, руководитель", f["Согласует"])
	require.NotNil(t, msgs[0].Source)
	assert.Contains(t, msgs[0].Source.Title, "отгул")

	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	assert.Equal(t, "waiting", r.Messages[0].Result.Status)
	_, err := e.store.Resolve(externalRef(t, e, r.Messages[0].Result.RequestID), true, "")
	require.NoError(t, err)
	_, err = e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)
	var list []models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/requests", nil, &list))
	assert.Equal(t, "approved", list[0].Status)
	assert.Equal(t, "leave", list[0].Kind)
	assert.True(t, strings.HasPrefix(list[0].Title, fmt.Sprintf("Отгул за свой счет %d ", workday.Day())), list[0].Title)

	// Отмена черновика.
	card = confirmOf(t, e.say("intern_maria", map[string]any{"text": "отгул " + ddmm(workday.AddDate(0, 0, 7))}))
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/cancel", nil, &r))
	assert.Contains(t, r.Messages[0].Text, "заявку не создаю")
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil))

	// Выходной день.
	msgs = e.say("intern_maria", map[string]any{"text": "отгул " + ddmm(saturday)})
	assert.Contains(t, msgs[0].Text, "выходной")
}

func TestRequestServiceUnavailable(t *testing.T) {
	e := newChatEnv(t, "")
	card := confirmOf(t, e.say("intern_maria", map[string]any{"text": "запроси доступ к gitlab"}))
	// Имитация недоступна: стажер видит ошибку, черновик остается, можно повторить.
	srvDown := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer srvDown.Close()
	cfg := testConfig(t, true)
	cfg.MockServicesURL = srvDown.URL
	svc, err := NewServices(cfg, testPool(t), quietLog)
	require.NoError(t, err)
	down := httptest.NewServer(NewRouter(cfg, svc, quietLog))
	defer down.Close()
	resp := do(t, http.MethodPost, down.URL+"/api/chat/actions/"+card.ActionID+"/confirm", e.tok["intern_maria"], nil)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil), "после восстановления заявка отправляется")
}

// Модель по OpenAI-совместимому API (так подключается A-Vibe): выбирает инструмент, потом пишет ответ.
func TestAssistantWithOpenAICompatibleModel(t *testing.T) {
	calls := 0
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Messages []struct {
				Role string `json:"role"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		if req.Messages[len(req.Messages)-1].Role == "tool" {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Ревью делает коллега из команды, после одобрения код вливается."}}]}`))
			return
		}
		if strings.Contains(string(body), "сломай") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"search_kb","arguments":"{\"query\":\"код-ревью\"}"}}]}}]}`))
	}))
	defer fake.Close()
	e := newChatEnv(t, fake.URL)

	msgs := e.say("intern_maria", map[string]any{"text": "кто смотрит мой код?"})
	assert.Equal(t, "Ревью делает коллега из команды, после одобрения код вливается.", msgs[0].Text)
	require.NotNil(t, msgs[0].Source)
	assert.Equal(t, 2, calls)

	// Модель недоступна – ответ собирается офлайн, чат не падает.
	msgs = e.say("intern_maria", map[string]any{"text": "сломай и расскажи про код-ревью"})
	require.NotNil(t, msgs[0].Source)
}

// Ответы наставника видят только его стажеры и руководитель команды.
func TestMentorAnswerVisibility(t *testing.T) {
	e := newEnv(t)
	pool := testPool(t)
	const article = "c2000000-0000-4000-8000-000000000016"
	const otherMentor = "aaaaaaaa-0000-4000-8000-000000000099"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE kb_articles SET author_id = 'aaaaaaaa-0000-4000-8000-000000000002' WHERE id = $1`, article)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, otherMentor)
	})
	assert.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/kb/articles/"+article, nil, nil))
	assert.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/kb/articles/"+article, nil, nil))

	// Ответ другого наставника той же команды Мария не видит, руководитель – видит.
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, login, password_hash, name, role, team_id)
		VALUES ($1, 'mentor_test_other', 'x', 'Петр', 'mentor', '11111111-1111-4111-8111-111111111111')`, otherMentor)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE kb_articles SET author_id = $2 WHERE id = $1`, article, otherMentor)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodGet, "/api/kb/articles/"+article, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodGet, "/api/kb/articles/"+article, nil, nil))
	assert.Equal(t, http.StatusOK, e.call("lead_artemiy", http.MethodGet, "/api/kb/articles/"+article, nil, nil))
}

func externalRef(t *testing.T, e *chatEnv, requestID string) string {
	t.Helper()
	var ref string
	require.NoError(t, testPool(t).QueryRow(context.Background(),
		`SELECT external_ref FROM requests WHERE id = $1`, requestID).Scan(&ref))
	_ = e
	return ref
}

// «Что делать сейчас?» после карточки: помощник помнит переписку и подсказывает по ее состоянию.
func TestNextStepUsesChatState(t *testing.T) {
	e := newChatEnv(t, "")

	msgs := e.say("intern_maria", map[string]any{"text": "как получить доступ к GitLab"})
	card := confirmOf(t, msgs)

	next := e.say("intern_maria", map[string]any{"text": "что делать сейчас?"})
	require.Len(t, next, 1)
	assert.Contains(t, next[0].Text, "Подтвердить")
	assert.Contains(t, next[0].Text, "Артемий")
	assert.Nil(t, next[0].Confirm)

	// Повторная просьба, пока карточка не подтверждена, вторую карточку не создает.
	again := e.say("intern_maria", map[string]any{"text": "дай доступ к гитлабу"})
	require.Len(t, again, 1)
	assert.Nil(t, again[0].Confirm)
	assert.Contains(t, again[0].Text, "уже выше")

	// После подтверждения подсказка – про ожидание решения.
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	after := e.say("intern_maria", map[string]any{"text": "а дальше?"})
	assert.Contains(t, after[0].Text, "ждет решения")

	// Незнакомый вопрос по-прежнему честно отвечает «не знаю».
	unknown := e.say("intern_maria", map[string]any{"text": "сколько стоит компьютер"})
	assert.Contains(t, unknown[0].Text, "Точного ответа")

	// Стажер без заявок получает подсказку про «Мой путь» или про свою заявку.
	pavel := e.say("intern_pavel", map[string]any{"text": "что мне делать дальше"})
	assert.Regexp(t, "Моем пути|ждет решения", pavel[0].Text)
}

// «Кто мой наставник?», правка карточки текстом, «какие у меня заявки», «поговорить с наставником»,
// «следующий шаг» с задачей «Моего пути».
func TestAssistantIntents(t *testing.T) {
	e := newChatEnv(t, "")

	who := e.say("intern_maria", map[string]any{"text": "Кто мой наставник?"})
	require.Len(t, who, 1)
	assert.Contains(t, who[0].Text, "Игорь")
	assert.Contains(t, who[0].Text, "Артемий")
	assert.Contains(t, who[0].Text, "Чат с наставником")
	pavelWho := e.say("intern_pavel", map[string]any{"text": "кто у меня руководитель"})
	assert.Contains(t, pavelWho[0].Text, "не назначили")
	assert.Contains(t, pavelWho[0].Text, "Нина")

	next := e.say("intern_maria", map[string]any{"text": "Какой мой следующий шаг?"})
	assert.Contains(t, next[0].Text, "Моем пути")
	assert.Contains(t, next[0].Text, "GitLab")

	// Карточка GitLab, затем правка текстом: старая отменяется, новая – с другим уровнем.
	first := confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужен доступ в GitLab"}))
	assert.Contains(t, fieldsMap(first.Fields)["Уровень прав"], "Developer")
	edited := e.say("intern_maria", map[string]any{"text": "сделай только чтение"})
	second := confirmOf(t, edited)
	assert.NotEqual(t, first.ActionID, second.ActionID)
	assert.Contains(t, fieldsMap(second.Fields)["Уровень прав"], "Reporter")
	assert.Contains(t, edited[0].Text, "Обновил карточку")
	var r models.ChatReply
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+first.ActionID+"/confirm", nil, &r),
		"старая карточка больше не подтверждается")
	same := e.say("intern_maria", map[string]any{"text": "поменяй на чтение"})
	assert.Contains(t, same[len(same)-1].Text, "уже выше", "тот же уровень – карточка не меняется")

	// Отгул: карточка GitLab убирается с пояснением, новая дата заменяет карточку отгула.
	leave := e.say("intern_maria", map[string]any{"text": "хочу отгул 20 ноября"})
	assert.Contains(t, leave[0].Text, "Карточку «Доступ к GitLab» выше убрал")
	assert.NotContains(t, leave[0].Text, "Обновил")
	l1 := confirmOf(t, leave)
	moved := e.say("intern_maria", map[string]any{"text": "перенеси на 25 ноября"})
	l2 := confirmOf(t, moved)
	assert.NotEqual(t, l1.ActionID, l2.ActionID)
	assert.Contains(t, fieldsMap(l2.Fields)["Дата"], "25 ноября")

	// Заявка в отдел: комментарий дописывается в новую карточку.
	confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужен ноутбук"}))
	withComment := confirmOf(t, e.say("intern_maria", map[string]any{"text": "добавь комментарий: нужен macOS"}))
	assert.Contains(t, fieldsMap(withComment.Fields)["Комментарий"], "macOS")

	list := e.say("intern_maria", map[string]any{"text": "Какие у меня заявки?"})
	assert.Contains(t, list[0].Text, "№")

	handoff := e.say("intern_maria", map[string]any{"text": "Хочу поговорить с наставником"})
	assert.Contains(t, handoff[0].Text, "Игорю")
	assert.Equal(t, "mentor", handoff[0].Mood)

	// Оборудование без предмета: помощник спрашивает, что именно, и перечисляет каталог отдела.
	help := e.say("intern_maria", map[string]any{"text": "нужна заявка на оборудование"})
	assert.Contains(t, help[0].Text, "ноутбук")
	assert.Contains(t, help[0].Text, "другое оборудование")
	dock := confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужна док-станция"}))
	assert.Equal(t, "Другое оборудование", fieldsMap(dock.Fields)["Что нужно"])
}

// С настоящей моделью: в запрос уходят история и неподтвержденная карточка, пояснение к карточке
// формулирует модель, но без фактов backend ее текст отбрасывается.
func TestModelGetsHistoryAndDraft(t *testing.T) {
	var bodies []string
	paraphrase := "Доступ к GitLab согласует Артемий, обычно до 24 ч."
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		answer := func(content string) {
			out, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}}})
			_, _ = w.Write(out)
		}
		switch {
		case strings.Contains(body, "Перескажи") && strings.Contains(body, "выдумай"):
			answer("Доступ выдаст Петр за час.")
		case strings.Contains(body, "Перескажи"):
			answer(paraphrase)
		case strings.Contains(body, "что делать") && !strings.Contains(body, "выдумай"):
			answer("Нажми «Подтвердить» в карточке выше.")
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"prepare_access_request","arguments":"{\"resource\":\"GitLab\"}"}}]}}]}`))
		}
	}))
	defer fake.Close()
	e := newChatEnv(t, fake.URL)

	msgs := e.say("intern_maria", map[string]any{"text": "как получить доступ к GitLab"})
	assert.Equal(t, paraphrase, msgs[0].Text)
	confirmOf(t, msgs)

	next := e.say("intern_maria", map[string]any{"text": "что делать сейчас?"})
	assert.Equal(t, "Нажми «Подтвердить» в карточке выше.", next[0].Text)
	last := bodies[len(bodies)-1]
	assert.Contains(t, last, "как получить доступ к GitLab", "история переписки")
	assert.Contains(t, last, "неподтвержденная карточка", "состояние чата")

	// Пересказ с чужими фактами не проходит: остается текст backend.
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+confirmOf(t, msgs).ActionID+"/cancel", nil, nil))
	msgs = e.say("intern_maria", map[string]any{"text": "выдумай и оформи доступ к GitLab"})
	assert.Contains(t, msgs[0].Text, "Артемий")
	assert.NotContains(t, msgs[0].Text, "Петр")
}

// Заявка в отдел: ноутбук уходит в отдел оборудования, наставник получает уведомление,
// решение отдела приходит автоматически из mock-services.
func TestServiceRequestToDepartment(t *testing.T) {
	e := newChatEnv(t, "")

	msgs := e.say("intern_maria", map[string]any{"text": "мне не выдали ноутбук, нужен"})
	require.Len(t, msgs, 2)
	assert.Contains(t, msgs[0].Text, "Отдел оборудования")
	assert.Contains(t, msgs[0].Text, "Игорь")
	card := confirmOf(t, msgs)
	assert.Equal(t, "Отправить заявку: Отдел оборудования?", card.Title)
	assert.Equal(t, map[string]string{"Что нужно": "Ноутбук", "Исполняет": "Отдел оборудования"}, fieldsMap(card.Fields))

	// Вторая просьба, пока карточка не подтверждена, дубль не создает.
	again := e.say("intern_maria", map[string]any{"text": "нужен ноутбук"})
	assert.Contains(t, again[0].Text, "уже выше")

	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	res := r.Messages[0].Result
	require.NotNil(t, res)
	assert.Equal(t, "waiting", res.Status)
	assert.Equal(t, "Выдача ноутбука", res.Title)
	assert.Contains(t, r.Messages[0].Text, "Наставник (Игорь) получил уведомление")

	var note string
	require.NoError(t, testPool(t).QueryRow(context.Background(), `
		SELECT text FROM notifications WHERE user_id = 'aaaaaaaa-0000-4000-8000-000000000002' AND request_id = $1`, res.RequestID).Scan(&note))
	assert.Contains(t, note, "Мария: заявка №"+res.Number)
	ref := externalRef(t, e, res.RequestID)
	assert.True(t, strings.HasPrefix(ref, "SD-"), ref)

	// Пока заявка в работе, повторная просьба сообщает ее номер.
	assert.Contains(t, e.say("intern_maria", map[string]any{"text": "нужен ноутбук"})[0].Text, "№"+res.Number)

	// Отдел выполнил заявку.
	_, err := e.store.Resolve(ref, true, "")
	require.NoError(t, err)
	_, err = e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)
	var list []models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/requests", nil, &list))
	assert.Equal(t, "done", list[0].Status)
	assert.Equal(t, "other", list[0].Kind)
	assert.Equal(t, "Выполнена", list[0].History[len(list[0].History)-1].Title)
	assert.Equal(t, "Отдел оборудования", fieldsMap(list[0].Fields)["Выполнил"])

	// Отпуск – в HR, справка 2-НДФЛ – в бухгалтерию.
	assert.Equal(t, "Отправить заявку: HR-поддержка?", confirmOf(t, e.say("intern_maria", map[string]any{"text": "хочу оформить отпуск с 20 декабря"})).Title)
	assert.Equal(t, "Отправить заявку: Бухгалтерия?", confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужна справка 2-НДФЛ за 2026"})).Title)

	// Непонятный предмет – подсказка, что можно заказать.
	help := e.say("intern_maria", map[string]any{"text": "нужна справка о погоде"})
	assert.Contains(t, help[0].Text, "Отдел оборудования:")
	assert.Nil(t, help[0].Confirm)

	// Вопрос без намерения не превращается в заявку.
	assert.Contains(t, e.say("intern_maria", map[string]any{"text": "сколько стоит компьютер"})[0].Text, "Точного ответа")
}

// С моделью: в ответ на поиск уходят полные тексты статей, markdown из ответа убирается,
// а когда справочник молчит, модель может ответить сама (без ссылки на источник).
func TestModelGetsArticleTextAndCleansMarkdown(t *testing.T) {
	var toolBodies []string
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		w.Header().Set("Content-Type", "application/json")
		answer := func(content string) {
			out, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}}})
			_, _ = w.Write(out)
		}
		switch {
		case strings.Contains(body, `"role":"tool"`):
			toolBodies = append(toolBodies, body)
			if strings.Contains(body, "В справочнике ничего не нашлось.") {
				answer("В справочнике этого нет, но обычно помогает перезагрузка. Передать вопрос наставнику?")
				return
			}
			answer("**Ревью** делает наставник.\n* сначала тесты\n* потом MR")
		default:
			q := "код-ревью"
			if strings.Contains(body, "хромодинамик") {
				q = "квантовая хромодинамика"
			}
			out, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{
				"role": "assistant", "content": "",
				"tool_calls": []any{map[string]any{"id": "c1", "type": "function", "function": map[string]any{"name": "search_kb", "arguments": `{"query":"` + q + `"}`}}},
			}}}})
			_, _ = w.Write(out)
		}
	}))
	defer fake.Close()
	e := newChatEnv(t, fake.URL)

	msgs := e.say("intern_maria", map[string]any{"text": "как устроено код-ревью?"})
	assert.Equal(t, "Ревью делает наставник.\n– сначала тесты\n– потом MR", msgs[0].Text)
	require.NotNil(t, msgs[0].Source)
	require.NotEmpty(t, toolBodies)
	assert.Contains(t, toolBodies[0], "Как подготовить MR", "модель получила текст статьи")
	assert.Contains(t, toolBodies[0], "Игорь", "в тексте статьи – наставник стажера")
	assert.Contains(t, toolBodies[0], "Наставник стажера: Игорь", "в промпте – люди стажера")

	msgs = e.say("intern_maria", map[string]any{"text": "что такое квантовая хромодинамика"})
	assert.Contains(t, msgs[0].Text, "В справочнике этого нет")
	assert.Nil(t, msgs[0].Source)
}

// Переписка сохраняется: после перезагрузки страницы чат восстанавливается, «Новый диалог»
// прячет старые сообщения (в базе они остаются) и отменяет неподтвержденные карточки.
func TestChatHistoryAndNewDialog(t *testing.T) {
	e := newChatEnv(t, "")
	history := func() []models.ChatMessage {
		t.Helper()
		var r models.ChatReply
		require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/chat/messages", nil, &r))
		return r.Messages
	}

	msgs := e.say("intern_maria", map[string]any{"text": "как получить доступ к GitLab"})
	card := confirmOf(t, msgs)
	h := history()
	require.GreaterOrEqual(t, len(h), 3)
	last := h[len(h)-3:]
	assert.Equal(t, "user", last[0].Role)
	assert.Equal(t, "как получить доступ к GitLab", last[0].Text)
	require.NotNil(t, last[2].Confirm, "карточка восстанавливается вместе с состоянием")
	assert.Equal(t, "pending", last[2].Confirm.State)
	require.NotNil(t, last[1].Source)

	// Новый диалог: история пустая, карточка старого диалога отменена.
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/reset", nil, &r))
	assert.Empty(t, r.Messages)
	assert.Empty(t, history())
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil))

	// В новом диалоге помощник не видит старую карточку и готовит новую.
	assert.NotNil(t, confirmOf(t, e.say("intern_maria", map[string]any{"text": "как получить доступ к GitLab"})))
	assert.Len(t, history(), 3)

	// Наставнику чат с помощником недоступен.
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodPost, "/api/chat/reset", nil, nil))
}

// История диалогов: «Новый диалог» закрывает текущий, прошлый можно открыть для чтения.
func TestChatDialogsHistory(t *testing.T) {
	e := newChatEnv(t, "")
	dialogs := func(user string) []models.ChatDialog {
		t.Helper()
		var out []models.ChatDialog
		require.Equal(t, http.StatusOK, e.call(user, http.MethodGet, "/api/chat/dialogs", nil, &out))
		return out
	}
	reset := func() {
		t.Helper()
		require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/reset", nil, nil))
	}
	reset()
	before := len(dialogs("intern_maria"))
	reset() // пустой диалог не закрывается
	require.Len(t, dialogs("intern_maria"), before)
	assert.True(t, dialogs("intern_maria")[0].Current)

	// Диалог с ответом из справочника и неподтвержденной карточкой.
	e.say("intern_maria", map[string]any{"text": "как устроено код-ревью в команде"})
	card := confirmOf(t, e.say("intern_maria", map[string]any{"text": "как получить доступ к GitLab"}))
	cur := dialogs("intern_maria")[0]
	assert.True(t, cur.Current)
	assert.Equal(t, "как устроено код-ревью в команде", cur.Title)
	reset()

	list := dialogs("intern_maria")
	require.Len(t, list, before+1)
	assert.True(t, list[0].Current)
	assert.Equal(t, "Новый диалог", list[0].Title)
	past := list[1]
	assert.False(t, past.Current)
	assert.Equal(t, "как устроено код-ревью в команде", past.Title)
	assert.Equal(t, "answer", past.Kind)
	assert.Equal(t, "Ответ из справочника", past.Preview)

	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/chat/dialogs/"+past.ID+"/messages", nil, &r))
	require.GreaterOrEqual(t, len(r.Messages), 4)
	assert.Equal(t, "как устроено код-ревью в команде", r.Messages[0].Text)
	var cardMsg *models.ChatMessage
	for i := range r.Messages {
		if r.Messages[i].Confirm != nil && r.Messages[i].Confirm.ActionID == card.ActionID {
			cardMsg = &r.Messages[i]
		}
	}
	require.NotNil(t, cardMsg)
	assert.Equal(t, "cancelled", cardMsg.Confirm.State, "в прошлом диалоге карточку не нажать")

	// Диалог с заявкой: в итоге – номер заявки.
	card = confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужен монитор"}))
	var cr models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &cr))
	reset()
	withRequest := dialogs("intern_maria")[1]
	assert.Equal(t, "request", withRequest.Kind)
	assert.Equal(t, "Заявка №"+cr.Messages[0].Result.Number+" создана", withRequest.Preview)
	assert.Equal(t, "нужен монитор", withRequest.Title)

	// Диалог с передачей наставнику.
	e.say("intern_maria", map[string]any{"text": "передай наставнику: как попасть на демо?"})
	e.say("intern_maria", map[string]any{"text": "да"})
	reset()
	assert.Equal(t, "handoff", dialogs("intern_maria")[1].Kind)
	assert.Equal(t, "Передано наставнику", dialogs("intern_maria")[1].Preview)

	// Текущий диалог по-прежнему отдает GET /api/chat/messages, чужой диалог – 404.
	var cur2 models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/chat/messages", nil, &cur2))
	assert.Empty(t, cur2.Messages)
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodGet, "/api/chat/dialogs/"+past.ID+"/messages", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodGet, "/api/chat/dialogs/missing/messages", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodGet, "/api/chat/dialogs", nil, nil))
}

// Модель пишет «передал» или «создам заявку», но инструмент не вызывает. Помощник не должен
// обманывать: вопрос уходит наставнику на самом деле, а заявка готовится повторным вызовом модели.
func TestModelClaimsActionWithoutTool(t *testing.T) {
	var mu sync.Mutex
	var bodies []string
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		text := func(s string) {
			out, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": s}}}})
			_, _ = w.Write(out)
		}
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(raw, &req)
		last := req.Messages[len(req.Messages)-1]
		lastUser := ""
		for _, m := range req.Messages {
			if m.Role == "user" {
				lastUser = m.Content
			}
		}
		switch {
		case strings.Contains(body, "Перескажи"):
			text("ok")
		case last.Role == "tool":
			text("Стажеру хватает роли Developer в группе backend.")
		case strings.Contains(lastUser, "роль нужна"):
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c3","type":"function","function":{"name":"handoff_to_mentor","arguments":"{}"}}]}}]}`))
		case strings.Contains(body, "Сформулируй вопрос стажера"):
			text("В справочнике этого нет, могу передать наставнику.") // плохой ответ модели – отбрасывается
		case last.Role == "system" && strings.Contains(lastUser, "GitLab"):
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c2","type":"function","function":{"name":"prepare_access_request","arguments":"{\\"resource\\":\\"GitLab\\",\\"access_level\\":\\"read\\"}"}}]}}]}`))
		case last.Role == "system":
			text("Я передал наставнику твой вопрос.")
		case strings.Contains(lastUser, "GitLab"):
			text("Сейчас создам заявку на доступ к GitLab, после этого подтвердишь ее.")
		case strings.Contains(lastUser, "работать месяц из другого"):
			text("В справочнике этого нет. Могу передать вопрос наставнику?")
		default:
			text("Я передал наставнику Игорю твой вопрос, он ответит здесь же.")
		}
	}))
	defer fake.Close()
	e := newChatEnv(t, fake.URL)
	pool := testPool(t)
	forwarded := func() []string {
		rows, err := pool.Query(context.Background(), `
			SELECT m.text FROM handoffs h JOIN messages m ON m.id = h.forwarded_id
			WHERE m.created_at > now() - interval '1 minute' ORDER BY m.created_at`)
		require.NoError(t, err)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			require.NoError(t, rows.Scan(&s))
			out = append(out, s)
		}
		return out
	}

	lastOf := func(l []string) string {
		require.NotEmpty(t, l)
		return l[len(l)-1]
	}

	// 1. Помощник предлагает передать, стажер отвечает «давай» – сначала видит формулировку, потом «да».
	e.say("intern_maria", map[string]any{"text": "можно ли работать месяц из другого города?"})
	msgs := e.say("intern_maria", map[string]any{"text": "давай"})
	assert.Contains(t, msgs[0].Text, "в таком виде?\n«можно ли работать месяц из другого города?»")
	assert.Empty(t, forwarded(), "без согласия ничего не уходит")
	msgs = e.say("intern_maria", map[string]any{"text": "да"})
	assert.Contains(t, msgs[0].Text, "Передал Игорю вопрос")
	assert.Equal(t, []string{"можно ли работать месяц из другого города?"}, forwarded())

	// 1б. «ну что?» вопросом не считается; на «нет» ничего не уходит.
	e.say("intern_maria", map[string]any{"text": "можно ли работать месяц из другого города летом?"})
	msgs = e.say("intern_maria", map[string]any{"text": "ну что?"})
	assert.Contains(t, msgs[0].Text, "«можно ли работать месяц из другого города летом?»")
	msgs = e.say("intern_maria", map[string]any{"text": "нет"})
	assert.Contains(t, msgs[0].Text, "не передаю")
	assert.Len(t, forwarded(), 1)

	// 2. Точная формулировка в кавычках – согласие сразу.
	e.say("intern_maria", map[string]any{"text": `передай ему вопрос "почему у меня заявка об отгуле на согласовании"`})
	assert.Equal(t, "почему у меня заявка об отгуле на согласовании", lastOf(forwarded()))

	// 3. Модель пишет «передал» без инструмента – вместо этого предложение; стажер правит формулировку.
	msgs = e.say("intern_maria", map[string]any{"text": "как мне сдать отчет о практике в вуз"})
	assert.Contains(t, msgs[0].Text, "в таком виде")
	msgs = e.say("intern_maria", map[string]any{"text": "лучше так: Как оформить отчет о практике для вуза?"})
	assert.Contains(t, msgs[0].Text, "«Как оформить отчет о практике для вуза?»")
	e.say("intern_maria", map[string]any{"text": "да"})
	assert.Equal(t, "Как оформить отчет о практике для вуза?", lastOf(forwarded()))

	// 3б. Модель сразу хочет передать вопрос, на который уже есть ответ наставника в справочнике:
	// помощник отвечает из справочника, наставника не тревожит.
	msgs = e.say("intern_maria", map[string]any{"text": "какая роль нужна в GitLab стажеру?"})
	assert.Contains(t, msgs[0].Text, "уже отвечал")
	assert.Contains(t, msgs[0].Text, "роли Developer")
	require.NotNil(t, msgs[0].Source)
	assert.NotContains(t, msgs[0].Text, "в таком виде")

	// 4. «Сейчас создам заявку» без инструмента: повторный вызов модели готовит карточку.
	msgs = e.say("intern_maria", map[string]any{"text": "нужен доступ к GitLab для первого проекта"})
	card := confirmOf(t, msgs)
	assert.Equal(t, "GitLab", fieldsMap(card.Fields)["Ресурс"])

	// 5. Стиль ответа из настроек доходит до модели.
	e.say("intern_maria", map[string]any{"text": "что такое спринт", "style": "short"})
	mu.Lock()
	all := strings.Join(bodies, "\n")
	mu.Unlock()
	assert.Contains(t, all, "отвечать коротко")
}

// Стажер может отозвать заявку, пока она на согласовании: через чат (с «да») и на странице «Заявки».
func TestWithdrawRequest(t *testing.T) {
	e := newChatEnv(t, "")

	// Вопрос о конкретной заявке – статус из базы, без попыток «подготовить карточку».
	card := confirmOf(t, e.say("intern_maria", map[string]any{"text": "Запроси доступ к GitLab"}))
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	num := r.Messages[0].Result.Number
	msgs := e.say("intern_maria", map[string]any{"text": "Расскажи про заявку №" + num})
	assert.Contains(t, msgs[0].Text, "№"+num)
	assert.Contains(t, msgs[0].Text, "на согласовании")

	// «а можно ее отозвать?» – помощник спрашивает согласие, «нет» ничего не меняет.
	msgs = e.say("intern_maria", map[string]any{"text": "а можно ее отозвать?"})
	assert.Contains(t, msgs[0].Text, "Отозвать заявку №"+num)
	msgs = e.say("intern_maria", map[string]any{"text": "нет"})
	assert.Contains(t, msgs[0].Text, "остается на согласовании")
	assert.Equal(t, "waiting", taskStatus(t, e, gitlabTask).Status)

	// «отзови заявку» + «да» – заявка отозвана, в сервисе закрыта, задача снова в работе.
	e.say("intern_maria", map[string]any{"text": "отзови заявку №" + num})
	msgs = e.say("intern_maria", map[string]any{"text": "да"})
	assert.Contains(t, msgs[0].Text, "отозвана")
	var list []models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, "/api/requests", nil, &list))
	var got *models.RequestItem
	for i := range list {
		if list[i].Number == num {
			got = &list[i]
		}
	}
	require.NotNil(t, got, "отозванная заявка видна в списке")
	assert.Equal(t, "cancelled", got.Status)
	assert.Equal(t, "Отозвана стажером", got.History[len(got.History)-1].Title)
	assert.Equal(t, "in_progress", taskStatus(t, e, gitlabTask).Status)
	ticket, err := e.store.Ticket(context.Background(), externalRef(t, e, got.ID))
	require.NoError(t, err)
	assert.Equal(t, "rejected", ticket.Status, "в сервисе заявок тоже закрыта")

	// Повторно отозвать нельзя (409), чужую – 404.
	assert.Equal(t, http.StatusConflict, e.call("intern_maria", http.MethodPost, "/api/requests/"+got.ID+"/withdraw", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodPost, "/api/requests/"+got.ID+"/withdraw", nil, nil))

	// Через страницу «Заявки»: новая заявка, отзыв кнопкой.
	card = confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужен монитор"}))
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	var item models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/requests/"+r.Messages[0].Result.RequestID+"/withdraw", nil, &item))
	assert.Equal(t, "cancelled", item.Status)
}

// Отгул на прошедшую или слишком далекую дату не оформляется; отзыв по номеру работает
// и при открытой карточке другой заявки.
func TestLeaveDatesAndWithdrawWithDraft(t *testing.T) {
	e := newChatEnv(t, "")
	for _, q := range []string{"хочу отгул вчера", "хочу отгул 1 января 2020"} {
		msgs := e.say("intern_maria", map[string]any{"text": q})
		assert.Contains(t, msgs[0].Text, "уже прошла", q)
	}
	msgs := e.say("intern_maria", map[string]any{"text": "хочу отгул 2 декабря 2028"})
	assert.Contains(t, msgs[0].Text, "не дальше чем на год", "далекая дата")

	card := confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужен монитор"}))
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	number := r.Messages[0].Result.Number
	confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужна клавиатура и мышь"}))
	offer := e.say("intern_maria", map[string]any{"text": "отзови заявку №" + number})
	assert.Contains(t, offer[0].Text, "Отозвать заявку №"+number)
	assert.NotContains(t, offer[0].Text, "Моем пути", "у заявки в отдел нет задачи пути")
	done := e.say("intern_maria", map[string]any{"text": "да"})
	assert.Contains(t, done[0].Text, "отозвана")
}

// «отмена» текстом при открытой карточке отменяет ее; отгул с датой заменяет любую открытую карточку.
func TestCancelDraftByText(t *testing.T) {
	e := newChatEnv(t, "")
	card := confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужен монитор"}))
	msgs := e.say("intern_maria", map[string]any{"text": "отмена"})
	assert.Contains(t, msgs[0].Text, "заявку не создаю")
	var r models.ChatReply
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужна гарнитура"}))
	leave := e.say("intern_maria", map[string]any{"text": "хочу отгул вчера"})
	assert.Contains(t, leave[0].Text, "уже прошла")
}

// Ответ наставника находится и по другой формулировке: модель сверяет вопрос по смыслу с уже отвеченными.
func TestMentorAnswerBySense(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		w.Header().Set("Content-Type", "application/json")
		answer := func(content string) {
			out, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}}})
			_, _ = w.Write(out)
		}
		switch {
		case strings.Contains(body, "Сформулируй вопрос"):
			answer("Можно ли мне месяц поработать удаленно из Казани?")
		case strings.Contains(body, "сравниваешь вопросы") && strings.Contains(body, "другого города"):
			answer("1")
		case strings.Contains(body, "сравниваешь вопросы"):
			answer("0")
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"handoff_to_mentor","arguments":"{}"}}]}}]}`))
		}
	}))
	defer fake.Close()
	e := newChatEnv(t, fake.URL)

	e.say("intern_maria", map[string]any{"text": "передай наставнику: можно ли месяц поработать удаленно из Казани?"})
	sent := e.say("intern_maria", map[string]any{"text": "да"})
	require.Contains(t, sent[0].Text, "Передал")
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPost, "/api/mentor/threads/"+annaID+"/messages",
		map[string]any{"text": "Месяц можно, предупреди Артемия за две недели.", "addToKb": true}, nil))

	var mats []models.Material
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/materials", nil, &mats))
	t.Cleanup(func() { e.call("mentor_igor", http.MethodDelete, "/api/materials/"+mats[0].ID, nil, nil) })

	got := e.say("intern_maria", map[string]any{"text": "Могу ли я работать из другого города?"})
	assert.Contains(t, got[0].Text, "Игорь уже отвечал")
	assert.Contains(t, got[0].Text, "предупреди Артемия за две недели")
	other := e.say("intern_maria", map[string]any{"text": "можно ли брать собаку в офис?"})
	assert.NotContains(t, other[0].Text, "уже отвечал")
}

// Модель на вопрос дважды «оформляет» текстом без инструмента: вопрос идет в справочник, затем наставнику.
func TestQuestionNotTreatedAsRequest(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		content := "Подготавливаю заявку на удаленную работу, нажми «да»."
		if strings.Contains(string(raw), "сравниваешь вопросы") {
			content = "0"
		}
		out, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}}})
		_, _ = w.Write(out)
	}))
	defer fake.Close()
	e := newChatEnv(t, fake.URL)
	msgs := e.say("intern_maria", map[string]any{"text": "Могу ли я работать из другого города?"})
	assert.NotContains(t, msgs[0].Text, "Карточку заявки")
	assert.Contains(t, msgs[len(msgs)-1].Text, "в таком виде")
}

// «да» после карточки подтверждает ее; «нужен ноутбук» дает карточку и с моделью, которая только обещает.
func TestYesConfirmsCardAndNeedLaptop(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		out, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": "Давай подготовим заявку, подтверди."}}}})
		_, _ = w.Write(out)
	}))
	defer fake.Close()
	e := newChatEnv(t, fake.URL)
	card := confirmOf(t, e.say("intern_maria", map[string]any{"text": "Мне нужен ноутбук помощнее"}))
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/cancel", nil, nil))

	confirmOf(t, e.say("intern_maria", map[string]any{"text": "нужна гарнитура"}))
	done := e.say("intern_maria", map[string]any{"text": "да"})
	require.NotEmpty(t, done)
	assert.Equal(t, "result", done[len(done)-1].Kind)
	assert.Contains(t, done[len(done)-1].Text, "Заявка №")
}

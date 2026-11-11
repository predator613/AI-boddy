package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/database"
	"ai-buddy/backend/internal/mockservices"
	"ai-buddy/backend/internal/models"
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

	// Снимок прогресса Анны и метка времени: после теста все возвращаем как было.
	start := time.Now()
	var snapshot []byte
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COALESCE(json_agg(p), '[]') FROM intern_task_progress p WHERE intern_id = $1`, annaID).Scan(&snapshot))
	t.Cleanup(func() {
		bg := context.Background()
		for _, q := range []string{
			`DELETE FROM messages WHERE created_at >= $1`,
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
	for _, u := range []string{"intern_anna", "intern_pavel", "mentor_igor"} {
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
	for _, s := range e.path("intern_anna").Stages {
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
	msgs := e.say("intern_anna", map[string]any{"text": "Запроси доступ к GitLab"})
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
		"Согласует":    "Олег, владелец проекта",
	}, fieldsMap(card.Fields))

	// Черновик не виден на странице «Заявки» и ничего не меняет в пути.
	var list []models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/requests", nil, &list))
	for _, r := range list {
		assert.NotEqual(t, card.ActionID, r.ID)
	}

	// Чужой черновик подтвердить нельзя.
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil))

	// 2. Подтверждение: заявка уходит во внешнюю систему, задача пути – «на согласовании».
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	require.Len(t, r.Messages, 1)
	res := r.Messages[0].Result
	require.NotNil(t, res)
	assert.Equal(t, "waiting", res.Status)
	assert.Equal(t, "Доступ к GitLab", res.Title)
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil), "повторно не отправляется")

	task := taskStatus(t, e, gitlabTask)
	assert.Equal(t, "waiting", task.Status)
	assert.Contains(t, task.Note, "№"+res.Number)

	// Повторная просьба не создает дубль.
	again := e.say("intern_anna", map[string]any{"text": "дай доступ к гитлабу"})
	assert.Contains(t, again[0].Text, "уже на согласовании")

	// Пока не решено – синхронизация ничего не меняет.
	n, err := e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)
	assert.Zero(t, n)

	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/requests", nil, &list))
	require.Equal(t, res.Number, list[0].Number)
	assert.Equal(t, "waiting", list[0].Status)
	assert.Equal(t, "now", list[0].History[len(list[0].History)-1].State)
	assert.Contains(t, list[0].History[len(list[0].History)-1].Title, "Олег, владелец проекта")

	// 3. Согласующий одобрил: доступ выдан, пункт пути закрыт, стажер получил сообщение.
	ref := externalRef(t, e, res.RequestID)
	_, err = e.store.Decide(ref, true, "")
	require.NoError(t, err)
	n, err = e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/requests", nil, &list))
	assert.Equal(t, "done", list[0].Status)
	assert.Equal(t, "Доступ выдан", list[0].History[len(list[0].History)-1].Title)
	assert.Equal(t, "Олег, владелец проекта", fieldsMap(list[0].Fields)["Согласовал"])
	assert.Equal(t, "done", taskStatus(t, e, gitlabTask).Status)

	var hist models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/chat/messages", nil, &hist))
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
	assert.Contains(t, e.say("intern_anna", map[string]any{"text": "нужен доступ к gitlab"})[0].Text, "уже выдан")
}

func TestAccessRejectedAndAuto(t *testing.T) {
	e := newChatEnv(t, "")

	// Кнопка «Сделать с AI-buddy» у задачи: действие известно, проект в старом формате (group).
	msgs := e.say("intern_anna", map[string]any{
		"text":   "Получить доступ к GitLab",
		"action": map[string]any{"kind": "access_request", "payload": map[string]string{"service": "gitlab", "group": "backend", "role": "developer"}},
		"taskId": gitlabTask,
	})
	card := confirmOf(t, msgs)
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	_, err := e.store.Decide(externalRef(t, e, r.Messages[0].Result.RequestID), false, "Сначала пройди инструктаж по безопасности")
	require.NoError(t, err)
	_, err = e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)

	var list []models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/requests", nil, &list))
	assert.Equal(t, "rejected", list[0].Status)
	require.NotNil(t, list[0].RejectReason)
	assert.Equal(t, "Сначала пройди инструктаж по безопасности", *list[0].RejectReason)
	assert.Equal(t, "rejected", list[0].History[len(list[0].History)-1].State)
	task := taskStatus(t, e, gitlabTask)
	assert.Equal(t, "in_progress", task.Status, "после отказа задача снова в работе")
	assert.Contains(t, task.Note, "инструктаж")

	// Трекер прогресса выдается сразу: задача 2-й недели закрывается без согласования.
	msgs = e.say("intern_anna", map[string]any{"text": "Запроси трекер прогресса и внеси мои цели"})
	card = confirmOf(t, msgs)
	assert.Equal(t, "выдается автоматически", fieldsMap(card.Fields)["Согласование"])
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	assert.Equal(t, "done", r.Messages[0].Result.Status)
	assert.Contains(t, r.Messages[0].Text, "Пункт на «Моем пути» закрыт")
	assert.Equal(t, "done", taskStatus(t, e, "b7000000-0000-4000-8000-000000000205").Status)
}

func TestAssistantAnswersAndLimits(t *testing.T) {
	e := newChatEnv(t, "")

	// Ответ из справочника со ссылкой на статью.
	msgs := e.say("intern_anna", map[string]any{"text": "что такое код-ревью?"})
	assert.Equal(t, "text", msgs[0].Kind)
	require.NotNil(t, msgs[0].Source)
	assert.Contains(t, msgs[0].Source.Title, "код-ревью")
	require.NotNil(t, msgs[0].Source.ArticleID)

	// Ответа нет – помощник не выдумывает, а предлагает передать вопрос.
	msgs = e.say("intern_anna", map[string]any{"text": "сколько стоит квантовый компьютер"})
	assert.Nil(t, msgs[0].Source)
	assert.Contains(t, msgs[0].Text, "передай наставнику")

	// Доступ, запрещенный роли, не оформляется.
	msgs = e.say("intern_anna", map[string]any{"text": "дай доступ к логам продакшена"})
	assert.Equal(t, "error", msgs[0].Kind)
	assert.Equal(t, "sad", msgs[0].Mood)
	assert.Contains(t, msgs[0].Text, "Стажерам этот доступ не выдается")

	// Администрирование запрещено, но помощник предлагает разрешенный уровень.
	msgs = e.say("intern_anna", map[string]any{
		"text":   "Maintainer в GitLab",
		"action": map[string]any{"kind": "access_request", "payload": map[string]string{"service": "gitlab", "role": "maintainer"}},
	})
	assert.Equal(t, "error", msgs[0].Kind)
	assert.Contains(t, msgs[0].Text, "редактирование (Developer)")

	// Заявка на стенд уже есть (демо-данные) – дубль не создается.
	msgs = e.say("intern_anna", map[string]any{"text": "получить доступ к тестовому стенду"})
	assert.Contains(t, msgs[0].Text, "№1038 уже на согласовании: Кирилл, DevOps")

	// Непонятный сервис.
	msgs = e.say("intern_anna", map[string]any{"text": "дай доступ к холодильнику"})
	assert.Contains(t, msgs[0].Text, "Не понял, к какому сервису")

	// Статус заявок.
	msgs = e.say("intern_anna", map[string]any{"text": "какой статус моих заявок?"})
	assert.Contains(t, msgs[0].Text, "№1038 Доступ к тестовому стенду – на согласовании (Кирилл, DevOps)")
	msgs = e.say("intern_anna", map[string]any{"text": "что с заявкой №1027"})
	assert.Contains(t, msgs[0].Text, "отклонена")
	assert.NotContains(t, msgs[0].Text, "1038")

	// Передать вопрос наставнику.
	msgs = e.say("intern_anna", map[string]any{"text": "передай наставнику: можно брать задачи из чужого спринта?"})
	assert.Contains(t, msgs[0].Text, "Игорь ответит")
	pool := testPool(t)
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT count(*) FROM handoffs h JOIN messages m ON m.id = h.forwarded_id
		WHERE m.text LIKE '%чужого спринта%'`).Scan(&n))
	assert.Equal(t, 1, n)

	// Ответ по статье и действие «открыть статью».
	msgs = e.say("intern_anna", map[string]any{
		"text":   "Открой памятку",
		"action": map[string]any{"kind": "open_article", "payload": map[string]string{"articleId": "c2000000-0000-4000-8000-000000000001"}},
	})
	require.NotNil(t, msgs[0].Source)

	// Вопрос-действие ask_buddy идет через модель.
	msgs = e.say("intern_anna", map[string]any{
		"text":   "Как устроено ревью?",
		"action": map[string]any{"kind": "ask_buddy", "payload": map[string]string{"question": "как устроено код-ревью"}},
	})
	require.NotNil(t, msgs[0].Source)

	// Права и ввод.
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodPost, "/api/chat/messages", map[string]any{"text": "привет"}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodGet, "/api/chat/messages", nil, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("intern_anna", http.MethodPost, "/api/chat/messages", map[string]any{"text": "  "}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("intern_anna", http.MethodPost, "/api/chat/messages", map[string]any{"text": strings.Repeat("я", 2001)}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodPost, "/api/chat/actions/nope/confirm", nil, nil))
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

	msgs := e.say("intern_anna", map[string]any{"text": "хочу отгул " + ddmm(workday) + " за свой счет"})
	card := confirmOf(t, msgs)
	f := fieldsMap(card.Fields)
	assert.True(t, strings.HasPrefix(f["Дата"], fmt.Sprintf("%d ", workday.Day())), f["Дата"])
	assert.Equal(t, "отгул за свой счет", f["Тип"])
	assert.Equal(t, "Олег, руководитель", f["Согласует"])
	require.NotNil(t, msgs[0].Source)
	assert.Contains(t, msgs[0].Source.Title, "отгул")

	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	assert.Equal(t, "waiting", r.Messages[0].Result.Status)
	_, err := e.store.Decide(externalRef(t, e, r.Messages[0].Result.RequestID), true, "")
	require.NoError(t, err)
	_, err = e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)
	var list []models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/requests", nil, &list))
	assert.Equal(t, "approved", list[0].Status)
	assert.Equal(t, "leave", list[0].Kind)
	assert.True(t, strings.HasPrefix(list[0].Title, fmt.Sprintf("Отгул за свой счет %d ", workday.Day())), list[0].Title)

	// Отмена черновика.
	card = confirmOf(t, e.say("intern_anna", map[string]any{"text": "отгул " + ddmm(workday.AddDate(0, 0, 7))}))
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/cancel", nil, &r))
	assert.Contains(t, r.Messages[0].Text, "заявку не создаю")
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil))

	// Выходной день.
	msgs = e.say("intern_anna", map[string]any{"text": "отгул " + ddmm(saturday)})
	assert.Contains(t, msgs[0].Text, "выходной")
}

func TestRequestServiceUnavailable(t *testing.T) {
	e := newChatEnv(t, "")
	card := confirmOf(t, e.say("intern_anna", map[string]any{"text": "запроси доступ к gitlab"}))
	// Имитация недоступна: стажер видит ошибку, черновик остается, можно повторить.
	srvDown := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer srvDown.Close()
	cfg := testConfig(t, true)
	cfg.MockServicesURL = srvDown.URL
	svc, err := NewServices(cfg, testPool(t), quietLog)
	require.NoError(t, err)
	down := httptest.NewServer(NewRouter(cfg, svc, quietLog))
	defer down.Close()
	resp := do(t, http.MethodPost, down.URL+"/api/chat/actions/"+card.ActionID+"/confirm", e.tok["intern_anna"], nil)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, nil), "после восстановления заявка отправляется")
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

	msgs := e.say("intern_anna", map[string]any{"text": "кто смотрит мой код?"})
	assert.Equal(t, "Ревью делает коллега из команды, после одобрения код вливается.", msgs[0].Text)
	require.NotNil(t, msgs[0].Source)
	assert.Equal(t, 2, calls)

	// Модель недоступна – ответ собирается офлайн, чат не падает.
	msgs = e.say("intern_anna", map[string]any{"text": "сломай и расскажи про код-ревью"})
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
	assert.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/kb/articles/"+article, nil, nil))
	assert.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/kb/articles/"+article, nil, nil))

	// Ответ другого наставника той же команды Анна не видит, руководитель – видит.
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, login, password_hash, name, role, team_id)
		VALUES ($1, 'mentor_test_other', 'x', 'Петр', 'mentor', '11111111-1111-4111-8111-111111111111')`, otherMentor)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE kb_articles SET author_id = $2 WHERE id = $1`, article, otherMentor)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodGet, "/api/kb/articles/"+article, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodGet, "/api/kb/articles/"+article, nil, nil))
	assert.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodGet, "/api/kb/articles/"+article, nil, nil))
}

func externalRef(t *testing.T, e *chatEnv, requestID string) string {
	t.Helper()
	var ref string
	require.NoError(t, testPool(t).QueryRow(context.Background(),
		`SELECT external_ref FROM requests WHERE id = $1`, requestID).Scan(&ref))
	_ = e
	return ref
}

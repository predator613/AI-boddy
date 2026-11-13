package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/integrations"
	"ai-buddy/backend/internal/mockservices"
	"ai-buddy/backend/internal/models"
)

const (
	igorID  = "aaaaaaaa-0000-4000-8000-000000000002"
	reviewM = annaOneOnOne // «1:1 с наставником» – встреча Марии
)

func integrationsOrder(kind string) integrations.Order {
	return integrations.Order{Kind: kind, Requester: "intern_anna", Title: "test"}
}

// newStaffEnv – чат-окружение плюс руководители и Лиза; после теста убирает обращения, заметки и отметки.
func newStaffEnv(t *testing.T) *chatEnv {
	t.Helper()
	e := newChatEnv(t, "")
	for _, u := range []string{"lead_oleg", "lead_nina", "intern_liza"} {
		code, token, _ := login(t, e.base, u, "demo123")
		require.Equal(t, http.StatusOK, code)
		e.tok[u] = token
	}
	start := time.Now()
	pool := testPool(t)
	var marks, notes, pavel []byte
	ctx := context.Background()
	require.NoError(t, pool.QueryRow(ctx, `SELECT COALESCE(json_agg(p), '[]') FROM intern_task_progress p WHERE intern_id = $1`, pavelID).Scan(&pavel))
	require.NoError(t, pool.QueryRow(ctx, `SELECT COALESCE(json_agg(m), '[]') FROM mentor_checklist_marks m`).Scan(&marks))
	require.NoError(t, pool.QueryRow(ctx, `SELECT COALESCE(json_agg(n), '[]') FROM meeting_notes n`).Scan(&notes))
	t.Cleanup(func() {
		bg := context.Background()
		for _, q := range []string{
			`DELETE FROM support_tickets WHERE created_at >= $1`,
			`UPDATE notifications SET read_at = NULL WHERE read_at >= $1`,
			`DELETE FROM meeting_answers WHERE created_at >= $1`,
		} {
			_, err := pool.Exec(bg, q, start)
			assert.NoError(t, err, q)
		}
		_, err := pool.Exec(bg, `DELETE FROM intern_task_progress WHERE intern_id = $1`, pavelID)
		assert.NoError(t, err)
		_, err = pool.Exec(bg, `INSERT INTO intern_task_progress SELECT * FROM json_populate_recordset(NULL::intern_task_progress, $1)`, pavel)
		assert.NoError(t, err)
		for table, snap := range map[string][]byte{"mentor_checklist_marks": marks, "meeting_notes": notes} {
			_, err := pool.Exec(bg, `DELETE FROM `+table)
			assert.NoError(t, err)
			_, err = pool.Exec(bg, `INSERT INTO `+table+` SELECT * FROM json_populate_recordset(NULL::`+table+`, $1)`, snap)
			assert.NoError(t, err)
		}
	})
	return e
}

func feedOf(t *testing.T, e *chatEnv, user string) models.Feed {
	t.Helper()
	var f models.Feed
	require.Equal(t, http.StatusOK, e.call(user, http.MethodGet, "/api/mentor/feed", nil, &f))
	return f
}

// Заявка на GitLab уходит владельцу проекта: он видит ее в «Согласованиях», решает, а стажер,
// его путь и лента наставника обновляются.
func TestApprovalsFlow(t *testing.T) {
	e := newStaffEnv(t)

	card := confirmOf(t, e.say("intern_anna", map[string]any{"text": "нужен доступ в GitLab"}))
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	res := r.Messages[0].Result
	require.NotNil(t, res)
	require.NotNil(t, res.External, "в карточке чата – ключ заявки в Jira")
	assert.Regexp(t, `^ACCESS-\d+$`, res.External.Key)
	assert.Equal(t, "Jira Service Desk", res.External.System)

	// Стажер и чужой руководитель согласований не видят.
	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodGet, "/api/approvals", nil, nil))
	var list []models.Approval
	require.Equal(t, http.StatusOK, e.call("lead_nina", http.MethodGet, "/api/approvals", nil, &list))
	assert.Empty(t, list)

	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodGet, "/api/approvals", nil, &list))
	require.NotEmpty(t, list)
	ap := list[0]
	assert.Equal(t, res.RequestID, ap.ID, "ожидающая заявка – первой")
	assert.Equal(t, "waiting", ap.Status)
	assert.Equal(t, "Мария", ap.InternName)
	assert.Equal(t, "Доступ к GitLab", ap.Title)
	f := fieldsMap(ap.Fields)
	assert.Contains(t, f["Уровень прав"], "Developer")
	assert.Contains(t, f["Почему ты"], "владелец проекта")
	assert.Contains(t, f["Почему ты"], res.External.Key)
	assert.NotContains(t, f, "Согласует")

	path := "/api/approvals/" + ap.ID
	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodPost, path, map[string]any{"decision": "approve"}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("lead_nina", http.MethodPost, path, map[string]any{"decision": "approve"}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPost, path, map[string]any{"decision": "approve"}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("lead_oleg", http.MethodPost, path, map[string]any{"decision": "reject"}, nil), "отказ без причины")
	assert.Equal(t, http.StatusBadRequest, e.call("lead_oleg", http.MethodPost, path, map[string]any{"decision": "maybe"}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("lead_oleg", http.MethodPost, path,
		map[string]any{"decision": "reject", "reason": string(make([]rune, 301))}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("lead_oleg", http.MethodPost, "/api/approvals/00000000-0000-4000-8000-000000000000",
		map[string]any{"decision": "approve"}, nil))

	before := feedOf(t, e, "mentor_igor").Unread
	var done models.Approval
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodPost, path, map[string]any{"decision": "approve"}, &done))
	assert.Equal(t, "approved", done.Status)
	assert.Equal(t, http.StatusConflict, e.call("lead_oleg", http.MethodPost, path, map[string]any{"decision": "approve"}, nil), "повторное решение")

	// У стажера: заявка выполнена, задача пути закрыта, в чате сообщение.
	var reqs []models.RequestItem
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/requests", nil, &reqs))
	assert.Equal(t, "done", reqs[0].Status)
	require.NotNil(t, reqs[0].External)
	assert.Equal(t, res.External.Key, reqs[0].External.Key)
	titles := []string{}
	for _, h := range reqs[0].History {
		titles = append(titles, h.Title)
	}
	assert.Contains(t, titles, "Передана в Jira Service Desk")
	assert.Equal(t, "done", taskStatus(t, e, gitlabTask).Status)

	// Сервис заявок знает о решении, синхронизация повторно ничего не меняет.
	n, err := e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)
	assert.Zero(t, n)

	// Лента наставника: решение руководителя.
	feed := feedOf(t, e, "mentor_igor")
	assert.Equal(t, before+1, feed.Unread)
	assert.Equal(t, "Артемий согласовал заявку Марии «Доступ к GitLab»", feed.Items[0].Text)
	assert.Equal(t, models.Actor{Name: "Артемий", Kind: "lead"}, feed.Items[0].Actor)
	assert.False(t, feed.Items[0].Read)

	// Отказ с причиной: отгул остается у руководителя, причина видна стажеру.
	leave := confirmOf(t, e.say("intern_anna", map[string]any{"text": "хочу отгул 20 ноября"}))
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+leave.ActionID+"/confirm", nil, &r))
	assert.Nil(t, r.Messages[0].Result.External, "отгул в Jira не уходит")
	var rejected models.Approval
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodPost, "/api/approvals/"+leave.ActionID,
		map[string]any{"decision": "reject", "reason": "В этот день релиз"}, &rejected))
	assert.Equal(t, "rejected", rejected.Status)
	require.NotNil(t, rejected.RejectReason)
	assert.Contains(t, fieldsMap(rejected.Fields)["Почему ты"], "руководитель")
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/requests", nil, &reqs))
	assert.Equal(t, "rejected", reqs[0].Status)
	assert.Equal(t, "В этот день релиз", *reqs[0].RejectReason)
	assert.Equal(t, "Артемий отклонил заявку Марии «Отгул 20 ноября»", feedOf(t, e, "mentor_igor").Items[0].Text)
}

// Обычные доступы решает команда доступов в Jira: руководителю они не приходят.
func TestAccessGoesToJira(t *testing.T) {
	e := newStaffEnv(t)
	pool := testPool(t)
	_, err := pool.Exec(context.Background(), `UPDATE requests SET status = 'cancelled' WHERE number = 1038`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), `UPDATE requests SET status = 'waiting' WHERE number = 1038`)
		assert.NoError(t, err)
	})

	msgs := e.say("intern_anna", map[string]any{"text": "нужен доступ к тестовому стенду"})
	card := confirmOf(t, msgs)
	assert.Contains(t, msgs[0].Text, "Jira Service Desk, команда доступов")
	assert.Equal(t, "Jira Service Desk, команда доступов", fieldsMap(card.Fields)["Куда ушла"])

	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	res := r.Messages[0].Result
	assert.Equal(t, "waiting", res.Status)

	var list []models.Approval
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodGet, "/api/approvals", nil, &list))
	for _, a := range list {
		assert.NotEqual(t, res.RequestID, a.ID, "доступ к стенду не идет руководителю")
	}

	// Решение сервиса попадает в ленту наставника.
	_, err = e.store.Resolve(res.External.Key, true, "")
	require.NoError(t, err)
	n, err := e.svc.Requests.SyncOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	top := feedOf(t, e, "mentor_igor").Items[0]
	assert.Equal(t, "Заявка стажера «Доступ к тестовому стенду» выполнена", top.Text)
	assert.Equal(t, models.Actor{Name: "Jira Service Desk", Kind: "other"}, top.Actor)

	// Карточка заявки в имитации Jira.
	srv := httptest.NewServer(mockservices.Handler(e.store))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/browse/" + res.External.Key)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), res.External.Key)
	assert.Contains(t, string(body), "Доступ к тестовому стенду")
	assert.Contains(t, string(body), "Выполнена")
	resp, err = http.Get(srv.URL + "/browse/ACCESS-1")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestMentorFeed(t *testing.T) {
	e := newStaffEnv(t)

	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodGet, "/api/mentor/feed", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodPost, "/api/mentor/feed/read", nil, nil))

	feed := feedOf(t, e, "mentor_igor")
	require.NotEmpty(t, feed.Items)
	assert.LessOrEqual(t, len(feed.Items), 20)
	assert.Positive(t, feed.Unread)
	for i := 1; i < len(feed.Items); i++ {
		assert.False(t, feed.Items[i].At.After(feed.Items[i-1].At), "новые сверху")
	}
	before := feed.Unread

	// Стажер закрыл задачу и ответил на приглашение – наставник видит оба события.
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/path/tasks/"+gitlabTask+"/complete", nil, nil))
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/me/calendar/events/"+reviewM+"/answer",
		map[string]any{"answer": "accept"}, nil))
	feed = feedOf(t, e, "mentor_igor")
	assert.Equal(t, before+2, feed.Unread)
	assert.Equal(t, "Мария придет на «1:1 с наставником»", feed.Items[0].Text)
	assert.Equal(t, models.Actor{Name: "Мария", Kind: "intern"}, feed.Items[0].Actor)
	assert.Equal(t, "Мария закрыл(а) задачу «Получить доступ к GitLab»", feed.Items[1].Text)

	// Наставнику не пишем о его собственных действиях; у стажера без наставника запись некуда класть.
	require.Equal(t, http.StatusOK, e.call("intern_pavel", http.MethodPost, "/api/path/tasks/b7000000-0000-4000-8000-000000001102/complete", nil, nil))
	assert.Equal(t, before+2, feedOf(t, e, "mentor_igor").Unread)

	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodPost, "/api/mentor/feed/read", nil, nil))
	feed = feedOf(t, e, "mentor_igor")
	assert.Zero(t, feed.Unread)
	for _, it := range feed.Items {
		assert.True(t, it.Read)
	}

	// У руководителя своя лента (может быть пустой).
	lead := feedOf(t, e, "lead_oleg")
	assert.NotNil(t, lead.Items)
}

func TestMentorChecklist(t *testing.T) {
	e := newStaffEnv(t)
	path := "/api/mentor/interns/" + annaID + "/checklist"

	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodGet, path, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("lead_nina", http.MethodGet, path, nil, nil), "чужая команда")
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodGet, "/api/mentor/interns/"+pavelID+"/checklist", nil, nil), "не его стажер")
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodGet, "/api/mentor/interns/"+igorID+"/checklist", nil, nil), "не стажер")

	var c models.MentorChecklist
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, path, nil, &c))
	assert.Equal(t, annaID, c.InternID)
	require.Len(t, c.Items, 10)
	assert.Equal(t, "Добавить в чаты и каналы", c.Items[0].Title)
	assert.Equal(t, 1, c.Items[0].FromWeek)
	assert.Equal(t, "неделя 1", c.Items[0].Period)
	assert.True(t, c.Items[0].Done)
	assert.True(t, c.Items[3].Done)
	assert.False(t, c.Items[4].Done)
	assert.Equal(t, 13, c.Items[9].FromWeek)

	item := c.Items[4].ID
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPost, path+"/"+item, map[string]any{"done": true}, &c))
	assert.True(t, c.Items[4].Done)
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPost, path+"/"+item, map[string]any{"done": true}, &c), "повторная отметка")
	assert.True(t, c.Items[4].Done)

	// Руководитель видит отметки наставника стажера и может их менять.
	var lead models.MentorChecklist
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodGet, path, nil, &lead))
	assert.True(t, lead.Items[4].Done)
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodPost, path+"/"+item, map[string]any{"done": false}, &lead))
	assert.False(t, lead.Items[4].Done)
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, path, nil, &c))
	assert.False(t, c.Items[4].Done)

	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, path+"/"+item, map[string]any{}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPost, path+"/00000000-0000-4000-8000-000000000000", map[string]any{"done": true}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPost, path+"/00000000-0000-4000-8000-000000000000", map[string]any{"done": false}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodPost, path+"/"+item, map[string]any{"done": true}, nil))
}

func TestMeetingNotes(t *testing.T) {
	e := newStaffEnv(t)

	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodGet, "/api/me/notes", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodPut, "/api/me/notes/"+reviewM, map[string]any{"text": "x"}, nil))

	var notes []models.MeetingNote
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/me/notes", nil, &notes))
	require.Len(t, notes, 1)
	assert.Equal(t, reviewM, notes[0].EventID)

	var n models.MeetingNote
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPut, "/api/me/notes/"+reviewM, map[string]any{"text": "  Новый конспект  "}, &n))
	assert.Equal(t, "Новый конспект", n.Text)
	assert.Equal(t, reviewM, n.EventID)
	assert.WithinDuration(t, time.Now(), n.UpdatedAt, time.Minute)

	// Заметка к своей записи календаря.
	var view models.CalendarView
	from, to := time.Now().AddDate(0, 0, -1).Format(time.RFC3339), time.Now().AddDate(0, 0, 14).Format(time.RFC3339)
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/me/calendar?from="+from[:10]+"T00:00:00Z&to="+to[:10]+"T00:00:00Z", nil, &view))
	own := ""
	for _, ev := range view.Events {
		if ev.Status == "own" {
			own = ev.ID
		}
	}
	require.NotEmpty(t, own)
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPut, "/api/me/notes/"+own, map[string]any{"text": "план на фокус"}, &n))
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/me/notes", nil, &notes))
	assert.Len(t, notes, 2)

	// Пустой текст удаляет; чужое событие и длинный текст не принимаются.
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPut, "/api/me/notes/"+own, map[string]any{"text": " "}, &n))
	assert.Empty(t, n.Text)
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/me/notes", nil, &notes))
	assert.Len(t, notes, 1)
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodPut, "/api/me/notes/"+reviewM, map[string]any{"text": "x"}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodPut, "/api/me/notes/not-a-uuid", map[string]any{"text": "x"}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("intern_anna", http.MethodPut, "/api/me/notes/"+reviewM, map[string]any{"text": string(make([]rune, 4001))}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("intern_anna", http.MethodPut, "/api/me/notes/"+reviewM, map[string]any{}, nil))

	// Заметки видит только автор.
	require.Equal(t, http.StatusOK, e.call("intern_pavel", http.MethodGet, "/api/me/notes", nil, &notes))
	assert.Empty(t, notes)
}

func TestSupportTickets(t *testing.T) {
	e := newStaffEnv(t)

	var list []models.SupportTicket
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/support/tickets", nil, &list))
	assert.Empty(t, list)

	e.say("intern_anna", map[string]any{"text": "как оформить отпуск"})
	var tk models.SupportTicket
	require.Equal(t, http.StatusCreated, e.call("intern_anna", http.MethodPost, "/api/support/tickets",
		map[string]any{"type": "Помощник ответил неверно", "details": "  Ответ про отпуск не по теме  ", "internId": nil, "attachDialog": true}, &tk))
	assert.Regexp(t, `^SUP-\d+$`, tk.Number)
	assert.Equal(t, "new", tk.Status)
	assert.Equal(t, "Ответ про отпуск не по теме", tk.Details)
	assert.Nil(t, tk.Answer)
	var dialog *string
	require.NoError(t, testPool(t).QueryRow(context.Background(), `SELECT dialog_id::text FROM support_tickets WHERE id = $1`, tk.ID).Scan(&dialog))
	assert.NotNil(t, dialog, "к обращению приложен открытый диалог")

	for name, body := range map[string]map[string]any{
		"без описания":    {"type": "Другое", "details": " "},
		"без типа":        {"type": "", "details": "x"},
		"длинное":         {"type": "Другое", "details": string(make([]rune, 2001))},
		"стажер о других": {"type": "Другое", "details": "x", "internId": pavelID},
	} {
		assert.Equal(t, http.StatusBadRequest, e.call("intern_anna", http.MethodPost, "/api/support/tickets", body, nil), name)
	}

	// Наставник пишет про своего стажера; про чужого – нельзя.
	var mt models.SupportTicket
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/support/tickets",
		map[string]any{"type": "Не согласуется заявка", "details": "Заявка Марии висит", "internId": annaID, "attachDialog": true}, &mt))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/support/tickets",
		map[string]any{"type": "Другое", "details": "x", "internId": pavelID}, nil))
	require.Equal(t, http.StatusCreated, e.call("lead_oleg", http.MethodPost, "/api/support/tickets",
		map[string]any{"type": "Другое", "details": "x"}, nil))

	// Каждый видит только свои обращения, новые сверху.
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/support/tickets", nil, &list))
	require.Len(t, list, 1)
	assert.Equal(t, tk.Number, list[0].Number)
	assert.Equal(t, "Помощник ответил неверно", list[0].Type)

	// Поддержка ответила: статус, ответ и уведомление автору.
	_, err := e.store.Resolve(tk.Number, true, "")
	require.NoError(t, err)
	n, err := e.svc.Support.SyncOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	n, err = e.svc.Support.SyncOnce(context.Background())
	require.NoError(t, err)
	assert.Zero(t, n)
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/support/tickets", nil, &list))
	assert.Equal(t, "done", list[0].Status)
	require.NotNil(t, list[0].Answer)
	assert.Equal(t, mockservices.SupportAnswer, *list[0].Answer)
	var notice string
	require.NoError(t, testPool(t).QueryRow(context.Background(),
		`SELECT text FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, annaID).Scan(&notice))
	assert.Contains(t, notice, tk.Number+" решено")
}

// Обращение само проходит «в работе» → «решено» по времени имитации.
func TestSupportTicketTimeline(t *testing.T) {
	store := mockservices.NewStore(time.Minute)
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	store.SetNow(func() time.Time { return now })
	ctx := context.Background()
	tk, err := store.Submit(ctx, integrationsOrder("support"))
	require.NoError(t, err)
	assert.Equal(t, "waiting", tk.Status)
	now = now.Add(61 * time.Second)
	tk, _ = store.Ticket(ctx, tk.Ref)
	assert.Equal(t, "in_progress", tk.Status)
	now = now.Add(time.Minute)
	tk, _ = store.Ticket(ctx, tk.Ref)
	assert.Equal(t, "done", tk.Status)
	assert.Equal(t, mockservices.SupportAnswer, tk.Answer)

	// Заявку, которую решает человек, имитация ждет дольше обычной.
	o := integrationsOrder("access")
	o.Manual = true
	manual, _ := store.Submit(ctx, o)
	plain, _ := store.Submit(ctx, integrationsOrder("access"))
	now = now.Add(2 * time.Minute)
	manual, _ = store.Ticket(ctx, manual.Ref)
	plain, _ = store.Ticket(ctx, plain.Ref)
	assert.Equal(t, "waiting", manual.Status)
	assert.Equal(t, "granted", plain.Status)
	now = now.Add(5 * time.Minute)
	manual, _ = store.Ticket(ctx, manual.Ref)
	assert.Equal(t, "granted", manual.Status)
}

// Подробности задачи: описание, кто поможет (с именами из связей) и цель стажировки.
func TestTaskDetails(t *testing.T) {
	e := newStaffEnv(t)
	task := taskStatus(t, e, gitlabTask)
	require.NotNil(t, task.Details)
	assert.Contains(t, task.Details.About, "GitLab")
	assert.Contains(t, task.Details.Helper, "Артемий")
	assert.NotContains(t, task.Details.Helper, "{{")
	assert.NotEmpty(t, task.Details.DoneWhen)
	require.NotNil(t, task.Details.Goal)
	assert.Equal(t, "Пройти путь задачи до релиза", *task.Details.Goal)

	// Задача без цели и задача команды без описаний.
	intro := taskStatus(t, e, "b7000000-0000-4000-8000-000000000101")
	require.NotNil(t, intro.Details)
	assert.Nil(t, intro.Details.Goal)
	assert.Contains(t, intro.Details.Helper, "Игорь")
	for _, s := range e.path("intern_pavel").Stages {
		for _, task := range s.Tasks {
			assert.Nil(t, task.Details, task.Title)
		}
	}
}

// Цель с шагами и компетенцией: шаг-задача привязывается к задаче плана по названию.
func TestGoalStepsAndCompetency(t *testing.T) {
	e := newStaffEnv(t)
	pool := testPool(t)
	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), `DELETE FROM intern_goals WHERE title LIKE 'test:%'`)
		assert.NoError(t, err)
	})

	var g models.Goal
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/interns/"+lizaID+"/goals", map[string]any{
		"title": "test: первые шаги", "criterion": "все шаги закрыты", "competency": "Код и ревью · L1",
		"steps": []map[string]any{
			{"title": "Получить доступ к GitLab", "source": "task"},
			{"title": "Задача, которой нет в плане", "source": "task"},
			{"title": "Merge в main", "source": "mentor"},
		},
	}, &g))
	require.NotNil(t, g.Competency)
	assert.Equal(t, "Код и ревью · L1", *g.Competency)
	require.Len(t, g.Steps, 3)
	require.NotNil(t, g.Steps[0].TaskID)
	assert.Equal(t, gitlabTask, *g.Steps[0].TaskID)
	assert.Nil(t, g.Steps[1].TaskID)
	assert.Equal(t, "mentor", g.Steps[2].Source)

	// Наставник отметил шаг; PATCH без steps шаги не трогает, компетенцию очищает пустой строкой.
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPost, "/api/goals/"+g.ID+"/steps/"+g.Steps[2].ID, map[string]any{"done": true}, &g))
	assert.Equal(t, 33, g.ProgressPct)
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPatch, "/api/goals/"+g.ID, map[string]any{"competency": ""}, &g))
	assert.Nil(t, g.Competency)
	require.Len(t, g.Steps, 3)
	assert.Equal(t, 33, g.ProgressPct)

	// PATCH со steps заменяет список; отметка шага с тем же названием сохраняется.
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPatch, "/api/goals/"+g.ID, map[string]any{
		"steps": []map[string]any{{"title": "Merge в main", "source": "mentor"}, {"title": "Демо команде", "source": "lead"}},
	}, &g))
	require.Len(t, g.Steps, 2)
	assert.Equal(t, "done", g.Steps[0].Status)
	assert.Equal(t, 50, g.ProgressPct)
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPatch, "/api/goals/"+g.ID, map[string]any{"steps": []map[string]any{}}, &g))
	assert.Empty(t, g.Steps)

	for name, body := range map[string]map[string]any{
		"много шагов":     {"steps": make([]map[string]any, 9)},
		"пустой шаг":      {"steps": []map[string]any{{"title": " ", "source": "mentor"}}},
		"неизвестный вид": {"steps": []map[string]any{{"title": "x", "source": "buddy"}}},
		"длинная компет.": {"competency": string(make([]rune, 81))},
	} {
		assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPatch, "/api/goals/"+g.ID, body, nil), name)
	}

	// Руководитель подтвердил шаг и согласовал цели – наставник видит это в ленте.
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPatch, "/api/goals/"+g.ID, map[string]any{
		"steps": []map[string]any{{"title": "Демо команде", "source": "lead"}},
	}, &g))
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodPost, "/api/goals/"+g.ID+"/steps/"+g.Steps[0].ID, map[string]any{"done": true}, &g))
	assert.Equal(t, "Артемий подтвердил шаг «Демо команде» у Лизы", feedOf(t, e, "mentor_igor").Items[0].Text)
}

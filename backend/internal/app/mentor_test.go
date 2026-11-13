package app

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/models"
)

func newMentorEnv(t *testing.T) *chatEnv {
	t.Helper()
	e := newStaffEnv(t)
	start := time.Now()
	pool := testPool(t)
	t.Cleanup(func() {
		bg := context.Background()
		for _, q := range []string{
			`DELETE FROM kb_articles WHERE created_at >= $1 AND source <> 'seed'`,
			`DELETE FROM kb_sections WHERE hand = 'материалы' AND NOT EXISTS (SELECT 1 FROM kb_articles a WHERE a.section_id = kb_sections.id)`,
			`DELETE FROM meetings WHERE created_at >= $1`,
			`DELETE FROM conversation_reads WHERE last_read_at >= $1`,
			`DELETE FROM mentor_automations`,
			`DELETE FROM automation_runs WHERE created_at >= $1`,
		} {
			var err error
			if bytes.Contains([]byte(q), []byte("$1")) {
				_, err = pool.Exec(bg, q, start)
			} else {
				_, err = pool.Exec(bg, q)
			}
			assert.NoError(t, err, q)
		}
	})
	return e
}

// multipartCall – POST /api/materials формой.
func (e *chatEnv) multipartCall(user string, fields map[string]string, file string, content []byte, out any) int {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	if file != "" {
		fw, _ := w.CreateFormFile("file", file)
		_, _ = fw.Write(content)
	}
	_ = w.Close()
	req, _ := http.NewRequest(http.MethodPost, e.base+"/api/materials", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+e.tok[user])
	resp, err := http.DefaultClient.Do(req)
	require.NoError(e.t, err)
	defer resp.Body.Close()
	if out != nil && resp.StatusCode < 300 {
		require.NoError(e.t, json.NewDecoder(resp.Body).Decode(out))
	}
	return resp.StatusCode
}

// Вопрос, который AI-buddy не знает: передача наставнику, ответ в переписке и в чате стажера,
// сохранение ответа в справочник – дальше AI-buddy отвечает им сам.
func TestMentorThreadsAndHandoff(t *testing.T) {
	e := newMentorEnv(t)

	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodGet, "/api/mentor/threads", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("lead_oleg", http.MethodGet, "/api/mentor/threads", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodGet, "/api/me/mentor-thread", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodGet, "/api/me/mentor-thread", nil, nil), "без наставника")
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodGet, "/api/mentor/threads/"+pavelID, nil, nil))

	e.say("intern_anna", map[string]any{"text": "передай наставнику: где взять пример конфига для auth-service?"})
	e.say("intern_anna", map[string]any{"text": "да"}) // передача только после согласия стажера

	var list []models.ThreadSummary
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/mentor/threads", nil, &list))
	require.NotEmpty(t, list)
	assert.Equal(t, annaID, list[0].InternID, "свежая переписка сверху")
	assert.Positive(t, list[0].Unread)
	assert.NotEmpty(t, list[0].StageCode)

	var th models.Thread
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/mentor/threads/"+annaID, nil, &th))
	last := th.Messages[len(th.Messages)-1]
	assert.Equal(t, "handoff", last.Kind)
	assert.Equal(t, "buddy", last.Author)
	assert.Equal(t, "Игорь", th.Mentor.Name)

	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/mentor/threads/"+annaID+"/messages", map[string]any{"text": " "}, nil))
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPost, "/api/mentor/threads/"+annaID+"/messages",
		map[string]any{"text": "Пример конфига лежит в репозитории auth-service, папка deploy/example.", "addToKb": true}, &th))
	assert.Equal(t, "mentor", th.Messages[len(th.Messages)-1].Author)
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/mentor/threads", nil, &list))
	assert.Zero(t, list[0].Unread)

	// Ответ пришел стажеру в чат AI-buddy.
	var h models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/chat/messages", nil, &h))
	assert.Contains(t, h.Messages[len(h.Messages)-1].Text, "Игорь ответил на твой вопрос: Пример конфига")

	// Ответ в справочнике и в материалах.
	var mats []models.Material
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/materials", nil, &mats))
	assert.Equal(t, "answer", mats[0].Kind)
	assert.Equal(t, "Ответы наставников", *mats[0].Section)
	var found models.KBSearch
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/kb/search?q=пример+конфига+deploy", nil, &found))
	assert.NotEmpty(t, found.Results)

	// Стажер пишет сам и подтверждает карточку изменения.
	assert.Equal(t, http.StatusBadRequest, e.call("intern_anna", http.MethodPost, "/api/me/mentor-thread/messages", map[string]any{"text": ""}, nil))
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/me/mentor-thread/messages", map[string]any{"text": "Спасибо, нашла!"}, &th))
	assert.Equal(t, "intern", th.Messages[len(th.Messages)-1].Author)
	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodPost, "/api/mentor/threads/"+annaID+"/read", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPost, "/api/mentor/threads/"+pavelID+"/read", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodPost, "/api/me/mentor-thread/updates/"+th.Messages[0].ID+"/ack", nil, nil), "не карточка")
}

func TestMentorMaterials(t *testing.T) {
	e := newMentorEnv(t)
	var m models.Material
	assert.Equal(t, http.StatusForbidden, e.multipartCall("intern_anna", map[string]string{"title": "x", "url": "https://x.ru"}, "", nil, nil))
	assert.Equal(t, http.StatusBadRequest, e.multipartCall("mentor_igor", map[string]string{"title": "x"}, "", nil, nil), "ни файла, ни ссылки")
	assert.Equal(t, http.StatusBadRequest, e.multipartCall("mentor_igor", map[string]string{"title": "x", "url": "ftp://x"}, "", nil, nil))
	assert.Equal(t, http.StatusBadRequest, e.multipartCall("mentor_igor", map[string]string{"title": "x"}, "a.exe", []byte("x"), nil))
	assert.Equal(t, http.StatusBadRequest, e.multipartCall("mentor_igor", map[string]string{"title": "x", "url": "https://x.ru", "stageCode": "99"}, "", nil, nil))
	assert.Equal(t, http.StatusBadRequest, e.multipartCall("mentor_igor", map[string]string{"title": ""}, "", nil, nil))

	require.Equal(t, http.StatusCreated, e.multipartCall("mentor_igor", map[string]string{
		"title": "Как мы деплоим", "stageCode": "03", "section": "Деплой и стенды", "useInAi": "true", "notify": "true",
	}, "deploy.md", []byte("Деплой идет через пайплайн релизов.\n\nКанареечный выкат занимает полчаса."), &m))
	assert.Equal(t, "md", m.Kind)
	assert.Equal(t, "03", *m.StageCode)
	assert.Equal(t, "Деплой и стенды", *m.Section)
	assert.Contains(t, m.SizeLabel, "Б")

	// Новый раздел появился в справочнике стажера, AI-buddy находит текст, стажеру пришла карточка.
	var kb models.KnowledgeBase
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/kb", nil, &kb))
	titles := map[string]bool{}
	for _, s := range kb.Sections {
		titles[s.Title] = true
	}
	assert.True(t, titles["Деплой и стенды"])
	var found models.KBSearch
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/kb/search?q=канареечный+выкат", nil, &found))
	assert.NotEmpty(t, found.Results)
	var th models.Thread
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/me/mentor-thread", nil, &th))
	card := th.Messages[len(th.Messages)-1]
	require.Equal(t, "update", card.Kind)
	assert.Equal(t, "material", card.Update.Type)
	assert.False(t, card.Update.Acked)
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/me/mentor-thread/updates/"+card.ID+"/ack", nil, &th))
	assert.True(t, th.Messages[len(th.Messages)-1].Update.Acked)

	var link models.Material
	require.Equal(t, http.StatusCreated, e.multipartCall("lead_oleg", map[string]string{"title": "Гайд по ревью", "url": "https://example.com/review", "useInAi": "false"}, "", nil, &link))
	assert.Equal(t, "link", link.Kind)
	assert.Equal(t, "Материалы наставника", *link.Section)
	var pdf models.Material
	require.Equal(t, http.StatusCreated, e.multipartCall("mentor_igor", map[string]string{"title": "Слайды"}, "s.pdf", bytes.Repeat([]byte("x"), 3000), &pdf))
	assert.Equal(t, "pdf", pdf.Kind)
	assert.Equal(t, "3 КБ", pdf.SizeLabel)

	var mats []models.Material
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/materials", nil, &mats))
	assert.Equal(t, pdf.ID, mats[0].ID)
	assert.Equal(t, http.StatusNotFound, e.call("lead_nina", http.MethodDelete, "/api/materials/"+link.ID, nil, nil), "чужая команда")
	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodDelete, "/api/materials/"+link.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodDelete, "/api/materials/"+link.ID, nil, nil))
}

func TestMentorMeetingsAndStats(t *testing.T) {
	e := newMentorEnv(t)
	start := time.Now().Add(72 * time.Hour).Truncate(time.Minute)
	from, to := isoURL(time.Now().Add(-time.Hour)), isoURL(time.Now().Add(10*24*time.Hour))

	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodGet, "/api/mentor/meetings?from="+from+"&to="+to, nil, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodGet, "/api/mentor/meetings?from=x&to=y", nil, nil))

	var m models.MentorMeeting
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/mentor/meetings",
		map[string]any{"internId": annaID, "title": "Разбор MR", "startsAt": start, "durationMin": 30, "notify": true}, &m))
	assert.Equal(t, "review", m.Kind)
	assert.Equal(t, "Мария", m.WithName)
	var all models.MentorMeeting
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/mentor/meetings",
		map[string]any{"internId": "all", "title": "Синк стажеров", "startsAt": start.Add(2 * time.Hour), "durationMin": 60, "notify": false}, &all))
	assert.Equal(t, "team", all.Kind)
	assert.Nil(t, all.InternID)
	assert.Equal(t, "все стажеры", all.WithName)

	for name, body := range map[string]map[string]any{
		"прошлое":      {"internId": annaID, "title": "x", "startsAt": time.Now().Add(-48 * time.Hour), "durationMin": 30},
		"длительность": {"internId": annaID, "title": "x", "startsAt": start, "durationMin": 5},
		"без темы":     {"internId": annaID, "title": " ", "startsAt": start, "durationMin": 30},
		"чужой":        {"internId": pavelID, "title": "x", "startsAt": start, "durationMin": 30},
	} {
		assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/mentor/meetings", body, nil), name)
	}

	var list []models.MentorMeeting
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/mentor/meetings?from="+from+"&to="+to, nil, &list))
	ids := map[string]bool{}
	for _, x := range list {
		ids[x.ID] = true
	}
	assert.True(t, ids[m.ID] && ids[all.ID])

	moved := start.Add(24 * time.Hour)
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPatch, "/api/mentor/meetings/"+m.ID, map[string]any{"startsAt": moved}, &m))
	assert.True(t, m.StartsAt.Equal(moved))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPatch, "/api/mentor/meetings/"+m.ID, map[string]any{}, nil))
	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodDelete, "/api/mentor/meetings/"+m.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodDelete, "/api/mentor/meetings/"+m.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPatch, "/api/mentor/meetings/"+m.ID, map[string]any{"startsAt": moved}, nil))

	// Стажер получил карточки: новая, перенесена, отменена.
	var th models.Thread
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/me/mentor-thread", nil, &th))
	n := len(th.Messages)
	require.GreaterOrEqual(t, n, 3)
	assert.Contains(t, th.Messages[n-3].Update.Title, "Новая встреча")
	assert.Contains(t, th.Messages[n-2].Update.Title, "перенесена")
	assert.Contains(t, th.Messages[n-1].Update.Title, "отменена")

	// Статистика и метрики.
	e.say("intern_anna", map[string]any{"text": "как оформить отпуск"})
	var st models.MentorStats
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/mentor/stats", nil, &st))
	assert.Equal(t, 7, st.PeriodDays)
	assert.Positive(t, st.AnsweredByBuddy)
	assert.Equal(t, st.AnsweredByBuddy*12, st.SavedMinutes)
	assert.Equal(t, 44, st.MentorHoursPlan)
	var mt models.MentorMetrics
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodGet, "/api/mentor/metrics", nil, &mt))
	assert.Equal(t, 12, mt.MinutesPerQuestion)
	assert.NotNil(t, mt.AccessHours, "в демо есть выданные доступы")
	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodGet, "/api/mentor/metrics", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodGet, "/api/mentor/stats", nil, nil))
}

func isoURL(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05Z") }

func TestMentorAutomations(t *testing.T) {
	e := newMentorEnv(t)
	var list []models.Automation
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/mentor/automations", nil, &list))
	require.Len(t, list, 4)
	for _, a := range list {
		assert.True(t, a.Enabled)
		assert.Equal(t, a.Kind, a.ID)
	}
	var a models.Automation
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPatch, "/api/mentor/automations/deadline_reminder", map[string]any{"enabled": false}, &a))
	assert.False(t, a.Enabled)
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/mentor/automations", nil, &list))
	assert.False(t, list[0].Enabled)
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPatch, "/api/mentor/automations/overdue_alert", map[string]any{}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPatch, "/api/mentor/automations/nope", map[string]any{"enabled": true}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodGet, "/api/mentor/automations", nil, nil))

	// Проход правил: у Дмитрия (старт в октябре) есть просроченные задачи – Игорю приходит сигнал,
	// повторный проход ничего не дублирует.
	n, err := e.svc.Mentor.RunAutomations(context.Background())
	require.NoError(t, err)
	assert.Positive(t, n)
	again, err := e.svc.Mentor.RunAutomations(context.Background())
	require.NoError(t, err)
	assert.Zero(t, again)
	var cnt int
	require.NoError(t, testPool(t).QueryRow(context.Background(),
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND text LIKE 'У Дмитрий просрочена%'`, igorID).Scan(&cnt))
	assert.Positive(t, cnt)
	_, _ = testPool(t).Exec(context.Background(), `DELETE FROM notifications WHERE actor_name = 'AI-buddy' AND created_at > now() - interval '5 minutes' AND kind IN ('tasks','requests','goals')`)
}

// Путь на всю стажировку: период этапа, этапы 05–07, действие join_chats закрывает задачу сразу.
func TestFullPathAndJoinChats(t *testing.T) {
	e := newMentorEnv(t)
	v := e.path("intern_liza")
	require.Len(t, v.Stages, 7)
	require.NotNil(t, v.Stages[0].Period)
	assert.Equal(t, "неделя 1", *v.Stages[0].Period)
	assert.Equal(t, "Итоги", v.Stages[6].Title)
	var chats, ask models.TaskView
	for _, s := range v.Stages {
		for _, task := range s.Tasks {
			switch task.Title {
			case "Добавиться в чаты и каналы команды":
				chats = task
			case "Пройти материалы по защите данных":
				ask = task
			}
		}
	}
	require.NotNil(t, chats.Action)
	assert.Equal(t, "join_chats", chats.Action.Kind)
	assert.Contains(t, string(ask.Action.Payload), `"prompt"`)

	msgs := e.say("intern_liza", map[string]any{"text": "Добавь меня в чаты", "action": chats.Action, "taskId": chats.ID})
	card := confirmOf(t, msgs)
	for _, m := range msgs {
		if m.Confirm != nil {
			assert.Equal(t, "Добавить в чаты и каналы?", m.Text)
		}
	}
	f := fieldsMap(card.Fields)
	assert.NotEmpty(t, f["Чаты"])
	assert.NotEmpty(t, f["Закрытые"])
	var r models.ChatReply
	require.Equal(t, http.StatusOK, e.call("intern_liza", http.MethodPost, "/api/chat/actions/"+card.ActionID+"/confirm", nil, &r))
	assert.Equal(t, "done", r.Messages[0].Result.Status)
	assert.Contains(t, r.Messages[0].Text, "Готово")
	for _, s := range e.path("intern_liza").Stages {
		for _, task := range s.Tasks {
			if task.ID == chats.ID {
				assert.Equal(t, "done", task.Status)
			}
		}
	}
	pool := testPool(t)
	_, err := pool.Exec(context.Background(), `DELETE FROM intern_task_progress WHERE intern_id = $1 AND task_id = $2`, lizaID, chats.ID)
	require.NoError(t, err)

	// ask_buddy с prompt уходит помощнику готовым текстом.
	out := e.say("intern_liza", map[string]any{"text": "Пройти материалы", "action": ask.Action, "taskId": ask.ID})
	assert.NotEmpty(t, out[0].Text)

	// Период этапа задается в консоли.
	var st models.StageTemplate
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/console/stages",
		map[string]any{"title": "test этап", "period": "месяц 3"}, &st))
	assert.Equal(t, "месяц 3", st.Period)
	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodDelete, "/api/console/stages/"+st.ID, nil, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/console/stages",
		map[string]any{"title": "x", "period": string(make([]rune, 41))}, nil))
}

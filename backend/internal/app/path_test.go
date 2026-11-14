package app

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/database"
	"ai-buddy/backend/internal/models"
)

// Интеграционные тесты страницы «Мой путь» и консоли наставника на настоящей базе.
// Тесты возвращают данные в исходное состояние, чтобы их можно было гонять повторно.

const (
	annaID   = "aaaaaaaa-0000-4000-8000-000000000001"
	pavelID  = "aaaaaaaa-0000-4000-8000-000000000004"
	gitlabID = "b7000000-0000-4000-8000-000000000204" // Backend, этап 02
	envTask  = "b7000000-0000-4000-8000-000000000301" // Backend, этап 03, изначально todo
	stage02  = "b5000000-0000-4000-8000-000000000002"
)

type env struct {
	t    *testing.T
	base string
	tok  map[string]string
}

func newEnv(t *testing.T) *env {
	srv := newServer(t, true)
	e := &env{t: t, base: srv.URL, tok: map[string]string{}}
	for _, u := range []string{"intern_maria", "intern_pavel", "intern_dima", "intern_liza", "mentor_igor", "lead_artemiy", "lead_nina"} {
		code, token, _ := login(t, srv.URL, u, "demo123")
		require.Equal(t, http.StatusOK, code)
		e.tok[u] = token
	}
	return e
}

// call выполняет запрос и (если out != nil) читает JSON-ответ.
func (e *env) call(user, method, path string, body any, out any) int {
	e.t.Helper()
	resp := do(e.t, method, e.base+path, e.tok[user], body)
	if out != nil {
		b, _ := io.ReadAll(resp.Body)
		if len(b) > 0 {
			require.NoError(e.t, json.Unmarshal(b, out), string(b))
		}
	}
	return resp.StatusCode
}

func (e *env) path(user string) models.PathView {
	var v models.PathView
	require.Equal(e.t, http.StatusOK, e.call(user, http.MethodGet, "/api/path", nil, &v))
	return v
}

func TestMyPathForAnna(t *testing.T) {
	e := newEnv(t)
	v := e.path("intern_maria")

	assert.Equal(t, "Мария", v.Intern.Name)
	require.NotNil(t, v.Intern.Mentor)
	assert.Equal(t, "Игорь", v.Intern.Mentor.Name)
	assert.Equal(t, 24, v.Week.Total)
	assert.NotEmpty(t, v.Week.Milestones)
	require.GreaterOrEqual(t, len(v.Stages), 4)
	assert.Equal(t, "Доступы", v.Stages[1].Title)
	assert.Equal(t, models.StageCurrent, v.Stages[1].Status)
	require.NotNil(t, v.NextStep)
	assert.Equal(t, gitlabID, v.NextStep.TaskID)
	require.NotNil(t, v.NextStep.Action)
	assert.Equal(t, "access_request", v.NextStep.Action.Kind)
	assert.NotEmpty(t, v.Buddy.Text)
}

func TestPathPermissions(t *testing.T) {
	e := newEnv(t)
	var list []models.InternSummary

	// Павел (другая команда) не видит Марию и консоль.
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodGet, "/api/interns/"+annaID+"/path", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_pavel", http.MethodGet, "/api/interns", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_pavel", http.MethodGet, "/api/console/templates", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodPost, "/api/path/tasks/"+gitlabID+"/complete", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_pavel", http.MethodPut, "/api/interns/"+pavelID+"/tasks/"+gitlabID,
		map[string]string{"status": "done"}, nil))

	// Павел видит свой путь команды Frontend.
	pv := e.path("intern_pavel")
	assert.Equal(t, "Frontend", pv.Intern.TeamName)
	assert.Nil(t, pv.Intern.Mentor)

	// Игорь видит своих стажеров (Мария, Дмитрий, Лиза), Павла – нет.
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/interns", nil, &list))
	require.Len(t, list, 3)
	byLogin := map[string]models.InternSummary{}
	for _, it := range list {
		byLogin[it.Login] = it
	}
	assert.Equal(t, "02", byLogin["intern_maria"].StageCode)
	assert.Equal(t, "04", byLogin["intern_dima"].StageCode)
	assert.Equal(t, 6, byLogin["intern_dima"].Week)
	assert.Equal(t, "01", byLogin["intern_liza"].StageCode)
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodGet, "/api/interns/"+pavelID+"/path", nil, nil))
	assert.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/interns/"+annaID+"/path", nil, nil))

	// Наставник не стажер: своего «пути» у него нет.
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodGet, "/api/path", nil, nil))
	// Стажер не может «завершить» задачу как наставник.
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodPost, "/api/path/tasks/"+envTask+"/complete", nil, nil))

	// Артемий видит стажеров своей команды.
	require.Equal(t, http.StatusOK, e.call("lead_artemiy", http.MethodGet, "/api/interns", nil, &list))
	assert.Len(t, list, 3)

	// Без токена – 401.
	assert.Equal(t, http.StatusUnauthorized, e.call("nobody", http.MethodGet, "/api/path", nil, nil))
}

func TestInternCompletesTaskAndMentorChangesStatus(t *testing.T) {
	e := newEnv(t)
	before := e.path("intern_maria").Progress
	t.Cleanup(func() {
		// вернуть задачу в «впереди»
		e.call("mentor_igor", http.MethodPut, "/api/interns/"+annaID+"/tasks/"+envTask, map[string]string{"status": "todo"}, nil)
	})

	var v models.PathView
	require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodPost, "/api/path/tasks/"+envTask+"/complete", nil, &v))
	assert.Equal(t, before.Done+1, v.Progress.Done)

	note := "  Проверено на встрече  "
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPut, "/api/interns/"+annaID+"/tasks/"+envTask,
		map[string]any{"status": "waiting", "note": note}, &v))
	assert.Equal(t, before.InProgress+1, v.Progress.InProgress)
	assert.Equal(t, "Проверено на встрече", v.Stages[2].Tasks[0].Note)

	// Пустая пометка = вернуть текст из шаблона.
	empty := " "
	require.Equal(t, http.StatusOK, e.call("lead_artemiy", http.MethodPut, "/api/interns/"+annaID+"/tasks/"+envTask,
		map[string]any{"status": "in_progress", "note": empty}, &v))
	assert.Equal(t, "Репозиторий, Docker, локальный запуск", v.Stages[2].Tasks[0].Note)

	// Ошибки ввода.
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPut, "/api/interns/"+annaID+"/tasks/"+envTask,
		map[string]string{"status": "almost"}, nil))
	long := string(make([]byte, 201))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPut, "/api/interns/"+annaID+"/tasks/"+envTask,
		map[string]any{"status": "done", "note": long + "x"}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPut, "/api/interns/"+annaID+"/tasks/"+envTask,
		"not an object", nil))
	// Задача команды Frontend Марии не принадлежит.
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPut, "/api/interns/"+annaID+"/tasks/b7000000-0000-4000-8000-000000001201",
		map[string]string{"status": "done"}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodPost, "/api/path/tasks/not-a-task/complete", nil, nil))
}

func TestMeetings(t *testing.T) {
	e := newEnv(t)
	start := time.Now().Add(time.Hour).Truncate(time.Second)

	var m models.Meeting
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/interns/"+annaID+"/meetings",
		map[string]any{"startsAt": start}, &m))
	assert.Equal(t, "Встреча 1:1", m.Title, "название по умолчанию")
	assert.Equal(t, "Игорь", m.HostName)

	v := e.path("intern_maria")
	require.NotNil(t, v.NextMeeting)
	assert.Equal(t, m.ID, v.NextMeeting.ID, "новая встреча ближайшая")

	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/interns/"+annaID+"/meetings",
		map[string]any{"title": "x"}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/interns/"+annaID+"/meetings",
		map[string]any{"startsAt": time.Now().Add(-48 * time.Hour)}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/interns/"+annaID+"/meetings",
		map[string]any{"title": string(make([]byte, 121)), "startsAt": start}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_maria", http.MethodPost, "/api/interns/"+annaID+"/meetings",
		map[string]any{"startsAt": start}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPost, "/api/interns/"+pavelID+"/meetings",
		map[string]any{"startsAt": start}, nil))

	// убрать за собой
	pool := testPool(t)
	_, err := pool.Exec(t.Context(), `DELETE FROM meetings WHERE id::text = $1`, m.ID)
	require.NoError(t, err)
}

func TestConsoleTemplatesLifecycle(t *testing.T) {
	e := newEnv(t)
	var kinds []map[string]string
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/console/action-kinds", nil, &kinds))
	assert.Len(t, kinds, 4)

	var tpl []models.StageTemplate
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/console/templates", nil, &tpl))
	stagesBefore := len(tpl)

	// Создать этап.
	var st models.StageTemplate
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/console/stages",
		map[string]any{"title": "Итоги", "color": "#FFFFFF", "suggestions": []string{"Как готовиться к отзыву?"}}, &st))
	t.Cleanup(func() { e.call("lead_artemiy", http.MethodDelete, "/api/console/stages/"+st.ID, nil, nil) })
	assert.Equal(t, stagesBefore+1, st.Position)

	// Изменить этап.
	require.Equal(t, http.StatusOK, e.call("lead_artemiy", http.MethodPut, "/api/console/stages/"+st.ID,
		map[string]any{"title": "Итоги стажировки", "color": "#F7C1C5", "position": st.Position}, &st))
	assert.Equal(t, "Итоги стажировки", st.Title)
	assert.Equal(t, http.StatusBadRequest, e.call("lead_artemiy", http.MethodPut, "/api/console/stages/"+st.ID,
		map[string]any{"title": "x", "position": 0}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("lead_artemiy", http.MethodPut, "/api/console/stages/"+st.ID,
		map[string]any{"title": ""}, nil))

	// Задача с действием AI-buddy.
	var task models.TaskTemplate
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/tasks",
		map[string]any{"title": "Оформить отгул", "dueDay": 80, "action": map[string]any{"kind": "leave_request", "payload": map[string]string{"type": "day_off"}}}, &task))
	require.NotNil(t, task.Action)
	assert.JSONEq(t, `{"type":"day_off"}`, string(task.Action.Payload))

	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPut, "/api/console/tasks/"+task.ID,
		map[string]any{"title": "Оформить отгул на день отзыва", "position": 1}, &task))
	assert.Nil(t, task.Action, "действие можно убрать")
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPut, "/api/console/tasks/"+task.ID,
		map[string]any{"title": "x", "position": 0}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPut, "/api/console/tasks/"+task.ID,
		map[string]any{"title": ""}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/tasks",
		map[string]any{"title": "x", "action": map[string]any{"kind": "rm"}}, nil))

	// Заявка на доступ с проектом; старое поле group превращается в проект.
	var gl models.TaskTemplate
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/tasks",
		map[string]any{"title": "GitLab", "action": map[string]any{"kind": "access_request",
			"payload": map[string]string{"service": "gitlab", "role": "developer", "group": "backend"}}}, &gl))
	assert.JSONEq(t, `{"service":"gitlab","role":"developer","project":"backend/auth-service"}`, string(gl.Action.Payload))
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPut, "/api/console/tasks/"+gl.ID,
		map[string]any{"title": "Стенд", "position": 2, "action": map[string]any{"kind": "access_request",
			"payload": map[string]string{"service": "test_stand", "role": "read", "project": "auth-staging"}}}, &gl))
	assert.JSONEq(t, `{"service":"test_stand","role":"read","project":"auth-staging"}`, string(gl.Action.Payload))
	for name, payload := range map[string]map[string]string{
		"unknown service":       {"service": "jira", "role": "read"},
		"role of other service": {"service": "wiki", "role": "developer"},
		"project of other srv":  {"service": "gitlab", "role": "developer", "project": "auth-staging"},
	} {
		assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPut, "/api/console/tasks/"+gl.ID,
			map[string]any{"title": "x", "position": 2, "action": map[string]any{"kind": "access_request", "payload": payload}}, nil), name)
	}
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/tasks",
		map[string]any{"title": "x", "action": map[string]any{"kind": "open_article", "payload": map[string]string{"articleId": "not-a-uuid"}}}, nil))
	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodDelete, "/api/console/tasks/"+gl.ID, nil, nil))

	// Подсказки этапа хранятся отдельной таблицей и возвращаются по порядку.
	var st2 models.StageTemplate
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPut, "/api/console/stages/"+st.ID,
		map[string]any{"title": "Итоги стажировки", "color": "#F7C1C5", "position": st.Position, "suggestions": []string{"Б", "А"}}, &st2))
	assert.Equal(t, []string{"Б", "А"}, st2.Suggestions)

	// Карточка, связанная с задачей: тег у стажера считается сам.
	var res models.ResourceTemplate
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/resources",
		map[string]any{"kind": "access", "name": "Отгул", "taskId": task.ID}, &res))
	assert.Equal(t, "none", res.TagKind)
	// Карточка «человек» ссылается на сотрудника команды по имени; чужого имени нет – 400.
	var person models.ResourceTemplate
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/resources",
		map[string]any{"kind": "person", "name": "Артемий", "note": "руководитель"}, &person))
	assert.Equal(t, "Артемий", person.Name)
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/resources",
		map[string]any{"kind": "person", "name": "Нина"}, nil), "Нина из другой команды")
	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodDelete, "/api/console/resources/"+person.ID, nil, nil))
	var plain models.ResourceTemplate
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/resources",
		map[string]any{"kind": "article", "name": "Памятка", "tag": "читать", "tagKind": "none"}, &plain))

	for name, body := range map[string]map[string]any{
		"bad kind":     {"kind": "video", "name": "x"},
		"bad tag kind": {"kind": "tool", "name": "x", "tagKind": "red"},
		"no name":      {"kind": "tool", "name": ""},
		"long tag":     {"kind": "tool", "name": "x", "tag": string(make([]byte, 41))},
		"foreign task": {"kind": "access", "name": "x", "taskId": "b7000000-0000-4000-8000-000000001201"},
		"missing task": {"kind": "access", "name": "x", "taskId": "00000000-0000-4000-8000-000000000000"},
	} {
		assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/resources", body, nil), name)
	}

	// Мария видит новый этап с задачей и карточкой.
	v := e.path("intern_maria")
	last := v.Stages[len(v.Stages)-1]
	assert.Equal(t, "Итоги стажировки", last.Title)
	require.Len(t, last.Tasks, 1)
	require.Len(t, last.Resources, 2)
	assert.Equal(t, "не запрошен", last.Resources[0].Tag)

	// Чужие и запрещенные операции.
	assert.Equal(t, http.StatusForbidden, e.call("intern_maria", http.MethodPost, "/api/console/stages", map[string]any{"title": "x"}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_maria", http.MethodPut, "/api/console/stages/"+st.ID, map[string]any{"title": "x", "position": 1}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_maria", http.MethodDelete, "/api/console/tasks/"+task.ID, nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_pavel", http.MethodDelete, "/api/console/resources/"+res.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPost, "/api/console/stages/b5000000-0000-4000-8000-000000000011/tasks",
		map[string]any{"title": "чужая команда"}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPut, "/api/console/tasks/b7000000-0000-4000-8000-000000001201",
		map[string]any{"title": "чужая", "position": 1}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPut, "/api/console/stages/missing", map[string]any{"title": "x", "position": 1}, nil))

	// Удаление.
	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodDelete, "/api/console/resources/"+plain.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodDelete, "/api/console/resources/"+plain.ID, nil, nil))
	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodDelete, "/api/console/tasks/"+task.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodDelete, "/api/console/tasks/"+task.ID, nil, nil))
	assert.Equal(t, http.StatusNoContent, e.call("mentor_igor", http.MethodDelete, "/api/console/stages/"+st.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodDelete, "/api/console/stages/"+st.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPost, "/api/console/stages/"+st.ID+"/resources",
		map[string]any{"kind": "tool", "name": "x"}, nil))

	// Создание этапа с ошибками.
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/console/stages", map[string]any{"title": "x", "color": "red"}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/console/stages", map[string]any{"title": "x", "extra": 1}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_maria", http.MethodGet, "/api/console/templates", nil, nil))
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := database.Connect(t.Context(), testConfig(t, true).DatabaseURL, 10*time.Second)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

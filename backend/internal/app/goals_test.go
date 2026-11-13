package app

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/models"
)

// Цели стажировки: просмотр, прогресс, редактирование, согласование и права.

const (
	dimaID      = "aaaaaaaa-0000-4000-8000-000000000005"
	lizaID      = "aaaaaaaa-0000-4000-8000-000000000006"
	annaGoal2   = "f3000000-0000-4000-8000-000000000002"
	dimaGoal2   = "f3000000-0000-4000-8000-000000000012"
	annaGoal1   = "f3000000-0000-4000-8000-000000000001"
	backendOKR1 = "f2000000-0000-4000-8000-000000000001"
)

func TestGoalsForAnna(t *testing.T) {
	e := newEnv(t)
	var v models.GoalsView
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/goals", nil, &v))
	require.Len(t, v.Goals, 3)
	require.NotNil(t, v.Agreed)
	assert.Equal(t, "Олег", v.Agreed.By.Name)
	assert.True(t, v.FromOKR)
	assert.Len(t, v.OKRs, 2)
	assert.False(t, v.CanEdit, "после согласования стажер цели не меняет")
	assert.False(t, v.CanAgree)
	assert.Equal(t, "Разобраться в сервисе авторизации", v.Goals[1].Title)
	// Процент – доля шагов: из четырех выполнен один (разбор архитектуры с наставником).
	assert.Equal(t, 25, v.Goals[1].ProgressPct)
	require.Len(t, v.Goals[1].Steps, 4)
	st := v.Goals[1].Steps
	assert.Equal(t, "task", st[0].Source)
	assert.Equal(t, "in_progress", st[0].Status, "заявка на стенд ждет согласования")
	require.NotNil(t, st[0].StageCode)
	assert.Equal(t, "02", *st[0].StageCode)
	assert.Equal(t, "done", st[1].Status)
	assert.Nil(t, st[1].TaskID)
	assert.Equal(t, "lead", st[3].Source)
	require.NotNil(t, v.Goals[1].LastUpdate)
	assert.Equal(t, "Анна", v.Goals[1].LastUpdate.Author.Name)

	// Наставник и стажер другой команды.
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodGet, "/api/goals", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodGet, "/api/interns/"+annaID+"/goals", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("lead_nina", http.MethodGet, "/api/interns/"+annaID+"/goals", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodGet, "/api/interns/"+dimaID+"/goals", nil, nil))
}

// Стажер больше не выставляет процент сам: он складывается из шагов. Наставнику и руководителю
// ручка оставлена для заметок к цели без шагов.
func TestGoalProgress(t *testing.T) {
	e := newEnv(t)
	pool := testPool(t)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE intern_goals SET progress_pct = 40 WHERE id = $1`, dimaGoal2)
		_, _ = pool.Exec(context.Background(), `DELETE FROM goal_updates WHERE goal_id = $1 AND note LIKE 'test:%'`, dimaGoal2)
		_, _ = pool.Exec(context.Background(), `DELETE FROM notifications WHERE kind = 'goals' AND text LIKE '%Дмитрий%'`)
	})

	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodPost, "/api/goals/"+annaGoal2+"/progress",
		map[string]any{"progressPct": 100}, nil), "стажер не ставит себе 100%")

	var g models.Goal
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodPost, "/api/goals/"+dimaGoal2+"/progress",
		map[string]any{"progressPct": 45, "note": "test: покрытие выросло"}, &g))
	assert.Equal(t, 45, g.ProgressPct)
	require.NotNil(t, g.LastUpdate)
	assert.Equal(t, "Олег", g.LastUpdate.Author.Name)
	assert.Empty(t, g.Steps)

	// Наставник получил уведомление об отметке руководителя.
	var n int
	require.NoError(t, pool.QueryRow(t.Context(),
		`SELECT count(*) FROM notifications WHERE user_id = 'aaaaaaaa-0000-4000-8000-000000000002' AND kind = 'goals' AND text LIKE 'Олег%Дмитрий%45%'`).Scan(&n))
	assert.Equal(t, 1, n)

	for name, body := range map[string]map[string]any{
		"no pct":    {"note": "x"},
		"too big":   {"progressPct": 101},
		"negative":  {"progressPct": -1},
		"long note": {"progressPct": 1, "note": string(make([]rune, 501))},
	} {
		assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/goals/"+dimaGoal2+"/progress", body, nil), name)
	}
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodPost, "/api/goals/"+dimaGoal2+"/progress",
		map[string]any{"progressPct": 50}, nil), "чужая цель скрыта")
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPost, "/api/goals/missing/progress",
		map[string]any{"progressPct": 50}, nil))
}

// Шаги цели: шаг наставника подтверждает наставник или руководитель, шаг-задача закрывается задачей пути.
func TestGoalSteps(t *testing.T) {
	e := newEnv(t)
	pool := testPool(t)
	const gitlabTask = "b7000000-0000-4000-8000-000000000204"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE goal_steps SET done_at = NULL, done_by = NULL WHERE goal_id = $1`, annaGoal1)
		_, _ = pool.Exec(context.Background(), `UPDATE intern_task_progress SET status = 'in_progress' WHERE intern_id = $1 AND task_id = $2`, annaID, gitlabTask)
	})
	var v models.GoalsView
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/goals", nil, &v))
	goal := v.Goals[0]
	require.Len(t, goal.Steps, 5)
	assert.Equal(t, 0, goal.ProgressPct)
	taskStep, mentorStep := goal.Steps[0], goal.Steps[3]
	require.NotNil(t, taskStep.TaskID)
	assert.Equal(t, gitlabTask, *taskStep.TaskID)

	url := func(stepID string) string { return "/api/goals/" + annaGoal1 + "/steps/" + stepID }
	var g models.Goal
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPost, url(mentorStep.ID), map[string]any{"done": true}, &g))
	assert.Equal(t, 20, g.ProgressPct)
	assert.Equal(t, "done", g.Steps[3].Status)

	// Задача пути закрыта – процент вырос сам.
	_, err := pool.Exec(t.Context(), `UPDATE intern_task_progress SET status = 'done' WHERE intern_id = $1 AND task_id = $2`, annaID, gitlabTask)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/goals", nil, &v))
	assert.Equal(t, 40, v.Goals[0].ProgressPct)
	assert.Equal(t, "done", v.Goals[0].Steps[0].Status)

	// Снять отметку может руководитель.
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodPost, url(mentorStep.ID), map[string]any{"done": false}, &g))
	assert.Equal(t, 20, g.ProgressPct)

	assert.Equal(t, http.StatusForbidden, e.call("intern_anna", http.MethodPost, url(mentorStep.ID), map[string]any{"done": true}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, url(taskStep.ID), map[string]any{"done": true}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, url(mentorStep.ID), map[string]any{}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodPost, url("missing"), map[string]any{"done": true}, nil))
	assert.Equal(t, http.StatusNotFound, e.call("lead_nina", http.MethodPost, url(mentorStep.ID), map[string]any{"done": true}, nil))
}

func TestGoalsLifecycleForLiza(t *testing.T) {
	e := newEnv(t)
	pool := testPool(t)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM intern_goals WHERE intern_id = $1`, lizaID)
		_, _ = pool.Exec(context.Background(), `UPDATE intern_profiles SET goals_agreed_by = NULL, goals_agreed_at = NULL WHERE user_id = $1`, lizaID)
	})

	var v models.GoalsView
	require.Equal(t, http.StatusOK, e.call("intern_liza", http.MethodGet, "/api/goals", nil, &v))
	assert.Empty(t, v.Goals)
	assert.Nil(t, v.Agreed)
	assert.True(t, v.CanEdit, "до согласования стажер вносит цели сама")

	// Согласовать пустой список нельзя.
	assert.Equal(t, http.StatusBadRequest, e.call("lead_oleg", http.MethodPost, "/api/interns/"+lizaID+"/goals/agree", nil, nil))

	// Лиза вносит цель после встречи с Олегом.
	var g models.Goal
	require.Equal(t, http.StatusCreated, e.call("intern_liza", http.MethodPost, "/api/interns/"+lizaID+"/goals",
		map[string]any{"title": "Пройти путь задачи до релиза", "criterion": "первый merge в main до 4 недели", "dueWeek": 4, "okrId": backendOKR1}, &g))
	assert.Equal(t, 1, g.Position)
	require.NotNil(t, g.OKR)
	require.NotNil(t, g.CreatedBy)
	assert.Equal(t, "Лиза", g.CreatedBy.Name)

	// Наставник добавляет вторую, правит и убирает OKR.
	var g2 models.Goal
	require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/interns/"+lizaID+"/goals",
		map[string]any{"title": "Провести ревью коллег", "criterion": "5 ревью к 12 неделе"}, &g2))
	assert.Equal(t, 2, g2.Position)
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPatch, "/api/goals/"+g.ID,
		map[string]any{"criterion": "первый merge в main до 5 недели", "okrId": "", "dueWeek": 0}, &g))
	assert.Equal(t, "первый merge в main до 5 недели", g.Criterion)
	assert.Equal(t, "Пройти путь задачи до релиза", g.Title, "незатронутые поля не меняются")
	assert.Nil(t, g.OKR)
	assert.Nil(t, g.DueWeek)

	for name, body := range map[string]map[string]any{
		"no title":     {"criterion": "x"},
		"no criterion": {"title": "x"},
		"bad week":     {"title": "x", "criterion": "y", "dueWeek": 60},
		"foreign okr":  {"title": "x", "criterion": "y", "okrId": "00000000-0000-4000-8000-000000000000"},
		"extra field":  {"title": "x", "criterion": "y", "owner": "me"},
	} {
		assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/interns/"+lizaID+"/goals", body, nil), name)
	}
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPatch, "/api/goals/"+g.ID, map[string]any{"title": " "}, nil))

	// Согласует только руководитель команды.
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodPost, "/api/interns/"+lizaID+"/goals/agree", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_liza", http.MethodPost, "/api/interns/"+lizaID+"/goals/agree", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("lead_nina", http.MethodPost, "/api/interns/"+lizaID+"/goals/agree", nil, nil))
	require.Equal(t, http.StatusOK, e.call("lead_oleg", http.MethodPost, "/api/interns/"+lizaID+"/goals/agree", nil, &v))
	require.NotNil(t, v.Agreed)
	assert.Equal(t, "Олег", v.Agreed.By.Name)
	assert.False(t, v.CanAgree, "согласованные цели повторно не согласуют")

	// После согласования стажер не меняет и не удаляет цели; процент сам тоже не ставит.
	assert.Equal(t, http.StatusForbidden, e.call("intern_liza", http.MethodPatch, "/api/goals/"+g.ID, map[string]any{"title": "Другое"}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_liza", http.MethodDelete, "/api/goals/"+g.ID, nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_liza", http.MethodPost, "/api/interns/"+lizaID+"/goals",
		map[string]any{"title": "x", "criterion": "y"}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_liza", http.MethodPost, "/api/goals/"+g.ID+"/progress", map[string]any{"progressPct": 25}, nil))

	// Удаление.
	assert.Equal(t, http.StatusNoContent, e.call("lead_oleg", http.MethodDelete, "/api/goals/"+g2.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("lead_oleg", http.MethodDelete, "/api/goals/"+g2.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodDelete, "/api/goals/"+g.ID, nil, nil))
}

func TestGoalsLimit(t *testing.T) {
	e := newEnv(t)
	pool := testPool(t)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM intern_goals WHERE intern_id = $1 AND title LIKE 'limit %'`, dimaID)
	})
	// У Дмитрия 3 цели, можно добавить еще 3.
	for i := 0; i < 3; i++ {
		require.Equal(t, http.StatusCreated, e.call("mentor_igor", http.MethodPost, "/api/interns/"+dimaID+"/goals",
			map[string]any{"title": "limit " + string(rune('a'+i)), "criterion": "c"}, nil))
	}
	assert.Equal(t, http.StatusBadRequest, e.call("mentor_igor", http.MethodPost, "/api/interns/"+dimaID+"/goals",
		map[string]any{"title": "limit z", "criterion": "c"}, nil))
}

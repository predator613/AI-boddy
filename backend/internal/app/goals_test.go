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
	assert.Equal(t, 10, v.Goals[1].ProgressPct)
	require.NotNil(t, v.Goals[1].LastUpdate)
	assert.Equal(t, "Анна", v.Goals[1].LastUpdate.Author.Name)

	// Наставник и стажер другой команды.
	assert.Equal(t, http.StatusNotFound, e.call("mentor_igor", http.MethodGet, "/api/goals", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodGet, "/api/interns/"+annaID+"/goals", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("lead_nina", http.MethodGet, "/api/interns/"+annaID+"/goals", nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodGet, "/api/interns/"+dimaID+"/goals", nil, nil))
}

func TestGoalProgress(t *testing.T) {
	e := newEnv(t)
	pool := testPool(t)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE intern_goals SET progress_pct = 10 WHERE id = $1`, annaGoal2)
		_, _ = pool.Exec(context.Background(), `DELETE FROM goal_updates WHERE goal_id = $1 AND note LIKE 'test:%'`, annaGoal2)
		_, _ = pool.Exec(context.Background(), `DELETE FROM notifications WHERE kind = 'goals' AND text LIKE 'Анна%'`)
	})

	var g models.Goal
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodPost, "/api/goals/"+annaGoal2+"/progress",
		map[string]any{"progressPct": 35, "note": "test: разобрала схему токенов"}, &g))
	assert.Equal(t, 35, g.ProgressPct)
	require.NotNil(t, g.LastUpdate)
	assert.Equal(t, "test: разобрала схему токенов", g.LastUpdate.Note)

	// Наставник видит обновление и получил уведомление.
	var v models.GoalsView
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/interns/"+annaID+"/goals", nil, &v))
	assert.Equal(t, 35, v.Goals[1].ProgressPct)
	assert.True(t, v.CanEdit)
	assert.False(t, v.CanAgree, "согласует только руководитель")
	var n int
	require.NoError(t, pool.QueryRow(t.Context(),
		`SELECT count(*) FROM notifications WHERE user_id = 'aaaaaaaa-0000-4000-8000-000000000002' AND kind = 'goals' AND text LIKE 'Анна%35%'`).Scan(&n))
	assert.Equal(t, 1, n)

	// Наставник тоже может отметить прогресс, тогда себе уведомление не шлет.
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodPost, "/api/goals/"+annaGoal2+"/progress",
		map[string]any{"progressPct": 40, "note": "test: с 1:1"}, &g))

	for name, body := range map[string]map[string]any{
		"no pct":    {"note": "x"},
		"too big":   {"progressPct": 101},
		"negative":  {"progressPct": -1},
		"long note": {"progressPct": 1, "note": string(make([]rune, 501))},
	} {
		assert.Equal(t, http.StatusBadRequest, e.call("intern_anna", http.MethodPost, "/api/goals/"+annaGoal2+"/progress", body, nil), name)
	}
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodPost, "/api/goals/"+dimaGoal2+"/progress",
		map[string]any{"progressPct": 50}, nil), "чужая цель скрыта")
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodPost, "/api/goals/missing/progress",
		map[string]any{"progressPct": 50}, nil))
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
	assert.True(t, v.CanAgree)

	// После согласования стажер не меняет и не удаляет цели, но прогресс отмечает.
	assert.Equal(t, http.StatusForbidden, e.call("intern_liza", http.MethodPatch, "/api/goals/"+g.ID, map[string]any{"title": "Другое"}, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_liza", http.MethodDelete, "/api/goals/"+g.ID, nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("intern_liza", http.MethodPost, "/api/interns/"+lizaID+"/goals",
		map[string]any{"title": "x", "criterion": "y"}, nil))
	assert.Equal(t, http.StatusOK, e.call("intern_liza", http.MethodPost, "/api/goals/"+g.ID+"/progress", map[string]any{"progressPct": 25}, nil))

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

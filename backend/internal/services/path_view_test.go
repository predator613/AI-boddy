package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/models"
)

func ptr[T any](v T) *T { return &v }

// Шаблон из двух этапов, как кусок макета.
func sampleInput(now time.Time) pathInput {
	return pathInput{
		Intern:  models.User{ID: "anna", Name: "Анна", TeamName: "Backend", Role: models.RoleIntern},
		Mentor:  &models.PersonRef{ID: "igor", Name: "Игорь"},
		Profile: models.InternProfile{StartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), DurationWeeks: 24},
		Now:     now,
		Stages: []models.StageTemplate{
			{ID: "s1", Title: "Знакомство", Suggestions: []string{"q1"}, Tasks: []models.TaskTemplate{
				{ID: "t1", Title: "Встреча", Note: "шаблон"},
				{ID: "t2", Title: "Наставник"},
			}},
			{ID: "s2", Title: "Доступы", Suggestions: []string{"Запросить GitLab"}, Tasks: []models.TaskTemplate{
				{ID: "t3", Title: "Стенд", DueDay: ptr(5)},
				{ID: "t4", Title: "GitLab", DueDay: ptr(10), Action: &models.Action{Kind: "access_request", Payload: json.RawMessage(`{"service":"gitlab"}`)}},
				{ID: "t5", Title: "Вики"},
			}, Resources: []models.ResourceTemplate{
				{ID: "r1", Kind: "access", Name: "Стенд", TaskID: ptr("t3")},
				{ID: "r2", Kind: "access", Name: "GitLab", TaskID: ptr("t4")},
				{ID: "r3", Kind: "access", Name: "Вики", TaskID: ptr("t5")},
				{ID: "r4", Kind: "person", Name: "Олег", Tag: "Backend", TagKind: "none"},
			}},
		},
		Progress: map[string]models.TaskProgress{
			"t1": {Status: models.TaskDone, Note: ptr("Прошла в понедельник")},
			"t2": {Status: models.TaskDone},
			"t3": {Status: models.TaskWaiting},
			"t4": {Status: models.TaskInProgress},
			"t5": {Status: models.TaskDone},
		},
		Milestones: []models.Milestone{{Week: 1, Title: "Старт"}},
	}
}

func TestBuildPathLikeMockup(t *testing.T) {
	// 10 ноября 12:00 по Москве = 10-й день стажировки, 2-я неделя.
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	v := buildPath(sampleInput(now))

	assert.Equal(t, models.WeekInfo{Current: 2, Total: 24, Day: 10, StartDate: "2026-11-01",
		Milestones: []models.Milestone{{Week: 1, Title: "Старт"}}}, v.Week)
	assert.Equal(t, models.Progress{Done: 3, InProgress: 2, Todo: 0, Total: 5, Percent: 60}, v.Progress)

	require.Len(t, v.Stages, 2)
	assert.Equal(t, "01", v.Stages[0].Code)
	assert.Equal(t, models.StageDone, v.Stages[0].Status)
	assert.Equal(t, "Прошла в понедельник", v.Stages[0].Tasks[0].Note, "личная пометка заменяет шаблон")
	assert.Equal(t, models.StageCurrent, v.Stages[1].Status)

	require.NotNil(t, v.NextStep)
	assert.Equal(t, "t4", v.NextStep.TaskID)
	assert.Equal(t, "02", v.NextStep.StageCode)
	assert.Equal(t, "2026-11-10", *v.NextStep.DueDate)
	assert.Equal(t, 0, *v.NextStep.DueInDays)
	assert.True(t, v.Stages[1].Tasks[1].IsNext)

	tags := map[string]string{}
	for _, r := range v.Stages[1].Resources {
		tags[r.Name] = r.Tag + "/" + r.TagKind
	}
	assert.Equal(t, "на согласовании/wait", tags["Стенд"])
	assert.Equal(t, "не запрошен/none", tags["GitLab"])
	assert.Equal(t, "активен/ok", tags["Вики"])
	assert.Equal(t, "Backend/none", tags["Олег"], "карточка без задачи берет тег из шаблона")

	assert.Contains(t, v.Buddy.Text, "«Доступы»")
	assert.Contains(t, v.Buddy.Text, "Осталось 2 из 3")
	assert.Contains(t, v.Buddy.Text, "Могу помочь")
	assert.Equal(t, []string{"Запросить GitLab"}, v.Buddy.Suggestions)
}

func TestBuildPathNextIsFirstTodoAndNoAction(t *testing.T) {
	in := sampleInput(time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC))
	in.Progress = map[string]models.TaskProgress{"t1": {Status: models.TaskDone}}
	v := buildPath(in)

	assert.Equal(t, "t2", v.NextStep.TaskID)
	assert.Nil(t, v.NextStep.DueDate)
	assert.Equal(t, models.StageCurrent, v.Stages[0].Status)
	assert.Equal(t, models.StageTodo, v.Stages[1].Status)
	assert.Contains(t, v.Buddy.Text, "Следующий шаг – «Наставник».")
	assert.NotContains(t, v.Buddy.Text, "Могу помочь")
	assert.Equal(t, 20, v.Progress.Percent)
}

func TestBuildPathAllDone(t *testing.T) {
	in := sampleInput(time.Date(2027, 9, 1, 9, 0, 0, 0, time.UTC)) // далеко после конца стажировки
	for _, id := range []string{"t1", "t2", "t3", "t4", "t5"} {
		in.Progress[id] = models.TaskProgress{Status: models.TaskDone}
	}
	v := buildPath(in)
	assert.Nil(t, v.NextStep)
	assert.Equal(t, 24, v.Week.Current, "неделя не больше длительности стажировки")
	assert.Equal(t, 100, v.Progress.Percent)
	assert.Contains(t, v.Buddy.Text, "все этапы адаптации пройдены")
}

func TestBuildPathEmptyAndDefaults(t *testing.T) {
	v := buildPath(pathInput{
		Intern:  models.User{Name: "Павел"},
		Profile: models.InternProfile{StartDate: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)}, // еще не начал
		Now:     time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC),
	})
	assert.Equal(t, 1, v.Week.Day)
	assert.Equal(t, 24, v.Week.Total)
	assert.Empty(t, v.Stages)
	assert.NotNil(t, v.Week.Milestones)
	assert.Equal(t, 0, v.Progress.Percent)
	assert.Contains(t, v.Buddy.Text, "план адаптации еще не заполнен")
	assert.NotNil(t, v.Buddy.Suggestions)
}

func TestAccessRules(t *testing.T) {
	anna := &models.User{ID: "anna", Role: models.RoleIntern, TeamID: "backend", MentorID: ptr("igor")}
	pavel := &models.User{ID: "pavel", Role: models.RoleIntern, TeamID: "frontend"}
	igor := &models.User{ID: "igor", Role: models.RoleMentor, TeamID: "backend"}
	oleg := &models.User{ID: "oleg", Role: models.RoleLead, TeamID: "backend"}

	assert.True(t, CanViewIntern(anna, anna))
	assert.False(t, CanViewIntern(pavel, anna))
	assert.True(t, CanViewIntern(igor, anna))
	assert.False(t, CanViewIntern(igor, pavel))
	assert.True(t, CanViewIntern(oleg, anna))
	assert.False(t, CanViewIntern(oleg, pavel))
	assert.False(t, CanViewIntern(oleg, igor), "смотреть можно только стажеров")
	assert.False(t, CanViewIntern(&models.User{Role: "guest"}, anna))
	assert.False(t, CanViewIntern(nil, anna))

	assert.False(t, CanManageIntern(anna, anna), "стажер сам себе статусы не ставит")
	assert.True(t, CanManageIntern(igor, anna))

	assert.True(t, CanEditTemplates(igor, "backend"))
	assert.True(t, CanEditTemplates(oleg, "backend"))
	assert.False(t, CanEditTemplates(igor, "frontend"))
	assert.False(t, CanEditTemplates(anna, "backend"))
}

func TestCleanStage(t *testing.T) {
	st, err := cleanStage(StageInput{Title: "  Итоги ", Suggestions: []string{" a ", "", "b"}})
	require.NoError(t, err)
	assert.Equal(t, "Итоги", st.Title)
	assert.Equal(t, "#C9FFBF", st.Color)
	assert.Equal(t, []string{"a", "b"}, st.Suggestions)

	long := string(make([]rune, 81))
	for name, in := range map[string]StageInput{
		"empty title":      {Title: " "},
		"bad color":        {Title: "x", Color: "red"},
		"long tip":         {Title: "x", Tip: string(make([]rune, 301))},
		"many suggestions": {Title: "x", Suggestions: []string{"1", "2", "3", "4", "5", "6", "7"}},
		"long suggestion":  {Title: "x", Suggestions: []string{long + "x"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := cleanStage(in)
			var ve *ValidationError
			assert.ErrorAs(t, err, &ve)
		})
	}
}

func TestCleanTask(t *testing.T) {
	task, err := cleanTask(TaskInput{Title: "GitLab", Action: &models.Action{Kind: "access_request"}})
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(task.Action.Payload))

	for name, in := range map[string]TaskInput{
		"empty title":    {Title: ""},
		"long note":      {Title: "x", Note: string(make([]rune, 301))},
		"bad due day":    {Title: "x", DueDay: ptr(0)},
		"unknown action": {Title: "x", Action: &models.Action{Kind: "drop_db"}},
		"payload array":  {Title: "x", Action: &models.Action{Kind: "ask_buddy", Payload: json.RawMessage(`[1]`)}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := cleanTask(in)
			assert.Error(t, err)
		})
	}
}

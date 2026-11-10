package services

import (
	"fmt"
	"math"
	"time"

	"ai-buddy/backend/internal/models"
)

// Moscow – часовой пояс для «сегодня» и сроков. Фиксированный UTC+3, чтобы не зависеть от tzdata в образе.
var Moscow = time.FixedZone("MSK", 3*60*60)

// pathInput – все сырые данные, из которых собирается экран «Мой путь».
type pathInput struct {
	Intern     models.User
	Mentor     *models.PersonRef
	Profile    models.InternProfile
	Stages     []models.StageTemplate
	Progress   map[string]models.TaskProgress
	Milestones []models.Milestone
	Meeting    *models.Meeting
	Now        time.Time
}

// buildPath – чистая функция без базы: шаблон + прогресс → то, что показывает экран.
func buildPath(in pathInput) models.PathView {
	today := dateOnly(in.Now.In(Moscow))
	start := dateOnly(in.Profile.StartDate)
	day := int(today.Sub(start).Hours()/24) + 1
	if day < 1 {
		day = 1
	}
	total := in.Profile.DurationWeeks
	if total <= 0 {
		total = 24
	}
	week := (day-1)/7 + 1
	if week > total {
		week = total
	}

	view := models.PathView{
		Intern: models.PathIntern{ID: in.Intern.ID, Name: in.Intern.Name, TeamName: in.Intern.TeamName, Mentor: in.Mentor},
		Week: models.WeekInfo{
			Current: week, Total: total, Day: day,
			StartDate: start.Format("2006-01-02"), Milestones: nonNil(in.Milestones),
		},
		Stages:      make([]models.StageView, 0, len(in.Stages)),
		NextMeeting: in.Meeting,
	}

	var next *models.NextStep
	var firstTodo *models.NextStep
	currentFound := false

	for i, st := range in.Stages {
		code := fmt.Sprintf("%02d", i+1)
		sv := models.StageView{
			ID: st.ID, Code: code, Title: st.Title, Color: st.Color, Tip: st.Tip,
			Suggestions: nonNil(st.Suggestions), Tasks: make([]models.TaskView, 0, len(st.Tasks)),
			Resources: make([]models.ResourceView, 0, len(st.Resources)),
		}

		for _, t := range st.Tasks {
			status, note := models.TaskTodo, t.Note
			if p, ok := in.Progress[t.ID]; ok {
				status = p.Status
				if p.Note != nil && *p.Note != "" {
					note = *p.Note
				}
			}

			tv := models.TaskView{ID: t.ID, Title: t.Title, Note: note, Status: status, Action: t.Action}
			var dueIn *int
			if t.DueDay != nil {
				due := start.AddDate(0, 0, *t.DueDay-1)
				s := due.Format("2006-01-02")
				tv.DueDate = &s
				d := int(math.Round(due.Sub(today).Hours() / 24))
				dueIn = &d
			}

			switch status {
			case models.TaskDone:
				view.Progress.Done++
				sv.DoneCount++
			case models.TaskInProgress, models.TaskWaiting:
				view.Progress.InProgress++
			default:
				view.Progress.Todo++
			}

			step := &models.NextStep{TaskID: t.ID, StageID: st.ID, StageCode: code, Title: t.Title,
				Status: status, DueDate: tv.DueDate, DueInDays: dueIn, Action: t.Action}
			if status == models.TaskInProgress && next == nil {
				next = step
			}
			if status == models.TaskTodo && firstTodo == nil {
				firstTodo = step
			}
			sv.Tasks = append(sv.Tasks, tv)
		}
		sv.TotalCount = len(st.Tasks)

		switch {
		case sv.TotalCount > 0 && sv.DoneCount == sv.TotalCount:
			sv.Status = models.StageDone
		case !currentFound:
			sv.Status = models.StageCurrent
			currentFound = true
		default:
			sv.Status = models.StageTodo
		}

		for _, x := range st.Resources {
			rv := models.ResourceView{ID: x.ID, Kind: x.Kind, Name: x.Name, Note: x.Note, Tag: x.Tag, TagKind: x.TagKind}
			if x.TaskID != nil {
				status := models.TaskTodo
				if p, ok := in.Progress[*x.TaskID]; ok {
					status = p.Status
				}
				rv.Tag, rv.TagKind = accessTag(status)
			}
			sv.Resources = append(sv.Resources, rv)
		}
		view.Stages = append(view.Stages, sv)
	}

	view.Progress.Total = view.Progress.Done + view.Progress.InProgress + view.Progress.Todo
	if view.Progress.Total > 0 {
		view.Progress.Percent = int(math.Round(float64(view.Progress.Done) * 100 / float64(view.Progress.Total)))
	}

	if next == nil {
		next = firstTodo
	}
	view.NextStep = next
	for si := range view.Stages {
		for ti := range view.Stages[si].Tasks {
			if next != nil && view.Stages[si].Tasks[ti].ID == next.TaskID {
				view.Stages[si].Tasks[ti].IsNext = true
			}
		}
	}

	view.Buddy = buddyHint(in.Intern.Name, view)
	return view
}

// accessTag – подпись для карточки доступа по статусу связанной задачи.
func accessTag(status string) (string, string) {
	switch status {
	case models.TaskDone:
		return "активен", "ok"
	case models.TaskWaiting:
		return "на согласовании", "wait"
	}
	// todo и in_progress: стажер еще не отправил заявку
	return "не запрошен", "none"
}

// buddyHint – реплика маскота на главной. Шаблонная, без LLM: экран не ждет модель.
func buddyHint(name string, v models.PathView) models.BuddyHint {
	var cur *models.StageView
	for i := range v.Stages {
		if v.Stages[i].Status == models.StageCurrent {
			cur = &v.Stages[i]
			break
		}
	}
	if cur == nil {
		if len(v.Stages) == 0 {
			return models.BuddyHint{Text: name + ", план адаптации еще не заполнен. Пока можешь спросить меня о чем угодно.", Suggestions: []string{}}
		}
		return models.BuddyHint{
			Text:        name + ", все этапы адаптации пройдены! Если появятся вопросы о процессах, я рядом.",
			Suggestions: v.Stages[len(v.Stages)-1].Suggestions,
		}
	}
	left := cur.TotalCount - cur.DoneCount
	text := fmt.Sprintf("%s, ты на этапе «%s». Осталось %d из %d.", name, cur.Title, left, cur.TotalCount)
	if v.NextStep != nil {
		if v.NextStep.Action != nil {
			text += fmt.Sprintf(" Следующий шаг – «%s». Могу помочь сделать его прямо сейчас.", v.NextStep.Title)
		} else {
			text += fmt.Sprintf(" Следующий шаг – «%s».", v.NextStep.Title)
		}
	}
	return models.BuddyHint{Text: text, Suggestions: cur.Suggestions}
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

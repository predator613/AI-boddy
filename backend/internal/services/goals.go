package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// Цели стажировки. На 1-й неделе стажер с руководителем обсуждает цели, критерии успеха и OKR
// команды, на 2-й вносит их в трекер прогресса, дальше отмечает продвижение (по пятницам напоминает
// AI-buddy), а наставник видит обновления.
//
// Права:
//   смотреть и отмечать прогресс – тот, кто видит стажера (он сам, его наставник, руководитель команды);
//   добавлять, менять, удалять – наставник и руководитель; стажер – только пока цели не согласованы;
//   согласовать – только руководитель команды стажера.

// GoalsStore – что нужно сервису целей от базы.
type GoalsStore interface {
	List(ctx context.Context, internID string) ([]models.Goal, error)
	Get(ctx context.Context, id string) (*models.Goal, string, error)
	Agreement(ctx context.Context, internID string) (*models.GoalsAgreement, error)
	OKRs(ctx context.Context, teamID string) ([]models.TeamOKR, error)
	Count(ctx context.Context, internID string) (int, error)
	Create(ctx context.Context, internID, by string, f models.GoalFields) (*models.Goal, error)
	Update(ctx context.Context, id string, f models.GoalFields) (*models.Goal, error)
	Delete(ctx context.Context, id string) error
	SetProgress(ctx context.Context, goalID, authorID string, pct int, note string) (*models.Goal, error)
	Agree(ctx context.Context, internID, by string) error
	Step(ctx context.Context, goalID, stepID string) (*models.GoalStep, error)
	SetStepDone(ctx context.Context, goalID, stepID, by string, done bool) (*models.Goal, error)
}

// StepInput – подтверждение шага цели.
type StepInput struct {
	Done *bool `json:"done"`
}

// GoalInput – поля цели из формы. В PATCH отсутствующее поле не меняется;
// dueWeek = 0 и okrId = "" очищают значение.
type GoalInput struct {
	Title     *string `json:"title"`
	Criterion *string `json:"criterion"`
	DueWeek   *int    `json:"dueWeek"`
	OKRID     *string `json:"okrId"`
	// Competency – компетенция; пустая строка очищает.
	Competency *string `json:"competency"`
	// Steps – шаги цели. В PATCH список заменяется целиком, без поля шаги не меняются.
	Steps *[]models.GoalStepInput `json:"steps"`
}

// ProgressInput – отметка о прогрессе.
type ProgressInput struct {
	ProgressPct *int   `json:"progressPct"`
	Note        string `json:"note"`
}

// MaxGoals – сколько целей можно поставить на стажировку.
const MaxGoals = 6

// Goals – сервис целей стажировки.
type Goals struct {
	store GoalsStore
	users UserLookup
	feed  FeedWriter
	log   *slog.Logger
}

// WithFeed подключает ленту наставника: согласование целей и подтверждение шагов руководителем.
func (s *Goals) WithFeed(w FeedWriter, log *slog.Logger) *Goals {
	s.feed, s.log = w, log
	return s
}

// NewGoals создает сервис.
func NewGoals(store GoalsStore, users UserLookup) *Goals {
	return &Goals{store: store, users: users}
}

// View – цели стажера. internID == "" – свои (только для стажера).
func (s *Goals) View(ctx context.Context, viewerID, internID string) (*models.GoalsView, error) {
	viewer, intern, err := s.access(ctx, viewerID, internID)
	if err != nil {
		return nil, err
	}
	return s.build(ctx, viewer, intern)
}

// Create добавляет цель стажеру.
func (s *Goals) Create(ctx context.Context, viewerID, internID string, in GoalInput) (*models.Goal, error) {
	viewer, intern, err := s.access(ctx, viewerID, internID)
	if err != nil {
		return nil, err
	}
	if err := s.checkEdit(ctx, viewer, intern); err != nil {
		return nil, err
	}
	f, err := s.clean(ctx, intern, models.GoalFields{}, in)
	if err != nil {
		return nil, err
	}
	n, err := s.store.Count(ctx, intern.ID)
	if err != nil {
		return nil, err
	}
	if n >= MaxGoals {
		return nil, invalid("max 6 goals")
	}
	return storeResult(s.store.Create(ctx, intern.ID, viewer.ID, f))
}

// Update меняет цель (частично).
func (s *Goals) Update(ctx context.Context, viewerID, goalID string, in GoalInput) (*models.Goal, error) {
	viewer, intern, goal, err := s.goal(ctx, viewerID, goalID)
	if err != nil {
		return nil, err
	}
	if err := s.checkEdit(ctx, viewer, intern); err != nil {
		return nil, err
	}
	cur := models.GoalFields{Title: goal.Title, Criterion: goal.Criterion, DueWeek: goal.DueWeek, Competency: goal.Competency}
	if goal.OKR != nil {
		cur.OKRID = &goal.OKR.ID
	}
	f, err := s.clean(ctx, intern, cur, in)
	if err != nil {
		return nil, err
	}
	return storeResult(s.store.Update(ctx, goalID, f))
}

// Delete удаляет цель.
func (s *Goals) Delete(ctx context.Context, viewerID, goalID string) error {
	viewer, intern, _, err := s.goal(ctx, viewerID, goalID)
	if err != nil {
		return err
	}
	if err := s.checkEdit(ctx, viewer, intern); err != nil {
		return err
	}
	return mapNotFound(s.store.Delete(ctx, goalID))
}

// Progress – отметка о прогрессе по цели.
func (s *Goals) Progress(ctx context.Context, viewerID, goalID string, in ProgressInput) (*models.Goal, error) {
	viewer, intern, _, err := s.goal(ctx, viewerID, goalID)
	if err != nil {
		return nil, err
	}
	// Стажер не выставляет себе процент сам: он складывается из шагов (задачи пути и отметки
	// наставника). Наставнику и руководителю ручка оставлена для заметок к цели.
	if viewer.ID == intern.ID {
		return nil, ErrForbidden
	}
	if in.ProgressPct == nil || *in.ProgressPct < 0 || *in.ProgressPct > 100 {
		return nil, invalid("progressPct must be between 0 and 100")
	}
	note := strings.TrimSpace(in.Note)
	if len([]rune(note)) > 500 {
		return nil, invalid("note is too long (max 500)")
	}
	return storeResult(s.store.SetProgress(ctx, goalID, viewer.ID, *in.ProgressPct, note))
}

// Step – наставник стажера или руководитель отмечает шаг цели выполненным (или снимает отметку).
// Шаги-задачи закрываются задачей «Моего пути», отметить их здесь нельзя.
func (s *Goals) Step(ctx context.Context, viewerID, goalID, stepID string, in StepInput) (*models.Goal, error) {
	viewer, intern, _, err := s.goal(ctx, viewerID, goalID)
	if err != nil {
		return nil, err
	}
	if !CanManageIntern(viewer, intern) {
		return nil, ErrForbidden
	}
	if in.Done == nil {
		return nil, invalid("done is required")
	}
	st, err := s.store.Step(ctx, goalID, stepID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if st.Source == "task" {
		return nil, invalid("this step is done when the intern completes the task on the path")
	}
	g, err := storeResult(s.store.SetStepDone(ctx, goalID, stepID, viewer.ID, *in.Done))
	if err == nil && *in.Done && viewer.Role == models.RoleLead {
		pushFeed(ctx, s.feed, s.log, intern.ID, models.Notice{
			Kind: "goals", Link: "/interns/" + intern.ID,
			Text:    fmt.Sprintf("%s подтвердил шаг «%s» у %s", viewer.Name, st.Title, genitive(intern.Name)),
			ActorID: &viewer.ID, ActorKind: "lead", ActorName: viewer.Name,
		})
	}
	return g, err
}

// Agree – руководитель согласовал цели стажера.
func (s *Goals) Agree(ctx context.Context, viewerID, internID string) (*models.GoalsView, error) {
	viewer, intern, err := s.access(ctx, viewerID, internID)
	if err != nil {
		return nil, err
	}
	if viewer.Role != models.RoleLead {
		return nil, ErrForbidden
	}
	n, err := s.store.Count(ctx, intern.ID)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, invalid("add at least one goal before agreeing")
	}
	if err := s.store.Agree(ctx, intern.ID, viewer.ID); err != nil {
		return nil, mapNotFound(err)
	}
	pushFeed(ctx, s.feed, s.log, intern.ID, models.Notice{
		Kind: "goals", Link: "/interns/" + intern.ID, Text: fmt.Sprintf("%s согласовал цели %s", viewer.Name, genitive(intern.Name)),
		ActorID: &viewer.ID, ActorKind: "lead", ActorName: viewer.Name,
	})
	return s.build(ctx, viewer, intern)
}

// ---------- проверки ----------

// access – зритель и стажер, которого ему можно видеть. Чужой стажер – 404.
func (s *Goals) access(ctx context.Context, viewerID, internID string) (*models.User, *models.User, error) {
	viewer, err := s.users.GetByID(ctx, viewerID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, ErrForbidden
	}
	if err != nil {
		return nil, nil, err
	}
	if internID == "" || internID == viewer.ID {
		if viewer.Role != models.RoleIntern {
			return nil, nil, ErrNotFound
		}
		return viewer, viewer, nil
	}
	intern, err := s.users.GetByID(ctx, internID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if !CanViewIntern(viewer, intern) {
		return nil, nil, ErrNotFound
	}
	return viewer, intern, nil
}

// goal – цель, ее стажер и зритель с правом видеть этого стажера.
func (s *Goals) goal(ctx context.Context, viewerID, goalID string) (*models.User, *models.User, *models.Goal, error) {
	g, internID, err := s.store.Get(ctx, goalID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, nil, err
	}
	viewer, intern, err := s.access(ctx, viewerID, internID)
	if err != nil {
		return nil, nil, nil, err
	}
	return viewer, intern, g, nil
}

// checkEdit: наставник и руководитель – всегда; стажер – только свои и только до согласования.
func (s *Goals) checkEdit(ctx context.Context, viewer, intern *models.User) error {
	ok, err := s.canEdit(ctx, viewer, intern)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Goals) canEdit(ctx context.Context, viewer, intern *models.User) (bool, error) {
	if CanManageIntern(viewer, intern) {
		return true, nil
	}
	if viewer.ID != intern.ID {
		return false, nil
	}
	agreed, err := s.store.Agreement(ctx, intern.ID)
	if err != nil {
		return false, err
	}
	return agreed == nil, nil
}

// clean накладывает изменения in на cur и проверяет результат (при создании cur пустой).
func (s *Goals) clean(ctx context.Context, intern *models.User, cur models.GoalFields, in GoalInput) (models.GoalFields, error) {
	f := cur
	if in.Title != nil {
		f.Title = strings.TrimSpace(*in.Title)
	}
	if in.Criterion != nil {
		f.Criterion = strings.TrimSpace(*in.Criterion)
	}
	if in.DueWeek != nil {
		if *in.DueWeek == 0 {
			f.DueWeek = nil
		} else {
			w := *in.DueWeek
			f.DueWeek = &w
		}
	}
	switch {
	case f.Title == "" || len([]rune(f.Title)) > 120:
		return f, invalid("title is required (max 120)")
	case f.Criterion == "" || len([]rune(f.Criterion)) > 160:
		return f, invalid("criterion is required (max 160)")
	case f.DueWeek != nil && (*f.DueWeek < 1 || *f.DueWeek > 52):
		return f, invalid("dueWeek must be between 1 and 52")
	}
	if in.Competency != nil {
		c := strings.TrimSpace(*in.Competency)
		if len([]rune(c)) > 80 {
			return f, invalid("competency is too long (max 80)")
		}
		f.Competency = nil
		if c != "" {
			f.Competency = &c
		}
	}
	f.Steps = nil
	if in.Steps != nil {
		if len(*in.Steps) > 8 {
			return f, invalid("max 8 steps")
		}
		f.Steps = []models.GoalStepInput{}
		for _, st := range *in.Steps {
			st.Title = strings.TrimSpace(st.Title)
			if st.Title == "" || len([]rune(st.Title)) > 120 {
				return f, invalid("step title is required (max 120)")
			}
			if st.Source != "task" && st.Source != "mentor" && st.Source != "lead" {
				return f, invalid("step source must be task, mentor or lead")
			}
			f.Steps = append(f.Steps, st)
		}
	}
	if in.OKRID != nil {
		id := strings.TrimSpace(*in.OKRID)
		if id == "" {
			f.OKRID = nil
		} else {
			okrs, err := s.store.OKRs(ctx, intern.TeamID)
			if err != nil {
				return f, err
			}
			found := false
			for _, o := range okrs {
				found = found || o.ID == id
			}
			if !found {
				return f, invalid("okrId must be an OKR of the intern's team")
			}
			f.OKRID = &id
		}
	}
	return f, nil
}

func (s *Goals) build(ctx context.Context, viewer, intern *models.User) (*models.GoalsView, error) {
	v := &models.GoalsView{InternID: intern.ID}
	var err error
	if v.Goals, err = s.store.List(ctx, intern.ID); err != nil {
		return nil, err
	}
	if v.Agreed, err = s.store.Agreement(ctx, intern.ID); err != nil {
		return nil, err
	}
	if v.OKRs, err = s.store.OKRs(ctx, intern.TeamID); err != nil {
		return nil, err
	}
	for _, g := range v.Goals {
		v.FromOKR = v.FromOKR || g.OKR != nil
	}
	if v.CanEdit, err = s.canEdit(ctx, viewer, intern); err != nil {
		return nil, err
	}
	v.CanAgree = viewer.Role == models.RoleLead && len(v.Goals) > 0 && v.Agreed == nil
	return v, nil
}

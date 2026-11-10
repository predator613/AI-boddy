package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// OnboardingStore – все, что нужно от базы для «Моего пути» и консоли наставника.
type OnboardingStore interface {
	GetInternProfile(ctx context.Context, userID string) (*models.InternProfile, error)
	ListStages(ctx context.Context, teamID string) ([]models.StageTemplate, error)
	GetProgress(ctx context.Context, internID string) (map[string]models.TaskProgress, error)
	ListMilestones(ctx context.Context, teamID string) ([]models.Milestone, error)
	NextMeeting(ctx context.Context, internID string, after time.Time) (*models.Meeting, error)
	ListTeamInterns(ctx context.Context, teamID string) ([]models.User, error)
	GetStage(ctx context.Context, id string) (*models.StageTemplate, error)
	GetTask(ctx context.Context, id string) (*models.TaskTemplate, string, error)
	GetResource(ctx context.Context, id string) (*models.ResourceTemplate, string, error)
	SetTaskStatus(ctx context.Context, internID, taskID, status string, note *string, by string) error
	CreateMeeting(ctx context.Context, internID, hostID, title string, startsAt time.Time) (*models.Meeting, error)
	CreateStage(ctx context.Context, s models.StageTemplate) (*models.StageTemplate, error)
	UpdateStage(ctx context.Context, s models.StageTemplate) (*models.StageTemplate, error)
	DeleteStage(ctx context.Context, id string) error
	CreateTask(ctx context.Context, t models.TaskTemplate) (*models.TaskTemplate, error)
	UpdateTask(ctx context.Context, t models.TaskTemplate) (*models.TaskTemplate, error)
	DeleteTask(ctx context.Context, id string) error
	CreateResource(ctx context.Context, x models.ResourceTemplate) (*models.ResourceTemplate, error)
	DeleteResource(ctx context.Context, id string) error
}

// UserLookup – поиск пользователя по id.
type UserLookup interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
}

// Path – логика страницы «Мой путь» и консоли наставника.
type Path struct {
	store OnboardingStore
	users UserLookup
	now   func() time.Time
}

// NewPath создает сервис.
func NewPath(store OnboardingStore, users UserLookup) *Path {
	return &Path{store: store, users: users, now: time.Now}
}

// GetPath – экран «Мой путь». internID == "" значит «мой собственный».
func (s *Path) GetPath(ctx context.Context, viewerID, internID string) (*models.PathView, error) {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if internID == "" {
		internID = viewer.ID
	}
	intern, err := s.visibleIntern(ctx, viewer, internID)
	if err != nil {
		return nil, err
	}
	return s.buildFor(ctx, intern)
}

// CompleteTask – стажер нажал «Уже сделано».
func (s *Path) CompleteTask(ctx context.Context, viewerID, taskID string) (*models.PathView, error) {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if viewer.Role != models.RoleIntern {
		return nil, ErrForbidden
	}
	if err := s.taskOfTeam(ctx, taskID, viewer.TeamID); err != nil {
		return nil, err
	}
	if err := s.store.SetTaskStatus(ctx, viewer.ID, taskID, models.TaskDone, nil, viewer.ID); err != nil {
		return nil, err
	}
	return s.buildFor(ctx, viewer)
}

// SetInternTaskStatus – наставник или руководитель меняет статус задачи стажера.
func (s *Path) SetInternTaskStatus(ctx context.Context, viewerID, internID, taskID, status string, note *string) (*models.PathView, error) {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	intern, err := s.visibleIntern(ctx, viewer, internID)
	if err != nil {
		return nil, err
	}
	if !CanManageIntern(viewer, intern) {
		return nil, ErrForbidden
	}
	switch status {
	case models.TaskTodo, models.TaskInProgress, models.TaskWaiting, models.TaskDone:
	default:
		return nil, invalid("status must be one of: todo, in_progress, waiting, done")
	}
	if note != nil {
		trimmed := strings.TrimSpace(*note)
		if len([]rune(trimmed)) > 200 {
			return nil, invalid("note is too long (max 200)")
		}
		if trimmed == "" {
			note = nil
		} else {
			note = &trimmed
		}
	}
	if err := s.taskOfTeam(ctx, taskID, intern.TeamID); err != nil {
		return nil, err
	}
	if err := s.store.SetTaskStatus(ctx, intern.ID, taskID, status, note, viewer.ID); err != nil {
		return nil, err
	}
	return s.buildFor(ctx, intern)
}

// ListInterns – стажеры, которых видит наставник или руководитель, с прогрессом.
func (s *Path) ListInterns(ctx context.Context, viewerID string) ([]models.InternSummary, error) {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if viewer.Role == models.RoleIntern {
		return nil, ErrForbidden
	}
	interns, err := s.store.ListTeamInterns(ctx, viewer.TeamID)
	if err != nil {
		return nil, err
	}
	out := []models.InternSummary{}
	for i := range interns {
		in := &interns[i]
		if !CanViewIntern(viewer, in) {
			continue
		}
		v, err := s.buildFor(ctx, in)
		if err != nil {
			return nil, err
		}
		sum := models.InternSummary{ID: in.ID, Login: in.Login, Name: in.Name, TeamName: in.TeamName,
			Mentor: v.Intern.Mentor, Week: v.Week.Current, Progress: v.Progress}
		for _, st := range v.Stages {
			if st.Status == models.StageCurrent {
				sum.StageCode, sum.Stage = st.Code, st.Title
				break
			}
		}
		out = append(out, sum)
	}
	return out, nil
}

// CreateMeeting – наставник назначает встречу стажеру.
func (s *Path) CreateMeeting(ctx context.Context, viewerID, internID, title string, startsAt time.Time) (*models.Meeting, error) {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	intern, err := s.visibleIntern(ctx, viewer, internID)
	if err != nil {
		return nil, err
	}
	if !CanManageIntern(viewer, intern) {
		return nil, ErrForbidden
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Встреча 1:1"
	}
	if len([]rune(title)) > 120 {
		return nil, invalid("title is too long (max 120)")
	}
	if startsAt.IsZero() {
		return nil, invalid("startsAt is required (RFC 3339, e.g. 2026-11-12T15:00:00+03:00)")
	}
	if startsAt.Before(s.now().Add(-time.Hour)) {
		return nil, invalid("startsAt must be in the future")
	}
	return s.store.CreateMeeting(ctx, intern.ID, viewer.ID, title, startsAt)
}

// ---------- внутреннее ----------

func (s *Path) viewer(ctx context.Context, id string) (*models.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrForbidden
	}
	return u, err
}

// visibleIntern – стажер, если viewer имеет право его видеть. Чужих не раскрываем: отвечаем «не найдено».
func (s *Path) visibleIntern(ctx context.Context, viewer *models.User, internID string) (*models.User, error) {
	if internID == viewer.ID {
		if viewer.Role != models.RoleIntern {
			return nil, ErrNotFound
		}
		return viewer, nil
	}
	intern, err := s.users.GetByID(ctx, internID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !CanViewIntern(viewer, intern) {
		return nil, ErrNotFound
	}
	return intern, nil
}

func (s *Path) taskOfTeam(ctx context.Context, taskID, teamID string) error {
	_, taskTeam, err := s.store.GetTask(ctx, taskID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if taskTeam != teamID {
		return ErrNotFound
	}
	return nil
}

func (s *Path) buildFor(ctx context.Context, intern *models.User) (*models.PathView, error) {
	now := s.now()
	in := pathInput{Intern: *intern, Now: now}

	profile, err := s.store.GetInternProfile(ctx, intern.ID)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		in.Profile = models.InternProfile{StartDate: dateOnly(now.In(Moscow)), DurationWeeks: 24}
	case err != nil:
		return nil, err
	default:
		in.Profile = *profile
	}

	if intern.MentorID != nil {
		mentor, err := s.users.GetByID(ctx, *intern.MentorID)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		if mentor != nil {
			in.Mentor = &models.PersonRef{ID: mentor.ID, Name: mentor.Name}
		}
	}

	if in.Stages, err = s.store.ListStages(ctx, intern.TeamID); err != nil {
		return nil, err
	}
	if in.Progress, err = s.store.GetProgress(ctx, intern.ID); err != nil {
		return nil, err
	}
	if in.Milestones, err = s.store.ListMilestones(ctx, intern.TeamID); err != nil {
		return nil, err
	}
	if in.Meeting, err = s.store.NextMeeting(ctx, intern.ID, now); err != nil {
		return nil, err
	}
	v := buildPath(in)
	return &v, nil
}

package services

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// Консоль наставника: универсальные шаблоны этапов, задач и карточек адаптации.
// Все, что наставник создаст здесь, сразу появляется на экране «Мой путь» у стажеров команды.

// ActionKinds – какие действия умеет AI-buddy по кнопке «Сделать с AI-buddy».
// Новые виды добавляются сюда и в agent/mock-services.
var ActionKinds = map[string]string{
	"access_request": "заявка на доступ (payload: service, group, role)",
	"leave_request":  "заявка на отгул (payload: type)",
	"ask_buddy":      "открыть чат с готовым вопросом (payload: question)",
	"open_article":   "открыть статью справочника (payload: articleId)",
}

var (
	resourceKinds = map[string]bool{"person": true, "access": true, "tool": true, "article": true, "link": true}
	tagKinds      = map[string]bool{"ok": true, "wait": true, "none": true, "mentor": true}
	hexColor      = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// StageInput – поля этапа из формы консоли.
type StageInput struct {
	Title       string   `json:"title"`
	Color       string   `json:"color"`
	Tip         string   `json:"tip"`
	Suggestions []string `json:"suggestions"`
	Position    int      `json:"position"` // учитывается только при изменении
}

// TaskInput – поля задачи из формы консоли.
type TaskInput struct {
	Title    string         `json:"title"`
	Note     string         `json:"note"`
	DueDay   *int           `json:"dueDay"`
	Action   *models.Action `json:"action"`
	Position int            `json:"position"`
}

// ResourceInput – поля карточки из формы консоли.
type ResourceInput struct {
	Kind    string  `json:"kind"`
	Name    string  `json:"name"`
	Note    string  `json:"note"`
	Tag     string  `json:"tag"`
	TagKind string  `json:"tagKind"`
	TaskID  *string `json:"taskId"`
}

// ListTemplates – весь шаблон команды наставника.
func (s *Path) ListTemplates(ctx context.Context, viewerID string) ([]models.StageTemplate, error) {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if !CanEditTemplates(viewer, viewer.TeamID) {
		return nil, ErrForbidden
	}
	return s.store.ListStages(ctx, viewer.TeamID)
}

// CreateStage добавляет этап в шаблон команды.
func (s *Path) CreateStage(ctx context.Context, viewerID string, in StageInput) (*models.StageTemplate, error) {
	viewer, err := s.editor(ctx, viewerID, "")
	if err != nil {
		return nil, err
	}
	st, err := cleanStage(in)
	if err != nil {
		return nil, err
	}
	st.TeamID = viewer.TeamID
	return s.store.CreateStage(ctx, st)
}

// UpdateStage меняет этап (в том числе порядок через position).
func (s *Path) UpdateStage(ctx context.Context, viewerID, stageID string, in StageInput) (*models.StageTemplate, error) {
	if _, err := s.editableStage(ctx, viewerID, stageID); err != nil {
		return nil, err
	}
	st, err := cleanStage(in)
	if err != nil {
		return nil, err
	}
	if in.Position < 1 {
		return nil, invalid("position must be >= 1")
	}
	st.ID, st.Position = stageID, in.Position
	return s.store.UpdateStage(ctx, st)
}

// DeleteStage удаляет этап с задачами.
func (s *Path) DeleteStage(ctx context.Context, viewerID, stageID string) error {
	if _, err := s.editableStage(ctx, viewerID, stageID); err != nil {
		return err
	}
	return mapNotFound(s.store.DeleteStage(ctx, stageID))
}

// CreateTask добавляет задачу в этап.
func (s *Path) CreateTask(ctx context.Context, viewerID, stageID string, in TaskInput) (*models.TaskTemplate, error) {
	if _, err := s.editableStage(ctx, viewerID, stageID); err != nil {
		return nil, err
	}
	t, err := cleanTask(in)
	if err != nil {
		return nil, err
	}
	t.StageID = stageID
	return s.store.CreateTask(ctx, t)
}

// UpdateTask меняет задачу.
func (s *Path) UpdateTask(ctx context.Context, viewerID, taskID string, in TaskInput) (*models.TaskTemplate, error) {
	cur, err := s.editableTask(ctx, viewerID, taskID)
	if err != nil {
		return nil, err
	}
	t, err := cleanTask(in)
	if err != nil {
		return nil, err
	}
	if in.Position < 1 {
		return nil, invalid("position must be >= 1")
	}
	t.ID, t.StageID, t.Position = taskID, cur.StageID, in.Position
	return s.store.UpdateTask(ctx, t)
}

// DeleteTask удаляет задачу.
func (s *Path) DeleteTask(ctx context.Context, viewerID, taskID string) error {
	if _, err := s.editableTask(ctx, viewerID, taskID); err != nil {
		return err
	}
	return mapNotFound(s.store.DeleteTask(ctx, taskID))
}

// CreateResource добавляет карточку в боковую панель этапа.
func (s *Path) CreateResource(ctx context.Context, viewerID, stageID string, in ResourceInput) (*models.ResourceTemplate, error) {
	stage, err := s.editableStage(ctx, viewerID, stageID)
	if err != nil {
		return nil, err
	}
	x := models.ResourceTemplate{
		StageID: stageID, Kind: strings.TrimSpace(in.Kind), Name: strings.TrimSpace(in.Name),
		Note: strings.TrimSpace(in.Note), Tag: strings.TrimSpace(in.Tag), TagKind: strings.TrimSpace(in.TagKind),
	}
	if x.TagKind == "" {
		x.TagKind = "none"
	}
	switch {
	case !resourceKinds[x.Kind]:
		return nil, invalid("kind must be one of: person, access, tool, article, link")
	case !tagKinds[x.TagKind]:
		return nil, invalid("tagKind must be one of: ok, wait, none, mentor")
	case x.Name == "" || len([]rune(x.Name)) > 80:
		return nil, invalid("name is required (max 80)")
	case len([]rune(x.Note)) > 120 || len([]rune(x.Tag)) > 40:
		return nil, invalid("note max 120, tag max 40")
	}
	if in.TaskID != nil && *in.TaskID != "" {
		_, team, err := s.store.GetTask(ctx, *in.TaskID)
		if errors.Is(err, repository.ErrNotFound) || (err == nil && team != stage.TeamID) {
			return nil, invalid("taskId must be a task of your team")
		}
		if err != nil {
			return nil, err
		}
		x.TaskID = in.TaskID
	}
	return s.store.CreateResource(ctx, x)
}

// DeleteResource удаляет карточку.
func (s *Path) DeleteResource(ctx context.Context, viewerID, resourceID string) error {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return err
	}
	_, team, err := s.store.GetResource(ctx, resourceID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := s.checkEditor(viewer, team); err != nil {
		return err
	}
	return mapNotFound(s.store.DeleteResource(ctx, resourceID))
}

// ---------- проверки ----------

// editor – пользователь, который может редактировать шаблоны своей команды (или команды teamID).
func (s *Path) editor(ctx context.Context, viewerID, teamID string) (*models.User, error) {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if teamID == "" {
		teamID = viewer.TeamID
	}
	if err := s.checkEditor(viewer, teamID); err != nil {
		return nil, err
	}
	return viewer, nil
}

// checkEditor: стажеру – 403, наставнику чужой команды – 404 (чужие шаблоны не раскрываем).
func (s *Path) checkEditor(viewer *models.User, teamID string) error {
	if viewer.Role == models.RoleIntern {
		return ErrForbidden
	}
	if !CanEditTemplates(viewer, teamID) {
		return ErrNotFound
	}
	return nil
}

func (s *Path) editableStage(ctx context.Context, viewerID, stageID string) (*models.StageTemplate, error) {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if viewer.Role == models.RoleIntern {
		return nil, ErrForbidden
	}
	st, err := s.store.GetStage(ctx, stageID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.checkEditor(viewer, st.TeamID); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *Path) editableTask(ctx context.Context, viewerID, taskID string) (*models.TaskTemplate, error) {
	viewer, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if viewer.Role == models.RoleIntern {
		return nil, ErrForbidden
	}
	t, team, err := s.store.GetTask(ctx, taskID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.checkEditor(viewer, team); err != nil {
		return nil, err
	}
	return t, nil
}

// ---------- очистка и валидация ввода ----------

func cleanStage(in StageInput) (models.StageTemplate, error) {
	st := models.StageTemplate{
		Title: strings.TrimSpace(in.Title), Color: strings.TrimSpace(in.Color), Tip: strings.TrimSpace(in.Tip),
		Suggestions: []string{},
	}
	if st.Color == "" {
		st.Color = "#C9FFBF"
	}
	if st.Title == "" || len([]rune(st.Title)) > 80 {
		return st, invalid("title is required (max 80)")
	}
	if !hexColor.MatchString(st.Color) {
		return st, invalid("color must look like #RRGGBB")
	}
	if len([]rune(st.Tip)) > 300 {
		return st, invalid("tip is too long (max 300)")
	}
	if len(in.Suggestions) > 6 {
		return st, invalid("max 6 suggestions")
	}
	for _, q := range in.Suggestions {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		if len([]rune(q)) > 80 {
			return st, invalid("suggestion is too long (max 80)")
		}
		st.Suggestions = append(st.Suggestions, q)
	}
	return st, nil
}

func cleanTask(in TaskInput) (models.TaskTemplate, error) {
	t := models.TaskTemplate{Title: strings.TrimSpace(in.Title), Note: strings.TrimSpace(in.Note), DueDay: in.DueDay}
	if t.Title == "" || len([]rune(t.Title)) > 120 {
		return t, invalid("title is required (max 120)")
	}
	if len([]rune(t.Note)) > 300 {
		return t, invalid("note is too long (max 300)")
	}
	if t.DueDay != nil && (*t.DueDay < 1 || *t.DueDay > 730) {
		return t, invalid("dueDay must be between 1 and 730")
	}
	if in.Action != nil {
		if _, ok := ActionKinds[in.Action.Kind]; !ok {
			return t, invalid("unknown action kind; allowed: access_request, leave_request, ask_buddy, open_article")
		}
		payload := in.Action.Payload
		if len(payload) == 0 || string(payload) == "null" {
			payload = json.RawMessage(`{}`)
		}
		var obj map[string]any
		if err := json.Unmarshal(payload, &obj); err != nil {
			return t, invalid("action.payload must be a JSON object")
		}
		t.Action = &models.Action{Kind: in.Action.Kind, Payload: payload}
	}
	return t, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/models"
)

// Onboarding – запросы для страницы «Мой путь» и консоли наставника.
type Onboarding struct {
	db *pgxpool.Pool
}

// NewOnboarding создает репозиторий адаптации.
func NewOnboarding(db *pgxpool.Pool) *Onboarding {
	return &Onboarding{db: db}
}

// ---------- чтение ----------

// GetInternProfile возвращает сроки стажировки.
func (r *Onboarding) GetInternProfile(ctx context.Context, userID string) (*models.InternProfile, error) {
	var p models.InternProfile
	err := r.db.QueryRow(ctx,
		`SELECT start_date, duration_weeks FROM intern_profiles WHERE user_id::text = $1`, userID,
	).Scan(&p.StartDate, &p.DurationWeeks)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListStages возвращает шаблон команды: этапы с задачами и карточками, по порядку.
func (r *Onboarding) ListStages(ctx context.Context, teamID string) ([]models.StageTemplate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, team_id::text, position, title, color, tip, suggestions
		FROM onboarding_stages WHERE team_id::text = $1
		ORDER BY position, created_at`, teamID)
	if err != nil {
		return nil, err
	}
	stages, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.StageTemplate, error) { return scanStage(row) })
	if err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return stages, nil
	}

	byID := make(map[string]*models.StageTemplate, len(stages))
	for i := range stages {
		stages[i].Tasks = []models.TaskTemplate{}
		stages[i].Resources = []models.ResourceTemplate{}
		byID[stages[i].ID] = &stages[i]
	}

	trows, err := r.db.Query(ctx, taskSelect+`
		JOIN onboarding_stages s ON s.id = t.stage_id
		WHERE s.team_id::text = $1
		ORDER BY t.position, t.title`, teamID)
	if err != nil {
		return nil, err
	}
	tasks, err := pgx.CollectRows(trows, func(row pgx.CollectableRow) (models.TaskTemplate, error) { return scanTask(row) })
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		byID[t.StageID].Tasks = append(byID[t.StageID].Tasks, t)
	}

	rrows, err := r.db.Query(ctx, resourceSelect+`
		JOIN onboarding_stages s ON s.id = x.stage_id
		WHERE s.team_id::text = $1
		ORDER BY x.position, x.name`, teamID)
	if err != nil {
		return nil, err
	}
	res, err := pgx.CollectRows(rrows, scanResource)
	if err != nil {
		return nil, err
	}
	for _, x := range res {
		byID[x.StageID].Resources = append(byID[x.StageID].Resources, x)
	}
	return stages, nil
}

// GetProgress возвращает личные статусы задач стажера: task_id → статус.
func (r *Onboarding) GetProgress(ctx context.Context, internID string) (map[string]models.TaskProgress, error) {
	rows, err := r.db.Query(ctx,
		`SELECT task_id::text, status, note FROM intern_task_progress WHERE intern_id::text = $1`, internID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]models.TaskProgress{}
	for rows.Next() {
		var id string
		var p models.TaskProgress
		if err := rows.Scan(&id, &p.Status, &p.Note); err != nil {
			return nil, err
		}
		out[id] = p
	}
	return out, rows.Err()
}

// ListMilestones – вехи шкалы недель для команды.
func (r *Onboarding) ListMilestones(ctx context.Context, teamID string) ([]models.Milestone, error) {
	rows, err := r.db.Query(ctx,
		`SELECT week, title FROM track_milestones WHERE team_id::text = $1 ORDER BY week`, teamID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Milestone, error) {
		var m models.Milestone
		err := row.Scan(&m.Week, &m.Title)
		return m, err
	})
}

// NextMeeting – ближайшая встреча стажера после момента after (nil, если нет).
func (r *Onboarding) NextMeeting(ctx context.Context, internID string, after time.Time) (*models.Meeting, error) {
	var m models.Meeting
	err := r.db.QueryRow(ctx, `
		SELECT m.id::text, m.title, m.starts_at, m.host_id::text, u.name
		FROM meetings m JOIN users u ON u.id = m.host_id
		WHERE m.intern_id::text = $1 AND m.starts_at >= $2
		ORDER BY m.starts_at LIMIT 1`, internID, after,
	).Scan(&m.ID, &m.Title, &m.StartsAt, &m.HostID, &m.HostName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ListTeamInterns – все стажеры команды.
func (r *Onboarding) ListTeamInterns(ctx context.Context, teamID string) ([]models.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id::text, u.login, u.name, u.role, u.team_id::text, t.name, u.mentor_id::text
		FROM users u JOIN teams t ON t.id = u.team_id
		WHERE u.role = 'intern' AND u.team_id::text = $1
		ORDER BY u.name`, teamID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.User, error) {
		var u models.User
		err := row.Scan(&u.ID, &u.Login, &u.Name, &u.Role, &u.TeamID, &u.TeamName, &u.MentorID)
		return u, err
	})
}

// GetStage – этап без задач (для проверок прав и редактирования).
func (r *Onboarding) GetStage(ctx context.Context, id string) (*models.StageTemplate, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id::text, team_id::text, position, title, color, tip, suggestions
		FROM onboarding_stages WHERE id::text = $1`, id)
	s, err := scanStage(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetTask – задача шаблона и id команды, которой она принадлежит.
func (r *Onboarding) GetTask(ctx context.Context, id string) (*models.TaskTemplate, string, error) {
	var teamID string
	row := r.db.QueryRow(ctx, `
		SELECT t.id::text, t.stage_id::text, t.position, t.title, t.note, t.due_day, t.action_kind, t.action_payload, s.team_id::text
		FROM onboarding_tasks t JOIN onboarding_stages s ON s.id = t.stage_id
		WHERE t.id::text = $1`, id)
	t, err := scanTaskWith(row, &teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	return &t, teamID, nil
}

// GetResource – карточка шаблона и id команды.
func (r *Onboarding) GetResource(ctx context.Context, id string) (*models.ResourceTemplate, string, error) {
	var teamID string
	var x models.ResourceTemplate
	err := r.db.QueryRow(ctx, `
		SELECT x.id::text, x.stage_id::text, x.position, x.kind, x.name, x.note, x.tag, x.tag_kind, x.task_id::text, s.team_id::text
		FROM onboarding_resources x JOIN onboarding_stages s ON s.id = x.stage_id
		WHERE x.id::text = $1`, id,
	).Scan(&x.ID, &x.StageID, &x.Position, &x.Kind, &x.Name, &x.Note, &x.Tag, &x.TagKind, &x.TaskID, &teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	return &x, teamID, nil
}

// ---------- запись: прогресс и встречи ----------

// SetTaskStatus сохраняет статус задачи стажера (upsert).
func (r *Onboarding) SetTaskStatus(ctx context.Context, internID, taskID, status string, note *string, by string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO intern_task_progress (intern_id, task_id, status, note, updated_by, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, now())
		ON CONFLICT (intern_id, task_id)
		DO UPDATE SET status = EXCLUDED.status, note = EXCLUDED.note,
		              updated_by = EXCLUDED.updated_by, updated_at = now()`,
		internID, taskID, status, note, by)
	return err
}

// CreateMeeting создает встречу.
func (r *Onboarding) CreateMeeting(ctx context.Context, internID, hostID, title string, startsAt time.Time) (*models.Meeting, error) {
	var m models.Meeting
	err := r.db.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO meetings (intern_id, host_id, title, starts_at)
			VALUES ($1::uuid, $2::uuid, $3, $4)
			RETURNING id, title, starts_at, host_id
		)
		SELECT ins.id::text, ins.title, ins.starts_at, ins.host_id::text, u.name
		FROM ins JOIN users u ON u.id = ins.host_id`,
		internID, hostID, title, startsAt,
	).Scan(&m.ID, &m.Title, &m.StartsAt, &m.HostID, &m.HostName)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ---------- запись: шаблоны (консоль наставника) ----------

// CreateStage добавляет этап в конец списка команды.
func (r *Onboarding) CreateStage(ctx context.Context, s models.StageTemplate) (*models.StageTemplate, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO onboarding_stages (team_id, position, title, color, tip, suggestions)
		VALUES ($1::uuid,
		        COALESCE((SELECT max(position) FROM onboarding_stages WHERE team_id = $1::uuid), 0) + 1,
		        $2, $3, $4, $5)
		RETURNING id::text, team_id::text, position, title, color, tip, suggestions`,
		s.TeamID, s.Title, s.Color, s.Tip, s.Suggestions)
	out, err := scanStage(row)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateStage меняет поля этапа (включая position для перестановки).
func (r *Onboarding) UpdateStage(ctx context.Context, s models.StageTemplate) (*models.StageTemplate, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE onboarding_stages SET position = $2, title = $3, color = $4, tip = $5, suggestions = $6
		WHERE id::text = $1
		RETURNING id::text, team_id::text, position, title, color, tip, suggestions`,
		s.ID, s.Position, s.Title, s.Color, s.Tip, s.Suggestions)
	out, err := scanStage(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteStage удаляет этап вместе с задачами и карточками.
func (r *Onboarding) DeleteStage(ctx context.Context, id string) error {
	return execOne(r.db.Exec(ctx, `DELETE FROM onboarding_stages WHERE id::text = $1`, id))
}

// CreateTask добавляет задачу в конец этапа.
func (r *Onboarding) CreateTask(ctx context.Context, t models.TaskTemplate) (*models.TaskTemplate, error) {
	kind, payload := actionColumns(t.Action)
	row := r.db.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO onboarding_tasks (stage_id, position, title, note, due_day, action_kind, action_payload)
			VALUES ($1::uuid,
			        COALESCE((SELECT max(position) FROM onboarding_tasks WHERE stage_id = $1::uuid), 0) + 1,
			        $2, $3, $4, $5, $6)
			RETURNING *
		)
		SELECT t.id::text, t.stage_id::text, t.position, t.title, t.note, t.due_day, t.action_kind, t.action_payload
		FROM ins t`,
		t.StageID, t.Title, t.Note, t.DueDay, kind, payload)
	out, err := scanTask(row)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateTask меняет поля задачи.
func (r *Onboarding) UpdateTask(ctx context.Context, t models.TaskTemplate) (*models.TaskTemplate, error) {
	kind, payload := actionColumns(t.Action)
	row := r.db.QueryRow(ctx, `
		UPDATE onboarding_tasks t
		SET position = $2, title = $3, note = $4, due_day = $5, action_kind = $6, action_payload = $7
		WHERE t.id::text = $1
		RETURNING t.id::text, t.stage_id::text, t.position, t.title, t.note, t.due_day, t.action_kind, t.action_payload`,
		t.ID, t.Position, t.Title, t.Note, t.DueDay, kind, payload)
	out, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteTask удаляет задачу (и прогресс стажеров по ней).
func (r *Onboarding) DeleteTask(ctx context.Context, id string) error {
	return execOne(r.db.Exec(ctx, `DELETE FROM onboarding_tasks WHERE id::text = $1`, id))
}

// CreateResource добавляет карточку в конец боковой панели этапа.
func (r *Onboarding) CreateResource(ctx context.Context, x models.ResourceTemplate) (*models.ResourceTemplate, error) {
	var out models.ResourceTemplate
	err := r.db.QueryRow(ctx, `
		INSERT INTO onboarding_resources (stage_id, position, kind, name, note, tag, tag_kind, task_id)
		VALUES ($1::uuid,
		        COALESCE((SELECT max(position) FROM onboarding_resources WHERE stage_id = $1::uuid), 0) + 1,
		        $2, $3, $4, $5, $6, $7::uuid)
		RETURNING id::text, stage_id::text, position, kind, name, note, tag, tag_kind, task_id::text`,
		x.StageID, x.Kind, x.Name, x.Note, x.Tag, x.TagKind, x.TaskID,
	).Scan(&out.ID, &out.StageID, &out.Position, &out.Kind, &out.Name, &out.Note, &out.Tag, &out.TagKind, &out.TaskID)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteResource удаляет карточку.
func (r *Onboarding) DeleteResource(ctx context.Context, id string) error {
	return execOne(r.db.Exec(ctx, `DELETE FROM onboarding_resources WHERE id::text = $1`, id))
}

// ---------- сканеры ----------

const taskSelect = `
	SELECT t.id::text, t.stage_id::text, t.position, t.title, t.note, t.due_day, t.action_kind, t.action_payload
	FROM onboarding_tasks t`

const resourceSelect = `
	SELECT x.id::text, x.stage_id::text, x.position, x.kind, x.name, x.note, x.tag, x.tag_kind, x.task_id::text
	FROM onboarding_resources x`

func scanStage(row pgx.Row) (models.StageTemplate, error) {
	var s models.StageTemplate
	err := row.Scan(&s.ID, &s.TeamID, &s.Position, &s.Title, &s.Color, &s.Tip, &s.Suggestions)
	if s.Suggestions == nil {
		s.Suggestions = []string{}
	}
	return s, err
}

func scanTask(row pgx.Row) (models.TaskTemplate, error) {
	return scanTaskWith(row)
}

// scanTaskWith читает задачу и, если переданы, дополнительные колонки в конце строки.
func scanTaskWith(row pgx.Row, extra ...any) (models.TaskTemplate, error) {
	var t models.TaskTemplate
	var kind *string
	var payload []byte
	dest := append([]any{&t.ID, &t.StageID, &t.Position, &t.Title, &t.Note, &t.DueDay, &kind, &payload}, extra...)
	if err := row.Scan(dest...); err != nil {
		return t, err
	}
	if kind != nil {
		t.Action = &models.Action{Kind: *kind, Payload: json.RawMessage(payload)}
	}
	return t, nil
}

func scanResource(row pgx.CollectableRow) (models.ResourceTemplate, error) {
	var x models.ResourceTemplate
	err := row.Scan(&x.ID, &x.StageID, &x.Position, &x.Kind, &x.Name, &x.Note, &x.Tag, &x.TagKind, &x.TaskID)
	return x, err
}

func actionColumns(a *models.Action) (*string, string) {
	if a == nil {
		return nil, "{}"
	}
	payload := string(a.Payload)
	if payload == "" || payload == "null" {
		payload = "{}"
	}
	return &a.Kind, payload
}

func execOne(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

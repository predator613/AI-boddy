package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		SELECT `+stageCols+`
		FROM onboarding_stages s WHERE s.team_id::text = $1
		ORDER BY s.position, s.created_at`, teamID)
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

	trows, err := r.db.Query(ctx, taskFrom+`
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
		ORDER BY x.position, x.id`, teamID)
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

// TaskDetails – подробности задач команды для стажера: описание, кто поможет, когда готово и цель
// стажировки, в шагах которой есть задача.
func (r *Onboarding) TaskDetails(ctx context.Context, teamID, internID string) (map[string]models.TaskDetails, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id::text, t.about, t.helper, t.done_when,
		       (SELECT g.title FROM goal_steps gs JOIN intern_goals g ON g.id = gs.goal_id
		        WHERE gs.task_id = t.id AND g.intern_id::text = $2 ORDER BY g.position LIMIT 1)
		FROM onboarding_tasks t JOIN onboarding_stages s ON s.id = t.stage_id
		WHERE s.team_id::text = $1`, teamID, internID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]models.TaskDetails{}
	for rows.Next() {
		var id string
		var d models.TaskDetails
		if err := rows.Scan(&id, &d.About, &d.Helper, &d.DoneWhen, &d.Goal); err != nil {
			return nil, err
		}
		out[id] = d
	}
	return out, rows.Err()
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
		JOIN meeting_participants mp ON mp.meeting_id = m.id
		WHERE mp.user_id::text = $1 AND m.starts_at >= $2 AND m.status = 'scheduled'
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
		SELECT u.id::text, u.login, u.name, u.role, u.team_id::text, t.name, p.mentor_id::text
		FROM users u JOIN teams t ON t.id = u.team_id
		LEFT JOIN intern_profiles p ON p.user_id = u.id
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
		SELECT `+stageCols+`
		FROM onboarding_stages s WHERE s.id::text = $1`, id)
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
	row := r.db.QueryRow(ctx, taskSelect+`, s.team_id::text
		FROM onboarding_tasks t
		LEFT JOIN task_actions a ON a.task_id = t.id
		JOIN onboarding_stages s ON s.id = t.stage_id
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
		SELECT `+resourceCols+`, s.team_id::text
		FROM onboarding_resources x`+resourceJoins+`
		JOIN onboarding_stages s ON s.id = x.stage_id
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

// CreateMeeting создает встречу один на один: сама встреча и участник – в одной транзакции.
func (r *Onboarding) CreateMeeting(ctx context.Context, internID, hostID, title string, startsAt time.Time) (*models.Meeting, error) {
	var m models.Meeting
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO meetings (host_id, title, starts_at, kind)
				VALUES ($1::uuid, $2, $3, 'one_on_one')
				RETURNING id, title, starts_at, host_id
			)
			SELECT ins.id::text, ins.title, ins.starts_at, ins.host_id::text, u.name
			FROM ins JOIN users u ON u.id = ins.host_id`,
			hostID, title, startsAt,
		).Scan(&m.ID, &m.Title, &m.StartsAt, &m.HostID, &m.HostName)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO meeting_participants (meeting_id, user_id) VALUES ($1::uuid, $2::uuid)`, m.ID, internID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ---------- запись: шаблоны (консоль наставника) ----------

// CreateStage добавляет этап в конец списка команды. Подсказки пишутся в stage_suggestions.
func (r *Onboarding) CreateStage(ctx context.Context, st models.StageTemplate) (*models.StageTemplate, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO onboarding_stages (team_id, position, title, color, tip)
			VALUES ($1::uuid,
			        COALESCE((SELECT max(position) FROM onboarding_stages WHERE team_id = $1::uuid), 0) + 1,
			        $2, $3, $4)
			RETURNING id::text`,
			st.TeamID, st.Title, st.Color, st.Tip).Scan(&id)
		if err != nil {
			return err
		}
		return replaceSuggestions(ctx, tx, id, st.Suggestions)
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	return r.GetStage(ctx, id)
}

// UpdateStage меняет поля этапа (включая position для перестановки) и список подсказок.
func (r *Onboarding) UpdateStage(ctx context.Context, st models.StageTemplate) (*models.StageTemplate, error) {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := execOne(tx.Exec(ctx, `
			UPDATE onboarding_stages SET position = $2, title = $3, color = $4, tip = $5
			WHERE id::text = $1`,
			st.ID, st.Position, st.Title, st.Color, st.Tip)); err != nil {
			return err
		}
		return replaceSuggestions(ctx, tx, st.ID, st.Suggestions)
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	return r.GetStage(ctx, st.ID)
}

// DeleteStage удаляет этап вместе с задачами и карточками.
func (r *Onboarding) DeleteStage(ctx context.Context, id string) error {
	return execOne(r.db.Exec(ctx, `DELETE FROM onboarding_stages WHERE id::text = $1`, id))
}

// CreateTask добавляет задачу в конец этапа вместе с действием AI-buddy.
func (r *Onboarding) CreateTask(ctx context.Context, t models.TaskTemplate) (*models.TaskTemplate, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO onboarding_tasks (stage_id, position, title, note, due_day)
			VALUES ($1::uuid,
			        COALESCE((SELECT max(position) FROM onboarding_tasks WHERE stage_id = $1::uuid), 0) + 1,
			        $2, $3, $4)
			RETURNING id::text`,
			t.StageID, t.Title, t.Note, t.DueDay).Scan(&id)
		if err != nil {
			return err
		}
		return replaceAction(ctx, tx, id, t.Action)
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	out, _, err := r.GetTask(ctx, id)
	return out, err
}

// UpdateTask меняет поля задачи и ее действие.
func (r *Onboarding) UpdateTask(ctx context.Context, t models.TaskTemplate) (*models.TaskTemplate, error) {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := execOne(tx.Exec(ctx, `
			UPDATE onboarding_tasks SET position = $2, title = $3, note = $4, due_day = $5
			WHERE id::text = $1`,
			t.ID, t.Position, t.Title, t.Note, t.DueDay)); err != nil {
			return err
		}
		return replaceAction(ctx, tx, t.ID, t.Action)
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	out, _, err := r.GetTask(ctx, t.ID)
	return out, err
}

// DeleteTask удаляет задачу (и прогресс стажеров по ней).
func (r *Onboarding) DeleteTask(ctx context.Context, id string) error {
	return execOne(r.db.Exec(ctx, `DELETE FROM onboarding_tasks WHERE id::text = $1`, id))
}

// CreateResource добавляет карточку в конец боковой панели этапа.
// Карточка «человек» ссылается на пользователя команды с таким именем, а не хранит имя копией.
func (r *Onboarding) CreateResource(ctx context.Context, x models.ResourceTemplate) (*models.ResourceTemplate, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		WITH person AS (
			SELECT u.id FROM users u JOIN onboarding_stages s ON s.team_id = u.team_id
			WHERE s.id = $1::uuid AND $2 = 'person' AND u.name = $3
			ORDER BY u.created_at LIMIT 1
		)
		INSERT INTO onboarding_resources (stage_id, position, kind, name, user_id, note, tag, tag_kind, task_id)
		VALUES ($1::uuid,
		        COALESCE((SELECT max(position) FROM onboarding_resources WHERE stage_id = $1::uuid), 0) + 1,
		        $2, CASE WHEN EXISTS (SELECT 1 FROM person) THEN NULL ELSE $3 END, (SELECT id FROM person),
		        $4, $5, $6, $7::uuid)
		RETURNING id::text`,
		x.StageID, x.Kind, x.Name, x.Note, x.Tag, x.TagKind, x.TaskID,
	).Scan(&id)
	if err != nil {
		return nil, constraintErr(err)
	}
	out, _, err := r.GetResource(ctx, id)
	return out, err
}

// DeleteResource удаляет карточку.
func (r *Onboarding) DeleteResource(ctx context.Context, id string) error {
	return execOne(r.db.Exec(ctx, `DELETE FROM onboarding_resources WHERE id::text = $1`, id))
}

// ---------- сканеры ----------

// Подсказки этапа собираются из stage_suggestions в массив по порядку.
const stageCols = `s.id::text, s.team_id::text, s.position, s.title, s.color, s.tip,
	ARRAY(SELECT q.text FROM stage_suggestions q WHERE q.stage_id = s.id ORDER BY q.position)`

// Действие задачи хранится строкой task_actions; фронту отдается прежний формат {kind, payload}.
const taskSelect = `
	SELECT t.id::text, t.stage_id::text, t.position, t.title, t.note, t.due_day, a.kind,
	       CASE WHEN a.kind IS NULL THEN NULL ELSE jsonb_strip_nulls(jsonb_build_object(
	           'service', a.service_code, 'role', a.service_role, 'project', a.project_code,
	           'type', a.leave_type, 'question', a.question, 'articleId', a.article_id::text)) END`

const taskFrom = taskSelect + `
	FROM onboarding_tasks t
	LEFT JOIN task_actions a ON a.task_id = t.id`

// Имя карточки берется из пользователя или статьи, если карточка на них ссылается.
const resourceCols = `x.id::text, x.stage_id::text, x.position, x.kind,
	COALESCE(pu.name, ka.title, x.name, ''), x.note, x.tag, x.tag_kind, x.task_id::text`

const resourceJoins = `
	LEFT JOIN users pu ON pu.id = x.user_id
	LEFT JOIN kb_articles ka ON ka.id = x.article_id`

const resourceSelect = `
	SELECT ` + resourceCols + `
	FROM onboarding_resources x` + resourceJoins

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

// actionPayload – поля payload действия. group – старое имя проекта GitLab (backend → backend/auth-service).
type actionPayload struct {
	Service   *string `json:"service"`
	Role      *string `json:"role"`
	Project   *string `json:"project"`
	Group     *string `json:"group"`
	Type      *string `json:"type"`
	Question  *string `json:"question"`
	ArticleID *string `json:"articleId"`
}

var legacyGroups = map[string]string{"backend": "backend/auth-service", "frontend": "frontend/web-app"}

func replaceSuggestions(ctx context.Context, tx pgx.Tx, stageID string, list []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM stage_suggestions WHERE stage_id::text = $1`, stageID); err != nil {
		return err
	}
	if len(list) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO stage_suggestions (stage_id, position, text)
		SELECT $1::uuid, x.ord, x.q FROM unnest($2::text[]) WITH ORDINALITY AS x(q, ord)`, stageID, list)
	return err
}

func replaceAction(ctx context.Context, tx pgx.Tx, taskID string, a *models.Action) error {
	if _, err := tx.Exec(ctx, `DELETE FROM task_actions WHERE task_id::text = $1`, taskID); err != nil {
		return err
	}
	if a == nil {
		return nil
	}
	var p actionPayload
	if len(a.Payload) > 0 {
		if err := json.Unmarshal(a.Payload, &p); err != nil {
			return ErrInvalidReference
		}
	}
	if p.Project == nil && p.Group != nil {
		if code, ok := legacyGroups[*p.Group]; ok {
			p.Project = &code
		}
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO task_actions (task_id, kind, service_code, service_role, project_code, leave_type, question, article_id)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8::uuid)`,
		taskID, a.Kind, p.Service, p.Role, p.Project, p.Type, p.Question, p.ArticleID)
	return err
}

// ErrInvalidReference – запись ссылается на несуществующие данные или нарушает правила таблицы
// (нет такого сервиса, роли, проекта, статьи, человека в команде). Для API это ошибка 400.
var ErrInvalidReference = errors.New("invalid reference")

// constraintErr превращает нарушения внешних ключей и CHECK в ErrInvalidReference.
func constraintErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503", "23514", "23502", "22P02":
			return fmt.Errorf("%w: %s", ErrInvalidReference, pgErr.ConstraintName)
		}
	}
	return err
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

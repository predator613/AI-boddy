package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/models"
)

// Goals – цели стажировки, OKR команды и история прогресса.
type Goals struct {
	db *pgxpool.Pool
}

// NewGoals создает репозиторий целей.
func NewGoals(db *pgxpool.Pool) *Goals {
	return &Goals{db: db}
}

// Последняя отметка о прогрессе берется из goal_updates, OKR и автор – по ссылкам.
// Если у цели есть шаги, процент – доля выполненных шагов (считается запросом, не хранится).
const goalSelect = `
	SELECT g.id::text, g.intern_id::text, g.position, g.title, g.criterion, g.due_week,
	       COALESCE((
	           SELECT round(100.0 * count(*) FILTER (WHERE ` + stepDone + `) / count(*))::int
	           FROM goal_steps s
	           LEFT JOIN intern_task_progress p ON p.task_id = s.task_id AND p.intern_id = g.intern_id
	           WHERE s.goal_id = g.id HAVING count(*) > 0), g.progress_pct),
	       g.updated_at,
	       o.id::text, o.period, o.title,
	       cu.id::text, cu.name,
	       lu.author_id::text, lu.author_name, lu.progress_pct, lu.note, lu.created_at
	FROM intern_goals g
	LEFT JOIN team_okrs o ON o.id = g.okr_id
	LEFT JOIN users cu ON cu.id = g.created_by
	LEFT JOIN LATERAL (
		SELECT x.author_id, a.name AS author_name, x.progress_pct, x.note, x.created_at
		FROM goal_updates x JOIN users a ON a.id = x.author_id
		WHERE x.goal_id = g.id ORDER BY x.created_at DESC, x.id DESC LIMIT 1
	) lu ON true`

// stepDone – шаг выполнен: задача пути закрыта или наставник (руководитель) отметил шаг.
const stepDone = `(s.source = 'task' AND p.status = 'done') OR s.done_at IS NOT NULL`

// List – цели стажера по порядку, с шагами.
func (r *Goals) List(ctx context.Context, internID string) ([]models.Goal, error) {
	rows, err := r.db.Query(ctx, goalSelect+` WHERE g.intern_id::text = $1 ORDER BY g.position, g.created_at`, internID)
	if err != nil {
		return nil, err
	}
	goals, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Goal, error) {
		g, _, err := scanGoal(row)
		return g, err
	})
	if err != nil {
		return nil, err
	}
	steps, err := r.steps(ctx, `g.intern_id::text = $1`, internID)
	if err != nil {
		return nil, err
	}
	for i := range goals {
		goals[i].Steps = orEmpty(steps[goals[i].ID])
	}
	return goals, nil
}

// steps – шаги целей по условию на g (intern_goals), сгруппированные по цели.
func (r *Goals) steps(ctx context.Context, where string, arg string) (map[string][]models.GoalStep, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.goal_id::text, s.id::text, s.title, s.source, lpad(st.position::text, 2, '0'), s.task_id::text,
		       CASE WHEN s.source = 'task' THEN
		                CASE COALESCE(p.status, 'todo') WHEN 'waiting' THEN 'in_progress' ELSE COALESCE(p.status, 'todo') END
		            WHEN s.done_at IS NOT NULL THEN 'done' ELSE 'todo' END
		FROM goal_steps s
		JOIN intern_goals g ON g.id = s.goal_id
		LEFT JOIN onboarding_tasks t ON t.id = s.task_id
		LEFT JOIN onboarding_stages st ON st.id = t.stage_id
		LEFT JOIN intern_task_progress p ON p.task_id = s.task_id AND p.intern_id = g.intern_id
		WHERE `+where+` ORDER BY s.goal_id, s.position`, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]models.GoalStep{}
	for rows.Next() {
		var goalID string
		var st models.GoalStep
		if err := rows.Scan(&goalID, &st.ID, &st.Title, &st.Source, &st.StageCode, &st.TaskID, &st.Status); err != nil {
			return nil, err
		}
		out[goalID] = append(out[goalID], st)
	}
	return out, rows.Err()
}

func orEmpty(s []models.GoalStep) []models.GoalStep {
	if s == nil {
		return []models.GoalStep{}
	}
	return s
}

// Step – шаг цели (для проверки перед отметкой).
func (r *Goals) Step(ctx context.Context, goalID, stepID string) (*models.GoalStep, error) {
	var st models.GoalStep
	err := r.db.QueryRow(ctx, `SELECT id::text, title, source FROM goal_steps WHERE id::text = $1 AND goal_id::text = $2`,
		stepID, goalID).Scan(&st.ID, &st.Title, &st.Source)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// SetStepDone отмечает шаг наставника или руководителя выполненным (done = false снимает отметку).
func (r *Goals) SetStepDone(ctx context.Context, goalID, stepID, by string, done bool) (*models.Goal, error) {
	var err error
	if done {
		err = execOne(r.db.Exec(ctx, `
			UPDATE goal_steps SET done_at = COALESCE(done_at, now()), done_by = COALESCE(done_by, $3::uuid)
			WHERE id::text = $1 AND goal_id::text = $2 AND source <> 'task'`, stepID, goalID, by))
	} else {
		err = execOne(r.db.Exec(ctx, `
			UPDATE goal_steps SET done_at = NULL, done_by = NULL
			WHERE id::text = $1 AND goal_id::text = $2 AND source <> 'task'`, stepID, goalID))
	}
	if err != nil {
		return nil, constraintErr(err)
	}
	g, _, err := r.Get(ctx, goalID)
	return g, err
}

// Get – цель и id стажера, которому она принадлежит.
func (r *Goals) Get(ctx context.Context, id string) (*models.Goal, string, error) {
	g, internID, err := scanGoal(r.db.QueryRow(ctx, goalSelect+` WHERE g.id::text = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	steps, err := r.steps(ctx, `g.id::text = $1`, id)
	if err != nil {
		return nil, "", err
	}
	g.Steps = orEmpty(steps[g.ID])
	return &g, internID, nil
}

// Agreement – кто и когда согласовал цели (nil – не согласованы).
func (r *Goals) Agreement(ctx context.Context, internID string) (*models.GoalsAgreement, error) {
	var by, name *string
	var at *time.Time
	err := r.db.QueryRow(ctx, `
		SELECT p.goals_agreed_by::text, u.name, p.goals_agreed_at
		FROM intern_profiles p LEFT JOIN users u ON u.id = p.goals_agreed_by
		WHERE p.user_id::text = $1`, internID,
	).Scan(&by, &name, &at)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && by == nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &models.GoalsAgreement{By: models.PersonRef{ID: *by, Name: *name}, At: *at}, nil
}

// OKRs – цели команды.
func (r *Goals) OKRs(ctx context.Context, teamID string) ([]models.TeamOKR, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, period, title FROM team_okrs WHERE team_id::text = $1 ORDER BY period, title`, teamID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.TeamOKR, error) {
		var o models.TeamOKR
		err := row.Scan(&o.ID, &o.Period, &o.Title)
		return o, err
	})
}

// Count – сколько целей у стажера.
func (r *Goals) Count(ctx context.Context, internID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM intern_goals WHERE intern_id::text = $1`, internID).Scan(&n)
	return n, err
}

// Create добавляет цель в конец списка.
func (r *Goals) Create(ctx context.Context, internID, by string, f models.GoalFields) (*models.Goal, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO intern_goals (intern_id, position, title, criterion, due_week, okr_id, created_by)
		VALUES ($1::uuid,
		        COALESCE((SELECT max(position) FROM intern_goals WHERE intern_id = $1::uuid), 0) + 1,
		        $2, $3, $4, $5::uuid, $6::uuid)
		RETURNING id::text`,
		internID, f.Title, f.Criterion, f.DueWeek, f.OKRID, by).Scan(&id)
	if err != nil {
		return nil, constraintErr(err)
	}
	g, _, err := r.Get(ctx, id)
	return g, err
}

// Update меняет текст, срок и OKR цели.
func (r *Goals) Update(ctx context.Context, id string, f models.GoalFields) (*models.Goal, error) {
	err := execOne(r.db.Exec(ctx, `
		UPDATE intern_goals SET title = $2, criterion = $3, due_week = $4, okr_id = $5::uuid, updated_at = now()
		WHERE id::text = $1`, id, f.Title, f.Criterion, f.DueWeek, f.OKRID))
	if err != nil {
		return nil, constraintErr(err)
	}
	g, _, err := r.Get(ctx, id)
	return g, err
}

// Delete удаляет цель.
func (r *Goals) Delete(ctx context.Context, id string) error {
	return execOne(r.db.Exec(ctx, `DELETE FROM intern_goals WHERE id::text = $1`, id))
}

// SetProgress сохраняет прогресс, пишет историю, событие для метрик и уведомление наставнику
// (если отметил не он сам). Все в одной транзакции.
func (r *Goals) SetProgress(ctx context.Context, goalID, authorID string, pct int, note string) (*models.Goal, error) {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var internID, title string
		err := tx.QueryRow(ctx, `
			UPDATE intern_goals SET progress_pct = $2, updated_at = now()
			WHERE id::text = $1
			RETURNING intern_id::text, title`, goalID, pct).Scan(&internID, &title)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO goal_updates (goal_id, author_id, progress_pct, note) VALUES ($1::uuid, $2::uuid, $3, $4)`,
			goalID, authorID, pct, note); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO events (user_id, event_type, entity_kind, entity_id) VALUES ($1::uuid, 'goal_progress', 'goal', $2)`,
			authorID, goalID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO notifications (user_id, kind, text, link)
			SELECT p.mentor_id, 'goals', u.name || ' обновил(а) цель «' || $2::text || '»: ' || $3::int || '%', '/mentor'
			FROM intern_profiles p JOIN users u ON u.id = p.user_id
			WHERE p.user_id::text = $1 AND p.mentor_id IS NOT NULL AND p.mentor_id::text <> $4`,
			internID, title, pct, authorID)
		return err
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	g, _, err := r.Get(ctx, goalID)
	return g, err
}

// Agree отмечает, что руководитель согласовал цели стажера.
func (r *Goals) Agree(ctx context.Context, internID, by string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := execOne(tx.Exec(ctx, `
			UPDATE intern_profiles SET goals_agreed_by = $2::uuid, goals_agreed_at = now()
			WHERE user_id::text = $1`, internID, by)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO events (user_id, event_type, entity_kind, entity_id) VALUES ($1::uuid, 'goals_agreed', 'intern', $2)`,
			by, internID)
		return err
	})
}

func scanGoal(row pgx.Row) (models.Goal, string, error) {
	var g models.Goal
	var internID string
	var okrID, okrPeriod, okrTitle, cuID, cuName, luAuthor, luName, luNote *string
	var luPct *int
	var luAt *time.Time
	err := row.Scan(&g.ID, &internID, &g.Position, &g.Title, &g.Criterion, &g.DueWeek, &g.ProgressPct, &g.UpdatedAt,
		&okrID, &okrPeriod, &okrTitle, &cuID, &cuName, &luAuthor, &luName, &luPct, &luNote, &luAt)
	if err != nil {
		return g, "", err
	}
	if okrID != nil {
		g.OKR = &models.TeamOKR{ID: *okrID, Period: *okrPeriod, Title: *okrTitle}
	}
	if cuID != nil {
		g.CreatedBy = &models.PersonRef{ID: *cuID, Name: *cuName}
	}
	if luAuthor != nil {
		g.LastUpdate = &models.GoalNote{
			Author: models.PersonRef{ID: *luAuthor, Name: *luName}, ProgressPct: *luPct, Note: *luNote, At: *luAt,
		}
	}
	return g, internID, nil
}

package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/models"
)

// Requests – заявки на доступ и кадровые заявки, справочник сервисов и правила доступа.
type Requests struct {
	db *pgxpool.Pool
}

// NewRequests создает репозиторий заявок.
func NewRequests(db *pgxpool.Pool) *Requests {
	return &Requests{db: db}
}

// ---------- справочник сервисов ----------

// ResolveService находит сервис по тексту вопроса или коду: самый длинный синоним, который
// встречается в тексте. Не нашли – ErrNotFound.
func (r *Requests) ResolveService(ctx context.Context, text string) (string, error) {
	var code string
	err := r.db.QueryRow(ctx, `
		SELECT code FROM (
			SELECT code, 1000 AS w FROM services WHERE code = lower($1) OR lower(title) = lower($1)
			UNION ALL
			SELECT service_code, length(alias) FROM service_aliases WHERE lower($1) LIKE '%' || alias || '%'
		) x ORDER BY w DESC LIMIT 1`, text).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return code, err
}

// Service – сервис по коду.
func (r *Requests) Service(ctx context.Context, code string) (*models.Service, error) {
	var s models.Service
	err := r.db.QueryRow(ctx, `
		SELECT code, title, request_title, approver_kind, approver_id::text, sla_hours FROM services WHERE code = $1`, code,
	).Scan(&s.Code, &s.Title, &s.RequestTitle, &s.ApproverKind, &s.ApproverID, &s.SLAHours)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ServiceRoles – роли сервиса по возрастанию уровня прав с решением политики для роли пользователя.
func (r *Requests) ServiceRoles(ctx context.Context, service, userRole string) ([]models.ServiceRole, error) {
	rows, err := r.db.Query(ctx, `
		SELECT sr.code, sr.title, sr.access_level, l.title, COALESCE(p.allowed, false), COALESCE(p.reason, '')
		FROM service_roles sr
		JOIN access_levels l ON l.code = sr.access_level
		LEFT JOIN service_access_policies p
		       ON p.service_code = sr.service_code AND p.service_role = sr.code AND p.user_role = $2
		WHERE sr.service_code = $1
		ORDER BY l.rank, sr.code`, service, userRole)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.ServiceRole, error) {
		var x models.ServiceRole
		err := row.Scan(&x.Code, &x.Title, &x.Level, &x.LevelTitle, &x.Allowed, &x.Reason)
		return x, err
	})
}

// TeamProject – проект сервиса, закрепленный за командой (первый по коду).
func (r *Requests) TeamProject(ctx context.Context, service, teamID string) (*models.Project, error) {
	return r.project(ctx, `WHERE service_code = $1 AND team_id::text = $2 ORDER BY code LIMIT 1`, service, teamID)
}

// Project – проект по коду внутри сервиса.
func (r *Requests) Project(ctx context.Context, service, code string) (*models.Project, error) {
	return r.project(ctx, `WHERE service_code = $1 AND (code = $2 OR lower(title) = lower($2))`, service, code)
}

func (r *Requests) project(ctx context.Context, where string, args ...any) (*models.Project, error) {
	var p models.Project
	err := r.db.QueryRow(ctx, `SELECT code, title, owner_id::text FROM projects `+where, args...).Scan(&p.Code, &p.Title, &p.OwnerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// TeamLead – руководитель команды.
func (r *Requests) TeamLead(ctx context.Context, teamID string) (*models.PersonRef, error) {
	var p models.PersonRef
	err := r.db.QueryRow(ctx, `SELECT id::text, name FROM users WHERE team_id::text = $1 AND role = 'lead'`, teamID).Scan(&p.ID, &p.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Person – имя пользователя по id.
func (r *Requests) Person(ctx context.Context, id string) (*models.PersonRef, error) {
	var p models.PersonRef
	err := r.db.QueryRow(ctx, `SELECT id::text, name FROM users WHERE id::text = $1`, id).Scan(&p.ID, &p.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// LinkedAccessTask – незавершенная задача пути стажера с действием «заявка на этот сервис».
// Если указан taskID, проверяется именно она (и что она из шаблона команды стажера).
func (r *Requests) LinkedAccessTask(ctx context.Context, internID, service string, taskID *string) (*models.LinkedTask, error) {
	var t models.LinkedTask
	err := r.db.QueryRow(ctx, `
		SELECT t.id::text, t.title
		FROM onboarding_tasks t
		JOIN onboarding_stages s ON s.id = t.stage_id
		JOIN users u ON u.team_id = s.team_id AND u.id::text = $1
		LEFT JOIN task_actions a ON a.task_id = t.id
		LEFT JOIN intern_task_progress p ON p.task_id = t.id AND p.intern_id = u.id
		WHERE COALESCE(p.status, 'todo') <> 'done'
		  AND (($3::text IS NOT NULL AND t.id::text = $3)
		       OR ($3::text IS NULL AND a.kind = 'access_request' AND a.service_code = $2))
		ORDER BY s.position, t.position LIMIT 1`, internID, service, taskID).Scan(&t.ID, &t.Title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// OpenRequest – уже поданная (не завершенная) заявка стажера на этот сервис.
func (r *Requests) OpenRequest(ctx context.Context, requesterID, service string) (*models.Request, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		SELECT q.id::text FROM requests q JOIN access_request_details d ON d.request_id = q.id
		WHERE q.requester_id::text = $1 AND d.service_code = $2 AND q.status IN ('waiting', 'approved', 'done')
		ORDER BY q.created_at DESC LIMIT 1`, requesterID, service).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

// ResolveItem находит позицию каталога отделов по тексту: самый длинный синоним в тексте,
// код или название. Не нашли – ErrNotFound.
func (r *Requests) ResolveItem(ctx context.Context, text string) (string, error) {
	var code string
	err := r.db.QueryRow(ctx, `
		SELECT code FROM (
			SELECT code, 1000 AS w FROM catalog_items WHERE code = lower($1) OR lower(title) = lower($1)
			UNION ALL
			SELECT item_code, length(alias) FROM catalog_aliases WHERE lower($1) LIKE '%' || alias || '%'
		) x ORDER BY w DESC LIMIT 1`, text).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return code, err
}

// CatalogItem – позиция каталога с отделом.
func (r *Requests) CatalogItem(ctx context.Context, code string) (*models.CatalogItem, error) {
	var c models.CatalogItem
	err := r.db.QueryRow(ctx, `
		SELECT ci.code, ci.title, ci.request_title, ci.hint, d.code, d.title, d.sla_hours
		FROM catalog_items ci JOIN departments d ON d.code = ci.department_code
		WHERE ci.code = $1`, code,
	).Scan(&c.Code, &c.Title, &c.RequestTitle, &c.Hint, &c.DepartmentCode, &c.DepartmentTitle, &c.SLAHours)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// OpenServiceRequest – заявка в отдел на эту позицию, которая еще в работе.
func (r *Requests) OpenServiceRequest(ctx context.Context, requesterID, item string) (*models.Request, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		SELECT q.id::text FROM requests q JOIN service_request_details d ON d.request_id = q.id
		WHERE q.requester_id::text = $1 AND d.item_code = $2 AND q.status = 'waiting'
		ORDER BY q.created_at DESC LIMIT 1`, requesterID, item).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

// Notify – уведомление пользователю (колокольчик), например наставнику о заявке стажера.
func (r *Requests) Notify(ctx context.Context, userID, kind, text, link string, requestID *string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO notifications (user_id, kind, text, link, request_id) VALUES ($1::uuid, $2, $3, $4, $5::uuid)`,
		userID, kind, text, link, requestID)
	return err
}

// ---------- запись ----------

// CreateServiceDraft создает черновик заявки в отдел (тип other, согласующего-человека нет).
func (r *Requests) CreateServiceDraft(ctx context.Context, d models.ServiceDraft) (*models.Request, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO requests (type, requester_id, status, via) VALUES ('other', $1::uuid, 'draft', 'buddy')
			RETURNING id::text`, d.RequesterID).Scan(&id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO service_request_details (request_id, item_code, comment) VALUES ($1::uuid, $2, $3)`,
			id, d.Item, d.Comment); err != nil {
			return err
		}
		return addEvent(ctx, tx, id, "draft", "buddy", nil, "Подготовлена AI-buddy")
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	return r.Get(ctx, id)
}

// CreateAccessDraft создает черновик заявки на доступ (ждет подтверждения стажера).
func (r *Requests) CreateAccessDraft(ctx context.Context, d models.AccessDraft) (*models.Request, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO requests (type, requester_id, status, via, task_id, approver_id)
			VALUES ('access', $1::uuid, 'draft', 'buddy', $2::uuid, $3::uuid) RETURNING id::text`,
			d.RequesterID, d.TaskID, d.ApproverID).Scan(&id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO access_request_details (request_id, service_code, service_role, project_code, justification)
			VALUES ($1::uuid, $2, $3, $4, $5)`, id, d.Service, d.Role, d.Project, d.Justification); err != nil {
			return err
		}
		return addEvent(ctx, tx, id, "draft", "buddy", nil, "Подготовлена AI-buddy")
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	return r.Get(ctx, id)
}

// CreateLeaveDraft создает черновик кадровой заявки.
func (r *Requests) CreateLeaveDraft(ctx context.Context, d models.LeaveDraft) (*models.Request, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO requests (type, requester_id, status, via, task_id, approver_id)
			VALUES ('leave', $1::uuid, 'draft', 'buddy', $2::uuid, $3::uuid) RETURNING id::text`,
			d.RequesterID, d.TaskID, d.ApproverID).Scan(&id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO leave_request_details (request_id, leave_date, leave_type, comment) VALUES ($1::uuid, $2, $3, $4)`,
			id, d.Date, d.Type, d.Comment); err != nil {
			return err
		}
		return addEvent(ctx, tx, id, "draft", "buddy", nil, "Подготовлена AI-buddy")
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	return r.Get(ctx, id)
}

// Submit – стажер подтвердил: заявка уходит во внешнюю систему. status – waiting или итоговый
// (выдано автоматически). Связанная задача пути переходит в waiting или done.
func (r *Requests) Submit(ctx context.Context, id, actorID, status, externalRef, taskNote string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var requester string
		var taskID *string
		err := tx.QueryRow(ctx, `
			UPDATE requests SET status = $2, external_ref = $3,
			       decided_at = CASE WHEN $2 IN ('approved', 'done') THEN now() END
			WHERE id::text = $1 AND status = 'draft'
			RETURNING requester_id::text, task_id::text`, id, status, externalRef).Scan(&requester, &taskID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := addEvent(ctx, tx, id, "waiting", "user", &actorID, "Подтверждена стажером"); err != nil {
			return err
		}
		if status != "waiting" {
			if err := addEvent(ctx, tx, id, status, "service", nil, "Выдано автоматически"); err != nil {
				return err
			}
		}
		return updateTask(ctx, tx, requester, taskID, status, taskNote)
	})
}

// Cancel – стажер отказался от черновика.
func (r *Requests) Cancel(ctx context.Context, id, actorID string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := execOne(tx.Exec(ctx, `UPDATE requests SET status = 'cancelled' WHERE id::text = $1 AND status = 'draft'`, id)); err != nil {
			return err
		}
		return addEvent(ctx, tx, id, "cancelled", "user", &actorID, "Отменена стажером")
	})
}

// ApplyDecision – решение внешней системы: статус, история, задача пути, уведомление и сообщение
// помощника в чат стажера. Возвращает false, если заявка уже не ждала решения.
func (r *Requests) ApplyDecision(ctx context.Context, id, status string, reason *string, taskNote, notice string) (bool, error) {
	changed := false
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var requester string
		var taskID *string
		err := tx.QueryRow(ctx, `
			UPDATE requests SET status = $2, reject_reason = $3, decided_at = now()
			WHERE id::text = $1 AND status = 'waiting'
			RETURNING requester_id::text, task_id::text`, id, status, reason).Scan(&requester, &taskID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		changed = true
		note := ""
		if reason != nil {
			note = *reason
		}
		if err := addEvent(ctx, tx, id, status, "service", nil, note); err != nil {
			return err
		}
		if err := updateTask(ctx, tx, requester, taskID, status, taskNote); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO notifications (user_id, kind, text, link, request_id) VALUES ($1::uuid, 'requests', $2, '/requests', $3::uuid)`,
			requester, notice, id); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			WITH c AS (
				INSERT INTO conversations (kind, intern_id) VALUES ('buddy', $1::uuid)
				ON CONFLICT (intern_id) WHERE kind = 'buddy' DO UPDATE SET kind = 'buddy'
				RETURNING id
			)
			INSERT INTO messages (conversation_id, author_kind, kind, text, mood, request_id)
			SELECT c.id, 'assistant', 'result', $2, CASE WHEN $4 = 'rejected' THEN 'sad' ELSE 'happy' END, $3::uuid FROM c`,
			requester, notice, id, status)
		return err
	})
	return changed, err
}

// updateTask переводит задачу пути вслед за заявкой: waiting – «на согласовании»,
// done/approved – выполнена, rejected – снова «в работе» с причиной.
func updateTask(ctx context.Context, tx pgx.Tx, internID string, taskID *string, status, note string) error {
	if taskID == nil {
		return nil
	}
	taskStatus := map[string]string{"waiting": "waiting", "approved": "done", "done": "done", "rejected": "in_progress"}[status]
	if taskStatus == "" {
		return nil
	}
	var n *string
	if note != "" {
		n = &note
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO intern_task_progress (intern_id, task_id, status, note, updated_by, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $1::uuid, now())
		ON CONFLICT (intern_id, task_id)
		DO UPDATE SET status = EXCLUDED.status, note = EXCLUDED.note, updated_by = EXCLUDED.updated_by, updated_at = now()`,
		internID, *taskID, taskStatus, n)
	return err
}

func addEvent(ctx context.Context, tx pgx.Tx, id, status, actorKind string, actorID *string, note string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO request_events (request_id, status, actor_kind, actor_id, note) VALUES ($1::uuid, $2, $3, $4::uuid, $5)`,
		id, status, actorKind, actorID, note)
	return err
}

// ---------- чтение ----------

const requestSelect = `
	SELECT q.id::text, q.number, q.type, q.status, q.via, q.requester_id::text, q.task_id::text,
	       ap.id::text, ap.name, q.external_ref, q.reject_reason, q.created_at, q.decided_at,
	       COALESCE(s.code, ''), COALESCE(s.title, ''), COALESCE(s.request_title, ''), COALESCE(s.approver_kind, 'lead'),
	       COALESCE(s.sla_hours, 24), COALESCE(sr.title, ''), COALESCE(al.code, ''), COALESCE(al.title, ''),
	       pr.title, COALESCE(d.justification, ''),
	       l.leave_date, COALESCE(lt.title, ''), COALESCE(l.comment, ''),
	       COALESCE(ci.code, ''), COALESCE(ci.title, ''), COALESCE(ci.request_title, ''),
	       COALESCE(dep.title, ''), COALESCE(dep.sla_hours, 24), COALESCE(sd.comment, '')
	FROM requests q
	LEFT JOIN users ap ON ap.id = q.approver_id
	LEFT JOIN access_request_details d ON d.request_id = q.id
	LEFT JOIN services s ON s.code = d.service_code
	LEFT JOIN service_roles sr ON sr.service_code = d.service_code AND sr.code = d.service_role
	LEFT JOIN access_levels al ON al.code = sr.access_level
	LEFT JOIN projects pr ON pr.code = d.project_code
	LEFT JOIN leave_request_details l ON l.request_id = q.id
	LEFT JOIN leave_types lt ON lt.code = l.leave_type
	LEFT JOIN service_request_details sd ON sd.request_id = q.id
	LEFT JOIN catalog_items ci ON ci.code = sd.item_code
	LEFT JOIN departments dep ON dep.code = ci.department_code`

func scanRequest(row pgx.Row) (*models.Request, error) {
	var q models.Request
	var apID, apName *string
	var leaveDate *time.Time
	var itemRequestTitle string
	var depSLA int
	err := row.Scan(&q.ID, &q.Number, &q.Type, &q.Status, &q.Via, &q.RequesterID, &q.TaskID,
		&apID, &apName, &q.ExternalRef, &q.RejectReason, &q.CreatedAt, &q.DecidedAt,
		&q.ServiceCode, &q.ServiceTitle, &q.RequestTitle, &q.ApproverKind,
		&q.SLAHours, &q.RoleTitle, &q.Level, &q.LevelTitle,
		&q.ProjectTitle, &q.Justification,
		&leaveDate, &q.LeaveTypeTitle, &q.LeaveComment,
		&q.ItemCode, &q.ItemTitle, &itemRequestTitle, &q.DepartmentTitle, &depSLA, &q.ItemComment)
	if err != nil {
		return nil, err
	}
	if q.ItemCode != "" {
		q.RequestTitle, q.SLAHours, q.ApproverKind = itemRequestTitle, depSLA, "department"
	}
	if apID != nil {
		q.Approver = &models.PersonRef{ID: *apID, Name: *apName}
	}
	q.LeaveDate = leaveDate
	return &q, nil
}

// Get – заявка по id.
func (r *Requests) Get(ctx context.Context, id string) (*models.Request, error) {
	q, err := scanRequest(r.db.QueryRow(ctx, requestSelect+` WHERE q.id::text = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return q, err
}

// ListByRequester – заявки пользователя без черновиков и отмененных, новые сверху.
func (r *Requests) ListByRequester(ctx context.Context, requesterID string) ([]*models.Request, error) {
	rows, err := r.db.Query(ctx, requestSelect+`
		WHERE q.requester_id::text = $1 AND q.status NOT IN ('draft', 'cancelled')
		ORDER BY q.created_at DESC, q.number DESC`, requesterID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*models.Request, error) { return scanRequest(row) })
}

// Waiting – заявки, которые ждут решения внешней системы.
func (r *Requests) Waiting(ctx context.Context, limit int) ([]*models.Request, error) {
	rows, err := r.db.Query(ctx, requestSelect+`
		WHERE q.status = 'waiting' AND q.external_ref IS NOT NULL ORDER BY q.created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*models.Request, error) { return scanRequest(row) })
}

// Events – история заявок по порядку.
func (r *Requests) Events(ctx context.Context, ids []string) ([]models.RequestEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.request_id::text, e.status, e.actor_kind, u.name, e.note, e.created_at
		FROM request_events e LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.request_id::text = ANY($1)
		ORDER BY e.created_at, e.id`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.RequestEvent, error) {
		var e models.RequestEvent
		err := row.Scan(&e.RequestID, &e.Status, &e.ActorKind, &e.ActorName, &e.Note, &e.At)
		return e, err
	})
}

package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/models"
)

// Staff – лента наставника, согласования, чек-лист наставника, конспект встреч и обращения в поддержку.
type Staff struct {
	db *pgxpool.Pool
}

// NewStaff создает репозиторий.
func NewStaff(db *pgxpool.Pool) *Staff {
	return &Staff{db: db}
}

// ---------- лента ----------

// PushToMentor добавляет запись в ленту наставника стажера. Без наставника ничего не делает;
// наставнику не пишем о его собственных действиях.
func (r *Staff) PushToMentor(ctx context.Context, internID string, n models.Notice) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO notifications (user_id, kind, text, link, request_id, actor_id, actor_kind, actor_name)
		SELECT p.mentor_id, $2, $3, NULLIF($4, ''), $5::uuid, $6::uuid, $7, $8
		FROM intern_profiles p
		WHERE p.user_id::text = $1 AND p.mentor_id IS NOT NULL
		  AND ($6::uuid IS NULL OR p.mentor_id <> $6::uuid)`,
		internID, n.Kind, n.Text, n.Link, n.RequestID, n.ActorID, n.ActorKind, n.ActorName)
	return err
}

// Feed – последние записи ленты пользователя и число непрочитанных.
func (r *Staff) Feed(ctx context.Context, userID string, limit int) (*models.Feed, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, actor_name, actor_kind, text, created_at, read_at IS NOT NULL
		FROM notifications WHERE user_id::text = $1
		ORDER BY created_at DESC, id LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.FeedItem, error) {
		var f models.FeedItem
		err := row.Scan(&f.ID, &f.Actor.Name, &f.Actor.Kind, &f.Text, &f.At, &f.Read)
		return f, err
	})
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []models.FeedItem{}
	}
	feed := &models.Feed{Items: items}
	err = r.db.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id::text = $1 AND read_at IS NULL`, userID).Scan(&feed.Unread)
	return feed, err
}

// ReadFeed помечает все записи ленты прочитанными.
func (r *Staff) ReadFeed(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = now() WHERE user_id::text = $1 AND read_at IS NULL`, userID)
	return err
}

// ---------- согласования ----------

// Approvals – заявки, где пользователь согласующий: сначала ожидающие, затем новые сверху.
func (r *Staff) Approvals(ctx context.Context, approverID string) ([]*models.Request, error) {
	rows, err := r.db.Query(ctx, requestSelect+`
		WHERE q.approver_id::text = $1 AND q.status IN ('waiting', 'approved', 'done', 'rejected')
		ORDER BY (q.status = 'waiting') DESC, q.created_at DESC LIMIT 100`, approverID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*models.Request, error) { return scanRequest(row) })
}

// Names – имена пользователей по id.
func (r *Staff) Names(ctx context.Context, ids []string) (map[string]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id::text, name FROM users WHERE id::text = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

// ---------- чек-лист наставника ----------

// Checklist – пункты чек-листа с отметками наставника по стажеру.
func (r *Staff) Checklist(ctx context.Context, mentorID, internID string) ([]models.ChecklistItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.id::text, i.title, i.hint, i.from_week, i.period, m.item_id IS NOT NULL
		FROM mentor_checklist_items i
		LEFT JOIN mentor_checklist_marks m
		       ON m.item_id = i.id AND m.mentor_id::text = $1 AND m.intern_id::text = $2
		ORDER BY i.position`, mentorID, internID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.ChecklistItem, error) {
		var c models.ChecklistItem
		err := row.Scan(&c.ID, &c.Title, &c.Hint, &c.FromWeek, &c.Period, &c.Done)
		return c, err
	})
}

// MarkChecklist ставит или снимает отметку. ErrNotFound – нет такого пункта.
func (r *Staff) MarkChecklist(ctx context.Context, mentorID, internID, itemID string, done bool) error {
	if !done {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM mentor_checklist_items WHERE id::text = $1)`, itemID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		_, err := r.db.Exec(ctx, `
			DELETE FROM mentor_checklist_marks WHERE mentor_id::text = $1 AND intern_id::text = $2 AND item_id::text = $3`,
			mentorID, internID, itemID)
		return err
	}
	return execOne(r.db.Exec(ctx, `
		INSERT INTO mentor_checklist_marks (mentor_id, intern_id, item_id)
		SELECT $1::uuid, $2::uuid, i.id FROM mentor_checklist_items i WHERE i.id::text = $3
		ON CONFLICT (mentor_id, intern_id, item_id) DO UPDATE SET done_at = mentor_checklist_marks.done_at`,
		mentorID, internID, itemID))
}

// ---------- конспект встреч ----------

// Notes – заметки стажера, свежие сверху.
func (r *Staff) Notes(ctx context.Context, internID string) ([]models.MeetingNote, error) {
	rows, err := r.db.Query(ctx, `
		SELECT event_id::text, text, updated_at FROM meeting_notes WHERE intern_id::text = $1
		ORDER BY updated_at DESC`, internID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.MeetingNote, error) {
		var n models.MeetingNote
		err := row.Scan(&n.EventID, &n.Text, &n.UpdatedAt)
		return n, err
	})
	if out == nil {
		out = []models.MeetingNote{}
	}
	return out, err
}

// OwnEvent – событие есть в календаре стажера: встреча с его участием или своя запись.
func (r *Staff) OwnEvent(ctx context.Context, internID, eventID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM meeting_participants WHERE user_id::text = $1 AND meeting_id::text = $2)
		    OR EXISTS (SELECT 1 FROM calendar_entries WHERE intern_id::text = $1 AND id::text = $2)`,
		internID, eventID).Scan(&ok)
	return ok, err
}

// SaveNote сохраняет заметку; пустой текст удаляет ее.
func (r *Staff) SaveNote(ctx context.Context, internID, eventID, text string) (*models.MeetingNote, error) {
	n := &models.MeetingNote{EventID: eventID}
	if text == "" {
		_, err := r.db.Exec(ctx, `DELETE FROM meeting_notes WHERE intern_id::text = $1 AND event_id::text = $2`, internID, eventID)
		if err != nil {
			return nil, err
		}
		return n, r.db.QueryRow(ctx, `SELECT now()`).Scan(&n.UpdatedAt)
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO meeting_notes (intern_id, event_id, text) VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (intern_id, event_id) DO UPDATE SET text = EXCLUDED.text, updated_at = now()
		RETURNING text, updated_at`, internID, eventID, text).Scan(&n.Text, &n.UpdatedAt)
	return n, err
}

// ---------- обращения в поддержку ----------

const ticketSelect = `
	SELECT id::text, COALESCE(external_ref, 'SUP-' || number), problem_type, text, status, created_at, answer
	FROM support_tickets`

func scanTicket(row pgx.Row) (models.SupportTicket, error) {
	var t models.SupportTicket
	err := row.Scan(&t.ID, &t.Number, &t.Type, &t.Details, &t.Status, &t.CreatedAt, &t.Answer)
	return t, err
}

// CreateTicket сохраняет обращение. С attachDialog к нему привязывается открытый диалог с AI-buddy.
func (r *Staff) CreateTicket(ctx context.Context, d models.SupportDraft) (*models.SupportTicket, error) {
	var ref *string
	if d.ExternalRef != "" {
		ref = &d.ExternalRef
	}
	t, err := scanTicket(r.db.QueryRow(ctx, `
		INSERT INTO support_tickets (user_id, problem_type, text, intern_id, attach_dialog, external_ref, dialog_id)
		VALUES ($1::uuid, $2, $3, $4::uuid, $5, $6,
		        CASE WHEN $5 THEN (
		            SELECT d.id FROM chat_dialogs d JOIN conversations c ON c.id = d.conversation_id
		            WHERE c.kind = 'buddy' AND c.intern_id = $1::uuid AND d.closed_at IS NULL LIMIT 1) END)
		RETURNING id::text, COALESCE(external_ref, 'SUP-' || number), problem_type, text, status, created_at, answer`,
		d.UserID, d.Type, d.Details, d.InternID, d.AttachDialog, ref))
	if err != nil {
		return nil, constraintErr(err)
	}
	return &t, nil
}

// Tickets – обращения пользователя, новые сверху.
func (r *Staff) Tickets(ctx context.Context, userID string) ([]models.SupportTicket, error) {
	rows, err := r.db.Query(ctx, ticketSelect+` WHERE user_id::text = $1 ORDER BY created_at DESC, number DESC`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.SupportTicket, error) { return scanTicket(row) })
	if out == nil {
		out = []models.SupportTicket{}
	}
	return out, err
}

// OpenTicket – обращение, которое еще не решено во внешнем сервисе.
type OpenTicket struct {
	ID, Ref, Status, UserID string
}

// OpenTickets – нерешенные обращения с ключом во внешнем сервисе.
func (r *Staff) OpenTickets(ctx context.Context, limit int) ([]OpenTicket, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, external_ref, status, user_id::text FROM support_tickets
		WHERE status <> 'done' AND external_ref IS NOT NULL ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (OpenTicket, error) {
		var t OpenTicket
		err := row.Scan(&t.ID, &t.Ref, &t.Status, &t.UserID)
		return t, err
	})
}

// UpdateTicket меняет статус обращения; при решении пишет ответ и уведомляет автора.
func (r *Staff) UpdateTicket(ctx context.Context, id, status string, answer *string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var userID, number string
		err := tx.QueryRow(ctx, `
			UPDATE support_tickets SET status = $2, answer = COALESCE($3, answer), updated_at = now()
			WHERE id::text = $1 AND status <> $2
			RETURNING user_id::text, COALESCE(external_ref, 'SUP-' || number)`, id, status, answer).Scan(&userID, &number)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil || status != "done" {
			return err
		}
		text := "Обращение " + number + " решено"
		if answer != nil && *answer != "" {
			text += ": " + *answer
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO notifications (user_id, kind, text, link, actor_kind, actor_name)
			VALUES ($1::uuid, 'tips', $2, '/support', 'other', 'Поддержка')`, userID, text)
		return err
	})
}

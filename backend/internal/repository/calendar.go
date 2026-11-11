package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/models"
)

// Calendar – календарь стажера: встречи, где он участник, и его собственные записи.
type Calendar struct {
	db *pgxpool.Pool
}

// NewCalendar создает репозиторий календаря.
func NewCalendar(db *pgxpool.Pool) *Calendar {
	return &Calendar{db: db}
}

// MeetingRow – встреча стажера с данными для карточки.
type MeetingRow struct {
	ID, Title, Kind  string
	HostID, HostName string
	HostRole, Team   string
	StartsAt         time.Time
	DurationMin      int
	Place, Note      string
	Answer           *string // accept | move | nil
}

const meetingSelect = `
	SELECT m.id::text, m.title, m.kind, h.id::text, h.name, h.role, t.name, m.starts_at, m.duration_min, m.place, m.note, a.answer
	FROM meetings m
	JOIN meeting_participants mp ON mp.meeting_id = m.id AND mp.user_id::text = $1
	JOIN users h ON h.id = m.host_id
	JOIN teams t ON t.id = h.team_id
	LEFT JOIN meeting_answers a ON a.meeting_id = m.id AND a.intern_id::text = $1
	WHERE m.status = 'scheduled'`

func scanMeeting(row pgx.Row) (MeetingRow, error) {
	var m MeetingRow
	err := row.Scan(&m.ID, &m.Title, &m.Kind, &m.HostID, &m.HostName, &m.HostRole, &m.Team,
		&m.StartsAt, &m.DurationMin, &m.Place, &m.Note, &m.Answer)
	return m, err
}

// Meetings – встречи стажера, начинающиеся в [from, to).
func (r *Calendar) Meetings(ctx context.Context, internID string, from, to time.Time) ([]MeetingRow, error) {
	rows, err := r.db.Query(ctx, meetingSelect+` AND m.starts_at >= $2 AND m.starts_at < $3 ORDER BY m.starts_at`,
		internID, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (MeetingRow, error) { return scanMeeting(row) })
}

// Meeting – одна встреча стажера. Не участник или встреча отменена – ErrNotFound.
func (r *Calendar) Meeting(ctx context.Context, internID, meetingID string) (*MeetingRow, error) {
	m, err := scanMeeting(r.db.QueryRow(ctx, meetingSelect+` AND m.id::text = $2`, internID, meetingID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// EntryRow – собственная запись стажера.
type EntryRow struct {
	ID, Title, Kind string
	StartsAt        time.Time
	DurationMin     int
	Shared          bool
}

// Entries – записи стажера, начинающиеся в [from, to).
func (r *Calendar) Entries(ctx context.Context, internID string, from, to time.Time) ([]EntryRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, title, kind, starts_at, duration_min, shared FROM calendar_entries
		WHERE intern_id::text = $1 AND starts_at >= $2 AND starts_at < $3 ORDER BY starts_at`, internID, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (EntryRow, error) {
		var e EntryRow
		err := row.Scan(&e.ID, &e.Title, &e.Kind, &e.StartsAt, &e.DurationMin, &e.Shared)
		return e, err
	})
}

// Overlap – название первого события стажера (встречи или записи), которое пересекается с [start, end).
// Пустая строка – пересечений нет.
func (r *Calendar) Overlap(ctx context.Context, internID string, start, end time.Time) (string, error) {
	var title string
	err := r.db.QueryRow(ctx, `
		SELECT title FROM (
			SELECT m.title, m.starts_at FROM meetings m
			JOIN meeting_participants mp ON mp.meeting_id = m.id AND mp.user_id::text = $1
			WHERE m.status = 'scheduled' AND m.starts_at < $3 AND m.starts_at + make_interval(mins => m.duration_min) > $2
			UNION ALL
			SELECT e.title, e.starts_at FROM calendar_entries e
			WHERE e.intern_id::text = $1 AND e.starts_at < $3 AND e.starts_at + make_interval(mins => e.duration_min) > $2
		) x ORDER BY starts_at LIMIT 1`, internID, start, end).Scan(&title)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return title, err
}

// CreateEntry сохраняет запись стажера.
func (r *Calendar) CreateEntry(ctx context.Context, e models.CalendarEntry) (*EntryRow, error) {
	var out EntryRow
	err := r.db.QueryRow(ctx, `
		INSERT INTO calendar_entries (intern_id, title, kind, starts_at, duration_min, shared)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		RETURNING id::text, title, kind, starts_at, duration_min, shared`,
		e.InternID, e.Title, e.Kind, e.StartsAt, e.DurationMin, e.Shared,
	).Scan(&out.ID, &out.Title, &out.Kind, &out.StartsAt, &out.DurationMin, &out.Shared)
	if err != nil {
		return nil, constraintErr(err)
	}
	return &out, nil
}

// DeleteEntry удаляет свою запись. Чужая или несуществующая – ErrNotFound.
func (r *Calendar) DeleteEntry(ctx context.Context, internID, id string) error {
	return execOne(r.db.Exec(ctx, `DELETE FROM calendar_entries WHERE id::text = $1 AND intern_id::text = $2`, id, internID))
}

// Answer сохраняет ответ стажера на встречу и уведомляет организатора. При просьбе перенести
// сообщение от стажера уходит в его переписку с наставником (mentorID), если наставник есть.
func (r *Calendar) Answer(ctx context.Context, m MeetingRow, internID, internName, answer, reason string, mentorID *string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO meeting_answers (meeting_id, intern_id, answer, reason) VALUES ($1::uuid, $2::uuid, $3, $4)
			ON CONFLICT (meeting_id, intern_id) DO UPDATE SET answer = EXCLUDED.answer, reason = EXCLUDED.reason, created_at = now()`,
			m.ID, internID, answer, reason); err != nil {
			return err
		}
		note := internName + " придет на «" + m.Title + "»"
		if answer == "move" {
			note = internName + " просит перенести «" + m.Title + "»"
			if reason != "" {
				note += ": " + reason
			}
		}
		if mentorID == nil || *mentorID != m.HostID || answer == "accept" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO notifications (user_id, kind, text, link) VALUES ($1::uuid, 'meetings', $2, '/schedule')`,
				m.HostID, note); err != nil {
				return err
			}
		}
		if answer != "move" || mentorID == nil {
			return nil
		}
		text := "Можно перенести «" + m.Title + "»?"
		if reason != "" {
			text += " " + reason
		}
		var conv string
		if err := tx.QueryRow(ctx, `
			INSERT INTO conversations (kind, intern_id, mentor_id) VALUES ('mentor', $1::uuid, $2::uuid)
			ON CONFLICT (intern_id, mentor_id) WHERE kind = 'mentor' DO UPDATE SET kind = 'mentor'
			RETURNING id::text`, internID, *mentorID).Scan(&conv); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO messages (conversation_id, author_kind, author_id, kind, text) VALUES ($1::uuid, 'user', $2::uuid, 'text', $3)`,
			conv, internID, text)
		return err
	})
}

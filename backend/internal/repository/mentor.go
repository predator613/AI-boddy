package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/models"
)

// Mentor – переписка наставника со стажером, материалы, расписание, автоматизации и статистика.
type Mentor struct {
	db *pgxpool.Pool
}

// NewMentor создает репозиторий.
func NewMentor(db *pgxpool.Pool) *Mentor {
	return &Mentor{db: db}
}

// ---------- переписка ----------

// Conversation – id переписки стажера с наставником (создается при первом обращении).
func (r *Mentor) Conversation(ctx context.Context, internID, mentorID string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO conversations (kind, intern_id, mentor_id) VALUES ('mentor', $1::uuid, $2::uuid)
		ON CONFLICT (intern_id, mentor_id) WHERE kind = 'mentor' DO UPDATE SET kind = 'mentor'
		RETURNING id::text`, internID, mentorID).Scan(&id)
	return id, err
}

// ThreadMessages – сообщения переписки по времени. Переданный AI-buddy вопрос – kind handoff.
func (r *Mentor) ThreadMessages(ctx context.Context, conv string) ([]models.ThreadMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id::text, m.kind, m.author_kind, COALESCE(u.name, 'AI-buddy'), m.text, m.created_at,
		       h.id IS NOT NULL, m.update_type, m.update_title, m.update_detail, m.update_link, m.acked_at IS NOT NULL
		FROM messages m
		LEFT JOIN users u ON u.id = m.author_id
		LEFT JOIN handoffs h ON h.forwarded_id = m.id
		WHERE m.conversation_id::text = $1
		ORDER BY m.created_at, m.id`, conv)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.ThreadMessage, error) {
		var m models.ThreadMessage
		var kind, author string
		var handoff, acked bool
		var uType, uTitle, uDetail *string
		var uLink *string
		if err := row.Scan(&m.ID, &kind, &author, &m.AuthorName, &m.Text, &m.At, &handoff, &uType, &uTitle, &uDetail, &uLink, &acked); err != nil {
			return m, err
		}
		m.Kind, m.Author = "text", map[string]string{"user": "intern", "mentor": "mentor"}[author]
		if m.Author == "" {
			m.Author, m.AuthorName = "buddy", "AI-buddy"
		}
		switch {
		case handoff:
			m.Kind, m.Author, m.AuthorName = "handoff", "buddy", "AI-buddy"
		case kind == "update" && uType != nil:
			m.Kind = "update"
			m.Update = &models.ThreadUpdate{Type: *uType, Title: deref(uTitle), Detail: deref(uDetail), Link: uLink, Acked: acked}
		}
		return m, nil
	})
	if out == nil {
		out = []models.ThreadMessage{}
	}
	return out, err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Post – сообщение в переписку: author – user (стажер) или mentor.
func (r *Mentor) Post(ctx context.Context, conv, authorKind, authorID, text string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO messages (conversation_id, author_kind, author_id, kind, text) VALUES ($1::uuid, $2, $3::uuid, 'text', $4)`,
		conv, authorKind, authorID, text)
	return err
}

// AddUpdate – карточка изменения от наставника в переписке со стажером.
func (r *Mentor) AddUpdate(ctx context.Context, internID, mentorID, kind, title, detail string, link *string) error {
	conv, err := r.Conversation(ctx, internID, mentorID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO messages (conversation_id, author_kind, author_id, kind, text, update_type, update_title, update_detail, update_link)
		VALUES ($1::uuid, 'mentor', $2::uuid, 'update', $3, $4, $3, $5, $6)`,
		conv, mentorID, title, kind, detail, link)
	return err
}

// Ack – стажер отметил карточку изменения. ErrNotFound – в его переписке такой карточки нет.
func (r *Mentor) Ack(ctx context.Context, conv, messageID string) error {
	return execOne(r.db.Exec(ctx, `
		UPDATE messages SET acked_at = COALESCE(acked_at, now())
		WHERE id::text = $1 AND conversation_id::text = $2 AND kind = 'update'`, messageID, conv))
}

// MarkRead – пользователь прочитал переписку.
func (r *Mentor) MarkRead(ctx context.Context, conv, userID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO conversation_reads (conversation_id, user_id, last_read_at) VALUES ($1::uuid, $2::uuid, now())
		ON CONFLICT (conversation_id, user_id) DO UPDATE SET last_read_at = now()`, conv, userID)
	return err
}

// Summaries – переписки со стажерами: последнее сообщение и непрочитанные от стажера (и AI-buddy).
func (r *Mentor) Summaries(ctx context.Context, mentorID string, internIDs []string) (map[string]models.ThreadSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.intern_id::text, COALESCE(l.text, ''), COALESCE(l.created_at, c.created_at),
		       (SELECT count(*) FROM messages m WHERE m.conversation_id = c.id AND m.author_kind <> 'mentor'
		          AND m.created_at > COALESCE((SELECT last_read_at FROM conversation_reads cr
		                                      WHERE cr.conversation_id = c.id AND cr.user_id = $1::uuid), '-infinity'))::int
		FROM conversations c
		LEFT JOIN LATERAL (SELECT text, created_at FROM messages m WHERE m.conversation_id = c.id
		                   ORDER BY created_at DESC, id DESC LIMIT 1) l ON true
		WHERE c.kind = 'mentor' AND c.mentor_id = $1::uuid AND c.intern_id::text = ANY($2)`, mentorID, internIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]models.ThreadSummary{}
	for rows.Next() {
		var s models.ThreadSummary
		if err := rows.Scan(&s.InternID, &s.LastText, &s.LastAt, &s.Unread); err != nil {
			return nil, err
		}
		out[s.InternID] = s
	}
	return out, rows.Err()
}

// AnswerHandoff – ответ наставника на переданный вопрос: копия ответа уходит стажеру в чат с AI-buddy.
// articleID – статья справочника, если наставник сохранил ответ. false – открытых вопросов не было.
func (r *Mentor) AnswerHandoff(ctx context.Context, conv, internID, mentorName, answer string, articleID *string) (bool, error) {
	answered := false
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var hid string
		err := tx.QueryRow(ctx, `
			SELECT h.id::text FROM handoffs h JOIN messages f ON f.id = h.forwarded_id
			WHERE f.conversation_id::text = $1 AND h.answered_at IS NULL
			ORDER BY h.created_at DESC LIMIT 1 FOR UPDATE OF h`, conv).Scan(&hid)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var msgID string
		err = tx.QueryRow(ctx, `
			WITH c AS (
				INSERT INTO conversations (kind, intern_id) VALUES ('buddy', $1::uuid)
				ON CONFLICT (intern_id) WHERE kind = 'buddy' DO UPDATE SET kind = 'buddy'
				RETURNING id
			)
			INSERT INTO messages (conversation_id, author_kind, kind, text, mood)
			SELECT c.id, 'assistant', 'text', $2, 'mentor' FROM c RETURNING id::text`,
			internID, mentorName+" ответил на твой вопрос: "+answer).Scan(&msgID)
		if err != nil {
			return err
		}
		answered = true
		_, err = tx.Exec(ctx, `UPDATE handoffs SET answer_id = $2::uuid, answer_article_id = $3::uuid, answered_at = now() WHERE id::text = $1`,
			hid, msgID, articleID)
		return err
	})
	return answered, err
}

// LastHandoffQuestion – текст последнего переданного вопроса без ответа (для заголовка статьи).
func (r *Mentor) LastHandoffQuestion(ctx context.Context, conv string) (string, error) {
	var q string
	err := r.db.QueryRow(ctx, `
		SELECT f.text FROM handoffs h JOIN messages f ON f.id = h.forwarded_id
		WHERE f.conversation_id::text = $1 AND h.answered_at IS NULL ORDER BY h.created_at DESC LIMIT 1`, conv).Scan(&q)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return q, err
}

// ---------- материалы ----------

const materialSelect = `
	SELECT a.id::text, a.title, a.source, f.filename, f.size_bytes, a.url,
	       CASE WHEN st.id IS NOT NULL THEN to_char(st.position, 'FM00') END, sec.title,
	       a.use_in_ai, COALESCE(u.name, ''), a.created_at
	FROM kb_articles a
	JOIN kb_sections sec ON sec.id = a.section_id
	LEFT JOIN kb_files f ON f.article_id = a.id
	LEFT JOIN onboarding_stages st ON st.id = a.stage_id
	LEFT JOIN users u ON u.id = a.author_id`

func scanMaterial(row pgx.Row) (models.Material, error) {
	var m models.Material
	var source string
	var file, url *string
	var size *int64
	if err := row.Scan(&m.ID, &m.Title, &source, &file, &size, &url, &m.StageCode, &m.Section, &m.UseInAI, &m.AuthorName, &m.CreatedAt); err != nil {
		return m, err
	}
	switch {
	case source == "mentor_answer":
		m.Kind, m.SizeLabel = "answer", "ответ в чате"
	case source == "link":
		m.Kind, m.SizeLabel = "link", "ссылка"
	default:
		m.Kind = "md"
		if file != nil {
			for _, k := range []string{"pdf", "docx"} {
				if len(*file) > len(k) && (*file)[len(*file)-len(k):] == k {
					m.Kind = k
				}
			}
		}
		m.SizeLabel = sizeLabel(size)
	}
	return m, nil
}

func sizeLabel(n *int64) string {
	switch {
	case n == nil:
		return "файл"
	case *n < 1024:
		return fmt.Sprintf("%d Б", *n)
	case *n < 1024*1024:
		return fmt.Sprintf("%d КБ", (*n+1023)/1024)
	}
	return fmt.Sprintf("%.1f МБ", float64(*n)/(1024*1024))
}

// Materials – материалы команды: загруженные файлы, ссылки и ответы наставников.
func (r *Mentor) Materials(ctx context.Context, teamID string) ([]models.Material, error) {
	rows, err := r.db.Query(ctx, materialSelect+`
		WHERE a.team_id::text = $1 AND a.source IN ('upload', 'link', 'mentor_answer')
		ORDER BY a.created_at DESC, a.id`, teamID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Material, error) { return scanMaterial(row) })
	if out == nil {
		out = []models.Material{}
	}
	return out, err
}

// StageByCode – этап команды по номеру «02».
func (r *Mentor) StageByCode(ctx context.Context, teamID, code string) (*string, error) {
	var id string
	err := r.db.QueryRow(ctx, `SELECT id::text FROM onboarding_stages WHERE team_id::text = $1 AND to_char(position, 'FM00') = $2`,
		teamID, code).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &id, err
}

// CreateMaterial сохраняет материал статьей справочника в разделе (новый раздел создается) и
// режет текст на фрагменты для поиска, если материал доступен AI-buddy.
func (r *Mentor) CreateMaterial(ctx context.Context, d models.MaterialDraft, source string) (*models.Material, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var section string
		err := tx.QueryRow(ctx, `SELECT id::text FROM kb_sections WHERE lower(title) = lower($1)`, d.Section).Scan(&section)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `
				INSERT INTO kb_sections (code, title, hand, color, position)
				VALUES ('s-' || substr(md5($1), 1, 8), $1, 'материалы', '#FFFFFF', (SELECT COALESCE(max(position), 0) + 1 FROM kb_sections))
				RETURNING id::text`, d.Section).Scan(&section)
		}
		if err != nil {
			return err
		}
		var url *string
		if d.URL != "" {
			url = &d.URL
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO kb_articles (section_id, team_id, stage_id, title, summary, body, source, url, author_id, use_in_ai)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9::uuid, $10) RETURNING id::text`,
			section, d.TeamID, d.StageID, d.Title, clipText(d.Body, 200), d.Body, source, url, d.AuthorID, d.UseInAI).Scan(&id); err != nil {
			return err
		}
		if d.FileName != "" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO kb_files (article_id, filename, mime_type, size_bytes, sha256, content) VALUES ($1::uuid, $2, $3, $4, $5, $6)`,
				id, d.FileName, d.MimeType, len(d.Content), d.SHA256, d.Content); err != nil {
				return err
			}
		}
		if !d.UseInAI {
			return nil
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO kb_chunks (article_id, chunk_no, content)
			SELECT $1::uuid, p.ord, p.part FROM regexp_split_to_table($2, E'\n\n') WITH ORDINALITY AS p(part, ord)
			WHERE btrim(p.part) <> ''`, id, d.Title+"\n\n"+d.Body)
		return err
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	m, err := scanMaterial(r.db.QueryRow(ctx, materialSelect+` WHERE a.id::text = $1`, id))
	return &m, err
}

func clipText(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "…"
}

// DeleteMaterial удаляет материал команды. ErrNotFound – нет такого или он не из материалов.
func (r *Mentor) DeleteMaterial(ctx context.Context, teamID, id string) error {
	return execOne(r.db.Exec(ctx, `
		DELETE FROM kb_articles WHERE id::text = $1 AND team_id::text = $2 AND source IN ('upload', 'link', 'mentor_answer')`, id, teamID))
}

// ---------- расписание ----------

const mentorMeetingSelect = `
	SELECT m.id::text, m.title, m.kind, m.starts_at, m.duration_min,
	       ARRAY(SELECT p.user_id::text FROM meeting_participants p WHERE p.meeting_id = m.id ORDER BY p.user_id),
	       ARRAY(SELECT u.name FROM meeting_participants p JOIN users u ON u.id = p.user_id WHERE p.meeting_id = m.id ORDER BY p.user_id)
	FROM meetings m`

// MeetingRow – встреча наставника с участниками.
type MentorMeetingRow struct {
	models.MentorMeeting
	Participants []string
	Names        []string
}

func scanMentorMeeting(row pgx.Row) (MentorMeetingRow, error) {
	var m MentorMeetingRow
	var kind string
	err := row.Scan(&m.ID, &m.Title, &kind, &m.StartsAt, &m.DurationMin, &m.Participants, &m.Names)
	m.Kind = kind
	if kind != "review" && kind != "team" {
		m.Kind = "one_on_one"
	}
	switch len(m.Participants) {
	case 1:
		m.InternID, m.WithName = &m.Participants[0], m.Names[0]
	default:
		m.WithName = "все стажеры"
		if len(m.Participants) > 1 {
			m.Kind = "team"
		}
	}
	return m, err
}

// Meetings – встречи наставника в интервале.
func (r *Mentor) Meetings(ctx context.Context, hostID string, from, to time.Time) ([]MentorMeetingRow, error) {
	rows, err := r.db.Query(ctx, mentorMeetingSelect+`
		WHERE m.host_id::text = $1 AND m.status = 'scheduled' AND m.starts_at >= $2 AND m.starts_at < $3
		ORDER BY m.starts_at, m.id`, hostID, from, to)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (MentorMeetingRow, error) { return scanMentorMeeting(row) })
	if out == nil {
		out = []MentorMeetingRow{}
	}
	return out, err
}

// Meeting – встреча наставника. ErrNotFound – чужая, отмененная или нет такой.
func (r *Mentor) Meeting(ctx context.Context, hostID, id string) (*MentorMeetingRow, error) {
	m, err := scanMentorMeeting(r.db.QueryRow(ctx, mentorMeetingSelect+`
		WHERE m.id::text = $1 AND m.host_id::text = $2 AND m.status = 'scheduled'`, id, hostID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

// CreateMeeting – встреча наставника с одним или несколькими стажерами.
func (r *Mentor) CreateMeeting(ctx context.Context, hostID, title, kind string, startsAt time.Time, durationMin int, interns []string) (*MentorMeetingRow, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO meetings (host_id, title, starts_at, kind, duration_min) VALUES ($1::uuid, $2, $3, $4, $5) RETURNING id::text`,
			hostID, title, startsAt, kind, durationMin).Scan(&id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO meeting_participants (meeting_id, user_id) SELECT $1::uuid, unnest($2::uuid[])`, id, interns)
		return err
	})
	if err != nil {
		return nil, constraintErr(err)
	}
	return r.Meeting(ctx, hostID, id)
}

// MoveMeeting переносит встречу; ответы стажеров сбрасываются.
func (r *Mentor) MoveMeeting(ctx context.Context, hostID, id string, startsAt time.Time) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := execOne(tx.Exec(ctx, `UPDATE meetings SET starts_at = $3 WHERE id::text = $1 AND host_id::text = $2 AND status = 'scheduled'`,
			id, hostID, startsAt)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM meeting_answers WHERE meeting_id::text = $1`, id)
		return err
	})
}

// CancelMeeting отменяет встречу.
func (r *Mentor) CancelMeeting(ctx context.Context, hostID, id string) error {
	return execOne(r.db.Exec(ctx, `UPDATE meetings SET status = 'cancelled' WHERE id::text = $1 AND host_id::text = $2 AND status = 'scheduled'`,
		id, hostID))
}

// ---------- автоматизации ----------

// Automations – включенные правила наставника: kind → enabled (без записи – по умолчанию).
func (r *Mentor) Automations(ctx context.Context, mentorID string) (map[string]bool, error) {
	rows, err := r.db.Query(ctx, `SELECT kind, enabled FROM mentor_automations WHERE mentor_id::text = $1`, mentorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		var e bool
		if err := rows.Scan(&k, &e); err != nil {
			return nil, err
		}
		out[k] = e
	}
	return out, rows.Err()
}

// SetAutomation включает или выключает правило.
func (r *Mentor) SetAutomation(ctx context.Context, mentorID, kind string, enabled bool) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO mentor_automations (mentor_id, kind, enabled) VALUES ($1::uuid, $2, $3)
		ON CONFLICT (mentor_id, kind) DO UPDATE SET enabled = EXCLUDED.enabled`, mentorID, kind, enabled)
	return err
}

// RunAutomations – один проход правил. disabled – выключенные правила у наставников; уведомление
// пишется один раз на объект и день (automation_runs). Возвращает число новых уведомлений.
func (r *Mentor) RunAutomations(ctx context.Context, now time.Time) (int, error) {
	total := 0
	for _, q := range automationQueries {
		tag, err := r.db.Exec(ctx, q, now)
		if err != nil {
			return total, err
		}
		total += int(tag.RowsAffected())
	}
	return total, nil
}

// enabledFor – правило включено у наставника стажера (без записи – включено).
const enabledFor = `COALESCE((SELECT enabled FROM mentor_automations ma WHERE ma.mentor_id = p.mentor_id AND ma.kind = '%s'), true)`

// dueTasks – задачи стажеров с датой срока по Москве.
const dueTasks = `
	SELECT p.user_id AS intern_id, p.mentor_id, u.name AS intern_name, t.id AS task_id, t.title,
	       (p.start_date + t.due_day - 1) AS due
	FROM intern_profiles p JOIN users u ON u.id = p.user_id
	JOIN onboarding_stages s ON s.team_id = u.team_id
	JOIN onboarding_tasks t ON t.stage_id = s.id AND t.due_day IS NOT NULL
	LEFT JOIN intern_task_progress pr ON pr.intern_id = p.user_id AND pr.task_id = t.id
	WHERE COALESCE(pr.status, 'todo') <> 'done'`

var automationQueries = []string{
	// Напоминание стажеру за день до срока.
	`WITH d AS (` + dueTasks + ` AND (p.start_date + t.due_day - 1) = ($1::timestamptz AT TIME ZONE 'Europe/Moscow')::date + 1
	               AND p.mentor_id IS NOT NULL AND ` + fmt.Sprintf(enabledFor, "deadline_reminder") + `),
	 r AS (INSERT INTO automation_runs (kind, key) SELECT 'deadline_reminder', d.intern_id || ':' || d.task_id || ':' || d.due FROM d
	       ON CONFLICT DO NOTHING RETURNING key)
	 INSERT INTO notifications (user_id, kind, text, link, actor_kind, actor_name)
	 SELECT d.intern_id, 'tasks', 'Завтра срок задачи «' || d.title || '»', '/', 'buddy', 'AI-buddy'
	 FROM d JOIN r ON r.key = d.intern_id || ':' || d.task_id || ':' || d.due`,
	// Наставнику: задача стажера просрочена больше чем на 2 дня.
	`WITH d AS (` + dueTasks + ` AND (p.start_date + t.due_day - 1) < ($1::timestamptz AT TIME ZONE 'Europe/Moscow')::date - 2
	               AND p.mentor_id IS NOT NULL AND ` + fmt.Sprintf(enabledFor, "overdue_alert") + `),
	 r AS (INSERT INTO automation_runs (kind, key) SELECT 'overdue_alert', d.intern_id || ':' || d.task_id FROM d
	       ON CONFLICT DO NOTHING RETURNING key)
	 INSERT INTO notifications (user_id, kind, text, link, actor_id, actor_kind, actor_name)
	 SELECT d.mentor_id, 'tasks', 'У ' || d.intern_name || ' просрочена задача «' || d.title || '»', '/interns/' || d.intern_id,
	        NULL, 'buddy', 'AI-buddy'
	 FROM d JOIN r ON r.key = d.intern_id || ':' || d.task_id`,
	// Согласующему: заявка стажера ждет больше 3 дней (напоминание раз в день).
	`WITH d AS (SELECT q.id, q.approver_id, q.number, u.name, ip.mentor_id
	            FROM requests q JOIN users u ON u.id = q.requester_id JOIN intern_profiles ip ON ip.user_id = q.requester_id
	            WHERE q.status = 'waiting' AND q.approver_id IS NOT NULL AND q.created_at < $1::timestamptz - interval '3 days'
	              AND COALESCE((SELECT enabled FROM mentor_automations ma WHERE ma.mentor_id = ip.mentor_id AND ma.kind = 'approver_reminder'), true)),
	 r AS (INSERT INTO automation_runs (kind, key)
	       SELECT 'approver_reminder', d.id || ':' || ($1::timestamptz AT TIME ZONE 'Europe/Moscow')::date FROM d
	       ON CONFLICT DO NOTHING RETURNING key)
	 INSERT INTO notifications (user_id, kind, text, link, request_id, actor_kind, actor_name)
	 SELECT d.approver_id, 'requests', 'Заявка №' || d.number || ' от ' || d.name || ' ждет решения больше 3 дней', '/approvals', d.id,
	        'buddy', 'AI-buddy'
	 FROM d JOIN r ON r.key = d.id || ':' || ($1::timestamptz AT TIME ZONE 'Europe/Moscow')::date`,
	// Пятница после 17:00: чек-ин по целям стажеру.
	`WITH d AS (SELECT p.user_id FROM intern_profiles p
	            WHERE extract(isodow FROM $1::timestamptz AT TIME ZONE 'Europe/Moscow') = 5
	              AND extract(hour FROM $1::timestamptz AT TIME ZONE 'Europe/Moscow') >= 17
	              AND p.mentor_id IS NOT NULL AND ` + fmt.Sprintf(enabledFor, "weekly_goal_checkin") + `
	              AND EXISTS (SELECT 1 FROM intern_goals g WHERE g.intern_id = p.user_id)),
	 r AS (INSERT INTO automation_runs (kind, key)
	       SELECT 'weekly_goal_checkin', d.user_id || ':' || to_char($1::timestamptz AT TIME ZONE 'Europe/Moscow', 'IYYY-IW') FROM d
	       ON CONFLICT DO NOTHING RETURNING key)
	 INSERT INTO notifications (user_id, kind, text, link, actor_kind, actor_name)
	 SELECT d.user_id, 'goals', 'Пятничный чек-ин: отметь, что получилось по целям за неделю', '/', 'buddy', 'AI-buddy'
	 FROM d JOIN r ON r.key = d.user_id || ':' || to_char($1::timestamptz AT TIME ZONE 'Europe/Moscow', 'IYYY-IW')`,
}

// ---------- статистика ----------

// Activity – активность стажеров за период: вопросы AI-buddy, передачи наставнику, время наставника.
type Activity struct {
	Questions, HandedOff, MentorMinutes int
	AccessHours                         *float64
	OnTimePercent                       *int
}

// Activity считает показатели по стажерам internIDs с момента since. hostID – чье время считаем.
func (r *Mentor) Activity(ctx context.Context, hostID string, internIDs []string, since time.Time) (*Activity, error) {
	var a Activity
	err := r.db.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM messages m JOIN conversations c ON c.id = m.conversation_id
		   WHERE c.kind = 'buddy' AND c.intern_id::text = ANY($2) AND m.author_kind = 'user' AND m.created_at >= $3)::int,
		  (SELECT count(*) FROM handoffs h JOIN messages f ON f.id = h.forwarded_id JOIN conversations c ON c.id = f.conversation_id
		   WHERE c.intern_id::text = ANY($2) AND h.created_at >= $3)::int,
		  (COALESCE((SELECT sum(duration_min) FROM meetings m WHERE m.host_id::text = $1 AND m.status = 'scheduled'
		             AND m.starts_at >= $3 AND m.starts_at < now()), 0)
		   + 5 * (SELECT count(*) FROM messages m JOIN conversations c ON c.id = m.conversation_id
		          WHERE c.kind = 'mentor' AND m.author_id::text = $1 AND m.created_at >= $3))::int,
		  (SELECT round(avg(extract(epoch FROM q.decided_at - q.created_at) / 3600)::numeric, 1)::float8
		   FROM requests q WHERE q.type = 'access' AND q.status = 'done' AND q.decided_at IS NOT NULL
		     AND q.requester_id::text = ANY($2)),
		  (SELECT round(100.0 * count(*) FILTER (WHERE (pr.updated_at AT TIME ZONE 'Europe/Moscow')::date <= p.start_date + t.due_day - 1)
		                / NULLIF(count(*), 0))::int
		   FROM intern_task_progress pr JOIN onboarding_tasks t ON t.id = pr.task_id
		   JOIN intern_profiles p ON p.user_id = pr.intern_id
		   WHERE pr.status = 'done' AND t.due_day IS NOT NULL AND pr.intern_id::text = ANY($2))`,
		hostID, internIDs, since).Scan(&a.Questions, &a.HandedOff, &a.MentorMinutes, &a.AccessHours, &a.OnTimePercent)
	return &a, err
}

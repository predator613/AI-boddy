package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Chat – переписка с помощником, передача вопросов наставнику и журнал вызовов инструментов.
type Chat struct {
	db *pgxpool.Pool
}

// NewChat создает репозиторий чата.
func NewChat(db *pgxpool.Pool) *Chat {
	return &Chat{db: db}
}

// NewMessage – сообщение для записи.
type NewMessage struct {
	ConversationID string
	AuthorKind     string // user | assistant | mentor | system
	AuthorID       *string
	Kind           string // text | confirm | result | error
	Text           string
	Mood           *string
	RequestID      *string
	ArticleID      *string // источник ответа: статья
	SourceLabel    *string // или подпись («Мой путь → этап 02»)
}

// StoredMessage – сообщение из базы.
type StoredMessage struct {
	ID          string
	AuthorKind  string
	Kind        string
	Text        string
	Mood        *string
	RequestID   *string
	ArticleID   *string
	SourceLabel *string
	SourceTitle *string
	CreatedAt   time.Time
}

// BuddyConversation – переписка стажера с помощником (создается при первом обращении).
func (r *Chat) BuddyConversation(ctx context.Context, internID string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO conversations (kind, intern_id) VALUES ('buddy', $1::uuid)
		ON CONFLICT (intern_id) WHERE kind = 'buddy' DO UPDATE SET kind = 'buddy'
		RETURNING id::text`, internID).Scan(&id)
	return id, err
}

// AddMessage записывает сообщение и его источник.
func (r *Chat) AddMessage(ctx context.Context, m NewMessage) (string, time.Time, error) {
	var id string
	var at time.Time
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO messages (conversation_id, author_kind, author_id, kind, text, mood, request_id)
			VALUES ($1::uuid, $2, $3::uuid, $4, $5, $6, $7::uuid)
			RETURNING id::text, created_at`,
			m.ConversationID, m.AuthorKind, m.AuthorID, m.Kind, m.Text, m.Mood, m.RequestID).Scan(&id, &at); err != nil {
			return err
		}
		if m.ArticleID == nil && m.SourceLabel == nil {
			return nil
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO message_sources (message_id, position, article_id, label) VALUES ($1::uuid, 1, $2::uuid, $3)`,
			id, m.ArticleID, m.SourceLabel)
		return err
	})
	return id, at, err
}

// Messages – последние сообщения переписки в хронологическом порядке.
func (r *Chat) Messages(ctx context.Context, conversationID string, limit int) ([]StoredMessage, error) {
	return r.messages(ctx, `m.dialog_id = (SELECT id FROM chat_dialogs WHERE conversation_id::text = $1 AND closed_at IS NULL)`,
		limit, conversationID)
}

// DialogMessages – сообщения одного диалога этой переписки. Чужой или несуществующий диалог – ErrNotFound.
func (r *Chat) DialogMessages(ctx context.Context, conversationID, dialogID string, limit int) ([]StoredMessage, error) {
	var ok bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM chat_dialogs WHERE id::text = $1 AND conversation_id::text = $2)`,
		dialogID, conversationID).Scan(&ok); err != nil || !ok {
		if err == nil {
			err = ErrNotFound
		}
		return nil, err
	}
	return r.messages(ctx, `m.dialog_id::text = $1`, limit, dialogID)
}

func (r *Chat) messages(ctx context.Context, where string, limit int, arg string) ([]StoredMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT * FROM (
			SELECT m.id::text, m.author_kind, m.kind, m.text, m.mood, m.request_id::text,
			       s.article_id::text, s.label, a.title, m.created_at
			FROM messages m
			LEFT JOIN message_sources s ON s.message_id = m.id AND s.position = 1
			LEFT JOIN kb_articles a ON a.id = s.article_id
			WHERE `+where+`
			ORDER BY m.created_at DESC, m.id DESC LIMIT $2
		) x ORDER BY created_at, id`, arg, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StoredMessage, error) {
		var m StoredMessage
		err := row.Scan(&m.ID, &m.AuthorKind, &m.Kind, &m.Text, &m.Mood, &m.RequestID,
			&m.ArticleID, &m.SourceLabel, &m.SourceTitle, &m.CreatedAt)
		return m, err
	})
}

// DialogRow – диалог для колонки истории. Заголовок и итог считаются по сообщениям.
type DialogRow struct {
	ID        string
	Title     string // первая реплика стажера
	Kind      string // request | handoff | answer
	Preview   string
	UpdatedAt time.Time
	Current   bool
}

// Dialogs – диалоги переписки, новые сверху: открытый и прошлые, в которых были сообщения.
func (r *Chat) Dialogs(ctx context.Context, conversationID string, limit int) ([]DialogRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id::text,
		       COALESCE((SELECT left(m.text, 60) FROM messages m
		                 WHERE m.dialog_id = d.id AND m.author_kind = 'user' ORDER BY m.created_at LIMIT 1), ''),
		       q.number, q.status,
		       EXISTS (SELECT 1 FROM handoffs h JOIN messages m ON m.id = h.question_id WHERE m.dialog_id = d.id),
		       EXISTS (SELECT 1 FROM message_sources s JOIN messages m ON m.id = s.message_id
		               WHERE m.dialog_id = d.id AND s.article_id IS NOT NULL),
		       COALESCE((SELECT max(m.created_at) FROM messages m WHERE m.dialog_id = d.id), d.started_at),
		       d.closed_at IS NULL
		FROM chat_dialogs d
		LEFT JOIN LATERAL (
			SELECT r.number, r.status FROM messages m JOIN requests r ON r.id = m.request_id
			WHERE m.dialog_id = d.id AND m.kind = 'result' ORDER BY m.created_at DESC LIMIT 1
		) q ON true
		WHERE d.conversation_id::text = $1
		  AND (d.closed_at IS NULL OR EXISTS (SELECT 1 FROM messages m WHERE m.dialog_id = d.id))
		ORDER BY d.closed_at IS NULL DESC, 7 DESC
		LIMIT $2`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (DialogRow, error) {
		var d DialogRow
		var number *int
		var status *string
		var handoff, fromKB bool
		err := row.Scan(&d.ID, &d.Title, &number, &status, &handoff, &fromKB, &d.UpdatedAt, &d.Current)
		switch {
		case number != nil:
			d.Kind = "request"
			d.Preview = fmt.Sprintf("Заявка №%d ", *number) + map[string]string{
				"waiting": "создана", "approved": "одобрена", "done": "выполнена", "rejected": "отклонена", "cancelled": "отменена",
			}[*status]
		case handoff:
			d.Kind, d.Preview = "handoff", "Передано наставнику"
		case fromKB:
			d.Kind, d.Preview = "answer", "Ответ из справочника"
		default:
			d.Kind, d.Preview = "answer", "Разговор с AI-buddy"
		}
		if d.Title == "" {
			d.Title = "Новый диалог"
		}
		return d, err
	})
}

// Clear – «Новый диалог»: открытый диалог закрывается (сообщения остаются в истории) и открывается
// следующий. Пустой диалог не закрываем. Неподтвержденные карточки закрытого диалога отменяются.
func (r *Chat) Clear(ctx context.Context, conversationID string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var open string
		err := tx.QueryRow(ctx, `
			SELECT d.id::text FROM chat_dialogs d
			WHERE d.conversation_id::text = $1 AND d.closed_at IS NULL
			  AND EXISTS (SELECT 1 FROM messages m WHERE m.dialog_id = d.id)
			FOR UPDATE`, conversationID).Scan(&open)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			WITH c AS (
				UPDATE requests SET status = 'cancelled'
				WHERE status = 'draft' AND id IN (
					SELECT request_id FROM messages WHERE dialog_id::text = $1 AND kind = 'confirm' AND request_id IS NOT NULL)
				RETURNING id, requester_id
			)
			INSERT INTO request_events (request_id, status, actor_kind, actor_id, note)
			SELECT id, 'cancelled', 'user', requester_id, 'Новый диалог: черновик отменен' FROM c`, open); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE chat_dialogs SET closed_at = clock_timestamp() WHERE id::text = $1`, open); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO chat_dialogs (conversation_id) VALUES ($1::uuid)`, conversationID)
		return err
	})
}

// Handoff передает вопрос наставнику: копия вопроса уходит в переписку стажера с наставником.
func (r *Chat) Handoff(ctx context.Context, questionID, internID, mentorID, text string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var conv, fwd string
		if err := tx.QueryRow(ctx, `
			INSERT INTO conversations (kind, intern_id, mentor_id) VALUES ('mentor', $1::uuid, $2::uuid)
			ON CONFLICT (intern_id, mentor_id) WHERE kind = 'mentor' DO UPDATE SET kind = 'mentor'
			RETURNING id::text`, internID, mentorID).Scan(&conv); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO messages (conversation_id, author_kind, author_id, kind, text)
			VALUES ($1::uuid, 'user', $2::uuid, 'text', $3) RETURNING id::text`, conv, internID, text).Scan(&fwd); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO handoffs (question_id, forwarded_id) VALUES ($1::uuid, $2::uuid)
			ON CONFLICT (question_id) DO NOTHING`, questionID, fwd)
		return err
	})
}

// ToolCall – запись журнала вызова модели или инструмента.
type ToolCall struct {
	ConversationID string
	MessageID      *string
	Server         string
	Tool           string
	Status         string // ok | error | denied | timeout
	Input          any
	Output         any
	Latency        time.Duration
}

// LogToolCall пишет журнал. Ошибка журнала не должна ломать ответ, поэтому она только возвращается.
func (r *Chat) LogToolCall(ctx context.Context, c ToolCall) error {
	in, _ := json.Marshal(c.Input)
	out, _ := json.Marshal(c.Output)
	_, err := r.db.Exec(ctx, `
		INSERT INTO agent_tool_calls (conversation_id, message_id, server, tool, status, input, output, latency_ms)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8)`,
		c.ConversationID, c.MessageID, c.Server, c.Tool, c.Status, in, out, c.Latency.Milliseconds())
	return err
}

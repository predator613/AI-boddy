package repository

import (
	"context"
	"encoding/json"
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
	rows, err := r.db.Query(ctx, `
		SELECT * FROM (
			SELECT m.id::text, m.author_kind, m.kind, m.text, m.mood, m.request_id::text,
			       s.article_id::text, s.label, a.title, m.created_at
			FROM messages m
			LEFT JOIN message_sources s ON s.message_id = m.id AND s.position = 1
			LEFT JOIN kb_articles a ON a.id = s.article_id
			WHERE m.conversation_id::text = $1
			ORDER BY m.created_at DESC, m.id DESC LIMIT $2
		) x ORDER BY created_at, id`, conversationID, limit)
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

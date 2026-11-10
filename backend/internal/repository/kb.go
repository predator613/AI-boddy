package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/models"
)

// KB – запросы справочника. Везде teamID – команда пользователя: статьи с team_id = NULL видят все,
// остальные – только своя команда. Фильтр стоит до ранжирования, чужое не попадает даже в подсчеты.
type KB struct {
	db *pgxpool.Pool
}

// NewKB создает репозиторий справочника.
func NewKB(db *pgxpool.Pool) *KB {
	return &KB{db: db}
}

// Время чтения: 180 слов в минуту, минимум 1 минута. Не храним – считаем.
const kbMinutes = `GREATEST(1, ceil(array_length(regexp_split_to_array(btrim(a.body), '\s+'), 1) / 180.0))::int`

// Номер этапа – только если этап принадлежит команде пользователя.
const kbStageCode = `CASE WHEN st.team_id = $1::uuid THEN to_char(st.position, 'FM00') END`

const kbVisible = `(a.team_id IS NULL OR a.team_id = $1::uuid)`

// Sections – все разделы с числом видимых статей.
func (r *KB) Sections(ctx context.Context, teamID string) ([]models.KBSection, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.code, s.title, s.hand, s.color,
		       (SELECT count(*) FROM kb_articles a WHERE a.section_id = s.id AND `+kbVisible+`)::int
		FROM kb_sections s
		ORDER BY s.position, s.code`, teamID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.KBSection, error) {
		var s models.KBSection
		err := row.Scan(&s.ID, &s.Title, &s.Hand, &s.Color, &s.Count)
		return s, err
	})
}

// Articles – видимые статьи без текста, по разделам.
func (r *KB) Articles(ctx context.Context, teamID string) ([]models.KBArticle, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text, s.code, a.title, a.summary, `+kbMinutes+`, `+kbStageCode+`
		FROM kb_articles a
		JOIN kb_sections s ON s.id = a.section_id
		LEFT JOIN onboarding_stages st ON st.id = a.stage_id
		WHERE `+kbVisible+`
		ORDER BY s.position, a.created_at, a.id`, teamID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.KBArticle, error) {
		var a models.KBArticle
		err := row.Scan(&a.ID, &a.SectionID, &a.Title, &a.Summary, &a.Minutes, &a.StageCode)
		return a, err
	})
}

// Terms – словарик: общие слова и слова команды.
func (r *KB) Terms(ctx context.Context, teamID string) ([]models.KBTerm, error) {
	rows, err := r.db.Query(ctx, `
		SELECT term, definition FROM kb_terms
		WHERE team_id IS NULL OR team_id = $1::uuid
		ORDER BY lower(term)`, teamID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.KBTerm, error) {
		var t models.KBTerm
		err := row.Scan(&t.Term, &t.Def)
		return t, err
	})
}

// Article – статья целиком. Чужая или несуществующая – ErrNotFound.
func (r *KB) Article(ctx context.Context, teamID, id string) (*models.KBArticleDetail, error) {
	var a models.KBArticleDetail
	err := r.db.QueryRow(ctx, `
		SELECT a.id::text, s.code, a.title, a.summary, `+kbMinutes+`, `+kbStageCode+`,
		       a.body, a.source, a.url, u.name, a.use_in_ai, a.updated_at
		FROM kb_articles a
		JOIN kb_sections s ON s.id = a.section_id
		LEFT JOIN onboarding_stages st ON st.id = a.stage_id
		LEFT JOIN users u ON u.id = a.author_id
		WHERE a.id::text = $2 AND `+kbVisible, teamID, id,
	).Scan(&a.ID, &a.SectionID, &a.Title, &a.Summary, &a.Minutes, &a.StageCode,
		&a.Body, &a.Source, &a.URL, &a.AuthorName, &a.UseInAI, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Search – полнотекстовый поиск (русская морфология). Запрос понимается как «любое из слов»,
// совпадение в заголовке весит вдвое больше, чем в тексте. Из каждой статьи берется лучший фрагмент.
// onlyAI – искать только в материалах, по которым разрешено отвечать помощнику.
func (r *KB) Search(ctx context.Context, teamID, query string, limit int, onlyAI bool) ([]models.KBSearchMatch, error) {
	rows, err := r.db.Query(ctx, `
		WITH q AS (
			SELECT replace(plainto_tsquery('russian', $2)::text, '&', '|')::tsquery AS q
		),
		vis AS (
			SELECT a.* FROM kb_articles a
			WHERE `+kbVisible+` AND (NOT $4 OR a.use_in_ai)
		),
		hits AS (
			SELECT v.id, 2 * ts_rank(v.title_tsv, q.q) AS score, v.summary AS frag
			FROM vis v, q WHERE v.title_tsv @@ q.q
			UNION ALL
			SELECT v.id, 2 * ts_rank(v.title_tsv, q.q) + ts_rank_cd(c.tsv, q.q), c.content
			FROM kb_chunks c JOIN vis v ON v.id = c.article_id, q
			WHERE c.tsv @@ q.q
		),
		best AS (
			SELECT DISTINCT ON (id) id, score, frag FROM hits ORDER BY id, score DESC
		)
		SELECT a.id::text, s.code, a.title, a.summary, left(b.frag, 220), `+kbStageCode+`,
		       round(b.score::numeric, 4)::float8
		FROM best b
		JOIN kb_articles a ON a.id = b.id
		JOIN kb_sections s ON s.id = a.section_id
		LEFT JOIN onboarding_stages st ON st.id = a.stage_id
		ORDER BY b.score DESC, a.title
		LIMIT $3`, teamID, query, limit, onlyAI)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.KBSearchMatch, error) {
		var m models.KBSearchMatch
		err := row.Scan(&m.ArticleID, &m.SectionID, &m.Title, &m.Summary, &m.Snippet, &m.StageCode, &m.Score)
		return m, err
	})
}

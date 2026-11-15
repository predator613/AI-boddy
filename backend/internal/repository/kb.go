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

// kbVisible – статья видна пользователю. Общие статьи видят все, командные – своя команда.
// Ответы наставника (source = mentor_answer) видят только сам наставник, его стажеры
// и руководитель команды: у другого наставника ответ на тот же вопрос может быть другим.
// $1 – команда пользователя, $v – его id.
func kbVisible(v string) string {
	return `(a.team_id IS NULL OR a.team_id = $1::uuid) AND (a.use_in_ai OR a.source NOT IN ('upload', 'link')) AND (a.source <> 'mentor_answer'
		OR a.author_id::text = ` + v + `
		OR a.author_id = (SELECT ip.mentor_id FROM intern_profiles ip WHERE ip.user_id::text = ` + v + `)
		OR EXISTS (SELECT 1 FROM users lu WHERE lu.id::text = ` + v + ` AND lu.role = 'lead' AND lu.team_id = a.team_id))`
}

// Sections – все разделы с числом видимых статей.
func (r *KB) Sections(ctx context.Context, teamID, viewerID string) ([]models.KBSection, error) {
	rows, err := r.db.Query(ctx, `
		SELECT * FROM (
			SELECT s.code, s.title, s.hand, s.color,
			       (SELECT count(*) FROM kb_articles a WHERE a.section_id = s.id AND `+kbVisible("$2")+`)::int AS n,
			       s.position
			FROM kb_sections s
		) x WHERE n > 0 -- пустой раздел (например, удалили единственный ответ наставника) не показываем
		ORDER BY position, code`, teamID, viewerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.KBSection, error) {
		var s models.KBSection
		var pos int
		err := row.Scan(&s.ID, &s.Title, &s.Hand, &s.Color, &s.Count, &pos)
		return s, err
	})
}

// Articles – видимые статьи без текста, по разделам.
func (r *KB) Articles(ctx context.Context, teamID, viewerID string) ([]models.KBArticle, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text, s.code, a.title, a.summary, `+kbMinutes+`, `+kbStageCode+`
		FROM kb_articles a
		JOIN kb_sections s ON s.id = a.section_id
		LEFT JOIN onboarding_stages st ON st.id = a.stage_id
		WHERE `+kbVisible("$2")+`
		ORDER BY s.position, a.created_at, a.id`, teamID, viewerID)
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
func (r *KB) Article(ctx context.Context, teamID, viewerID, id string) (*models.KBArticleDetail, error) {
	var a models.KBArticleDetail
	err := r.db.QueryRow(ctx, `
		SELECT a.id::text, s.code, a.title, a.summary, `+kbMinutes+`, `+kbStageCode+`,
		       a.body, a.source, a.url, u.name, a.use_in_ai, a.updated_at
		FROM kb_articles a
		JOIN kb_sections s ON s.id = a.section_id
		LEFT JOIN onboarding_stages st ON st.id = a.stage_id
		LEFT JOIN users u ON u.id = a.author_id
		WHERE a.id::text = $2 AND `+kbVisible("$3"), teamID, id, viewerID,
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
func (r *KB) Search(ctx context.Context, teamID, viewerID, query string, limit int, onlyAI bool) ([]models.KBSearchMatch, error) {
	rows, err := r.db.Query(ctx, `
		WITH q AS (
			SELECT replace(plainto_tsquery('russian', $2)::text, '&', '|')::tsquery AS q
		),
		lex AS (
			SELECT DISTINCT quote_literal(l)::tsquery AS lq
			FROM unnest(tsvector_to_array(to_tsvector('russian', $2))) AS l
		),
		vis AS (
			SELECT a.* FROM kb_articles a
			WHERE `+kbVisible("$5")+` AND (NOT $4 OR a.use_in_ai)
		),
		hits AS (
			SELECT v.id, 2 * ts_rank(v.title_tsv, q.q) AS score, v.summary AS frag, v.title_tsv AS doc
			FROM vis v, q WHERE v.title_tsv @@ q.q
			UNION ALL
			SELECT v.id, 2 * ts_rank(v.title_tsv, q.q) + ts_rank_cd(c.tsv, q.q), c.content, v.title_tsv || c.tsv
			FROM kb_chunks c JOIN vis v ON v.id = c.article_id, q
			WHERE c.tsv @@ q.q
		),
		best AS (
			SELECT DISTINCT ON (id) id, score, frag, doc FROM hits ORDER BY id, score DESC
		)
		SELECT a.id::text, s.code, a.title, a.summary, left(b.frag, 220), `+kbStageCode+`,
		       round(b.score::numeric, 4)::float8,
		       COALESCE((SELECT count(*) FILTER (WHERE b.doc @@ lq)::float8 / NULLIF(count(*), 0) FROM lex), 0)
		FROM best b
		JOIN kb_articles a ON a.id = b.id
		JOIN kb_sections s ON s.id = a.section_id
		LEFT JOIN onboarding_stages st ON st.id = a.stage_id
		ORDER BY b.score DESC, a.title
		LIMIT $3`, teamID, query, limit, onlyAI, viewerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.KBSearchMatch, error) {
		var m models.KBSearchMatch
		err := row.Scan(&m.ArticleID, &m.SectionID, &m.Title, &m.Summary, &m.Snippet, &m.StageCode, &m.Score, &m.Coverage)
		return m, err
	})
}

// ArticleRef – подпись источника «Раздел → Статья».
func (r *KB) ArticleRef(ctx context.Context, id string) (string, error) {
	var ref string
	err := r.db.QueryRow(ctx, `
		SELECT s.title || ' → ' || a.title FROM kb_articles a JOIN kb_sections s ON s.id = a.section_id
		WHERE a.id::text = $1`, id).Scan(&ref)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return ref, err
}

// MentorAnswers – последние ответы наставников, которые видит пользователь: id и вопрос стажера (заголовок).
func (r *KB) MentorAnswers(ctx context.Context, teamID, viewerID string, limit int) ([]models.KBArticle, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text, a.title FROM kb_articles a
		WHERE a.source = 'mentor_answer' AND a.use_in_ai AND `+kbVisible("$2")+`
		ORDER BY a.updated_at DESC LIMIT $3`, teamID, viewerID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.KBArticle, error) {
		var a models.KBArticle
		err := row.Scan(&a.ID, &a.Title)
		return a, err
	})
}

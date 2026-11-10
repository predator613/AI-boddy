-- 006. Справочник (база знаний) и индекс для поиска AI-buddy.

CREATE TABLE kb_sections (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code     TEXT NOT NULL UNIQUE,                  -- s-access
    title    TEXT NOT NULL,
    hand     TEXT NOT NULL DEFAULT '',              -- подпись от руки на стикере
    color    TEXT NOT NULL DEFAULT '#FFFFFF' CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    position INT  NOT NULL
);

-- Статья или загруженный материал. Число статей в разделе и время чтения не храним: считаем запросом.
CREATE TABLE kb_articles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id UUID NOT NULL REFERENCES kb_sections (id),
    team_id    UUID NULL REFERENCES teams (id) ON DELETE CASCADE,   -- NULL – видят все команды
    stage_id   UUID NULL REFERENCES onboarding_stages (id) ON DELETE SET NULL, -- «нужно на твоем этапе»
    title      TEXT NOT NULL,
    summary    TEXT NOT NULL DEFAULT '',
    body       TEXT NOT NULL DEFAULT '',
    source     TEXT NOT NULL CHECK (source IN ('seed', 'upload', 'link', 'mentor_answer')),
    url        TEXT NULL,
    author_id  UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    use_in_ai  BOOLEAN NOT NULL DEFAULT true,       -- помощник может отвечать по этому материалу
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- заголовок и аннотация для поиска: совпадение в заголовке весит больше, чем в тексте
    title_tsv  TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('russian', title), 'A') || setweight(to_tsvector('russian', summary), 'B')) STORED,
    CHECK (source <> 'link' OR url IS NOT NULL)
);
CREATE INDEX idx_kb_articles_title_tsv ON kb_articles USING GIN (title_tsv);
CREATE INDEX idx_kb_articles_section ON kb_articles (section_id);
CREATE INDEX idx_kb_articles_team ON kb_articles (team_id);

-- Файл загруженного материала (PDF, DOCX, MD). Метаданные отдельно от статьи: у статьи файла может не быть.
CREATE TABLE kb_files (
    article_id UUID PRIMARY KEY REFERENCES kb_articles (id) ON DELETE CASCADE,
    filename   TEXT   NOT NULL,
    mime_type  TEXT   NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes BETWEEN 1 AND 10485760),
    sha256     TEXT   NOT NULL,
    content    BYTEA  NOT NULL,
    pages      INT    NULL
);

-- Словарик.
CREATE TABLE kb_terms (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    term       TEXT NOT NULL,
    definition TEXT NOT NULL,
    team_id    UUID NULL REFERENCES teams (id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX kb_terms_term_key ON kb_terms (lower(term), COALESCE(team_id, '00000000-0000-0000-0000-000000000000'));

-- Фрагменты статей для поиска. tsv считается сам из текста (русская морфология).
CREATE TABLE kb_chunks (
    id         BIGSERIAL PRIMARY KEY,
    article_id UUID NOT NULL REFERENCES kb_articles (id) ON DELETE CASCADE,
    chunk_no   INT  NOT NULL,
    content    TEXT NOT NULL,
    tsv        TSVECTOR GENERATED ALWAYS AS (to_tsvector('russian', content)) STORED,
    UNIQUE (article_id, chunk_no)
);
CREATE INDEX idx_kb_chunks_tsv ON kb_chunks USING GIN (tsv);

-- Вектор фрагмента зависит от фрагмента и модели, поэтому отдельная таблица с ключом (chunk_id, model).
-- Без фиксированной размерности: можно сменить модель эмбеддингов без миграции.
CREATE TABLE kb_chunk_embeddings (
    chunk_id   BIGINT NOT NULL REFERENCES kb_chunks (id) ON DELETE CASCADE,
    model      TEXT   NOT NULL,
    embedding  VECTOR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (chunk_id, model)
);

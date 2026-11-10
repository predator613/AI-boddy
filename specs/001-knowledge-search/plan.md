# 001. План: поиск по справочнику

## Данные (миграция 006)

- `kb_sections` (код, название, подпись, цвет, порядок), `kb_articles` (раздел, команда или NULL, этап, текст, источник, `use_in_ai`), `kb_files`, `kb_terms`.
- `kb_chunks` – статьи, разбитые по абзацам; колонка `tsv` вычисляется самой базой (`to_tsvector('russian', content)`), индекс GIN.
- `kb_articles.title_tsv` – заголовок с весом A и аннотация с весом B.
- `kb_chunk_embeddings` – задел под векторный поиск (pgvector, без фиксированной размерности); включается флагом `kb.semantic_search`.

Время чтения и число статей в разделе не храним – считаем запросом (3НФ).

## Поиск

```sql
q = replace(plainto_tsquery('russian', $query)::text, '&', '|')::tsquery   -- «любое из слов»
score = 2 * ts_rank(title_tsv, q) + ts_rank_cd(chunk.tsv, q)
```

Фильтр видимости (`team_id IS NULL OR team_id = команда пользователя`) стоит до ранжирования, поэтому чужие статьи не влияют ни на выдачу, ни на счетчики. Из нескольких фрагментов статьи берется лучший (`DISTINCT ON`).

Почему не `websearch_to_tsquery`: он требует все слова сразу, а вопросы стажеров длинные («как получить доступ к тестовому стенду») и почти никогда целиком не совпадают с текстом.

## API

`GET /api/kb`, `GET /api/kb/articles/{id}`, `GET /api/kb/search?q=&limit=` – контракт в `docs/api-goals-kb.md`.

## Код

- `internal/repository/kb.go` – SQL.
- `internal/services/kb.go` – проверка запроса, лимит 1–20, команда пользователя.
- `internal/handlers/kb.go` – HTTP.
- Фронт: `src/api/assistant.ts` (`getKbArticle`, `searchKb`), `src/hooks/useKb.ts` (поиск с задержкой 250 мс), `src/pages/KnowledgePage.tsx`.

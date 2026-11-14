-- Помощник ссылается на статьи справочника в чате (message_sources). Если наставник удаляет
-- материал, ссылка обнулялась и нарушала проверку «статья или подпись»: удаление падало.
-- Теперь перед удалением статьи в ссылке остается ее название, а история чата не ломается.
CREATE OR REPLACE FUNCTION keep_source_title() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE message_sources SET label = COALESCE(label, OLD.title) WHERE article_id = OLD.id;
    RETURN OLD;
END $$;

DROP TRIGGER IF EXISTS kb_articles_keep_source_title ON kb_articles;
CREATE TRIGGER kb_articles_keep_source_title
    BEFORE DELETE ON kb_articles
    FOR EACH ROW EXECUTE FUNCTION keep_source_title();

-- Загруженные материалы и ответы наставников: куски без заголовка пересобираются так,
-- чтобы в каждом были заголовок и абзац. Раньше заголовок был отдельным куском, и поиск по вопросу
-- находил заголовок без ответа.
DELETE FROM kb_chunks c USING kb_articles a
WHERE c.article_id = a.id AND a.source IN ('mentor_answer', 'upload') AND a.use_in_ai;

INSERT INTO kb_chunks (article_id, chunk_no, content)
SELECT a.id, p.ord, a.title || E'\n' || btrim(p.part)
FROM kb_articles a, regexp_split_to_table(a.body, E'\n\n') WITH ORDINALITY AS p(part, ord)
WHERE a.source IN ('mentor_answer', 'upload') AND a.use_in_ai AND btrim(p.part) <> '';

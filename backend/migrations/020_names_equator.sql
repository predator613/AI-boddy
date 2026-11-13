-- Демо-имена: стажер Анна -> Мария, руководитель Олег -> Артемий (логины intern_anna и lead_oleg прежние).
-- Имена встречаются в seed-текстах (статьи, сообщения, поля заявок), поэтому замена идет по всем
-- текстовым колонкам с учетом падежей. Плюс понятная формулировка задачи про экватор.

CREATE OR REPLACE FUNCTION pg_temp.rename_text(s TEXT) RETURNS TEXT LANGUAGE sql IMMUTABLE AS $$
  SELECT regexp_replace(regexp_replace(regexp_replace(regexp_replace(regexp_replace(
         regexp_replace(regexp_replace(regexp_replace(regexp_replace(regexp_replace(s,
    '(?<![А-Яа-яЁё])Олегом(?![А-Яа-яЁё])', 'Артемием', 'g'),
    '(?<![А-Яа-яЁё])Олега(?![А-Яа-яЁё])', 'Артемия', 'g'),
    '(?<![А-Яа-яЁё])Олегу(?![А-Яа-яЁё])', 'Артемию', 'g'),
    '(?<![А-Яа-яЁё])Олеге(?![А-Яа-яЁё])', 'Артемии', 'g'),
    '(?<![А-Яа-яЁё])Олег(?![А-Яа-яЁё])', 'Артемий', 'g'),
    '(?<![А-Яа-яЁё])Анной(?![А-Яа-яЁё])', 'Марией', 'g'),
    '(?<![А-Яа-яЁё])Анну(?![А-Яа-яЁё])', 'Марию', 'g'),
    '(?<![А-Яа-яЁё])Анне(?![А-Яа-яЁё])', 'Марии', 'g'),
    '(?<![А-Яа-яЁё])Анны(?![А-Яа-яЁё])', 'Марии', 'g'),
    '(?<![А-Яа-яЁё])Анна(?![А-Яа-яЁё])', 'Мария', 'g')
$$;

DO $$
DECLARE c RECORD; n BIGINT;
BEGIN
  FOR c IN
    SELECT table_name, column_name FROM information_schema.columns
    WHERE table_schema = 'public' AND data_type IN ('text', 'character varying')
      AND column_name NOT IN ('login', 'password_hash')
      AND table_name NOT IN ('schema_migrations')
      AND table_name IN (SELECT table_name FROM information_schema.tables
                         WHERE table_schema = 'public' AND table_type = 'BASE TABLE')
  LOOP
    EXECUTE format(
      'UPDATE %I SET %I = pg_temp.rename_text(%I) WHERE %I ~ ''(Олег|Анн[аыеуо])''',
      c.table_name, c.column_name, c.column_name, c.column_name);
  END LOOP;
END $$;

-- jsonb-колонки (поля заявок, журналы) тоже могут хранить имена.
DO $$
DECLARE c RECORD;
BEGIN
  FOR c IN
    SELECT table_name, column_name FROM information_schema.columns
    WHERE table_schema = 'public' AND data_type = 'jsonb'
      AND table_name IN (SELECT table_name FROM information_schema.tables
                         WHERE table_schema = 'public' AND table_type = 'BASE TABLE')
  LOOP
    EXECUTE format(
      'UPDATE %I SET %I = pg_temp.rename_text(%I::text)::jsonb WHERE %I::text ~ ''(Олег|Анн[аыеуо])''',
      c.table_name, c.column_name, c.column_name, c.column_name);
  END LOOP;
END $$;

-- Экватор – середина стажировки, а не место: задача про мастер-классы перед ним.
UPDATE onboarding_tasks SET title = 'Пройти мастер-классы перед экватором',
  done_when = 'ты был хотя бы на одном мастер-классе по подготовке к экватору'
WHERE id = 'b7000000-0000-4000-8000-000000000601';

-- Демо-встречи и записи стажера считаются от даты создания базы и могли попасть на субботу,
-- воскресенье или раньше 10:00, а календарь стажера показывает пн–пт с 10:00 до 18:00.
-- Переносим будущие события в рабочее окно: выходные – на понедельник, утро – на 10:00.

UPDATE meetings
SET starts_at = starts_at + CASE extract(isodow FROM starts_at AT TIME ZONE 'Europe/Moscow')
                                WHEN 6 THEN interval '2 days' ELSE interval '1 day' END
WHERE starts_at > now() - interval '1 day'
  AND extract(isodow FROM starts_at AT TIME ZONE 'Europe/Moscow') IN (6, 7);

UPDATE calendar_entries
SET starts_at = starts_at + CASE extract(isodow FROM starts_at AT TIME ZONE 'Europe/Moscow')
                                WHEN 6 THEN interval '2 days' ELSE interval '1 day' END
WHERE starts_at > now() - interval '1 day'
  AND extract(isodow FROM starts_at AT TIME ZONE 'Europe/Moscow') IN (6, 7);

UPDATE calendar_entries
SET starts_at = (date_trunc('day', starts_at AT TIME ZONE 'Europe/Moscow') + time '10:00') AT TIME ZONE 'Europe/Moscow'
WHERE extract(hour FROM starts_at AT TIME ZONE 'Europe/Moscow') < 10;

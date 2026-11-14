-- Исправления по итогам проверки демо (docs/qa-report.md).

-- «Знакомство с DevOps» стояло в одно время с «1:1 с Лизой» в понедельник 12:00:
-- у стажера две встречи одновременно. Переносим знакомство на 15:00.
UPDATE meetings m
SET starts_at = m.starts_at + interval '3 hours'
WHERE m.title = 'Знакомство с DevOps' AND m.status = 'scheduled'
  AND EXISTS (SELECT 1 FROM meeting_participants p
              JOIN meeting_participants p2 ON p2.user_id = p.user_id AND p2.meeting_id <> p.meeting_id
              JOIN meetings o ON o.id = p2.meeting_id AND o.status = 'scheduled'
              WHERE p.meeting_id = m.id
                AND o.starts_at < m.starts_at + make_interval(mins => m.duration_min)
                AND m.starts_at < o.starts_at + make_interval(mins => o.duration_min));

-- Демо-отгулы считаются от даты создания базы и могли попасть на выходной («отгул 15 ноября,
-- воскресенье» – согласован). Переносим на ближайший понедельник.
UPDATE leave_request_details
SET leave_date = leave_date + CASE extract(isodow FROM leave_date) WHEN 6 THEN 2 ELSE 1 END
WHERE extract(isodow FROM leave_date) IN (6, 7);

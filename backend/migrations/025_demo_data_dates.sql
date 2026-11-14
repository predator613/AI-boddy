-- Демо-данные для показа (docs/qa-report.md, «Наблюдения по данным»).

-- Лиза – стажер первой недели, но ее старт был за 3 дня до создания базы, и на главной наставника
-- горело «просрочено: 8». Теперь у нее второй день: задачи первого дня (доступы к трекеру и вики)
-- сделаны, «Добавиться в чаты» и знакомство с наставником – на сегодня, ничего не просрочено.
UPDATE intern_profiles SET start_date = current_date - 1
WHERE user_id = 'aaaaaaaa-0000-4000-8000-000000000006';

INSERT INTO intern_task_progress (intern_id, task_id, status, updated_at)
SELECT 'aaaaaaaa-0000-4000-8000-000000000006', t.id, 'done', ((current_date - 1) + time '15:00') AT TIME ZONE 'Europe/Moscow'
FROM onboarding_tasks t
JOIN onboarding_stages s ON s.id = t.stage_id
JOIN users u ON u.team_id = s.team_id AND u.id = 'aaaaaaaa-0000-4000-8000-000000000006'
WHERE t.due_day = 1
ON CONFLICT (intern_id, task_id) DO UPDATE SET status = 'done', updated_at = EXCLUDED.updated_at;

-- Дмитрий на 6-й неделе, а чек-лист наставника по нему был «0 из 10». Пункты первых двух недель
-- Игорь давно выполнил, впереди – подготовка к экватору.
INSERT INTO mentor_checklist_marks (mentor_id, intern_id, item_id)
SELECT 'aaaaaaaa-0000-4000-8000-000000000002', 'aaaaaaaa-0000-4000-8000-000000000005', id
FROM mentor_checklist_items WHERE from_week <= 2
ON CONFLICT DO NOTHING;

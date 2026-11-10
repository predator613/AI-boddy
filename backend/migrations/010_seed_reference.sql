-- 010. Справочные данные (синтетика) и перенос действий задач из JSON в task_actions.

-- Согласующий DevOps (approver_id) появляется в 011 вместе с демо-пользователями.
INSERT INTO services (code, title, request_title, description, approver_kind, sla_hours) VALUES
    ('tracker',          'Трекер задач',      'Доступ к трекеру задач',      'Задачи команды',                        'auto',          1),
    ('wiki',             'Вики команды',      'Доступ к вики команды',       'Документация команды',                  'auto',          1),
    ('progress_tracker', 'Трекер прогресса',  'Доступ к трекеру прогресса',  'Цели стажировки и отметки о прогрессе', 'auto',          1),
    ('test_stand',       'Тестовый стенд',    'Доступ к тестовому стенду',   'Стенд для проверки изменений',          'devops',        24),
    ('gitlab',           'GitLab',            'Доступ к GitLab',             'Репозитории и код-ревью',               'project_owner', 24),
    ('prod_logs',        'Логи продакшена',   'Доступ к логам продакшена',   'Просмотр логов боевого окружения',      'devops',        24);

INSERT INTO service_roles (service_code, code, title, access_level) VALUES
    ('tracker', 'read_write', 'чтение и запись', 'edit'),
    ('wiki', 'read', 'чтение', 'read'),
    ('progress_tracker', 'editor', 'редактирование своих целей', 'edit'),
    ('test_stand', 'read', 'чтение', 'read'),
    ('test_stand', 'deploy', 'выкладка веток', 'edit'),
    ('gitlab', 'reporter', 'Reporter', 'read'), ('gitlab', 'developer', 'Developer', 'edit'), ('gitlab', 'maintainer', 'Maintainer', 'admin'),
    ('prod_logs', 'read', 'чтение', 'read');

-- Проекты. Владелец auth-service – Олег; владелец web-app (Нина) проставляется в 011.
INSERT INTO projects (code, service_code, title, team_id, owner_id) VALUES
    ('backend/auth-service', 'gitlab',     'backend / auth-service', '11111111-1111-4111-8111-111111111111', 'aaaaaaaa-0000-4000-8000-000000000003'),
    ('frontend/web-app',     'gitlab',     'frontend / web-app',     '22222222-2222-4222-8222-222222222222', NULL),
    ('auth-staging',         'test_stand', 'auth-staging',           '11111111-1111-4111-8111-111111111111', NULL),
    ('web-staging',          'test_stand', 'web-staging',            '22222222-2222-4222-8222-222222222222', NULL);

-- Наставникам и руководителям можно все; стажерам – кроме Maintainer и логов продакшена.
INSERT INTO service_access_policies (service_code, service_role, user_role, allowed, reason)
SELECT sr.service_code, sr.code, r.code,
       NOT (r.code = 'intern' AND (sr.service_code = 'prod_logs' OR sr.access_level = 'admin')),
       CASE WHEN r.code = 'intern' AND sr.service_code = 'prod_logs'
                THEN 'Стажерам этот доступ не выдается. Логи можно смотреть вместе с наставником.'
            WHEN r.code = 'intern' AND sr.access_level = 'admin'
                THEN 'Стажерам выдается роль Developer, ее хватает для работы.'
            ELSE '' END
FROM service_roles sr CROSS JOIN roles r;

-- Перенос действий задач из 003/004. Группа GitLab превращается в проект, стенд – в стенд команды.
INSERT INTO task_actions (task_id, kind, service_code, service_role, project_code, leave_type, question)
SELECT t.id, t.action_kind,
       t.action_payload ->> 'service', t.action_payload ->> 'role',
       CASE WHEN t.action_payload ->> 'service' = 'gitlab' THEN
                CASE t.action_payload ->> 'group' WHEN 'backend' THEN 'backend/auth-service'
                                                  WHEN 'frontend' THEN 'frontend/web-app' END
            WHEN t.action_payload ->> 'service' = 'test_stand' THEN
                CASE s.team_id WHEN '11111111-1111-4111-8111-111111111111' THEN 'auth-staging' ELSE 'web-staging' END
       END,
       t.action_payload ->> 'type', t.action_payload ->> 'question'
FROM onboarding_tasks t JOIN onboarding_stages s ON s.id = t.stage_id
WHERE t.action_kind IS NOT NULL
  -- переносим только действия, которые проходят новые проверки (данные из консоли могли быть любыми)
  AND CASE t.action_kind
        WHEN 'access_request' THEN EXISTS (SELECT 1 FROM service_roles sr
                                           WHERE sr.service_code = t.action_payload ->> 'service'
                                             AND sr.code = t.action_payload ->> 'role')
        WHEN 'leave_request'  THEN EXISTS (SELECT 1 FROM leave_types lt WHERE lt.code = t.action_payload ->> 'type')
        WHEN 'ask_buddy'      THEN coalesce(t.action_payload ->> 'question', '') <> ''
        ELSE false
      END;
ALTER TABLE onboarding_tasks DROP COLUMN action_kind, DROP COLUMN action_payload;

INSERT INTO event_types (code, description) VALUES
    ('login',               'вход в приложение'),
    ('page_view',           'открыт экран'),
    ('chat_message_sent',   'стажер написал помощнику'),
    ('chat_answer_shown',   'помощник ответил'),
    ('answer_from_kb',      'ответ со ссылкой на справочник'),
    ('handoff_created',     'вопрос передан наставнику'),
    ('handoff_answered',    'наставник ответил на переданный вопрос'),
    ('action_proposed',     'помощник предложил действие'),
    ('action_confirmed',    'стажер подтвердил действие'),
    ('action_cancelled',    'стажер отменил действие'),
    ('request_created',     'заявка отправлена в сервис'),
    ('request_decided',     'по заявке принято решение'),
    ('task_completed',      'задача пути выполнена'),
    ('stage_completed',     'этап пути пройден'),
    ('kb_search',           'поиск по справочнику'),
    ('kb_article_opened',   'открыта статья'),
    ('feedback_given',      'оценка ответа помощника'),
    ('mentor_reply_sent',   'наставник ответил в переписке'),
    ('meeting_created',     'назначена встреча'),
    ('goal_progress',       'отмечен прогресс по цели'),
    ('goals_agreed',        'руководитель согласовал цели'),
    ('llm_error',           'сбой модели, сработал запасной режим');

INSERT INTO feature_flags (key, enabled, description) VALUES
    ('buddy.llm',            false, 'Ответы через внешнюю LLM. Выключено – работает mock-LLM без ключа'),
    ('buddy.actions',        true,  'Помощник может создавать заявки через сервисы'),
    ('buddy.handoff',        true,  'Помощник передает вопросы наставнику'),
    ('kb.semantic_search',   false, 'Векторный поиск по справочнику (нужны эмбеддинги)'),
    ('notifications.stream', true,  'Уведомления в реальном времени (SSE)');

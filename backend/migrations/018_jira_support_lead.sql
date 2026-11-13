-- Заявки через Jira Service Desk, обращения в поддержку, согласования руководителя, лента наставника,
-- подробности задач, чек-лист наставника, конспект встреч и компетенции у целей.
-- Контракты: docs/api-jira.md, docs/api-support.md, docs/api-lead-feed.md.

-- ---------- Календарь: две демо-встречи в одно время ----------
-- После переноса с выходных (017) две демо-встречи 1:1 Анны могли встать на один и тот же час.
-- Более старую сдвигаем на следующий рабочий день.
UPDATE meetings m
SET starts_at = m.starts_at + CASE extract(isodow FROM m.starts_at AT TIME ZONE 'Europe/Moscow')
                                  WHEN 5 THEN interval '3 days' ELSE interval '1 day' END
WHERE m.title = 'Встреча 1:1' AND m.starts_at > now()
  AND EXISTS (SELECT 1 FROM meetings o
              WHERE o.id <> m.id AND o.host_id = m.host_id AND o.status = 'scheduled'
                AND o.starts_at < m.starts_at + make_interval(mins => m.duration_min)
                AND m.starts_at < o.starts_at + make_interval(mins => o.duration_min));

-- ---------- Jira: ключи заявок ----------
-- Доступы в имитации Jira получают ключ ACCESS-…, раньше был ACC-….
UPDATE requests SET external_ref = 'ACCESS-' || substr(external_ref, 5) WHERE external_ref LIKE 'ACC-%';

-- Обычные доступы (devops) решает сервис заявок, а не человек: согласующий остается только
-- у заявок владельцу проекта и у отгулов.
UPDATE requests q SET approver_id = NULL
FROM access_request_details d
JOIN services s ON s.code = d.service_code
WHERE d.request_id = q.id AND s.approver_kind IN ('auto', 'devops');

UPDATE onboarding_tasks SET note = 'Заявка уходит в Jira Service Desk, команда доступов'
WHERE note = 'Согласует руководитель команды' AND title = 'Доступ к тестовому стенду';
UPDATE onboarding_stages SET tip = 'Обычные доступы выдает команда доступов через Jira Service Desk, доступ к репозиторию согласует владелец проекта. AI-buddy сам подберет, куда отправить заявку, и уровень прав.'
WHERE tip LIKE 'Согласует владелец ресурса: руководитель%';

-- ---------- Лента наставника: кто сделал ----------
ALTER TABLE notifications
    ADD COLUMN actor_id   UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN actor_kind TEXT NOT NULL DEFAULT 'buddy'
        CHECK (actor_kind IN ('intern', 'mentor', 'lead', 'buddy', 'other')),
    ADD COLUMN actor_name TEXT NOT NULL DEFAULT 'AI-buddy' CHECK (length(actor_name) BETWEEN 1 AND 80);
CREATE INDEX idx_notifications_user ON notifications (user_id, created_at DESC);

-- Демо-лента наставника Игоря.
INSERT INTO notifications (user_id, kind, text, link, actor_id, actor_kind, actor_name, created_at, read_at)
SELECT 'aaaaaaaa-0000-4000-8000-000000000002', v.kind, v.text, v.link, v.actor_id::uuid, v.actor_kind, v.actor_name,
       now() - v.ago, CASE WHEN v.read THEN now() - v.ago + interval '10 minutes' END
FROM (VALUES
    ('goals',    'Олег согласовал цели Анны', '/interns/aaaaaaaa-0000-4000-8000-000000000001',
     'aaaaaaaa-0000-4000-8000-000000000003', 'lead', 'Олег', interval '25 minutes', false),
    ('meetings', 'Анна придет на «Встреча со стажерами»', '/schedule',
     'aaaaaaaa-0000-4000-8000-000000000001', 'intern', 'Анна', interval '50 minutes', false),
    ('tasks',    'Лиза закрыла задачу «Встреча-знакомство с командой»', '/interns/aaaaaaaa-0000-4000-8000-000000000006',
     'aaaaaaaa-0000-4000-8000-000000000006', 'intern', 'Лиза', interval '20 hours', false),
    ('requests', 'Команда доступов взяла в работу заявку Анны «Доступ к тестовому стенду»', '/interns/aaaaaaaa-0000-4000-8000-000000000001',
     NULL, 'other', 'Jira Service Desk', interval '22 hours', true),
    ('tips',     'AI-buddy ответил Дмитрию по памятке стажера', '/interns/aaaaaaaa-0000-4000-8000-000000000005',
     NULL, 'buddy', 'AI-buddy', interval '26 hours', true)
) AS v (kind, text, link, actor_id, actor_kind, actor_name, ago, read);

-- ---------- Обращения в поддержку ----------
ALTER TABLE support_tickets DROP CONSTRAINT support_tickets_problem_type_fkey;
ALTER TABLE support_tickets DROP CONSTRAINT support_tickets_status_check;
UPDATE support_tickets SET status = CASE status WHEN 'open' THEN 'new' WHEN 'closed' THEN 'done' ELSE status END;
ALTER TABLE support_tickets
    ALTER COLUMN status SET DEFAULT 'new',
    ADD CONSTRAINT support_tickets_status_check CHECK (status IN ('new', 'in_progress', 'done')),
    ADD CONSTRAINT support_tickets_type_len CHECK (length(problem_type) BETWEEN 1 AND 120),
    ADD COLUMN intern_id     UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN attach_dialog BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN dialog_id     UUID NULL REFERENCES chat_dialogs (id) ON DELETE SET NULL,
    ADD COLUMN external_ref  TEXT NULL UNIQUE,   -- ключ в очереди SUP сервиса заявок
    ADD COLUMN answer        TEXT NULL CHECK (answer IS NULL OR length(answer) <= 2000),
    ADD COLUMN updated_at    TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX idx_support_tickets_user ON support_tickets (user_id, created_at DESC);

-- ---------- Подробности задачи у стажера ----------
ALTER TABLE onboarding_tasks
    ADD COLUMN about     TEXT NOT NULL DEFAULT '' CHECK (length(about) <= 600),
    ADD COLUMN helper    TEXT NOT NULL DEFAULT '' CHECK (length(helper) <= 300),
    ADD COLUMN done_when TEXT NOT NULL DEFAULT '' CHECK (length(done_when) <= 300);

UPDATE onboarding_tasks t SET about = v.about, helper = v.helper, done_when = v.done_when
FROM (VALUES
    ('b7000000-0000-4000-8000-000000000101', 'Короткая встреча, на которой команда рассказывает, кто чем занимается и к кому идти с вопросами.',
     '{{mentor}} представит тебя команде', 'ты знаешь, кто за что отвечает'),
    ('b7000000-0000-4000-8000-000000000102', 'Первая встреча 1:1: договоритесь, как часто встречаться и куда писать вопросы.',
     '{{mentor}} назначит время в календаре', 'есть регулярная встреча 1:1 в календаре'),
    ('b7000000-0000-4000-8000-000000000103', 'В памятке собрано, как устроена команда, какие у нее процессы и что важно в первые недели.',
     'AI-buddy ответит на вопросы по памятке', 'памятка прочитана, вопросы заданы'),
    ('b7000000-0000-4000-8000-000000000104', 'Цели стажировки ставятся от OKR команды. Их предлагает наставник, согласует руководитель.',
     '{{lead}} согласует цели, {{mentor}} поможет сформулировать', 'цели согласованы и видны на главной'),
    ('b7000000-0000-4000-8000-000000000201', 'В трекере лежат все задачи команды. Доступ выдается автоматически при выходе.',
     'AI-buddy проверит статус заявки', 'ты видишь доску команды'),
    ('b7000000-0000-4000-8000-000000000202', 'В вики описаны сервисы, договоренности и инструкции команды.',
     'AI-buddy проверит статус заявки', 'ты открываешь страницы команды'),
    ('b7000000-0000-4000-8000-000000000203', 'На тестовом стенде проверяют изменения до релиза. Стажерам доступно чтение.',
     'AI-buddy заполнит заявку, ее выполнит команда доступов в Jira Service Desk', 'заявка выполнена и стенд открывается'),
    ('b7000000-0000-4000-8000-000000000204', 'Без GitLab не получится взять первую задачу: в нем код, ветки и merge request.',
     'AI-buddy заполнит заявку, согласует {{lead}} как владелец проекта', 'заявка согласована и ты видишь репозиторий'),
    ('b7000000-0000-4000-8000-000000000205', 'В трекере прогресса фиксируются цели, задачи и ожидания от стажировки.',
     'AI-buddy перенесет цели со встречи с руководителем', 'цели внесены в трекер'),
    ('b7000000-0000-4000-8000-000000000301', 'Нужно склонировать репозиторий, поднять сервис в Docker и запустить тесты локально.',
     '{{mentor}} поможет, если сборка не проходит', 'сервис запускается у тебя локально'),
    ('b7000000-0000-4000-8000-000000000302', 'Ревью в команде проходит по памятке: что смотрят, сколько ждать и как отвечать на замечания.',
     'AI-buddy расскажет по памятке, пример покажет {{mentor}}', 'ты знаешь, как отправить код на ревью'),
    ('b7000000-0000-4000-8000-000000000303', 'Разберись, какие проверки проходит код и как он попадает в прод.',
     'AI-buddy объяснит по справочнику', 'ты можешь описать путь от merge до релиза'),
    ('b7000000-0000-4000-8000-000000000304', 'На дашбордах видно ошибки и нагрузку сервиса. Пригодится на первой задаче.',
     '{{mentor}} покажет, куда смотреть', 'ты нашел дашборд своего сервиса'),
    ('b7000000-0000-4000-8000-000000000305', 'Подпишись на чат команды и канал алертов, чтобы не пропускать важное.',
     'AI-buddy подскажет список каналов', 'уведомления приходят'),
    ('b7000000-0000-4000-8000-000000000401', 'Маленькая задача, чтобы пройти весь цикл: ветка, merge request, ревью, релиз.',
     '{{mentor}} проверит merge request', 'изменение в основной ветке'),
    ('b7000000-0000-4000-8000-000000000402', 'Тесты помогают разобраться в модуле и не бояться его менять.',
     '{{mentor}} подскажет, что покрыть в первую очередь', 'тесты проходят в CI'),
    ('b7000000-0000-4000-8000-000000000403', 'Ревью чужого кода – способ быстро узнать кодовую базу и стиль команды.',
     '{{mentor}} посмотрит твои комментарии', 'оставлены комментарии к merge request коллеги')
) AS v (id, about, helper, done_when)
WHERE t.id = v.id::uuid;

-- ---------- Чек-лист наставника ----------
-- Пункты общие (по чек-листу стажировки), отметки – на пару наставник–стажер.
CREATE TABLE mentor_checklist_items (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    position  INT  NOT NULL UNIQUE CHECK (position > 0),
    title     TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 120),
    hint      TEXT NOT NULL DEFAULT '' CHECK (length(hint) <= 200),
    from_week INT  NOT NULL CHECK (from_week >= 1),
    period    TEXT NOT NULL CHECK (length(period) BETWEEN 1 AND 40)
);
CREATE TABLE mentor_checklist_marks (
    mentor_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    intern_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    item_id   UUID NOT NULL REFERENCES mentor_checklist_items (id) ON DELETE CASCADE,
    done_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (mentor_id, intern_id, item_id)
);

INSERT INTO mentor_checklist_items (id, position, title, hint, from_week, period) VALUES
    ('c1000000-0000-4000-8000-000000000001', 1, 'Добавить в чаты и каналы', 'в том числе закрытые', 1, 'неделя 1'),
    ('c1000000-0000-4000-8000-000000000002', 2, 'Проверить, что доступы запрошены', 'трекер, вики, GitLab, стенд', 1, 'неделя 1'),
    ('c1000000-0000-4000-8000-000000000003', 3, 'Назначить регулярные 1:1', 'про команду, инструменты и задачи', 1, 'неделя 1'),
    ('c1000000-0000-4000-8000-000000000004', 4, 'Обсудить цели и критерии успеха', 'вместе с руководителем, от OKR команды', 1, 'неделя 1'),
    ('c1000000-0000-4000-8000-000000000005', 5, 'Запросить методичку наставника', 'у куратора стажировки', 1, 'неделя 1'),
    ('c1000000-0000-4000-8000-000000000006', 6, 'Проверить цели в трекере прогресса', 'цели, задачи и ожидания внесены', 2, 'неделя 2'),
    ('c1000000-0000-4000-8000-000000000007', 7, 'Сходить на бриф перед экватором', 'приглашение придет в календарь', 12, 'перед экватором'),
    ('c1000000-0000-4000-8000-000000000008', 8, 'Проверить презентацию стажера', 'задачи и цели отражены верно', 12, 'перед экватором'),
    ('c1000000-0000-4000-8000-000000000009', 9, 'Подготовить напутственное слово', 'не обратная связь, а благодарность и пожелания', 13, 'экватор'),
    ('c1000000-0000-4000-8000-000000000010', 10, '1:1 с обратной связью по целям', 'промежуточные итоги стажировки', 13, 'после экватора');

INSERT INTO mentor_checklist_marks (mentor_id, intern_id, item_id)
SELECT 'aaaaaaaa-0000-4000-8000-000000000002', 'aaaaaaaa-0000-4000-8000-000000000001', id
FROM mentor_checklist_items WHERE position <= 4;
INSERT INTO mentor_checklist_marks (mentor_id, intern_id, item_id)
SELECT 'aaaaaaaa-0000-4000-8000-000000000002', 'aaaaaaaa-0000-4000-8000-000000000006', id
FROM mentor_checklist_items WHERE position = 1;

-- ---------- Конспект встреч стажера ----------
-- event_id – id события календаря: встречи (meetings) или своей записи (calendar_entries).
CREATE TABLE meeting_notes (
    intern_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    event_id   UUID NOT NULL,
    text       TEXT NOT NULL CHECK (length(text) BETWEEN 1 AND 4000),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (intern_id, event_id)
);

INSERT INTO meeting_notes (intern_id, event_id, text, updated_at) VALUES
    ('aaaaaaaa-0000-4000-8000-000000000001', 'f1000000-0000-4000-8000-000000000005',
     'Спросить про первую задачу и ревью. Вопросы по коду – Игорю, по доступу к репозиторию – Олегу.',
     now() - interval '2 days');

-- ---------- Компетенция у цели ----------
ALTER TABLE intern_goals
    ADD COLUMN competency TEXT NULL CHECK (competency IS NULL OR length(competency) BETWEEN 1 AND 80);
UPDATE intern_goals SET competency = 'Код и ревью · L1' WHERE title = 'Пройти путь задачи до релиза';
UPDATE intern_goals SET competency = 'Обмен знаниями' WHERE title = 'Провести ревью коллег';

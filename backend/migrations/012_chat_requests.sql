-- 012. Чат с помощником и заявки: синонимы сервисов, связь с внешним сервисом заявок,
-- задача «трекер прогресса» становится заявкой на доступ, ответы наставника видят только его стажеры.

-- Как люди называют сервис в вопросах. Нужны помощнику, чтобы понять «дай доступ к репозиторию».
-- Самое длинное совпадение важнее («трекер прогресса» раньше «трекер»).
CREATE TABLE service_aliases (
    alias        TEXT PRIMARY KEY CHECK (alias = lower(alias) AND length(alias) BETWEEN 2 AND 40),
    service_code TEXT NOT NULL REFERENCES services (code) ON DELETE CASCADE
);
INSERT INTO service_aliases (alias, service_code) VALUES
    ('gitlab', 'gitlab'), ('гитлаб', 'gitlab'), ('bitbucket', 'gitlab'), ('битбакет', 'gitlab'),
    ('репозитор', 'gitlab'), ('pull request', 'gitlab'), ('мерж', 'gitlab'),
    ('тестов стенд', 'test_stand'), ('стенд', 'test_stand'), ('dev-окружени', 'test_stand'), ('staging', 'test_stand'),
    ('лог', 'prod_logs'), ('продакшен', 'prod_logs'), ('прод', 'prod_logs'), ('grafana', 'prod_logs'),
    ('вики', 'wiki'), ('confluence', 'wiki'), ('конфлюенс', 'wiki'), ('документаци', 'wiki'),
    ('трекер задач', 'tracker'), ('jira', 'tracker'), ('джир', 'tracker'),
    ('трекер прогресс', 'progress_tracker'), ('трекер цел', 'progress_tracker'), ('цели в трекер', 'progress_tracker');

-- Номер заявки во внешнем сервисе уникален: по нему синхронизируем статусы.
CREATE UNIQUE INDEX requests_external_ref_key ON requests (external_ref) WHERE external_ref IS NOT NULL;
-- Заявки, которые ждут решения, синхронизатор выбирает часто.
CREATE INDEX idx_requests_waiting ON requests (created_at) WHERE status = 'waiting' AND external_ref IS NOT NULL;

-- Задача 2-й недели: помощник запрашивает трекер прогресса (выдается сразу) и переносит цели.
UPDATE task_actions
   SET kind = 'access_request', service_code = 'progress_tracker', service_role = 'editor', question = NULL
 WHERE task_id = 'b7000000-0000-4000-8000-000000000205';

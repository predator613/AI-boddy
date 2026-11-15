-- Демо сокращено до команды Backend: убираем Павла (стажер Frontend) и Нину (руководитель Frontend).
-- Шаблон плана и статьи команды Frontend остаются, но пользователей у команды больше нет.

DELETE FROM handoffs WHERE forwarded_id IN (SELECT m.id FROM messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.intern_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')))
   OR question_id IN (SELECT m.id FROM messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.intern_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')));
DELETE FROM messages WHERE author_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'))
   OR conversation_id IN (SELECT id FROM conversations WHERE intern_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')) OR mentor_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')));
DELETE FROM conversations WHERE intern_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')) OR mentor_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
DELETE FROM request_events WHERE actor_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'))
   OR request_id IN (SELECT id FROM requests WHERE requester_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')));
UPDATE messages SET request_id = NULL WHERE request_id IN (SELECT id FROM requests WHERE requester_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')));
DELETE FROM notifications WHERE request_id IN (SELECT id FROM requests WHERE requester_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')));
DELETE FROM requests WHERE requester_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
UPDATE requests SET approver_id = NULL WHERE approver_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
UPDATE services SET approver_id = NULL WHERE approver_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
UPDATE projects SET owner_id = NULL WHERE owner_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
DELETE FROM goal_updates WHERE author_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'))
   OR goal_id IN (SELECT id FROM intern_goals WHERE intern_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')));
DELETE FROM intern_goals WHERE intern_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')) OR created_by IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
UPDATE goal_steps SET done_by = NULL, done_at = NULL WHERE done_by IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
UPDATE intern_profiles SET goals_agreed_by = NULL WHERE goals_agreed_by IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
DELETE FROM intern_task_progress WHERE intern_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina')) OR updated_by IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
DELETE FROM meetings WHERE host_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
DELETE FROM template_changes WHERE actor_id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));
DELETE FROM users WHERE id IN (SELECT id FROM users WHERE login IN ('intern_pavel', 'lead_nina'));

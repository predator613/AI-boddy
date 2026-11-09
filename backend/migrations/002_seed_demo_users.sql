-- Синтетические тестовые данные. Это не реальные сотрудники Авито.
-- Пароль у всех пользователей: demo123 (ниже его bcrypt-хеш, cost 10).

INSERT INTO teams (id, name) VALUES
    ('11111111-1111-4111-8111-111111111111', 'Backend'),
    ('22222222-2222-4222-8222-222222222222', 'Frontend')
ON CONFLICT (id) DO NOTHING;

-- Сначала наставник, потому что стажер ссылается на него через mentor_id.
INSERT INTO users (id, login, password_hash, name, role, team_id, mentor_id) VALUES
    ('aaaaaaaa-0000-4000-8000-000000000002', 'mentor_igor',
     '$2b$10$rUthU7u06PVrgi89KBKpre35gRUkAsu.r9NOF6ZTY1M5bhbP6A2d6',
     'Игорь', 'mentor', '11111111-1111-4111-8111-111111111111', NULL),
    ('aaaaaaaa-0000-4000-8000-000000000003', 'lead_oleg',
     '$2b$10$rUthU7u06PVrgi89KBKpre35gRUkAsu.r9NOF6ZTY1M5bhbP6A2d6',
     'Олег', 'lead', '11111111-1111-4111-8111-111111111111', NULL),
    ('aaaaaaaa-0000-4000-8000-000000000001', 'intern_anna',
     '$2b$10$rUthU7u06PVrgi89KBKpre35gRUkAsu.r9NOF6ZTY1M5bhbP6A2d6',
     'Анна', 'intern', '11111111-1111-4111-8111-111111111111', 'aaaaaaaa-0000-4000-8000-000000000002'),
    ('aaaaaaaa-0000-4000-8000-000000000004', 'intern_pavel',
     '$2b$10$rUthU7u06PVrgi89KBKpre35gRUkAsu.r9NOF6ZTY1M5bhbP6A2d6',
     'Павел', 'intern', '22222222-2222-4222-8222-222222222222', NULL)
ON CONFLICT (login) DO NOTHING;

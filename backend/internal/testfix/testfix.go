// Package testfix – данные только для тестов. Из демо Павел и Нина удалены (миграция 026),
// но тесты изоляции команд проверяют чужую команду на них, поэтому в тестовой базе они возвращаются.
package testfix

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const otherTeam = `
INSERT INTO users (id, login, password_hash, name, role, team_id) VALUES
 ('aaaaaaaa-0000-4000-8000-000000000004', 'intern_pavel', '$2b$10$rUthU7u06PVrgi89KBKpre35gRUkAsu.r9NOF6ZTY1M5bhbP6A2d6', 'Павел', 'intern', '22222222-2222-4222-8222-222222222222'),
 ('aaaaaaaa-0000-4000-8000-000000000007', 'lead_nina', '$2b$10$rUthU7u06PVrgi89KBKpre35gRUkAsu.r9NOF6ZTY1M5bhbP6A2d6', 'Нина', 'lead', '22222222-2222-4222-8222-222222222222')
ON CONFLICT (login) DO NOTHING;
INSERT INTO intern_profiles (user_id, start_date, duration_weeks) VALUES ('aaaaaaaa-0000-4000-8000-000000000004', current_date - 2, 24)
ON CONFLICT (user_id) DO NOTHING;
UPDATE projects SET owner_id = 'aaaaaaaa-0000-4000-8000-000000000007' WHERE code = 'frontend/web-app' AND owner_id IS NULL;
INSERT INTO intern_task_progress (intern_id, task_id, status) VALUES ('aaaaaaaa-0000-4000-8000-000000000004', 'b7000000-0000-4000-8000-000000001101', 'done')
ON CONFLICT (intern_id, task_id) DO NOTHING;`

// OtherTeam возвращает в тестовую базу стажера и руководителя команды Frontend.
func OtherTeam(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, otherTeam)
	return err
}

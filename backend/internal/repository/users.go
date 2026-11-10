// Package repository – SQL-запросы к PostgreSQL.
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/models"
)

// ErrNotFound – пользователь не найден.
var ErrNotFound = errors.New("not found")

// Users работает с таблицами users и teams.
type Users struct {
	db *pgxpool.Pool
}

// NewUsers создает репозиторий пользователей.
func NewUsers(db *pgxpool.Pool) *Users {
	return &Users{db: db}
}

// Наставник хранится в профиле стажера (миграция 007), поэтому LEFT JOIN.
// Роль staff (согласующие вроде DevOps) в приложение не входит: для фронта есть только
// стажер, наставник и руководитель.
const selectUser = `
	SELECT u.id::text, u.login, u.name, u.role, u.team_id::text, t.name, p.mentor_id::text, u.password_hash
	FROM users u
	JOIN teams t ON t.id = u.team_id
	LEFT JOIN intern_profiles p ON p.user_id = u.id`

// GetByLogin ищет пользователя по логину (для входа, вместе с хешем пароля).
func (r *Users) GetByLogin(ctx context.Context, login string) (*models.UserWithPassword, error) {
	return r.getOne(ctx, selectUser+` WHERE u.login = $1 AND u.role <> 'staff'`, login)
}

// GetByID ищет пользователя по id (для /me).
func (r *Users) GetByID(ctx context.Context, id string) (*models.User, error) {
	u, err := r.getOne(ctx, selectUser+` WHERE u.id::text = $1`, id)
	if err != nil {
		return nil, err
	}
	return &u.User, nil
}

// ListTestUsers возвращает карточки всех пользователей для демо-входа.
func (r *Users) ListTestUsers(ctx context.Context) ([]models.TestUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.login, u.name, u.role, t.name
		FROM users u
		JOIN teams t ON t.id = u.team_id
		WHERE u.role <> 'staff'
		ORDER BY t.name, CASE u.role WHEN 'intern' THEN 1 WHEN 'mentor' THEN 2 ELSE 3 END, u.login`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.TestUser, error) {
		var tu models.TestUser
		err := row.Scan(&tu.Login, &tu.Name, &tu.Role, &tu.TeamName)
		return tu, err
	})
}

func (r *Users) getOne(ctx context.Context, query string, arg string) (*models.UserWithPassword, error) {
	var u models.UserWithPassword
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&u.ID, &u.Login, &u.Name, &u.Role, &u.TeamID, &u.TeamName, &u.MentorID, &u.PasswordHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

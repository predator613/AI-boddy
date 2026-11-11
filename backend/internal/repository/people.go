package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/models"
)

// People – связи пользователя (наставник, руководитель) и отделы компании.
type People struct {
	db *pgxpool.Pool
}

// NewPeople создает репозиторий связей.
func NewPeople(db *pgxpool.Pool) *People {
	return &People{db: db}
}

// Circle – команда, наставник (только у стажера) и руководитель команды пользователя.
func (r *People) Circle(ctx context.Context, userID string) (*models.Circle, error) {
	var c models.Circle
	var mID, mName, lID, lName *string
	err := r.db.QueryRow(ctx, `
		SELECT t.name, u.role, m.id::text, m.name, l.id::text, l.name
		FROM users u
		JOIN teams t ON t.id = u.team_id
		LEFT JOIN intern_profiles p ON p.user_id = u.id
		LEFT JOIN users m ON m.id = p.mentor_id
		LEFT JOIN users l ON l.team_id = u.team_id AND l.role = 'lead'
		WHERE u.id::text = $1`, userID).Scan(&c.Team, &c.Role, &mID, &mName, &lID, &lName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if mID != nil {
		c.Mentor = &models.PersonRef{ID: *mID, Name: *mName}
	}
	if lID != nil {
		c.Lead = &models.PersonRef{ID: *lID, Name: *lName}
	}
	return &c, nil
}

// Departments – отделы с тем, что можно у них заказать.
func (r *People) Departments(ctx context.Context) ([]models.Department, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.code, d.title, d.sla_hours, d.hours, d.about,
		       COALESCE(string_agg(ci.title, ', ' ORDER BY ci.code), '')
		FROM departments d
		LEFT JOIN catalog_items ci ON ci.department_code = d.code
		GROUP BY d.code ORDER BY d.position`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Department, error) {
		var d models.Department
		err := row.Scan(&d.Code, &d.Title, &d.SLAHours, &d.Hours, &d.About, &d.Items)
		return d, err
	})
}

package services

import (
	"context"
	"errors"
	"strings"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// PeopleStore – связи пользователя и отделы.
type PeopleStore interface {
	Circle(ctx context.Context, userID string) (*models.Circle, error)
	Departments(ctx context.Context) ([]models.Department, error)
}

// People – «свои люди» пользователя: имена берутся из связей в базе, а не из текста.
type People struct {
	store PeopleStore
}

// NewPeople создает сервис связей.
func NewPeople(store PeopleStore) *People {
	return &People{store: store}
}

// Contacts – наставник, руководитель и отделы для страницы «Поддержка».
func (s *People) Contacts(ctx context.Context, viewerID string) (*models.Contacts, error) {
	c, err := s.store.Circle(ctx, viewerID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	deps, err := s.store.Departments(ctx)
	if err != nil {
		return nil, err
	}
	return &models.Contacts{Circle: *c, Departments: deps}, nil
}

// personalize подставляет в текст справочника людей читателя: {{mentor}}, {{lead}}, {{team}}.
// У стажера без наставника – «пока не назначен», наставник читает свое имя.
func personalize(c *models.Circle, viewerName, text string) string {
	if c == nil || !strings.Contains(text, "{{") {
		return text
	}
	mentor := "пока не назначен"
	switch {
	case c.Mentor != nil:
		mentor = c.Mentor.Name
	case c.Role == models.RoleMentor:
		mentor = viewerName
	case c.Role == models.RoleLead:
		mentor = "закреплен за каждым стажером"
	}
	lead := "руководитель команды"
	if c.Lead != nil {
		lead = c.Lead.Name
	}
	return strings.NewReplacer("{{mentor}}", mentor, "{{lead}}", lead, "{{team}}", c.Team).Replace(text)
}

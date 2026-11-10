package services

import (
	"context"
	"errors"
	"strings"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// KBStore – что нужно сервису справочника от базы.
type KBStore interface {
	Sections(ctx context.Context, teamID string) ([]models.KBSection, error)
	Articles(ctx context.Context, teamID string) ([]models.KBArticle, error)
	Terms(ctx context.Context, teamID string) ([]models.KBTerm, error)
	Article(ctx context.Context, teamID, id string) (*models.KBArticleDetail, error)
	Search(ctx context.Context, teamID, query string, limit int, onlyAI bool) ([]models.KBSearchMatch, error)
}

// KB – справочник. Пользователь видит общие материалы и материалы своей команды.
type KB struct {
	store KBStore
	users UserLookup
}

// NewKB создает сервис справочника.
func NewKB(store KBStore, users UserLookup) *KB {
	return &KB{store: store, users: users}
}

// Ограничения поиска.
const (
	kbQueryMax     = 200
	kbDefaultLimit = 10
	kbMaxLimit     = 20
)

// Get – разделы, статьи и словарик для страницы «Справочник».
func (s *KB) Get(ctx context.Context, viewerID string) (*models.KnowledgeBase, error) {
	team, err := s.team(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	out := &models.KnowledgeBase{}
	if out.Sections, err = s.store.Sections(ctx, team); err != nil {
		return nil, err
	}
	if out.Articles, err = s.store.Articles(ctx, team); err != nil {
		return nil, err
	}
	if out.Terms, err = s.store.Terms(ctx, team); err != nil {
		return nil, err
	}
	return out, nil
}

// Article – статья целиком. Чужой команды – 404, как будто ее нет.
func (s *KB) Article(ctx context.Context, viewerID, id string) (*models.KBArticleDetail, error) {
	team, err := s.team(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	a, err := s.store.Article(ctx, team, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return a, err
}

// Search – поиск по справочнику. limit <= 0 – значение по умолчанию.
func (s *KB) Search(ctx context.Context, viewerID, query string, limit int) (*models.KBSearch, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, invalid("q is required")
	}
	if len([]rune(q)) > kbQueryMax {
		return nil, invalid("q is too long (max 200)")
	}
	switch {
	case limit <= 0:
		limit = kbDefaultLimit
	case limit > kbMaxLimit:
		limit = kbMaxLimit
	}
	team, err := s.team(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	res, err := s.store.Search(ctx, team, q, limit, false)
	if err != nil {
		return nil, err
	}
	return &models.KBSearch{Query: q, Results: res}, nil
}

func (s *KB) team(ctx context.Context, viewerID string) (string, error) {
	u, err := s.users.GetByID(ctx, viewerID)
	if errors.Is(err, repository.ErrNotFound) {
		return "", ErrForbidden
	}
	if err != nil {
		return "", err
	}
	return u.TeamID, nil
}

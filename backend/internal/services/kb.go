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
	Sections(ctx context.Context, teamID, viewerID string) ([]models.KBSection, error)
	Articles(ctx context.Context, teamID, viewerID string) ([]models.KBArticle, error)
	Terms(ctx context.Context, teamID string) ([]models.KBTerm, error)
	Article(ctx context.Context, teamID, viewerID, id string) (*models.KBArticleDetail, error)
	Search(ctx context.Context, teamID, viewerID, query string, limit int, onlyAI bool) ([]models.KBSearchMatch, error)
}

// KB – справочник. Пользователь видит общие материалы и материалы своей команды,
// ответы наставника – только своего.
type KB struct {
	store  KBStore
	users  UserLookup
	people PeopleStore
}

// NewKB создает сервис справочника. people – чтобы подставить в статьи наставника и руководителя
// читателя (в тексте статей метки {{mentor}}, {{lead}}, {{team}}).
func NewKB(store KBStore, users UserLookup, people PeopleStore) *KB {
	return &KB{store: store, users: users, people: people}
}

// reader – кто читает: команда и люди для подстановки.
type reader struct {
	team   string
	name   string
	circle *models.Circle
}

func (r reader) text(s string) string { return personalize(r.circle, r.name, s) }

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
	if out.Sections, err = s.store.Sections(ctx, team.team, viewerID); err != nil {
		return nil, err
	}
	if out.Articles, err = s.store.Articles(ctx, team.team, viewerID); err != nil {
		return nil, err
	}
	for i := range out.Articles {
		out.Articles[i].Title, out.Articles[i].Summary = team.text(out.Articles[i].Title), team.text(out.Articles[i].Summary)
	}
	if out.Terms, err = s.store.Terms(ctx, team.team); err != nil {
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
	a, err := s.store.Article(ctx, team.team, viewerID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.Title, a.Summary, a.Body = team.text(a.Title), team.text(a.Summary), team.text(a.Body)
	return a, nil
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
	res, err := s.store.Search(ctx, team.team, viewerID, q, limit, false)
	if err != nil {
		return nil, err
	}
	for i := range res {
		res[i].Title, res[i].Summary, res[i].Snippet = team.text(res[i].Title), team.text(res[i].Summary), team.text(res[i].Snippet)
	}
	return &models.KBSearch{Query: q, Results: res}, nil
}

func (s *KB) team(ctx context.Context, viewerID string) (reader, error) {
	u, err := s.users.GetByID(ctx, viewerID)
	if errors.Is(err, repository.ErrNotFound) {
		return reader{}, ErrForbidden
	}
	if err != nil {
		return reader{}, err
	}
	c, err := s.people.Circle(ctx, viewerID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return reader{}, err
	}
	return reader{team: u.TeamID, name: u.Name, circle: c}, nil
}

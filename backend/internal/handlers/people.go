package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"ai-buddy/backend/internal/models"
)

// PeopleService – свои люди и отделы пользователя.
type PeopleService interface {
	Contacts(ctx context.Context, viewerID string) (*models.Contacts, error)
}

// People – обработчик /api/me/contacts.
type People struct {
	svc PeopleService
	log *slog.Logger
}

// NewPeople создает обработчик контактов.
func NewPeople(svc PeopleService, log *slog.Logger) *People {
	return &People{svc: svc, log: log}
}

// Contacts – GET /api/me/contacts: наставник, руководитель (из связей пользователя) и отделы.
func (h *People) Contacts(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Contacts(r.Context(), id)
	reply(w, h.log, http.StatusOK, v, err)
}

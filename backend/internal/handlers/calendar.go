package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/services"
)

// CalendarService – календарь стажера.
type CalendarService interface {
	Events(ctx context.Context, viewerID, from, to string) (*models.CalendarView, error)
	CreateEntry(ctx context.Context, viewerID string, in services.EntryInput) (*models.CalendarEvent, error)
	DeleteEntry(ctx context.Context, viewerID, id string) error
	Answer(ctx context.Context, viewerID, meetingID string, in services.AnswerInput) (*models.CalendarEvent, error)
}

// Calendar – обработчики /api/me/calendar (только для стажера).
type Calendar struct {
	svc CalendarService
	log *slog.Logger
}

// NewCalendar создает обработчики календаря.
func NewCalendar(svc CalendarService, log *slog.Logger) *Calendar {
	return &Calendar{svc: svc, log: log}
}

// Events – GET /api/me/calendar?from=ISO&to=ISO.
func (h *Calendar) Events(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	v, err := h.svc.Events(r.Context(), id, q.Get("from"), q.Get("to"))
	reply(w, h.log, http.StatusOK, v, err)
}

// CreateEntry – POST /api/me/calendar/entries: {title, kind, startsAt, durationMin, shareWithMentor}.
func (h *Calendar) CreateEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.EntryInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.svc.CreateEntry(r.Context(), id, in)
	reply(w, h.log, http.StatusCreated, v, err)
}

// DeleteEntry – DELETE /api/me/calendar/entries/{id}.
func (h *Calendar) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	replyEmpty(w, h.log, h.svc.DeleteEntry(r.Context(), id, chi.URLParam(r, "id")))
}

// Answer – POST /api/me/calendar/events/{meetingId}/answer: {answer: accept|move, reason}.
func (h *Calendar) Answer(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.AnswerInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.svc.Answer(r.Context(), id, chi.URLParam(r, "meetingId"), in)
	reply(w, h.log, http.StatusOK, v, err)
}

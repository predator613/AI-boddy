package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/services"
)

// StaffService – лента наставника, согласования, чек-лист наставника и конспект встреч.
type StaffService interface {
	Feed(ctx context.Context, viewerID string) (*models.Feed, error)
	ReadFeed(ctx context.Context, viewerID string) error
	Approvals(ctx context.Context, viewerID string) ([]models.Approval, error)
	Decide(ctx context.Context, viewerID, requestID string, in services.DecisionInput) (*models.Approval, error)
	Checklist(ctx context.Context, viewerID, internID string) (*models.MentorChecklist, error)
	MarkChecklist(ctx context.Context, viewerID, internID, itemID string, done *bool) (*models.MentorChecklist, error)
	Notes(ctx context.Context, viewerID string) ([]models.MeetingNote, error)
	SaveNote(ctx context.Context, viewerID, eventID string, text *string) (*models.MeetingNote, error)
}

// SupportService – обращения в поддержку.
type SupportService interface {
	Create(ctx context.Context, viewerID string, in services.SupportInput) (*models.SupportTicket, error)
	List(ctx context.Context, viewerID string) ([]models.SupportTicket, error)
}

// Staff – обработчики кабинета наставника и руководителя, конспекта стажера и поддержки.
type Staff struct {
	svc     StaffService
	support SupportService
	log     *slog.Logger
}

// NewStaff создает обработчики.
func NewStaff(svc StaffService, support SupportService, log *slog.Logger) *Staff {
	return &Staff{svc: svc, support: support, log: log}
}

// Feed – GET /api/mentor/feed.
func (h *Staff) Feed(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Feed(r.Context(), id)
	reply(w, h.log, http.StatusOK, v, err)
}

// ReadFeed – POST /api/mentor/feed/read.
func (h *Staff) ReadFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	replyEmpty(w, h.log, h.svc.ReadFeed(r.Context(), id))
}

// Approvals – GET /api/approvals.
func (h *Staff) Approvals(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Approvals(r.Context(), id)
	reply(w, h.log, http.StatusOK, v, err)
}

// Decide – POST /api/approvals/{id}: {decision: approve} или {decision: reject, reason}.
func (h *Staff) Decide(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.DecisionInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.svc.Decide(r.Context(), id, chi.URLParam(r, "id"), in)
	reply(w, h.log, http.StatusOK, v, err)
}

// Checklist – GET /api/mentor/interns/{internId}/checklist.
func (h *Staff) Checklist(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Checklist(r.Context(), id, chi.URLParam(r, "internId"))
	reply(w, h.log, http.StatusOK, v, err)
}

// MarkChecklist – POST /api/mentor/interns/{internId}/checklist/{itemId}: {done}.
func (h *Staff) MarkChecklist(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in struct {
		Done *bool `json:"done"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.svc.MarkChecklist(r.Context(), id, chi.URLParam(r, "internId"), chi.URLParam(r, "itemId"), in.Done)
	reply(w, h.log, http.StatusOK, v, err)
}

// Notes – GET /api/me/notes.
func (h *Staff) Notes(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Notes(r.Context(), id)
	reply(w, h.log, http.StatusOK, v, err)
}

// SaveNote – PUT /api/me/notes/{eventId}: {text}; пустой текст удаляет заметку.
func (h *Staff) SaveNote(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in struct {
		Text *string `json:"text"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.svc.SaveNote(r.Context(), id, chi.URLParam(r, "eventId"), in.Text)
	reply(w, h.log, http.StatusOK, v, err)
}

// CreateTicket – POST /api/support/tickets: {type, details, internId, attachDialog}.
func (h *Staff) CreateTicket(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.SupportInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.support.Create(r.Context(), id, in)
	reply(w, h.log, http.StatusCreated, v, err)
}

// Tickets – GET /api/support/tickets.
func (h *Staff) Tickets(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.support.List(r.Context(), id)
	reply(w, h.log, http.StatusOK, v, err)
}

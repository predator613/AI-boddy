package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/services"
)

// ChatService – помощник стажера.
type ChatService interface {
	Send(ctx context.Context, viewerID string, in services.SendInput) (*models.ChatReply, error)
	Confirm(ctx context.Context, viewerID, actionID string) (*models.ChatReply, error)
	Cancel(ctx context.Context, viewerID, actionID string) (*models.ChatReply, error)
	History(ctx context.Context, viewerID string) (*models.ChatReply, error)
	Reset(ctx context.Context, viewerID string) (*models.ChatReply, error)
}

// RequestsService – страница «Заявки».
type RequestsService interface {
	List(ctx context.Context, viewerID string) ([]models.RequestItem, error)
}

// Chat – обработчики /api/chat и /api/requests.
type Chat struct {
	svc  ChatService
	reqs RequestsService
	log  *slog.Logger
}

// NewChat создает обработчики чата и заявок.
func NewChat(svc ChatService, reqs RequestsService, log *slog.Logger) *Chat {
	return &Chat{svc: svc, reqs: reqs, log: log}
}

// Send – POST /api/chat/messages: {text, action?, taskId?}.
func (h *Chat) Send(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.SendInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.svc.Send(r.Context(), id, in)
	reply(w, h.log, http.StatusOK, v, err)
}

// Reset – POST /api/chat/reset: «Новый диалог».
func (h *Chat) Reset(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Reset(r.Context(), id)
	reply(w, h.log, http.StatusOK, v, err)
}

// History – GET /api/chat/messages: последние сообщения.
func (h *Chat) History(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.History(r.Context(), id)
	reply(w, h.log, http.StatusOK, v, err)
}

// Confirm – POST /api/chat/actions/{actionId}/confirm.
func (h *Chat) Confirm(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Confirm(r.Context(), id, chi.URLParam(r, "actionId"))
	reply(w, h.log, http.StatusOK, v, err)
}

// Cancel – POST /api/chat/actions/{actionId}/cancel.
func (h *Chat) Cancel(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Cancel(r.Context(), id, chi.URLParam(r, "actionId"))
	reply(w, h.log, http.StatusOK, v, err)
}

// Requests – GET /api/requests: мои заявки.
func (h *Chat) Requests(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.reqs.List(r.Context(), id)
	reply(w, h.log, http.StatusOK, v, err)
}

package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/services"
)

// GoalsService – логика целей стажировки.
type GoalsService interface {
	View(ctx context.Context, viewerID, internID string) (*models.GoalsView, error)
	Create(ctx context.Context, viewerID, internID string, in services.GoalInput) (*models.Goal, error)
	Update(ctx context.Context, viewerID, goalID string, in services.GoalInput) (*models.Goal, error)
	Delete(ctx context.Context, viewerID, goalID string) error
	Progress(ctx context.Context, viewerID, goalID string, in services.ProgressInput) (*models.Goal, error)
	Agree(ctx context.Context, viewerID, internID string) (*models.GoalsView, error)
}

// Goals – обработчики /api/goals и /api/interns/{internId}/goals.
type Goals struct {
	svc GoalsService
	log *slog.Logger
}

// NewGoals создает обработчики целей.
func NewGoals(svc GoalsService, log *slog.Logger) *Goals {
	return &Goals{svc: svc, log: log}
}

// Mine – GET /api/goals: цели текущего стажера.
func (h *Goals) Mine(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.View(r.Context(), id, "")
	reply(w, h.log, http.StatusOK, v, err)
}

// OfIntern – GET /api/interns/{internId}/goals: цели стажера для наставника и руководителя.
func (h *Goals) OfIntern(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.View(r.Context(), id, chi.URLParam(r, "internId"))
	reply(w, h.log, http.StatusOK, v, err)
}

// Create – POST /api/interns/{internId}/goals.
func (h *Goals) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.GoalInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.svc.Create(r.Context(), id, chi.URLParam(r, "internId"), in)
	reply(w, h.log, http.StatusCreated, v, err)
}

// Update – PATCH /api/goals/{goalId}.
func (h *Goals) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.GoalInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.svc.Update(r.Context(), id, chi.URLParam(r, "goalId"), in)
	reply(w, h.log, http.StatusOK, v, err)
}

// Delete – DELETE /api/goals/{goalId}.
func (h *Goals) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	replyEmpty(w, h.log, h.svc.Delete(r.Context(), id, chi.URLParam(r, "goalId")))
}

// Progress – POST /api/goals/{goalId}/progress: {progressPct, note}.
func (h *Goals) Progress(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.ProgressInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, err := h.svc.Progress(r.Context(), id, chi.URLParam(r, "goalId"), in)
	reply(w, h.log, http.StatusOK, v, err)
}

// Agree – POST /api/interns/{internId}/goals/agree: руководитель согласовал цели.
func (h *Goals) Agree(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Agree(r.Context(), id, chi.URLParam(r, "internId"))
	reply(w, h.log, http.StatusOK, v, err)
}

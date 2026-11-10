package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"ai-buddy/backend/internal/httpx"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/services"
)

// PathService – логика «Моего пути» и консоли наставника.
type PathService interface {
	GetPath(ctx context.Context, viewerID, internID string) (*models.PathView, error)
	CompleteTask(ctx context.Context, viewerID, taskID string) (*models.PathView, error)
	SetInternTaskStatus(ctx context.Context, viewerID, internID, taskID, status string, note *string) (*models.PathView, error)
	ListInterns(ctx context.Context, viewerID string) ([]models.InternSummary, error)
	CreateMeeting(ctx context.Context, viewerID, internID, title string, startsAt time.Time) (*models.Meeting, error)

	ListTemplates(ctx context.Context, viewerID string) ([]models.StageTemplate, error)
	CreateStage(ctx context.Context, viewerID string, in services.StageInput) (*models.StageTemplate, error)
	UpdateStage(ctx context.Context, viewerID, stageID string, in services.StageInput) (*models.StageTemplate, error)
	DeleteStage(ctx context.Context, viewerID, stageID string) error
	CreateTask(ctx context.Context, viewerID, stageID string, in services.TaskInput) (*models.TaskTemplate, error)
	UpdateTask(ctx context.Context, viewerID, taskID string, in services.TaskInput) (*models.TaskTemplate, error)
	DeleteTask(ctx context.Context, viewerID, taskID string) error
	CreateResource(ctx context.Context, viewerID, stageID string, in services.ResourceInput) (*models.ResourceTemplate, error)
	DeleteResource(ctx context.Context, viewerID, resourceID string) error
}

// Path – обработчики /api/path, /api/interns и /api/console.
type Path struct {
	svc PathService
	log *slog.Logger
}

// NewPath создает обработчики.
func NewPath(svc PathService, log *slog.Logger) *Path {
	return &Path{svc: svc, log: log}
}

// ---------- стажер ----------

// MyPath – GET /api/path: экран «Мой путь» текущего стажера.
func (h *Path) MyPath(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.GetPath(r.Context(), id, "")
	h.reply(w, http.StatusOK, v, err)
}

// CompleteTask – POST /api/path/tasks/{taskId}/complete: «Уже сделано».
func (h *Path) CompleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.CompleteTask(r.Context(), id, chi.URLParam(r, "taskId"))
	h.reply(w, http.StatusOK, v, err)
}

// ---------- наставник и руководитель ----------

// ListInterns – GET /api/interns.
func (h *Path) ListInterns(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	list, err := h.svc.ListInterns(r.Context(), id)
	h.reply(w, http.StatusOK, list, err)
}

// InternPath – GET /api/interns/{internId}/path.
func (h *Path) InternPath(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.GetPath(r.Context(), id, chi.URLParam(r, "internId"))
	h.reply(w, http.StatusOK, v, err)
}

type setStatusRequest struct {
	Status string  `json:"status"`
	Note   *string `json:"note"`
}

// SetTaskStatus – PUT /api/interns/{internId}/tasks/{taskId}.
func (h *Path) SetTaskStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var req setStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	v, err := h.svc.SetInternTaskStatus(r.Context(), id, chi.URLParam(r, "internId"), chi.URLParam(r, "taskId"), req.Status, req.Note)
	h.reply(w, http.StatusOK, v, err)
}

type meetingRequest struct {
	Title    string    `json:"title"`
	StartsAt time.Time `json:"startsAt"`
}

// CreateMeeting – POST /api/interns/{internId}/meetings.
func (h *Path) CreateMeeting(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var req meetingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	m, err := h.svc.CreateMeeting(r.Context(), id, chi.URLParam(r, "internId"), req.Title, req.StartsAt)
	h.reply(w, http.StatusCreated, m, err)
}

// ---------- консоль наставника: шаблоны ----------

// Templates – GET /api/console/templates.
func (h *Path) Templates(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	list, err := h.svc.ListTemplates(r.Context(), id)
	h.reply(w, http.StatusOK, list, err)
}

// ActionKinds – GET /api/console/action-kinds: список действий AI-buddy для выпадающего списка в форме.
func (h *Path) ActionKinds(w http.ResponseWriter, _ *http.Request) {
	type kind struct {
		Kind        string `json:"kind"`
		Description string `json:"description"`
	}
	order := []string{"access_request", "leave_request", "ask_buddy", "open_article"}
	out := make([]kind, 0, len(order))
	for _, k := range order {
		out = append(out, kind{Kind: k, Description: services.ActionKinds[k]})
	}
	httpx.JSON(w, http.StatusOK, out)
}

// CreateStage – POST /api/console/stages.
func (h *Path) CreateStage(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.StageInput
	if !decodeJSON(w, r, &in) {
		return
	}
	st, err := h.svc.CreateStage(r.Context(), id, in)
	h.reply(w, http.StatusCreated, st, err)
}

// UpdateStage – PUT /api/console/stages/{stageId}.
func (h *Path) UpdateStage(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.StageInput
	if !decodeJSON(w, r, &in) {
		return
	}
	st, err := h.svc.UpdateStage(r.Context(), id, chi.URLParam(r, "stageId"), in)
	h.reply(w, http.StatusOK, st, err)
}

// DeleteStage – DELETE /api/console/stages/{stageId}.
func (h *Path) DeleteStage(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	h.replyEmpty(w, h.svc.DeleteStage(r.Context(), id, chi.URLParam(r, "stageId")))
}

// CreateTask – POST /api/console/stages/{stageId}/tasks.
func (h *Path) CreateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.TaskInput
	if !decodeJSON(w, r, &in) {
		return
	}
	t, err := h.svc.CreateTask(r.Context(), id, chi.URLParam(r, "stageId"), in)
	h.reply(w, http.StatusCreated, t, err)
}

// UpdateTask – PUT /api/console/tasks/{taskId}.
func (h *Path) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.TaskInput
	if !decodeJSON(w, r, &in) {
		return
	}
	t, err := h.svc.UpdateTask(r.Context(), id, chi.URLParam(r, "taskId"), in)
	h.reply(w, http.StatusOK, t, err)
}

// DeleteTask – DELETE /api/console/tasks/{taskId}.
func (h *Path) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	h.replyEmpty(w, h.svc.DeleteTask(r.Context(), id, chi.URLParam(r, "taskId")))
}

// CreateResource – POST /api/console/stages/{stageId}/resources.
func (h *Path) CreateResource(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	var in services.ResourceInput
	if !decodeJSON(w, r, &in) {
		return
	}
	x, err := h.svc.CreateResource(r.Context(), id, chi.URLParam(r, "stageId"), in)
	h.reply(w, http.StatusCreated, x, err)
}

// DeleteResource – DELETE /api/console/resources/{resourceId}.
func (h *Path) DeleteResource(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	h.replyEmpty(w, h.svc.DeleteResource(r.Context(), id, chi.URLParam(r, "resourceId")))
}

// ---------- ответы ----------

func (h *Path) reply(w http.ResponseWriter, status int, body any, err error) {
	if err != nil {
		serviceError(w, h.log, err)
		return
	}
	httpx.JSON(w, status, body)
}

func (h *Path) replyEmpty(w http.ResponseWriter, err error) {
	if err != nil {
		serviceError(w, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

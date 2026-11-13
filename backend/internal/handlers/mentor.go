package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"ai-buddy/backend/internal/httpx"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/services"
)

// MentorService – кабинет наставника (docs/api-mentor-console.md).
type MentorService interface {
	Stats(ctx context.Context, viewerID string) (*models.MentorStats, error)
	Metrics(ctx context.Context, viewerID string) (*models.MentorMetrics, error)
	Threads(ctx context.Context, viewerID string) ([]models.ThreadSummary, error)
	Thread(ctx context.Context, viewerID, internID string) (*models.Thread, error)
	Reply(ctx context.Context, viewerID, internID string, in services.ReplyInput) (*models.Thread, error)
	Read(ctx context.Context, viewerID, internID string) error
	MyThread(ctx context.Context, viewerID string) (*models.Thread, error)
	SendToMentor(ctx context.Context, viewerID, text string) (*models.Thread, error)
	Ack(ctx context.Context, viewerID, messageID string) (*models.Thread, error)
	Materials(ctx context.Context, viewerID string) ([]models.Material, error)
	CreateMaterial(ctx context.Context, viewerID string, in services.MaterialInput) (*models.Material, error)
	DeleteMaterial(ctx context.Context, viewerID, id string) error
	Meetings(ctx context.Context, viewerID, from, to string) ([]models.MentorMeeting, error)
	CreateMeeting(ctx context.Context, viewerID string, in services.MeetingInput) (*models.MentorMeeting, error)
	MoveMeeting(ctx context.Context, viewerID, id string, startsAt *time.Time) (*models.MentorMeeting, error)
	CancelMeeting(ctx context.Context, viewerID, id string) error
	Automations(ctx context.Context, viewerID string) ([]models.Automation, error)
	SetAutomation(ctx context.Context, viewerID, id string, enabled *bool) (*models.Automation, error)
}

// Mentor – обработчики кабинета наставника и переписки стажера.
type Mentor struct {
	svc MentorService
	log *slog.Logger
}

// NewMentor создает обработчики.
func NewMentor(svc MentorService, log *slog.Logger) *Mentor {
	return &Mentor{svc: svc, log: log}
}

// with – общий шаблон: id из токена, вызов, ответ.
func (h *Mentor) with(status int, fn func(r *http.Request, id string) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := viewerID(w, r)
		if !ok {
			return
		}
		v, err := fn(r, id)
		if status == http.StatusNoContent {
			replyEmpty(w, h.log, err)
			return
		}
		reply(w, h.log, status, v, err)
	}
}

// Routes подключает маршруты (внутри группы с авторизацией).
func (h *Mentor) Routes(r chi.Router) {
	ok, created, none := http.StatusOK, http.StatusCreated, http.StatusNoContent
	r.Get("/mentor/stats", h.with(ok, func(r *http.Request, id string) (any, error) { return h.svc.Stats(r.Context(), id) }))
	r.Get("/mentor/metrics", h.with(ok, func(r *http.Request, id string) (any, error) { return h.svc.Metrics(r.Context(), id) }))
	r.Get("/mentor/threads", h.with(ok, func(r *http.Request, id string) (any, error) { return h.svc.Threads(r.Context(), id) }))
	r.Get("/mentor/threads/{internId}", h.with(ok, func(r *http.Request, id string) (any, error) {
		return h.svc.Thread(r.Context(), id, chi.URLParam(r, "internId"))
	}))
	r.Post("/mentor/threads/{internId}/read", h.with(none, func(r *http.Request, id string) (any, error) {
		return nil, h.svc.Read(r.Context(), id, chi.URLParam(r, "internId"))
	}))
	r.Post("/mentor/threads/{internId}/messages", jsonHandler(h, ok, func(r *http.Request, id string, in *services.ReplyInput) (any, error) {
		return h.svc.Reply(r.Context(), id, chi.URLParam(r, "internId"), *in)
	}))
	r.Get("/me/mentor-thread", h.with(ok, func(r *http.Request, id string) (any, error) { return h.svc.MyThread(r.Context(), id) }))
	r.Post("/me/mentor-thread/messages", jsonHandler(h, ok, func(r *http.Request, id string, in *struct {
		Text string `json:"text"`
	}) (any, error) {
		return h.svc.SendToMentor(r.Context(), id, in.Text)
	}))
	r.Post("/me/mentor-thread/updates/{messageId}/ack", h.with(ok, func(r *http.Request, id string) (any, error) {
		return h.svc.Ack(r.Context(), id, chi.URLParam(r, "messageId"))
	}))
	r.Get("/materials", h.with(ok, func(r *http.Request, id string) (any, error) { return h.svc.Materials(r.Context(), id) }))
	r.Post("/materials", h.with(created, h.material))
	r.Delete("/materials/{id}", h.with(none, func(r *http.Request, id string) (any, error) {
		return nil, h.svc.DeleteMaterial(r.Context(), id, chi.URLParam(r, "id"))
	}))
	r.Get("/mentor/meetings", h.with(ok, func(r *http.Request, id string) (any, error) {
		q := r.URL.Query()
		return h.svc.Meetings(r.Context(), id, q.Get("from"), q.Get("to"))
	}))
	r.Post("/mentor/meetings", jsonHandler(h, created, func(r *http.Request, id string, in *services.MeetingInput) (any, error) {
		return h.svc.CreateMeeting(r.Context(), id, *in)
	}))
	r.Patch("/mentor/meetings/{id}", jsonHandler(h, ok, func(r *http.Request, id string, in *struct {
		StartsAt *time.Time `json:"startsAt"`
	}) (any, error) {
		return h.svc.MoveMeeting(r.Context(), id, chi.URLParam(r, "id"), in.StartsAt)
	}))
	r.Delete("/mentor/meetings/{id}", h.with(none, func(r *http.Request, id string) (any, error) {
		return nil, h.svc.CancelMeeting(r.Context(), id, chi.URLParam(r, "id"))
	}))
	r.Get("/mentor/automations", h.with(ok, func(r *http.Request, id string) (any, error) { return h.svc.Automations(r.Context(), id) }))
	r.Patch("/mentor/automations/{id}", jsonHandler(h, ok, func(r *http.Request, id string, in *struct {
		Enabled *bool `json:"enabled"`
	}) (any, error) {
		return h.svc.SetAutomation(r.Context(), id, chi.URLParam(r, "id"), in.Enabled)
	}))
}

// jsonHandler – как with, но сначала читает тело запроса в T.
func jsonHandler[T any](h *Mentor, status int, fn func(r *http.Request, id string, in *T) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := viewerID(w, r)
		if !ok {
			return
		}
		in := new(T)
		if !decodeJSON(w, r, in) {
			return
		}
		v, err := fn(r, id, in)
		reply(w, h.log, status, v, err)
	}
}

// material – POST /api/materials, multipart: title, stageCode, section, useInAi, notify, file или url.
func (h *Mentor) material(r *http.Request, id string) (any, error) {
	if err := r.ParseMultipartForm(11 << 20); err != nil {
		return nil, &services.ValidationError{Message: "Ожидается multipart/form-data, файл до 10 МБ"}
	}
	f := r.FormValue
	in := services.MaterialInput{Title: f("title"), StageCode: f("stageCode"), Section: f("section"), URL: f("url"),
		UseInAI: f("useInAi") != "false", Notify: f("notify") == "true"}
	if file, hdr, err := r.FormFile("file"); err == nil {
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, 10<<20+1))
		if err != nil {
			return nil, err
		}
		in.FileName, in.MimeType, in.Content = hdr.Filename, hdr.Header.Get("Content-Type"), data
	}
	return h.svc.CreateMaterial(r.Context(), id, in)
}

var _ = httpx.CodeValidation

package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"ai-buddy/backend/internal/models"
)

// KBService – логика справочника.
type KBService interface {
	Get(ctx context.Context, viewerID string) (*models.KnowledgeBase, error)
	Article(ctx context.Context, viewerID, id string) (*models.KBArticleDetail, error)
	Search(ctx context.Context, viewerID, query string, limit int) (*models.KBSearch, error)
}

// KB – обработчики /api/kb.
type KB struct {
	svc KBService
	log *slog.Logger
}

// NewKB создает обработчики справочника.
func NewKB(svc KBService, log *slog.Logger) *KB {
	return &KB{svc: svc, log: log}
}

// Get – GET /api/kb: разделы, статьи и словарик.
func (h *KB) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Get(r.Context(), id)
	reply(w, h.log, http.StatusOK, v, err)
}

// Article – GET /api/kb/articles/{articleId}: статья с текстом.
func (h *KB) Article(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Article(r.Context(), id, chi.URLParam(r, "articleId"))
	reply(w, h.log, http.StatusOK, v, err)
}

// Search – GET /api/kb/search?q=...&limit=10.
func (h *KB) Search(w http.ResponseWriter, r *http.Request) {
	id, ok := viewerID(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := h.svc.Search(r.Context(), id, r.URL.Query().Get("q"), limit)
	reply(w, h.log, http.StatusOK, v, err)
}

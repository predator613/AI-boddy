package handlers

import (
	"net/http"

	"ai-buddy/backend/internal/httpx"
)

// Health – GET /api/health. Нужен Docker healthcheck и фронту.
func Health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

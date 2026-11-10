package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"ai-buddy/backend/internal/httpx"
	"ai-buddy/backend/internal/middleware"
	"ai-buddy/backend/internal/services"
)

// viewerID – id пользователя из токена. false – ответ 401 уже записан.
func viewerID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "unauthorized")
		return "", false
	}
	return id.UserID, true
}

// decodeJSON читает тело запроса. false – ответ 400 уже записан.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeValidation, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// serviceError превращает ошибку сервиса в HTTP-ответ.
func serviceError(w http.ResponseWriter, log *slog.Logger, err error) {
	var ve *services.ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Error(w, http.StatusBadRequest, httpx.CodeValidation, ve.Message)
	case errors.Is(err, services.ErrForbidden):
		httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden, "not allowed for your role")
	case errors.Is(err, services.ErrNotFound):
		httpx.NotFound(w, nil)
	default:
		log.Error("request failed", "err", err)
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternal, "internal server error")
	}
}

// reply пишет JSON-ответ или ошибку сервиса.
func reply(w http.ResponseWriter, log *slog.Logger, status int, body any, err error) {
	if err != nil {
		serviceError(w, log, err)
		return
	}
	httpx.JSON(w, status, body)
}

// replyEmpty – 204 без тела или ошибка сервиса.
func replyEmpty(w http.ResponseWriter, log *slog.Logger, err error) {
	if err != nil {
		serviceError(w, log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

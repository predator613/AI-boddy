// Package httpx – помощники для JSON-ответов в едином формате.
package httpx

import (
	"encoding/json"
	"net/http"
)

// Коды ошибок из контракта API.
const (
	CodeValidation         = "validation_error"
	CodeInvalidCredentials = "invalid_credentials"
	CodeUnauthorized       = "unauthorized"
	CodeNotFound           = "not_found"
	CodeMethodNotAllowed   = "method_not_allowed"
	CodeInternal           = "internal_error"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON пишет любой объект как JSON с нужным статусом.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error пишет ошибку в формате { "error": { "code", "message" } }.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// NotFound – JSON-ответ 404 для неизвестных маршрутов.
func NotFound(w http.ResponseWriter, _ *http.Request) {
	Error(w, http.StatusNotFound, CodeNotFound, "resource not found")
}

// MethodNotAllowed – JSON-ответ 405.
func MethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	Error(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
}

package services

import "errors"

// Общие ошибки бизнес-логики. Обработчики превращают их в HTTP-коды.
var (
	ErrForbidden = errors.New("forbidden") // 403: действие запрещено для роли
	ErrNotFound  = errors.New("not found") // 404: нет такого объекта или он скрыт от пользователя
)

// ValidationError – неверные входные данные (400). Message показывается пользователю.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(msg string) error { return &ValidationError{Message: msg} }

// ConflictError – действие противоречит текущему состоянию (409), например запись в календаре
// пересекается с другой. Message показывается пользователю как есть.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string { return e.Message }

func conflict(msg string) error { return &ConflictError{Message: msg} }

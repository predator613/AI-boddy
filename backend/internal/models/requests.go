package models

import "time"

// ---------- Заявки: внутренние структуры сервиса (наружу отдаются как RequestItem) ----------

// Service – корпоративный сервис и вид его согласующего.
type Service struct {
	Code         string
	Title        string
	RequestTitle string // «Доступ к GitLab»
	ApproverKind string // auto | lead | project_owner | devops
	ApproverID   *string
	SLAHours     int
}

// ServiceRole – роль в сервисе, уровень прав и разрешено ли это роли пользователя.
type ServiceRole struct {
	Code       string
	Title      string // Developer
	Level      string // read | edit | admin
	LevelTitle string // редактирование
	Allowed    bool
	Reason     string // почему нельзя
}

// Project – проект в сервисе (репозиторий, стенд).
type Project struct {
	Code    string
	Title   string
	OwnerID *string
}

// PersonRef уже объявлен в path.go.

// LinkedTask – задача пути, которую закроет заявка.
type LinkedTask struct {
	ID    string
	Title string
}

// AccessDraft – черновик заявки на доступ.
type AccessDraft struct {
	RequesterID   string
	Service       string
	Role          string
	Project       *string
	Justification string
	ApproverID    *string
	TaskID        *string
}

// LeaveDraft – черновик кадровой заявки.
type LeaveDraft struct {
	RequesterID string
	Date        time.Time
	Type        string
	Comment     string
	ApproverID  *string
	TaskID      *string
}

// Request – заявка со всеми полями для карточки.
type Request struct {
	ID           string
	Number       int64
	Type         string // access | leave | other
	Status       string // draft | waiting | approved | done | rejected | cancelled
	Via          string
	RequesterID  string
	TaskID       *string
	Approver     *PersonRef
	ApproverKind string
	ExternalRef  *string
	RejectReason *string
	CreatedAt    time.Time
	DecidedAt    *time.Time

	ServiceCode   string
	ServiceTitle  string
	RequestTitle  string
	RoleTitle     string
	LevelTitle    string
	Level         string
	ProjectTitle  *string
	Justification string
	SLAHours      int

	LeaveDate      *time.Time
	LeaveTypeTitle string
	LeaveComment   string
}

// RequestEvent – шаг истории заявки.
type RequestEvent struct {
	RequestID string
	Status    string
	ActorKind string
	ActorName *string
	Note      string
	At        time.Time
}

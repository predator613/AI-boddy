// Package integrations – адаптер к корпоративным системам заявок (доступы, кадровые заявки).
// В демо за ним стоит имитация mock-services (REST, контракт в api/mock-services.openapi.yaml).
// Для внедрения достаточно новой реализации интерфейса Corporate поверх настоящего API.
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Статусы заявки во внешней системе.
const (
	TicketWaiting  = "waiting"  // ждет согласующего
	TicketApproved = "approved" // одобрена (кадровая заявка)
	TicketGranted  = "granted"  // доступ выдан
	TicketRejected = "rejected" // отклонена, причина в Reason

	// Обращение в поддержку (очередь SUP): waiting → in_progress → done с ответом в Answer.
	TicketInProgress = "in_progress"
	TicketDone       = "done"
)

// JiraName – как сервис заявок называется в интерфейсе.
const JiraName = "Jira Service Desk"

// Order – заявка, которую помощник отправляет во внешнюю систему.
type Order struct {
	Kind          string `json:"kind"` // access | leave | other (заявка в отдел) | support (обращение)
	RequestNumber int64  `json:"requestNumber"`
	Requester     string `json:"requester"`
	Approver      string `json:"approver,omitempty"` // пусто – выдается автоматически
	Service       string `json:"service,omitempty"`
	Role          string `json:"role,omitempty"`
	Project       string `json:"project,omitempty"`
	AccessLevel   string `json:"accessLevel,omitempty"`
	Justification string `json:"justification,omitempty"`
	LeaveDate     string `json:"leaveDate,omitempty"`
	LeaveType     string `json:"leaveType,omitempty"`
	Department    string `json:"department,omitempty"` // заявка в отдел
	Item          string `json:"item,omitempty"`
	Comment       string `json:"comment,omitempty"`
	Auto          bool   `json:"auto"`
	// Manual – решение принимает человек (владелец проекта, руководитель) в AI-buddy, а не сервис.
	Manual  bool   `json:"manual,omitempty"`
	Title   string `json:"title,omitempty"`   // что запрошено, для карточки заявки
	Details string `json:"details,omitempty"` // текст обращения в поддержку
}

// Ticket – состояние заявки во внешней системе.
type Ticket struct {
	Ref       string     `json:"ref"`
	Status    string     `json:"status"`
	Reason    string     `json:"reason,omitempty"`
	Answer    string     `json:"answer,omitempty"` // ответ поддержки
	DecidedAt *time.Time `json:"decidedAt,omitempty"`
}

// ErrUnknownTicket – внешняя система не знает такую заявку.
var ErrUnknownTicket = errors.New("integrations: unknown ticket")

// Corporate – система заявок.
type Corporate interface {
	Submit(ctx context.Context, o Order) (Ticket, error)
	Ticket(ctx context.Context, ref string) (Ticket, error)
	// Decide передает в сервис решение согласующего, принятое в AI-buddy.
	Decide(ctx context.Context, ref string, approve bool, reason string) (Ticket, error)
}

// HTTP – клиент REST-API системы заявок.
type HTTP struct {
	base   string
	client *http.Client
}

// NewHTTP создает клиента. base – адрес вида http://mock-services:8090.
func NewHTTP(base string, client *http.Client) *HTTP {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &HTTP{base: strings.TrimRight(base, "/"), client: client}
}

// Submit создает заявку.
func (h *HTTP) Submit(ctx context.Context, o Order) (Ticket, error) {
	body, err := json.Marshal(o)
	if err != nil {
		return Ticket{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.base+"/v1/tickets", bytes.NewReader(body))
	if err != nil {
		return Ticket{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	return h.do(req, http.StatusCreated)
}

// Ticket возвращает состояние заявки.
func (h *HTTP) Ticket(ctx context.Context, ref string) (Ticket, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.base+"/v1/tickets/"+ref, nil)
	if err != nil {
		return Ticket{}, err
	}
	return h.do(req, http.StatusOK)
}

// Decide передает решение согласующего.
func (h *HTTP) Decide(ctx context.Context, ref string, approve bool, reason string) (Ticket, error) {
	body, err := json.Marshal(map[string]any{"approve": approve, "reason": reason})
	if err != nil {
		return Ticket{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.base+"/v1/tickets/"+ref+"/decision", bytes.NewReader(body))
	if err != nil {
		return Ticket{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	return h.do(req, http.StatusOK)
}

func (h *HTTP) do(req *http.Request, want int) (Ticket, error) {
	resp, err := h.client.Do(req)
	if err != nil {
		return Ticket{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Ticket{}, ErrUnknownTicket
	}
	if resp.StatusCode != want {
		return Ticket{}, fmt.Errorf("integrations: %s %s: status %d", req.Method, req.URL.Path, resp.StatusCode)
	}
	var t Ticket
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return Ticket{}, fmt.Errorf("integrations: bad response: %w", err)
	}
	return t, nil
}

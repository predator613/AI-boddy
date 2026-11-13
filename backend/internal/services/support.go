package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"ai-buddy/backend/internal/integrations"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// SupportStore – обращения в поддержку в базе.
type SupportStore interface {
	CreateTicket(ctx context.Context, d models.SupportDraft) (*models.SupportTicket, error)
	Tickets(ctx context.Context, userID string) ([]models.SupportTicket, error)
	OpenTickets(ctx context.Context, limit int) ([]repository.OpenTicket, error)
	UpdateTicket(ctx context.Context, id, status string, answer *string) error
}

// SupportInput – форма «Сообщить о проблеме».
type SupportInput struct {
	Type         string  `json:"type"`
	Details      string  `json:"details"`
	InternID     *string `json:"internId"`
	AttachDialog bool    `json:"attachDialog"`
}

// Support – обращения в поддержку. Уходят в сервис заявок (очередь SUP) тем же адаптером, что и
// заявки на доступ; номер обращения берется оттуда.
type Support struct {
	store SupportStore
	users UserLookup
	corp  integrations.Corporate
	log   *slog.Logger
}

// NewSupport создает сервис.
func NewSupport(store SupportStore, users UserLookup, corp integrations.Corporate, log *slog.Logger) *Support {
	return &Support{store: store, users: users, corp: corp, log: log}
}

// Create отправляет обращение.
func (s *Support) Create(ctx context.Context, viewerID string, in SupportInput) (*models.SupportTicket, error) {
	u, err := s.users.GetByID(ctx, viewerID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	d := models.SupportDraft{UserID: u.ID, Type: strings.TrimSpace(in.Type), Details: strings.TrimSpace(in.Details), AttachDialog: in.AttachDialog}
	switch {
	case d.Type == "" || len([]rune(d.Type)) > 120:
		return nil, invalid("Выбери, что случилось")
	case d.Details == "":
		return nil, invalid("Опиши, что случилось")
	case len([]rune(d.Details)) > 2000:
		return nil, invalid("Описание слишком длинное (до 2000 символов)")
	}
	if in.InternID != nil && strings.TrimSpace(*in.InternID) != "" {
		if u.Role == models.RoleIntern {
			return nil, invalid("internId is only for mentors and leads")
		}
		intern, err := s.users.GetByID(ctx, strings.TrimSpace(*in.InternID))
		if errors.Is(err, repository.ErrNotFound) || (err == nil && !CanViewIntern(u, intern)) {
			return nil, invalid("Стажер не найден")
		}
		if err != nil {
			return nil, err
		}
		d.InternID = &intern.ID
	}
	if u.Role != models.RoleIntern {
		d.AttachDialog = false // диалог с AI-buddy есть только у стажера
	}
	t, err := s.corp.Submit(ctx, integrations.Order{Kind: "support", Requester: u.Login, Title: d.Type, Details: d.Details})
	if err != nil {
		s.log.Warn("support ticket was not sent", "err", err)
		return nil, ErrUnavailable
	}
	d.ExternalRef = t.Ref
	return s.store.CreateTicket(ctx, d)
}

// List – мои обращения, новые сверху.
func (s *Support) List(ctx context.Context, viewerID string) ([]models.SupportTicket, error) {
	if _, err := s.users.GetByID(ctx, viewerID); err != nil {
		return nil, mapNotFound(err)
	}
	return s.store.Tickets(ctx, viewerID)
}

// SyncOnce забирает статусы обращений из сервиса заявок. Возвращает, сколько изменилось.
func (s *Support) SyncOnce(ctx context.Context) (int, error) {
	list, err := s.store.OpenTickets(ctx, 50)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, t := range list {
		ext, err := s.corp.Ticket(ctx, t.Ref)
		if errors.Is(err, integrations.ErrUnknownTicket) {
			continue
		}
		if err != nil {
			return changed, err
		}
		status := map[string]string{integrations.TicketInProgress: "in_progress", integrations.TicketDone: "done"}[ext.Status]
		if status == "" || status == t.Status {
			continue
		}
		var answer *string
		if status == "done" && ext.Answer != "" {
			answer = &ext.Answer
		}
		if err := s.store.UpdateTicket(ctx, t.ID, status, answer); err != nil {
			return changed, err
		}
		changed++
	}
	return changed, nil
}

// Run синхронизирует обращения каждые interval, пока не отменен ctx.
func (s *Support) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := s.SyncOnce(ctx); err != nil {
				s.log.Warn("support sync failed", "err", err)
			}
		}
	}
}

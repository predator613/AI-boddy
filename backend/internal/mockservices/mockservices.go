// Package mockservices – имитация корпоративной системы заявок (доступы и кадровые заявки).
// Хранит заявки в памяти, автоматически выдает то, что не требует согласования, остальное
// «согласует» через DecisionDelay. Решение можно принять и вручную: POST /v1/tickets/{ref}/decision.
// Код имитации не входит в расчет покрытия прикладного кода.
package mockservices

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"ai-buddy/backend/internal/integrations"
)

//go:embed openapi.yaml
var openAPI []byte

type ticket struct {
	integrations.Ticket
	order    integrations.Order
	decideAt time.Time
}

// Store – заявки в памяти.
type Store struct {
	mu            sync.Mutex
	tickets       map[string]*ticket
	seq           int
	DecisionDelay time.Duration // через сколько согласующий «одобрит» заявку; 0 – только вручную
	now           func() time.Time
}

// NewStore создает хранилище.
func NewStore(delay time.Duration) *Store {
	return &Store{tickets: map[string]*ticket{}, seq: 8000, DecisionDelay: delay, now: time.Now}
}

// Submit принимает заявку (реализует integrations.Corporate внутри одного процесса).
func (s *Store) Submit(_ context.Context, o integrations.Order) (integrations.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	prefix := map[string]string{"access": "ACC", "leave": "HR", "other": "SD"}[o.Kind]
	t := &ticket{order: o}
	t.Ref = fmt.Sprintf("%s-%d", prefix, s.seq)
	t.Status = integrations.TicketWaiting
	now := s.now()
	if o.Auto {
		t.Status = final(o)
		t.DecidedAt = &now
	} else if s.DecisionDelay > 0 {
		t.decideAt = now.Add(s.DecisionDelay)
	}
	s.tickets[t.Ref] = t
	return t.Ticket, nil
}

// Ticket возвращает состояние; ожидающую заявку «согласует», если подошел срок.
func (s *Store) Ticket(_ context.Context, ref string) (integrations.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[ref]
	if !ok {
		return integrations.Ticket{}, integrations.ErrUnknownTicket
	}
	if t.Status == integrations.TicketWaiting && !t.decideAt.IsZero() && !s.now().Before(t.decideAt) {
		now := s.now()
		t.Status, t.DecidedAt = final(t.order), &now
	}
	return t.Ticket, nil
}

// Decide – ручное решение согласующего.
func (s *Store) Decide(ref string, approve bool, reason string) (integrations.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[ref]
	if !ok {
		return integrations.Ticket{}, integrations.ErrUnknownTicket
	}
	now := s.now()
	t.DecidedAt = &now
	if approve {
		t.Status, t.Reason = final(t.order), ""
	} else {
		t.Status, t.Reason = integrations.TicketRejected, strings.TrimSpace(reason)
		if t.Reason == "" {
			t.Reason = "Отклонено согласующим"
		}
	}
	return t.Ticket, nil
}

func final(o integrations.Order) string {
	if o.Kind == "leave" {
		return integrations.TicketApproved
	}
	return integrations.TicketGranted
}

// Handler – REST-API имитации.
func Handler(s *Store) http.Handler {
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	notFound := func(w http.ResponseWriter) {
		write(w, http.StatusNotFound, map[string]string{"error": "ticket not found"})
	}
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(openAPI)
	})
	mux.HandleFunc("POST /v1/tickets", func(w http.ResponseWriter, r *http.Request) {
		var o integrations.Order
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&o); err != nil || (o.Kind != "access" && o.Kind != "leave" && o.Kind != "other") {
			write(w, http.StatusBadRequest, map[string]string{"error": "kind must be access, leave or other"})
			return
		}
		t, _ := s.Submit(r.Context(), o)
		write(w, http.StatusCreated, t)
	})
	mux.HandleFunc("GET /v1/tickets/{ref}", func(w http.ResponseWriter, r *http.Request) {
		t, err := s.Ticket(r.Context(), r.PathValue("ref"))
		if err != nil {
			notFound(w)
			return
		}
		write(w, http.StatusOK, t)
	})
	mux.HandleFunc("POST /v1/tickets/{ref}/decision", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Approve bool   `json:"approve"`
			Reason  string `json:"reason"`
		}
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&in)
		t, err := s.Decide(r.PathValue("ref"), in.Approve, in.Reason)
		if err != nil {
			notFound(w)
			return
		}
		write(w, http.StatusOK, t)
	})
	return mux
}

// Package mockservices – имитация корпоративной системы заявок (Jira Service Desk): доступы (ACCESS),
// заявки в отделы (SD), кадровые заявки (HR) и обращения в поддержку (SUP).
// Хранит заявки в памяти, автоматически выдает то, что не требует согласования, остальное
// «согласует» через DecisionDelay. Заявки, которые решает человек в AI-buddy (Manual), ждут в
// manualFactor раз дольше – чтобы демо стажера не зависало, если руководитель так и не зашел.
// Решение можно принять и вручную: POST /v1/tickets/{ref}/decision. Карточка заявки: GET /browse/{ref}.
// Код имитации не входит в расчет покрытия прикладного кода.
package mockservices

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
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
	order     integrations.Order
	createdAt time.Time
	decideAt  time.Time
}

// manualFactor – во сколько раз дольше ждет заявка, которую решает человек.
const manualFactor = 6

// SupportAnswer – ответ поддержки в имитации.
const SupportAnswer = "Исправили, проверь, пожалуйста"

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

// SetNow подменяет часы (для тестов).
func (s *Store) SetNow(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

// Submit принимает заявку (реализует integrations.Corporate внутри одного процесса).
func (s *Store) Submit(_ context.Context, o integrations.Order) (integrations.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	prefix := map[string]string{"access": "ACCESS", "leave": "HR", "other": "SD", "support": "SUP"}[o.Kind]
	now := s.now()
	t := &ticket{order: o, createdAt: now}
	t.Ref = fmt.Sprintf("%s-%d", prefix, s.seq)
	t.Status = integrations.TicketWaiting
	if o.Auto {
		t.Status = final(o)
		t.DecidedAt = &now
	} else if s.DecisionDelay > 0 {
		t.decideAt = now.Add(s.DecisionDelay)
		if o.Manual {
			t.decideAt = now.Add(manualFactor * s.DecisionDelay)
		}
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
	s.advance(t)
	return t.Ticket, nil
}

// advance двигает заявку по времени: обращение в поддержку берут в работу через DecisionDelay и
// решают еще через один, остальные заявки «согласуются» в назначенный срок.
func (s *Store) advance(t *ticket) {
	if t.decideAt.IsZero() || t.DecidedAt != nil {
		return
	}
	now := s.now()
	if t.order.Kind == "support" {
		switch {
		case !now.Before(t.decideAt.Add(s.DecisionDelay)):
			t.Status, t.Answer, t.DecidedAt = integrations.TicketDone, SupportAnswer, &now
		case !now.Before(t.decideAt):
			t.Status = integrations.TicketInProgress
		}
		return
	}
	if t.Status == integrations.TicketWaiting && !now.Before(t.decideAt) {
		t.Status, t.DecidedAt = final(t.order), &now
	}
}

// Decide (integrations.Corporate) – решение согласующего из AI-buddy.
func (s *Store) Decide(_ context.Context, ref string, approve bool, reason string) (integrations.Ticket, error) {
	return s.Resolve(ref, approve, reason)
}

// Resolve – ручное решение: согласующий по заявке или ответ поддержки по обращению.
func (s *Store) Resolve(ref string, approve bool, reason string) (integrations.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[ref]
	if !ok {
		return integrations.Ticket{}, integrations.ErrUnknownTicket
	}
	now := s.now()
	t.DecidedAt = &now
	if t.order.Kind == "support" {
		t.Status, t.Answer = integrations.TicketDone, strings.TrimSpace(reason)
		if t.Answer == "" {
			t.Answer = SupportAnswer
		}
		return t.Ticket, nil
	}
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
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&o); err != nil ||
			(o.Kind != "access" && o.Kind != "leave" && o.Kind != "other" && o.Kind != "support") {
			write(w, http.StatusBadRequest, map[string]string{"error": "kind must be access, leave, other or support"})
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
		t, err := s.Resolve(r.PathValue("ref"), in.Approve, in.Reason)
		if err != nil {
			notFound(w)
			return
		}
		write(w, http.StatusOK, t)
	})
	mux.HandleFunc("GET /browse/{ref}", func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.card(r.PathValue("ref"))
		if !ok {
			http.Error(w, "Заявка не найдена: имитация хранит заявки в памяти и забывает их при перезапуске.", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = browsePage.Execute(w, v)
	})
	return mux
}

type cardRow struct{ Label, Value string }

type cardView struct {
	Ref, Title, Queue, Status, StatusKind string
	Rows                                  []cardRow
}

// card собирает страницу заявки.
func (s *Store) card(ref string) (cardView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[ref]
	if !ok {
		return cardView{}, false
	}
	s.advance(t)
	o := t.order
	v := cardView{Ref: t.Ref, Title: o.Title}
	v.Queue = map[string]string{"access": "Доступы", "leave": "Кадровые заявки", "other": "Заявки в отделы", "support": "Поддержка AI-buddy"}[o.Kind]
	if v.Title == "" {
		v.Title = v.Queue
	}
	v.Status = map[string]string{
		integrations.TicketWaiting: "Ожидает решения", integrations.TicketApproved: "Согласована",
		integrations.TicketGranted: "Выполнена", integrations.TicketRejected: "Отклонена",
		integrations.TicketInProgress: "В работе", integrations.TicketDone: "Решено",
	}[t.Status]
	v.StatusKind = map[string]string{
		integrations.TicketWaiting: "wait", integrations.TicketInProgress: "wait", integrations.TicketRejected: "bad",
	}[t.Status]
	if v.StatusKind == "" {
		v.StatusKind = "ok"
	}
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			v.Rows = append(v.Rows, cardRow{label, value})
		}
	}
	add("Автор", o.Requester)
	add("Ресурс", o.Service)
	add("Проект", o.Project)
	add("Уровень прав", o.AccessLevel)
	add("Обоснование", o.Justification)
	add("Отдел", o.Department)
	add("Что нужно", o.Item)
	add("Дата", o.LeaveDate)
	add("Тип", o.LeaveType)
	add("Комментарий", o.Comment)
	add("Описание", o.Details)
	switch {
	case o.Auto:
		add("Исполнитель", "выдается автоматически")
	case o.Manual:
		add("Согласующий", o.Approver)
	case o.Kind == "access":
		add("Исполнитель", "команда доступов")
	default:
		add("Исполнитель", o.Approver)
	}
	if o.RequestNumber != 0 {
		add("Номер в AI-buddy", fmt.Sprint(o.RequestNumber))
	}
	add("Создана", t.createdAt.Format("02.01.2006 15:04"))
	add("Причина отказа", t.Reason)
	add("Ответ", t.Answer)
	return v, true
}

var browsePage = template.Must(template.New("browse").Parse(`<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><title>{{.Ref}} – Jira Service Desk (имитация)</title>
<style>
body{font:15px/1.5 -apple-system,Segoe UI,Roboto,sans-serif;margin:0;background:#f4f5f7;color:#172b4d}
header{background:#0052cc;color:#fff;padding:12px 32px;font-weight:600}
header small{font-weight:400;opacity:.8;margin-left:12px}
main{max-width:720px;margin:32px auto;background:#fff;border-radius:8px;padding:28px 32px;box-shadow:0 1px 3px rgba(9,30,66,.2)}
.key{color:#5e6c84;font-size:13px}
h1{font-size:22px;margin:4px 0 16px}
.st{display:inline-block;padding:2px 10px;border-radius:4px;font-size:12px;font-weight:700;text-transform:uppercase}
.st.wait{background:#deebff;color:#0747a6}.st.ok{background:#e3fcef;color:#006644}.st.bad{background:#ffebe6;color:#bf2600}
table{border-collapse:collapse;width:100%;margin-top:20px}
td{padding:8px 0;border-top:1px solid #ebecf0;vertical-align:top}
td:first-child{color:#5e6c84;width:200px}
footer{max-width:720px;margin:0 auto 32px;color:#5e6c84;font-size:12px;padding:0 32px}
</style></head><body>
<header>Jira Service Desk<small>имитация для демо AI-buddy, данные синтетические</small></header>
<main>
<div class="key">{{.Queue}} / {{.Ref}}</div>
<h1>{{.Title}}</h1>
<span class="st {{.StatusKind}}">{{.Status}}</span>
<table>{{range .Rows}}<tr><td>{{.Label}}</td><td>{{.Value}}</td></tr>{{end}}</table>
</main>
<footer>Страница обновляется при перезагрузке. Решение по заявке AI-buddy забирает сам.</footer>
</body></html>`))

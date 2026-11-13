package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"ai-buddy/backend/internal/integrations"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// RequestStore – заявки в базе.
type RequestStore interface {
	ResolveService(ctx context.Context, text string) (string, error)
	Service(ctx context.Context, code string) (*models.Service, error)
	ServiceRoles(ctx context.Context, service, userRole string) ([]models.ServiceRole, error)
	TeamProject(ctx context.Context, service, teamID string) (*models.Project, error)
	Project(ctx context.Context, service, code string) (*models.Project, error)
	TeamLead(ctx context.Context, teamID string) (*models.PersonRef, error)
	Person(ctx context.Context, id string) (*models.PersonRef, error)
	LinkedAccessTask(ctx context.Context, internID, service string, taskID *string) (*models.LinkedTask, error)
	OpenRequest(ctx context.Context, requesterID, service string) (*models.Request, error)
	CreateAccessDraft(ctx context.Context, d models.AccessDraft) (*models.Request, error)
	CreateLeaveDraft(ctx context.Context, d models.LeaveDraft) (*models.Request, error)
	ResolveItem(ctx context.Context, text string) (string, error)
	CatalogItem(ctx context.Context, code string) (*models.CatalogItem, error)
	OpenServiceRequest(ctx context.Context, requesterID, item string) (*models.Request, error)
	CreateServiceDraft(ctx context.Context, d models.ServiceDraft) (*models.Request, error)
	Notify(ctx context.Context, userID, kind, text, link string, requestID *string) error
	Submit(ctx context.Context, id, actorID, status, externalRef, taskNote string) error
	Cancel(ctx context.Context, id, actorID string) error
	ApplyDecision(ctx context.Context, id, status string, reason *string, taskNote, notice string, actorID *string) (bool, error)
	Get(ctx context.Context, id string) (*models.Request, error)
	ListByRequester(ctx context.Context, requesterID string) ([]*models.Request, error)
	Waiting(ctx context.Context, limit int) ([]*models.Request, error)
	Events(ctx context.Context, ids []string) ([]models.RequestEvent, error)
}

// Requests – страница «Заявки» и синхронизация статусов с внешней системой.
type Requests struct {
	store RequestStore
	users UserLookup
	corp  integrations.Corporate
	log   *slog.Logger
	now   func() time.Time
	feed  FeedWriter
}

// WithFeed подключает ленту наставника: в нее пишутся решения сервиса заявок.
func (s *Requests) WithFeed(w FeedWriter) *Requests {
	s.feed = w
	return s
}

// NewRequests создает сервис заявок.
func NewRequests(store RequestStore, users UserLookup, corp integrations.Corporate, log *slog.Logger) *Requests {
	return &Requests{store: store, users: users, corp: corp, log: log, now: time.Now}
}

// List – мои заявки, новые сверху.
func (s *Requests) List(ctx context.Context, viewerID string) ([]models.RequestItem, error) {
	if _, err := s.users.GetByID(ctx, viewerID); err != nil {
		return nil, mapNotFound(err)
	}
	list, err := s.store.ListByRequester(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(list))
	for i, q := range list {
		ids[i] = q.ID
	}
	events, err := s.store.Events(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := map[string][]models.RequestEvent{}
	for _, e := range events {
		byID[e.RequestID] = append(byID[e.RequestID], e)
	}
	out := make([]models.RequestItem, len(list))
	for i, q := range list {
		out[i] = RequestItem(q, byID[q.ID], s.now())
	}
	return out, nil
}

// SyncOnce забирает решения по ожидающим заявкам. Возвращает, сколько заявок изменилось.
func (s *Requests) SyncOnce(ctx context.Context) (int, error) {
	list, err := s.store.Waiting(ctx, 50)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, q := range list {
		t, err := s.corp.Ticket(ctx, *q.ExternalRef)
		if errors.Is(err, integrations.ErrUnknownTicket) {
			continue // заявка из демо-данных или система перезапускалась
		}
		if err != nil {
			return changed, err
		}
		status, ok := ticketStatus(q.Type, t.Status)
		if !ok || status == "waiting" {
			continue
		}
		var reason *string
		if status == "rejected" {
			r := t.Reason
			reason = &r
		}
		applied, err := s.store.ApplyDecision(ctx, q.ID, status, reason, taskNote(q, status, t.Reason), decisionNotice(q, status, t.Reason), nil)
		if err != nil {
			return changed, err
		}
		if applied {
			changed++
			word := "выполнена"
			if status == "rejected" {
				word = "отклонена"
			}
			pushFeed(ctx, s.feed, s.log, q.RequesterID, models.Notice{
				Kind: "requests", Link: "/interns/" + q.RequesterID, RequestID: &q.ID,
				Text:      fmt.Sprintf("Заявка стажера «%s» %s", RequestTitle(q), word),
				ActorKind: "other", ActorName: integrations.JiraName,
			})
		}
	}
	return changed, nil
}

// Run синхронизирует статусы каждые interval, пока не отменен ctx.
func (s *Requests) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n, err := s.SyncOnce(ctx); err != nil {
				s.log.Warn("request sync failed", "err", err)
			} else if n > 0 {
				s.log.Info("request statuses updated", "count", n)
			}
		}
	}
}

// ticketStatus переводит статус внешней системы в статус заявки AI-buddy.
func ticketStatus(kind, ticket string) (string, bool) {
	switch ticket {
	case integrations.TicketWaiting:
		return "waiting", true
	case integrations.TicketRejected:
		return "rejected", true
	case integrations.TicketGranted:
		return "done", true
	case integrations.TicketApproved:
		if kind == "access" {
			return "done", true
		}
		return "approved", true
	}
	return "", false
}

// ---------- тексты ----------

// RequestTitle – «Доступ к GitLab» или «Отгул 14 ноября».
func RequestTitle(q *models.Request) string {
	if q.Type == "leave" && q.LeaveDate != nil {
		return capitalize(q.LeaveTypeTitle) + " " + dayMonth(*q.LeaveDate)
	}
	if q.RequestTitle != "" {
		return q.RequestTitle
	}
	return "Заявка"
}

func decisionNotice(q *models.Request, status, reason string) string {
	title := RequestTitle(q)
	switch status {
	case "done":
		if q.Type == "other" {
			return fmt.Sprintf("Готово: %s выполнил заявку №%d «%s».", q.DepartmentTitle, q.Number, title)
		}
		return fmt.Sprintf("Готово: %s выдан. Заявка №%d закрыта.", lowerFirst(title), q.Number)
	case "approved":
		return fmt.Sprintf("%s согласован. Заявка №%d.", title, q.Number)
	}
	return fmt.Sprintf("Заявка №%d «%s» отклонена: %s. Могу подсказать, что делать дальше.", q.Number, title, reason)
}

func taskNote(q *models.Request, status, reason string) string {
	switch status {
	case "done", "approved":
		return fmt.Sprintf("Заявка №%d выполнена", q.Number)
	case "rejected":
		return fmt.Sprintf("Заявка №%d отклонена: %s", q.Number, reason)
	}
	return ""
}

// accessTeam – кто решает обычные доступы.
const accessTeam = integrations.JiraName + ", команда доступов"

// viaService – заявку на доступ решает сервис заявок, а не человек: обычные доступы (devops) и те,
// у кого согласующий-человек не нашелся.
func viaService(q *models.Request) bool {
	return q.Type == "access" && q.ApproverKind != "auto" && (q.ApproverKind == "devops" || q.Approver == nil)
}

// jiraURL – адрес карточек заявок для браузера (MOCK_SERVICES_PUBLIC_URL); пусто – ссылок нет.
var jiraURL string

// SetJiraURL задает адрес карточек заявок.
func SetJiraURL(base string) { jiraURL = strings.TrimRight(base, "/") }

// External – заявка в Jira Service Desk. Отгулы остаются у руководителя, у них ссылки нет.
func External(q *models.Request) *models.ExternalRef {
	if q.ExternalRef == nil || *q.ExternalRef == "" || q.Type == "leave" {
		return nil
	}
	e := &models.ExternalRef{System: integrations.JiraName, Key: *q.ExternalRef}
	if jiraURL != "" {
		u := jiraURL + "/browse/" + *q.ExternalRef
		e.URL = &u
	}
	return e
}

// approverText – «Олег, владелец проекта», «Jira Service Desk, команда доступов».
func approverText(q *models.Request) string {
	if q.ApproverKind == "department" {
		return q.DepartmentTitle
	}
	if viaService(q) {
		return accessTeam
	}
	if q.ApproverKind == "auto" || q.Approver == nil {
		return "выдается автоматически"
	}
	role := map[string]string{"lead": "руководитель", "project_owner": "владелец проекта"}[q.ApproverKind]
	if q.Type == "leave" {
		role = "руководитель"
	}
	if role == "" {
		return q.Approver.Name
	}
	return q.Approver.Name + ", " + role
}

// levelText – «редактирование (Developer)».
func levelText(q *models.Request) string {
	if q.RoleTitle == "" || strings.EqualFold(q.RoleTitle, q.LevelTitle) {
		return q.LevelTitle
	}
	return q.LevelTitle + " (" + q.RoleTitle + ")"
}

// RequestFields – строки карточки заявки (одинаковые в чате и на странице «Заявки»).
func RequestFields(q *models.Request) []models.Field {
	var f []models.Field
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			f = append(f, models.Field{Label: label, Value: value})
		}
	}
	if q.Type == "leave" {
		if q.LeaveDate != nil {
			add("Дата", dayMonth(*q.LeaveDate)+", "+weekday(*q.LeaveDate))
		}
		add("Тип", q.LeaveTypeTitle)
		add("Комментарий", q.LeaveComment)
	} else if q.Type == "other" {
		add("Что нужно", q.ItemTitle)
		add("Комментарий", q.ItemComment)
	} else {
		add("Ресурс", q.ServiceTitle)
		if q.ProjectTitle != nil {
			add("Проект", *q.ProjectTitle)
		}
		add("Уровень прав", levelText(q))
		add("Обоснование", q.Justification)
	}
	label := "Согласует"
	switch q.Status {
	case "approved", "done":
		label = "Согласовал"
	case "rejected":
		label = "Решение"
	}
	if q.ApproverKind == "auto" && q.Type == "access" {
		label = "Согласование"
	}
	if viaService(q) {
		label = "Куда ушла"
		if q.Status == "done" {
			label = "Выполнил"
		}
	}
	if q.Type == "other" {
		label = "Исполняет"
		if q.Status == "done" {
			label = "Выполнил"
		}
	}
	add(label, approverText(q))
	return f
}

// RequestItem – заявка в формате страницы «Заявки».
func RequestItem(q *models.Request, events []models.RequestEvent, now time.Time) models.RequestItem {
	item := models.RequestItem{
		ID: q.ID, Number: fmt.Sprint(q.Number), Kind: q.Type, Title: RequestTitle(q),
		Status: q.Status, CreatedAt: q.CreatedAt, Via: q.Via, Fields: RequestFields(q),
		RejectReason: q.RejectReason, History: []models.HistoryEntry{}, External: External(q),
	}
	for _, e := range events {
		h := models.HistoryEntry{At: stamp(e.At), State: "done"}
		switch e.Status {
		case "waiting":
			h.Title = map[string]string{"buddy": "Создана AI-buddy", "auto": "Создана при выходе на стажировку", "manual": "Создана вручную"}[q.Via]
		case "approved":
			h.Title = "Одобрена"
		case "done":
			h.Title = "Доступ выдан"
			if q.Type != "access" {
				h.Title = "Выполнена"
			}
		case "rejected":
			h.Title, h.State = "Отклонена", "rejected"
		default:
			continue
		}
		if h.Title == "" {
			h.Title = "Создана"
		}
		item.History = append(item.History, h)
		if e.Status == "waiting" && item.External != nil {
			item.History = append(item.History, models.HistoryEntry{Title: "Передана в " + integrations.JiraName, At: h.At, State: "done"})
		}
	}
	if q.Status == "waiting" {
		at := fmt.Sprintf("обычно до %d ч", q.SLAHours)
		if now.Sub(q.CreatedAt) > time.Duration(q.SLAHours)*time.Hour {
			at = "дольше обычного, AI-buddy напомнит"
		}
		item.History = append(item.History, models.HistoryEntry{Title: "Ждет решения: " + approverText(q), At: at, State: "now"})
	}
	return item
}

// ---------- даты по-русски (Москва) ----------

var moscow = time.FixedZone("MSK", 3*60*60)

var monthsGen = []string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
var monthsShort = []string{"янв", "фев", "мар", "апр", "мая", "июн", "июл", "авг", "сен", "окт", "нояб", "дек"}
var weekdays = []string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}

func dayMonth(t time.Time) string { return fmt.Sprintf("%d %s", t.Day(), monthsGen[t.Month()-1]) }
func weekday(t time.Time) string  { return weekdays[t.Weekday()] }

// stamp – «9 нояб, 11:05».
func stamp(t time.Time) string {
	t = t.In(moscow)
	return fmt.Sprintf("%d %s, %02d:%02d", t.Day(), monthsShort[t.Month()-1], t.Hour(), t.Minute())
}

func capitalize(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

func lowerFirst(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return strings.ToLower(string(r[0])) + string(r[1:])
}

var _ = repository.ErrNotFound

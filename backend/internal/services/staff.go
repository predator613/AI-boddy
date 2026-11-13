package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"ai-buddy/backend/internal/integrations"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// FeedWriter – запись в ленту наставника стажера (колокольчик).
type FeedWriter interface {
	PushToMentor(ctx context.Context, internID string, n models.Notice) error
}

// StaffStore – лента, согласования, чек-лист наставника и конспект встреч в базе.
type StaffStore interface {
	FeedWriter
	Feed(ctx context.Context, userID string, limit int) (*models.Feed, error)
	ReadFeed(ctx context.Context, userID string) error
	Approvals(ctx context.Context, approverID string) ([]*models.Request, error)
	Names(ctx context.Context, ids []string) (map[string]string, error)
	Checklist(ctx context.Context, mentorID, internID string) ([]models.ChecklistItem, error)
	MarkChecklist(ctx context.Context, mentorID, internID, itemID string, done bool) error
	Notes(ctx context.Context, internID string) ([]models.MeetingNote, error)
	OwnEvent(ctx context.Context, internID, eventID string) (bool, error)
	SaveNote(ctx context.Context, internID, eventID, text string) (*models.MeetingNote, error)
}

// DecisionStore – заявки для согласований.
type DecisionStore interface {
	Get(ctx context.Context, id string) (*models.Request, error)
	ApplyDecision(ctx context.Context, id, status string, reason *string, taskNote, notice string, actorID *string) (bool, error)
}

// feedSize – сколько записей показывает колокольчик.
const feedSize = 20

// Staff – то, что видят наставник и руководитель: лента, согласования, чек-лист; и конспект стажера.
type Staff struct {
	store StaffStore
	reqs  DecisionStore
	users UserLookup
	corp  integrations.Corporate
	log   *slog.Logger
}

// NewStaff создает сервис.
func NewStaff(store StaffStore, reqs DecisionStore, users UserLookup, corp integrations.Corporate, log *slog.Logger) *Staff {
	return &Staff{store: store, reqs: reqs, users: users, corp: corp, log: log}
}

func (s *Staff) viewer(ctx context.Context, id string) (*models.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrForbidden
	}
	return u, err
}

// staff – наставник или руководитель; стажеру 403.
func (s *Staff) staff(ctx context.Context, id string) (*models.User, error) {
	u, err := s.viewer(ctx, id)
	if err != nil {
		return nil, err
	}
	if u.Role == models.RoleIntern {
		return nil, ErrForbidden
	}
	return u, nil
}

// ---------- лента ----------

// Feed – колокольчик наставника: последние записи и число непрочитанных.
func (s *Staff) Feed(ctx context.Context, viewerID string) (*models.Feed, error) {
	u, err := s.staff(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	return s.store.Feed(ctx, u.ID, feedSize)
}

// ReadFeed помечает ленту прочитанной.
func (s *Staff) ReadFeed(ctx context.Context, viewerID string) error {
	u, err := s.staff(ctx, viewerID)
	if err != nil {
		return err
	}
	return s.store.ReadFeed(ctx, u.ID)
}

// ---------- согласования ----------

// DecisionInput – решение согласующего.
type DecisionInput struct {
	Decision string `json:"decision"` // approve | reject
	Reason   string `json:"reason"`
}

// Approvals – заявки стажеров, где пользователь согласующий.
func (s *Staff) Approvals(ctx context.Context, viewerID string) ([]models.Approval, error) {
	u, err := s.staff(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	list, err := s.store.Approvals(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, q := range list {
		ids = append(ids, q.RequesterID)
	}
	names, err := s.store.Names(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]models.Approval, len(list))
	for i, q := range list {
		out[i] = approvalView(q, names[q.RequesterID])
	}
	return out, nil
}

// Decide – согласовать или отклонить заявку. Решение уходит в сервис заявок, статус у стажера
// меняется сразу, задача плана закрывается, стажер и его наставник получают уведомление.
func (s *Staff) Decide(ctx context.Context, viewerID, requestID string, in DecisionInput) (*models.Approval, error) {
	u, err := s.staff(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	q, err := s.reqs.Get(ctx, requestID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if q.Approver == nil || q.Approver.ID != u.ID || q.Status == "draft" || q.Status == "cancelled" {
		return nil, ErrNotFound
	}
	reason := strings.TrimSpace(in.Reason)
	switch {
	case in.Decision != "approve" && in.Decision != "reject":
		return nil, invalid("decision must be approve or reject")
	case in.Decision == "reject" && reason == "":
		return nil, invalid("Напиши причину отказа: ее увидит стажер")
	case len([]rune(reason)) > 300:
		return nil, invalid("Причина слишком длинная (до 300 символов)")
	}
	if q.Status != "waiting" {
		return nil, conflict("Решение по этой заявке уже принято")
	}
	approve := in.Decision == "approve"
	if q.ExternalRef != nil && *q.ExternalRef != "" {
		if _, err := s.corp.Decide(ctx, *q.ExternalRef, approve, reason); err != nil && !errors.Is(err, integrations.ErrUnknownTicket) {
			s.log.Warn("decision was not sent to the request service", "err", err)
			return nil, ErrUnavailable
		}
	}
	status, verb := "rejected", "отклонил"
	var why *string
	if approve {
		status, verb, reason = "done", "согласовал", ""
		if q.Type != "access" {
			status = "approved"
		}
	} else {
		why = &reason
	}
	applied, err := s.reqs.ApplyDecision(ctx, q.ID, status, why, taskNote(q, status, reason), decisionNotice(q, status, reason), &u.ID)
	if err != nil {
		return nil, err
	}
	if !applied {
		return nil, conflict("Решение по этой заявке уже принято")
	}
	names, err := s.store.Names(ctx, []string{q.RequesterID})
	if err != nil {
		return nil, err
	}
	intern := names[q.RequesterID]
	pushFeed(ctx, s.store, s.log, q.RequesterID, models.Notice{
		Kind: "requests", Link: "/interns/" + q.RequesterID, RequestID: &q.ID,
		Text:    fmt.Sprintf("%s %s заявку %s «%s»", u.Name, verb, genitive(intern), RequestTitle(q)),
		ActorID: &u.ID, ActorKind: actorKind(u.Role), ActorName: u.Name,
	})
	q.Status, q.RejectReason = status, why
	v := approvalView(q, intern)
	return &v, nil
}

// approvalView – заявка глазами согласующего: те же поля, что у стажера, и почему заявка пришла ему.
func approvalView(q *models.Request, internName string) models.Approval {
	a := models.Approval{
		ID: q.ID, Number: fmt.Sprint(q.Number), InternID: q.RequesterID, InternName: internName,
		Title: RequestTitle(q), CreatedAt: q.CreatedAt, Status: q.Status, RejectReason: q.RejectReason,
		Fields: []models.Field{},
	}
	if a.Status == "done" {
		a.Status = "approved"
	}
	for _, f := range RequestFields(q) {
		switch f.Label {
		case "Согласует", "Согласовал", "Решение":
			continue // согласующий – сам читатель
		}
		a.Fields = append(a.Fields, f)
	}
	if q.Via == "buddy" {
		a.Fields = append(a.Fields, models.Field{Label: "Оформил", Value: "AI-buddy, стажер подтвердил заявку"})
	}
	why := "ты руководитель команды"
	if q.Type == "access" {
		why = "ты владелец проекта"
		if q.ApproverKind != "project_owner" {
			why = "ты руководитель команды"
		}
		if e := External(q); e != nil {
			why += ", заявка " + e.Key + " в " + e.System
		}
	}
	a.Fields = append(a.Fields, models.Field{Label: "Почему ты", Value: why})
	return a
}

// ---------- чек-лист наставника ----------

// internOf – стажер, которого ведет viewer (наставник – своих, руководитель – своей команды).
func (s *Staff) internOf(ctx context.Context, viewerID, internID string) (*models.User, *models.User, error) {
	u, err := s.staff(ctx, viewerID)
	if err != nil {
		return nil, nil, err
	}
	intern, err := s.users.GetByID(ctx, internID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if intern.Role != models.RoleIntern || !CanManageIntern(u, intern) {
		return nil, nil, ErrNotFound
	}
	return u, intern, nil
}

// Checklist – чек-лист наставника по стажеру.
func (s *Staff) Checklist(ctx context.Context, viewerID, internID string) (*models.MentorChecklist, error) {
	u, intern, err := s.internOf(ctx, viewerID, internID)
	if err != nil {
		return nil, err
	}
	return s.checklist(ctx, u, intern)
}

// MarkChecklist ставит или снимает отметку пункта.
func (s *Staff) MarkChecklist(ctx context.Context, viewerID, internID, itemID string, done *bool) (*models.MentorChecklist, error) {
	u, intern, err := s.internOf(ctx, viewerID, internID)
	if err != nil {
		return nil, err
	}
	if done == nil {
		return nil, invalid("done is required")
	}
	if err := s.store.MarkChecklist(ctx, checklistOwner(u, intern), intern.ID, itemID, *done); err != nil {
		return nil, mapNotFound(err)
	}
	return s.checklist(ctx, u, intern)
}

// checklistOwner – чьи отметки показываем: наставника стажера (руководитель видит и правит их же).
func checklistOwner(viewer, intern *models.User) string {
	if viewer.Role == models.RoleLead && intern.MentorID != nil {
		return *intern.MentorID
	}
	return viewer.ID
}

func (s *Staff) checklist(ctx context.Context, viewer, intern *models.User) (*models.MentorChecklist, error) {
	items, err := s.store.Checklist(ctx, checklistOwner(viewer, intern), intern.ID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []models.ChecklistItem{}
	}
	return &models.MentorChecklist{InternID: intern.ID, Items: items}, nil
}

// ---------- конспект встреч ----------

func (s *Staff) intern(ctx context.Context, viewerID string) (*models.User, error) {
	u, err := s.viewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if u.Role != models.RoleIntern {
		return nil, ErrForbidden
	}
	return u, nil
}

// Notes – конспекты встреч стажера. Их видит только он сам.
func (s *Staff) Notes(ctx context.Context, viewerID string) ([]models.MeetingNote, error) {
	u, err := s.intern(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	return s.store.Notes(ctx, u.ID)
}

// SaveNote сохраняет заметку к событию календаря; пустой текст удаляет ее.
func (s *Staff) SaveNote(ctx context.Context, viewerID, eventID string, text *string) (*models.MeetingNote, error) {
	u, err := s.intern(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if text == nil {
		return nil, invalid("text is required")
	}
	t := strings.TrimSpace(*text)
	if len([]rune(t)) > 4000 {
		return nil, invalid("Заметка слишком длинная (до 4000 символов)")
	}
	ok, err := s.store.OwnEvent(ctx, u.ID, eventID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return s.store.SaveNote(ctx, u.ID, eventID, t)
}

// ---------- общее ----------

// pushFeed пишет в ленту наставника; сбой ленты не должен ломать основное действие.
func pushFeed(ctx context.Context, w FeedWriter, log *slog.Logger, internID string, n models.Notice) {
	if w == nil {
		return
	}
	if n.ActorKind == "" {
		n.ActorKind, n.ActorName = "buddy", "AI-buddy"
	}
	if err := w.PushToMentor(ctx, internID, n); err != nil && log != nil {
		log.Warn("mentor feed write failed", "err", err)
	}
}

// actorKind – вид автора записи ленты по роли пользователя.
func actorKind(role string) string {
	switch role {
	case models.RoleIntern:
		return "intern"
	case models.RoleMentor:
		return "mentor"
	case models.RoleLead:
		return "lead"
	}
	return "other"
}

// genitive – имя в родительном падеже: «цели Марии», «заявку Дмитрия».
func genitive(name string) string {
	r := []rune(name)
	if len(r) < 2 {
		return name
	}
	last, prev := r[len(r)-1], r[len(r)-2]
	stem := string(r[:len(r)-1])
	switch last {
	case 'а':
		if strings.ContainsRune("гкхжчшщ", prev) {
			return stem + "и"
		}
		return stem + "ы"
	case 'я':
		return stem + "и"
	case 'й', 'ь':
		return stem + "я"
	}
	if strings.ContainsRune("бвгджзклмнпрстфхцчшщ", last) {
		return name + "а"
	}
	return name
}

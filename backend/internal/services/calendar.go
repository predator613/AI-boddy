package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// Календарь стажера (docs/api-intern-calendar.md): встречи из кабинета наставника и руководителя
// плюс собственные записи стажера. Только для роли intern, каждый видит только свое.

// CalendarStore – что нужно календарю от базы.
type CalendarStore interface {
	Meetings(ctx context.Context, internID string, from, to time.Time) ([]repository.MeetingRow, error)
	Meeting(ctx context.Context, internID, meetingID string) (*repository.MeetingRow, error)
	Entries(ctx context.Context, internID string, from, to time.Time) ([]repository.EntryRow, error)
	Overlap(ctx context.Context, internID string, start, end time.Time) (string, error)
	CreateEntry(ctx context.Context, e models.CalendarEntry) (*repository.EntryRow, error)
	DeleteEntry(ctx context.Context, internID, id string) error
	Answer(ctx context.Context, m repository.MeetingRow, internID, internName, answer, reason string, mentorID *string) error
}

// EntryInput – новая запись стажера.
type EntryInput struct {
	Title           string    `json:"title"`
	Kind            string    `json:"kind"`
	StartsAt        time.Time `json:"startsAt"`
	DurationMin     int       `json:"durationMin"`
	ShareWithMentor *bool     `json:"shareWithMentor"`
}

// AnswerInput – ответ на встречу.
type AnswerInput struct {
	Answer string `json:"answer"` // accept | move
	Reason string `json:"reason"`
}

// Рабочий день в календаре заканчивается в 18:00 по Москве: позже записи не ставим.
const workdayEndMin = 18 * 60

// maxCalendarSpan – самый длинный запрошенный период.
const maxCalendarSpan = 93 * 24 * time.Hour

// Calendar – сервис календаря стажера.
type Calendar struct {
	store  CalendarStore
	users  UserLookup
	people PeopleStore
}

// NewCalendar создает сервис календаря.
func NewCalendar(store CalendarStore, users UserLookup, people PeopleStore) *Calendar {
	return &Calendar{store: store, users: users, people: people}
}

// Events – события стажера с from (включительно) по to (не включительно), по времени начала.
func (s *Calendar) Events(ctx context.Context, viewerID, from, to string) (*models.CalendarView, error) {
	u, err := s.intern(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	start, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return nil, invalid("from must be an ISO 8601 date-time")
	}
	end, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return nil, invalid("to must be an ISO 8601 date-time")
	}
	if !end.After(start) || end.Sub(start) > maxCalendarSpan {
		return nil, invalid("to must be after from, period up to 93 days")
	}
	meetings, err := s.store.Meetings(ctx, u.ID, start, end)
	if err != nil {
		return nil, err
	}
	entries, err := s.store.Entries(ctx, u.ID, start, end)
	if err != nil {
		return nil, err
	}
	out := make([]models.CalendarEvent, 0, len(meetings)+len(entries))
	for _, m := range meetings {
		out = append(out, meetingEvent(m))
	}
	for _, e := range entries {
		out = append(out, entryEvent(e))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	return &models.CalendarView{Events: out}, nil
}

// CreateEntry – своя запись: фокус-время, учеба или личное.
func (s *Calendar) CreateEntry(ctx context.Context, viewerID string, in EntryInput) (*models.CalendarEvent, error) {
	u, err := s.intern(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if n := len([]rune(title)); n < 1 || n > 120 {
		return nil, invalid("title must be 1-120 characters")
	}
	if in.Kind != "focus" && in.Kind != "study" && in.Kind != "personal" {
		return nil, invalid("kind must be focus, study or personal")
	}
	if in.StartsAt.IsZero() {
		return nil, invalid("startsAt is required")
	}
	if in.DurationMin < 15 || in.DurationMin > 480 {
		return nil, invalid("durationMin must be between 15 and 480")
	}
	start := in.StartsAt.In(moscow)
	end := start.Add(time.Duration(in.DurationMin) * time.Minute)
	if end.Format(time.DateOnly) != start.Format(time.DateOnly) || end.Hour()*60+end.Minute() > workdayEndMin {
		return nil, invalid("Запись должна закончиться не позже 18:00")
	}
	title2, err := s.store.Overlap(ctx, u.ID, start, end)
	if err != nil {
		return nil, err
	}
	if title2 != "" {
		return nil, conflict(fmt.Sprintf("Пересекается с «%s»", title2))
	}
	shared := true
	if in.ShareWithMentor != nil {
		shared = *in.ShareWithMentor
	}
	e, err := s.store.CreateEntry(ctx, models.CalendarEntry{
		InternID: u.ID, Title: title, Kind: in.Kind, StartsAt: start, DurationMin: in.DurationMin, Shared: shared,
	})
	if err != nil {
		return nil, mapNotFound(err)
	}
	ev := entryEvent(*e)
	return &ev, nil
}

// DeleteEntry удаляет свою запись. Чужая или несуществующая – 404.
func (s *Calendar) DeleteEntry(ctx context.Context, viewerID, id string) error {
	u, err := s.intern(ctx, viewerID)
	if err != nil {
		return err
	}
	return mapNotFound(s.store.DeleteEntry(ctx, u.ID, id))
}

// Answer – «Приду» или «Попросить перенести». Перенос пишет сообщение в чат с наставником.
func (s *Calendar) Answer(ctx context.Context, viewerID, meetingID string, in AnswerInput) (*models.CalendarEvent, error) {
	u, err := s.intern(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if in.Answer != "accept" && in.Answer != "move" {
		return nil, invalid("answer must be accept or move")
	}
	reason := strings.TrimSpace(in.Reason)
	if len([]rune(reason)) > 200 {
		return nil, invalid("reason is too long (max 200)")
	}
	if in.Answer == "accept" {
		reason = ""
	}
	m, err := s.store.Meeting(ctx, u.ID, meetingID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var mentorID *string
	if c, err := s.people.Circle(ctx, u.ID); err == nil && c.Mentor != nil {
		mentorID = &c.Mentor.ID
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if err := s.store.Answer(ctx, *m, u.ID, u.Name, in.Answer, reason, mentorID); err != nil {
		return nil, mapNotFound(err)
	}
	m.Answer = &in.Answer
	ev := meetingEvent(*m)
	return &ev, nil
}

func (s *Calendar) intern(ctx context.Context, viewerID string) (*models.User, error) {
	u, err := s.users.GetByID(ctx, viewerID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	if u.Role != models.RoleIntern {
		return nil, ErrForbidden
	}
	return u, nil
}

// meetingEvent – встреча в формате календаря. Вид встречи для фронта: команда, знакомство,
// руководитель или наставник (1:1 и разборы).
func meetingEvent(m repository.MeetingRow) models.CalendarEvent {
	kind := "mentor"
	switch {
	case m.Kind == "team" || m.Kind == "intro" || m.Kind == "lead":
		kind = m.Kind
	case m.HostRole == models.RoleLead:
		kind = "lead"
	}
	status := "pending"
	if m.Answer != nil {
		status = map[string]string{"accept": "accepted", "move": "move_requested"}[*m.Answer]
	}
	return models.CalendarEvent{
		ID: m.ID, Title: m.Title, Kind: kind, StartsAt: m.StartsAt.In(moscow), DurationMin: m.DurationMin,
		WithName: m.HostName + ", " + hostTitle(m), Place: m.Place, Note: m.Note, Status: status,
	}
}

func hostTitle(m repository.MeetingRow) string {
	switch m.HostRole {
	case models.RoleMentor:
		return "наставник"
	case models.RoleLead:
		return "руководитель"
	}
	return "команда " + m.Team
}

func entryEvent(e repository.EntryRow) models.CalendarEvent {
	return models.CalendarEvent{
		ID: e.ID, Title: e.Title, Kind: e.Kind, StartsAt: e.StartsAt.In(moscow), DurationMin: e.DurationMin,
		WithName: "только ты", Status: "own", Shared: e.Shared,
	}
}

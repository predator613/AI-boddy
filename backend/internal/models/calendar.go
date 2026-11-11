package models

import "time"

// CalendarEvent – событие календаря стажера: встреча (назначил наставник, руководитель или коллега)
// или собственная запись стажера (фокус-время, учеба, личное).
type CalendarEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Kind        string    `json:"kind"` // mentor | team | lead | intro | focus | study | personal
	StartsAt    time.Time `json:"startsAt"`
	DurationMin int       `json:"durationMin"`
	WithName    string    `json:"withName"`
	Place       string    `json:"place"`
	Note        string    `json:"note"`
	Status      string    `json:"status"` // pending | accepted | move_requested | own
	Shared      bool      `json:"shared"`
}

// CalendarView – ответ GET /api/me/calendar.
type CalendarView struct {
	Events []CalendarEvent `json:"events"`
}

// CalendarEntry – новая собственная запись стажера.
type CalendarEntry struct {
	InternID    string
	Title       string
	Kind        string
	StartsAt    time.Time
	DurationMin int
	Shared      bool
}

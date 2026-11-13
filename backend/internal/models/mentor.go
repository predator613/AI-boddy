package models

import "time"

// ---------- Кабинет наставника (docs/api-mentor-console.md) ----------

// MentorStats – плашки на «Моих стажерах» за periodDays дней.
type MentorStats struct {
	PeriodDays       int `json:"periodDays"`
	AnsweredByBuddy  int `json:"answeredByBuddy"`
	HandedOff        int `json:"handedOff"`
	SavedMinutes     int `json:"savedMinutes"`
	MentorHoursSpent int `json:"mentorHoursSpent"`
	MentorHoursPlan  int `json:"mentorHoursPlan"`
}

// MentorMetrics – польза AI-buddy для наставника (docs/api-lead-feed.md, раздел 11).
type MentorMetrics struct {
	AnsweredByBuddy    int      `json:"answeredByBuddy"`
	Escalated          int      `json:"escalated"`
	HoursSaved         float64  `json:"hoursSaved"`
	MinutesPerQuestion int      `json:"minutesPerQuestion"`
	AccessHours        *float64 `json:"accessHours"`
	OnTimePercent      *int     `json:"onTimePercent"`
}

// ThreadUpdate – карточка изменения от наставника.
type ThreadUpdate struct {
	Type   string  `json:"type"` // plan | material | meeting
	Title  string  `json:"title"`
	Detail string  `json:"detail"`
	Link   *string `json:"link"`
	Acked  bool    `json:"acked"`
}

// ThreadMessage – сообщение переписки стажера с наставником.
type ThreadMessage struct {
	ID         string        `json:"id"`
	Kind       string        `json:"kind"`   // text | handoff | update
	Author     string        `json:"author"` // intern | mentor | buddy
	AuthorName string        `json:"authorName"`
	Text       string        `json:"text"`
	At         time.Time     `json:"at"`
	Update     *ThreadUpdate `json:"update"`
}

// Thread – переписка целиком.
type Thread struct {
	InternID   string          `json:"internId"`
	InternName string          `json:"internName"`
	Mentor     PersonRef       `json:"mentor"`
	StageCode  string          `json:"stageCode"`
	StageTitle string          `json:"stageTitle"`
	Messages   []ThreadMessage `json:"messages"`
}

// ThreadSummary – строка списка переписок наставника.
type ThreadSummary struct {
	InternID   string    `json:"internId"`
	InternName string    `json:"internName"`
	StageCode  string    `json:"stageCode"`
	StageTitle string    `json:"stageTitle"`
	LastText   string    `json:"lastText"`
	LastAt     time.Time `json:"lastAt"`
	Unread     int       `json:"unread"`
}

// Material – материал наставника: файл, ссылка или ответ, сохраненный в справочник.
type Material struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Kind       string    `json:"kind"` // pdf | docx | md | link | answer
	StageCode  *string   `json:"stageCode"`
	Section    *string   `json:"section"`
	UseInAI    bool      `json:"useInAi"`
	AuthorName string    `json:"authorName"`
	CreatedAt  time.Time `json:"createdAt"`
	SizeLabel  string    `json:"sizeLabel"`
}

// MaterialDraft – новый материал.
type MaterialDraft struct {
	AuthorID, TeamID, Title, Section, URL string
	StageID                               *string
	UseInAI                               bool
	FileName, MimeType, SHA256, Body      string
	Content                               []byte
}

// MentorMeeting – встреча в расписании наставника.
type MentorMeeting struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	InternID    *string   `json:"internId"`
	WithName    string    `json:"withName"`
	Kind        string    `json:"kind"` // one_on_one | review | team
	StartsAt    time.Time `json:"startsAt"`
	DurationMin int       `json:"durationMin"`
}

// Automation – правило AI-buddy, которое наставник включает и выключает.
type Automation struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Note    string `json:"note"`
	Enabled bool   `json:"enabled"`
}

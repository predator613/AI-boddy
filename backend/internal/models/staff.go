package models

import "time"

// ---------- Лента наставника (колокольчик) ----------

// Actor – кто сделал то, о чем запись ленты.
type Actor struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // intern | mentor | lead | buddy | other
}

// FeedItem – запись ленты.
type FeedItem struct {
	ID    string    `json:"id"`
	Actor Actor     `json:"actor"`
	Text  string    `json:"text"`
	At    time.Time `json:"at"`
	Read  bool      `json:"read"`
}

// Feed – последние записи и число непрочитанных.
type Feed struct {
	Items  []FeedItem `json:"items"`
	Unread int        `json:"unread"`
}

// Notice – запись для ленты: кому, о чем и кто сделал.
type Notice struct {
	Kind      string // вид уведомления: tasks | requests | meetings | goals | tips…
	Text      string
	Link      string
	ActorID   *string
	ActorKind string // intern | mentor | lead | buddy | other
	ActorName string
	RequestID *string
}

// ---------- Согласования руководителя ----------

// Approval – заявка стажера, которую решает пользователь.
type Approval struct {
	ID           string    `json:"id"`
	Number       string    `json:"number"`
	InternID     string    `json:"internId"`
	InternName   string    `json:"internName"`
	Title        string    `json:"title"`
	Fields       []Field   `json:"fields"`
	CreatedAt    time.Time `json:"createdAt"`
	Status       string    `json:"status"` // waiting | approved | rejected
	RejectReason *string   `json:"rejectReason"`
}

// ---------- Чек-лист наставника ----------

// ChecklistItem – пункт чек-листа наставника по стажеру.
type ChecklistItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Hint     string `json:"hint"`
	FromWeek int    `json:"fromWeek"`
	Period   string `json:"period"`
	Done     bool   `json:"done"`
}

// MentorChecklist – чек-лист по одному стажеру.
type MentorChecklist struct {
	InternID string          `json:"internId"`
	Items    []ChecklistItem `json:"items"`
}

// ---------- Конспект встреч ----------

// MeetingNote – заметка стажера к событию календаря.
type MeetingNote struct {
	EventID   string    `json:"eventId"`
	Text      string    `json:"text"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ---------- Обращения в поддержку ----------

// SupportTicket – обращение «Сообщить о проблеме».
type SupportTicket struct {
	ID        string    `json:"id"`
	Number    string    `json:"number"` // SUP-1042
	Type      string    `json:"type"`
	Details   string    `json:"details"`
	Status    string    `json:"status"` // new | in_progress | done
	CreatedAt time.Time `json:"createdAt"`
	Answer    *string   `json:"answer"`
}

// SupportDraft – новое обращение.
type SupportDraft struct {
	UserID       string
	Type         string
	Details      string
	InternID     *string
	AttachDialog bool
	ExternalRef  string
}

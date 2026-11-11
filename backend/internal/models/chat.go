package models

import "time"

// ---------- Чат с помощником и заявки (формат схем frontend/src/schemas/assistant.ts) ----------

// Field – строка карточки «Ресурс: GitLab».
type Field struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ChatMessage – сообщение в чате.
type ChatMessage struct {
	ID      string         `json:"id"`
	Role    string         `json:"role"` // user | assistant
	Kind    string         `json:"kind"` // text | confirm | result | error
	Text    string         `json:"text"`
	Source  *MessageSource `json:"source"`
	Confirm *ConfirmCard   `json:"confirm"`
	Result  *ResultCard    `json:"result"`
	Mood    string         `json:"mood"` // calm | happy | think | mentor | sad
	At      time.Time      `json:"at"`
}

// MessageSource – откуда ответ: статья справочника или подпись вроде «Мой путь → этап 02».
type MessageSource struct {
	Title     string  `json:"title"`
	ArticleID *string `json:"articleId"`
}

// ConfirmCard – действие, которое ждет подтверждения. ActionID – id черновика заявки.
type ConfirmCard struct {
	ActionID string  `json:"actionId"`
	Title    string  `json:"title"`
	Fields   []Field `json:"fields"`
	State    string  `json:"state"` // pending | confirmed | cancelled
}

// ResultCard – созданная заявка.
type ResultCard struct {
	RequestID string `json:"requestId"`
	Number    string `json:"number"`
	Title     string `json:"title"`
	Status    string `json:"status"`
}

// ChatReply – ответ на сообщение или подтверждение.
type ChatReply struct {
	Messages []ChatMessage `json:"messages"`
}

// RequestItem – заявка на странице «Заявки».
type RequestItem struct {
	ID           string         `json:"id"`
	Number       string         `json:"number"`
	Kind         string         `json:"kind"` // access | leave | other
	Title        string         `json:"title"`
	Status       string         `json:"status"` // waiting | approved | done | rejected
	CreatedAt    time.Time      `json:"createdAt"`
	Via          string         `json:"via"`
	Fields       []Field        `json:"fields"`
	History      []HistoryEntry `json:"history"`
	RejectReason *string        `json:"rejectReason"`
}

// HistoryEntry – шаг истории заявки.
type HistoryEntry struct {
	Title string `json:"title"`
	At    string `json:"at"`
	State string `json:"state"` // done | now | rejected
}

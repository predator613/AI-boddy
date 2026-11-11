package models

import "time"

// ---------- Справочник (GET /api/kb) ----------

// KnowledgeBase – разделы, статьи и словарик одним запросом. Формат совпадает со схемой фронта.
type KnowledgeBase struct {
	Sections []KBSection `json:"sections"`
	Articles []KBArticle `json:"articles"`
	Terms    []KBTerm    `json:"terms"`
}

// KBSection – стикер раздела. ID – код раздела (s-access), Count – сколько статей видит пользователь.
type KBSection struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Hand  string `json:"hand"`
	Color string `json:"color"`
	Count int    `json:"count"`
}

// KBArticle – строка списка статей. StageCode – номер этапа пути команды пользователя («02») или null.
type KBArticle struct {
	ID        string  `json:"id"`
	SectionID string  `json:"sectionId"`
	Title     string  `json:"title"`
	Summary   string  `json:"summary"`
	Minutes   int     `json:"minutes"`
	StageCode *string `json:"stageCode"`
}

// KBTerm – слово из словарика.
type KBTerm struct {
	Term string `json:"term"`
	Def  string `json:"def"`
}

// KBArticleDetail – статья целиком (GET /api/kb/articles/{id}).
type KBArticleDetail struct {
	KBArticle
	Body       string    `json:"body"`   // абзацы разделены пустой строкой
	Source     string    `json:"source"` // seed, upload, link, mentor_answer
	URL        *string   `json:"url"`
	AuthorName *string   `json:"authorName"`
	UseInAI    bool      `json:"useInAi"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// KBSearch – ответ поиска (GET /api/kb/search?q=).
type KBSearch struct {
	Query   string          `json:"query"`
	Results []KBSearchMatch `json:"results"`
}

// KBSearchMatch – найденная статья и самый подходящий фрагмент.
type KBSearchMatch struct {
	ArticleID string  `json:"articleId"`
	SectionID string  `json:"sectionId"`
	Title     string  `json:"title"`
	Summary   string  `json:"summary"`
	Snippet   string  `json:"snippet"`
	StageCode *string `json:"stageCode"`
	Score     float64 `json:"score"`
	Coverage  float64 `json:"coverage"` // доля слов запроса, найденных в статье (0..1)
}

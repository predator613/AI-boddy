package models

import (
	"encoding/json"
	"time"
)

// Статусы задачи стажера.
const (
	TaskTodo       = "todo"
	TaskInProgress = "in_progress"
	TaskWaiting    = "waiting"
	TaskDone       = "done"
)

// Статусы этапа (считаются из задач).
const (
	StageDone    = "done"
	StageCurrent = "current"
	StageTodo    = "todo"
)

// ---------- Шаблоны (то, что редактирует наставник) ----------

// StageTemplate – этап адаптации команды.
type StageTemplate struct {
	ID          string             `json:"id"`
	TeamID      string             `json:"teamId"`
	Position    int                `json:"position"`
	Title       string             `json:"title"`
	Color       string             `json:"color"`
	Tip         string             `json:"tip"`
	Period      string             `json:"period"` // «неделя 1», «месяцы 1–2»; пусто – не задан
	Suggestions []string           `json:"suggestions"`
	Tasks       []TaskTemplate     `json:"tasks"`
	Resources   []ResourceTemplate `json:"resources"`
}

// TaskTemplate – задача этапа.
type TaskTemplate struct {
	ID       string  `json:"id"`
	StageID  string  `json:"stageId"`
	Position int     `json:"position"`
	Title    string  `json:"title"`
	Note     string  `json:"note"`
	DueDay   *int    `json:"dueDay"`
	Action   *Action `json:"action"`
}

// Action – что AI-buddy может сделать по задаче (кнопка «Сделать с AI-buddy»).
type Action struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

// ResourceTemplate – карточка в боковой панели этапа.
type ResourceTemplate struct {
	ID       string  `json:"id"`
	StageID  string  `json:"stageId"`
	Position int     `json:"position"`
	Kind     string  `json:"kind"`
	Name     string  `json:"name"`
	Note     string  `json:"note"`
	Tag      string  `json:"tag"`
	TagKind  string  `json:"tagKind"`
	TaskID   *string `json:"taskId"`
}

// Milestone – веха на шкале недель.
type Milestone struct {
	Week  int    `json:"week"`
	Title string `json:"title"`
}

// InternProfile – сроки стажировки.
type InternProfile struct {
	StartDate     time.Time
	DurationWeeks int
}

// Meeting – встреча стажера.
type Meeting struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	StartsAt time.Time `json:"startsAt"`
	HostID   string    `json:"hostId"`
	HostName string    `json:"hostName"`
}

// TaskProgress – личный статус задачи.
type TaskProgress struct {
	Status string
	Note   *string
}

// ---------- Ответ страницы «Мой путь» ----------

// PathView – все данные экрана «Мой путь» одним запросом.
type PathView struct {
	Intern      PathIntern  `json:"intern"`
	Week        WeekInfo    `json:"week"`
	Progress    Progress    `json:"progress"`
	Stages      []StageView `json:"stages"`
	NextStep    *NextStep   `json:"nextStep"`
	NextMeeting *Meeting    `json:"nextMeeting"`
	Buddy       BuddyHint   `json:"buddy"`
}

// PathIntern – кто стажер и кто его наставник.
type PathIntern struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	TeamName string     `json:"teamName"`
	Mentor   *PersonRef `json:"mentor"`
}

// PersonRef – короткая ссылка на человека.
type PersonRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// WeekInfo – «Неделя 2 из 24» и вехи.
type WeekInfo struct {
	Current    int         `json:"current"`
	Total      int         `json:"total"`
	Day        int         `json:"day"`
	StartDate  string      `json:"startDate"`
	Milestones []Milestone `json:"milestones"`
}

// Progress – цифры для кольца прогресса.
type Progress struct {
	Done       int `json:"done"`
	InProgress int `json:"inProgress"`
	Todo       int `json:"todo"`
	Total      int `json:"total"`
	Percent    int `json:"percent"`
}

// StageView – этап с задачами и статусами конкретного стажера.
type StageView struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Title       string         `json:"title"`
	Color       string         `json:"color"`
	Tip         string         `json:"tip"`
	Period      *string        `json:"period"`
	Status      string         `json:"status"`
	DoneCount   int            `json:"doneCount"`
	TotalCount  int            `json:"totalCount"`
	Suggestions []string       `json:"suggestions"`
	Tasks       []TaskView     `json:"tasks"`
	Resources   []ResourceView `json:"resources"`
}

// TaskView – задача глазами стажера.
type TaskView struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Note    string  `json:"note"`
	Status  string  `json:"status"`
	IsNext  bool    `json:"isNext"`
	DueDate *string `json:"dueDate"`
	Action  *Action `json:"action"`
	// Details – подробности для раскрытия по клику; null – показываются только note и срок.
	Details *TaskDetails `json:"details"`
}

// TaskDetails – о чем задача, кто поможет, когда она готова и в какую цель стажировки идет.
type TaskDetails struct {
	About    string  `json:"about"`
	Helper   string  `json:"helper"`
	DoneWhen string  `json:"doneWhen"`
	Goal     *string `json:"goal"`
}

// ResourceView – карточка боковой панели с уже посчитанным тегом.
type ResourceView struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Note    string `json:"note"`
	Tag     string `json:"tag"`
	TagKind string `json:"tagKind"`
}

// NextStep – карточка «Следующий шаг».
type NextStep struct {
	TaskID    string  `json:"taskId"`
	StageID   string  `json:"stageId"`
	StageCode string  `json:"stageCode"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	DueDate   *string `json:"dueDate"`
	DueInDays *int    `json:"dueInDays"`
	Action    *Action `json:"action"`
}

// BuddyHint – реплика маскота и быстрые вопросы.
type BuddyHint struct {
	Text        string   `json:"text"`
	Suggestions []string `json:"suggestions"`
}

// InternSummary – строка в списке стажеров наставника.
type InternSummary struct {
	ID        string     `json:"id"`
	Login     string     `json:"login"`
	Name      string     `json:"name"`
	TeamName  string     `json:"teamName"`
	Mentor    *PersonRef `json:"mentor"`
	Week      int        `json:"week"`
	Progress  Progress   `json:"progress"`
	StageCode string     `json:"currentStageCode"`
	Stage     string     `json:"currentStage"`
}

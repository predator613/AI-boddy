package models

import "time"

// ---------- Цели стажировки ----------

// GoalsView – цели стажера и их согласование (GET /api/goals, GET /api/interns/{id}/goals).
type GoalsView struct {
	InternID string          `json:"internId"`
	Agreed   *GoalsAgreement `json:"agreed"`  // null – цели еще не согласованы с руководителем
	FromOKR  bool            `json:"fromOkr"` // хотя бы одна цель привязана к OKR команды
	Goals    []Goal          `json:"goals"`
	OKRs     []TeamOKR       `json:"okrs"` // OKR команды для выпадающего списка
	CanEdit  bool            `json:"canEdit"`
	CanAgree bool            `json:"canAgree"`
}

// GoalsAgreement – кто и когда согласовал цели.
type GoalsAgreement struct {
	By PersonRef `json:"by"`
	At time.Time `json:"at"`
}

// Goal – цель с критерием успеха и прогрессом.
type Goal struct {
	ID          string     `json:"id"`
	Position    int        `json:"position"`
	Title       string     `json:"title"`
	Criterion   string     `json:"criterion"`
	DueWeek     *int       `json:"dueWeek"`
	ProgressPct int        `json:"progressPct"`
	OKR         *TeamOKR   `json:"okr"`
	LastUpdate  *GoalNote  `json:"lastUpdate"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CreatedBy   *PersonRef `json:"createdBy"`
	Competency  *string    `json:"competency"` // компетенция из матрицы роли, «Код и ревью · L1»
	Steps       []GoalStep `json:"steps"`      // если шаги есть, progressPct – доля выполненных
}

// GoalStep – шаг цели: задача «Моего пути» или отметка наставника либо руководителя.
type GoalStep struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Source    string  `json:"source"`    // task | mentor | lead
	StageCode *string `json:"stageCode"` // этап задачи, «02»
	TaskID    *string `json:"taskId"`
	Status    string  `json:"status"` // todo | in_progress | done
}

// GoalNote – последняя отметка о прогрессе.
type GoalNote struct {
	Author      PersonRef `json:"author"`
	ProgressPct int       `json:"progressPct"`
	Note        string    `json:"note"`
	At          time.Time `json:"at"`
}

// TeamOKR – цель команды на период.
type TeamOKR struct {
	ID     string `json:"id"`
	Period string `json:"period"`
	Title  string `json:"title"`
}

// GoalFields – поля цели для записи.
type GoalFields struct {
	Title     string
	Criterion string
	DueWeek   *int
	OKRID     *string
	// Competency – компетенция из матрицы роли; nil – не задана.
	Competency *string
	// Steps – новый список шагов; nil – шаги не менять.
	Steps []GoalStepInput
}

// GoalStepInput – шаг цели в форме.
type GoalStepInput struct {
	Title  string `json:"title"`
	Source string `json:"source"`
}

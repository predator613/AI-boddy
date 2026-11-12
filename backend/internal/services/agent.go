package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"ai-buddy/backend/internal/integrations"
	"ai-buddy/backend/internal/llm"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// Агент AI-buddy. Схема одного сообщения:
//
//	вопрос → модель выбирает инструмент → агент проверяет права и выполняет инструмент →
//	ответ (текст со ссылкой на источник) или карточка подтверждения.
//
// Граница самостоятельности (согласована с экспертом Авито): справочные ответы и статусы помощник
// дает сам; все, что создает заявку, – только после «Подтвердить» стажера (черновик заявки).
// Доступ, запрещенный роли, помощник не оформляет и объясняет почему.

// ChatStore – переписка в базе.
type ChatStore interface {
	BuddyConversation(ctx context.Context, internID string) (string, error)
	AddMessage(ctx context.Context, m repository.NewMessage) (string, time.Time, error)
	Messages(ctx context.Context, conversationID string, limit int) ([]repository.StoredMessage, error)
	Clear(ctx context.Context, conversationID string) error
	Dialogs(ctx context.Context, conversationID string, limit int) ([]repository.DialogRow, error)
	DialogMessages(ctx context.Context, conversationID, dialogID string, limit int) ([]repository.StoredMessage, error)
	Handoff(ctx context.Context, questionID, internID, mentorID, text string) error
	LogToolCall(ctx context.Context, c repository.ToolCall) error
}

// KBSearcher – поиск по справочнику.
type KBSearcher interface {
	Search(ctx context.Context, teamID, viewerID, query string, limit int, onlyAI bool) ([]models.KBSearchMatch, error)
	ArticleRef(ctx context.Context, articleID string) (string, error)
	Article(ctx context.Context, teamID, viewerID, id string) (*models.KBArticleDetail, error)
}

// minCoverage – минимальная доля слов вопроса в статье, чтобы отвечать по ней.
const minCoverage = 0.5

// ErrUnavailable – внешняя система заявок не отвечает.
var ErrUnavailable = errors.New("service unavailable")

// SendInput – сообщение стажера. Action – действие задачи из «Моего пути», TaskID – эта задача.
type SendInput struct {
	Text   string         `json:"text"`
	Action *models.Action `json:"action"`
	TaskID *string        `json:"taskId"`
}

// Agent – помощник стажера.
type Agent struct {
	chat     ChatStore
	reqs     RequestStore
	kb       KBSearcher
	users    UserLookup
	people   PeopleStore
	corp     integrations.Corporate
	model    llm.Provider
	timeout  time.Duration
	log      *slog.Logger
	now      func() time.Time
	fallback llm.Provider
	path     PathReader
}

// PathReader – «Мой путь» стажера: следующая задача для подсказок «что дальше».
type PathReader interface {
	GetPath(ctx context.Context, viewerID, internID string) (*models.PathView, error)
}

// WithPath подключает «Мой путь»: агент знает следующую задачу стажера.
func (a *Agent) WithPath(p PathReader) *Agent {
	a.path = p
	return a
}

// NewAgent создает агента. timeout – сколько ждать модель, потом ответ собирается офлайн.
func NewAgent(chat ChatStore, reqs RequestStore, kb KBSearcher, users UserLookup, people PeopleStore,
	corp integrations.Corporate, model llm.Provider, timeout time.Duration, log *slog.Logger) *Agent {
	return &Agent{chat: chat, reqs: reqs, kb: kb, users: users, people: people, corp: corp, model: model,
		timeout: timeout, log: log, now: time.Now, fallback: llm.Mock{}}
}

// turn – контекст одного ответа.
type turn struct {
	viewer     *models.User
	conv       string
	questionID string
	question   string
	out        []models.ChatMessage
	// Контекст переписки: последние сообщения до текущего вопроса и неподтвержденная карточка.
	history []llm.Message
	draft   *models.Request
	circle  *models.Circle   // наставник, руководитель, команда – для подстановки в статьи и промпт
	next    *models.NextStep // следующая задача «Моего пути»
	// replaced – старая карточка заменена новой (стажер поменял поля), dropped – убрана карточка другой заявки.
	replaced bool
	dropped  string
}

// ---------- публичные методы ----------

// Send – сообщение стажера помощнику.
func (a *Agent) Send(ctx context.Context, viewerID string, in SendInput) (*models.ChatReply, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" || len([]rune(text)) > 2000 {
		return nil, invalid("text is required (max 2000)")
	}
	t, err := a.begin(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	qid, _, err := a.chat.AddMessage(ctx, repository.NewMessage{
		ConversationID: t.conv, AuthorKind: "user", AuthorID: &t.viewer.ID, Kind: "text", Text: text,
	})
	if err != nil {
		return nil, err
	}
	t.questionID, t.question = qid, text
	if err := a.loadContext(ctx, t); err != nil {
		return nil, err
	}

	if in.Action != nil {
		err = a.runAction(ctx, t, *in.Action, in.TaskID)
	} else {
		err = a.runModel(ctx, t, text)
	}
	if err != nil {
		return nil, err
	}
	return &models.ChatReply{Messages: t.out}, nil
}

// Confirm – стажер подтвердил действие: заявка уходит во внешнюю систему.
func (a *Agent) Confirm(ctx context.Context, viewerID, actionID string) (*models.ChatReply, error) {
	t, q, err := a.draft(ctx, viewerID, actionID)
	if err != nil {
		return nil, err
	}
	order := integrations.Order{
		Kind: q.Type, RequestNumber: q.Number, Requester: t.viewer.Login,
		Service: q.ServiceCode, Project: deref(q.ProjectTitle), AccessLevel: q.Level, Justification: q.Justification,
		Auto: q.Type == "access" && q.ApproverKind == "auto",
	}
	if q.Approver != nil {
		order.Approver = q.Approver.Name
	}
	if q.LeaveDate != nil {
		order.LeaveDate = q.LeaveDate.Format(time.DateOnly)
		order.LeaveType = q.LeaveTypeTitle
	}
	if q.Type == "other" {
		order.Department, order.Item, order.Comment, order.Approver = q.DepartmentTitle, q.ItemCode, q.ItemComment, q.DepartmentTitle
	}
	started := a.now()
	ticket, err := a.corp.Submit(ctx, order)
	a.logCall(ctx, t.conv, "mock-services", q.Type+".submit", order, ticket, err, started)
	if err != nil {
		a.log.Warn("submit request failed", "err", err)
		return nil, ErrUnavailable
	}
	status, ok := ticketStatus(q.Type, ticket.Status)
	if !ok {
		return nil, ErrUnavailable
	}
	note := fmt.Sprintf("Заявка №%d на согласовании: %s", q.Number, approverText(q))
	if status != "waiting" {
		note = fmt.Sprintf("Заявка №%d выполнена автоматически", q.Number)
	}
	if err := a.reqs.Submit(ctx, q.ID, t.viewer.ID, status, ticket.Ref, note); err != nil {
		return nil, mapNotFound(err)
	}
	q.Status = status
	mentorNote := a.notifyMentor(ctx, t, q)

	var text, mood string
	switch status {
	case "waiting":
		text = fmt.Sprintf("Заявка №%d создана и отправлена: %s. Обычно решение до %d ч, я сообщу сюда же.",
			q.Number, approverText(q), q.SLAHours)
		if q.Type == "other" {
			text = fmt.Sprintf("Заявка №%d отправлена: %s, обычно до %d ч. Ответ придет сюда же.", q.Number, q.DepartmentTitle, q.SLAHours)
		}
		text += mentorNote
		mood = "happy"
	default:
		text = fmt.Sprintf("Готово: %s выдан сразу, согласование не нужно.", lowerFirst(RequestTitle(q)))
		if q.TaskID != nil {
			text += " Пункт на «Моем пути» закрыт."
		}
		mood = "happy"
	}
	if err := a.say(ctx, t, "result", a.phrase(ctx, t, text), mood, &sayOpts{request: q}); err != nil {
		return nil, err
	}
	return &models.ChatReply{Messages: t.out}, nil
}

// Cancel – стажер отказался от действия.
func (a *Agent) Cancel(ctx context.Context, viewerID, actionID string) (*models.ChatReply, error) {
	t, q, err := a.draft(ctx, viewerID, actionID)
	if err != nil {
		return nil, err
	}
	if err := a.reqs.Cancel(ctx, q.ID, t.viewer.ID); err != nil {
		return nil, mapNotFound(err)
	}
	if err := a.say(ctx, t, "text", a.phrase(ctx, t, "Хорошо, заявку не создаю. Если передумаешь, просто напиши."), "calm", nil); err != nil {
		return nil, err
	}
	return &models.ChatReply{Messages: t.out}, nil
}

// Reset – «Новый диалог»: переписка остается в базе, но чат и модель начинают с чистого листа.
func (a *Agent) Reset(ctx context.Context, viewerID string) (*models.ChatReply, error) {
	t, err := a.begin(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if err := a.chat.Clear(ctx, t.conv); err != nil {
		return nil, err
	}
	return &models.ChatReply{Messages: []models.ChatMessage{}}, nil
}

// History – последние сообщения переписки (для восстановления чата после перезагрузки).
func (a *Agent) History(ctx context.Context, viewerID string) (*models.ChatReply, error) {
	t, err := a.begin(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	stored, err := a.chat.Messages(ctx, t.conv, 50)
	if err != nil {
		return nil, err
	}
	return &models.ChatReply{Messages: a.toMessages(ctx, stored)}, nil
}

// Dialogs – колонка истории: открытый диалог и прошлые, новые сверху (до 50).
func (a *Agent) Dialogs(ctx context.Context, viewerID string) ([]models.ChatDialog, error) {
	t, err := a.begin(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	rows, err := a.chat.Dialogs(ctx, t.conv, 50)
	if err != nil {
		return nil, err
	}
	out := make([]models.ChatDialog, len(rows))
	for i, d := range rows {
		out[i] = models.ChatDialog{ID: d.ID, Title: d.Title, Preview: d.Preview, Kind: d.Kind,
			UpdatedAt: d.UpdatedAt.In(moscow), Current: d.Current}
	}
	return out, nil
}

// DialogMessages – сообщения прошлого (или текущего) диалога для чтения. Чужой диалог – 404.
// Карточки приходят с итоговым состоянием: черновики закрытого диалога отменены.
func (a *Agent) DialogMessages(ctx context.Context, viewerID, dialogID string) (*models.ChatReply, error) {
	t, err := a.begin(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	stored, err := a.chat.DialogMessages(ctx, t.conv, dialogID, 200)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &models.ChatReply{Messages: a.toMessages(ctx, stored)}, nil
}

// toMessages – сохраненные сообщения в формате чата, карточки – с актуальным состоянием заявки.
func (a *Agent) toMessages(ctx context.Context, stored []repository.StoredMessage) []models.ChatMessage {
	out := make([]models.ChatMessage, 0, len(stored))
	for _, m := range stored {
		msg := models.ChatMessage{ID: m.ID, Role: "assistant", Kind: m.Kind, Text: m.Text, Mood: "calm", At: m.CreatedAt}
		if m.AuthorKind == "user" {
			msg.Role = "user"
		}
		if m.Mood != nil {
			msg.Mood = *m.Mood
		}
		switch {
		case m.ArticleID != nil && m.SourceTitle != nil:
			msg.Source = &models.MessageSource{Title: *m.SourceTitle, ArticleID: m.ArticleID}
		case m.SourceLabel != nil:
			msg.Source = &models.MessageSource{Title: *m.SourceLabel}
		}
		if m.RequestID != nil {
			if q, err := a.reqs.Get(ctx, *m.RequestID); err == nil {
				attachCard(&msg, q)
			}
		}
		out = append(out, msg)
	}
	return out
}

// ---------- инструменты ----------

var agentTools = []llm.Tool{
	{Name: "search_kb", Description: "Найти ответ в справочнике стажера (доступы, процессы, код-ревью, отгулы, люди).",
		Parameters: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"вопрос своими словами"}},"required":["query"]}`)},
	{Name: "prepare_access_request", Description: "Подготовить заявку на доступ к сервису. Заявка создается только после подтверждения стажера.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"resource":{"type":"string","description":"сервис: GitLab, тестовый стенд, вики, трекер задач, трекер прогресса, логи"},"access_level":{"type":"string","enum":["read","edit","admin"]},"project":{"type":"string"},"justification":{"type":"string"}},"required":["resource"]}`)},
	{Name: "prepare_leave_request", Description: "Подготовить заявку на отгул. Создается только после подтверждения.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"date":{"type":"string","description":"дата словами или ДД.ММ"},"type":{"type":"string","enum":["day_off","day_off_unpaid","study"]}},"required":["date"]}`)},
	{Name: "prepare_service_request", Description: "Подготовить заявку в отдел компании: оборудование (ноутбук, монитор, гарнитура, клавиатура и мышь, пропуск), ИТ-поддержка (VPN, почта, сброс пароля, двухфакторная аутентификация), HR (отпуск, справка с места работы, документы для вуза, изменение личных данных), бухгалтерия (справка 2-НДФЛ, вопрос по выплате). Создается только после подтверждения.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"item":{"type":"string","description":"что нужно: ноутбук, VPN, справка 2-НДФЛ и т. п."},"comment":{"type":"string","description":"подробности со слов стажера, если есть"}},"required":["item"]}`)},
	{Name: "list_requests", Description: "Показать статус заявок стажера.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"number":{"type":"string"}}}`)},
	{Name: "handoff_to_mentor", Description: "Передать вопрос наставнику, если в справочнике нет ответа или стажер просит человека.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"question":{"type":"string"}},"required":["question"]}`)},
}

func (a *Agent) systemPrompt(t *turn) string {
	return fmt.Sprintf(`Ты AI-buddy – дружелюбный помощник стажера в IT-компании (демо-прототип: данные синтетические, это не регламенты Авито).
Сегодня %s. Стажер: %s, команда %s. %s

Как отвечать:
– по-русски, на «ты», живо и по делу, как опытный коллега, а не как справочное окно;
– сначала прямой ответ на вопрос, затем, если нужно, шаги списком «1.», «2.» или пункты «–»; обычно 2–6 предложений;
– факты о компании (сроки, кто согласует, правила) бери только из результатов search_kb и данных ниже, ничего не выдумывай;
– если в справочнике ничего не нашлось, можно дать общий совет, но прямо скажи, что в справочнике этого нет, и предложи передать вопрос наставнику;
– называй людей по именам из данных ниже, не придумывай других людей;
– в конце предложи один конкретный следующий шаг, если он есть;
– без markdown: не используй звездочки, решетки и таблицы.

Инструменты: search_kb – вопросы о работе и процессах; prepare_access_request – доступ к сервисам (GitLab, стенд, вики, трекер); prepare_service_request – заявка в отдел (техника, VPN, пароли, справки, отпуск, выплаты); prepare_leave_request – отгул; list_requests – статус заявок; handoff_to_mentor – передать вопрос наставнику.
Заявка создается только после того, как стажер нажмет «Подтвердить» в карточке.`,
		dayMonth(a.now().In(moscow))+", "+weekday(a.now().In(moscow)), t.viewer.Name, t.viewer.TeamName, a.peopleNote(t)) +
		nextNote(t.next) + draftNote(t.draft)
}

// peopleNote – кто вокруг стажера: имена только из связей в базе.
func (a *Agent) peopleNote(t *turn) string {
	mentor, lead := "пока не назначен", "не указан"
	if c := t.circle; c != nil {
		if c.Mentor != nil {
			mentor = c.Mentor.Name
		}
		if c.Lead != nil {
			lead = c.Lead.Name
		}
	}
	return fmt.Sprintf("Наставник стажера: %s. Руководитель команды: %s (согласует доступы к GitLab, отгулы, цели). "+
		"Отделы: оборудование, ИТ-поддержка, HR-поддержка, бухгалтерия – заявки в них оформляешь ты.", mentor, lead)
}

// nextNote – следующая задача «Моего пути»: на вопрос «что дальше» модель называет ее.
func nextNote(n *models.NextStep) string {
	if n == nil {
		return ""
	}
	return "\nСледующая задача стажера на «Моем пути»: " + nextTaskText(n) + ". На вопрос «что дальше» или «какой следующий шаг» назови ее."
}

// nextTaskText – «Получить доступ к GitLab» (этап 02, срок сегодня).
func nextTaskText(n *models.NextStep) string {
	due := ""
	if n.DueInDays != nil {
		switch d := *n.DueInDays; {
		case d < 0:
			due = ", срок прошел"
		case d == 0:
			due = ", срок сегодня"
		case d == 1:
			due = ", срок завтра"
		default:
			due = fmt.Sprintf(", срок через %d дн.", d)
		}
	}
	return fmt.Sprintf("«%s» (этап %s%s)", n.Title, n.StageCode, due)
}

// draftNote – что модель должна знать о неподтвержденной карточке в чате.
func draftNote(q *models.Request) string {
	if q == nil {
		return ""
	}
	var parts []string
	for _, f := range RequestFields(q) {
		parts = append(parts, f.Label+": "+f.Value)
	}
	return fmt.Sprintf(`
Сейчас в чате есть неподтвержденная карточка «%s» (%s). Заявка уйдет, только когда стажер нажмет
«Подтвердить» в карточке; «Отмена» – передумал. Если стажер спрашивает, что делать дальше, объясни это.
Если стажер просит поменять поля карточки (уровень прав, проект, дату, комментарий), вызови тот же инструмент
с новыми значениями – старая карточка заменится новой. Без изменений вторую такую же заявку не готовь.`, RequestTitle(q), strings.Join(parts, "; "))
}

// historySize – сколько последних сообщений переписки видит модель.
const historySize = 8

// loadContext – история переписки до текущего вопроса и последняя карточка, которая ждет «Подтвердить».
func (a *Agent) loadContext(ctx context.Context, t *turn) error {
	stored, err := a.chat.Messages(ctx, t.conv, historySize+1)
	if err != nil {
		return err
	}
	for _, m := range stored {
		if m.ID == t.questionID {
			continue
		}
		if m.Kind == "confirm" && m.RequestID != nil {
			if q, err := a.reqs.Get(ctx, *m.RequestID); err == nil && q.Status == "draft" && q.RequesterID == t.viewer.ID {
				t.draft = q
			} else {
				t.draft = nil
			}
		}
		if m.AuthorKind == "user" {
			t.history = append(t.history, llm.Message{Role: "user", Content: m.Text})
			continue
		}
		text := m.Text
		if m.Kind == "confirm" {
			text = "[карточка подтверждения] " + text
		}
		t.history = append(t.history, llm.Message{Role: "assistant", Content: text})
	}
	if a.path != nil {
		if v, err := a.path.GetPath(ctx, t.viewer.ID, ""); err == nil {
			t.next = v.NextStep
		}
	}
	return nil
}

// runModel – вопрос текстом: модель выбирает инструмент.
func (a *Agent) runModel(ctx context.Context, t *turn, text string) error {
	low := strings.ToLower(text)
	if reWho.MatchString(low) {
		return a.whoAnswer(ctx, t)
	}
	if t.draft != nil && reEdit.MatchString(low) {
		if done, err := a.editDraft(ctx, t, text); done {
			return err
		}
	}
	msgs := append([]llm.Message{{Role: "system", Content: a.systemPrompt(t)}}, t.history...)
	msgs = append(msgs, llm.Message{Role: "user", Content: text})
	resp := a.ask(ctx, t, llm.Request{Messages: msgs, Tools: agentTools})
	if len(resp.ToolCalls) == 0 {
		if resp.Content == "" {
			return a.noAnswer(ctx, t)
		}
		return a.say(ctx, t, "text", cleanText(resp.Content), "calm", nil)
	}
	call := resp.ToolCalls[0]
	var args map[string]string
	_ = json.Unmarshal(call.Arguments, &args)
	switch call.Name {
	case "search_kb":
		return a.searchAnswer(ctx, t, msgs, call, first(args["query"], text))
	case "prepare_access_request":
		code, err := a.reqs.ResolveService(ctx, first(args["resource"], text))
		if errors.Is(err, repository.ErrNotFound) {
			return a.say(ctx, t, "text", a.phrase(ctx, t, "Не понял, к какому сервису нужен доступ. Например: GitLab, тестовый стенд, вики, трекер задач."), "think", nil)
		}
		if err != nil {
			return err
		}
		return a.prepareAccess(ctx, t, code, "", args["access_level"], args["project"], args["justification"], nil)
	case "prepare_leave_request":
		return a.prepareLeave(ctx, t, first(args["date"], text), args["type"], nil)
	case "prepare_service_request":
		return a.prepareService(ctx, t, first(args["item"], text), args["comment"])
	case "list_requests":
		return a.listRequests(ctx, t, args["number"])
	case "handoff_to_mentor":
		return a.handoff(ctx, t)
	}
	return a.noAnswer(ctx, t)
}

// runAction – кнопка «Сделать с AI-buddy» у задачи пути: действие известно заранее.
func (a *Agent) runAction(ctx context.Context, t *turn, act models.Action, taskID *string) error {
	var p map[string]string
	_ = json.Unmarshal(act.Payload, &p)
	switch act.Kind {
	case "access_request":
		return a.prepareAccess(ctx, t, p["service"], p["role"], "", first(p["project"], legacyProject(p["group"])), "", taskID)
	case "leave_request":
		return a.prepareLeave(ctx, t, t.question, p["type"], taskID)
	case "ask_buddy":
		return a.runModel(ctx, t, first(p["question"], t.question))
	case "open_article":
		id := p["articleId"]
		return a.say(ctx, t, "text", "Открой статью из справочника, там все по шагам.", "calm", &sayOpts{article: &id})
	}
	return a.runModel(ctx, t, t.question)
}

func legacyProject(group string) string {
	return map[string]string{"backend": "backend/auth-service", "frontend": "frontend/web-app"}[group]
}

// searchAnswer – ответ из справочника со ссылкой на статью.
func (a *Agent) searchAnswer(ctx context.Context, t *turn, msgs []llm.Message, call llm.ToolCall, query string) error {
	started := a.now()
	all, err := a.kb.Search(ctx, t.viewer.TeamID, t.viewer.ID, query, 5, true)
	a.logCall(ctx, t.conv, "kb", "search_kb", map[string]string{"query": query}, len(all), err, started)
	if err != nil {
		return err
	}
	// Отвечаем только по статьям, где нашлась хотя бы половина слов вопроса: иначе
	// «сколько стоит компьютер» совпадет со «Сколько ждать согласования» по одному слову.
	var found []models.KBSearchMatch
	for _, f := range all {
		if f.Coverage >= minCoverage && len(found) < 3 {
			found = append(found, f)
		}
	}
	// Модели отдаем не только фрагмент, но и текст двух лучших статей: из одного абзаца
	// содержательного ответа не собрать.
	type hit struct {
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
		Text    string `json:"text,omitempty"`
	}
	hits := make([]hit, len(found))
	for i, f := range found {
		hits[i] = hit{Title: a.personal(t, f.Title), Snippet: a.personal(t, f.Snippet)}
		if i < 2 {
			if art, err := a.kb.Article(ctx, t.viewer.TeamID, t.viewer.ID, f.ArticleID); err == nil {
				hits[i].Text = clip(a.personal(t, art.Body), 2500)
			}
		}
	}
	if len(found) == 0 {
		if a.isMock() {
			return a.noAnswer(ctx, t)
		}
		hits = nil
	}
	res := map[string]any{"results": hits}
	if len(found) == 0 {
		res["note"] = "В справочнике ничего не нашлось."
	}
	raw, _ := json.Marshal(res)
	msgs = append(msgs,
		llm.Message{Role: "assistant", ToolCalls: []llm.ToolCall{call}},
		llm.Message{Role: "tool", ToolCallID: call.ID, Content: string(raw)},
	)
	resp := a.ask(ctx, t, llm.Request{Messages: msgs})
	text := cleanText(resp.Content)
	if len(found) == 0 {
		if text == "" {
			return a.noAnswer(ctx, t)
		}
		return a.say(ctx, t, "text", text, "think", nil)
	}
	if text == "" {
		text = hits[0].Snippet
	}
	id := found[0].ArticleID
	return a.say(ctx, t, "text", text, "calm", &sayOpts{article: &id})
}

// prepareAccess – проверить права и подготовить черновик заявки на доступ.
func (a *Agent) prepareAccess(ctx context.Context, t *turn, service, role, level, project, justification string, taskID *string) error {
	svc, err := a.reqs.Service(ctx, service)
	if errors.Is(err, repository.ErrNotFound) {
		return a.say(ctx, t, "error", a.phrase(ctx, t, "Такого сервиса нет в списке доступов. Проверь название или спроси наставника."), "sad", nil)
	}
	if err != nil {
		return err
	}
	roles, err := a.reqs.ServiceRoles(ctx, svc.Code, t.viewer.Role)
	if err != nil {
		return err
	}
	chosen, alt := pickRole(roles, role, level)
	if chosen == nil || !chosen.Allowed {
		a.logCall(ctx, t.conv, "policy", "access.check", map[string]string{"service": svc.Code, "role": role, "level": level}, "denied", nil, a.now())
		reason := "Этот доступ твоей роли не выдается."
		if chosen != nil && chosen.Reason != "" {
			reason = chosen.Reason
		}
		if alt != nil {
			reason += fmt.Sprintf(" Могу запросить уровень «%s» – напиши «запроси %s».", levelLabel(*alt), strings.ToLower(svc.Title))
		} else {
			reason += " Если он все же нужен для задачи, передам вопрос наставнику."
		}
		return a.say(ctx, t, "error", a.phrase(ctx, t, reason), "sad", nil)
	}

	// Карточка на этот сервис уже ждет подтверждения: без изменений вторую не создаем,
	// с другим уровнем или проектом – заменяем.
	if q := t.draft; q != nil && q.Type == "access" && q.ServiceCode == svc.Code &&
		q.RoleTitle == chosen.Title && (project == "" || q.ProjectTitle == nil || strings.EqualFold(*q.ProjectTitle, project)) {
		return a.say(ctx, t, "text", a.phrase(ctx, t, "Карточка на этот доступ уже выше: проверь поля и нажми «Подтвердить»."), "mentor", nil)
	}
	// Уже есть доступ или заявка на согласовании – не создаем дубль.
	if open, err := a.reqs.OpenRequest(ctx, t.viewer.ID, svc.Code); err == nil {
		text := fmt.Sprintf("%s уже выдан (заявка №%d).", svc.RequestTitle, open.Number)
		if open.Status == "waiting" {
			text = fmt.Sprintf("Заявка №%d уже на согласовании: %s. Обычно решение до %d ч, я напомню, если затянется.",
				open.Number, approverText(open), open.SLAHours)
		}
		return a.say(ctx, t, "text", a.phrase(ctx, t, text), "calm", nil)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	var proj *models.Project
	if project != "" {
		proj, err = a.reqs.Project(ctx, svc.Code, project)
	} else {
		proj, err = a.reqs.TeamProject(ctx, svc.Code, t.viewer.TeamID)
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	approver, err := a.approver(ctx, t.viewer, svc, proj)
	if err != nil {
		return err
	}
	task, err := a.reqs.LinkedAccessTask(ctx, t.viewer.ID, svc.Code, taskID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	if strings.TrimSpace(justification) == "" {
		justification = "Для работы над задачами стажировки"
		if task != nil {
			justification = "Нужен для задачи «" + task.Title + "»"
		}
	}
	if err := a.dropDraft(ctx, t, t.draft != nil && t.draft.Type == "access" && t.draft.ServiceCode == svc.Code); err != nil {
		return err
	}
	d := models.AccessDraft{RequesterID: t.viewer.ID, Service: svc.Code, Role: chosen.Code, Justification: justification}
	if proj != nil {
		d.Project = &proj.Code
	}
	if approver != nil {
		d.ApproverID = &approver.ID
	}
	if task != nil {
		d.TaskID = &task.ID
	}
	q, err := a.reqs.CreateAccessDraft(ctx, d)
	if err != nil {
		return mapNotFound(err)
	}

	intro := fmt.Sprintf("%s согласует %s, обычно до %d ч.", svc.Title, approverText(q), svc.SLAHours)
	if svc.ApproverKind == "auto" {
		intro = fmt.Sprintf("%s выдается автоматически, сразу после подтверждения.", svc.Title)
	}
	if t.replaced {
		intro = fmt.Sprintf("Обновил карточку: уровень «%s». ", levelLabel(*chosen)) + intro
	}
	intro = droppedNote(t) + intro
	source := a.articleFor(ctx, t, "как запросить доступ к сервису")
	if err := a.say(ctx, t, "text", a.phrase(ctx, t, intro), "calm", &sayOpts{article: source}); err != nil {
		return err
	}
	return a.say(ctx, t, "confirm", "Создать заявку на доступ?", "mentor", &sayOpts{request: q})
}

// pickRole – роль по коду или уровню; по умолчанию самая сильная разрешенная до «редактирования».
// alt – разрешенная альтернатива, если выбранная роль запрещена.
func pickRole(roles []models.ServiceRole, code, level string) (chosen, alt *models.ServiceRole) {
	for i := range roles {
		r := &roles[i]
		if r.Allowed && r.Level != "admin" {
			alt = r
		}
		if (code != "" && r.Code == code) || (code == "" && level != "" && r.Level == level && chosen == nil) {
			chosen = r
		}
	}
	if chosen == nil && code == "" && level == "" {
		chosen = alt
		if chosen == nil && len(roles) > 0 {
			chosen = &roles[0]
		}
	}
	if chosen != nil && chosen.Allowed {
		alt = nil
	}
	return chosen, alt
}

func levelLabel(r models.ServiceRole) string {
	if strings.EqualFold(r.Title, r.LevelTitle) {
		return r.LevelTitle
	}
	return r.LevelTitle + " (" + r.Title + ")"
}

// approver – кто согласует: руководитель, владелец проекта или DevOps; нет – руководитель команды.
func (a *Agent) approver(ctx context.Context, u *models.User, svc *models.Service, proj *models.Project) (*models.PersonRef, error) {
	var id *string
	switch svc.ApproverKind {
	case "auto":
		return nil, nil
	case "project_owner":
		if proj != nil {
			id = proj.OwnerID
		}
	case "devops":
		id = svc.ApproverID
	}
	if id != nil {
		p, err := a.reqs.Person(ctx, *id)
		if err == nil || !errors.Is(err, repository.ErrNotFound) {
			return p, err
		}
	}
	p, err := a.reqs.TeamLead(ctx, u.TeamID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	return p, err
}

// prepareLeave – черновик заявки на отгул. Без даты в тексте берется ближайший срок по правилу
// «за два рабочих дня».
func (a *Agent) prepareLeave(ctx context.Context, t *turn, dateText, kind string, taskID *string) error {
	today := a.now().In(moscow)
	day, ok := ParseLeaveDate(dateText, today)
	if !ok {
		day = AddWorkdays(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, moscow), 2)
	}
	if IsWeekend(day) {
		return a.say(ctx, t, "text", fmt.Sprintf("%s – выходной, отгул не нужен. Назови рабочий день.", capitalize(dayMonth(day))), "think", nil)
	}
	switch {
	case kind == "" && strings.Contains(strings.ToLower(dateText), "свой счет"):
		kind = "day_off_unpaid"
	case kind == "" && strings.Contains(strings.ToLower(dateText), "учеб"):
		kind = "study"
	case kind == "":
		kind = "day_off"
	}
	lead, err := a.reqs.TeamLead(ctx, t.viewer.TeamID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	d := models.LeaveDraft{RequesterID: t.viewer.ID, Date: day, Type: kind, TaskID: taskID}
	if lead != nil {
		d.ApproverID = &lead.ID
	}
	if err := a.dropDraft(ctx, t, t.draft != nil && t.draft.Type == "leave"); err != nil {
		return err
	}
	q, err := a.reqs.CreateLeaveDraft(ctx, d)
	if err != nil {
		return mapNotFound(err)
	}
	who := "руководителю"
	if lead != nil {
		who = "руководителю (" + lead.Name + ")"
	}
	intro := "Отгул оформляется заявкой " + who + " минимум за два рабочих дня."
	if WorkdaysBetween(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, moscow), day) < 2 {
		intro = "До этой даты меньше двух рабочих дней: заявку создать можно, но лучше предупредить " + who + " лично."
	}
	if t.circle != nil && t.circle.Mentor != nil {
		intro += " Наставника (" + t.circle.Mentor.Name + ") я предупрежу, когда подтвердишь."
	}
	if t.replaced {
		intro = "Обновил карточку: новая дата " + dayMonth(day) + ". " + intro
	}
	intro = droppedNote(t) + intro
	source := a.articleFor(ctx, t, "как взять отгул")
	if err := a.say(ctx, t, "text", a.phrase(ctx, t, intro), "calm", &sayOpts{article: source}); err != nil {
		return err
	}
	return a.say(ctx, t, "confirm", "Создать заявку на отгул?", "mentor", &sayOpts{request: q})
}

// listRequests – статус заявок текстом.
func (a *Agent) listRequests(ctx context.Context, t *turn, number string) error {
	list, err := a.reqs.ListByRequester(ctx, t.viewer.ID)
	if err != nil {
		return err
	}
	status := map[string]string{"waiting": "на согласовании", "approved": "одобрена", "done": "выполнена", "rejected": "отклонена"}
	var lines []string
	for _, q := range list {
		if number != "" && fmt.Sprint(q.Number) != number {
			continue
		}
		line := fmt.Sprintf("№%d %s – %s", q.Number, RequestTitle(q), status[q.Status])
		if q.Status == "waiting" {
			line += " (" + approverText(q) + ")"
		}
		if q.Status == "rejected" && q.RejectReason != nil {
			line += ": " + *q.RejectReason
		}
		lines = append(lines, line)
		if len(lines) == 4 {
			break
		}
	}
	if len(lines) == 0 {
		return a.say(ctx, t, "text", a.phrase(ctx, t, "Заявок пока нет. Могу оформить доступ, отгул или заявку в отдел."), "calm", nil)
	}
	label := "Заявки"
	return a.say(ctx, t, "text", a.phrase(ctx, t, "Вот твои заявки:\n"+strings.Join(lines, "\n")), "calm", &sayOpts{label: &label})
}

// handoff – передать вопрос наставнику (нет наставника – руководителю).
func (a *Agent) handoff(ctx context.Context, t *turn) error {
	to := t.viewer.MentorID
	name := "наставник"
	if to != nil {
		if p, err := a.reqs.Person(ctx, *to); err == nil {
			name = p.Name
		}
	} else if lead, err := a.reqs.TeamLead(ctx, t.viewer.TeamID); err == nil {
		to, name = &lead.ID, lead.Name
	}
	if to == nil {
		return a.say(ctx, t, "error", "Не нашел, кому передать вопрос: у тебя пока нет наставника и руководителя в системе.", "sad", nil)
	}
	if err := a.chat.Handoff(ctx, t.questionID, t.viewer.ID, *to, t.question); err != nil {
		return err
	}
	a.logCall(ctx, t.conv, "buddy", "handoff", map[string]string{"to": *to}, "ok", nil, a.now())
	return a.say(ctx, t, "text", a.phrase(ctx, t, fmt.Sprintf("Передал вопрос: %s ответит здесь же, обычно в течение дня.", name)), "mentor", nil)
}

// reNextStep – вопрос «что делать дальше?» без подробностей: ответ зависит от состояния чата.
var reNextStep = regexp.MustCompile(`(?i)что (мне )?(теперь |сейчас |дальше )?(делать|дальше|теперь|нажать|сейчас)|а дальше|и дальше|что потом|следующий шаг|куда нажать|как подтвердить`)

func (a *Agent) noAnswer(ctx context.Context, t *turn) error {
	if reNextStep.MatchString(t.question) {
		return a.nextStep(ctx, t)
	}
	text := "Точного ответа в справочнике нет. Напиши «передай наставнику», и я передам вопрос."
	if t.circle != nil && t.circle.Mentor != nil {
		text = fmt.Sprintf("Точного ответа в справочнике нет. Могу передать вопрос наставнику (%s) – напиши «передай наставнику».", t.circle.Mentor.Name)
	}
	return a.say(ctx, t, "text", a.phrase(ctx, t, text), "mentor", nil)
}

// nextStep – подсказка по состоянию: карточка ждет подтверждения, заявка на согласовании или «Мой путь».
func (a *Agent) nextStep(ctx context.Context, t *turn) error {
	if q := t.draft; q != nil {
		text := fmt.Sprintf("Нажми «Подтвердить» в карточке выше: заявка «%s» уйдет, согласует %s. Если что-то не так, нажми «Отмена» и напиши, что поменять.",
			RequestTitle(q), approverText(q))
		if q.ApproverKind == "auto" || q.Approver == nil {
			text = fmt.Sprintf("Нажми «Подтвердить» в карточке выше: %s выдается автоматически, сразу после подтверждения.", lowerFirst(RequestTitle(q)))
		}
		return a.say(ctx, t, "text", a.phrase(ctx, t, text), "mentor", nil)
	}
	list, err := a.reqs.ListByRequester(ctx, t.viewer.ID)
	if err != nil {
		return err
	}
	var parts []string
	if n := t.next; n != nil {
		s := "Следующий шаг на «Моем пути» – " + nextTaskText(n) + "."
		if n.Action != nil && (n.Action.Kind == "access_request" || n.Action.Kind == "leave_request") {
			s += " Могу оформить заявку прямо здесь – нажми «Сделать с AI-buddy» в карточке на главной или напиши, что нужно."
		}
		parts = append(parts, s)
	}
	for _, q := range list {
		if q.Status == "waiting" {
			parts = append(parts, fmt.Sprintf("Заявка №%d «%s» ждет решения: %s, обычно до %d ч, я напишу, когда придет ответ.",
				q.Number, RequestTitle(q), approverText(q), q.SLAHours))
			break
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "Все задачи на «Моем пути» закрыты, заявок на согласовании нет. Спроси наставника, что взять дальше, или загляни в справочник.")
	}
	return a.say(ctx, t, "text", a.phrase(ctx, t, strings.Join(parts, " ")), "calm", nil)
}

// reWho – вопрос «кто мой наставник / руководитель»: ответ из связей в базе, а не из справочника.
var reWho = regexp.MustCompile(`кто\s+(у меня\s+|мой\s+|моя\s+|мне\s+|наш\s+)?(наставник|ментор|руководител|тимлид|лид\b)|как зовут\s+(моего\s+|нашего\s+)?(наставник|ментор|руководител|тимлид)|кто меня (ведет|курирует)|к кому (мне )?обращаться`)

// whoAnswer – кто вокруг стажера и как с ними связаться.
func (a *Agent) whoAnswer(ctx context.Context, t *turn) error {
	c := t.circle
	var parts []string
	if c != nil && c.Mentor != nil {
		parts = append(parts, fmt.Sprintf("Твой наставник – %s. Написать ему можно в разделе «Чат с наставником», а если вопрос возник здесь, напиши «передай наставнику», и я перешлю.", c.Mentor.Name))
	} else {
		parts = append(parts, "Наставника тебе пока не назначили.")
	}
	if c != nil && c.Lead != nil {
		parts = append(parts, fmt.Sprintf("Руководитель команды – %s: согласует доступ к GitLab, отгулы и цели стажировки.", c.Lead.Name))
	}
	parts = append(parts, "Все контакты и отделы – в разделе «Поддержка».")
	return a.say(ctx, t, "text", a.phrase(ctx, t, strings.Join(parts, " ")), "mentor", nil)
}

// reEdit – стажер правит неподтвержденную карточку: «поменяй», «сделай только чтение», «на другую дату».
var reEdit = regexp.MustCompile(`измени|поменя|исправ|замени|сделай|вместо|не та|не тот|друг(ой|ую|ая)|перенес|передвин|добавь|допиши|уровень|только чтени|только смотр|на чтение|на запись|комментари`)

// editDraft – правка карточки текстом. done=false – правка не распознана, вопрос уходит модели.
func (a *Agent) editDraft(ctx context.Context, t *turn, text string) (bool, error) {
	q := t.draft
	low := strings.ToLower(text)
	switch q.Type {
	case "access":
		level := ""
		switch {
		case regexp.MustCompile(`maintainer|админ|администр`).MatchString(low):
			level = "admin"
		case regexp.MustCompile(`чтени|смотр|read|reporter|просмотр`).MatchString(low):
			level = "read"
		case regexp.MustCompile(`редакт|запис|developer|выклад|деплой|пуш|push|изменени`).MatchString(low):
			level = "edit"
		}
		if level == "" {
			return false, nil
		}
		if level == q.Level {
			return true, a.say(ctx, t, "text", a.phrase(ctx, t, fmt.Sprintf(
				"В карточке уже уровень «%s», она уже выше: проверь поля и нажми «Подтвердить».", q.LevelTitle)), "mentor", nil)
		}
		return true, a.prepareAccess(ctx, t, q.ServiceCode, "", level, "", q.Justification, q.TaskID)
	case "leave":
		today := a.now().In(moscow)
		if _, ok := ParseLeaveDate(text, today); !ok {
			return false, nil
		}
		return true, a.prepareLeave(ctx, t, text, "", q.TaskID)
	case "other":
		if !regexp.MustCompile(`комментари|добавь|допиши|укажи|измени|поменя|исправ`).MatchString(low) {
			return false, nil
		}
		comment := reCommentLead.ReplaceAllString(strings.TrimSpace(text), "")
		return true, a.prepareService(ctx, t, q.ItemCode, comment)
	}
	return false, nil
}

// reCommentLead – «добавь комментарий:» в начале правки: в заявку уходит только сам комментарий.
var reCommentLead = regexp.MustCompile(`(?i)^(добавь|допиши|укажи|измени|поменяй|исправь)\s+(в\s+)?(комментари\S*|заявк\S*)?\s*[:,–-]?\s*`)

// dropDraft – перед новой карточкой старая неподтвержденная отменяется: в чате одна карточка «Подтвердить».
// same – новая карточка про то же самое (правка), иначе стажер переключился на другую заявку.
func (a *Agent) dropDraft(ctx context.Context, t *turn, same bool) error {
	q := t.draft
	if q == nil {
		return nil
	}
	if err := a.reqs.Cancel(ctx, q.ID, t.viewer.ID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	t.draft = nil
	if same {
		t.replaced = true
	} else {
		t.dropped = RequestTitle(q)
	}
	return nil
}

// droppedNote – пояснение, что прошлая неподтвержденная карточка убрана.
func droppedNote(t *turn) string {
	if t.dropped == "" {
		return ""
	}
	return fmt.Sprintf("Карточку «%s» выше убрал – подтвердить можно одну заявку за раз. ", t.dropped)
}

// articleFor – статья-источник для пояснения к заявке.
func (a *Agent) articleFor(ctx context.Context, t *turn, query string) *string {
	found, err := a.kb.Search(ctx, t.viewer.TeamID, t.viewer.ID, query, 1, true)
	if err != nil || len(found) == 0 {
		return nil
	}
	return &found[0].ArticleID
}

// ---------- служебное ----------

func (a *Agent) begin(ctx context.Context, viewerID string) (*turn, error) {
	u, err := a.users.GetByID(ctx, viewerID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	if u.Role != models.RoleIntern {
		return nil, ErrForbidden
	}
	conv, err := a.chat.BuddyConversation(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	c, err := a.people.Circle(ctx, u.ID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	return &turn{viewer: u, conv: conv, circle: c}, nil
}

// draft – черновик заявки текущего стажера. Чужой, выполненный или отмененный – 404.
func (a *Agent) draft(ctx context.Context, viewerID, actionID string) (*turn, *models.Request, error) {
	t, err := a.begin(ctx, viewerID)
	if err != nil {
		return nil, nil, err
	}
	q, err := a.reqs.Get(ctx, actionID)
	if err != nil {
		return nil, nil, mapNotFound(err)
	}
	if q.RequesterID != t.viewer.ID || q.Status != "draft" {
		return nil, nil, ErrNotFound
	}
	return t, q, nil
}

// ask – вызов модели с таймаутом; при ошибке ответ собирается офлайн (mock), чтобы чат не падал.
func (a *Agent) ask(ctx context.Context, t *turn, req llm.Request) llm.Response {
	started := a.now()
	cctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	resp, err := a.model.Chat(cctx, req)
	a.logCall(ctx, t.conv, "llm", a.model.Name(), map[string]int{"messages": len(req.Messages)}, resp, err, started)
	if err != nil {
		a.log.Warn("llm failed, using offline answer", "provider", a.model.Name(), "err", err)
		resp, _ = a.fallback.Chat(ctx, req)
	}
	return resp
}

// phrase – пояснение к карточке словами модели. Факты (кто согласует, сроки) собирает backend,
// модель только формулирует. Ответ без любого из чисел или имен исходного текста отбрасывается,
// как и пустой или слишком длинный: тогда остается текст backend. В режиме mock – сразу он.
func (a *Agent) phrase(ctx context.Context, t *turn, facts string) string {
	if a.isMock() {
		return facts
	}
	ask := "Факты: " + facts
	if t.question != "" {
		ask = "Сообщение стажера: " + t.question + "\n" + ask
	}
	resp := a.ask(ctx, t, llm.Request{Messages: []llm.Message{
		{Role: "system", Content: `Ты AI-buddy, помощник стажера. Перескажи факты живым коротким ответом стажеру, на «ты», по-русски,
как коллега в мессенджере: 1–3 предложения, список – если в фактах список.
Сохрани все номера, имена, сроки и названия кнопок в кавычках. Не добавляй новых фактов, людей, сроков и обещаний.
Без markdown и приветствий. Ответь только текстом сообщения.`},
		{Role: "user", Content: ask},
	}})
	out := cleanText(resp.Content)
	if out == "" || len([]rune(out)) > 2*len([]rune(facts))+80 || len(resp.ToolCalls) > 0 {
		return facts
	}
	for _, w := range reFact.FindAllString(facts, -1) {
		if !strings.Contains(out, strings.TrimLeft(w, " (")) {
			return facts
		}
	}
	return out
}

// reFact – числа и слова с заглавной буквы (имена, сервисы) в середине фразы: они должны остаться в пересказе.
var reFact = regexp.MustCompile(`\d+|[ (]\p{Lu}[\p{L}-]+`)

// sayOpts – необязательные части сообщения: заявка (карточка), статья-источник или подпись.
type sayOpts struct {
	request *models.Request
	article *string
	label   *string
}

// say записывает сообщение помощника и добавляет его в ответ.
func (a *Agent) say(ctx context.Context, t *turn, kind, text, mood string, o *sayOpts) error {
	if o == nil {
		o = &sayOpts{}
	}
	m := repository.NewMessage{ConversationID: t.conv, AuthorKind: "assistant", Kind: kind, Text: text, Mood: &mood,
		ArticleID: o.article, SourceLabel: o.label}
	if o.request != nil {
		m.RequestID = &o.request.ID
	}
	id, at, err := a.chat.AddMessage(ctx, m)
	if err != nil {
		return err
	}
	msg := models.ChatMessage{ID: id, Role: "assistant", Kind: kind, Text: text, Mood: mood, At: at}
	switch {
	case o.article != nil:
		msg.Source = a.source(ctx, *o.article)
	case o.label != nil:
		msg.Source = &models.MessageSource{Title: *o.label}
	}
	if o.request != nil {
		attachCard(&msg, o.request)
	}
	t.out = append(t.out, msg)
	return nil
}

// source – подпись «Раздел → Статья» под ответом.
func (a *Agent) source(ctx context.Context, articleID string) *models.MessageSource {
	title, err := a.kb.ArticleRef(ctx, articleID)
	if err != nil {
		title = "Справочник"
	}
	return &models.MessageSource{Title: title, ArticleID: &articleID}
}

// attachCard – карточка подтверждения или результата по состоянию заявки.
func attachCard(msg *models.ChatMessage, q *models.Request) {
	switch msg.Kind {
	case "confirm":
		state := "confirmed"
		switch q.Status {
		case "draft":
			state = "pending"
		case "cancelled":
			state = "cancelled"
		}
		title := "Создать заявку на доступ?"
		switch q.Type {
		case "leave":
			title = "Создать заявку на отгул?"
		case "other":
			title = "Отправить заявку: " + q.DepartmentTitle + "?"
		}
		msg.Confirm = &models.ConfirmCard{ActionID: q.ID, Title: title, Fields: RequestFields(q), State: state}
	case "result":
		status := q.Status
		if status == "draft" || status == "cancelled" {
			status = "waiting"
		}
		msg.Result = &models.ResultCard{RequestID: q.ID, Number: fmt.Sprint(q.Number), Title: RequestTitle(q), Status: status}
	}
}

func (a *Agent) logCall(ctx context.Context, conv, server, tool string, in, out any, err error, started time.Time) {
	status := "ok"
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		status = "timeout"
	case err != nil:
		status, out = "error", map[string]string{"error": err.Error()}
	case out == "denied":
		status = "denied"
	}
	if e := a.chat.LogToolCall(ctx, repository.ToolCall{
		ConversationID: conv, Server: server, Tool: tool, Status: status, Input: in, Output: out, Latency: a.now().Sub(started),
	}); e != nil {
		a.log.Warn("tool call log failed", "err", e)
	}
}

func first(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

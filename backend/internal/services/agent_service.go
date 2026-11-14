package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"ai-buddy/backend/internal/llm"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// Заявки в отделы (оборудование, ИТ-поддержка, HR, бухгалтерия) и служебные функции агента.

// prepareService – черновик заявки в отдел: ноутбук, VPN, справка и т. п. Решение принимает
// отдел (в демо – автоматически в mock-services), наставник получает уведомление.
func (a *Agent) prepareService(ctx context.Context, t *turn, itemText, comment string) error {
	code, err := a.reqs.ResolveItem(ctx, itemText)
	if errors.Is(err, repository.ErrNotFound) {
		return a.catalogHelp(ctx, t)
	}
	if err != nil {
		return err
	}
	item, err := a.reqs.CatalogItem(ctx, code)
	if err != nil {
		return mapNotFound(err)
	}
	if q := t.draft; q != nil && q.Type == "other" && q.ItemCode == item.Code &&
		(strings.TrimSpace(comment) == "" || strings.TrimSpace(comment) == q.ItemComment) {
		return a.say(ctx, t, "text", a.phrase(ctx, t, "Карточка на это уже выше: проверь ее и нажми «Подтвердить»."), "mentor", nil)
	}
	if open, err := a.reqs.OpenServiceRequest(ctx, t.viewer.ID, item.Code); err == nil {
		return a.say(ctx, t, "text", a.phrase(ctx, t, fmt.Sprintf(
			"Заявка №%d «%s» уже в работе: %s, обычно до %d ч. Ответ придет сюда же.",
			open.Number, RequestTitle(open), open.DepartmentTitle, open.SLAHours)), "calm", nil)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	if err := a.dropDraft(ctx, t, t.draft != nil && t.draft.Type == "other" && t.draft.ItemCode == item.Code); err != nil {
		return err
	}
	q, err := a.reqs.CreateServiceDraft(ctx, models.ServiceDraft{
		RequesterID: t.viewer.ID, Item: item.Code, Comment: clip(strings.TrimSpace(comment), 500), TaskID: t.actionTask,
	})
	if err != nil {
		return mapNotFound(err)
	}
	intro := fmt.Sprintf("%s оформляется заявкой, ее выполнит %s, обычно до %d ч.", item.Title, item.DepartmentTitle, item.SLAHours)
	if item.Code == chatsItem {
		intro = "Добавлю тебя в чаты и каналы команды сразу после подтверждения."
	}
	if t.replaced {
		intro = "Обновил карточку, старую убрал. " + intro
	}
	intro = droppedNote(t) + intro
	if t.circle != nil && t.circle.Mentor != nil {
		intro += fmt.Sprintf(" Наставника (%s) я предупрежу.", t.circle.Mentor.Name)
	}
	if item.Hint != "" {
		intro += " " + item.Hint
	}
	source := a.articleFor(ctx, t, item.Title+" "+item.DepartmentTitle)
	if err := a.say(ctx, t, "text", a.phrase(ctx, t, intro), "calm", &sayOpts{article: source}); err != nil {
		return err
	}
	title := "Отправить заявку?"
	if item.Code == chatsItem {
		title = "Добавить в чаты и каналы?"
	}
	return a.say(ctx, t, "confirm", title, "mentor", &sayOpts{request: q})
}

// chatsItem – заявка «добавить в чаты и каналы» (действие join_chats у задачи пути): выполняется сразу.
const chatsItem = "team_chats"

// catalogHelp – что можно заказать через помощника, по отделам.
func (a *Agent) catalogHelp(ctx context.Context, t *turn) error {
	deps, err := a.people.Departments(ctx)
	if err != nil {
		return err
	}
	lines := []string{"Не понял, что именно нужно. Через меня можно оформить заявку:"}
	for _, d := range deps {
		lines = append(lines, fmt.Sprintf("– %s: %s", d.Title, strings.ToLower(d.Items)))
	}
	lines = append(lines, "Напиши, например, «нужен монитор» или «нужна справка 2-НДФЛ».")
	return a.say(ctx, t, "text", strings.Join(lines, "\n"), "think", nil)
}

// notifyMentor – уведомление наставнику о заявке стажера (отгул или заявка в отдел).
// Возвращает фразу для ответа стажеру или пустую строку.
func (a *Agent) notifyMentor(ctx context.Context, t *turn, q *models.Request) string {
	if q.Type == "access" || t.circle == nil || t.circle.Mentor == nil {
		return ""
	}
	text := fmt.Sprintf("%s: заявка №%d «%s» – %s", t.viewer.Name, q.Number, RequestTitle(q), approverText(q))
	if err := a.reqs.Notify(ctx, t.circle.Mentor.ID, "requests", text, "/interns/"+t.viewer.ID, &q.ID); err != nil {
		a.log.Warn("mentor notification failed", "err", err)
		return ""
	}
	return fmt.Sprintf(" Наставник (%s) получил уведомление.", t.circle.Mentor.Name)
}

// personal – текст справочника с людьми этого стажера вместо меток {{mentor}}, {{lead}}, {{team}}.
func (a *Agent) personal(t *turn, s string) string { return personalize(t.circle, t.viewer.Name, s) }

func (a *Agent) isMock() bool { return a.model.Name() == (llm.Mock{}).Name() }

// clip обрезает текст до n символов по границе слова.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := string(r[:n])
	if i := strings.LastIndexAny(cut, " \n"); i > n/2 {
		cut = cut[:i]
	}
	return cut + "…"
}

var (
	reMDBold   = regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	reMDHead   = regexp.MustCompile(`(?m)^#{1,6}\s*`)
	reMDBullet = regexp.MustCompile(`(?m)^\s*[*•]\s+`)
)

// cleanText убирает markdown из ответа модели: фронт показывает текст как есть.
func cleanText(s string) string {
	// служебная пометка из истории, которую модель иногда повторяет
	s = strings.ReplaceAll(s, "[карточка подтверждения] ", "")
	s = strings.ReplaceAll(s, "[карточка подтверждения]", "")
	s = reMDBold.ReplaceAllString(s, "$1$2")
	s = reMDHead.ReplaceAllString(s, "")
	s = reMDBullet.ReplaceAllString(s, "– ")
	s = strings.NewReplacer("—", "–", "ё", "е", "Ё", "Е").Replace(s)
	return strings.TrimSpace(s)
}

package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ai-buddy/backend/internal/models"
)

func TestParseLeaveDate(t *testing.T) {
	today := time.Date(2026, 11, 11, 15, 0, 0, 0, moscow) // среда
	cases := map[string]string{
		"отгул завтра":        "2026-11-12",
		"послезавтра":         "2026-11-13",
		"14 ноября":           "2026-11-14",
		"3 января":            "2027-01-03",
		"1 мая":               "2027-05-01",
		"5 марта":             "2027-03-05",
		"20.11":               "2026-11-20",
		"20.11.27":            "2027-11-20",
		"в пятницу":           "2026-11-13",
		"в среду":             "2026-11-18",
		"в понедельник утром": "2026-11-16",
	}
	for text, want := range cases {
		got, ok := ParseLeaveDate(text, today)
		assert.True(t, ok, text)
		assert.Equal(t, want, got.Format(time.DateOnly), text)
	}
	for _, text := range []string{"когда-нибудь", "31 февраля", "10.13", "31.02.2027"} {
		_, ok := ParseLeaveDate(text, today)
		assert.False(t, ok, text)
	}
	assert.Equal(t, "2026-11-13", AddWorkdays(today, 2).Format(time.DateOnly))
	assert.Equal(t, "2026-11-16", AddWorkdays(today, 3).Format(time.DateOnly))
	assert.Equal(t, 3, WorkdaysBetween(today, time.Date(2026, 11, 16, 0, 0, 0, 0, moscow)))
}

func TestPickRole(t *testing.T) {
	roles := []models.ServiceRole{
		{Code: "reporter", Level: "read", Allowed: true},
		{Code: "developer", Level: "edit", Allowed: true},
		{Code: "maintainer", Level: "admin", Allowed: false},
	}
	c, alt := pickRole(roles, "", "")
	assert.Equal(t, "developer", c.Code)
	assert.Nil(t, alt)
	c, _ = pickRole(roles, "", "read")
	assert.Equal(t, "reporter", c.Code)
	c, alt = pickRole(roles, "maintainer", "")
	assert.False(t, c.Allowed)
	assert.Equal(t, "developer", alt.Code)
	c, alt = pickRole([]models.ServiceRole{{Code: "read", Level: "read"}}, "", "")
	assert.Equal(t, "read", c.Code)
	assert.Nil(t, alt)
	c, _ = pickRole(nil, "", "")
	assert.Nil(t, c)
}

func TestRequestTexts(t *testing.T) {
	proj := "backend / auth-service"
	q := &models.Request{
		ID: "1", Number: 1042, Type: "access", Status: "waiting", Via: "buddy",
		ServiceTitle: "GitLab", RequestTitle: "Доступ к GitLab", RoleTitle: "Developer", LevelTitle: "редактирование",
		ProjectTitle: &proj, Justification: "Для задачи", ApproverKind: "project_owner", SLAHours: 24,
		Approver:  &models.PersonRef{ID: "o", Name: "Олег"},
		CreatedAt: time.Now().Add(-30 * time.Hour),
	}
	item := RequestItem(q, []models.RequestEvent{
		{Status: "draft"}, {Status: "waiting", At: time.Now()},
	}, time.Now())
	assert.Equal(t, "Олег, владелец проекта", item.Fields[len(item.Fields)-1].Value)
	assert.Len(t, item.History, 2)
	assert.Equal(t, "Создана AI-buddy", item.History[0].Title)
	assert.Equal(t, "дольше обычного, AI-buddy напомнит", item.History[1].At)

	q.Status = "rejected"
	reason := "нет"
	q.RejectReason = &reason
	item = RequestItem(q, []models.RequestEvent{{Status: "rejected"}, {Status: "cancelled"}}, time.Now())
	assert.Equal(t, "rejected", item.History[0].State)
	assert.Equal(t, "Решение", item.Fields[len(item.Fields)-1].Label)

	day := time.Date(2026, 11, 13, 0, 0, 0, 0, moscow)
	leave := &models.Request{Number: 7, Type: "leave", Status: "approved", Via: "manual", LeaveDate: &day,
		LeaveTypeTitle: "отгул", LeaveComment: "к врачу", Approver: &models.PersonRef{Name: "Олег"}}
	item = RequestItem(leave, []models.RequestEvent{{Status: "waiting"}, {Status: "approved"}, {Status: "done"}}, time.Now())
	assert.Equal(t, "Отгул 13 ноября", item.Title)
	assert.Equal(t, "13 ноября, пятница", item.Fields[0].Value)
	assert.Equal(t, "Олег, руководитель", item.Fields[len(item.Fields)-1].Value)
	assert.Equal(t, []string{"Создана вручную", "Одобрена", "Выполнена"}, []string{item.History[0].Title, item.History[1].Title, item.History[2].Title})
	assert.Contains(t, decisionNotice(leave, "approved", ""), "согласован")
	assert.Contains(t, decisionNotice(q, "rejected", "нет"), "отклонена: нет")
	assert.Empty(t, taskNote(q, "waiting", ""))

	auto := &models.Request{Type: "access", ApproverKind: "auto", RoleTitle: "чтение", LevelTitle: "чтение"}
	assert.Equal(t, "выдается автоматически", approverText(auto))
	assert.Equal(t, "чтение", levelText(auto))
	assert.Equal(t, "Заявка", RequestTitle(auto))
	assert.Equal(t, "Кирилл", approverText(&models.Request{ApproverKind: "other", Approver: &models.PersonRef{Name: "Кирилл"}}))

	for in, want := range map[[2]string]string{
		{"access", "granted"}: "done", {"access", "approved"}: "done", {"leave", "approved"}: "approved",
		{"leave", "rejected"}: "rejected", {"leave", "waiting"}: "waiting",
	} {
		got, ok := ticketStatus(in[0], in[1])
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}
	_, ok := ticketStatus("access", "lost")
	assert.False(t, ok)
	assert.Equal(t, "", capitalize(""))
	assert.Equal(t, "", lowerFirst(""))
	assert.Equal(t, "", deref(nil))
	assert.Equal(t, "", first(" ", ""))
}

func TestGenitiveAndActorKind(t *testing.T) {
	for name, want := range map[string]string{
		"Анна": "Анны", "Лиза": "Лизы", "Ольга": "Ольги", "Дмитрий": "Дмитрия", "Игорь": "Игоря",
		"Павел": "Павела", "Олег": "Олега", "Мария": "Марии", "Я": "Я", "Ли": "Ли",
	} {
		assert.Equal(t, want, genitive(name), name)
	}
	assert.Equal(t, "intern", actorKind(models.RoleIntern))
	assert.Equal(t, "mentor", actorKind(models.RoleMentor))
	assert.Equal(t, "lead", actorKind(models.RoleLead))
	assert.Equal(t, "other", actorKind("staff"))
}

func TestExternalRef(t *testing.T) {
	ref := "ACCESS-8001"
	SetJiraURL("")
	e := External(&models.Request{Type: "access", ExternalRef: &ref})
	assert.Equal(t, "ACCESS-8001", e.Key)
	assert.Nil(t, e.URL)
	SetJiraURL("http://localhost:8090/")
	defer SetJiraURL("")
	e = External(&models.Request{Type: "other", ExternalRef: &ref})
	assert.Equal(t, "http://localhost:8090/browse/ACCESS-8001", *e.URL)
	assert.Nil(t, External(&models.Request{Type: "leave", ExternalRef: &ref}), "отгул остается у руководителя")
	assert.Nil(t, External(&models.Request{Type: "access"}))
}

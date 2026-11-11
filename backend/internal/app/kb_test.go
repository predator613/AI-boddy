package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/models"
)

// Справочник: формат ответа совпадает со схемой фронта, чужие материалы скрыты.

const (
	backendOnlyArticle  = "c2000000-0000-4000-8000-000000000014" // «Кто за что отвечает в команде Backend»
	frontendOnlyArticle = "c2000000-0000-4000-8000-000000000015" // «Команда Frontend: к кому идти»
	accessArticle       = "c2000000-0000-4000-8000-000000000004" // этап 02 Backend
)

func TestKnowledgeBaseForAnna(t *testing.T) {
	e := newEnv(t)
	var kb models.KnowledgeBase
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/kb", nil, &kb))

	require.Len(t, kb.Sections, 8)
	assert.Equal(t, "s-start", kb.Sections[0].ID)
	assert.NotEmpty(t, kb.Terms)

	ids := map[string]models.KBArticle{}
	total := 0
	for _, a := range kb.Articles {
		ids[a.ID] = a
		assert.GreaterOrEqual(t, a.Minutes, 1)
	}
	for _, s := range kb.Sections {
		total += s.Count
	}
	assert.Equal(t, len(kb.Articles), total, "счетчики разделов считают только видимые статьи")
	assert.Contains(t, ids, backendOnlyArticle)
	assert.NotContains(t, ids, frontendOnlyArticle)
	require.NotNil(t, ids[accessArticle].StageCode)
	assert.Equal(t, "02", *ids[accessArticle].StageCode)
	assert.Equal(t, "s-access", ids[accessArticle].SectionID)
}

func TestKnowledgeBaseForPavel(t *testing.T) {
	e := newEnv(t)
	var kb models.KnowledgeBase
	require.Equal(t, http.StatusOK, e.call("intern_pavel", http.MethodGet, "/api/kb", nil, &kb))
	ids := map[string]models.KBArticle{}
	for _, a := range kb.Articles {
		ids[a.ID] = a
	}
	assert.Contains(t, ids, frontendOnlyArticle)
	assert.NotContains(t, ids, backendOnlyArticle)
	// Этап статьи – из шаблона Backend, у Frontend его нет.
	assert.Nil(t, ids[accessArticle].StageCode)
}

func TestKnowledgeArticle(t *testing.T) {
	e := newEnv(t)
	var a models.KBArticleDetail
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/kb/articles/"+accessArticle, nil, &a))
	assert.Equal(t, "Как запросить доступ к сервису", a.Title)
	assert.Contains(t, a.Body, "\n\n")
	assert.Equal(t, "seed", a.Source)
	assert.True(t, a.UseInAI)

	var answer models.KBArticleDetail
	require.Equal(t, http.StatusOK, e.call("mentor_igor", http.MethodGet, "/api/kb/articles/c2000000-0000-4000-8000-000000000016", nil, &answer))
	require.NotNil(t, answer.AuthorName)
	assert.Equal(t, "Игорь", *answer.AuthorName)

	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodGet, "/api/kb/articles/"+frontendOnlyArticle, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_anna", http.MethodGet, "/api/kb/articles/not-an-id", nil, nil))
	assert.Equal(t, http.StatusUnauthorized, e.call("nobody", http.MethodGet, "/api/kb", nil, nil))
}

// В статьях нет имен: наставник и руководитель подставляются из связей читателя.
func TestKnowledgePersonalized(t *testing.T) {
	e := newEnv(t)
	article := func(user, id string) models.KBArticleDetail {
		t.Helper()
		var a models.KBArticleDetail
		require.Equal(t, http.StatusOK, e.call(user, http.MethodGet, "/api/kb/articles/"+id, nil, &a))
		assert.NotContains(t, a.Body, "{{")
		return a
	}
	anna := article("intern_anna", backendOnlyArticle)
	assert.Contains(t, anna.Body, "Наставник (Игорь)")
	assert.Contains(t, anna.Body, "Руководитель команды (Олег)")

	pavel := article("intern_pavel", frontendOnlyArticle)
	assert.Contains(t, pavel.Body, "(Нина)")
	assert.Contains(t, pavel.Body, "Наставник: пока не назначен")

	igor := article("mentor_igor", "c2000000-0000-4000-8000-000000000002")
	assert.Contains(t, igor.Body, "наставником (Игорь)")

	// Поиск отдает фрагменты тоже с именами читателя.
	var res models.KBSearch
	require.Equal(t, http.StatusOK, e.call("intern_pavel", http.MethodGet, "/api/kb/search?q="+url.QueryEscape("кто согласует отгул"), nil, &res))
	for _, m := range res.Results {
		assert.NotContains(t, m.Snippet, "{{")
		assert.NotContains(t, m.Snippet, "Олег")
	}

	// Контакты для «Поддержки»: свои люди и отделы.
	var c models.Contacts
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/me/contacts", nil, &c))
	require.NotNil(t, c.Mentor)
	assert.Equal(t, "Игорь", c.Mentor.Name)
	require.NotNil(t, c.Lead)
	assert.Equal(t, "Олег", c.Lead.Name)
	assert.Equal(t, "Backend", c.Team)
	require.Len(t, c.Departments, 4)
	assert.Equal(t, "Отдел оборудования", c.Departments[0].Title)
	assert.Contains(t, c.Departments[0].Items, "Ноутбук")

	require.Equal(t, http.StatusOK, e.call("intern_pavel", http.MethodGet, "/api/me/contacts", nil, &c))
	assert.Nil(t, c.Mentor)
	assert.Equal(t, "Нина", c.Lead.Name)
	assert.Equal(t, http.StatusUnauthorized, e.call("nobody", http.MethodGet, "/api/me/contacts", nil, nil))
}

func TestKnowledgeSearch(t *testing.T) {
	e := newEnv(t)
	search := func(user, q string) models.KBSearch {
		t.Helper()
		var res models.KBSearch
		require.Equal(t, http.StatusOK, e.call(user, http.MethodGet, "/api/kb/search?q="+url.QueryEscape(q), nil, &res))
		return res
	}

	res := search("intern_anna", "как получить доступ к тестовому стенду")
	require.NotEmpty(t, res.Results)
	top := []string{}
	for _, m := range res.Results[:min(3, len(res.Results))] {
		top = append(top, m.ArticleID)
		assert.NotEmpty(t, m.Snippet)
	}
	assert.Contains(t, top, accessArticle, "статья про доступы в первой тройке")
	assert.Contains(t, strings.ToLower(res.Results[0].Title), "стенд")

	res = search("intern_anna", "как взять отгул")
	require.NotEmpty(t, res.Results)
	assert.Equal(t, "Как взять отгул", res.Results[0].Title)

	// Морфология: «ревью» и «ревьюер» находят статью про код-ревью.
	res = search("intern_anna", "код-ревью")
	require.NotEmpty(t, res.Results)

	// Чужая команда не попадает в выдачу.
	for _, m := range search("intern_anna", "руководитель Frontend Нина").Results {
		assert.NotEqual(t, frontendOnlyArticle, m.ArticleID)
	}
	found := false
	for _, m := range search("intern_pavel", "руководитель Frontend Нина").Results {
		found = found || m.ArticleID == frontendOnlyArticle
	}
	assert.True(t, found)

	// Ничего не найдено – пустой список, не ошибка.
	assert.Empty(t, search("intern_anna", "квантовая хромодинамика").Results)

	var limited models.KBSearch
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/kb/search?limit=1&q=доступ", nil, &limited))
	assert.Len(t, limited.Results, 1)
	require.Equal(t, http.StatusOK, e.call("intern_anna", http.MethodGet, "/api/kb/search?limit=500&q=доступ", nil, &limited))
	assert.LessOrEqual(t, len(limited.Results), 20)

	assert.Equal(t, http.StatusBadRequest, e.call("intern_anna", http.MethodGet, "/api/kb/search?q=%20", nil, nil))
	long := make([]byte, 201)
	for i := range long {
		long[i] = 'a'
	}
	assert.Equal(t, http.StatusBadRequest, e.call("intern_anna", http.MethodGet, "/api/kb/search?q="+string(long), nil, nil))
}

package app

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-buddy/backend/internal/models"
)

const (
	annaOneOnOne = "f1000000-0000-4000-8000-000000000005"
	devopsIntro  = "f1000000-0000-4000-8000-000000000006"
	lizaOneOnOne = "f1000000-0000-4000-8000-000000000002"
)

func TestInternCalendar(t *testing.T) {
	e := newEnv(t)
	pool := testPool(t)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM calendar_entries WHERE title LIKE 'test %'`)
		_, _ = pool.Exec(ctx, `DELETE FROM meeting_answers WHERE meeting_id IN ($1, $2)`, annaOneOnOne, devopsIntro)
		_, _ = pool.Exec(ctx, `DELETE FROM messages WHERE text LIKE 'Можно перенести%'`)
		_, _ = pool.Exec(ctx, `DELETE FROM notifications WHERE kind = 'meetings' AND text LIKE 'Мария%'`)
	})
	msk, _ := time.LoadLocation("Europe/Moscow")
	var oneOnOne time.Time
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT starts_at FROM meetings WHERE id = $1`, annaOneOnOne).Scan(&oneOnOne))
	day := oneOnOne.In(msk)
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, msk)
	at := func(h, m int) time.Time { return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute) }
	saturday := day
	for saturday.Weekday() != time.Saturday {
		saturday = saturday.AddDate(0, 0, 1)
	}
	assert.NotContains(t, []time.Weekday{time.Saturday, time.Sunday}, day.Weekday(), "демо-встречи переносятся на будни")

	list := func(from, to time.Time) []models.CalendarEvent {
		t.Helper()
		var v models.CalendarView
		q := "/api/me/calendar?from=" + url.QueryEscape(from.Format(time.RFC3339)) + "&to=" + url.QueryEscape(to.Format(time.RFC3339))
		require.Equal(t, http.StatusOK, e.call("intern_maria", http.MethodGet, q, nil, &v))
		return v.Events
	}
	find := func(evs []models.CalendarEvent, id string) *models.CalendarEvent {
		for i := range evs {
			if evs[i].ID == id {
				return &evs[i]
			}
		}
		return nil
	}

	// Неделя: 1:1 с наставником, знакомство с DevOps, встреча стажеров и свои записи.
	week := list(day.AddDate(0, 0, -1), day.AddDate(0, 0, 6))
	one := find(week, annaOneOnOne)
	require.NotNil(t, one)
	assert.Equal(t, "mentor", one.Kind)
	assert.Equal(t, "Игорь, наставник", one.WithName)
	assert.Equal(t, "видеозвонок", one.Place)
	assert.Equal(t, "pending", one.Status)
	assert.Equal(t, 45, one.DurationMin)
	intro := find(week, devopsIntro)
	require.NotNil(t, intro)
	assert.Equal(t, "intro", intro.Kind)
	assert.Equal(t, "Кирилл, команда Платформа", intro.WithName)
	team := find(week, "f1000000-0000-4000-8000-000000000003")
	require.NotNil(t, team)
	assert.Equal(t, "accepted", team.Status)
	assert.Nil(t, find(week, lizaOneOnOne), "чужая встреча не видна")
	var own int
	for i, ev := range week {
		if i > 0 {
			assert.False(t, ev.StartsAt.Before(week[i-1].StartsAt), "по возрастанию startsAt")
		}
		if ev.Status == "own" {
			own++
			assert.Equal(t, "только ты", ev.WithName)
		}
	}
	assert.Equal(t, 2, own)
	_, offset := one.StartsAt.Zone()
	assert.Equal(t, 3*3600, offset, "время по Москве")

	// Своя запись после 1:1: сохраняется и удаляется.
	var ev models.CalendarEvent
	require.Equal(t, http.StatusCreated, e.call("intern_maria", http.MethodPost, "/api/me/calendar/entries", map[string]any{
		"title": "test фокус", "kind": "focus", "startsAt": at(16, 0), "durationMin": 60, "shareWithMentor": false,
	}, &ev))
	assert.Equal(t, "own", ev.Status)
	assert.False(t, ev.Shared)
	assert.NotNil(t, find(list(day, day.AddDate(0, 0, 1)), ev.ID))

	// Пересечение со встречей – 409 с понятным текстом.
	var errBody struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	require.Equal(t, http.StatusConflict, e.call("intern_maria", http.MethodPost, "/api/me/calendar/entries", map[string]any{
		"title": "test учеба", "kind": "study", "startsAt": at(15, 30), "durationMin": 30,
	}, &errBody))
	assert.Equal(t, "conflict", errBody.Error.Code)
	assert.Equal(t, "Пересекается с «1:1 с наставником»", errBody.Error.Message)
	assert.Equal(t, http.StatusConflict, e.call("intern_maria", http.MethodPost, "/api/me/calendar/entries", map[string]any{
		"title": "test личное", "kind": "personal", "startsAt": at(16, 30), "durationMin": 15,
	}, nil), "пересечение со своей записью")

	for name, body := range map[string]map[string]any{
		"после 18:00":  {"title": "test поздно", "kind": "focus", "startsAt": at(17, 30), "durationMin": 60},
		"пустое":       {"title": " ", "kind": "focus", "startsAt": at(9, 0), "durationMin": 30},
		"вид":          {"title": "test вид", "kind": "party", "startsAt": at(9, 0), "durationMin": 30},
		"длительность": {"title": "test мало", "kind": "focus", "startsAt": at(9, 0), "durationMin": 5},
		"без времени":  {"title": "test без", "kind": "focus", "durationMin": 30},
		"раньше 10:00": {"title": "test рано", "kind": "focus", "startsAt": at(9, 30), "durationMin": 30},
		"выходной":     {"title": "test суббота", "kind": "focus", "startsAt": saturday.Add(12 * time.Hour), "durationMin": 30},
	} {
		assert.Equal(t, http.StatusBadRequest, e.call("intern_maria", http.MethodPost, "/api/me/calendar/entries", body, nil), name)
	}

	assert.Equal(t, http.StatusNotFound, e.call("intern_pavel", http.MethodDelete, "/api/me/calendar/entries/"+ev.ID, nil, nil), "чужая запись")
	assert.Equal(t, http.StatusNoContent, e.call("intern_maria", http.MethodDelete, "/api/me/calendar/entries/"+ev.ID, nil, nil))
	assert.Equal(t, http.StatusNotFound, e.call("intern_maria", http.MethodDelete, "/api/me/calendar/entries/"+ev.ID, nil, nil))

	// «Приду» – статус и уведомление организатору.
	answer := func(meeting string, body map[string]any) (int, models.CalendarEvent) {
		var out models.CalendarEvent
		code := e.call("intern_maria", http.MethodPost, "/api/me/calendar/events/"+meeting+"/answer", body, &out)
		return code, out
	}
	code, out := answer(devopsIntro, map[string]any{"answer": "accept"})
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "accepted", out.Status)
	var n int
	require.NoError(t, pool.QueryRow(t.Context(), `
		SELECT count(*) FROM notifications WHERE user_id = 'aaaaaaaa-0000-4000-8000-000000000008' AND text = 'Мария придет на «Знакомство с DevOps»'`).Scan(&n))
	assert.Equal(t, 1, n)

	// «Попросить перенести» – сообщение в чат с наставником.
	code, out = answer(annaOneOnOne, map[string]any{"answer": "move", "reason": "в это время пара"})
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "move_requested", out.Status)
	var text string
	require.NoError(t, pool.QueryRow(t.Context(), `
		SELECT m.text FROM messages m JOIN conversations c ON c.id = m.conversation_id
		WHERE c.kind = 'mentor' AND c.intern_id = $1 AND c.mentor_id = 'aaaaaaaa-0000-4000-8000-000000000002'
		ORDER BY m.created_at DESC LIMIT 1`, annaID).Scan(&text))
	assert.Equal(t, "Можно перенести «1:1 с наставником»? в это время пара", text)
	assert.Equal(t, "move_requested", find(list(day, day.AddDate(0, 0, 1)), annaOneOnOne).Status)

	code, _ = answer(lizaOneOnOne, map[string]any{"answer": "accept"})
	assert.Equal(t, http.StatusNotFound, code, "чужая встреча")
	code, _ = answer(annaOneOnOne, map[string]any{"answer": "maybe"})
	assert.Equal(t, http.StatusBadRequest, code)
	code, _ = answer(annaOneOnOne, map[string]any{"answer": "move", "reason": string(make([]rune, 201))})
	assert.Equal(t, http.StatusBadRequest, code)

	// Только для стажера; период проверяется.
	assert.Equal(t, http.StatusForbidden, e.call("mentor_igor", http.MethodGet, "/api/me/calendar?from=2026-11-01T00:00:00Z&to=2026-11-08T00:00:00Z", nil, nil))
	assert.Equal(t, http.StatusForbidden, e.call("lead_artemiy", http.MethodPost, "/api/me/calendar/entries", map[string]any{}, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("intern_maria", http.MethodGet, "/api/me/calendar?from=вчера&to=2026-11-08T00:00:00Z", nil, nil))
	assert.Equal(t, http.StatusBadRequest, e.call("intern_maria", http.MethodGet, "/api/me/calendar?from=2026-11-08T00:00:00Z&to=2026-11-01T00:00:00Z", nil, nil))
	assert.Equal(t, http.StatusUnauthorized, e.call("nobody", http.MethodGet, "/api/me/calendar", nil, nil))
}

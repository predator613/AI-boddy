package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
)

// MentorStore – данные кабинета наставника.
type MentorStore interface {
	Conversation(ctx context.Context, internID, mentorID string) (string, error)
	ThreadMessages(ctx context.Context, conv string) ([]models.ThreadMessage, error)
	Post(ctx context.Context, conv, authorKind, authorID, text string) error
	AddUpdate(ctx context.Context, internID, mentorID, kind, title, detail string, link *string) error
	Ack(ctx context.Context, conv, messageID string) error
	MarkRead(ctx context.Context, conv, userID string) error
	Summaries(ctx context.Context, mentorID string, internIDs []string) (map[string]models.ThreadSummary, error)
	AnswerHandoff(ctx context.Context, conv, internID, mentorName, answer string, articleID *string) (bool, error)
	LastHandoffQuestion(ctx context.Context, conv string) (string, error)
	Materials(ctx context.Context, teamID string) ([]models.Material, error)
	StageByCode(ctx context.Context, teamID, code string) (*string, error)
	CreateMaterial(ctx context.Context, d models.MaterialDraft, source string) (*models.Material, error)
	DeleteMaterial(ctx context.Context, teamID, id string) error
	Meetings(ctx context.Context, hostID string, from, to time.Time) ([]repository.MentorMeetingRow, error)
	Meeting(ctx context.Context, hostID, id string) (*repository.MentorMeetingRow, error)
	CreateMeeting(ctx context.Context, hostID, title, kind string, startsAt time.Time, durationMin int, interns []string) (*repository.MentorMeetingRow, error)
	MoveMeeting(ctx context.Context, hostID, id string, startsAt time.Time) error
	CancelMeeting(ctx context.Context, hostID, id string) error
	Automations(ctx context.Context, mentorID string) (map[string]bool, error)
	SetAutomation(ctx context.Context, mentorID, kind string, enabled bool) error
	RunAutomations(ctx context.Context, now time.Time) (int, error)
	Activity(ctx context.Context, hostID string, internIDs []string, since time.Time) (*repository.Activity, error)
}

// TeamInterns – стажеры команды.
type TeamInterns interface {
	ListTeamInterns(ctx context.Context, teamID string) ([]models.User, error)
}

// Константы статистики: минут наставника на один вопрос и ориентир Авито на стажировку.
const (
	minutesPerQuestion = 12
	mentorHoursPlan    = 44
	statsDays          = 7
	maxMaterialBytes   = 10 << 20
)

// automationInfo – правила AI-buddy по порядку.
var automationInfo = []models.Automation{
	{Kind: "deadline_reminder", Title: "Напоминание о сроке задачи", Note: "Стажер получает напоминание за день до срока"},
	{Kind: "weekly_goal_checkin", Title: "Еженедельный чек-ин по целям", Note: "По пятницам в 17:00 AI-buddy просит отметить прогресс"},
	{Kind: "overdue_alert", Title: "Сигнал о просрочке", Note: "Тебе приходит уведомление, если задача просрочена больше чем на 2 дня"},
	{Kind: "approver_reminder", Title: "Напоминание согласующему", Note: "Если заявка стажера ждет больше 3 дней, AI-buddy напомнит согласующему"},
}

// Mentor – кабинет наставника и переписка стажера с наставником.
type Mentor struct {
	store MentorStore
	users UserLookup
	team  TeamInterns
	path  PathReader
	log   *slog.Logger
	now   func() time.Time
}

// NewMentor создает сервис.
func NewMentor(store MentorStore, users UserLookup, team TeamInterns, path PathReader, log *slog.Logger) *Mentor {
	return &Mentor{store: store, users: users, team: team, path: path, log: log, now: time.Now}
}

func (s *Mentor) user(ctx context.Context, id string) (*models.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrForbidden
	}
	return u, err
}

// mentor – только наставник.
func (s *Mentor) mentor(ctx context.Context, id string) (*models.User, error) {
	u, err := s.user(ctx, id)
	if err == nil && u.Role != models.RoleMentor {
		return nil, ErrForbidden
	}
	return u, err
}

// staff – наставник или руководитель.
func (s *Mentor) staff(ctx context.Context, id string) (*models.User, error) {
	u, err := s.user(ctx, id)
	if err == nil && u.Role == models.RoleIntern {
		return nil, ErrForbidden
	}
	return u, err
}

// interns – стажеры, которых ведет пользователь: наставник – своих, руководитель – команды.
func (s *Mentor) interns(ctx context.Context, u *models.User) ([]models.User, error) {
	all, err := s.team.ListTeamInterns(ctx, u.TeamID)
	if err != nil {
		return nil, err
	}
	out := []models.User{}
	for _, i := range all {
		if CanManageIntern(u, &i) {
			out = append(out, i)
		}
	}
	return out, nil
}

func ids(list []models.User) []string {
	out := make([]string, len(list))
	for i, u := range list {
		out[i] = u.ID
	}
	return out
}

// myIntern – стажер наставника. Чужой – 404.
func (s *Mentor) myIntern(ctx context.Context, m *models.User, internID string) (*models.User, error) {
	i, err := s.users.GetByID(ctx, internID)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && (i.MentorID == nil || *i.MentorID != m.ID)) {
		return nil, ErrNotFound
	}
	return i, err
}

// stage – текущий этап стажера: «02», «Доступы».
func (s *Mentor) stage(ctx context.Context, internID string) (string, string) {
	if s.path == nil {
		return "", ""
	}
	v, err := s.path.GetPath(ctx, internID, "")
	if err != nil || len(v.Stages) == 0 {
		return "", ""
	}
	st := v.Stages[len(v.Stages)-1]
	for _, x := range v.Stages {
		if x.Status == models.StageCurrent {
			st = x
			break
		}
	}
	return st.Code, st.Title
}

// ---------- статистика ----------

// Stats – плашки на «Моих стажерах».
func (s *Mentor) Stats(ctx context.Context, viewerID string) (*models.MentorStats, error) {
	a, _, err := s.activity(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	answered := max(a.Questions-a.HandedOff, 0)
	return &models.MentorStats{
		PeriodDays: statsDays, AnsweredByBuddy: answered, HandedOff: a.HandedOff,
		SavedMinutes: answered * minutesPerQuestion, MentorHoursSpent: (a.MentorMinutes + 30) / 60, MentorHoursPlan: mentorHoursPlan,
	}, nil
}

// Metrics – польза AI-buddy одной строкой на главной наставника.
func (s *Mentor) Metrics(ctx context.Context, viewerID string) (*models.MentorMetrics, error) {
	a, _, err := s.activity(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	answered := max(a.Questions-a.HandedOff, 0)
	return &models.MentorMetrics{
		AnsweredByBuddy: answered, Escalated: a.HandedOff, MinutesPerQuestion: minutesPerQuestion,
		HoursSaved: float64(answered*minutesPerQuestion*10/60) / 10, AccessHours: a.AccessHours, OnTimePercent: a.OnTimePercent,
	}, nil
}

func (s *Mentor) activity(ctx context.Context, viewerID string) (*repository.Activity, *models.User, error) {
	u, err := s.staff(ctx, viewerID)
	if err != nil {
		return nil, nil, err
	}
	list, err := s.interns(ctx, u)
	if err != nil {
		return nil, nil, err
	}
	a, err := s.store.Activity(ctx, u.ID, ids(list), s.now().AddDate(0, 0, -statsDays))
	return a, u, err
}

// ---------- переписка ----------

// Threads – переписки наставника со своими стажерами, новые сверху.
func (s *Mentor) Threads(ctx context.Context, viewerID string) ([]models.ThreadSummary, error) {
	m, err := s.mentor(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	list, err := s.interns(ctx, m)
	if err != nil {
		return nil, err
	}
	sums, err := s.store.Summaries(ctx, m.ID, ids(list))
	if err != nil {
		return nil, err
	}
	out := make([]models.ThreadSummary, 0, len(list))
	for _, i := range list {
		t := sums[i.ID]
		t.InternID, t.InternName = i.ID, i.Name
		t.StageCode, t.StageTitle = s.stage(ctx, i.ID)
		out = append(out, t)
	}
	for a := 1; a < len(out); a++ {
		for b := a; b > 0 && out[b].LastAt.After(out[b-1].LastAt); b-- {
			out[b], out[b-1] = out[b-1], out[b]
		}
	}
	return out, nil
}

func (s *Mentor) thread(ctx context.Context, intern *models.User, mentor *models.User) (*models.Thread, string, error) {
	conv, err := s.store.Conversation(ctx, intern.ID, mentor.ID)
	if err != nil {
		return nil, "", err
	}
	msgs, err := s.store.ThreadMessages(ctx, conv)
	if err != nil {
		return nil, "", err
	}
	t := &models.Thread{InternID: intern.ID, InternName: intern.Name, Mentor: models.PersonRef{ID: mentor.ID, Name: mentor.Name}, Messages: msgs}
	t.StageCode, t.StageTitle = s.stage(ctx, intern.ID)
	return t, conv, nil
}

// Thread – переписка наставника со стажером.
func (s *Mentor) Thread(ctx context.Context, viewerID, internID string) (*models.Thread, error) {
	m, err := s.mentor(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	i, err := s.myIntern(ctx, m, internID)
	if err != nil {
		return nil, err
	}
	t, _, err := s.thread(ctx, i, m)
	return t, err
}

// ReplyInput – ответ наставника.
type ReplyInput struct {
	Text    string `json:"text"`
	AddToKB bool   `json:"addToKb"`
}

func cleanMessage(text string) (string, error) {
	t := strings.TrimSpace(text)
	if t == "" {
		return "", invalid("Напиши сообщение")
	}
	if utf8.RuneCountInString(t) > 2000 {
		return "", invalid("Сообщение слишком длинное (до 2000 символов)")
	}
	return t, nil
}

// Reply – ответ наставника. Если стажер ждет ответа на переданный AI-buddy вопрос, ответ приходит
// и в чат с AI-buddy; с addToKb ответ сохраняется в справочник и дальше AI-buddy отвечает им сам.
func (s *Mentor) Reply(ctx context.Context, viewerID, internID string, in ReplyInput) (*models.Thread, error) {
	m, err := s.mentor(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	i, err := s.myIntern(ctx, m, internID)
	if err != nil {
		return nil, err
	}
	text, err := cleanMessage(in.Text)
	if err != nil {
		return nil, err
	}
	conv, err := s.store.Conversation(ctx, i.ID, m.ID)
	if err != nil {
		return nil, err
	}
	var article *string
	if in.AddToKB {
		q, err := s.store.LastHandoffQuestion(ctx, conv)
		if err != nil {
			return nil, err
		}
		title := "Ответ наставника"
		if q != "" {
			title += ": " + clip(q, 100)
		}
		var stage *string
		if code, _ := s.stage(ctx, i.ID); code != "" {
			stage, _ = s.store.StageByCode(ctx, i.TeamID, code)
		}
		mat, err := s.store.CreateMaterial(ctx, models.MaterialDraft{
			AuthorID: m.ID, TeamID: m.TeamID, Title: clip(title, 120), Section: "Ответы наставников", Body: text, StageID: stage, UseInAI: true,
		}, "mentor_answer")
		if err != nil {
			return nil, err
		}
		article = &mat.ID
	}
	if err := s.store.Post(ctx, conv, "mentor", m.ID, text); err != nil {
		return nil, err
	}
	if _, err := s.store.AnswerHandoff(ctx, conv, i.ID, m.Name, text, article); err != nil {
		return nil, err
	}
	if err := s.store.MarkRead(ctx, conv, m.ID); err != nil {
		return nil, err
	}
	t, _, err := s.thread(ctx, i, m)
	return t, err
}

// Read – наставник прочитал переписку.
func (s *Mentor) Read(ctx context.Context, viewerID, internID string) error {
	m, err := s.mentor(ctx, viewerID)
	if err != nil {
		return err
	}
	i, err := s.myIntern(ctx, m, internID)
	if err != nil {
		return err
	}
	conv, err := s.store.Conversation(ctx, i.ID, m.ID)
	if err != nil {
		return err
	}
	return s.store.MarkRead(ctx, conv, m.ID)
}

// internSide – стажер и его наставник. Без наставника – 404.
func (s *Mentor) internSide(ctx context.Context, viewerID string) (*models.User, *models.User, error) {
	i, err := s.user(ctx, viewerID)
	if err != nil {
		return nil, nil, err
	}
	if i.Role != models.RoleIntern {
		return nil, nil, ErrForbidden
	}
	if i.MentorID == nil {
		return nil, nil, ErrNotFound
	}
	m, err := s.users.GetByID(ctx, *i.MentorID)
	if err != nil {
		return nil, nil, mapNotFound(err)
	}
	return i, m, nil
}

// MyThread – переписка стажера с наставником.
func (s *Mentor) MyThread(ctx context.Context, viewerID string) (*models.Thread, error) {
	i, m, err := s.internSide(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	t, conv, err := s.thread(ctx, i, m)
	if err != nil {
		return nil, err
	}
	return t, s.store.MarkRead(ctx, conv, i.ID)
}

// SendToMentor – сообщение стажера наставнику.
func (s *Mentor) SendToMentor(ctx context.Context, viewerID, text string) (*models.Thread, error) {
	i, m, err := s.internSide(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	t, err := cleanMessage(text)
	if err != nil {
		return nil, err
	}
	conv, err := s.store.Conversation(ctx, i.ID, m.ID)
	if err != nil {
		return nil, err
	}
	if err := s.store.Post(ctx, conv, "user", i.ID, t); err != nil {
		return nil, err
	}
	return s.MyThread(ctx, viewerID)
}

// Ack – стажер отметил карточку изменения.
func (s *Mentor) Ack(ctx context.Context, viewerID, messageID string) (*models.Thread, error) {
	i, m, err := s.internSide(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	conv, err := s.store.Conversation(ctx, i.ID, m.ID)
	if err != nil {
		return nil, err
	}
	if err := s.store.Ack(ctx, conv, messageID); err != nil {
		return nil, mapNotFound(err)
	}
	return s.MyThread(ctx, viewerID)
}

// notifyInterns – карточка изменения всем стажерам наставника (или одному).
func (s *Mentor) notifyInterns(ctx context.Context, m *models.User, internIDs []string, kind, title, detail string, link *string) {
	for _, id := range internIDs {
		if err := s.store.AddUpdate(ctx, id, m.ID, kind, title, detail, link); err != nil {
			s.log.Warn("update card failed", "err", err)
		}
	}
}

// ---------- материалы ----------

// MaterialInput – форма материала (multipart).
type MaterialInput struct {
	Title, StageCode, Section, URL string
	UseInAI, Notify                bool
	FileName, MimeType             string
	Content                        []byte
}

// Materials – материалы команды.
func (s *Mentor) Materials(ctx context.Context, viewerID string) ([]models.Material, error) {
	u, err := s.user(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	return s.store.Materials(ctx, u.TeamID)
}

// CreateMaterial – файл или ссылка. С useInAi текст попадает в поиск AI-buddy, с notify стажерам
// наставника приходит карточка «новый материал».
func (s *Mentor) CreateMaterial(ctx context.Context, viewerID string, in MaterialInput) (*models.Material, error) {
	u, err := s.staff(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	d := models.MaterialDraft{AuthorID: u.ID, TeamID: u.TeamID, Title: strings.TrimSpace(in.Title), Section: strings.TrimSpace(in.Section), UseInAI: in.UseInAI}
	switch {
	case d.Title == "" || utf8.RuneCountInString(d.Title) > 120:
		return nil, invalid("Введи название (до 120 символов)")
	case utf8.RuneCountInString(d.Section) > 60:
		return nil, invalid("Название раздела слишком длинное (до 60 символов)")
	}
	if d.Section == "" {
		d.Section = "Материалы наставника"
	}
	if code := strings.TrimSpace(in.StageCode); code != "" {
		if d.StageID, err = s.store.StageByCode(ctx, u.TeamID, code); err != nil {
			return nil, invalid("Нет такого этапа")
		}
	}
	source := "upload"
	switch {
	case len(in.Content) > 0:
		if len(in.Content) > maxMaterialBytes {
			return nil, invalid("Файл больше 10 МБ")
		}
		name := strings.ToLower(in.FileName)
		ext := name[strings.LastIndex(name, ".")+1:]
		switch ext {
		case "pdf", "docx", "md", "txt":
		default:
			return nil, invalid("Поддерживаются PDF, DOCX, MD и TXT")
		}
		sum := sha256.Sum256(in.Content)
		d.FileName, d.MimeType, d.Content, d.SHA256 = in.FileName, first(in.MimeType, "application/octet-stream"), in.Content, hex.EncodeToString(sum[:])
		d.Body = d.Title
		if (ext == "md" || ext == "txt") && utf8.Valid(in.Content) {
			d.Body = strings.TrimSpace(string(in.Content))
		}
	case strings.TrimSpace(in.URL) != "":
		u2, err := url.Parse(strings.TrimSpace(in.URL))
		if err != nil || (u2.Scheme != "http" && u2.Scheme != "https") || u2.Host == "" {
			return nil, invalid("Ссылка должна начинаться с http:// или https://")
		}
		source, d.URL, d.Body = "link", u2.String(), d.Title+"\n\nСсылка: "+u2.String()
	default:
		return nil, invalid("Выбери файл или вставь ссылку")
	}
	mat, err := s.store.CreateMaterial(ctx, d, source)
	if err != nil {
		return nil, mapInvalid(err)
	}
	if in.Notify {
		list, err := s.interns(ctx, u)
		if err != nil {
			return nil, err
		}
		link := "/kb"
		s.notifyInterns(ctx, u, ids(list), "material", "Новый материал: "+mat.Title, "в разделе «"+d.Section+"» справочника", &link)
	}
	return mat, nil
}

// DeleteMaterial удаляет материал команды.
func (s *Mentor) DeleteMaterial(ctx context.Context, viewerID, id string) error {
	u, err := s.staff(ctx, viewerID)
	if err != nil {
		return err
	}
	return mapNotFound(s.store.DeleteMaterial(ctx, u.TeamID, id))
}

// ---------- расписание ----------

// MeetingInput – новая встреча наставника.
type MeetingInput struct {
	InternID    string    `json:"internId"` // id стажера или "all"
	Title       string    `json:"title"`
	StartsAt    time.Time `json:"startsAt"`
	DurationMin int       `json:"durationMin"`
	Notify      bool      `json:"notify"`
}

// Meetings – встречи наставника в интервале.
func (s *Mentor) Meetings(ctx context.Context, viewerID, from, to string) ([]models.MentorMeeting, error) {
	m, err := s.mentor(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	f, t, err := parseRange(from, to)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.Meetings(ctx, m.ID, f, t)
	if err != nil {
		return nil, err
	}
	out := make([]models.MentorMeeting, len(rows))
	for i, r := range rows {
		out[i] = r.MentorMeeting
	}
	return out, nil
}

func parseRange(from, to string) (time.Time, time.Time, error) {
	f, err1 := time.Parse(time.RFC3339, from)
	t, err2 := time.Parse(time.RFC3339, to)
	if err1 != nil || err2 != nil || !t.After(f) || t.Sub(f) > 92*24*time.Hour {
		return f, t, invalid("from and to must be ISO dates, to after from, at most 92 days")
	}
	return f, t, nil
}

// whenText – «понедельник, 16 ноября, 15:00».
func whenText(t time.Time) string {
	t = t.In(moscow)
	return fmt.Sprintf("%s, %s, %02d:%02d", weekday(t), dayMonth(t), t.Hour(), t.Minute())
}

// CreateMeeting – встреча со стажером или со всеми стажерами.
func (s *Mentor) CreateMeeting(ctx context.Context, viewerID string, in MeetingInput) (*models.MentorMeeting, error) {
	m, err := s.mentor(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	switch {
	case title == "" || utf8.RuneCountInString(title) > 120:
		return nil, invalid("Введи тему (до 120 символов)")
	case in.StartsAt.IsZero() || in.StartsAt.Before(s.now().Add(-time.Hour)):
		return nil, invalid("Встречу можно назначить только на будущее время")
	case in.DurationMin < 15 || in.DurationMin > 240:
		return nil, invalid("Длительность от 15 до 240 минут")
	}
	var interns []string
	kind := "one_on_one"
	if in.InternID == "all" {
		list, err := s.interns(ctx, m)
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, invalid("У тебя пока нет стажеров")
		}
		interns, kind = ids(list), "team"
	} else {
		i, err := s.myIntern(ctx, m, in.InternID)
		if err != nil {
			return nil, invalid("Стажер не найден")
		}
		interns = []string{i.ID}
		if low := strings.ToLower(title); strings.Contains(low, "ревью") || strings.Contains(low, "разбор") {
			kind = "review"
		}
	}
	row, err := s.store.CreateMeeting(ctx, m.ID, title, kind, in.StartsAt, in.DurationMin, interns)
	if err != nil {
		return nil, mapInvalid(err)
	}
	if in.Notify {
		link := "/calendar"
		s.notifyInterns(ctx, m, interns, "meeting", "Новая встреча: "+title, whenText(in.StartsAt), &link)
	}
	return &row.MentorMeeting, nil
}

// MoveMeeting переносит встречу; стажеры получают карточку.
func (s *Mentor) MoveMeeting(ctx context.Context, viewerID, id string, startsAt *time.Time) (*models.MentorMeeting, error) {
	m, err := s.mentor(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if startsAt == nil || startsAt.Before(s.now().Add(-time.Hour)) {
		return nil, invalid("Встречу можно перенести только на будущее время")
	}
	if err := s.store.MoveMeeting(ctx, m.ID, id, *startsAt); err != nil {
		return nil, mapNotFound(err)
	}
	row, err := s.store.Meeting(ctx, m.ID, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	link := "/calendar"
	s.notifyInterns(ctx, m, row.Participants, "meeting", "Встреча перенесена: "+row.Title, whenText(*startsAt), &link)
	return &row.MentorMeeting, nil
}

// CancelMeeting отменяет встречу; стажеры получают карточку.
func (s *Mentor) CancelMeeting(ctx context.Context, viewerID, id string) error {
	m, err := s.mentor(ctx, viewerID)
	if err != nil {
		return err
	}
	row, err := s.store.Meeting(ctx, m.ID, id)
	if err != nil {
		return mapNotFound(err)
	}
	if err := s.store.CancelMeeting(ctx, m.ID, id); err != nil {
		return mapNotFound(err)
	}
	s.notifyInterns(ctx, m, row.Participants, "meeting", "Встреча отменена: "+row.Title, "была на "+whenText(row.StartsAt), nil)
	return nil
}

// ---------- автоматизации ----------

// Automations – правила AI-buddy наставника.
func (s *Mentor) Automations(ctx context.Context, viewerID string) ([]models.Automation, error) {
	m, err := s.staff(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	set, err := s.store.Automations(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Automation, len(automationInfo))
	for i, a := range automationInfo {
		a.ID, a.Enabled = a.Kind, true
		if e, ok := set[a.Kind]; ok {
			a.Enabled = e
		}
		out[i] = a
	}
	return out, nil
}

// SetAutomation включает или выключает правило.
func (s *Mentor) SetAutomation(ctx context.Context, viewerID, id string, enabled *bool) (*models.Automation, error) {
	m, err := s.staff(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if enabled == nil {
		return nil, invalid("enabled is required")
	}
	for _, a := range automationInfo {
		if a.Kind == id {
			if err := s.store.SetAutomation(ctx, m.ID, id, *enabled); err != nil {
				return nil, err
			}
			a.ID, a.Enabled = a.Kind, *enabled
			return &a, nil
		}
	}
	return nil, ErrNotFound
}

// RunAutomations – один проход правил (воркер вызывает его по расписанию).
func (s *Mentor) RunAutomations(ctx context.Context) (int, error) {
	return s.store.RunAutomations(ctx, s.now())
}

// Run проходит правила каждые interval, пока не отменен ctx.
func (s *Mentor) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n, err := s.RunAutomations(ctx); err != nil {
				s.log.Warn("automations failed", "err", err)
			} else if n > 0 {
				s.log.Info("automations sent", "count", n)
			}
		}
	}
}

// mapInvalid – ошибка ограничений базы как 400.
func mapInvalid(err error) error {
	if errors.Is(err, repository.ErrInvalidReference) {
		return invalid("Проверь поля формы")
	}
	return err
}

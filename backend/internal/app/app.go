// Package app собирает все слои вместе: роутер, middleware, запуск и остановка сервера.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-buddy/backend/internal/config"
	"ai-buddy/backend/internal/database"
	"ai-buddy/backend/internal/handlers"
	"ai-buddy/backend/internal/httpx"
	"ai-buddy/backend/internal/integrations"
	"ai-buddy/backend/internal/llm"
	"ai-buddy/backend/internal/middleware"
	"ai-buddy/backend/internal/mockservices"
	"ai-buddy/backend/internal/models"
	"ai-buddy/backend/internal/repository"
	"ai-buddy/backend/internal/services"
	"ai-buddy/backend/migrations"
)

// AuthService объединяет все, что роутеру нужно от сервиса авторизации.
type AuthService interface {
	handlers.AuthService
	middleware.TokenParser
}

// Services – все сервисы, которые нужны роутеру.
type Services struct {
	Auth     AuthService
	Path     handlers.PathService
	KB       handlers.KBService
	Goals    handlers.GoalsService
	Chat     handlers.ChatService
	People   handlers.PeopleService
	Calendar handlers.CalendarService
	Requests *services.Requests // и страница «Заявки», и синхронизация статусов
	Staff    handlers.StaffService
	Support  *services.Support // обращения в поддержку и синхронизация их статусов
	Mentor   *services.Mentor  // кабинет наставника и воркер автоматизаций
}

// NewServices собирает сервисы поверх базы, модели и системы заявок.
func NewServices(cfg config.Config, pool *pgxpool.Pool, log *slog.Logger) (Services, error) {
	users := repository.NewUsers(pool)
	model, err := llm.New(llm.Config{Provider: cfg.LLMProvider, BaseURL: cfg.LLMBaseURL, APIKey: cfg.LLMAPIKey, Model: cfg.LLMModel, Project: cfg.LLMProject})
	if err != nil {
		return Services{}, err
	}
	var corp integrations.Corporate = mockservices.NewStore(cfg.MockDecisionDelay)
	if cfg.MockServicesURL != "" {
		corp = integrations.NewHTTP(cfg.MockServicesURL, nil)
	}
	reqStore := repository.NewRequests(pool)
	kb := repository.NewKB(pool)
	people := repository.NewPeople(pool)
	staff := repository.NewStaff(pool)
	onboarding := repository.NewOnboarding(pool)
	services.SetJiraURL(cfg.MockServicesPublicURL)
	reqs := services.NewRequests(reqStore, users, corp, log).WithFeed(staff)
	path := services.NewPath(onboarding, users).WithExtras(pathExtras{onboarding, people}).WithFeed(staff, log)
	return Services{
		Auth:     services.NewAuth(users, cfg.JWTSecret, cfg.JWTTTL),
		Path:     path,
		KB:       services.NewKB(kb, users, people),
		Goals:    services.NewGoals(repository.NewGoals(pool), users).WithFeed(staff, log),
		Chat:     services.NewAgent(repository.NewChat(pool), reqStore, kb, users, people, corp, model, cfg.LLMTimeout, log).WithPath(path),
		People:   services.NewPeople(people),
		Calendar: services.NewCalendar(repository.NewCalendar(pool), users, people),
		Requests: reqs,
		Staff:    services.NewStaff(staff, reqStore, users, corp, log),
		Support:  services.NewSupport(staff, users, corp, log),
		Mentor:   services.NewMentor(repository.NewMentor(pool), users, onboarding, path, log),
	}, nil
}

// pathExtras – подробности задач и люди вокруг стажера для «Моего пути».
type pathExtras struct {
	*repository.Onboarding
	people *repository.People
}

func (p pathExtras) Circle(ctx context.Context, userID string) (*models.Circle, error) {
	return p.people.Circle(ctx, userID)
}

// NewRouter описывает все маршруты API.
func NewRouter(cfg config.Config, svc Services, log *slog.Logger) http.Handler {
	auth := svc.Auth
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(chimw.Timeout(30 * time.Second)) // ответ модели может идти дольше простых запросов
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.CORSOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:         300,
	}))
	r.NotFound(httpx.NotFound)
	r.MethodNotAllowed(httpx.MethodNotAllowed)

	authH := handlers.NewAuth(auth, cfg.DemoMode, log)
	pathH := handlers.NewPath(svc.Path, log)
	kbH := handlers.NewKB(svc.KB, log)
	goalsH := handlers.NewGoals(svc.Goals, log)
	chatH := handlers.NewChat(svc.Chat, svc.Requests, log)
	peopleH := handlers.NewPeople(svc.People, log)
	calendarH := handlers.NewCalendar(svc.Calendar, log)
	staffH := handlers.NewStaff(svc.Staff, svc.Support, log)
	mentorH := handlers.NewMentor(svc.Mentor, log)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handlers.Health)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authH.Login)
			r.Get("/test-users", authH.TestUsers)

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(auth))
				r.Get("/me", authH.Me)
			})
		})

		// Все остальное – только с токеном.
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(auth))

			// Страница «Мой путь» (стажер).
			r.Get("/path", pathH.MyPath)
			r.Post("/path/tasks/{taskId}/complete", pathH.CompleteTask)

			// Стажеры глазами наставника и руководителя.
			r.Get("/interns", pathH.ListInterns)
			r.Get("/interns/{internId}/path", pathH.InternPath)
			r.Put("/interns/{internId}/tasks/{taskId}", pathH.SetTaskStatus)
			r.Post("/interns/{internId}/meetings", pathH.CreateMeeting)

			// Цели стажировки.
			r.Get("/goals", goalsH.Mine)
			r.Patch("/goals/{goalId}", goalsH.Update)
			r.Delete("/goals/{goalId}", goalsH.Delete)
			r.Post("/goals/{goalId}/progress", goalsH.Progress)
			r.Post("/goals/{goalId}/steps/{stepId}", goalsH.Step)
			r.Get("/interns/{internId}/goals", goalsH.OfIntern)
			r.Post("/interns/{internId}/goals", goalsH.Create)
			r.Post("/interns/{internId}/goals/agree", goalsH.Agree)

			// Чат с помощником и заявки.
			r.Get("/chat/messages", chatH.History)
			r.Post("/chat/messages", chatH.Send)
			r.Post("/chat/reset", chatH.Reset)
			r.Get("/chat/dialogs", chatH.Dialogs)
			r.Get("/chat/dialogs/{dialogId}/messages", chatH.DialogMessages)
			r.Post("/chat/actions/{actionId}/confirm", chatH.Confirm)
			r.Post("/chat/actions/{actionId}/cancel", chatH.Cancel)
			r.Get("/requests", chatH.Requests)
			r.Post("/requests/{id}/withdraw", chatH.Withdraw)

			// Свои люди (наставник, руководитель) и отделы – для «Поддержки».
			r.Get("/me/contacts", peopleH.Contacts)

			// Календарь стажера: встречи из кабинета наставника и свои записи.
			r.Get("/me/calendar", calendarH.Events)
			r.Post("/me/calendar/entries", calendarH.CreateEntry)
			r.Delete("/me/calendar/entries/{id}", calendarH.DeleteEntry)
			r.Post("/me/calendar/events/{meetingId}/answer", calendarH.Answer)

			// Конспект встреч стажера.
			r.Get("/me/notes", staffH.Notes)
			r.Put("/me/notes/{eventId}", staffH.SaveNote)

			// Обращения в поддержку (любая роль).
			r.Post("/support/tickets", staffH.CreateTicket)
			r.Get("/support/tickets", staffH.Tickets)

			// Согласования: заявки стажеров, где пользователь – согласующий.
			r.Get("/approvals", staffH.Approvals)
			r.Post("/approvals/{id}", staffH.Decide)

			// Кабинет наставника: лента (колокольчик) и чек-лист по стажеру.
			r.Get("/mentor/feed", staffH.Feed)
			r.Post("/mentor/feed/read", staffH.ReadFeed)
			r.Get("/mentor/interns/{internId}/checklist", staffH.Checklist)
			r.Post("/mentor/interns/{internId}/checklist/{itemId}", staffH.MarkChecklist)

			// Кабинет наставника: статистика, переписка, материалы, расписание, автоматизации;
			// переписка стажера с наставником.
			mentorH.Routes(r)

			// Справочник.
			r.Get("/kb", kbH.Get)
			r.Get("/kb/search", kbH.Search)
			r.Get("/kb/articles/{articleId}", kbH.Article)

			// Консоль наставника: шаблоны адаптации.
			r.Route("/console", func(r chi.Router) {
				r.Get("/templates", pathH.Templates)
				r.Get("/action-kinds", pathH.ActionKinds)
				r.Post("/stages", pathH.CreateStage)
				r.Put("/stages/{stageId}", pathH.UpdateStage)
				r.Delete("/stages/{stageId}", pathH.DeleteStage)
				r.Post("/stages/{stageId}/tasks", pathH.CreateTask)
				r.Post("/stages/{stageId}/resources", pathH.CreateResource)
				r.Put("/tasks/{taskId}", pathH.UpdateTask)
				r.Delete("/tasks/{taskId}", pathH.DeleteTask)
				r.Delete("/resources/{resourceId}", pathH.DeleteResource)
			})
		})
	})
	return r
}

// Run подключается к базе, применяет миграции и запускает HTTP-сервер до отмены ctx.
// Если ready не nil, в него отправляется адрес, на котором сервер начал слушать.
func Run(ctx context.Context, cfg config.Config, log *slog.Logger, ready chan<- string) error {
	pool, err := database.Connect(ctx, cfg.DatabaseURL, 30*time.Second)
	if err != nil {
		return err
	}
	defer pool.Close()

	applied, err := database.Migrate(ctx, pool, migrations.FS)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Info("migrations done", "applied", applied)

	ln, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	svc, err := NewServices(cfg, pool, log)
	if err != nil {
		return err
	}
	syncCtx, stopSync := context.WithCancel(ctx)
	defer stopSync()
	if cfg.RequestSync > 0 {
		go svc.Requests.Run(syncCtx, cfg.RequestSync)
		go svc.Support.Run(syncCtx, cfg.RequestSync)
		go svc.Mentor.Run(syncCtx, time.Minute)
	}
	log.Info("assistant ready", "llm", cfg.LLMProvider, "requestService", withDefault(cfg.MockServicesURL, "in-process mock"))

	srv := &http.Server{
		Handler:           NewRouter(cfg, svc, log),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()
	log.Info("backend started", "addr", ln.Addr().String(), "demoMode", cfg.DemoMode)
	if ready != nil {
		ready <- ln.Addr().String()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("backend stopped")
	return nil
}

func withDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

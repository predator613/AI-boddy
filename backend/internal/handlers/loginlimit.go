package handlers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// loginLimiter ограничивает подбор пароля: после maxFails неудачных попыток для пары
// «адрес + логин» вход блокируется на window. Хранится в памяти процесса – для прототипа
// достаточно; в продакшене счетчик живет в Redis, общий для всех экземпляров.
type loginLimiter struct {
	mu       sync.Mutex
	fails    map[string]*loginFails
	maxFails int
	window   time.Duration
	now      func() time.Time
}

type loginFails struct {
	count int
	first time.Time
}

func newLoginLimiter(maxFails int, window time.Duration) *loginLimiter {
	return &loginLimiter{fails: map[string]*loginFails{}, maxFails: maxFails, window: window, now: time.Now}
}

func loginKey(r *http.Request, login string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return host + "|" + strings.ToLower(login)
}

// blocked – сколько ждать, если попыток слишком много; 0 – можно входить.
func (l *loginLimiter) blocked(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, ok := l.fails[key]
	if !ok {
		return 0
	}
	left := f.first.Add(l.window).Sub(l.now())
	if left <= 0 {
		delete(l.fails, key)
		return 0
	}
	if f.count >= l.maxFails {
		return left
	}
	return 0
}

func (l *loginLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, ok := l.fails[key]
	if !ok || l.now().Sub(f.first) > l.window {
		l.fails[key] = &loginFails{count: 1, first: l.now()}
		return
	}
	f.count++
}

func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, key)
}

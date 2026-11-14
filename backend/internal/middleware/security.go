package middleware

import "net/http"

// SecurityHeaders – базовые заголовки защиты для ответов API: браузер не угадывает тип содержимого,
// страницу нельзя встроить во фрейм, адрес не утекает в Referer, ответы с данными не кешируются.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

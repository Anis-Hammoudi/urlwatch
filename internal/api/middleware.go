package api

import (
	"log/slog"
	"net/http"
	"time"
)

// responseRecorder intercepts the status code so we can log it.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// LoggingMiddleware logs every request using slog.
func LoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ignore healthz to prevent log pollution
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		attrs := []any{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Int("duration_ms", int(time.Since(start).Milliseconds())),
		}

		if batchID := rec.Header().Get("X-Batch-ID"); batchID != "" {
			attrs = append(attrs, slog.String("batch_id", batchID))
		}

		logger.Info("HTTP Request", attrs...)
	})
}

// RecoveryMiddleware catches panics and returns a clean 500 response.
func RecoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic recovered", slog.Any("error", err))
				http.Error(w, `{"error": {"code": "internal_error", "message": "serveur en erreur"}}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

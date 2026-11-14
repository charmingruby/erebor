package httpx

import (
	"context"
	"errors"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/charmingruby/erebor/o11y/logging"
)

// Server wraps an HTTP server with graceful shutdown support.
type Server struct {
	http.Server
}

// NewServer creates a server and router with logging, recovery, and health probes.
func NewServer(port string) (*Server, chi.Router) {
	addr := ":" + port

	r := chi.NewRouter()
	r.Use(
		withRecovery,
		withLogging,
	)

	registerProbes(r)

	return &Server{
		Server: http.Server{
			WriteTimeout: 10 * time.Second,
			ReadTimeout:  5 * time.Second,
			IdleTimeout:  120 * time.Second,
			Addr:         addr,
			Handler:      r,
		},
	}, r
}

// Start listens for HTTP requests until the server is closed.
func (s *Server) Start() error {
	if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Close gracefully shuts down the server.
func (s *Server) Close(ctx context.Context) error {
	return s.Shutdown(ctx)
}

func registerProbes(r chi.Router) {
	r.Get("/livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logging.Log.With(
			"path", r.URL.Path,
			"method", r.Method,
		)

		ctx := logging.WithLogger(r.Context(), log)

		log.InfoContext(ctx, "request started")
		defer log.InfoContext(ctx, "request finished")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			logging.LoggerFromContext(r.Context()).Error("panic recovered",
				"panic", recovered,
				"stack", string(debug.Stack()),
			)

			writeResponse(w, http.StatusInternalServerError, ErrorResponse{
				Message: "Internal Server Error",
			})
		}()

		next.ServeHTTP(w, r)
	})
}

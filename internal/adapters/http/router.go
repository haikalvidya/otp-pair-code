package httpadapter

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "otp-pair-code-interview/docs"
)

func NewRouter(handler *Handler, logger zerolog.Logger, requestTimeout time.Duration) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(RequestLoggingMiddleware(logger))
	r.Use(chimiddleware.Recoverer)
	if requestTimeout > 0 {
		r.Use(chimiddleware.Timeout(requestTimeout))
	}

	r.Get("/healthz", handler.Healthz)
	r.Route("/otp", func(r chi.Router) {
		r.Post("/request", handler.RequestOTP)
		r.Post("/validate", handler.ValidateOTP)
	})
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	return r
}

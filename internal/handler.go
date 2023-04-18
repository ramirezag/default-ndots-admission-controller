package internal

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"time"
)

func NewHandlers(ndotsValue int, requestTimeout time.Duration) *chi.Mux {
	defaultNDotsAdmitHandler := NewDefaultNDotsAdmitHandler(ndotsValue)
	// Create a chi Router
	r := chi.NewRouter()
	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(requestTimeout))
	r.Get("/healthz", health)
	r.Post("/webhook", func(w http.ResponseWriter, r *http.Request) {
		serve(w, r, newDelegateToV1AdmitHandler(defaultNDotsAdmitHandler.admitHander))
	})
	return r
}

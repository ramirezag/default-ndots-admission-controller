package internal

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	chiTrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-chi/chi.v5"
	"net/http"
	"time"
)

func NewHandlers(ndotsValue int) *chi.Mux {
	defaultNDotsAdmitHandler := NewDefaultNDotsAdmitHandler(ndotsValue)
	// Create a chi Router
	r := chi.NewRouter()
	// Use the tracer middleware with the default service name "chi.router".
	r.Use(chiTrace.Middleware())
	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(1 * time.Minute))
	r.Get("/healthz", health)
	r.Post("/webhook", func(w http.ResponseWriter, r *http.Request) {
		serve(w, r, newDelegateToV1AdmitHandler(defaultNDotsAdmitHandler.admitHander))
	})
	return r
}

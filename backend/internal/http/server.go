package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/raisin/travel-split/backend/internal/expense"
	"github.com/raisin/travel-split/backend/internal/settlement"
	"github.com/raisin/travel-split/backend/internal/trip"
)

// Pinger is the subset of the DB pool the health check needs.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Deps struct {
	DB          Pinger
	Trips       *trip.Service
	Expenses    *expense.Service
	Settlements *settlement.Service
	CORSOrigins []string
}

type Server struct {
	Deps
}

func NewServer(d Deps) *Server { return &Server{Deps: d} }

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: s.CORSOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "X-Member-Id"},
		MaxAge:         300,
	}))

	r.Get("/healthz", s.healthz)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", s.healthz)
		r.Post("/trips", s.createTrip)

		// {trip} is an invite code on GET /trips/{trip} and a trip UUID below it.
		r.Route("/trips/{trip}", func(r chi.Router) {
			r.Get("/", s.getTripByInvite)

			r.Group(func(r chi.Router) {
				r.Use(s.loadTrip)
				// Open to anyone holding the trip ID: a newcomer must see
				// the member list (and add themselves) before having an ID.
				r.Get("/members", s.listMembers)
				r.Post("/members", s.addMember)

				r.Group(func(r chi.Router) {
					r.Use(s.requireMember)
					r.Get("/expenses", s.listExpenses)
					r.Post("/expenses", s.createExpense)
					r.Get("/expenses/{expense}", s.getExpense)
					r.Put("/expenses/{expense}", s.updateExpense)
					r.Delete("/expenses/{expense}", s.deleteExpense)
					r.Get("/balances", s.balances)
					r.Get("/settlement", s.settlementPlan)
					r.Post("/settlements/paid", s.markPaid)
				})
			})
		})
	})
	return r
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.DB.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database is not reachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

package http

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/raisin/travel-split/backend/internal/domain/auth"
	"github.com/raisin/travel-split/backend/internal/domain/expense"
	"github.com/raisin/travel-split/backend/internal/domain/fx"
	"github.com/raisin/travel-split/backend/internal/domain/settlement"
	"github.com/raisin/travel-split/backend/internal/domain/slip"
	"github.com/raisin/travel-split/backend/internal/domain/trip"
)

// Pinger is the subset of the DB pool the health check needs.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Deps struct {
	DB          Pinger
	Auth        *auth.Service
	Passkeys    *auth.Passkeys // nil disables passkey sign-in
	AuthCfg     AuthConfig
	Trips       *trip.Service
	Expenses    *expense.Service
	Settlements *settlement.Service
	Slips       *slip.Service // nil when no AI provider is configured
	FX          *fx.Service
	CORSOrigins []string
	StaticDir   string // built web app to serve (production); empty in dev, where Vite serves it
	// TrustProxy: behind Cloudflare, take the client IP from CF-Connecting-IP
	// (for rate limits). Off when clients connect directly.
	TrustProxy bool
}

type Server struct {
	Deps
	lim         *limits
	logIPSource sync.Once
}

func NewServer(d Deps) *Server { return &Server{Deps: d, lim: defaultLimits()} }

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLog)
	r.Use(middleware.Recoverer)
	// Production serves the app and API from one origin and needs no CORS;
	// it's only for a frontend on another origin during development.
	if len(s.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: s.CORSOrigins,
			AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", CSRFHeader},
			MaxAge:         300,
		}))
	}
	byIP := s.clientIP

	r.Get("/healthz", s.healthz)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(apiHeaders, csrf)
		r.Get("/healthz", s.healthz)

		r.Route("/auth", func(r chi.Router) {
			r.Get("/config", s.authConfig)
			r.Post("/logout", s.logout)
			// Sign-in steps anyone can call: limited per IP so nobody can
			// flood the DB with challenges or accounts.
			r.Group(func(r chi.Router) {
				r.Use(s.limit(s.lim.authIP, byIP))
				r.Get("/google/start", s.googleStart)
				r.Get("/google/callback", s.googleCallback)
				r.Post("/mock", s.mockLogin)
				r.With(s.limit(s.lim.signupIP, byIP)).Post("/passkey/signup/begin", s.passkeySignupBegin)
				r.Post("/passkey/signup/finish", s.passkeySignupFinish)
				r.Post("/passkey/login/begin", s.passkeyLoginBegin)
				r.Post("/passkey/login/finish", s.passkeyLoginFinish)
			})
			r.Group(func(r chi.Router) {
				r.Use(s.requireUser)
				r.Use(s.writesOnly(s.lim.writes, byUser))
				r.Get("/me", s.authMe)
				r.Post("/claim", s.claimDevices)
				r.Post("/passkey/register/begin", s.passkeyRegisterBegin)
				r.Post("/passkey/register/finish", s.passkeyRegisterFinish)
				r.Get("/passkeys", s.listPasskeys)
				r.Delete("/passkeys/{id}", s.deletePasskey)
			})
		})

		// Everything else needs a signed-in user.
		r.Group(func(r chi.Router) {
			r.Use(s.requireUser)
			r.Use(s.writesOnly(s.lim.writes, byUser))
			r.Get("/me/trips", s.myTrips)
			r.With(s.limit(s.lim.newTrips, byUser)).Post("/trips", s.createTrip)
			r.Get("/fx", s.fxRate)
			r.Get("/fx/currencies", s.fxCurrencies)

			// {trip} is an invite code on GET /trips/{trip} and a trip UUID below it.
			r.Route("/trips/{trip}", func(r chi.Router) {
				r.Get("/", s.getTripByInvite)

				r.Group(func(r chi.Router) {
					r.Use(s.loadTrip)
					// Anyone holding the trip ID may ask to join; nothing else.
					r.With(s.limit(s.lim.newTrips, byUser)).Post("/join", s.requestJoin)

					r.Group(func(r chi.Router) {
						r.Use(s.requireDevice)
						r.Get("/me", s.me) // pending devices poll this
						r.Delete("/me", s.leave)

						r.Group(func(r chi.Router) {
							r.Use(s.requireMember)
							r.Get("/members", s.listMembers)
							r.Get("/expenses", s.listExpenses)
							r.Post("/expenses", s.createExpense)
							r.Get("/expenses/{expense}", s.getExpense)
							r.Put("/expenses/{expense}", s.updateExpense)
							r.Delete("/expenses/{expense}", s.deleteExpense)
							r.Get("/balances", s.balances)
							r.Get("/settlement", s.settlementPlan)
							r.Post("/settlements/paid", s.markPaid)
							r.With(s.limit(s.lim.slips, byUser)).Post("/slips", s.uploadSlips)
							r.Get("/summary", s.summary)
							r.Get("/activity", s.activity)
							r.Patch("/me", s.updateMe)

							r.Group(func(r chi.Router) {
								r.Use(s.requireOwner)
								r.Patch("/", s.updateTrip)
								r.Get("/join-requests", s.listJoinRequests)
								r.Post("/join-requests/{request}/approve", s.approveJoin)
								r.Post("/join-requests/{request}/reject", s.rejectJoin)
								r.Post("/join-requests/reject-all", s.rejectAllJoins)
							})
						})
					})
				})
			})
		})
	})
	if s.StaticDir != "" {
		r.NotFound(spa(s.StaticDir, s.AuthCfg.secure()).ServeHTTP)
		r.MethodNotAllowed(spa(s.StaticDir, s.AuthCfg.secure()).ServeHTTP)
	}
	return r
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.DB.Ping(ctx); err != nil {
		// pgx error text never includes the password.
		slog.WarnContext(r.Context(), "health check: database unreachable", "err", err)
		writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database is not reachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

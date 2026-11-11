package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/raisin/travel-split/backend/internal/apperr"
	"github.com/raisin/travel-split/backend/internal/db"
	"github.com/raisin/travel-split/backend/internal/trip"
)

type ctxKey int

const (
	tripKey ctxKey = iota
	memberKey
)

func tripFrom(ctx context.Context) db.Trip     { return ctx.Value(tripKey).(db.Trip) }
func memberFrom(ctx context.Context) db.Member { return ctx.Value(memberKey).(db.Member) }

// loadTrip resolves {trip} as a trip UUID and stores the trip in context.
func (s *Server) loadTrip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "trip"))
		if err != nil {
			writeErr(w, r, trip.ErrTripNotFound)
			return
		}
		t, err := s.Trips.Get(r.Context(), id)
		if err != nil {
			writeErr(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tripKey, t)))
	})
}

// requireMember checks that X-Member-Id names a member of the loaded trip.
func (s *Server) requireMember(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(r.Header.Get("X-Member-Id"))
		if raw == "" {
			writeErr(w, r, apperr.Unauthorized("member_required", "ต้องระบุตัวตนด้วย X-Member-Id"))
			return
		}
		notMember := apperr.Forbidden("not_a_member", "คุณไม่ได้เป็นสมาชิกของทริปนี้")
		id, err := uuid.Parse(raw)
		if err != nil {
			writeErr(w, r, notMember)
			return
		}
		t := tripFrom(r.Context())
		m, err := s.Trips.Member(r.Context(), t.ID, id)
		if errors.Is(err, trip.ErrMemberNotFound) {
			writeErr(w, r, notMember)
			return
		}
		if err != nil {
			writeErr(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), memberKey, m)))
	})
}

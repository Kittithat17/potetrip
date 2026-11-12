package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/raisin/travel-split/backend/internal/apperr"
	"github.com/raisin/travel-split/backend/internal/db"
	"github.com/raisin/travel-split/backend/internal/trip"
)

// TokenHeader carries the device's secret token for the trip in the URL.
const TokenHeader = "X-Device-Token"

type ctxKey int

const (
	tripKey ctxKey = iota
	deviceKey
	memberKey
)

func tripFrom(ctx context.Context) db.Trip     { return ctx.Value(tripKey).(db.Trip) }
func deviceFrom(ctx context.Context) db.Device { return ctx.Value(deviceKey).(db.Device) }
func memberFrom(ctx context.Context) db.Member { return ctx.Value(memberKey).(db.Member) }
func with(r *http.Request, k ctxKey, v any) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), k, v))
}

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
		next.ServeHTTP(w, with(r, tripKey, t))
	})
}

// requireDevice accepts any device registered to this trip, approved or not.
func (s *Server) requireDevice(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.Header.Get(TokenHeader))
		if token == "" {
			writeErr(w, r, apperr.Unauthorized("token_required", "ต้องเข้าร่วมทริปก่อน"))
			return
		}
		d, err := s.Trips.Device(r.Context(), tripFrom(r.Context()).ID, token)
		if err != nil {
			writeErr(w, r, err)
			return
		}
		next.ServeHTTP(w, with(r, deviceKey, d))
	})
}

// requireMember lets only approved devices through, as their member.
func (s *Server) requireMember(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := deviceFrom(r.Context())
		switch {
		case d.Status == trip.StatusPending:
			writeErr(w, r, apperr.Forbidden("pending_approval", "รอเจ้าของทริปอนุมัติก่อน"))
			return
		case d.Status != trip.StatusApproved || d.MemberID == nil:
			writeErr(w, r, apperr.Forbidden("join_rejected", "คำขอเข้าทริปถูกปฏิเสธ"))
			return
		}
		m, err := s.Trips.Member(r.Context(), d.TripID, *d.MemberID)
		if err != nil {
			writeErr(w, r, err)
			return
		}
		next.ServeHTTP(w, with(r, memberKey, m))
	})
}

func (s *Server) requireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := tripFrom(r.Context())
		if t.OwnerMemberID == nil || *t.OwnerMemberID != memberFrom(r.Context()).ID {
			writeErr(w, r, apperr.Forbidden("owner_only", "เฉพาะเจ้าของทริปเท่านั้น"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

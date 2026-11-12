package http

import (
	"net/http"
	"time"

	"github.com/raisin/travel-split/backend/internal/apperr"
)

// fxRate: GET /fx?from=CNY&to=THB[&date=2026-11-12]. Public: rates aren't trip data.
func (s *Server) fxRate(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var date *time.Time
	if d := q.Get("date"); d != "" {
		t, err := time.Parse(time.DateOnly, d)
		if err != nil {
			writeErr(w, r, apperr.Invalid("invalid_date", "date ต้องอยู่ในรูปแบบ YYYY-MM-DD"))
			return
		}
		date = &t
	}
	quote, err := s.FX.Rate(r.Context(), q.Get("from"), q.Get("to"), date)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, quote)
}

func (s *Server) fxCurrencies(w http.ResponseWriter, r *http.Request) {
	list, err := s.FX.Currencies(r.Context())
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"currencies": list})
}

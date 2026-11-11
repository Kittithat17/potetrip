package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/raisin/travel-split/backend/internal/apperr"
	"github.com/raisin/travel-split/backend/internal/expense"
)

// ---- trips & members ----

func (s *Server) createTrip(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		BaseCurrency string `json:"base_currency"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	t, err := s.Trips.Create(r.Context(), req.Name, req.BaseCurrency)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) getTripByInvite(w http.ResponseWriter, r *http.Request) {
	t, err := s.Trips.GetByInviteCode(r.Context(), chi.URLParam(r, "trip"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	ms, err := s.Trips.ListMembers(r.Context(), tripFrom(r.Context()).ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ms)
}

func (s *Server) addMember(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string  `json:"name"`
		PromptPay *string `json:"promptpay"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	m, err := s.Trips.AddMember(r.Context(), tripFrom(r.Context()).ID, req.Name, req.PromptPay)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// ---- expenses ----

func (s *Server) listExpenses(w http.ResponseWriter, r *http.Request) {
	var f expense.Filter
	if c := strings.TrimSpace(r.URL.Query().Get("category")); c != "" {
		f.Category = &c
	}
	if m := strings.TrimSpace(r.URL.Query().Get("member")); m != "" {
		id, err := uuid.Parse(m)
		if err != nil {
			writeErr(w, r, apperr.Invalid("invalid_member", "member ต้องเป็น UUID"))
			return
		}
		f.MemberID = &id
	}
	es, err := s.Expenses.List(r.Context(), tripFrom(r.Context()).ID, f)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, es)
}

func (s *Server) createExpense(w http.ResponseWriter, r *http.Request) {
	var in expense.Input
	if err := decodeJSON(w, r, &in); err != nil {
		writeErr(w, r, err)
		return
	}
	ctx := r.Context()
	e, err := s.Expenses.Create(ctx, tripFrom(ctx), memberFrom(ctx).ID, in)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) getExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := expenseID(w, r)
	if !ok {
		return
	}
	e, err := s.Expenses.Get(r.Context(), tripFrom(r.Context()).ID, id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) updateExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := expenseID(w, r)
	if !ok {
		return
	}
	var in expense.Input
	if err := decodeJSON(w, r, &in); err != nil {
		writeErr(w, r, err)
		return
	}
	e, err := s.Expenses.Update(r.Context(), tripFrom(r.Context()), id, in)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) deleteExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := expenseID(w, r)
	if !ok {
		return
	}
	if err := s.Expenses.Delete(r.Context(), tripFrom(r.Context()).ID, id); err != nil {
		writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func expenseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "expense"))
	if err != nil {
		writeErr(w, r, expense.ErrNotFound)
		return uuid.Nil, false
	}
	return id, true
}

// ---- balances & settlement ----

func (s *Server) balances(w http.ResponseWriter, r *http.Request) {
	t := tripFrom(r.Context())
	bs, err := s.Settlements.Balances(r.Context(), t.ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"currency": t.BaseCurrency, "balances": bs})
}

func (s *Server) settlementPlan(w http.ResponseWriter, r *http.Request) {
	t := tripFrom(r.Context())
	ts, err := s.Settlements.Plan(r.Context(), t.ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"currency": t.BaseCurrency, "transfers": ts})
}

func (s *Server) markPaid(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From       uuid.UUID `json:"from"`
		To         uuid.UUID `json:"to"`
		AmountBase int64     `json:"amount_base"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	p, err := s.Settlements.MarkPaid(r.Context(), tripFrom(r.Context()).ID, req.From, req.To, req.AmountBase)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

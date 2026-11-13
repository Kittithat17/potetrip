package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/raisin/travel-split/backend/internal/domain/expense"
	"github.com/raisin/travel-split/backend/internal/domain/settlement"
	"github.com/raisin/travel-split/backend/internal/domain/trip"
	"github.com/raisin/travel-split/backend/internal/platform/apperr"
	"github.com/raisin/travel-split/backend/internal/platform/db"
)

// ---- trips, members & joining ----

func (s *Server) createTrip(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string  `json:"name"`
		BaseCurrency string  `json:"base_currency"`
		CreatorName  string  `json:"creator_name"`
		PromptPay    *string `json:"promptpay"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	u := userFrom(r.Context())
	if strings.TrimSpace(req.CreatorName) == "" {
		req.CreatorName = u.Name
	}
	out, err := s.Trips.Create(r.Context(), u.ID, req.Name, req.BaseCurrency, req.CreatorName, req.PromptPay)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// publicMember is what anyone with the invite code sees: enough to say
// "that's me" when joining, nothing more (no PromptPay numbers).
type publicMember struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Claimable bool      `json:"claimable"` // false once someone has signed in as them
}

func (s *Server) getTripByInvite(w http.ResponseWriter, r *http.Request) {
	t, err := s.Trips.GetByInviteCode(r.Context(), chi.URLParam(r, "trip"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	ms, err := s.Trips.PublicMembers(r.Context(), t.ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	names := make([]publicMember, len(ms))
	for i, m := range ms {
		names[i] = publicMember{ID: m.ID, Name: m.Name, Claimable: !m.Linked}
	}
	writeJSON(w, http.StatusOK, map[string]any{"trip": t, "members": names})
}

func (s *Server) requestJoin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string     `json:"name"`
		MemberID  *uuid.UUID `json:"member_id"` // "I'm already in this trip"
		PromptPay *string    `json:"promptpay"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	u := userFrom(r.Context())
	if req.MemberID == nil && strings.TrimSpace(req.Name) == "" {
		req.Name = u.Name
	}
	d, err := s.Trips.RequestJoin(r.Context(), u.ID, tripFrom(r.Context()).ID, req.Name, req.MemberID, req.PromptPay)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"request": requestView(d)})
}

// me tells the user where they stand in the trip: pending, rejected, or approved (with its member).
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	d, t := deviceFrom(ctx), tripFrom(ctx)
	out := map[string]any{"status": d.Status, "request": requestView(d), "member": nil, "is_owner": false}
	if d.Status == trip.StatusApproved && d.MemberID != nil {
		m, err := s.Trips.Member(ctx, t.ID, *d.MemberID)
		if err != nil {
			writeErr(w, r, err)
			return
		}
		out["member"] = m
		out["is_owner"] = t.OwnerMemberID != nil && *t.OwnerMemberID == m.ID
	}
	writeJSON(w, http.StatusOK, out)
}

// leave drops the user's membership or cancels their pending request.
func (s *Server) leave(w http.ResponseWriter, r *http.Request) {
	if err := s.Trips.Leave(r.Context(), tripFrom(r.Context()), deviceFrom(r.Context())); err != nil {
		writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateTrip(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         *string `json:"name"`
		BaseCurrency *string `json:"base_currency"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	t, err := s.Trips.Update(r.Context(), tripFrom(r.Context()).ID, req.Name, req.BaseCurrency)
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

type joinRequest struct {
	ID                uuid.UUID  `json:"id"`
	Status            string     `json:"status"`
	RequestedName     string     `json:"requested_name"`
	RequestedMemberID *uuid.UUID `json:"requested_member_id"` // set when claiming an existing name
	CreatedAt         time.Time  `json:"created_at"`
	// Who is asking (owner-only list): lets the owner check it's really their friend.
	Account *requestAccount `json:"account,omitempty"`
}

type requestAccount struct {
	Name      string  `json:"name"`
	Email     *string `json:"email"` // nil for a passkey-only account
	HasGoogle bool    `json:"has_google"`
}

func requestView(d db.Device) joinRequest {
	return joinRequest{ID: d.ID, Status: d.Status, RequestedName: d.RequestedName, RequestedMemberID: d.RequestedMemberID, CreatedAt: d.CreatedAt}
}

func (s *Server) listJoinRequests(w http.ResponseWriter, r *http.Request) {
	ds, err := s.Trips.PendingRequests(r.Context(), tripFrom(r.Context()).ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]joinRequest, len(ds))
	for i, d := range ds {
		out[i] = requestView(d.Device)
		if d.AccountName != nil {
			out[i].Account = &requestAccount{Name: *d.AccountName, Email: d.AccountEmail, HasGoogle: d.AccountHasGoogle}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) rejectAllJoins(w http.ResponseWriter, r *http.Request) {
	n, err := s.Trips.RejectAll(r.Context(), tripFrom(r.Context()).ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"rejected": n})
}

func (s *Server) approveJoin(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "request"))
	if err != nil {
		writeErr(w, r, trip.ErrRequestNotFound)
		return
	}
	m, err := s.Trips.Approve(r.Context(), tripFrom(r.Context()).ID, id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) rejectJoin(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "request"))
	if err != nil {
		writeErr(w, r, trip.ErrRequestNotFound)
		return
	}
	if err := s.Trips.Reject(r.Context(), tripFrom(r.Context()).ID, id); err != nil {
		writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	e, err := s.Expenses.Create(ctx, tripFrom(ctx), expenseActor(ctx), in)
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
	e, err := s.Expenses.Update(r.Context(), tripFrom(r.Context()), expenseActor(r.Context()), id, in)
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
	if err := s.Expenses.Delete(r.Context(), tripFrom(r.Context()).ID, expenseActor(r.Context()), id); err != nil {
		writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isOwner(ctx context.Context) bool {
	t := tripFrom(ctx)
	return t.OwnerMemberID != nil && *t.OwnerMemberID == memberFrom(ctx).ID
}

func expenseActor(ctx context.Context) expense.Actor {
	return expense.Actor{MemberID: memberFrom(ctx).ID}
}

// activity is the trip's change history (who added, edited, deleted what).
func (s *Server) activity(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Expenses.Activity(r.Context(), tripFrom(r.Context()).ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
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
	ctx := r.Context()
	p, err := s.Settlements.MarkPaid(ctx, tripFrom(ctx).ID, settlement.Actor{MemberID: memberFrom(ctx).ID, IsOwner: isOwner(ctx)}, req.From, req.To, req.AmountBase)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t := tripFrom(ctx)
	sum, err := s.Expenses.Summary(ctx, t.ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	bs, err := s.Settlements.Balances(ctx, t.ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	type memberRow struct {
		MemberID uuid.UUID `json:"member_id"`
		Paid     int64     `json:"paid"`     // what they paid for the group
		Consumed int64     `json:"consumed"` // their own share of the spending
	}
	members := make([]memberRow, len(bs))
	for i, b := range bs {
		members[i] = memberRow{MemberID: b.MemberID, Paid: b.Paid, Consumed: b.Owed}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"currency":    t.BaseCurrency,
		"total":       sum.Total,
		"count":       sum.Count,
		"by_category": sum.ByCategory,
		"by_currency": sum.ByCurrency,
		"by_member":   members,
	})
}

// updateMe lets a member edit their own details (currently PromptPay only).
func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PromptPay *string `json:"promptpay"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	m, err := s.Trips.UpdatePromptPay(r.Context(), tripFrom(r.Context()).ID, memberFrom(r.Context()).ID, req.PromptPay)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

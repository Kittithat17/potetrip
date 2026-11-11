package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestHealthz(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		pingErr  error
		status   int
		wantCode string
	}{
		{"ok root", "/healthz", nil, http.StatusOK, ""},
		{"ok api", "/api/v1/healthz", nil, http.StatusOK, ""},
		{"db down", "/healthz", errors.New("boom"), http.StatusServiceUnavailable, "db_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewServer(fakePinger{tc.pingErr}, []string{"http://localhost:5173"}).Routes()
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			if tc.wantCode != "" {
				var body errorBody
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.Error.Code != tc.wantCode {
					t.Fatalf("code = %q", body.Error.Code)
				}
			}
		})
	}
}

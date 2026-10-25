package hello

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter(t *testing.T) {
	cfg := Config{Env: "staging", Version: "abc123"}
	router := NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   *healthResponse
	}{
		{
			name:       "health returns ok with env and version",
			method:     http.MethodGet,
			path:       HealthPath,
			wantStatus: http.StatusOK,
			wantBody:   &healthResponse{Status: "ok", Env: "staging", Version: "abc123"},
		},
		{
			name:       "health only accepts GET",
			method:     http.MethodPost,
			path:       HealthPath,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "unknown path is not found",
			method:     http.MethodGet,
			path:       "/webhook",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody == nil {
				return
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			var got healthResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			if got != *tt.wantBody {
				t.Errorf("body = %+v, want %+v", got, *tt.wantBody)
			}
		})
	}
}

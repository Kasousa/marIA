package httpx

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Kasousa/maria/pkg/logging"
)

func TestRecoverer(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantLog    string // substring that must appear in the log ("" = no log)
		leaked     string // substring that must never appear in the log
	}{
		{
			name:       "no panic passes through",
			handler:    func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "string panic becomes 500 and only its length is logged",
			handler:    func(http.ResponseWriter, *http.Request) { panic("cliente disse: tomei o remédio") },
			wantStatus: http.StatusInternalServerError,
			wantLog:    `"panic":"[redacted len=30]"`,
			leaked:     "tomei o remédio",
		},
		{
			name:       "error panic keeps the error text with phones masked",
			handler:    func(http.ResponseWriter, *http.Request) { panic(errors.New("send to +5511987654321 failed")) },
			wantStatus: http.StatusInternalServerError,
			wantLog:    `"panic":"send to +55*******4321 failed"`,
			leaked:     "+5511987654321",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := Recoverer(logging.New(&buf, "test", slog.LevelInfo))(tt.handler)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			out := buf.String()
			if tt.wantLog == "" && out != "" {
				t.Errorf("expected no log, got %s", out)
			}
			if tt.wantLog != "" && !strings.Contains(out, tt.wantLog) {
				t.Errorf("log does not contain %s:\n%s", tt.wantLog, out)
			}
			if tt.leaked != "" && strings.Contains(out, tt.leaked) {
				t.Errorf("log leaked %q:\n%s", tt.leaked, out)
			}
		})
	}
}

func TestRecovererRepanicsOnAbortHandler(t *testing.T) {
	h := Recoverer(slog.New(slog.DiscardHandler))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler { //nolint:errorlint // identity check, as net/http does
			t.Errorf("recovered %v, want http.ErrAbortHandler", rec)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil))
	t.Fatal("expected the panic to propagate")
}

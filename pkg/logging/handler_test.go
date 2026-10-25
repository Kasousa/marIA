package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

const (
	rawPhone = "+5511987654321"
	rawText  = "tomei a vitamina D"
)

type stringer string

func (s stringer) String() string { return string(s) }

type valuer string

func (v valuer) LogValue() slog.Value { return slog.StringValue(string(v)) }

// decode returns the single JSON log line written to buf.
func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log output is not one JSON object: %v\n%s", err, buf.String())
	}
	return line
}

func TestMaskingHandler(t *testing.T) {
	tests := []struct {
		name string
		log  func(l *slog.Logger)
		path []string // keys to the attribute that must be checked
		want string
	}{
		{
			name: "sensitive string attribute is masked",
			log:  func(l *slog.Logger) { l.Info("received", "text", rawText) },
			path: []string{"text"},
			want: "[redacted len=18]",
		},
		{
			name: "phone under a sensitive key is masked as phone",
			log:  func(l *slog.Logger) { l.Info("sent", "to", rawPhone) },
			path: []string{"to"},
			want: "+55*******4321",
		},
		{
			name: "already masked value is not masked twice",
			log:  func(l *slog.Logger) { l.Info("sent", "phone", Mask(rawPhone)) },
			path: []string{"phone"},
			want: "+55*******4321",
		},
		{
			name: "non sensitive attribute is kept",
			log:  func(l *slog.Logger) { l.Info("sent", "wamid", "wamid.ABC") },
			path: []string{"wamid"},
			want: "wamid.ABC",
		},
		{
			name: "sensitive attribute inside a group is masked",
			log:  func(l *slog.Logger) { l.Info("received", slog.Group("msg", "body", rawText)) },
			path: []string{"msg", "body"},
			want: "[redacted len=18]",
		},
		{
			name: "sensitive attribute added with With is masked",
			log:  func(l *slog.Logger) { l.With("from", rawPhone).Info("received") },
			path: []string{"from"},
			want: "+55*******4321",
		},
		{
			name: "sensitive attribute after WithGroup is masked",
			log:  func(l *slog.Logger) { l.WithGroup("req").Info("received", "caption", rawText) },
			path: []string{"req", "caption"},
			want: "[redacted len=18]",
		},
		{
			name: "WithGroup followed by With is masked",
			log:  func(l *slog.Logger) { l.WithGroup("req").With("body", rawText).Info("received") },
			path: []string{"req", "body"},
			want: "[redacted len=18]",
		},
		{
			name: "sensitive key is matched case-insensitively",
			log:  func(l *slog.Logger) { l.Info("sent", "Phone", rawPhone) },
			path: []string{"Phone"},
			want: "+55*******4321",
		},
		{
			name: "wa_id is sensitive",
			log:  func(l *slog.Logger) { l.Info("received", "wa_id", rawPhone) },
			path: []string{"wa_id"},
			want: "+55*******4321",
		},
		{
			name: "bytes under a sensitive key are masked",
			log:  func(l *slog.Logger) { l.Info("received", "body", []byte(rawText)) },
			path: []string{"body"},
			want: "[redacted len=18]",
		},
		{
			name: "error under a sensitive key is masked",
			log:  func(l *slog.Logger) { l.Info("received", "message", errors.New(rawText)) },
			path: []string{"message"},
			want: "[redacted len=18]",
		},
		{
			name: "stringer under a sensitive key is masked",
			log:  func(l *slog.Logger) { l.Info("received", "text", stringer(rawText)) },
			path: []string{"text"},
			want: "[redacted len=18]",
		},
		{
			name: "log valuer under a sensitive key is resolved and masked",
			log:  func(l *slog.Logger) { l.Info("sent", "to", valuer(rawPhone)) },
			path: []string{"to"},
			want: "+55*******4321",
		},
		{
			name: "number under a sensitive key is masked",
			log:  func(l *slog.Logger) { l.Info("sent", "phone", int64(5511987654321)) },
			path: []string{"phone"},
			want: "+55*******4321",
		},
		{
			name: "phone inside an error under any key is masked",
			log: func(l *slog.Logger) {
				l.Error("send failed", "err", fmt.Errorf("meta: recipient %s not on allowlist", rawPhone))
			},
			path: []string{"err"},
			want: "meta: recipient +55*******4321 not on allowlist",
		},
		{
			name: "error without phone is kept",
			log:  func(l *slog.Logger) { l.Error("failed", "err", errors.New("timeout after 5s")) },
			path: []string{"err"},
			want: "timeout after 5s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.log(New(&buf, "test", slog.LevelInfo))

			out := buf.String()
			if strings.Contains(out, rawPhone) || strings.Contains(out, rawText) {
				t.Fatalf("raw sensitive value leaked to the log: %s", out)
			}

			var node any = decode(t, &buf)
			for _, key := range tt.path {
				m, ok := node.(map[string]any)
				if !ok {
					t.Fatalf("expected an object at %q in %s", key, out)
				}
				node = m[key]
			}
			if node != tt.want {
				t.Errorf("value at %v = %v, want %q", tt.path, node, tt.want)
			}
		})
	}
}

func TestNewAddsServiceAndRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "hello", slog.LevelWarn)

	l.Info("ignored")
	if buf.Len() != 0 {
		t.Fatalf("info should be dropped at warn level, got %s", buf.String())
	}

	l.Warn("kept")
	line := decode(t, &buf)
	if line["service"] != "hello" {
		t.Errorf("service = %v, want hello", line["service"])
	}
	if line["msg"] != "kept" {
		t.Errorf("msg = %v, want kept", line["msg"])
	}
}

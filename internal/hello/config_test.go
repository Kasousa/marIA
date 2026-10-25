package hello

import (
	"log/slog"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "defaults when only ENV is set",
			env:  map[string]string{"ENV": "staging"},
			want: Config{Env: "staging", Port: "8080", Version: "dev", LogLevel: slog.LevelInfo},
		},
		{
			name: "all variables set",
			env:  map[string]string{"ENV": "prod", "PORT": "9000", "VERSION": "abc123", "LOG_LEVEL": "debug"},
			want: Config{Env: "prod", Port: "9000", Version: "abc123", LogLevel: slog.LevelDebug},
		},
		{
			name:    "missing ENV fails",
			env:     map[string]string{},
			wantErr: "ENV is required",
		},
		{
			name:    "unknown ENV fails",
			env:     map[string]string{"ENV": "dev"},
			wantErr: `ENV must be staging or prod, got "dev"`,
		},
		{
			name:    "invalid LOG_LEVEL fails",
			env:     map[string]string{"ENV": "prod", "LOG_LEVEL": "loud"},
			wantErr: "LOG_LEVEL",
		},
		{
			name:    "reports every problem at once",
			env:     map[string]string{"LOG_LEVEL": "loud"},
			wantErr: "ENV is required\nLOG_LEVEL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := load(func(k string) string { return tt.env[k] })
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("load() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("load() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

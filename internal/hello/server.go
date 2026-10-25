package hello

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Kasousa/maria/pkg/httpx"
)

// HealthPath is the route used by ECS, the API Gateway smoke test and the
// container health check.
const HealthPath = "/healthz"

type healthResponse struct {
	Status  string `json:"status"`
	Env     string `json:"env"`
	Version string `json:"version"`
}

// NewRouter returns the HTTP handler of the hello service.
func NewRouter(cfg Config, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(httpx.Recoverer(logger))
	r.Get(HealthPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := healthResponse{Status: "ok", Env: cfg.Env, Version: cfg.Version}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			logger.Error("write health response", "err", err)
		}
	})
	return r
}

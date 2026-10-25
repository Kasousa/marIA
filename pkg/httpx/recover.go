// Package httpx holds HTTP helpers shared by the marIA services.
package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/Kasousa/maria/pkg/logging"
)

// Recoverer turns a panic in a handler into a 500 and logs it through logger,
// so the panic value goes through the masking handler. Use it instead of
// chi's middleware.Recoverer, which prints the value to stderr unmasked.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				err, isErr := rec.(error)
				if isErr && errors.Is(err, http.ErrAbortHandler) {
					panic(rec) // net/http handles it: abort the response silently
				}
				// Errors keep their text (the masking handler hides phones inside them);
				// any other panic value may be free text, so only its length is kept.
				var value any = logging.Mask(fmt.Sprint(rec))
				if isErr {
					value = err
				}
				logger.ErrorContext(r.Context(), "panic in http handler",
					"path", r.URL.Path,
					"panic", value,
					"stack", string(debug.Stack()),
				)
				w.WriteHeader(http.StatusInternalServerError)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

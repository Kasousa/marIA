package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
)

// sensitiveKeys are attribute keys (compared in lower case) whose values are
// always masked, whatever their type, as a safety net for calls that forgot Mask.
var sensitiveKeys = map[string]bool{
	"phone":     true,
	"to":        true,
	"from":      true,
	"text":      true,
	"body":      true,
	"message":   true,
	"caption":   true,
	"wa_id":     true,
	"msisdn":    true,
	"recipient": true,
}

// phoneInText finds E.164-looking numbers inside free text, such as error messages.
var phoneInText = regexp.MustCompile(`\+?\d{8,15}`)

// New returns a JSON logger for service that masks sensitive attributes.
func New(w io.Writer, service string, level slog.Leveler) *slog.Logger {
	inner := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(NewMaskingHandler(inner)).With("service", service)
}

// MaskingHandler wraps another handler and:
//   - masks the value of any attribute in sensitiveKeys, whatever its type
//     (string, []byte, error, fmt.Stringer, slog.LogValuer...), including
//     attributes inside groups; values that are already Masked pass through;
//   - masks phone numbers found inside error values under any key, because
//     errors from the Meta or Bedrock clients may embed them.
//
// Free text inside an error cannot be detected: callers must still use Mask.
type MaskingHandler struct {
	inner slog.Handler
}

// NewMaskingHandler wraps inner.
func NewMaskingHandler(inner slog.Handler) *MaskingHandler {
	return &MaskingHandler{inner: inner}
}

// Enabled implements slog.Handler.
func (h *MaskingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *MaskingHandler) Handle(ctx context.Context, r slog.Record) error {
	masked := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		masked.AddAttrs(maskAttr(a))
		return true
	})
	return h.inner.Handle(ctx, masked)
}

// WithAttrs implements slog.Handler.
func (h *MaskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		out[i] = maskAttr(a)
	}
	return &MaskingHandler{inner: h.inner.WithAttrs(out)}
}

// WithGroup implements slog.Handler.
func (h *MaskingHandler) WithGroup(name string) slog.Handler {
	return &MaskingHandler{inner: h.inner.WithGroup(name)}
}

func maskAttr(a slog.Attr) slog.Attr {
	if _, ok := a.Value.Any().(Masked); ok {
		return a
	}
	v := a.Value.Resolve()

	if v.Kind() == slog.KindGroup {
		group := v.Group()
		out := make([]any, len(group))
		for i, g := range group {
			out[i] = maskAttr(g)
		}
		return slog.Group(a.Key, out...)
	}
	if sensitiveKeys[strings.ToLower(a.Key)] {
		return slog.Any(a.Key, Mask(stringify(v)))
	}
	if err, ok := v.Any().(error); ok {
		return slog.String(a.Key, scrubPhones(err.Error()))
	}
	return slog.Attr{Key: a.Key, Value: v}
}

// stringify renders any resolved slog value as the text that would be logged.
func stringify(v slog.Value) string {
	if v.Kind() != slog.KindAny {
		return v.String()
	}
	switch x := v.Any().(type) {
	case []byte:
		return string(x)
	case error:
		return x.Error()
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}

func scrubPhones(s string) string {
	return phoneInText.ReplaceAllStringFunc(s, func(m string) string { return Mask(m).String() })
}

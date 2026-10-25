// Package logging builds the JSON slog loggers used by every marIA service and
// makes sure message text and phone numbers never reach the logs in clear.
package logging

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"
)

// Masked is a value that is safe to log. Build it with Mask.
type Masked struct {
	redacted string
}

// Mask returns a log-safe version of s.
//
// Phone numbers (E.164, with or without "+") keep the country code and the
// last four digits: "+5511987654321" becomes "+55*******4321". Any other
// text is replaced by its length only: "tomei o remédio" becomes
// "[redacted len=15]". The empty string stays empty.
func Mask(s string) Masked {
	return Masked{redacted: redact(s)}
}

// LogValue implements slog.LogValuer.
func (m Masked) LogValue() slog.Value {
	return slog.StringValue(m.redacted)
}

// String implements fmt.Stringer, so a Masked is also safe in fmt calls.
func (m Masked) String() string {
	return m.redacted
}

func redact(s string) string {
	if s == "" {
		return ""
	}
	if digits, ok := phoneDigits(s); ok {
		hidden := len(digits) - 6
		return "+" + digits[:2] + strings.Repeat("*", hidden) + digits[len(digits)-4:]
	}
	return fmt.Sprintf("[redacted len=%d]", utf8.RuneCountInString(s))
}

// phoneDigits reports whether s looks like an E.164 number (8 to 15 digits,
// optional leading "+") and returns its digits.
func phoneDigits(s string) (string, bool) {
	digits := strings.TrimPrefix(s, "+")
	if len(digits) < 8 || len(digits) > 15 {
		return "", false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return digits, true
}

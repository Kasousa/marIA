// Fixture for scripts/test-harness.sh: every line marked "want" must be reported by archcheck.
package conversation

import (
	"github.com/Kasousa/maria/internal/conversation/tools"
	"github.com/Kasousa/maria/internal/reminders" // want: cross-service import
	"github.com/Kasousa/maria/pkg/meta"           // want: only outbound sends to Meta
	"github.com/Kasousa/maria/pkg/meta/webhook"
)

var table = "maria-reminders-staging" // want: table owned by reminders

var region = os.Getenv("AWS_REGION") // want: env read outside config.go

func handle() {
	now := time.Now()                                 // want: injected clock
	logger.Info("received", "text", msg)              // want: unmasked text
	slog.ErrorContext(ctx, "send failed", "phone", p) // want: unmasked phone
	logger.Info("received", "text", logging.Mask(msg))
}

// Fixture for scripts/test-harness.sh: nothing here may be reported by archcheck.
package outbound

import (
	"github.com/Kasousa/maria/internal/outbound/window"
	"github.com/Kasousa/maria/pkg/meta"
)

var table = "maria-outbound-prod"

func send() {
	logger.Info("sent", "text", logging.Mask(text), "wamid", id)
	started := time.Now() // archcheck:allow measuring latency, not business logic
}

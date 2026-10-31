package webhook

import (
	"testing"

	"github.com/wI2L/fizz/internal/conformance/repro"
)

// TestReproWebhook rebuilds learn.openapis.org
// examples/v3.1/webhook-example.json.
func TestReproWebhook(t *testing.T) {
	repro.Run(t, New(), "v3.1/webhook-example.json", "allow.json")
}

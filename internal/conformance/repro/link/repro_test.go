package link

import (
	"testing"

	"github.com/wI2L/fizz/internal/conformance/repro"
	"github.com/wI2L/fizz/internal/plan"
)

// TestReproLink rebuilds learn.openapis.org examples/v3.0/link-example.json.
//
// The served spec $refs #/components/links/... entries that fizz cannot
// register (no RegisterLink), which the semantic lint reports as unresolved
// local references. Red until P10.9k; allow.json stays as it is.
func TestReproLink(t *testing.T) {
	plan.Pending(t, "P10.9k", func(tb testing.TB) {
		tb.Helper()
		repro.Run(tb, New(), "v3.0/link-example.json", "allow.json")
	})
}

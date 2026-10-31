package uspto

import (
	"testing"

	"github.com/wI2L/fizz/internal/conformance/repro"
)

// TestReproUSPTO rebuilds learn.openapis.org examples/v3.0/uspto.json.
func TestReproUSPTO(t *testing.T) {
	repro.Run(t, New(), "v3.0/uspto.json", "allow.json")
}

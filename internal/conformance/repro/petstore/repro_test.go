package petstore

import (
	"testing"

	"github.com/wI2L/fizz/internal/conformance/repro"
)

// TestReproPetstore rebuilds learn.openapis.org examples/v3.0/petstore.json.
func TestReproPetstore(t *testing.T) {
	repro.Run(t, New(), "v3.0/petstore.json", "allow.json")
}

package callback

import (
	"testing"

	"github.com/KeisukeFD/fizz/internal/conformance/repro"
)

// TestReproCallback rebuilds learn.openapis.org
// examples/v3.0/callback-example.json.
func TestReproCallback(t *testing.T) {
	repro.Run(t, New(), "v3.0/callback-example.json", "allow.json")
}

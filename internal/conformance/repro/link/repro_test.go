package link

import (
	"testing"

	"github.com/wI2L/fizz/internal/conformance/repro"
)

// TestReproLink rebuilds learn.openapis.org examples/v3.0/link-example.json.
func TestReproLink(t *testing.T) {
	repro.Run(t, New(), "v3.0/link-example.json", "allow.json")
}

package tags32

import (
	"testing"

	"github.com/wI2L/fizz/internal/conformance/repro"
)

// TestReproTags32 rebuilds learn.openapis.org examples/v3.2/3.2-tags-example.json.
func TestReproTags32(t *testing.T) {
	repro.Run(t, New(), "v3.2/3.2-tags-example.json", "allow.json")
}

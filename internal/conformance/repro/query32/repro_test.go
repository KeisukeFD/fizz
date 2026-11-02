package query32

import (
	"testing"

	"github.com/KeisukeFD/fizz/internal/conformance/repro"
)

// TestReproQuery32 rebuilds learn.openapis.org examples/v3.2/3.2-query-example.json.
func TestReproQuery32(t *testing.T) {
	repro.Run(t, New(), "v3.2/3.2-query-example.json", "allow.json")
}

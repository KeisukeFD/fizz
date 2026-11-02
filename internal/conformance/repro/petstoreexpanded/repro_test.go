package petstoreexpanded

import (
	"testing"

	"github.com/KeisukeFD/fizz/internal/conformance/repro"
)

// TestReproPetstoreExpanded rebuilds learn.openapis.org
// examples/v3.0/petstore-expanded.json.
func TestReproPetstoreExpanded(t *testing.T) {
	repro.Run(t, New(), "v3.0/petstore-expanded.json", "allow.json")
}

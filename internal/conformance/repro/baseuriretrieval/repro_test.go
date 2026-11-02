package baseuriretrieval

import (
	"testing"

	"github.com/KeisukeFD/fizz/internal/conformance/repro"
)

// TestReproBaseURIRetrieval rebuilds the spec-text document
// base-uri-from-the-retrieval-uri-1.
func TestReproBaseURIRetrieval(t *testing.T) {
	repro.RunAt(t, New(), "spectext/base-uri-from-the-retrieval-uri-1.json", "allow.json")
}

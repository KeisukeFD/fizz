package apiwithexamples

import (
	"testing"

	"github.com/KeisukeFD/fizz/internal/conformance/repro"
)

// TestReproAPIWithExamples rebuilds learn.openapis.org
// examples/v3.0/api-with-examples.json.
func TestReproAPIWithExamples(t *testing.T) {
	repro.Run(t, New(), "v3.0/api-with-examples.json", "allow.json")
}

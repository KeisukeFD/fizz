package nonoauthscopes

import (
	"testing"

	"github.com/wI2L/fizz/internal/conformance/repro"
)

// TestReproNonOAuthScopes rebuilds learn.openapis.org
// examples/v3.1/non-oauth-scopes.json.
func TestReproNonOAuthScopes(t *testing.T) {
	repro.Run(t, New(), "v3.1/non-oauth-scopes.json", "allow.json")
}

package nonoauthscopes

import (
	"testing"

	"github.com/KeisukeFD/fizz/internal/conformance/repro"
)

// TestReproNonOAuthScopes rebuilds learn.openapis.org
// examples/v3.1/non-oauth-scopes.json.
func TestReproNonOAuthScopes(t *testing.T) {
	repro.Run(t, New(), "v3.1/non-oauth-scopes.json", "allow.json")
}

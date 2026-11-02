package link

import "testing"

// TestReproLink rebuilds learn.openapis.org examples/v3.0/link-example.json.
//
// The served spec $refs #/components/links/... entries that fizz cannot
// register (no RegisterLink), which the semantic lint reports as unresolved
// local references.
func TestReproLink(t *testing.T) {
	t.Skip("known limitation: no RegisterLink")
}

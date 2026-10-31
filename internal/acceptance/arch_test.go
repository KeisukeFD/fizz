package acceptance

// Batch A (default build): architecture rules. See
// docs/plan/ACCEPTANCE.md, Batch A.
//
// Every package-level identifier is prefixed bA, as sibling files
// share the package.

import (
	"reflect"
	"slices"
	"testing"

	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/openapi"
)

// A09: only fizz-owned tags are configurable.
func TestBinderTagsNotConfigurable(t *testing.T) {
	at.Pending(t, "P6.1", func(tb testing.TB) {
		want := []string{
			"Description", "Example", "Format", "Deprecated", "Title", "Pattern",
			"ReadOnly", "WriteOnly", "Nullable", "MultipleOf", "AdditionalProperties",
			"Inline", "Style", "AllowReserved", "Cookie",
		}
		var got []string
		for f := range reflect.TypeFor[openapi.TagNames]().Fields() {
			got = append(got, f.Name)
		}
		if !slices.Equal(got, want) {
			tb.Errorf("TagNames fields = %v, want %v", got, want)
		}
		for _, banned := range []string{"Path", "Query", "Header", "Enum", "Default", "Validate", "Explode"} {
			if slices.Contains(got, banned) {
				tb.Errorf("TagNames has binder tag field %s", banned)
			}
		}
	})
}

package acceptance

// Batch A (default build): architecture rules. See
// docs/plan/ACCEPTANCE.md, Batch A.
//
// Every package-level identifier is prefixed bA, as sibling files
// share the package.

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/wI2L/fizz"
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

// bAWalk reports the types reachable from t, through fields, element
// types and function parameters and results, whose package is gin or
// gadgeto. seen guards the cycles.
func bAWalk(t reflect.Type, seen map[reflect.Type]bool, bad *[]string) {
	if t == nil || seen[t] {
		return
	}
	seen[t] = true
	if p := t.PkgPath(); strings.Contains(p, "gin-gonic/gin") || strings.Contains(p, "loopfz/gadgeto") {
		*bad = append(*bad, t.String())
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Chan:
		bAWalk(t.Elem(), seen, bad)
	case reflect.Map:
		bAWalk(t.Key(), seen, bad)
		bAWalk(t.Elem(), seen, bad)
	case reflect.Struct:
		for f := range t.Fields() {
			bAWalk(f.Type, seen, bad)
		}
	case reflect.Func:
		for i := range t.NumIn() {
			bAWalk(t.In(i), seen, bad)
		}
		for i := range t.NumOut() {
			bAWalk(t.Out(i), seen, bad)
		}
	}
}

// A07: the option and hook types expose no gin or gadgeto type.
func TestNoGinTypesInPublicSignatures(t *testing.T) {
	at.Pending(t, "P6.4", func(tb testing.TB) {
		for _, typ := range []reflect.Type{
			reflect.TypeFor[fizz.OperationOption](),
			reflect.TypeFor[openapi.OperationInfo](),
			reflect.TypeFor[openapi.Hooks](),
			reflect.TypeFor[openapi.SchemaContext](),
			reflect.TypeFor[openapi.FieldContext](),
			reflect.TypeFor[openapi.ParameterContext](),
			reflect.TypeFor[openapi.OperationContext](),
		} {
			var bad []string
			bAWalk(typ, map[reflect.Type]bool{}, &bad)
			if len(bad) > 0 {
				tb.Errorf("%s reaches %v", typ, bad)
			}
		}
	})
}

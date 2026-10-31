package kitchensink

import (
	"testing"

	"github.com/wI2L/fizz/internal/golden"
	"github.com/wI2L/fizz/internal/plan"
)

// TestKitchenSinkGolden asserts the whole document of NewWholeFizz (JSON
// and YAML).
func TestKitchenSinkGolden(t *testing.T) {
	f := NewWholeFizz()
	if errs := f.Errors(); len(errs) != 0 {
		t.Fatalf("registration errors: %v", errs)
	}
	doc, err := BuildSpec(f)
	if err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, wholeGoldenJSON, doc)
	golden.Assert(t, wholeGoldenYAML, serveSpec(t, f, "yaml"))
}

// TestKitchenSinkMeta runs the meta-schema gate on the whole fixture. It
// is red until P4.5: the generic component name
// "GenericPage[...LegacyPet]" breaks ^[a-zA-Z0-9._-]+$ (validate and
// component-names).
func TestKitchenSinkMeta(t *testing.T) {
	plan.Pending(t, "P4.5", func(tb testing.TB) {
		doc, err := BuildSpec(NewWholeFizz())
		if err != nil {
			tb.Fatal(err)
		}
		for _, p := range MetaProblems(doc) {
			tb.Error(p)
		}
	})
}

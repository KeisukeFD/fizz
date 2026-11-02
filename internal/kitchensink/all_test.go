package kitchensink

import (
	"testing"

	"github.com/KeisukeFD/fizz/internal/golden"
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

// TestKitchenSinkMeta runs the meta-schema gate on the whole fixture.
func TestKitchenSinkMeta(t *testing.T) {
	doc, err := BuildSpec(NewWholeFizz())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range MetaProblems(doc) {
		t.Error(p)
	}

}

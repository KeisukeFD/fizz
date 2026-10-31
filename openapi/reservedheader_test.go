package openapi

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

type (
	rhInput struct {
		A  string `header:"Accept"`
		CT string `header:"content-type"`
		Au string `header:"AUTHORIZATION"`
		X  string `header:"X-Keep"`
	}
	rhQuery struct {
		A string `query:"Accept"`
	}
)

// A header parameter named Accept, Content-Type or Authorization is not
// emitted (OAS 3.2 ignores it), with one warning each; the same names in
// other locations are unaffected.
func TestReservedHeaderParamsAreDropped(t *testing.T) {
	g, op := build(t, reflect.TypeFor[rhInput]())
	if got := paramKeys(op); !slices.Equal(got, []string{"header:X-Keep"}) {
		t.Errorf("params = %v", got)
	}
	if len(g.Errors()) != 3 {
		t.Fatalf("diagnostics = %q, want 3 warnings", errorsText(g))
	}
	for _, e := range g.Errors() {
		if _, ok := e.(*Warning); !ok {
			t.Errorf("%v is not a warning", e)
		}
	}
	text := errorsText(g)
	for _, s := range []string{`"Accept"`, `"content-type"`, `"AUTHORIZATION"`, "security scheme"} {
		if !strings.Contains(text, s) {
			t.Errorf("diagnostics %q lack %q", text, s)
		}
	}

	g, op = build(t, reflect.TypeFor[rhQuery]())
	if got := paramKeys(op); !slices.Equal(got, []string{"query:Accept"}) || len(g.Errors()) != 0 {
		t.Errorf("query Accept: params = %v, diagnostics = %q", got, errorsText(g))
	}
}

func TestReservedHeaderParamOptionIsDropped(t *testing.T) {
	g := gen(t)
	op, err := poAdd(g, "/p", &Parameter{Name: "authorization", In: InHeader}, &Parameter{Name: "X-Ok", In: InHeader})
	if err != nil {
		t.Fatal(err)
	}
	if got := paramKeys(op); !slices.Equal(got, []string{"header:X-Ok", "query:q"}) && !slices.Equal(got, []string{"query:q", "header:X-Ok"}) {
		t.Errorf("params = %v", got)
	}
	if text := errorsText(g); !strings.Contains(text, "authorization") {
		t.Errorf("diagnostics = %q, want a warning", text)
	}
}

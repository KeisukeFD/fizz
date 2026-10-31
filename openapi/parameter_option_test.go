package openapi

import (
	"reflect"
	"strings"
	"testing"
)

type poQuery struct {
	Q string `query:"q"`
}

func poAdd(g *Generator, path string, ps ...*Parameter) (*Operation, error) {
	return g.AddOperation(path, "GET", "", reflect.TypeFor[poQuery](), nil,
		&OperationInfo{ID: "po", StatusCode: 200, Parameters: ps})
}

func poParam(t *testing.T, op *Operation, name, in string) *Parameter {
	t.Helper()
	for _, p := range op.Parameters {
		if p.Value != nil && p.Value.Name == name && p.Value.In == in {
			return p.Value
		}
	}
	t.Fatalf("parameter %s/%s not found", name, in)
	return nil
}

func TestParameterOptionExtensionsMergeByKey(t *testing.T) {
	g := gen(t)
	op, err := poAdd(g, "/p",
		&Parameter{Name: "q", In: "query", Extensions: map[string]any{"x-a": "1", "x-b": "1"}},
		&Parameter{Name: "q", In: "query", Extensions: map[string]any{"x-b": "2", "x-c": "3"}})
	if err != nil {
		t.Fatal(err)
	}
	p := poParam(t, op, "q", "query")
	want := map[string]any{"x-a": "1", "x-b": "2", "x-c": "3"}
	if !reflect.DeepEqual(p.Extensions, want) {
		t.Errorf("extensions = %v, want %v", p.Extensions, want)
	}
	if p.Schema == nil {
		t.Error("the inferred schema must survive")
	}
}

func TestParameterOptionErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    *Parameter
		want string
	}{
		{"empty name", &Parameter{In: "query"}, "name"},
		{"empty in", &Parameter{Name: "z"}, "\"in\""},
		{"path not in template", &Parameter{Name: "id", In: "path"}, "path template"},
		{"invalid in", &Parameter{Name: "b", In: "body"}, `"body"`},
		{"schema and content", &Parameter{Name: "z", In: "query", Schema: &Schema{}, Content: map[string]*MediaType{"application/json": {}}}, "both schema and content"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := gen(t)
			op, err := poAdd(g, "/p", tc.p)
			if err != nil {
				t.Fatal(err)
			}
			if len(op.Parameters) != 1 {
				t.Errorf("parameters = %d, want only the inferred one", len(op.Parameters))
			}
			errs, _ := rboErrors(g)
			if len(errs) != 1 || !strings.Contains(errs[0].Error(), tc.want) {
				t.Errorf("errors = %v, want one mentioning %q", errs, tc.want)
			}
		})
	}
}

// TestParameterOptionSchemaContentExclusive: a parameter has a schema or
// a content, never both, so the one an option sets replaces the other.
func TestParameterOptionSchemaContentExclusive(t *testing.T) {
	t.Run("content replaces inferred schema", func(t *testing.T) {
		g := gen(t)
		op, err := poAdd(g, "/p", &Parameter{Name: "q", In: "query",
			Content: map[string]*MediaType{"application/json": {}}})
		if err != nil {
			t.Fatal(err)
		}
		p := poParam(t, op, "q", "query")
		if p.Schema != nil || len(p.Content) != 1 {
			t.Errorf("schema = %v, content = %v: want content only", p.Schema, p.Content)
		}
	})
	t.Run("schema replaces content", func(t *testing.T) {
		g := gen(t)
		op, err := poAdd(g, "/p",
			&Parameter{Name: "q", In: "query", Content: map[string]*MediaType{"application/json": {}}},
			&Parameter{Name: "q", In: "query", Schema: &Schema{Type: Types("integer")}})
		if err != nil {
			t.Fatal(err)
		}
		p := poParam(t, op, "q", "query")
		if p.Schema == nil || p.Content != nil {
			t.Errorf("schema = %v, content = %v: want schema only", p.Schema, p.Content)
		}
	})
}

func TestParameterOptionIsDeepCopied(t *testing.T) {
	g := gen(t)
	p := &Parameter{Name: "z", In: "query", Extensions: map[string]any{"x-a": "1"}, Schema: &Schema{Type: Types("string")}}
	op, err := poAdd(g, "/p", p)
	if err != nil {
		t.Fatal(err)
	}
	got := poParam(t, op, "z", "query")
	got.Extensions["x-z"] = "1"
	got.Schema.Title = "t"
	if _, ok := p.Extensions["x-z"]; ok || p.Schema.Title != "" {
		t.Error("the stored parameter shares memory with the option's")
	}
}

func TestParameterOptionMergeIntoRefIsError(t *testing.T) {
	g := gen(t)
	// The component name differs from the parameter name: the match is
	// made on the resolved parameter.
	if err := g.RegisterParameter("QueryQ", &Parameter{Name: "q", In: "query", Schema: &Schema{Type: Types("string")}}); err != nil {
		t.Fatal(err)
	}
	op := &Operation{Parameters: []*ParameterOrRef{{Ref: &Reference{Ref: "#/components/parameters/QueryQ"}}}}
	g.applyParameters(op, "/p", []*Parameter{{Name: "q", In: "header", Description: "x"}})
	if errs, _ := rboErrors(g); len(errs) != 0 || len(op.Parameters) != 2 {
		t.Fatalf("(q, header) must be appended, not matched to the (q, query) $ref: errors = %v, params = %d", errs, len(op.Parameters))
	}
	g.applyParameters(op, "/p", []*Parameter{{Name: "q", In: "query", Description: "x"}})
	errs, _ := rboErrors(g)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "$ref") {
		t.Errorf("errors = %v, want one mentioning $ref", errs)
	}
}

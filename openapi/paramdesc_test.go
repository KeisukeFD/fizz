package openapi

import (
	"reflect"
	"testing"
)

type (
	pdInput struct {
		Q string `query:"q" description:"query desc"`
		H string `header:"X-H" description:"header desc"`
		P string `path:"p" description:"path desc"`
		C string `cookie:"c" description:"cookie desc"`
	}
	pdBody struct {
		F string `json:"f" description:"field desc"`
	}
)

// The description tag of a parameter is the description of the
// parameter, not of its schema; a body field keeps it on its schema.
func TestParamDescriptionIsNotInSchema(t *testing.T) {
	_, op := build(t, reflect.TypeFor[pdInput]())
	want := map[string]string{"q": "query desc", "X-H": "header desc", "p": "path desc", "c": "cookie desc"}
	if len(op.Parameters) != len(want) {
		t.Fatalf("params = %v", paramKeys(op))
	}
	for _, pr := range op.Parameters {
		p := pr.Value
		if p.Description != want[p.Name] {
			t.Errorf("%s: description = %q, want %q", p.Name, p.Description, want[p.Name])
		}
		if p.Schema.Description != "" {
			t.Errorf("%s: schema description = %q, want none", p.Name, p.Schema.Description)
		}
	}

	g := gen(t)
	g.newSchemaFromType(reflect.TypeFor[pdBody]())
	if d := g.api.Components.Schemas["PdBody"].Properties["f"].Description; d != "field desc" {
		t.Errorf("body field description = %q, want %q", d, "field desc")
	}
}

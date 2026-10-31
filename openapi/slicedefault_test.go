package openapi

import (
	"reflect"
	"strings"
	"testing"
)

type (
	sdQuery struct {
		IDs   []int    `query:"ids" default:"1,2,3"`
		Names []string `query:"names" default:"a,b"`
		One   []string `query:"one" default:"x"`
		Ptrs  []*int   `query:"ptrs" default:"4,5"`
		Pair  [2]int   `query:"pair" default:"7,8"`
		NoExp []string `query:"noexp" explode:"false" default:"a,b"`
		Hdr   []string `header:"X-List" default:"h1,h2"`
	}
	sdBadElem struct {
		IDs []int `query:"ids" default:"1,x"`
	}
	sdBadLen struct {
		Pair [2]int `query:"pair" default:"7,8,9"`
	}
	sdBody struct {
		Tags []string `json:"tags" default:"a,b"`
	}
)

func TestSliceDefaultIsSplitPerElement(t *testing.T) {
	g, op := build(t, reflect.TypeFor[sdQuery]())
	if len(g.Errors()) != 0 {
		t.Fatalf("errors = %s", errorsText(g))
	}
	want := map[string]any{
		"ids":    []any{int64(1), int64(2), int64(3)},
		"names":  []any{"a", "b"},
		"one":    []any{"x"},
		"ptrs":   []any{int64(4), int64(5)},
		"pair":   []any{int64(7), int64(8)},
		"noexp":  []any{"a,b"},
		"X-List": []any{"h1", "h2"},
	}
	for _, p := range op.Parameters {
		w, ok := want[p.Value.Name]
		if !ok {
			t.Fatalf("unexpected parameter %q", p.Value.Name)
		}
		if got := p.Value.Schema.Default; !reflect.DeepEqual(got, w) {
			t.Errorf("%s: default = %#v, want %#v", p.Value.Name, got, w)
		}
		delete(want, p.Value.Name)
	}
	if len(want) != 0 {
		t.Errorf("missing parameters %v", want)
	}
}

func TestSliceDefaultErrors(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"bad element", reflect.TypeFor[sdBadElem](), "cannot be converted"},
		{"array length", reflect.TypeFor[sdBadLen](), "array of length 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, _ := build(t, tt.typ)
			errs := g.Errors()
			if len(errs) != 1 || !strings.Contains(errs[0].Error(), tt.want) {
				t.Errorf("errors = %q, want one mentioning %q", errorsText(g), tt.want)
			}
		})
	}
}

func TestSliceDefaultOnBodyField(t *testing.T) {
	g := gen(t)
	s := g.newSchemaFromType(reflect.TypeFor[sdBody]())
	if len(g.Errors()) != 0 {
		t.Fatalf("errors = %s", errorsText(g))
	}
	if s.Ref != "" {
		s = g.api.Components.Schemas[strings.TrimPrefix(s.Ref, "#/components/schemas/")]
	}
	if got, want := s.Properties["tags"].Default, []any{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("default = %#v, want %#v", got, want)
	}
}

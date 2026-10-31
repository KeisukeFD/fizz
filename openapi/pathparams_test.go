package openapi

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type (
	ppNone    struct{}
	ppPartial struct {
		Other string `query:"other"`
	}
	ppDeclared struct {
		ID string `path:"id" description:"The ID."`
	}
	ppWild struct {
		P string `path:"p" description:"The tail."`
	}
	ppWildBare struct {
		P string `path:"p"`
	}
)

func ppParams(t *testing.T, path string, in reflect.Type, wild []string) (*Generator, []*Parameter) {
	t.Helper()
	g := gen(t)
	op, err := g.AddOperation(path, "GET", "", in, nil, &OperationInfo{ID: "op", StatusCode: 200, WildcardParams: wild})
	if err != nil {
		t.Fatal(err)
	}
	var ps []*Parameter
	for _, p := range op.Parameters {
		ps = append(ps, p.Value)
	}
	return g, ps
}

func ppWarnings(t *testing.T, g *Generator, want int) {
	t.Helper()
	errs := g.Errors()
	if len(errs) != want {
		t.Fatalf("got %d errors %v, want %d", len(errs), errs, want)
	}
	for _, e := range errs {
		var w *Warning
		if !errors.As(e, &w) || !w.IsWarning() || w.Method != "GET" || w.ID != "op" {
			t.Errorf("not an attributed warning: %#v", e)
		}
	}
}

func TestSynthesizedPathParam(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   reflect.Type
	}{
		{"no input", nil},
		{"empty input", reflect.TypeFor[ppNone]()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, ps := ppParams(t, "/u/{id}", tt.in, nil)
			if len(ps) != 1 || ps[0].Name != "id" || ps[0].In != "path" || !ps[0].Required ||
				!reflect.DeepEqual(ps[0].Schema.Type, Types("string")) || ps[0].Description != "" {
				t.Fatalf("unexpected params: %+v", ps)
			}
			ppWarnings(t, g, 1)
			if !strings.Contains(g.Errors()[0].Error(), `"id"`) {
				t.Errorf("warning does not name the parameter: %v", g.Errors()[0])
			}
		})
	}
}

func TestSynthesizedPathParamPartialInput(t *testing.T) {
	g, ps := ppParams(t, "/u/{id}", reflect.TypeFor[ppPartial](), nil)
	if len(ps) != 2 || ps[0].Name != "id" || ps[0].In != "path" || ps[1].Name != "other" {
		t.Fatalf("path param must be synthesized and sorted first: %+v", ps)
	}
	ppWarnings(t, g, 1)

	// A declared parameter is not duplicated and warns of nothing.
	g, ps = ppParams(t, "/u/{id}", reflect.TypeFor[ppDeclared](), nil)
	if len(ps) != 1 || ps[0].Description != "The ID." {
		t.Fatalf("unexpected params: %+v", ps)
	}
	ppWarnings(t, g, 0)
}

func TestWildcardPathParam(t *testing.T) {
	note := "The value may contain '/'."
	for _, tt := range []struct {
		name  string
		in    reflect.Type
		want  string
		warns int
	}{
		{"declared with description", reflect.TypeFor[ppWild](), "The tail. " + note, 0},
		{"declared bare", reflect.TypeFor[ppWildBare](), note, 0},
		{"synthesized", nil, note, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, ps := ppParams(t, "/f/{p}", tt.in, []string{"p"})
			if len(ps) != 1 || !ps[0].Required || ps[0].Description != tt.want {
				t.Fatalf("unexpected params: %+v", ps)
			}
			ppWarnings(t, g, tt.warns)
		})
	}
	// Without the wildcard marker, the description is left alone.
	_, ps := ppParams(t, "/f/{p}", reflect.TypeFor[ppWild](), nil)
	if ps[0].Description != "The tail." {
		t.Errorf("description = %q", ps[0].Description)
	}
}

func TestWarningsOnlyIsNotAnError(t *testing.T) {
	g, _ := ppParams(t, "/u/{id}", nil, nil)
	for _, e := range g.Errors() {
		var w *Warning
		if !errors.As(e, &w) {
			t.Fatalf("want only warnings, got %v", e)
		}
	}
}

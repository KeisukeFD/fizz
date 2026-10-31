package openapi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	anShared struct{ V string }
	anHolder struct {
		A  anShared   `json:"a" description:"first"`
		B  anShared   `json:"b" description:"second" deprecated:"true"`
		N  *anShared  `json:"n" description:"nullable"`
		L  []anShared `json:"l" description:"list"`
		L2 []anShared `json:"l2" description:"plain list"`
		S  []string   `json:"s" enum:"p,q"`
		S2 []string   `json:"s2"`
	}
	anPlain struct{ P anShared }
)

const (
	anRef       = `{"$ref":"#/components/schemas/AnShared"}`
	anComponent = `{"type":"object","properties":{"V":{"type":"string"}}}`
)

func anGen(t *testing.T, style AnnotationStyle, types ...reflect.Type) *Generator {
	t.Helper()
	g := gen(t)
	g.SetAnnotationStyle(style)
	for _, typ := range types {
		require.NotNil(t, g.newSchemaFromType(typ))
	}
	return g
}

func TestUseSiteAnnotationsSiblings(t *testing.T) {
	orders := map[string][]reflect.Type{
		"holder first": {reflect.TypeFor[anHolder](), reflect.TypeFor[anPlain]()},
		"holder last":  {reflect.TypeFor[anPlain](), reflect.TypeFor[anHolder]()},
	}
	for name, types := range orders {
		t.Run(name, func(t *testing.T) {
			g := anGen(t, SiblingAnnotations, types...)
			assert.JSONEq(t, anComponent, schemaJSON(t, g.api.Components.Schemas["AnShared"]))
			props := g.api.Components.Schemas["AnHolder"].Properties
			assert.JSONEq(t, `{"$ref":"#/components/schemas/AnShared","description":"first"}`, schemaJSON(t, props["a"]))
			assert.JSONEq(t, `{"$ref":"#/components/schemas/AnShared","description":"second","deprecated":true}`, schemaJSON(t, props["b"]))
			assert.JSONEq(t, `{"anyOf":[`+anRef+`,{"type":"null"}],"description":"nullable"}`, schemaJSON(t, props["n"]))
			assert.JSONEq(t, `{"type":"array","items":`+anRef+`,"description":"list"}`, schemaJSON(t, props["l"]))
			assert.JSONEq(t, `{"type":"array","items":`+anRef+`,"description":"plain list"}`, schemaJSON(t, props["l2"]))
			assert.JSONEq(t, `{"type":"array","items":{"type":"string","enum":["p","q"]}}`, schemaJSON(t, props["s"]))
			assert.JSONEq(t, `{"type":"array","items":{"type":"string"}}`, schemaJSON(t, props["s2"]))
			assert.JSONEq(t, `{"$ref":"#/components/schemas/AnShared"}`, schemaJSON(t, g.api.Components.Schemas["AnPlain"].Properties["P"]))
		})
	}
}

func TestUseSiteAnnotationsAllOfWrap(t *testing.T) {
	g := anGen(t, AllOfWrap, reflect.TypeFor[anHolder](), reflect.TypeFor[anPlain]())
	assert.JSONEq(t, anComponent, schemaJSON(t, g.api.Components.Schemas["AnShared"]))
	props := g.api.Components.Schemas["AnHolder"].Properties
	assert.JSONEq(t, `{"allOf":[`+anRef+`],"description":"first"}`, schemaJSON(t, props["a"]))
	assert.JSONEq(t, `{"allOf":[`+anRef+`],"description":"second","deprecated":true}`, schemaJSON(t, props["b"]))
	// O26: nullable keeps the annotations beside the anyOf.
	assert.JSONEq(t, `{"anyOf":[`+anRef+`,{"type":"null"}],"description":"nullable"}`, schemaJSON(t, props["n"]))
	assert.JSONEq(t, `{"type":"array","items":`+anRef+`,"description":"list"}`, schemaJSON(t, props["l"]))
	// A bare reference is not wrapped.
	assert.JSONEq(t, anRef, schemaJSON(t, g.api.Components.Schemas["AnPlain"].Properties["P"]))
}

// resolveSchema returns either the inlined schema in s or the component
// it references. It is read-only: production code never resolves a
// reference to annotate it.
func (g *Generator) resolveSchema(s *Schema) *Schema {
	if s.Ref == "" {
		return s
	}
	name, ok := strings.CutPrefix(s.Ref, componentsSchemaPath)
	if !ok {
		return nil
	}
	return g.api.Components.Schemas[name]
}

func TestAnnotatedItems(t *testing.T) {
	enum := []any{"x", "y"}
	for _, tt := range []struct {
		name  string
		style AnnotationStyle
		in    *Schema
		want  string
	}{
		{"ref sibling", SiblingAnnotations, &Schema{Ref: "#/components/schemas/A"}, `{"$ref":"#/components/schemas/A","enum":["x","y"]}`},
		{"ref allOf", AllOfWrap, &Schema{Ref: "#/components/schemas/A"}, `{"allOf":[{"$ref":"#/components/schemas/A"}],"enum":["x","y"]}`},
		{"inline", AllOfWrap, &Schema{Type: Types("string")}, `{"type":"string","enum":["x","y"]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := gen(t)
			g.SetAnnotationStyle(tt.style)
			before := schemaJSON(t, tt.in)
			got := g.annotatedItems(tt.in, enum)
			assert.JSONEq(t, tt.want, schemaJSON(t, got))
			assert.JSONEq(t, before, schemaJSON(t, tt.in), "original items untouched")
		})
	}
}

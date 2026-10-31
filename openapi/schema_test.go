package openapi

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	SchL     []SchL
	SchM     map[string]SchM
	SchPlain []string
	SchA     []SchB
	SchB     map[string]SchA
	SchOv    string
)

func schemaJSON(t *testing.T, s *Schema) string {
	t.Helper()
	b, err := json.Marshal(s)
	require.NoError(t, err)
	return string(b)
}

func TestSchemaForElements(t *testing.T) {
	g := gen(t)
	require.NoError(t, g.OverrideDataType(reflect.TypeFor[SchOv](), "string", "ov"))

	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"slice of time", reflect.TypeFor[[]time.Time](), `{"type":"array","items":{"type":"string","format":"date-time"}}`},
		{"map of uuid", reflect.TypeFor[map[string]uuid.UUID](), `{"type":"object","additionalProperties":{"type":"string","format":"uuid"}}`},
		{"slice of override", reflect.TypeFor[[]SchOv](), `{"type":"array","items":{"type":"string","format":"ov"}}`},
		{"map of override", reflect.TypeFor[map[string]SchOv](), `{"type":"object","additionalProperties":{"type":"string","format":"ov"}}`},
		{"slice of pointer", reflect.TypeFor[[]*string](), `{"type":"array","items":{"type":["string","null"]}}`},
		{"array", reflect.TypeFor[[2]int](), `{"type":"array","items":{"type":"integer","format":"int64"},"minItems":2,"maxItems":2}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.JSONEq(t, tt.want, schemaJSON(t, g.newSchemaFromType(tt.typ)))
		})
	}
	assert.Empty(t, g.Errors())
}

func TestSchemaForSelfRecursiveNamedComposites(t *testing.T) {
	g := gen(t)

	s := g.newSchemaFromType(reflect.TypeFor[SchL]())
	assert.Equal(t, componentsSchemaPath+"SchL", s.Ref)
	assert.JSONEq(t, `{"type":"array","items":{"$ref":"#/components/schemas/SchL"}}`,
		schemaJSON(t, g.api.Components.Schemas["SchL"]))

	s = g.newSchemaFromType(reflect.TypeFor[SchM]())
	assert.Equal(t, componentsSchemaPath+"SchM", s.Ref)
	assert.JSONEq(t, `{"type":"object","additionalProperties":{"$ref":"#/components/schemas/SchM"}}`,
		schemaJSON(t, g.api.Components.Schemas["SchM"]))

	assert.Empty(t, g.stack)
	assert.Empty(t, g.Errors())
}

func TestSchemaForNonRecursiveNamedStaysInline(t *testing.T) {
	g := gen(t)

	s := g.newSchemaFromType(reflect.TypeFor[SchPlain]())
	assert.JSONEq(t, `{"type":"array","items":{"type":"string"}}`, schemaJSON(t, s))
	assert.Empty(t, g.api.Components.Schemas)
}

func TestSchemaForMutualRecursion(t *testing.T) {
	wantA := `{"type":"array","items":{"$ref":"#/components/schemas/SchB"}}`
	wantB := `{"type":"object","additionalProperties":{"$ref":"#/components/schemas/SchA"}}`

	// Every member of the cycle is promoted, whatever the entry type
	// and registration order.
	for _, order := range [][]reflect.Type{
		{reflect.TypeFor[SchA](), reflect.TypeFor[SchB]()},
		{reflect.TypeFor[SchB](), reflect.TypeFor[SchA]()},
	} {
		g := gen(t)
		for _, typ := range order {
			s := g.newSchemaFromType(typ)
			assert.Equal(t, componentsSchemaPath+typ.Name(), s.Ref)
		}
		assert.Len(t, g.api.Components.Schemas, 2)
		assert.JSONEq(t, wantA, schemaJSON(t, g.api.Components.Schemas["SchA"]))
		assert.JSONEq(t, wantB, schemaJSON(t, g.api.Components.Schemas["SchB"]))
		assert.Empty(t, g.stack)
		assert.Empty(t, g.Errors())
	}
}

type (
	SchZ       map[string]*SchYStruct
	SchYStruct struct{ K *SchZ }
)

func TestSchemaForCycleThroughStructNotPromoted(t *testing.T) {
	const nullYRef = `{"anyOf":[{"$ref":"#/components/schemas/SchYStruct"},{"type":"null"}]}`
	g := gen(t)

	s := g.newSchemaFromType(reflect.TypeFor[SchZ]())
	assert.JSONEq(t, `{"type":"object","additionalProperties":`+nullYRef+`}`, schemaJSON(t, s))
	assert.NotContains(t, g.api.Components.Schemas, "SchZ")
	assert.JSONEq(t,
		`{"type":"object","properties":{"K":{"type":["object","null"],"additionalProperties":`+nullYRef+`}}}`,
		schemaJSON(t, g.api.Components.Schemas["SchYStruct"]))
}

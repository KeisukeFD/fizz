package openapi

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ifaceShared = &Schema{
	Type:       Types("object"),
	Properties: map[string]*Schema{"a": {Type: Types("string")}},
	Enum:       []any{map[string]any{"k": "v"}},
}

type (
	ifValProv  struct{ X int }
	ifPtrProv  struct{ X int }
	ifNilProv  struct{ X int }
	ifSharedPv struct{ X int }

	ifValCust  struct{ X int }
	ifPtrCust  struct{ X int }
	ifFailCust struct{ X int }
	ifOrder    string
	ifValEnum  string
	ifPtrEnum  int
	ifU8Enum   uint8
	ifF32Enum  float32
	ifBadEnum  int8
	ifStrEnum  string
	ifPlain    string

	ifHolder struct {
		A ifValProv   `json:"a"`
		B ifPtrProv   `json:"b"`
		C ifValEnum   `json:"c"`
		D ifPtrEnum   `json:"d"`
		E ifPlain     `json:"e"`
		F []ifValEnum `json:"f"`
	}
)

func (ifValProv) OpenAPISchema() *Schema  { return &Schema{Type: Types("string"), Format: "val"} }
func (*ifPtrProv) OpenAPISchema() *Schema { return &Schema{Type: Types("string"), Format: "ptr"} }
func (ifNilProv) OpenAPISchema() *Schema  { return nil }
func (ifSharedPv) OpenAPISchema() *Schema { return ifaceShared }
func (ifValCust) CustomizeSchema(s *Schema) error {
	s.Description = "val"
	return nil
}
func (*ifPtrCust) CustomizeSchema(s *Schema) error {
	s.Description = "ptr"
	return nil
}
func (ifFailCust) CustomizeSchema(s *Schema) error {
	s.Description = "lost"
	s.Properties["X"].Description = "lost"
	return errors.New("boom")
}

func (ifOrder) Enum() []any { return []any{"a", "b"} }
func (ifOrder) CustomizeSchema(s *Schema) error {
	// The Enumer has run: the customizer can edit the enum.
	s.Enum = append(s.Enum, "c")
	return nil
}
func (ifValEnum) Enum() []any  { return []any{"x", ifValEnum("y")} }
func (*ifPtrEnum) Enum() []any { return []any{1, int8(2), 3.0} }
func (ifU8Enum) Enum() []any   { return []any{1, 300} }
func (ifF32Enum) Enum() []any  { return []any{0.1} }
func (ifBadEnum) Enum() []any  { return []any{"a", 1.5, -1, true, nil} }
func (ifStrEnum) Enum() []any  { return []any{"a", 1} }

func comp(t *testing.T, g *Generator, name string) string {
	t.Helper()
	s, ok := g.api.Components.Schemas[name]
	require.True(t, ok, "component %s", name)
	return schemaJSON(t, s)
}

func TestInterfaceReceivers(t *testing.T) {
	g := gen(t)
	s := g.newSchemaFromType(reflect.TypeFor[ifHolder]())
	require.NotNil(t, s)
	assert.Empty(t, g.Errors())

	assert.JSONEq(t, `{"type":"string","format":"val"}`, comp(t, g, "IfValProv"))
	assert.JSONEq(t, `{"type":"string","format":"ptr"}`, comp(t, g, "IfPtrProv"))
	assert.JSONEq(t, `{"type":"string","enum":["x","y"]}`, comp(t, g, "IfValEnum"))
	assert.JSONEq(t, `{"type":"integer","format":"int64","enum":[1,2,3]}`, comp(t, g, "IfPtrEnum"))

	props := g.api.Components.Schemas["IfHolder"].Properties
	assert.JSONEq(t, `{"$ref":"#/components/schemas/IfValEnum"}`, schemaJSON(t, props["c"]))
	// A plain named scalar stays inline.
	assert.JSONEq(t, `{"type":"string"}`, schemaJSON(t, props["e"]))
	assert.NotContains(t, g.api.Components.Schemas, "IfPlain")
	assert.JSONEq(t, `{"type":"array","items":{"$ref":"#/components/schemas/IfValEnum"}}`, schemaJSON(t, props["f"]))
}

func TestCustomizerReceivers(t *testing.T) {
	g := gen(t)
	g.newSchemaFromType(reflect.TypeFor[ifValCust]())
	g.newSchemaFromType(reflect.TypeFor[ifPtrCust]())
	assert.Empty(t, g.Errors())
	assert.Equal(t, "val", g.api.Components.Schemas["IfValCust"].Description)
	assert.Equal(t, "ptr", g.api.Components.Schemas["IfPtrCust"].Description)
}

func TestProviderNilFallsBack(t *testing.T) {
	g := gen(t)
	g.newSchemaFromType(reflect.TypeFor[ifNilProv]())
	assert.JSONEq(t, `{"type":"object","properties":{"X":{"type":"integer","format":"int64"}}}`, comp(t, g, "IfNilProv"))
	require.Len(t, g.Errors(), 1)
	var te *TypeError
	require.ErrorAs(t, g.Errors()[0], &te)
	assert.Equal(t, reflect.TypeFor[ifNilProv](), te.Type)
}

func TestProviderSharedPointerIsCopied(t *testing.T) {
	g := gen(t)
	g.newSchemaFromType(reflect.TypeFor[ifSharedPv]())
	got := g.api.Components.Schemas["IfSharedPv"]
	assert.NotSame(t, ifaceShared, got)
	// Mutating the component leaves the user's schema untouched.
	got.Properties["a"].Description = "changed"
	got.Enum[0].(map[string]any)["k"] = "changed"
	assert.Empty(t, ifaceShared.Properties["a"].Description)
	assert.Equal(t, "v", ifaceShared.Enum[0].(map[string]any)["k"])
}

func TestCustomizerErrorKeepsPreCustomize(t *testing.T) {
	g := gen(t)
	g.newSchemaFromType(reflect.TypeFor[ifFailCust]())
	assert.JSONEq(t, `{"type":"object","properties":{"X":{"type":"integer","format":"int64"}}}`, comp(t, g, "IfFailCust"))
	require.Len(t, g.Errors(), 1)
	var te *TypeError
	require.ErrorAs(t, g.Errors()[0], &te)
	assert.Contains(t, te.Message, "boom")
}

func TestEnumerBeforeCustomizer(t *testing.T) {
	g := gen(t)
	g.newSchemaFromType(reflect.TypeFor[ifOrder]())
	assert.JSONEq(t, `{"type":"string","enum":["a","b","c"]}`, comp(t, g, "IfOrder"))
}

func TestEnumerConversion(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
		errs int
	}{
		{"uint8 overflow", reflect.TypeFor[ifU8Enum](), `{"type":"integer","format":"uint8","minimum":0}`, 1},
		{"float32 literal", reflect.TypeFor[ifF32Enum](), `{"type":"number","format":"float","enum":[0.1]}`, 0},
		{"invalid values (-1 is valid)", reflect.TypeFor[ifBadEnum](), `{"type":"integer","format":"int8"}`, 4},
		{"kind mismatch", reflect.TypeFor[ifStrEnum](), `{"type":"string"}`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gen(t)
			g.newSchemaFromType(tt.typ)
			assert.Len(t, g.Errors(), tt.errs)
			name := g.typeName(tt.typ)
			assert.JSONEq(t, tt.want, comp(t, g, name))
		})
	}
}

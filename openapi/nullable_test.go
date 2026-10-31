package openapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	nlStruct  struct{ V string }
	nlValNull string
	nlPtrNull string
	nlNoNull  int
	nlSelf    struct {
		Next *nlSelf `json:"next" description:"the next one"`
	}
	nlHolder struct {
		P  *nlStruct  `json:"p" description:"desc"`
		PP **nlStruct `json:"pp"`
		I  **int      `json:"i"`
		S  *[]string  `json:"s"`
		A  any        `json:"a"`
		PA *any       `json:"pa"`
		V  nlValNull  `json:"v"`
		R  nlPtrNull  `json:"r"`
		N  *nlNoNull  `json:"n"`
		E  *[]string  `json:"e" enum:"a,b"`
	}
)

func (nlValNull) Nullable() bool  { return true }
func (*nlPtrNull) Nullable() bool { return true }
func (nlNoNull) Nullable() bool   { return false }

func TestNullablePointers(t *testing.T) {
	ref := `{"$ref":"#/components/schemas/NlStruct"}`
	tests := []struct {
		name string
		want string
	}{
		{"p", `{"anyOf":[` + ref + `,{"type":"null"}],"description":"desc"}`},
		{"pp", `{"anyOf":[` + ref + `,{"type":"null"}]}`},
		{"i", `{"type":["integer","null"],"format":"int64"}`},
		{"s", `{"type":["array","null"],"items":{"type":"string","enum":["a","b"]}}`},
		{"a", `{}`},
		{"pa", `{}`},
		{"v", `{"type":["string","null"]}`},
		{"r", `{"type":["string","null"]}`},
		{"n", `{"type":"integer","format":"int64"}`},
	}
	g := gen(t)
	root := g.newSchemaFromType(reflect.TypeFor[nlHolder]())
	require.NotNil(t, root)
	props := g.api.Components.Schemas["NlHolder"].Properties
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schemaJSON(t, props[tt.name])
			if tt.name == "s" {
				got = schemaJSON(t, g.newSchemaFromType(reflect.TypeFor[*[]string]()))
				tt.want = `{"type":["array","null"],"items":{"type":"string"}}`
			}
			assert.JSONEq(t, tt.want, got)
		})
	}
	assert.Equal(t, `{"type":"object","properties":{"V":{"type":"string"}}}`,
		schemaJSON(t, g.api.Components.Schemas["NlStruct"]), "component untouched")
}

func TestNullablePointerEnumOnArray(t *testing.T) {
	g := gen(t)
	g.newSchemaFromType(reflect.TypeFor[nlHolder]())
	got := g.api.Components.Schemas["NlHolder"].Properties["e"]
	assert.Equal(t, Types("array", "null"), got.Type)
	assert.Equal(t, []any{"a", "b"}, got.Items.Enum)
}

func TestNullableSelfReference(t *testing.T) {
	g := gen(t)
	g.newSchemaFromType(reflect.TypeFor[nlSelf]())
	assert.JSONEq(t,
		`{"anyOf":[{"$ref":"#/components/schemas/NlSelf"},{"type":"null"}],"description":"the next one"}`,
		schemaJSON(t, g.api.Components.Schemas["NlSelf"].Properties["next"]))
}

func TestSetPointersAreNullable(t *testing.T) {
	g := gen(t)
	g.SetPointersAreNullable(false)
	assert.JSONEq(t, `{"type":"integer","format":"int32"}`,
		schemaJSON(t, g.newSchemaFromType(reflect.TypeFor[*int32]())))
	assert.JSONEq(t, `{"$ref":"#/components/schemas/NlStruct"}`,
		schemaJSON(t, g.newSchemaFromType(reflect.TypeFor[*nlStruct]())))
	// The Nullable interface still applies.
	assert.JSONEq(t, `{"type":["string","null"]}`,
		schemaJSON(t, g.newSchemaFromType(reflect.TypeFor[*nlValNull]())))
	assert.JSONEq(t, `{"type":["string","null"]}`,
		schemaJSON(t, g.newSchemaFromType(reflect.TypeFor[nlPtrNull]())))
}

func TestSchemaFromModelPointerNotNullable(t *testing.T) {
	g := gen(t)
	assert.JSONEq(t, `{"$ref":"#/components/schemas/NlStruct"}`,
		schemaJSON(t, g.newSchemaFromModel(reflect.TypeFor[*nlStruct]())))
}

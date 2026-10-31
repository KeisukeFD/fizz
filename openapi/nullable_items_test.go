package openapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nlItemsHolder struct {
	E []*string          `json:"e" enum:"a,b"`
	V []string           `json:"v" enum:"a,b"`
	D []*string          `json:"d" validate:"dive,oneof=x y"`
	C []*string          `json:"c" validate:"dive,eq=k"`
	M map[string]*string `json:"m" validate:"dive,oneof=p q"`
}

// The enum and const of nullable items admit the null the items accept,
// as they do for a nullable top-level schema.
func TestNullableItemsEnumAndConst(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"e", `{"type":"array","items":{"type":["string","null"],"enum":["a","b",null]}}`},
		{"v", `{"type":"array","items":{"type":"string","enum":["a","b"]}}`},
		{"d", `{"type":"array","items":{"type":["string","null"],"enum":["x","y",null]}}`},
		{"c", `{"type":"array","items":{"type":["string","null"],"enum":["k",null]}}`},
		{"m", `{"type":"object","additionalProperties":{"type":["string","null"],"enum":["p","q",null]}}`},
	}
	g := gen(t)
	require.NotNil(t, g.newSchemaFromType(reflect.TypeFor[nlItemsHolder]()))
	props := g.api.Components.Schemas["NlItemsHolder"].Properties
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.JSONEq(t, tt.want, schemaJSON(t, props[tt.name]))
		})
	}
}

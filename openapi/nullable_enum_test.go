package openapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nlEnumHolder struct {
	E *string `json:"e" enum:"a,b"`
	O *string `json:"o" validate:"oneof=x y"`
	C *string `json:"c" validate:"eq=k"`
	N *int    `json:"n" validate:"eq=3"`
	V string  `json:"v" enum:"a,b"`
	F *string `json:"f" enum:"a,b" nullable:"false"`
}

func TestNullableEnumAndConst(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"e", `{"type":["string","null"],"enum":["a","b",null]}`},
		{"o", `{"type":["string","null"],"enum":["x","y",null]}`},
		{"c", `{"type":["string","null"],"enum":["k",null]}`},
		{"n", `{"type":["integer","null"],"format":"int64","enum":[3,null]}`},
		{"v", `{"type":"string","enum":["a","b"]}`},
		{"f", `{"type":"string","enum":["a","b"]}`},
	}
	g := gen(t)
	require.NotNil(t, g.newSchemaFromType(reflect.TypeFor[nlEnumHolder]()))
	props := g.api.Components.Schemas["NlEnumHolder"].Properties
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.JSONEq(t, tt.want, schemaJSON(t, props[tt.name]))
		})
	}
}

func TestMakeNullableEnumConstIdempotent(t *testing.T) {
	s := &Schema{Type: Types("string"), Enum: []any{"a"}}
	s.MakeNullable()
	s.MakeNullable()
	assert.Equal(t, []any{"a", nil}, s.Enum)
	c := &Schema{Type: Types("string"), Const: "k"}
	c.MakeNullable()
	c.MakeNullable()
	assert.Equal(t, []any{"k", nil}, c.Enum)
	assert.Nil(t, c.Const)
	assert.Equal(t, Types("string", "null"), c.Type)
}

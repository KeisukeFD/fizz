package openapi

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	valPtr struct {
		N *int `json:"n" default:"10"`
	}
	valBadBool struct {
		B bool `json:"b" default:"maybe"`
	}
	valBadBoolParam struct {
		B bool `query:"b" default:"maybe"`
	}
	valDurParam struct {
		D time.Duration `query:"d" default:"1s"`
	}
	valDurParamNS struct {
		D time.Duration `query:"d" default:"1000000000" enum:"1000000000,2000000000"`
	}
	valDurParamEnum struct {
		D time.Duration `query:"d" enum:"1s"`
	}
	valDurBody struct {
		A time.Duration  `json:"a" default:"1s" enum:"1s, 2000000000"`
		B *time.Duration `json:"b" default:"5"`
	}
	valDefReqParam struct {
		N int `query:"n" default:"3" validate:"required"`
	}
	valDefReqPath struct {
		ID int `path:"id" default:"3" validate:"required"`
	}
	valDefReqBody struct {
		N int `json:"n" default:"3" validate:"required"`
	}
	valFloat struct {
		F  float32  `json:"f" default:"0.1" enum:"0.1, 0.2" example:"0.3"`
		FP *float32 `json:"fp" default:"1e3"`
		D  float64  `json:"d" default:"0.1"`
		I  float64  `json:"i" default:"Inf"`
	}
	valEnumTrim struct {
		S string   `json:"s" enum:"a, b"`
		L []string `json:"l" enum:" x ,y "`
	}
	valQuoted struct {
		I int     `json:"i,string" default:"7" enum:"1, 2" example:"3"`
		B bool    `json:"b,string" default:"true"`
		F float32 `json:"f,string" default:"0.1"`
		S string  `json:"s,string" default:"x"`
		P *int64  `json:"p,string" default:"9"`
		U int     `json:"u" default:"7"`
	}
)

// bodyProperties returns the properties of the request body that
// buildParams adds for typ.
func bodyProperties(t *testing.T, g *Generator, typ reflect.Type) *Schema {
	t.Helper()
	op := &Operation{}
	require.NoError(t, g.buildParams(op, typ, true))
	require.NotNil(t, op.RequestBody)
	for _, mt := range op.RequestBody.Content {
		return mt.Schema
	}
	t.Fatal("no body schema")
	return nil
}

func paramSchemaOf(t *testing.T, g *Generator, typ reflect.Type) *Parameter {
	t.Helper()
	op := &Operation{}
	require.NoError(t, g.buildParams(op, typ, false))
	require.Len(t, op.Parameters, 1)
	return op.Parameters[0].Value
}

func errorsContain(g *Generator, target any) bool {
	for _, e := range g.Errors() {
		if errors.As(e, target) {
			return true
		}
	}
	return false
}

func TestDefaultDereferencesPointer(t *testing.T) {
	g := gen(t)
	s := bodyProperties(t, g, reflect.TypeFor[valPtr]())
	assert.Empty(t, g.Errors())
	assert.Equal(t, int64(10), s.Properties["n"].Default)
}

func TestInvalidBoolDefaultIsError(t *testing.T) {
	_, err := convertValue("maybe", reflect.TypeFor[bool](), false)
	assert.Error(t, err)

	g := gen(t)
	s := bodyProperties(t, g, reflect.TypeFor[valBadBool]())
	var fe *FieldError
	assert.True(t, errorsContain(g, &fe))
	assert.Nil(t, s.Properties["b"].Default)

	g = gen(t)
	p := paramSchemaOf(t, g, reflect.TypeFor[valBadBoolParam]())
	assert.True(t, errorsContain(g, &fe))
	assert.Nil(t, p.Schema.Default)
}

func TestDurationParamAcceptsOnlyNanoseconds(t *testing.T) {
	g := gen(t)
	p := paramSchemaOf(t, g, reflect.TypeFor[valDurParam]())
	var fe *FieldError
	assert.True(t, errorsContain(g, &fe), "%v", g.Errors())
	assert.Nil(t, p.Schema.Default)

	g = gen(t)
	p = paramSchemaOf(t, g, reflect.TypeFor[valDurParamNS]())
	assert.Empty(t, g.Errors())
	assert.Equal(t, int64(1e9), p.Schema.Default)
	assert.Equal(t, []any{int64(1e9), int64(2e9)}, p.Schema.Enum)

	g = gen(t)
	p = paramSchemaOf(t, g, reflect.TypeFor[valDurParamEnum]())
	assert.True(t, errorsContain(g, &fe))
	assert.Empty(t, p.Schema.Enum)
}

func TestDurationBodyAcceptsStringAndInteger(t *testing.T) {
	g := gen(t)
	s := bodyProperties(t, g, reflect.TypeFor[valDurBody]())
	assert.Empty(t, g.Errors())
	a := s.Properties["a"]
	assert.Equal(t, int64(time.Second), a.Default)
	assert.Equal(t, []any{int64(time.Second), int64(2 * time.Second)}, a.Enum)
	assert.Equal(t, int64(5), s.Properties["b"].Default)
	for _, v := range append([]any{a.Default, s.Properties["b"].Default}, a.Enum...) {
		assert.IsType(t, int64(0), v, "a time.Duration must never be stored in an any")
	}
}

func TestDefaultWithRequiredParamDropsRequired(t *testing.T) {
	g := gen(t)
	p := paramSchemaOf(t, g, reflect.TypeFor[valDefReqParam]())
	assert.False(t, p.Required)
	assert.Equal(t, int64(3), p.Schema.Default)
	var w *Warning
	assert.True(t, errorsContain(g, &w), "%v", g.Errors())
	var fe *FieldError
	assert.False(t, errorsContain(g, &fe))
}

func TestDefaultWithRequiredPathStaysRequired(t *testing.T) {
	g := gen(t)
	p := paramSchemaOf(t, g, reflect.TypeFor[valDefReqPath]())
	assert.True(t, p.Required)
	assert.Equal(t, int64(3), p.Schema.Default)
	var w *Warning
	assert.False(t, errorsContain(g, &w))
}

func TestDefaultWithRequiredBodyKeepsRequired(t *testing.T) {
	g := gen(t)
	s := bodyProperties(t, g, reflect.TypeFor[valDefReqBody]())
	assert.Equal(t, []string{"n"}, s.Required)
	assert.Nil(t, s.Properties["n"].Default)
	var w *Warning
	assert.True(t, errorsContain(g, &w), "%v", g.Errors())
	var fe *FieldError
	assert.False(t, errorsContain(g, &fe))
}

func TestFloat32LiteralIsNotWidened(t *testing.T) {
	g := gen(t)
	s := bodyProperties(t, g, reflect.TypeFor[valFloat]())
	f := s.Properties["f"]
	assert.Equal(t, json.Number("0.1"), f.Default)
	assert.Equal(t, []any{json.Number("0.1"), json.Number("0.2")}, f.Enum)
	assert.Equal(t, []any{json.Number("0.3")}, f.Examples)
	assert.Equal(t, json.Number("1e3"), s.Properties["fp"].Default)
	assert.Equal(t, 0.1, s.Properties["d"].Default)
	// Not a finite number.
	assert.Nil(t, s.Properties["i"].Default)
	assert.NotEmpty(t, g.Errors())

	b, err := json.Marshal(f)
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"number","format":"float","default":0.1,"enum":[0.1,0.2],"examples":[0.3]}`, string(b))
}

func TestEnumValuesAreTrimmed(t *testing.T) {
	g := gen(t)
	s := bodyProperties(t, g, reflect.TypeFor[valEnumTrim]())
	assert.Empty(t, g.Errors())
	assert.Equal(t, []any{"a", "b"}, s.Properties["s"].Enum)
	assert.Equal(t, []any{"x", "y"}, s.Properties["l"].Items.Enum)
}

func TestValuesFollowStringWireForm(t *testing.T) {
	g := gen(t)
	s := bodyProperties(t, g, reflect.TypeFor[valQuoted]())
	assert.Empty(t, g.Errors())
	i := s.Properties["i"]
	assert.Equal(t, "7", i.Default)
	assert.Equal(t, []any{"1", "2"}, i.Enum)
	assert.Equal(t, []any{"3"}, i.Examples)
	assert.Equal(t, "true", s.Properties["b"].Default)
	assert.Equal(t, "0.1", s.Properties["f"].Default)
	assert.Equal(t, `"x"`, s.Properties["s"].Default)
	assert.Equal(t, "9", s.Properties["p"].Default)
	assert.Equal(t, int64(7), s.Properties["u"].Default)
}

func TestExampleTagBecomesExamples(t *testing.T) {
	type ex struct {
		N int `json:"n" example:" 5 "`
	}
	g := gen(t)
	s := bodyProperties(t, g, reflect.TypeFor[ex]())
	assert.Equal(t, []any{int64(5)}, s.Properties["n"].Examples)
}

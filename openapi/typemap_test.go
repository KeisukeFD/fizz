package openapi

import (
	"encoding/json"
	"errors"
	"math/big"
	"net"
	"net/netip"
	"net/url"
	"reflect"
	"testing"
	"time"
	stduuid "uuid"

	gofrs "github.com/gofrs/uuid"
	google "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	tmText    struct{ v string }
	tmJSON    struct{ v int }
	tmBoth    struct{ v string }
	tmPtrText struct{ v string }
	tmPtrJSON struct{ v int }
	tmKeyText struct{ v string }
	tmKeyStr  string
	tmKeyInt  int
	tmKeyUint uint8
	tmPtrKey  struct{ v string }
	tmHolder  struct{ V tmJSON }
	tmUnsup   struct {
		OK string
		Ch chan int
		Fn func()
		Cx complex128
		Er error
		Up uintptr
	}
)

func (tmText) MarshalText() ([]byte, error)     { return nil, nil }
func (tmJSON) MarshalJSON() ([]byte, error)     { return nil, nil }
func (tmBoth) MarshalText() ([]byte, error)     { return nil, nil }
func (tmBoth) MarshalJSON() ([]byte, error)     { return nil, nil }
func (*tmPtrText) MarshalText() ([]byte, error) { return nil, nil }
func (*tmPtrJSON) MarshalJSON() ([]byte, error) { return nil, nil }
func (tmKeyText) MarshalText() ([]byte, error)  { return nil, nil }
func (tmKeyStr) MarshalText() ([]byte, error)   { return nil, nil }
func (tmKeyInt) MarshalText() ([]byte, error)   { return nil, nil }
func (*tmPtrKey) MarshalText() ([]byte, error)  { return nil, nil }

func TestKnownTypes(t *testing.T) {
	tests := []struct {
		typ  reflect.Type
		want string
	}{
		{reflect.TypeFor[int](), `{"type":"integer","format":"int64"}`},
		{reflect.TypeFor[int64](), `{"type":"integer","format":"int64"}`},
		{reflect.TypeFor[int32](), `{"type":"integer","format":"int32"}`},
		{reflect.TypeFor[int16](), `{"type":"integer","format":"int16"}`},
		{reflect.TypeFor[int8](), `{"type":"integer","format":"int8"}`},
		{reflect.TypeFor[uint](), `{"type":"integer","format":"uint64","minimum":0}`},
		{reflect.TypeFor[uint64](), `{"type":"integer","format":"uint64","minimum":0}`},
		{reflect.TypeFor[uint32](), `{"type":"integer","format":"uint32","minimum":0}`},
		{reflect.TypeFor[uint16](), `{"type":"integer","format":"uint16","minimum":0}`},
		{reflect.TypeFor[uint8](), `{"type":"integer","format":"uint8","minimum":0}`},
		{reflect.TypeFor[time.Duration](), `{"type":"integer","format":"int64"}`},
		{reflect.TypeFor[time.Time](), `{"type":"string","format":"date-time"}`},
		{reflect.TypeFor[json.RawMessage](), `{}`},
		{reflect.TypeFor[[]byte](), `{"type":"string","contentEncoding":"base64"}`},
		{reflect.TypeFor[stduuid.UUID](), `{"type":"string","format":"uuid"}`},
		{reflect.TypeFor[gofrs.UUID](), `{"type":"string","format":"uuid"}`},
		{reflect.TypeFor[google.UUID](), `{"type":"string","format":"uuid"}`},
		{reflect.TypeFor[net.IP](), `{"type":"string"}`},
		{reflect.TypeFor[netip.Addr](), `{"type":"string"}`},
		{reflect.TypeFor[netip.Prefix](), `{"type":"string"}`},
		{reflect.TypeFor[big.Int](), `{"type":"integer"}`},
		{reflect.TypeFor[big.Float](), `{"type":"string"}`},
		{reflect.TypeFor[big.Rat](), `{"type":"string"}`},
	}
	for _, tt := range tests {
		t.Run(tt.typ.String(), func(t *testing.T) {
			g := gen(t)
			assert.JSONEq(t, tt.want, schemaJSON(t, g.newSchemaFromType(tt.typ)))
			assert.Empty(t, g.Errors())
		})
	}
}

func TestKnownTypeURLIsObject(t *testing.T) {
	g := gen(t)
	g.UseFullSchemaNames(true)
	s := g.newSchemaFromType(reflect.TypeFor[url.URL]())
	require.NotNil(t, s)
	assert.Equal(t, componentsSchemaPath+"UrlURL", s.Ref)
	assert.Equal(t, Types("object"), g.api.Components.Schemas["UrlURL"].Type)
}

func TestMarshalerPolicy(t *testing.T) {
	str := `{"type":"string"}`
	tests := []struct {
		name string
		typ  reflect.Type
		want string // JSON of a scalar schema; empty for a component
	}{
		{"text only", reflect.TypeFor[tmText](), str},
		{"both", reflect.TypeFor[tmBoth](), str},
		{"pointer receiver text, value type", reflect.TypeFor[tmPtrText](), str},
		{"pointer receiver text, pointer type", reflect.TypeFor[*tmPtrText](), `{"type":["string","null"]}`},
		{"json only", reflect.TypeFor[tmJSON](), ""},
		{"pointer receiver json only", reflect.TypeFor[tmPtrJSON](), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gen(t)
			s := g.newSchemaFromType(tt.typ)
			require.NotNil(t, s)
			warnings := 0
			if tt.want != "" {
				assert.JSONEq(t, tt.want, schemaJSON(t, s))
			} else {
				// Kind-derived schema: a struct component.
				require.NotEmpty(t, s.Ref)
				comp := g.api.Components.Schemas[s.Ref[len(componentsSchemaPath):]]
				require.NotNil(t, comp)
				assert.Equal(t, Types("object"), comp.Type)
				warnings = 1
			}
			require.Len(t, g.Errors(), warnings)
			for _, err := range g.Errors() {
				var w *Warning
				require.True(t, errors.As(err, &w))
				assert.True(t, w.IsWarning())
				assert.Contains(t, w.Error(), "SchemaProvider")
				assert.Empty(t, w.Method+w.Path+w.ID)
			}
		})
	}
}

func TestMarshalerWarningOncePerType(t *testing.T) {
	g := gen(t)
	g.newSchemaFromType(reflect.TypeFor[[]tmJSON]())
	g.newSchemaFromType(reflect.TypeFor[map[string]tmJSON]())
	g.newSchemaFromType(reflect.TypeFor[tmHolder]())
	assert.Len(t, g.Errors(), 1)
}

func TestMarshalerWarningCarriesOperation(t *testing.T) {
	g := gen(t)
	_, err := g.AddOperation("/x", "POST", "", nil, reflect.TypeFor[tmHolder](), &OperationInfo{ID: "doX", StatusCode: 200})
	require.NoError(t, err)
	require.Len(t, g.Errors(), 1)
	var w *Warning
	require.True(t, errors.As(g.Errors()[0], &w))
	assert.Equal(t, "POST", w.Method)
	assert.Equal(t, "/x", w.Path)
	assert.Equal(t, "doX", w.ID)
	assert.Contains(t, w.Error(), "POST /x (doX)")
	assert.Nil(t, g.cur)
}

func TestWarning(t *testing.T) {
	var err error = &Warning{Msg: "m"}
	assert.Equal(t, "warning: m", err.Error())
	w := &Warning{Method: "GET", Path: "/p", ID: "i", Msg: "m"}
	assert.Equal(t, "warning: GET /p (i): m", w.Error())
	assert.True(t, w.IsWarning())
}

func TestMapKeys(t *testing.T) {
	obj := func(extra string) string {
		return `{"type":"object","additionalProperties":{"type":"string"}` + extra + `}`
	}
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"string", reflect.TypeFor[map[string]string](), obj("")},
		{"string kind beats TextMarshaler", reflect.TypeFor[map[tmKeyStr]string](), obj("")},
		{"TextMarshaler", reflect.TypeFor[map[tmKeyText]string](), obj("")},
		{"TextMarshaler beats integer kind", reflect.TypeFor[map[tmKeyInt]string](), obj("")},
		{"pointer receiver TextMarshaler", reflect.TypeFor[map[tmPtrKey]string](), obj("")},
		{"int", reflect.TypeFor[map[int]string](), obj(`,"propertyNames":{"pattern":"^-?[0-9]+$"}`)},
		{"int8", reflect.TypeFor[map[int8]string](), obj(`,"propertyNames":{"pattern":"^-?[0-9]+$"}`)},
		{"uint", reflect.TypeFor[map[tmKeyUint]string](), obj(`,"propertyNames":{"pattern":"^[0-9]+$"}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gen(t)
			assert.JSONEq(t, tt.want, schemaJSON(t, g.newSchemaFromType(tt.typ)))
			assert.Empty(t, g.Errors())
		})
	}
	for _, typ := range []reflect.Type{
		reflect.TypeFor[map[float64]string](),
		reflect.TypeFor[map[bool]string](),
		reflect.TypeFor[map[[2]int]string](),
		reflect.TypeFor[map[any]string](),
	} {
		t.Run("unsupported "+typ.String(), func(t *testing.T) {
			g := gen(t)
			assert.Nil(t, g.newSchemaFromType(typ))
			require.Len(t, g.Errors(), 1)
			var te *TypeError
			assert.True(t, errors.As(g.Errors()[0], &te))
		})
	}
}

func TestUnsupportedKindsOmitted(t *testing.T) {
	g := gen(t)
	s := g.newSchemaFromType(reflect.TypeFor[tmUnsup]())
	require.NotNil(t, s)
	comp := g.api.Components.Schemas[s.Ref[len(componentsSchemaPath):]]
	require.NotNil(t, comp)
	require.Len(t, comp.Properties, 1)
	assert.Contains(t, comp.Properties, "OK")
	assert.Len(t, g.Errors(), 5)
}

func TestUnsupportedBodyFieldKeepsOperation(t *testing.T) {
	type in struct {
		C chan int `json:"c"`
		N string   `json:"n"`
	}
	g := gen(t)
	op, err := g.AddOperation("/c", "POST", "", reflect.TypeFor[in](), nil, &OperationInfo{ID: "c", StatusCode: 200})
	require.NoError(t, err)
	require.NotNil(t, op)
	assert.NotEmpty(t, g.Errors())
}

func TestDurationDefaultIsNanoseconds(t *testing.T) {
	v, err := stringToType("1000000000", reflect.TypeFor[time.Duration]())
	require.NoError(t, err)
	assert.Equal(t, int64(time.Second), v)
	v, err = stringToType("2s", reflect.TypeFor[time.Duration]())
	require.NoError(t, err)
	assert.Equal(t, int64(2*time.Second), v)
}

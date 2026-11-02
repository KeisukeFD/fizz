package openapi

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KeisukeFD/fizz/internal/golden"
)

// TestSchemaValidation tests that the validator.v9
// tags are properly translated to JSON Schema validation
// fields for all supported types.
func TestSchemaValidation(t *testing.T) {
	type T struct {
		A string            `validate:"len=12"`
		B int               `validate:"min=5,max=100"`
		C []bool            `validate:"len=50"`
		D map[string]string `validate:"eq=5"`
		E string            `validate:"eq=15"`
		F string            `validate:"lt=8"`
		G string            `validate:"gt=6"`
		H int               `validate:"len=foo"`    // ignored, cannot convert "foo" to integer
		I []string          `validate:"dive,len=6"` // applies to the items
		J map[string]string `validate:"keys,max=6"` // ignored, malformed: keys must follow dive
	}
	g := gen(t)

	sor := g.newSchemaFromType(rt(T{}))
	assert.NotNil(t, sor)
	schema := g.resolveSchema(sor)
	assert.NotNil(t, schema)

	actual, err := json.Marshal(schema)
	if err != nil {
		t.Error(err)
	}
	golden.Assert(t, "../testdata/schemas/validation.json", actual)
}

// fieldSchemaJSON returns the generated schema of a single field of
// type typ carrying the given validate tag, and the generator errors.
func fieldSchemaJSON(t *testing.T, typ reflect.Type, tag string) (string, []error) {
	t.Helper()
	st := reflect.StructOf([]reflect.StructField{{
		Name: "F", Type: typ,
		Tag: reflect.StructTag(`json:"f" validate:"` + tag + `"`),
	}})
	g := gen(t)
	sor := g.newSchemaFromType(st)
	require.NotNil(t, sor)
	s := g.resolveSchema(sor)
	require.NotNil(t, s)
	b, err := json.Marshal(s.Properties["f"])
	require.NoError(t, err)
	// Drop the numeric type formats: they are not under test.
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	dropNumFormats(m)
	b, err = json.Marshal(m)
	require.NoError(t, err)
	return string(b), g.Errors()
}

func dropNumFormats(v any) {
	switch x := v.(type) {
	case map[string]any:
		switch x["format"] {
		case "int32", "int64", "uint8", "double", "float":
			delete(x, "format")
		}
		for _, c := range x {
			dropNumFormats(c)
		}
	case []any:
		for _, c := range x {
			dropNumFormats(c)
		}
	}
}

func TestRuleKeywordMapping(t *testing.T) {
	var (
		str   = reflect.TypeFor[string]()
		i     = reflect.TypeFor[int]()
		u     = reflect.TypeFor[uint8]()
		f     = reflect.TypeFor[float64]()
		d     = reflect.TypeFor[time.Duration]()
		b     = reflect.TypeFor[bool]()
		ints  = reflect.TypeFor[[]int]()
		strs  = reflect.TypeFor[[]string]()
		m     = reflect.TypeFor[map[string]int]()
		bytes = reflect.TypeFor[[]byte]()
		ptr   = reflect.TypeFor[*string]()
	)
	tests := []struct {
		name string
		typ  reflect.Type
		tag  string
		want string // JSON of the field schema
		errs int
	}{
		{"min-zero", i, "min=0", `{"type":"integer","minimum":0}`, 0},
		{"gte-lte-decimal", f, "gte=0.5,lte=1.5", `{"type":"number","minimum":0.5,"maximum":1.5}`, 0},
		{"gt-lt-float", f, "gt=1.5,lt=2.5", `{"type":"number","exclusiveMinimum":1.5,"exclusiveMaximum":2.5}`, 0},
		{"gt-lt-int", i, "gt=1,lt=9", `{"type":"integer","exclusiveMinimum":1,"exclusiveMaximum":9}`, 0},
		{"uint-bounds", u, "min=1,max=200", `{"type":"integer","minimum":1,"maximum":200}`, 0},
		{"bad-number", i, "min=1.5", `{"type":"integer"}`, 1},
		{"string-length", str, "min=2,max=5", `{"type":"string","minLength":2,"maxLength":5}`, 0},
		{"string-gt-lt", str, "gt=2,lt=5", `{"type":"string","minLength":3,"maxLength":4}`, 0},
		{"string-lt-zero", str, "lt=0", `{"type":"string"}`, 1},
		{"string-len", str, "len=3", `{"type":"string","minLength":3,"maxLength":3}`, 0},
		{"slice-bounds", ints, "min=1,max=4", `{"type":"array","items":{"type":"integer"},"minItems":1,"maxItems":4}`, 0},
		{"slice-len", ints, "len=2", `{"type":"array","items":{"type":"integer"},"minItems":2,"maxItems":2}`, 0},
		{"map-bounds", m, "min=1,max=3", `{"type":"object","additionalProperties":{"type":"integer"},"minProperties":1,"maxProperties":3}`, 0},
		{"map-eq", m, "eq=2", `{"type":"object","additionalProperties":{"type":"integer"},"minProperties":2,"maxProperties":2}`, 0},
		{"bytes-no-length", bytes, "min=1,max=4", `{"type":"string","contentEncoding":"base64"}`, 0},
		{"duration-ns", d, "min=1s,max=1m", `{"type":"integer","minimum":1000000000,"maximum":60000000000}`, 0},
		{"duration-bad", d, "min=soon", `{"type":"integer"}`, 1},
		{"eq-int", i, "eq=5", `{"type":"integer","const":5}`, 0},
		{"eq-string", str, "eq=x", `{"type":"string","const":"x"}`, 0},
		{"eq-bool", b, "eq=true", `{"type":"boolean","const":true}`, 0},
		{"ne-string", str, "ne=a", `{"type":"string","not":{"const":"a"}}`, 0},
		{"oneof-string", str, "oneof=a b", `{"type":"string","enum":["a","b"]}`, 0},
		{"oneof-int", i, "oneof=1 2 3", `{"type":"integer","enum":[1,2,3]}`, 0},
		{"oneof-bad-int", i, "oneof=1 x", `{"type":"integer"}`, 1},
		{"email-keeps-max", str, "required,email,max=20", `{"type":"string","format":"email","maxLength":20}`, 0},
		{"url-uri", str, "url", `{"type":"string","format":"uri"}`, 0},
		{"uuid5", str, "uuid5", `{"type":"string","format":"uuid"}`, 0},
		{"ipv4-hostname", str, "ipv4,hostname", `{"type":"string","format":"ipv4","allOf":[{"format":"hostname"}]}`, 0},
		{"datetime-ignored", str, "datetime=2006-01-02", `{"type":"string"}`, 0},
		{"alpha", str, "alpha", `{"type":"string","pattern":"^[a-zA-Z]+$"}`, 0},
		{"hexadecimal", str, "hexadecimal", `{"type":"string","pattern":"^[0-9a-fA-F]+$"}`, 0},
		{"unique", strs, "unique", `{"type":"array","items":{"type":"string"},"uniqueItems":true}`, 0},
		{"unique-on-string-ignored", str, "unique", `{"type":"string"}`, 0},
		{"or-group", i, "min=1|max=5", `{"type":"integer","anyOf":[{"minimum":1},{"maximum":5}]}`, 0},
		{"or-group-dropped", str, "min=1|isdefault", `{"type":"string"}`, 0},
		{"or-group-const", str, "eq=a|eq=b", `{"type":"string","anyOf":[{"const":"a"},{"const":"b"}]}`, 0},
		{"tighter-min", i, "min=2,min=5,min=3", `{"type":"integer","minimum":5}`, 0},
		{"tighter-max", i, "max=9,lte=4", `{"type":"integer","maximum":4}`, 0},
		{"tighter-length", str, "max=9,max=4", `{"type":"string","maxLength":4}`, 0},
		{"no-keyword-rules", str, "omitempty,required,isdefault,structonly,nostructlevel,bogus", `{"type":"string"}`, 0},
		{"pointer", ptr, "min=2", `{"type":["string","null"],"minLength":2}`, 0},
		{"dive-items", ints, "min=1,dive,min=2", `{"type":"array","items":{"type":"integer","minimum":2},"minItems":1}`, 0},
		{"dive-nested", reflect.TypeFor[[][]string](), "dive,min=1,dive,max=3",
			`{"type":"array","items":{"type":"array","items":{"type":"string","maxLength":3},"minItems":1}}`, 0},
		{"dive-map-values", m, "dive,max=9", `{"type":"object","additionalProperties":{"type":"integer","maximum":9}}`, 0},
		{"keys", m, "dive,keys,min=2,endkeys,min=1", `{"type":"object","additionalProperties":{"type":"integer","minimum":1},"propertyNames":{"minLength":2}}`, 0},
		{"keys-only", m, "dive,keys,len=2,endkeys", `{"type":"object","additionalProperties":{"type":"integer"},"propertyNames":{"minLength":2,"maxLength":2}}`, 0},
		{"dive-on-scalar", i, "dive,min=1", `{"type":"integer"}`, 1},
		{"parse-error", i, "min=1,,max=2", `{"type":"integer"}`, 1},
		{"keys-without-dive", m, "keys,min=1,endkeys", `{"type":"object","additionalProperties":{"type":"integer"}}`, 1},
		{"skip", i, "-", `{"type":"integer"}`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := fieldSchemaJSON(t, tt.typ, tt.tag)
			assert.JSONEq(t, tt.want, got)
			assert.Len(t, errs, tt.errs)
			for _, e := range errs {
				var fe *FieldError
				if assert.ErrorAs(t, e, &fe) {
					assert.Equal(t, "F", fe.Field)
				}
			}
		})
	}
}

// TestDiveRefItemsNotMutated checks that a dive over $ref items
// annotates a copy and leaves the component alone.
func TestDiveRefItemsNotMutated(t *testing.T) {
	type Elem struct{ A int }
	st := reflect.StructOf([]reflect.StructField{{
		Name: "F", Type: reflect.TypeFor[[]Elem](), Tag: `json:"f" validate:"dive,min=1"`,
	}})
	g := gen(t)
	sor := g.newSchemaFromType(st)
	require.NotNil(t, sor)
	before, _ := json.Marshal(g.resolveSchema(g.newSchemaFromType(reflect.TypeFor[Elem]())))
	s := g.resolveSchema(sor)
	items := s.Properties["f"].Items
	require.NotNil(t, items)
	assert.NotEmpty(t, items.Ref)
	// min on a struct has no keyword: items stay the plain $ref.
	assert.False(t, strings.Contains(items.Ref, "min"))
	after, _ := json.Marshal(g.resolveSchema(g.newSchemaFromType(reflect.TypeFor[Elem]())))
	assert.JSONEq(t, string(before), string(after))
}

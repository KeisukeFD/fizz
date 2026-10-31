package openapi

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type compPet struct {
	Name string `json:"name"`
}

func TestRegisterSchema(t *testing.T) {
	g := gen(t)
	s := &Schema{Type: Types("object"), Properties: map[string]*Schema{"amount": {Type: Types("number")}}}
	require.NoError(t, g.RegisterSchema("Money", s))

	// Deep copy: changing the argument does not change the document.
	s.Properties["amount"].Type = Types("string")
	s.Properties["extra"] = &Schema{}
	got := g.API().Components.Schemas["Money"]
	require.NotNil(t, got)
	assert.Len(t, got.Properties, 1)
	assert.Equal(t, Types("number"), got.Properties["amount"].Type)

	// Same name again: collision, first kept, error returned not recorded.
	err := g.RegisterSchema("Money", &Schema{Type: Types("string")})
	var ce *CollisionError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, "Money", ce.Name)
	assert.Empty(t, g.Errors())
	assert.NotEmpty(t, ce.Error())
	assert.Equal(t, Types("object"), g.API().Components.Schemas["Money"].Type)
}

func TestRegisterSchemaInvalid(t *testing.T) {
	g := gen(t)
	for _, name := range []string{"", "a b", "a/b", "é"} {
		assert.Error(t, g.RegisterSchema(name, &Schema{}), name)
	}
	assert.Error(t, g.RegisterSchema("Nil", nil))
	assert.Error(t, g.RegisterSchema("NaN", &Schema{Default: math.NaN()}))
	assert.NoError(t, g.RegisterSchema("a.b_c-D9", &Schema{}))
	assert.Len(t, g.API().Components.Schemas, 1)
}

func TestRegisterSchemaCollidesWithType(t *testing.T) {
	g := gen(t)
	require.NoError(t, g.RegisterType(rt(compPet{})))
	err := g.RegisterSchema("CompPet", &Schema{})
	var ce *CollisionError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, rt(compPet{}), ce.Kept)
	assert.NotEmpty(t, ce.Error())

	// The other way around: the registered schema keeps the name.
	g = gen(t)
	require.NoError(t, g.RegisterSchema("CompPet", &Schema{Description: "mine"}))
	err = g.RegisterType(rt(compPet{}))
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, rt(compPet{}), ce.Rejected)
	assert.NotEmpty(t, ce.Error())
	assert.Equal(t, "mine", g.API().Components.Schemas["CompPet"].Description)
}

func TestRegisterType(t *testing.T) {
	g := gen(t)
	require.NoError(t, g.RegisterType(reflect.TypeOf(&compPet{})))
	assert.Contains(t, g.API().Components.Schemas, "CompPet")
	// Idempotent.
	require.NoError(t, g.RegisterType(rt(compPet{})))

	assert.Error(t, g.RegisterType(nil))
	assert.Error(t, g.RegisterType(rt(0)))
	assert.Error(t, g.RegisterType(rt([]compPet{})))
	// A named type that is inlined has no component, and leaves no trace.
	type name string
	before := len(g.API().Components.Schemas)
	assert.Error(t, g.RegisterType(rt(name(""))))
	assert.Len(t, g.API().Components.Schemas, before)
}

var rbContent = map[string]*MediaType{"application/json": {}}

func TestRegisterOtherComponents(t *testing.T) {
	g := gen(t)
	p := &Parameter{Name: "limit", In: "query", Schema: &Schema{Type: Types("integer")}}
	require.NoError(t, g.RegisterParameter("Limit", p))
	require.NoError(t, g.RegisterResponse("NotFound", &Response{Description: "nf"}))
	require.NoError(t, g.RegisterExample("Ex", &Example{Summary: "s"}))
	require.NoError(t, g.RegisterRequestBody("Body", &RequestBody{Description: "b", Content: rbContent}))
	require.NoError(t, g.RegisterHeader("Rate", &Header{Description: "h"}))

	p.Name = "changed"
	c := g.API().Components
	assert.Equal(t, "limit", c.Parameters["Limit"].Value.Name)
	assert.Equal(t, "nf", c.Responses["NotFound"].Value.Description)
	assert.Equal(t, "s", c.Examples["Ex"].Value.Summary)
	assert.Equal(t, "b", c.RequestBodies["Body"].Value.Description)
	assert.Equal(t, "h", c.Headers["Rate"].Value.Description)

	// Duplicates, invalid names and nil values are errors, in each kind.
	assert.Error(t, g.RegisterParameter("Limit", p))
	assert.Error(t, g.RegisterResponse("NotFound", &Response{}))
	assert.Error(t, g.RegisterExample("Ex", &Example{}))
	assert.Error(t, g.RegisterRequestBody("Body", &RequestBody{Content: rbContent}))
	assert.Error(t, g.RegisterRequestBody("NoContent", &RequestBody{Description: "b"}))
	assert.NoError(t, g.RegisterRequestBody("EmptyContent", &RequestBody{Content: map[string]*MediaType{}}))
	assert.Error(t, g.RegisterHeader("Rate", &Header{}))
	assert.Error(t, g.RegisterParameter("bad name", p))
	assert.Error(t, g.RegisterParameter("Nil", nil))
	assert.Error(t, g.RegisterResponse("Nil", nil))
	assert.Error(t, g.RegisterExample("Nil", nil))
	assert.Error(t, g.RegisterRequestBody("Nil", nil))
	assert.Error(t, g.RegisterHeader("Nil", nil))
	assert.Empty(t, g.Errors())

	// Each kind has its own namespace, schemas included.
	assert.NoError(t, g.RegisterResponse("Limit", &Response{}))
	assert.NoError(t, g.RegisterSchema("Limit", &Schema{}))
}

func TestGeneratorSetExtension(t *testing.T) {
	g := gen(t)
	require.NoError(t, g.SetExtension("x-a", "one"))
	require.NoError(t, g.SetExtension("x-a", "two"))
	assert.Equal(t, "two", g.API().Extensions["x-a"])

	assert.ErrorIs(t, g.SetExtension("a", 1), ErrInvalidExtensionKey)
	assert.ErrorIs(t, g.SetExtension("x-", 1), ErrInvalidExtensionKey)
	assert.ErrorIs(t, g.SetExtension("x-tagGroups", 1), ErrInvalidExtensionKey)
	assert.ErrorIs(t, g.SetExtension("x-oai-a", 1), ErrReservedExtension)
	assert.ErrorIs(t, g.SetExtension("X-OAS-a", 1), ErrReservedExtension)
	require.Error(t, g.SetExtension("x-nan", math.NaN()))
	assert.NotContains(t, g.API().Extensions, "x-nan")

	// The value is copied.
	m := map[string]any{"k": "v"}
	require.NoError(t, g.SetExtension("x-m", m))
	m["k"] = "changed"
	assert.Equal(t, map[string]any{"k": "v"}, g.API().Extensions["x-m"])
}

func TestSetTag(t *testing.T) {
	g := gen(t)
	g.AddTag("b", "B")
	g.SetTag(nil)
	g.SetTag(&Tag{})
	g.SetTag(&Tag{Name: "a", Summary: "A"})
	g.SetTag(&Tag{Name: "b", Description: "B2", Kind: "nav"})
	tags := g.API().Tags
	require.Len(t, tags, 2)
	assert.Equal(t, "a", tags[0].Name)
	assert.Equal(t, "b", tags[1].Name)
	assert.Equal(t, "B2", tags[1].Description)
	assert.Equal(t, "nav", tags[1].Kind)
	assert.Empty(t, g.Errors())

	// AddTag after SetTag cannot override: it warns.
	g.AddTag("b", "B3")
	assert.Equal(t, "B2", g.API().Tags[1].Description)
	assert.Len(t, g.Errors(), 1)

	// Copy: later changes to the argument do not leak.
	tag := &Tag{Name: "c", Extensions: map[string]any{"x-k": "v"}}
	g.SetTag(tag)
	tag.Extensions["x-k"] = "changed"
	assert.Equal(t, "v", g.API().Tags[2].Extensions["x-k"])
}

func TestAddTagDescriptionRules(t *testing.T) {
	g := gen(t)
	g.AddTag("g", "first")
	g.AddTag("g", "")
	g.AddTag("g", "first")
	assert.Equal(t, "first", g.API().Tags[0].Description)
	assert.Empty(t, g.Errors())

	g.AddTag("g", "second")
	assert.Equal(t, "first", g.API().Tags[0].Description)
	errs := g.Errors()
	require.Len(t, errs, 1)
	var w *Warning
	require.True(t, errors.As(errs[0], &w))
	assert.Contains(t, w.Msg, `"g"`)

	// A tag created empty takes the first description it is given.
	g.AddTag("h", "")
	g.AddTag("h", "late")
	assert.Equal(t, "late", g.API().Tags[1].Description)
}

func TestDocumentSetters(t *testing.T) {
	g := gen(t)
	g.SetSelf("https://example.com/api.json")
	g.SetJSONSchemaDialect("https://spec.openapis.org/oas/3.1/dialect/base")
	d := &ExternalDocs{URL: "https://example.com/docs"}
	g.SetExternalDocs(d)
	d.URL = "changed"
	groups := []*XTagGroup{{Name: "g", Tags: []string{"a"}}}
	g.SetTagGroups(groups)
	groups[0].Tags[0] = "changed"

	api := g.API()
	assert.Equal(t, "https://example.com/api.json", api.Self)
	assert.Equal(t, "https://spec.openapis.org/oas/3.1/dialect/base", api.JSONSchemaDialect)
	assert.Equal(t, "https://example.com/docs", api.ExternalDocs.URL)
	require.Len(t, api.XTagGroups, 1)
	assert.Equal(t, []string{"a"}, api.XTagGroups[0].Tags)

	g.SetExternalDocs(nil)
	g.SetTagGroups(nil)
	api = g.API()
	assert.Nil(t, api.ExternalDocs)
	assert.Nil(t, api.XTagGroups)
}

func TestManualComponentsBumpGeneration(t *testing.T) {
	g := gen(t)
	calls := map[string]func(){
		"RegisterSchema":       func() { _ = g.RegisterSchema("S", &Schema{}) },
		"RegisterType":         func() { _ = g.RegisterType(rt(compPet{})) },
		"RegisterParameter":    func() { _ = g.RegisterParameter("P", &Parameter{Name: "p", In: "query"}) },
		"RegisterResponse":     func() { _ = g.RegisterResponse("R", &Response{}) },
		"RegisterExample":      func() { _ = g.RegisterExample("E", &Example{}) },
		"RegisterRequestBody":  func() { _ = g.RegisterRequestBody("B", &RequestBody{Content: rbContent}) },
		"RegisterHeader":       func() { _ = g.RegisterHeader("H", &Header{}) },
		"SetExtension":         func() { _ = g.SetExtension("x-a", 1) },
		"SetTag":               func() { g.SetTag(&Tag{Name: "t"}) },
		"SetTagGroups":         func() { g.SetTagGroups([]*XTagGroup{{Name: "g"}}) },
		"SetSelf":              func() { g.SetSelf("u") },
		"SetJSONSchemaDialect": func() { g.SetJSONSchemaDialect("u") },
		"SetExternalDocs":      func() { g.SetExternalDocs(&ExternalDocs{URL: "u"}) },
	}
	for name, call := range calls {
		before := g.Generation()
		call()
		assert.Greater(t, g.Generation(), before, name)
	}
}

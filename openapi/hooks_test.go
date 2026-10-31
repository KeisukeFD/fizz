package openapi

import (
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	hkAddr struct {
		City string `json:"city"`
	}
	hkBody struct {
		Name   string `json:"name" validate:"required"`
		Secret string `json:"secret" validate:"required"`
		Addr   hkAddr `json:"addr"`
		Inl    hkAddr `json:"inl" inline:"true"`
	}
	hkIn struct {
		ID     string `path:"id"`
		Q      string `query:"q"`
		Secret string `json:"secret" validate:"required"`
		Name   string `json:"name"`
	}
	hkEmpty struct{}
)

func skipFields(names ...string) FieldHook {
	return func(c *FieldContext, _ *Schema) bool {
		for _, n := range names {
			if c.StructField.Name == n {
				return true
			}
		}
		return false
	}
}

func TestFieldHookSkipRemovesRequired(t *testing.T) {
	g := gen(t)
	g.SetHooks(Hooks{Field: skipFields("Secret")})
	g.newSchemaFromType(reflect.TypeFor[hkBody]())
	c := g.api.Components.Schemas["HkBody"]
	assert.NotContains(t, c.Properties, "secret")
	assert.Equal(t, []string{"name"}, c.Required)

	// The only required field skipped: no required list at all.
	g = gen(t)
	g.SetHooks(Hooks{Field: skipFields("Secret", "Name")})
	g.newSchemaFromType(reflect.TypeFor[hkBody]())
	assert.Nil(t, g.api.Components.Schemas["HkBody"].Required)
}

func TestFieldHookInInputBody(t *testing.T) {
	g := gen(t)
	g.SetHooks(Hooks{Field: skipFields("Secret")})
	op, err := g.AddOperation("/x/{id}", "POST", "", reflect.TypeFor[hkIn](), nil, &OperationInfo{StatusCode: 200, ID: "x"})
	require.NoError(t, err)
	body := g.api.Components.Schemas["XInput"]
	require.NotNil(t, body)
	assert.NotContains(t, body.Properties, "secret")
	assert.Contains(t, body.Properties, "name")
	assert.Empty(t, body.Required)
	assert.True(t, op.RequestBody != nil && !op.RequestBody.Required)
}

func TestFieldHookNotCalledForParameters(t *testing.T) {
	var names []string
	g := gen(t)
	g.SetHooks(Hooks{Field: func(c *FieldContext, _ *Schema) bool {
		names = append(names, c.Name)
		return false
	}})
	_, err := g.AddOperation("/x/{id}", "POST", "", reflect.TypeFor[hkIn](), nil, &OperationInfo{StatusCode: 200, ID: "x"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"secret", "name"}, names)
}

func TestFieldHookInlineDoesNotAlias(t *testing.T) {
	g := gen(t)
	g.SetHooks(Hooks{Field: func(c *FieldContext, s *Schema) bool {
		if c.StructField.Name == "Inl" {
			s.Properties["city"].Description = "hooked"
			s.Description = "inl"
		}
		return false
	}})
	g.newSchemaFromType(reflect.TypeFor[hkBody]())
	addr := g.api.Components.Schemas["HkAddr"]
	require.NotNil(t, addr)
	assert.Empty(t, addr.Description)
	assert.Empty(t, addr.Properties["city"].Description)
	inl := g.api.Components.Schemas["HkBody"].Properties["inl"]
	assert.Equal(t, "hooked", inl.Properties["city"].Description)
}

func TestFieldHookRefLeavesComponent(t *testing.T) {
	g := gen(t)
	g.SetHooks(Hooks{Field: func(c *FieldContext, s *Schema) bool {
		if c.Name == "addr" {
			s.Description = "the addr"
		}
		return false
	}})
	g.newSchemaFromType(reflect.TypeFor[hkBody]())
	assert.Equal(t, "the addr", g.api.Components.Schemas["HkBody"].Properties["addr"].Description)
	assert.Empty(t, g.api.Components.Schemas["HkAddr"].Description)
}

func TestParameterHookSkip(t *testing.T) {
	g := gen(t)
	var ctxs []ParameterContext
	g.SetHooks(Hooks{Parameter: func(c *ParameterContext, _ *Parameter) bool {
		ctxs = append(ctxs, *c)
		return true // both are skipped; the path one cannot be
	}})
	op, err := g.AddOperation("/x/{id}", "GET", "", reflect.TypeFor[hkIn](), nil, &OperationInfo{StatusCode: 200, ID: "x"})
	require.NoError(t, err)
	require.Len(t, op.Parameters, 1)
	assert.Equal(t, "id", op.Parameters[0].Value.Name)

	require.Len(t, ctxs, 2)
	for _, c := range ctxs {
		assert.Equal(t, "x", c.OperationID)
		assert.Equal(t, "GET", c.Method)
		assert.Equal(t, "/x/{id}", c.Path)
	}
	var w *Warning
	require.Len(t, g.Errors(), 1)
	require.ErrorAs(t, g.Errors()[0], &w)
	assert.Contains(t, w.Msg, "cannot be skipped")
	assert.Equal(t, "x", w.ID)
}

func TestParameterHookNotCalledForSynthesized(t *testing.T) {
	var n int
	g := gen(t)
	g.SetHooks(Hooks{Parameter: func(*ParameterContext, *Parameter) bool { n++; return false }})
	op, err := g.AddOperation("/x/{id}", "GET", "", reflect.TypeFor[hkEmpty](), nil, &OperationInfo{StatusCode: 200, ID: "x"})
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Len(t, op.Parameters, 1)
}

func TestOperationHookErrorDiscardsChanges(t *testing.T) {
	g := gen(t)
	g.SetHooks(Hooks{Operation: func(_ *OperationContext, op *Operation) error {
		op.Summary = "lost"
		return errors.New("boom")
	}})
	op, err := g.AddOperation("/x", "GET", "", nil, nil, &OperationInfo{StatusCode: 200, ID: "x", Summary: "kept"})
	require.NoError(t, err)
	assert.Equal(t, "kept", op.Summary)
	assert.Same(t, op, g.api.Paths["/x"].GET)

	var oe *OperationError
	require.Len(t, g.Errors(), 1)
	require.ErrorAs(t, g.Errors()[0], &oe)
	assert.Equal(t, "x", oe.ID)
	assert.ErrorContains(t, oe, "boom")
}

func TestOperationHookSuccessReplaces(t *testing.T) {
	g := gen(t)
	g.SetHooks(Hooks{Operation: func(c *OperationContext, op *Operation) error {
		assert.Equal(t, OperationContext{OperationID: "x", Method: "GET", Path: "/x"}, *c)
		op.Summary = "hook"
		return nil
	}})
	op, err := g.AddOperation("/x", "GET", "", nil, nil, &OperationInfo{StatusCode: 200, ID: "x", Summary: "opt"})
	require.NoError(t, err)
	assert.Equal(t, "hook", op.Summary)
	assert.Equal(t, "hook", g.api.Paths["/x"].GET.Summary)
	assert.Empty(t, g.Errors())
}

func TestOperationHookRevertsID(t *testing.T) {
	g := gen(t)
	g.SetHooks(Hooks{Operation: func(_ *OperationContext, op *Operation) error {
		op.ID = "other"
		op.Summary = "hook"
		return nil
	}})
	op, err := g.AddOperation("/x", "GET", "", nil, nil, &OperationInfo{StatusCode: 200, ID: "x"})
	require.NoError(t, err)
	assert.Equal(t, "x", op.ID)
	assert.Equal(t, "hook", op.Summary)
	var w *Warning
	require.Len(t, g.Errors(), 1)
	require.ErrorAs(t, g.Errors()[0], &w)
	assert.Equal(t, "x", w.ID)
}

func TestSchemaHookRuns(t *testing.T) {
	var names []string
	g := gen(t)
	g.SetHooks(Hooks{Schema: func(c *SchemaContext, s *Schema) {
		names = append(names, c.Name)
		s.Title = "t-" + c.Name
	}})
	require.NoError(t, g.RegisterSchema("Manual", &Schema{Type: Types("string")}))
	g.newSchemaFromType(reflect.TypeFor[hkBody]())
	g.newSchemaFromType(reflect.TypeFor[hkBody]())
	// Manual schemas get no hook; each component exactly once.
	assert.ElementsMatch(t, []string{"HkBody", "HkAddr"}, names)
	assert.Equal(t, "t-HkAddr", g.api.Components.Schemas["HkAddr"].Title)
	assert.Empty(t, g.api.Components.Schemas["Manual"].Title)
}

func TestSetHooksBumpsGeneration(t *testing.T) {
	g := gen(t)
	before := g.Generation()
	g.SetHooks(Hooks{})
	assert.Greater(t, g.Generation(), before)
}

func TestHooksOffByDefault(t *testing.T) {
	g := gen(t)
	_, err := g.AddOperation("/x", "POST", "", reflect.TypeFor[hkBody](), nil, &OperationInfo{StatusCode: 200, ID: "x"})
	require.NoError(t, err)
	assert.Empty(t, g.Errors())
}

func TestDocumentRunsOnCopyWithoutLock(t *testing.T) {
	g := gen(t)
	_, err := g.AddOperation("/a", "GET", "", nil, nil, &OperationInfo{StatusCode: 200, ID: "a"})
	require.NoError(t, err)

	var calls atomic.Int32
	release := make(chan struct{})
	entered := make(chan struct{})
	g.SetHooks(Hooks{Document: func(doc *OpenAPI) error {
		calls.Add(1)
		doc.Tags = append(doc.Tags, &Tag{Name: "extra"})
		close(entered)
		<-release
		return nil
	}})
	type result struct {
		doc *OpenAPI
		err error
	}
	done := make(chan result)
	go func() {
		doc, _, err := g.Document()
		done <- result{doc, err}
	}()
	<-entered
	// The hook is blocked: the generator is not locked.
	added := make(chan struct{})
	go func() {
		_, _ = g.AddOperation("/b", "GET", "", nil, nil, &OperationInfo{StatusCode: 200, ID: "b"})
		close(added)
	}()
	select {
	case <-added:
	case <-time.After(5 * time.Second):
		t.Fatal("AddOperation is blocked by the DocumentHook")
	}
	close(release)
	r := <-done
	require.NoError(t, r.err)
	assert.Contains(t, r.doc.Tags, &Tag{Name: "extra"})
	assert.Nil(t, r.doc.Paths["/b"], "the copy predates the registration")
	assert.Empty(t, g.API().Tags, "the generator is untouched")
	assert.EqualValues(t, 1, calls.Load())
}

func TestDocumentHookError(t *testing.T) {
	g := gen(t)
	g.SetHooks(Hooks{Document: func(*OpenAPI) error { return errors.New("nope") }})
	doc, _, err := g.Document()
	assert.Nil(t, doc)
	assert.ErrorContains(t, err, "DocumentHook")
	assert.ErrorContains(t, err, "nope")
	assert.Empty(t, g.Errors(), "not recorded by the generator")
	assert.NotNil(t, g.API(), "API() never runs the hook")
}

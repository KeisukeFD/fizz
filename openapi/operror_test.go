package openapi

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type opErrModel struct {
	Name string `json:"name"`
}

type opErrBody struct {
	Item opErrModel `json:"item"`
}

type opErrConflict struct {
	ID string `path:"id" query:"id"`
}

type opErrBadField struct {
	C chan int `json:"c"`
}

// assertNoTrace checks that the generator holds nothing of an operation.
func assertNoTrace(t *testing.T, g *Generator, id string) {
	t.Helper()
	assert.Empty(t, g.api.Paths)
	assert.Empty(t, g.api.Components.Schemas)
	assert.Empty(t, g.registry.names)
	assert.Empty(t, g.schemaTypes)
	assert.Empty(t, g.promoted)
	assert.Empty(t, g.operationsIDS)
	assert.Nil(t, g.cur)
	if id != "" {
		assert.NotContains(t, g.operationsIDS, id)
	}
}

func TestAddOperationNilInfo(t *testing.T) {
	g := gen(t)
	assert.NotPanics(t, func() {
		_, err := g.AddOperation("/x", "GET", "", nil, nil, nil)
		// A zero info has no status code: the response is rejected.
		var oe *OperationError
		require.ErrorAs(t, err, &oe)
		assert.Equal(t, "GET", oe.Method)
		assert.Equal(t, "/x", oe.Path)
	})
	assertNoTrace(t, g, "")
}

func TestFailedAddOperationLeavesNoTrace(t *testing.T) {
	tests := []struct {
		name string
		in   reflect.Type
		out  reflect.Type
		info *OperationInfo
	}{
		{
			name: "bad response code, schemas created before",
			in:   reflect.TypeFor[opErrBody](),
			out:  reflect.TypeFor[opErrModel](),
			info: &OperationInfo{ID: "bad", StatusCode: 200, Responses: []*OperationResponse{
				{Code: "1000", Model: opErrBody{}},
			}},
		},
		{
			name: "response code clash",
			out:  reflect.TypeFor[opErrModel](),
			info: &OperationInfo{ID: "clash", StatusCode: 200, Responses: []*OperationResponse{
				{Code: "200", Model: opErrBody{}},
			}},
		},
		{
			name: "conflicting locations",
			in:   reflect.TypeFor[opErrConflict](),
			info: &OperationInfo{ID: "conflict", StatusCode: 200},
		},
		{
			name: "input not a struct",
			in:   reflect.TypeFor[int](),
			info: &OperationInfo{ID: "notstruct", StatusCode: 200},
		},
		{
			name: "example and examples",
			out:  reflect.TypeFor[opErrModel](),
			info: &OperationInfo{ID: "ex", StatusCode: 200, Responses: []*OperationResponse{
				{Code: "400", Model: opErrBody{}, Example: 1, Examples: map[string]any{"a": 1}},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gen(t)
			op, err := g.AddOperation("/p", "POST", "tag", tt.in, tt.out, tt.info)
			require.Error(t, err)
			assert.Nil(t, op)
			var oe *OperationError
			require.ErrorAs(t, err, &oe)
			assert.Equal(t, tt.info.ID, oe.ID)
			require.Len(t, g.Errors(), 1)
			assert.Same(t, err, g.Errors()[0])
			assertNoTrace(t, g, tt.info.ID)

			// The ID and the names are free for the next operation.
			op, err = g.AddOperation("/ok", "GET", "", nil, tt.out, &OperationInfo{ID: tt.info.ID, StatusCode: 200})
			require.NoError(t, err)
			assert.Equal(t, tt.info.ID, op.ID)
		})
	}
}

func TestFailedAddOperationKeepsEarlierState(t *testing.T) {
	g := gen(t)
	_, err := g.AddOperation("/a", "GET", "", nil, reflect.TypeFor[opErrModel](), &OperationInfo{ID: "a", StatusCode: 200})
	require.NoError(t, err)
	before := len(g.api.Components.Schemas)
	require.Equal(t, 1, before)

	_, err = g.AddOperation("/b", "GET", "", nil, reflect.TypeFor[opErrBody](), &OperationInfo{ID: "a", StatusCode: 200})
	require.Error(t, err) // duplicate ID
	_, err = g.AddOperation("/c", "GET", "", nil, reflect.TypeFor[opErrBody](), &OperationInfo{ID: "c", StatusCode: 99})
	require.Error(t, err)

	assert.Len(t, g.api.Components.Schemas, before)
	assert.Contains(t, g.api.Components.Schemas, "OpErrModel")
	assert.Contains(t, g.api.Paths, "/a")
	assert.NotContains(t, g.api.Paths, "/b")
	assert.NotContains(t, g.api.Paths, "/c")
}

func TestOperationErrorWrapsFieldErrors(t *testing.T) {
	g := gen(t)
	op, err := g.AddOperation("/x", "POST", "", reflect.TypeFor[opErrBadField](), nil, &OperationInfo{ID: "x", StatusCode: 200})
	require.NoError(t, err) // a field problem does not fail the operation
	require.NotNil(t, op)

	errs := g.Errors()
	require.Len(t, errs, 1)
	var oe *OperationError
	require.ErrorAs(t, errs[0], &oe)
	assert.Equal(t, "POST", oe.Method)
	assert.Equal(t, "/x", oe.Path)
	assert.Equal(t, "x", oe.ID)
	assert.Contains(t, errs[0].Error(), "POST /x (x)")
	var te *TypeError
	assert.ErrorAs(t, errs[0], &te)
	assert.Same(t, oe.Err, errors.Unwrap(errs[0]))

	// Not in an operation: recorded as is.
	g.RecordError(errors.New("plain"))
	assert.EqualError(t, g.Errors()[1], "plain")
}

func TestOperationErrorMessage(t *testing.T) {
	cause := errors.New("boom")
	e := &OperationError{Method: "GET", Path: "/a", Err: cause}
	assert.Equal(t, "GET /a: boom", e.Error())
	assert.ErrorIs(t, e, cause)
	e.ID = "id"
	assert.Equal(t, "GET /a (id): boom", e.Error())
	assert.Equal(t, "GET /a (id)", (&OperationError{Method: "GET", Path: "/a", ID: "id"}).Error())
}

func TestWarningsAreNotWrapped(t *testing.T) {
	g := gen(t)
	g.cur = &opContext{method: "GET", path: "/w"}
	g.warn("careful")
	g.cur = nil
	require.Len(t, g.Errors(), 1)
	var w *Warning
	require.True(t, errors.As(g.Errors()[0], &w))
	assert.Equal(t, "/w", w.Path)
	var oe *OperationError
	assert.False(t, errors.As(g.Errors()[0], &oe))
}

func TestErrorsReturnsCopy(t *testing.T) {
	g := gen(t)
	g.RecordError(errors.New("one"))
	errs := g.Errors()
	errs[0] = nil
	assert.EqualError(t, g.Errors()[0], "one")
}

func TestSetSecuritySchemesDropsNilOrRef(t *testing.T) {
	g := gen(t)
	g.SetSecuritySchemes(map[string]*SecuritySchemeOrRef{
		"empty": {},
		"nil":   nil,
		"ok":    {Value: &SecurityScheme{Type: "http", Scheme: "bearer"}},
	})
	assert.Len(t, g.api.Components.SecuritySchemes, 1)
	assert.Contains(t, g.api.Components.SecuritySchemes, "ok")
	require.Len(t, g.Errors(), 2)
	for _, e := range g.Errors() {
		assert.ErrorIs(t, e, ErrNilOrRef)
	}
	assert.ErrorContains(t, g.Errors()[0], `"empty"`)
}

package openapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultResponseDescription(t *testing.T) {
	for code, want := range map[string]string{
		"default": "Default response",
		"1XX":     "1XX response",
		"4XX":     "4XX response",
		"5XX":     "5XX response",
		"200":     "OK",
		"404":     "Not Found",
		"299":     "Status 299",
		"599":     "Status 599",
	} {
		assert.Equal(t, want, defaultResponseDescription(code), code)
	}
}

func TestResponseDescriptionsAndEmptyContent(t *testing.T) {
	g, err := NewGenerator(&SpecGenConfig{})
	require.NoError(t, err)
	op, err := g.AddOperation("/r", "GET", "Test", nil, nil, &OperationInfo{
		ID:         "getR",
		StatusCode: 204,
		Responses: []*OperationResponse{
			{Code: "default"},
			{Code: "4XX"},
			{Code: "299"},
			{Code: "400", Examples: map[string]any{}},
		},
	})
	require.NoError(t, err)
	for code, want := range map[string]string{
		"204": "No Content", "default": "Default response", "4XX": "4XX response",
		"299": "Status 299", "400": "Bad Request",
	} {
		r := op.Responses[code]
		require.NotNil(t, r, code)
		assert.Equal(t, want, r.Value.Description, code)
		assert.Empty(t, r.Value.Content, code)
	}
}

func TestResponseHeaderFields(t *testing.T) {
	g, err := NewGenerator(&SpecGenConfig{})
	require.NoError(t, err)
	ex := map[string]*ExampleOrRef{"a": {Value: &Example{Value: "x"}}}
	op, err := g.AddOperation("/r", "GET", "Test", nil, reflect.TypeFor[string](), &OperationInfo{
		ID:         "getR",
		StatusCode: 200,
		Headers: []*ResponseHeader{
			{Name: "X-A", Description: "d", Required: true, Deprecated: true, Example: "v"},
			{Name: "X-B", Examples: ex},
		},
	})
	require.NoError(t, err)
	h := op.Responses["200"].Value.Headers
	a := h["X-A"].Value
	assert.True(t, a.Required)
	assert.True(t, a.Deprecated)
	assert.Equal(t, "v", a.Example)
	assert.Equal(t, ex, h["X-B"].Value.Examples)
}

func TestDefaultMediaTypeConfig(t *testing.T) {
	g, err := NewGenerator(&SpecGenConfig{DefaultMediaType: "application/vnd.api+json"})
	require.NoError(t, err)
	op, err := g.AddOperation("/r", "GET", "Test", nil, reflect.TypeFor[string](), &OperationInfo{ID: "getR", StatusCode: 200})
	require.NoError(t, err)
	assert.Contains(t, op.Responses["200"].Value.Content, "application/vnd.api+json")
}

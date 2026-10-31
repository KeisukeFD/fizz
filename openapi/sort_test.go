package openapi

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSortParams tests that the parameters of an operation are
// ordered by location, then by name.
func TestSortParams(t *testing.T) {
	g := &Generator{api: &OpenAPI{Components: &Components{}}}
	params := []*ParameterOrRef{
		{Value: &Parameter{Name: "Paz", In: "path"}},
		{Value: &Parameter{Name: "Baz", In: "header"}},
		{Value: &Parameter{Name: "Bar", In: "query"}},
		{Value: &Parameter{Name: "Zap", In: "path"}},
		{Value: &Parameter{Name: "Foo", In: "query"}},
	}
	slices.SortStableFunc(params, g.compareParams)

	assert.Len(t, params, 5)

	assert.Equal(t, params[0].Value.Name, "Paz")
	assert.Equal(t, params[1].Value.Name, "Zap")
	assert.Equal(t, params[2].Value.Name, "Bar")
	assert.Equal(t, params[3].Value.Name, "Foo")
	assert.Equal(t, params[4].Value.Name, "Baz")
}

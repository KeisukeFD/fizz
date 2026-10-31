package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSortParams tests that the parameters of
// an operation can be sorted using multiple
// sort functions.
func TestSortParams(t *testing.T) {
	params := []*ParameterOrRef{
		{Value: &Parameter{Name: "Paz", In: "path"}},
		{Value: &Parameter{Name: "Baz", In: "header"}},
		{Value: &Parameter{Name: "Bar", In: "query"}},
		{Value: &Parameter{Name: "Zap", In: "path"}},
		{Value: &Parameter{Name: "Foo", In: "query"}},
	}
	// Sort by location, then by name in ascending order.
	paramsOrderedBy(byLocation, byName).Sort(params)

	assert.Len(t, params, 5)

	assert.Equal(t, params[0].Value.Name, "Paz")
	assert.Equal(t, params[1].Value.Name, "Zap")
	assert.Equal(t, params[2].Value.Name, "Bar")
	assert.Equal(t, params[3].Value.Name, "Foo")
	assert.Equal(t, params[4].Value.Name, "Baz")
}

func byName(p1, p2 *ParameterOrRef) bool {
	return p1.Value.Name < p2.Value.Name
}

func byLocation(p1, p2 *ParameterOrRef) bool {
	return locationsOrder[p1.Value.In] < locationsOrder[p2.Value.In]
}

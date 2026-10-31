// Package baseuriretrieval rebuilds the spec-text document
// base-uri-from-the-retrieval-uri-1 (a request body component whose schema
// is a relative $ref) with the public fizz API.
package baseuriretrieval

import (
	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// New returns the app: a fizz whose spec is the document.
func New() *fizz.Fizz {
	f := fizz.New()
	g := f.Generator()
	g.SetInfo(&openapi.Info{Title: "Example API", Version: "1.0"})
	err := g.RegisterRequestBody("Foo", &openapi.RequestBody{
		Content: map[string]*openapi.MediaType{
			"application/json": {Schema: &openapi.Schema{Ref: "schemas/foo"}},
		},
	})
	if err != nil {
		panic(err)
	}
	return f
}

package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/specvalidate"
	"github.com/KeisukeFD/fizz/openapi"
)

// P10.9c and P10.9n: the served document omits an empty "components" and an
// empty "paths", but never becomes invalid: OAS 3.2 wants one of paths,
// components and webhooks.
func TestServedEmptyContainers(t *testing.T) {
	route := func(f *fizz.Fizz) {
		f.GET("/x", []fizz.OperationOption{fizz.ID("getX")}, tonic.Handler(func(c *gin.Context) error { return nil }, 204))
	}
	tests := []struct {
		name         string
		setup        func(f *fizz.Fizz)
		paths, comps bool // keys expected in the document
	}{
		{"nothing", func(*fizz.Fizz) {}, true, false},
		{"route only", route, true, false},
		{"components only", func(f *fizz.Fizz) {
			if err := f.Generator().RegisterSchema("S", &openapi.Schema{Type: openapi.Types("string")}); err != nil {
				t.Fatal(err)
			}
		}, false, true},
		{"route and components", func(f *fizz.Fizz) {
			route(f)
			if err := f.Generator().RegisterSchema("S", &openapi.Schema{Type: openapi.Types("string")}); err != nil {
				t.Fatal(err)
			}
		}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fizz.New()
			f.Generator().SetInfo(&openapi.Info{Title: "t", Version: "1"})
			tt.setup(f)
			doc, err := f.Document()
			if err != nil {
				t.Fatal(err)
			}
			b, err := openapi.MarshalJSON(doc)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]json.RawMessage
			if err := json.Unmarshal(b, &m); err != nil {
				t.Fatal(err)
			}
			if _, ok := m["paths"]; ok != tt.paths {
				t.Errorf("paths present = %v, want %v: %s", ok, tt.paths, b)
			}
			if _, ok := m["components"]; ok != tt.comps {
				t.Errorf("components present = %v, want %v: %s", ok, tt.comps, b)
			}
			if err := specvalidate.Validate(b); err != nil {
				t.Errorf("served document is invalid: %v\n%s", err, b)
			}
		})
	}
}

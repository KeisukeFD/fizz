// Package webhook rebuilds learn.openapis.org
// examples/v3.1/webhook-example.json with the public fizz API.
package webhook

import (
	"reflect"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// Pet is a pet of the store. It is the payload of the newPet webhook.
type Pet struct {
	ID   int64  `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
	Tag  string `json:"tag,omitempty"`
}

// New returns the app: a fizz whose spec is the webhook example.
//
// The example has no path, only a webhook, and fizz has no API for
// webhooks: the newPet webhook is not reproduced (see allow.json). The Pet
// component it references is registered by type.
func New() *fizz.Fizz {
	f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
	g := f.Generator()
	g.SetInfo(&openapi.Info{Title: "Webhook Example", Version: "1.0.0"})
	if err := g.RegisterType(reflect.TypeFor[Pet]()); err != nil {
		panic(err)
	}
	return f
}

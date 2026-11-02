// Package petstore rebuilds learn.openapis.org examples/v3.0/petstore.json
// with the public fizz API.
package petstore

import (
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// Pet is a pet of the store.
type Pet struct {
	ID   int64  `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
	Tag  string `json:"tag,omitempty"`
}

// Pets is a page of pets.
type Pets []Pet

// CustomizeSchema caps the page at 100 items.
func (Pets) CustomizeSchema(s *openapi.Schema) error {
	s.MaxItems = new(uint64(100))
	return nil
}

// Error is the error body of every operation.
type Error struct {
	Code    int32  `json:"code" validate:"required"`
	Message string `json:"message" validate:"required"`
}

type listPetsIn struct {
	Limit int32 `query:"limit" description:"How many items to return at one time (max 100)" validate:"max=100"`
}

type showPetByIDIn struct {
	PetID string `path:"petId" description:"The id of the pet to retrieve"`
}

// New returns the app: a fizz whose spec is the petstore example.
func New() *fizz.Fizz {
	// The official components are named after the Go type: no package prefix.
	f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
	g := f.Generator()
	g.SetInfo(&openapi.Info{
		Title:   "Swagger Petstore",
		Version: "1.0.0",
		License: &openapi.License{Name: "MIT"},
	})
	g.SetServers([]*openapi.Server{{URL: "http://petstore.swagger.io/v1"}})

	unexpected := fizz.Response("default", "unexpected error", Error{}, nil, nil)

	f.GET("/pets", []fizz.OperationOption{
		fizz.ID("listPets"),
		fizz.Summary("List all pets"),
		fizz.Tags("pets"),
		fizz.StatusDescription("A paged array of pets"),
		fizz.Header("x-next", "A link to the next page of responses", fizz.String),
		unexpected,
	}, tonic.Handler(func(*gin.Context, *listPetsIn) (Pets, error) { return nil, nil }, 200))

	f.POST("/pets", []fizz.OperationOption{
		fizz.ID("createPets"),
		fizz.Summary("Create a pet"),
		fizz.Tags("pets"),
		fizz.StatusDescription("Null response"),
		fizz.RequestBody("", true, fizz.Content("application/json", Pet{})),
		unexpected,
	}, tonic.Handler(func(*gin.Context) error { return nil }, 201))

	f.GET("/pets/:petId", []fizz.OperationOption{
		fizz.ID("showPetById"),
		fizz.Summary("Info for a specific pet"),
		fizz.Tags("pets"),
		fizz.StatusDescription("Expected response to a valid request"),
		unexpected,
	}, tonic.Handler(func(*gin.Context, *showPetByIDIn) (*Pet, error) { return nil, nil }, 200))
	return f
}

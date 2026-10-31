// Package petstoreexpanded rebuilds learn.openapis.org
// examples/v3.0/petstore-expanded.json with the public fizz API.
package petstoreexpanded

import (
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// NewPet is the body of a pet to add.
type NewPet struct {
	Name string `json:"name" validate:"required"`
	Tag  string `json:"tag,omitempty"`
}

// Pet is a NewPet with an ID. The official schema composes the two with
// allOf, which a Go struct cannot say, so the type provides its schema.
type Pet struct {
	NewPet
	ID int64 `json:"id"`
}

// OpenAPISchema implements openapi.SchemaProvider.
func (Pet) OpenAPISchema() *openapi.Schema {
	return &openapi.Schema{AllOf: []*openapi.Schema{
		{Ref: "#/components/schemas/NewPet"},
		{
			Type:       openapi.Types("object"),
			Required:   []string{"id"},
			Properties: map[string]*openapi.Schema{"id": {Type: openapi.Types("integer"), Format: "int64"}},
		},
	}}
}

// Error is the error body of every operation.
type Error struct {
	Code    int32  `json:"code" validate:"required"`
	Message string `json:"message" validate:"required"`
}

type findPetsIn struct {
	Tags  []string `query:"tags" description:"tags to filter by"`
	Limit int32    `query:"limit" description:"maximum number of results to return"`
}

type getPetIn struct {
	ID int64 `path:"id" description:"ID of pet to fetch"`
}

type deletePetIn struct {
	ID int64 `path:"id" description:"ID of pet to delete"`
}

const findPetsDescription = `Returns all pets from the system that the user has access to
Nam sed condimentum est. Maecenas tempor sagittis sapien, nec rhoncus sem sagittis sit amet. Aenean at gravida augue, ac iaculis sem. Curabitur odio lorem, ornare eget elementum nec, cursus id lectus. Duis mi turpis, pulvinar ac eros ac, tincidunt varius justo. In hac habitasse platea dictumst. Integer at adipiscing ante, a sagittis ligula. Aenean pharetra tempor ante molestie imperdiet. Vivamus id aliquam diam. Cras quis velit non tortor eleifend sagittis. Praesent at enim pharetra urna volutpat venenatis eget eget mauris. In eleifend fermentum facilisis. Praesent enim enim, gravida ac sodales sed, placerat id erat. Suspendisse lacus dolor, consectetur non augue vel, vehicula interdum libero. Morbi euismod sagittis libero sed lacinia.

Sed tempus felis lobortis leo pulvinar rutrum. Nam mattis velit nisl, eu condimentum ligula luctus nec. Phasellus semper velit eget aliquet faucibus. In a mattis elit. Phasellus vel urna viverra, condimentum lorem id, rhoncus nibh. Ut pellentesque posuere elementum. Sed a varius odio. Morbi rhoncus ligula libero, vel eleifend nunc tristique vitae. Fusce et sem dui. Aenean nec scelerisque tortor. Fusce malesuada accumsan magna vel tempus. Quisque mollis felis eu dolor tristique, sit amet auctor felis gravida. Sed libero lorem, molestie sed nisl in, accumsan tempor nisi. Fusce sollicitudin massa ut lacinia mattis. Sed vel eleifend lorem. Pellentesque vitae felis pretium, pulvinar elit eu, euismod sapien.
`

// New returns the app: a fizz whose spec is the petstore-expanded example.
func New() *fizz.Fizz {
	f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
	g := f.Generator()
	// The official parameters are in declaration order, not sorted.
	g.SetSortParams(false)
	g.SetInfo(&openapi.Info{
		Title:          "Swagger Petstore",
		Version:        "1.0.0",
		Description:    "A sample API that uses a petstore as an example to demonstrate features in the OpenAPI 3.0 specification",
		TermsOfService: "http://swagger.io/terms/",
		Contact: &openapi.Contact{
			Name:  "Swagger API Team",
			Email: "apiteam@swagger.io",
			URL:   "http://swagger.io",
		},
		License: &openapi.License{
			Name: "Apache 2.0",
			URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
		},
	})
	g.SetServers([]*openapi.Server{{URL: "https://petstore.swagger.io/v2"}})

	unexpected := fizz.Response("default", "unexpected error", Error{}, nil, nil)

	f.GET("/pets", []fizz.OperationOption{
		fizz.ID("findPets"),
		fizz.Description(findPetsDescription),
		fizz.StatusDescription("pet response"),
		unexpected,
	}, tonic.Handler(func(*gin.Context, *findPetsIn) ([]Pet, error) { return nil, nil }, 200))

	f.POST("/pets", []fizz.OperationOption{
		fizz.ID("addPet"),
		fizz.Description("Creates a new pet in the store. Duplicates are allowed"),
		fizz.StatusDescription("pet response"),
		fizz.RequestBody("Pet to add to the store", true, fizz.Content("application/json", NewPet{})),
		unexpected,
	}, tonic.Handler(func(*gin.Context) (*Pet, error) { return nil, nil }, 200))

	f.GET("/pets/:id", []fizz.OperationOption{
		fizz.ID("find pet by id"),
		fizz.Description("Returns a user based on a single ID, if the user does not have access to the pet"),
		fizz.StatusDescription("pet response"),
		unexpected,
	}, tonic.Handler(func(*gin.Context, *getPetIn) (*Pet, error) { return nil, nil }, 200))

	f.DELETE("/pets/:id", []fizz.OperationOption{
		fizz.ID("deletePet"),
		fizz.Description("deletes a single pet based on the ID supplied"),
		fizz.StatusDescription("pet deleted"),
		unexpected,
	}, tonic.Handler(func(*gin.Context, *deletePetIn) error { return nil }, 204))
	return f
}

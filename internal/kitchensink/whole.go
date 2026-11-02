package kitchensink

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/kitchensink/a/models"
	"github.com/KeisukeFD/fizz/internal/kitchensink/generic"
	"github.com/KeisukeFD/fizz/openapi"
)

// Whole golden files: the output of NewWholeFizz.
const (
	wholeGoldenJSON = "testdata/kitchensink.json"
	wholeGoldenYAML = "testdata/kitchensink.yaml"
)

type legacyKind string

type legacyAddress struct {
	Street string `json:"street" description:"Street line"`
	City   string `json:"city" validate:"required"`
}

type legacyBase struct {
	Created time.Time `json:"created"`
}

type legacyPet struct {
	legacyBase
	ID       int               `json:"id" description:"Unique ID"`
	Name     string            `json:"name" validate:"required,min=1,max=64"`
	Kind     legacyKind        `json:"kind" enum:"cat,dog,fish" default:"dog"`
	Tags     []string          `json:"tags,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
	Owner    *legacyAddress    `json:"owner,omitempty"`
	Weight   float64           `json:"weight" validate:"gte=0"`
	Nickname *string           `json:"nickname,omitempty"`
	Secret   string            `json:"-"`
}

type legacyListIn struct {
	Limit  int      `query:"limit" default:"20" validate:"min=1,max=100" description:"Page size"`
	Kinds  []string `query:"kind" explode:"false"`
	Token  string   `header:"X-Token" description:"Auth token"`
	Search string   `query:"q"`
}

type legacyGetIn struct {
	ID int `path:"id" description:"Pet ID"`
}

type legacyCreateIn struct {
	DryRun bool      `query:"dry_run"`
	Pet    legacyPet `json:"pet"`
}

func legacyList(*gin.Context, *legacyListIn) (*generic.Page[legacyPet], error) { return nil, nil }
func legacyGet(*gin.Context, *legacyGetIn) (*legacyPet, error)                 { return nil, nil }
func legacyCreate(*gin.Context, *legacyCreateIn) (*legacyPet, error)           { return nil, nil }
func legacyDelete(*gin.Context, *legacyGetIn) error                            { return nil }
func legacyItem(*gin.Context, *struct{}) (*models.Item, error)                 { return nil, nil }

// NewWholeFizz returns the whole kitchen-sink fixture: a pet store with
// groups, parameters of every location, generics, examples, security
// and servers. It is the fixture behind the whole-document golden, the
// determinism test and the race test.
func NewWholeFizz() *fizz.Fizz {
	f := fizz.New()

	g := f.Group("/pets", "Pets", "Pet operations")
	g.GET("", []fizz.OperationOption{
		fizz.ID("listPets"), fizz.Summary("List pets"),
		fizz.Header("X-Rate-Limit", "Requests left", 0),
		fizz.ResponseWithExamples("400", "Bad request", nil, nil, map[string]any{
			"empty": generic.Page[legacyPet]{},
		}),
	}, tonic.Handler(legacyList, 200))
	g.GET("/:id", []fizz.OperationOption{
		fizz.ID("getPet"), fizz.Description("Fetch one pet."),
		fizz.Response("404", "Not found", nil, nil, nil),
	}, tonic.Handler(legacyGet, 200))
	g.POST("", []fizz.OperationOption{
		fizz.ID("createPet"), fizz.StatusDescription("Created"),
		fizz.Security(&openapi.SecurityRequirement{"api_key": []string{}}),
	}, tonic.Handler(legacyCreate, 201))
	g.DELETE("/:id", []fizz.OperationOption{
		fizz.ID("deletePet"), fizz.Deprecated(true),
	}, tonic.Handler(legacyDelete, 204))

	f.GET("/items", []fizz.OperationOption{fizz.ID("getItem")}, tonic.Handler(legacyItem, 200))

	gen := f.Generator()
	gen.SetServers([]*openapi.Server{{URL: "https://api.example.com/v1", Description: "production"}})
	gen.SetSecurityRequirement([]*openapi.SecurityRequirement{{"api_key": []string{}}})
	gen.SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
		"api_key": {Value: &openapi.SecurityScheme{Type: "apiKey", Name: "api_key", In: "header"}},
	})
	return f
}

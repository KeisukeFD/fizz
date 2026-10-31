package main

import (
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// NewRouter returns a new router for the
// Pet Store.
func NewRouter() (*fizz.Fizz, error) {
	engine := gin.New()
	engine.Use(cors.Default())

	f := fizz.NewFromEngine(engine,
		fizz.WithDefaultMediaType("application/json"),
		fizz.WithHooks(openapi.Hooks{Document: addTagGroups}),
	)

	// Override type names.
	// fizz.Generator().OverrideTypeName(reflect.TypeOf(Fruit{}), "SweetFruit")

	// Initialize the informations of
	// the API that will be served with
	// the specification.
	infos := &openapi.Info{
		Title:       "Fruits Market",
		Description: `This is a sample Fruits market server.`,
		Version:     "1.0.0",
	}
	// Create a new route that serve the OpenAPI spec.
	f.GET("/openapi.json", nil, f.OpenAPI(infos, "json"))

	// Serve the interactive documentation (Scalar) of the spec.
	f.Docs("/docs", "/openapi.json", fizz.ScalarUI())

	// Setup routes.
	routes(f.Group("/market", "market", "Your daily dose of freshness"))

	if len(f.Errors()) != 0 {
		return nil, fmt.Errorf("fizz errors: %v", f.Errors())
	}
	return f, nil
}

func routes(grp *fizz.RouterGroup) {
	// Add a new fruit to the market.
	grp.POST("", []fizz.OperationOption{
		fizz.Summary("Add a fruit to the market"),
		fizz.RequestBody("The fruit to add to the market", true,
			fizz.Content("application/json", Fruit{}),
		),
		fizz.Response("400", "Bad request", nil, nil,
			map[string]any{"error": "fruit already exists"},
		),
	}, tonic.Handler(CreateFruit, 200))

	// Remove a fruit from the market,
	// probably because it rotted.
	grp.DELETE("/:name", []fizz.OperationOption{
		fizz.Summary("Remove a fruit from the market"),
		fizz.ResponseWithExamples("400", "Bad request", nil, nil, map[string]any{
			"fruitNotFound": map[string]any{"error": "fruit not found"},
			"invalidApiKey": map[string]any{"error": "invalid api key"},
		}),
	}, tonic.Handler(DeleteFruit, 204))

	// List all available fruits.
	grp.GET("", []fizz.OperationOption{
		fizz.Summary("List the fruits of the market"),
		fizz.Response("400", "Bad request", nil, nil, nil),
		fizz.Header("X-Market-Listing-Size", "Listing size", fizz.Long),
		fizz.Header("X-Market-Currency", "Currency of the client", ""),
		fizz.CustomizeOperation(func(op *openapi.Operation) error {
			op.Description += "\n\nUse filter[origin]=spain to list the fruits of a single origin."
			return nil
		}),
	}, tonic.Handler(ListFruits, 200))
}

// addTagGroups groups the tags of the document under x-tagGroups, which
// documentation UIs use to organize the sidebar.
func addTagGroups(doc *openapi.OpenAPI) error {
	doc.XTagGroups = []*openapi.XTagGroup{
		{Name: "Shopping", Tags: []string{"market"}},
	}
	return nil
}

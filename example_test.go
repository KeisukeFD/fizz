package fizz_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

type optionsIn struct {
	Name *string `query:"name" doc:"Name filter"`
}

// This example reads the description from a "doc" tag and documents
// pointers as plain (non-null) types.
func ExampleNew_options() {
	f := fizz.New(
		fizz.WithTagNames(openapi.TagNames{Description: "doc"}),
		fizz.WithPointersAreNullable(false),
	)
	f.GET("/pets", []fizz.OperationOption{fizz.ID("listPets")},
		tonic.Handler(func(*gin.Context, *optionsIn) error { return nil }, 200))

	op := f.Generator().API().Paths["/pets"].GET
	b, err := json.Marshal(op.Parameters[0])
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output:
	// {"name":"name","in":"query","description":"Name filter","schema":{"type":"string","description":"Name filter"}}
}

// This example adds an x-tagGroups extension to the served document with
// a DocumentHook. The hook edits a copy: Generator().API() is unchanged.
func ExampleWithHooks() {
	f := fizz.New(fizz.WithHooks(openapi.Hooks{
		Document: func(doc *openapi.OpenAPI) error {
			doc.XTagGroups = append(doc.XTagGroups, &openapi.XTagGroup{
				Name: "Pets",
				Tags: []string{"pets"},
			})
			return nil
		},
	}))
	f.GET("/pets", []fizz.OperationOption{fizz.ID("listPets")},
		tonic.Handler(func(*gin.Context, *optionsIn) error { return nil }, 200))

	doc, err := f.Document()
	if err != nil {
		panic(err)
	}
	b, err := json.Marshal(doc.XTagGroups)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output:
	// [{"name":"Pets","tags":["pets"]}]
}

type exPet struct {
	Name string `json:"name"`
}

// This example documents a request body in XML instead of the JSON body
// inferred from the input, and marks it required.
func ExampleRequestBody() {
	f := fizz.New()
	f.POST("/pets", []fizz.OperationOption{
		fizz.ID("createPet"),
		fizz.RequestBody("The pet to create", true,
			fizz.Content("application/xml", exPet{})),
	}, tonic.Handler(func(*gin.Context, *exPet) error { return nil }, 201))

	doc, err := f.Document()
	if err != nil {
		panic(err)
	}
	b, err := json.Marshal(doc.Paths["/pets"].POST.RequestBody)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output:
	// {"description":"The pet to create","content":{"application/xml":{"schema":{"$ref":"#/components/schemas/Fizz_testExPet"}}},"required":true}
}

// This example edits the operation after every other option has been
// applied: CustomizeOperation runs last, whatever its position.
func ExampleCustomizeOperation() {
	f := fizz.New()
	f.GET("/pets", []fizz.OperationOption{
		fizz.CustomizeOperation(func(op *openapi.Operation) error {
			op.Summary += " (beta)"
			return nil
		}),
		fizz.ID("listPets"),
		fizz.Summary("List pets"),
	}, tonic.Handler(func(*gin.Context, *optionsIn) error { return nil }, 200))

	doc, err := f.Document()
	if err != nil {
		panic(err)
	}
	fmt.Println(doc.Paths["/pets"].GET.Summary)
	// Output:
	// List pets (beta)
}

type deepObjectIn struct {
	Filter map[string]string `query:"filter" style:"deepObject"`
}

// This example binds a deepObject query parameter: ?filter[color]=red
// fills the map. New installs the tonic bind hook that does it.
func Example_deepObject() {
	f := fizz.New()
	f.GET("/pets", []fizz.OperationOption{fizz.ID("listPets")},
		tonic.Handler(func(c *gin.Context, in *deepObjectIn) error {
			fmt.Println(in.Filter)
			return nil
		}, 204))

	req := httptest.NewRequest("GET", "/pets?filter[color]=red", nil)
	f.ServeHTTP(httptest.NewRecorder(), req)
	// Output:
	// map[color:red]
}

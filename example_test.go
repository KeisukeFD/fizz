package fizz_test

import (
	"encoding/json"
	"fmt"

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

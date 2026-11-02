package fizz_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// Status lists its valid values: the component gets an enum.
type Status string

func (Status) Enum() []any { return []any{"active", "archived"} }

// Money describes its own schema, instead of the one of its fields.
type Money struct{ Cents int64 }

func (Money) OpenAPISchema() *openapi.Schema {
	return &openapi.Schema{Type: openapi.Types("string"), Pattern: `^\d+\.\d{2}$`}
}

// Account uses both.
type Account struct {
	Status  Status `json:"status"`
	Balance Money  `json:"balance"`
}

// This example gives two types their schema with the Enumer and
// SchemaProvider interfaces. Both become components that the fields
// reference.
func Example_interfaces() {
	f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
	f.GET("/accounts", []fizz.OperationOption{fizz.ID("listAccounts")},
		tonic.Handler(func(*gin.Context) ([]Account, error) { return nil, nil }, 200))

	schemas := f.Generator().API().Components.Schemas
	for _, name := range []string{"Status", "Money"} {
		b, err := json.Marshal(schemas[name])
		if err != nil {
			panic(err)
		}
		fmt.Println(name, string(b))
	}
	// Output:
	// Status {"type":"string","enum":["active","archived"]}
	// Money {"type":"string","pattern":"^\\d+\\.\\d{2}$"}
}

type namedIn struct {
	ID   int    `path:"id"`
	Name string `json:"name"`
}

// This example names the operation and its request body component
// instead of using the defaults.
func Example_naming() {
	f := fizz.New(
		fizz.WithOperationIDNamer(func(method, path, handler string) string {
			path = strings.NewReplacer("/", "_", "{", "", "}", "").Replace(path)
			return strings.ToLower(method) + path
		}),
		fizz.WithRequestBodyNamer(func(opID string, _ reflect.Type) string {
			return opID + "_body"
		}),
	)
	f.POST("/pets/:id", nil,
		tonic.Handler(func(*gin.Context, *namedIn) error { return nil }, 201))

	op := f.Generator().API().Paths["/pets/{id}"].POST
	fmt.Println(op.ID)
	fmt.Println(op.RequestBody.Content["application/json"].Schema.Ref)
	// Output:
	// post_pets_id
	// #/components/schemas/post_pets_id_body
}

// Tagged shows some of the struct tags that enrich a schema.
type Tagged struct {
	Name  string  `json:"name" title:"Pet name" pattern:"^[a-z]+$" validate:"min=1,max=20"`
	Score float64 `json:"score" multipleOf:"0.5" validate:"gte=0,lte=10"`
	ID    string  `json:"id" readOnly:"true"`
}

// This example reads the tags of a struct: title, pattern, multipleOf and
// readOnly are fizz tags, min, max, gte and lte are validate rules that
// become schema keywords.
func Example_tags() {
	f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
	f.GET("/pets/best", []fizz.OperationOption{fizz.ID("bestPet")},
		tonic.Handler(func(*gin.Context) (*Tagged, error) { return nil, nil }, 200))

	b, err := json.Marshal(f.Generator().API().Components.Schemas["Tagged"].Properties)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output:
	// {"id":{"type":"string","readOnly":true},"name":{"type":"string","title":"Pet name","maxLength":20,"minLength":1,"pattern":"^[a-z]+$"},"score":{"type":"number","format":"double","multipleOf":0.5,"maximum":10,"minimum":0}}
}

// This example mounts a docs page for the specification: Swagger UI
// by default, Scalar or an embedded copy of either on request.
func ExampleFizz_Docs() {
	f := fizz.New()
	f.GET("/openapi.json", nil, f.OpenAPI(&openapi.Info{Title: "Pets", Version: "1.0.0"}, "json"))
	f.Docs("/docs", "/openapi.json")                              // Swagger UI, the default
	f.Docs("/scalar", "/openapi.json", fizz.ScalarUI())           // Scalar, from a CDN
	f.Docs("/offline", "/openapi.json", fizz.EmbeddedSwaggerUI()) // Swagger UI, from the binary

	for _, path := range []string{"/docs", "/scalar", "/offline"} {
		w := httptest.NewRecorder()
		f.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		body := w.Body.String()
		fmt.Println(path, w.Code, strings.Contains(body, "swagger-ui"), strings.Contains(body, "scalar"))
	}
	// Output:
	// /docs 200 true false
	// /scalar 200 false true
	// /offline 200 true false
}

// This example serves the specification as YAML. The YAML is derived
// from the JSON document, so both have the same content.
func Example_yaml() {
	f := fizz.New()
	f.GET("/openapi.yaml", nil, f.OpenAPI(&openapi.Info{Title: "Pets", Version: "1.0.0"}, "yaml"))

	w := httptest.NewRecorder()
	f.ServeHTTP(w, httptest.NewRequest("GET", "/openapi.yaml", nil))
	first, _, _ := strings.Cut(w.Body.String(), "\n")
	fmt.Println(w.Header().Get("Content-Type"))
	fmt.Println(first)
	// Output:
	// application/yaml
	// openapi: 3.2.1
}

// This example reports a generation error. Registration never panics:
// the failing operation is left out of the document, the route is still
// served, and Err joins the errors that are not warnings.
func ExampleFizz_Err() {
	f := fizz.New()
	h := tonic.Handler(func(*gin.Context) error { return nil }, 200)
	f.GET("/a", []fizz.OperationOption{fizz.ID("same")}, h)
	f.GET("/b", []fizz.OperationOption{fizz.ID("same")}, h)

	err := f.Err()
	fmt.Println(err)
	var opErr *openapi.OperationError
	if errors.As(err, &opErr) {
		fmt.Println(opErr.Method, opErr.Path)
	}
	// Output:
	// GET /b (same): ID same is already used by another operation
	// GET /b
}

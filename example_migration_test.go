package fizz_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// This example builds the schema and the parameters of the OpenAPI model
// in its v0.30 shape: a TypeSet, json.Number bounds, a pointer Explode, and
// an OrRef that holds a Value or a Ref.
func Example_migrateModel() {
	limit := &openapi.ParameterOrRef{Value: &openapi.Parameter{
		Name:    "limit",
		In:      "query",
		Explode: new(true),
		Schema: &openapi.Schema{
			Type:     openapi.Types("integer", "null"), // was Type: "integer", Nullable: true
			Minimum:  openapi.Num(0),                   // was Minimum: 0, an int
			Examples: []any{10},                        // was Example: 10
		},
	}}
	shared := &openapi.ParameterOrRef{Ref: &openapi.Reference{Ref: "#/components/parameters/Page"}}

	fmt.Println(limit.Value.Name, shared.Ref.Ref)
	for _, p := range []*openapi.ParameterOrRef{limit, shared} {
		b, err := json.Marshal(p)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))
	}
	// Output:
	// limit #/components/parameters/Page
	// {"name":"limit","in":"query","schema":{"type":["integer","null"],"examples":[10],"minimum":0},"explode":true}
	// {"$ref":"#/components/parameters/Page"}
}

// This example shows what replaces the binder fields of SpecGenConfig:
// they are fixed to the names of tonic, and a different value is an error.
// The names that fizz owns are set with WithTagNames.
func Example_migrateTagNames() {
	_, err := openapi.NewGenerator(&openapi.SpecGenConfig{ValidatorTag: "binding"})
	fmt.Println(err)
	fmt.Println(openapi.BinderTags.Validate, openapi.BinderTags.Query)

	f := fizz.New(fizz.WithTagNames(openapi.TagNames{Description: "doc", Example: "-"}))
	fmt.Println(f.Err())
	// Output:
	// openapi: SpecGenConfig.ValidatorTag = "binding": binder tags are fixed to "validate"
	// validate query
	// <nil>
}

type migrateBadIn struct {
	Events chan int `json:"events"`
}

// This example reads the errors of the generation. Panics are gone: every
// problem is recorded, an operation's problems are wrapped in an
// *openapi.OperationError, and errors.As finds what is inside.
func Example_migrateErrors() {
	f := fizz.New()
	f.POST("/jobs/:id", []fizz.OperationOption{fizz.ID("createJob")},
		tonic.Handler(func(*gin.Context, *migrateBadIn) error { return nil }, 201))

	for _, err := range f.Errors() {
		var op *openapi.OperationError
		var typ *openapi.TypeError
		var warn *openapi.Warning
		switch {
		case errors.As(err, &warn):
			fmt.Println("warning:", warn.Msg)
		case errors.As(err, &typ):
			fmt.Println("error:", typ.Message, typ.Type)
		}
		if errors.As(err, &op) {
			fmt.Println("  operation:", op.Method, op.Path, op.ID)
		}
	}
	fmt.Println(f.Err() != nil)
	// Output:
	// error: unsupported type chan int
	//   operation: POST /jobs/{id} createJob
	// warning: path parameter "id" is declared in the path but missing from the input: synthesized as a required string
	// true
}

// This example registers options that cannot all be applied. Each problem
// is recorded and the operation is still documented.
func Example_migrateRecordedErrors() {
	f := fizz.New()
	f.POST("/pets", []fizz.OperationOption{
		fizz.ID("createPet"),
		fizz.RequestBody("The pet", true,
			fizz.Content("application/json", exPet{}),
			fizz.Content("application/json", exPet{})),
		fizz.CustomizeOperation(func(*openapi.Operation) error { return errors.New("not today") }),
	}, tonic.Handler(func(*gin.Context, *exPet) error { return nil }, 201))

	for _, err := range f.Errors() {
		fmt.Println(err)
	}
	op := f.Generator().API().Paths["/pets"].POST
	fmt.Println(op.ID, slices.Sorted(maps.Keys(op.RequestBody.Content)))
	// Output:
	// POST /pets (createPet): RequestBody: media type "application/json" is given more than once: the first is kept
	// POST /pets (createPet): CustomizeOperation: not today
	// createPet [application/json]
}

// This example shows that Generator().API() returns a copy: change the
// document with the setters, a hook or an option, not through the copy.
func Example_migrateAPICopy() {
	f := fizz.New()
	doc := f.Generator().API()
	doc.Info.Title = "changed through the copy"
	fmt.Printf("%q\n", f.Generator().API().Info.Title)

	f.Generator().SetInfo(&openapi.Info{Title: "Pets", Version: "1.0.0"})
	fmt.Printf("%q\n", f.Generator().API().Info.Title)
	// Output:
	// ""
	// "Pets"
}

func listPets(*gin.Context) error { return nil }

type petAPI struct{}

func (*petAPI) get(*gin.Context) error { return nil }

// This example registers handlers without fizz.ID. The default operation ID
// is the name of the handler; if it is taken, the package-qualified name
// is tried, then the method and path.
func Example_migrateOperationIDs() {
	f := fizz.New()
	api := &petAPI{}
	f.GET("/pets", nil, tonic.Handler(listPets, 200))
	f.GET("/cats", nil, tonic.Handler(listPets, 200))
	f.GET("/dogs", nil, tonic.Handler(listPets, 200))
	f.GET("/pets/:id", nil, tonic.Handler(api.get, 200))

	paths := f.Generator().API().Paths
	for _, p := range []string{"/pets", "/cats", "/dogs", "/pets/{id}"} {
		fmt.Println(p, paths[p].GET.ID)
	}
	// Output:
	// /pets listPets
	// /cats fizz_test.listPets
	// /dogs get_dogs
	// /pets/{id} get
}

type migrateModel struct {
	Wait time.Duration `json:"wait"`
	Home url.URL       `json:"home"`
	Nick *string       `json:"nick"`
	Data []byte        `json:"data"`
}

// This example prints the schema of types whose mapping changed: a
// Duration is an integer of nanoseconds, a url.URL an object (a
// component), a pointer accepts null, and bytes are base64.
func Example_migrateTypes() {
	f := fizz.New()
	f.GET("/model", []fizz.OperationOption{fizz.ID("getModel")},
		tonic.Handler(func(*gin.Context) (*migrateModel, error) { return nil, nil }, 200))

	schemas := f.Generator().API().Components.Schemas
	props := schemas["Fizz_testMigrateModel"].Properties
	for _, name := range []string{"wait", "home", "nick", "data"} {
		b, err := json.Marshal(props[name])
		if err != nil {
			panic(err)
		}
		fmt.Println(name, string(b))
	}
	fmt.Println(schemas["Fizz_testMigrateModel"].Properties["home"].Ref != "")
	// Output:
	// wait {"type":"integer","format":"int64"}
	// home {"$ref":"#/components/schemas/UrlURL"}
	// nick {"type":["string","null"]}
	// data {"type":"string","contentEncoding":"base64"}
	// true
}

type migrateEnumIn struct {
	Status Status `query:"status"`
}

// This example uses a type that implements Enumer as a query parameter.
// The parameter references the component of the type, where it used to
// inline a plain string.
func Example_migrateEnumRef() {
	f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
	f.GET("/accounts", []fizz.OperationOption{fizz.ID("listAccounts")},
		tonic.Handler(func(*gin.Context, *migrateEnumIn) error { return nil }, 200))

	api := f.Generator().API()
	b, err := json.Marshal(api.Paths["/accounts"].GET.Parameters[0].Value.Schema)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output:
	// {"$ref":"#/components/schemas/Status"}
}

type migrateDeleteIn struct {
	ID     int    `path:"id"`
	Reason string `json:"reason" validate:"required"`
}

// This example documents a DELETE with a body. tonic binds a DELETE body,
// so fizz documents it; GET and HEAD bodies are still left out.
func Example_migrateDeleteBody() {
	f := fizz.New()
	f.DELETE("/pets/:id", []fizz.OperationOption{fizz.ID("deletePet")},
		tonic.Handler(func(*gin.Context, *migrateDeleteIn) error { return nil }, 204))

	body := f.Generator().API().Paths["/pets/{id}"].DELETE.RequestBody
	fmt.Println(body != nil, body.Required, slices.Sorted(maps.Keys(body.Content)))
	// Output:
	// true true [application/json]
}

type migrateYAMLValue struct{}

func (migrateYAMLValue) MarshalJSON() ([]byte, error) { return []byte(`"from json"`), nil }
func (migrateYAMLValue) MarshalYAML() (any, error)    { return "from yaml", nil }

// This example shows that the YAML document is derived from the JSON
// one. The MarshalYAML method of a user type is ignored, an example
// included.
func Example_migrateYAML() {
	f := fizz.New()
	f.GET("/pets", []fizz.OperationOption{
		fizz.ID("listPets"),
		fizz.Response("409", "Conflict", exPet{}, nil, migrateYAMLValue{}),
	}, tonic.Handler(listPets, 200))

	w := httptest.NewRecorder()
	f.SpecHandler("yaml").ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	fmt.Println(strings.Contains(body, "from json"), strings.Contains(body, "from yaml"))
	// Output:
	// true false
}

// This example shows that the operation of a request is a copy: changing
// it affects neither the specification nor the next call.
func Example_migrateOperationFromContext() {
	f := fizz.New()
	f.GET("/me", []fizz.OperationOption{fizz.ID("getMe"), fizz.Summary("Who am I")},
		tonic.Handler(func(c *gin.Context) error {
			op, err := fizz.OperationFromContext(c)
			if err != nil {
				return err
			}
			op.Summary = "changed"
			again, _ := fizz.OperationFromContext(c)
			fmt.Println(op.ID, again.Summary)
			return nil
		}, 204))

	f.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/me", nil))
	fmt.Println(f.Generator().API().Paths["/me"].GET.Summary)
	// Output:
	// getMe Who am I
	// Who am I
}

// This example owns the tonic bind hook. WithoutBindHook keeps New from
// installing the hook of fizz, and fizz.BindHook(next) puts it back around
// your hook: the deepObject and cookie parameters bind again.
func Example_migrateBindHook() {
	prev := tonic.GetBindHook()
	defer tonic.SetBindHook(prev)

	f := fizz.New(fizz.WithoutBindHook())
	tonic.SetBindHook(fizz.BindHook(func(c *gin.Context, in any) error {
		return tonic.DefaultBindingHook(c, in)
	}))
	f.GET("/pets", []fizz.OperationOption{fizz.ID("listPets")},
		tonic.Handler(func(c *gin.Context, in *deepObjectIn) error {
			fmt.Println(in.Filter)
			return nil
		}, 204))

	f.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/pets?filter[color]=red", nil))
	// Output:
	// map[color:red]
}

// This example is a DocumentHook for a tool that cannot read OpenAPI 3.1
// and later: it declares 3.0.3 and rewrites the one construct that this
// hook knows, the nullable type union. It is a starting point, not a
// converter: see MIGRATION.md for what it leaves out.
func ExampleWithHooks_downgrade() {
	f := fizz.New(fizz.WithHooks(openapi.Hooks{
		Document: func(doc *openapi.OpenAPI) error {
			doc.OpenAPI = "3.0.3"
			var visit func(*openapi.Schema)
			visit = func(s *openapi.Schema) {
				if s == nil {
					return
				}
				if i := slices.Index(s.Type, "null"); i >= 0 {
					s.Type = slices.Delete(slices.Clone(s.Type), i, i+1)
					s.Extensions = map[string]any{"nullable": true}
				}
				for _, p := range s.Properties {
					visit(p)
				}
				visit(s.Items)
			}
			for _, s := range doc.Components.Schemas {
				visit(s)
			}
			return nil
		},
	}), fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
	f.GET("/model", []fizz.OperationOption{fizz.ID("getModel")},
		tonic.Handler(func(*gin.Context) (*migrateModel, error) { return nil, nil }, 200))

	doc, err := f.Document()
	if err != nil {
		panic(err)
	}
	b, err := json.Marshal(doc.Components.Schemas["migrateModel"].Properties["nick"])
	if err != nil {
		panic(err)
	}
	fmt.Println(doc.OpenAPI, string(b))
	// Output:
	// 3.0.3 {"type":"string","nullable":true}
}

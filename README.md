<h1 align="center">Fizz</h1>
<p align="center"><img src="images/lemon.png" height="200px" width="auto" alt="Gin Fizz"></p><p align="center">Fizz is a wrapper for <strong>Gin</strong> based on <strong><a href="https://github.com/loopfz/gadgeto/tree/master/tonic">gadgeto/tonic</a></strong>.</p>
<p align="center">It generates wrapping gin-compatible handlers that do all the repetitive work and wrap the call to your handlers. It can also generate an <i>almost</i> complete <strong>OpenAPI 3.2.1</strong> specification of your API.</p>
<p align="center">
   <a href="https://pkg.go.dev/github.com/wI2L/fizz?tab=doc"><img src="https://img.shields.io/static/v1?label=godev&message=reference&color=00add8&logo=go"></a>
   <a href="https://goreportcard.com/report/wI2L/fizz"><img src="https://goreportcard.com/badge/github.com/wI2L/fizz"></a>
   <a href="https://github.com/wI2L/fizz/actions"><img src="https://github.com/wI2L/fizz/workflows/CI/badge.svg"></a>
   <a href="https://codecov.io/gh/wI2L/fizz"><img src="https://codecov.io/gh/wI2L/fizz/branch/master/graph/badge.svg"/></a>
   <a href="https://github.com/wI2L/fizz/releases"><img src="https://img.shields.io/github/v/tag/wI2L/fizz?color=blueviolet&label=version&sort=semver"></a>
   <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg"></a>
   <a href="https://github.com/avelino/awesome-go"><img src="https://awesome.re/mentioned-badge.svg"></a>
<br>
</p>

---

Fizz v0.30 requires **Go 1.27 or later** and emits **OpenAPI 3.2.1** only. If you are coming from v0.23 or earlier, read [MIGRATION.md](MIGRATION.md): the generated document changed.

### Getting started

To create a Fizz instance, you can pass an existing *Gin* engine to `fizz.NewFromEngine`, or use `fizz.New` that will use a new default *Gin* engine.

```go
engine := gin.Default()
engine.Use(...) // register global middlewares

f := fizz.NewFromEngine(engine)
```

A Fizz instance implements the `http.HandlerFunc` interface, which means it can be used as the base handler of your HTTP server.
```go
srv := &http.Server{
   Addr:    ":4242",
   Handler: f,
}
srv.ListenAndServe()
```

#### Options

Both constructors accept options. This example reads the descriptions from a `doc` tag and documents pointers as plain (non-null) types.

<!-- example: ExampleNew_options -->
```go
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
```

| option | effect |
| ------ | ------ |
| `WithTagNames(openapi.TagNames)` | Rename the struct tags that fizz owns. See [Tags](#tags). |
| `WithSchemaNamer(func(reflect.Type) string)` | Name the component schema of a type. See [Naming](#naming). |
| `WithRequestBodyNamer(func(opID string, t reflect.Type) string)` | Name the component schema of a request body. |
| `WithOperationIDNamer(func(method, path, handlerName string) string)` | Derive the ID of an operation registered without `fizz.ID`. |
| `WithPointersAreNullable(bool)` | Whether a pointer also accepts `null`. Default `true`. |
| `WithAnnotationStyle(openapi.AnnotationStyle)` | How annotations are attached to a `$ref`: `openapi.SiblingAnnotations` (default) or `openapi.AllOfWrap`. |
| `WithDefaultMediaType(string)` | The media type of bodies. Default `tonic.MediaType()`. |
| `WithHooks(openapi.Hooks)` | Adjust the generated document. See [Hooks](#hooks). |
| `WithoutBindHook()` | Do not install the bind hook. See [deepObject](#deepobject-query-parameters). |

### Handlers

Fizz abstracts the `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD` and `TRACE` methods of a *Gin* engine. These functions accept a variadic list of handlers as the last parameter, but since Fizz relies on *tonic* to retrieve the informations required to generate the *OpenAPI* specification of the operation, **only one of the handlers registered MUST be wrapped with Tonic**. If several are, the error is recorded (see [Errors](#errors)), the route is still served and the operation is not documented.

In the following example, the `BarHandler` is a simple middleware that will be executed before the `FooHandler`, but the generator will use the input/output type of the `FooHandler` to generate the specification of the operation.

```go
func BarHandler(c *gin.Context) { ... }
func FooHandler(*gin.Context, *Foo) (*Bar, error) { ... }

f := fizz.New()
f.GET("/foo/bar", nil, BarHandler, tonic.Handler(FooHandler, 200))
```

However, registering only standard handlers that follow the `gin.HandlerFunc` signature is accepted, but the *OpenAPI* generator will ignore the operation and it won't appear in the specification.

### Operation informations

To enrich an operation, you can pass a list of optional `OperationOption` functions as the second parameters of the `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` and `HEAD` methods.

```go
// Set the default response description.
// A default status text will be created from the code if it is omitted.
fizz.StatusDescription(desc string)

// Set the summary of the operation.
fizz.Summary(summary string)
fizz.Summaryf(format string, a ...any)

// Set the description of the operation.
fizz.Description(desc string)
fizz.Descriptionf(format string, a ...any)

// Override the ID of the operation.
// Must be a unique string used to identify the operation among
// all operations described in the API.
fizz.ID(id string)

// Mark the operation as deprecated.
fizz.Deprecated(deprecated bool)

// Add an additional response to the operation.
// The example argument will populate a single example in the response schema.
// For populating multiple examples, use fizz.ResponseWithExamples.
// Notice that example and examples fields are mutually exclusive.
// model, header, and example may be `nil`.
fizz.Response(statusCode, desc string, model any, headers []*ResponseHeader, example any)

// ResponseWithExamples is a variant of Response that supports providing multiple examples.
// Examples argument will populate multiple examples in the response schema.
// For populating a single example, use fizz.Response.
// Notice that example and examples fields are mutually exclusive.
// model, header, and examples may be `nil`.
fizz.ResponseWithExamples(statusCode, desc string, model any, headers []*ResponseHeader, examples map[string]any)

// ResponseWith adds a response built from options: WithModel, WithHeaders,
// WithExample, WithExamples, WithContent (an additional media type) and
// WithResponseExtension.
fizz.ResponseWith(code, desc string, opts ...ResponseOption)

// Add examples to the default response.
fizz.DefaultResponseExamples(examples map[string]any)

// Add an additional header to the default response.
// Model can be of any type, and may also be `nil`,
// in which case the string type will be used as default.
fizz.Header(name, desc string, model any)

// Override the binding model of the operation.
fizz.InputModel(model any)

// Describe the request body: its description, whether it is required and,
// optionally, its media types. Contents replace the inferred one.
fizz.RequestBody(desc string, required bool, contents ...ContentOption)
fizz.Content(mediaType string, model any, encs ...EncodingOption)
fizz.Encoding(property, contentType string)

// Add a parameter, or merge it into the inferred one that has the same
// name and location.
fizz.Parameter(p *openapi.Parameter)

// Add tags to the operation, after the tag of its group.
fizz.Tags(tags ...string)

// Set the external documentation of the operation.
fizz.ExternalDocs(url, desc string)

// Set an "x-" extension of the operation.
fizz.Extension(key string, value any)

// Edit the finished operation. Runs after every other option.
fizz.CustomizeOperation(fn func(*openapi.Operation) error)

// Overrides the top-level security requirement of an operation.
// Note that this function can be used more than once to add several requirements.
fizz.Security(security *openapi.SecurityRequirement)

// Add an empty security requirement to this operation to make other security requirements optional.
fizz.WithOptionalSecurity()

// Remove any top-level security requirements for this operation.
fizz.WithoutSecurity()

// Add a Code Sample to the operation.
fizz.XCodeSample(codeSample *XCodeSample)

// Mark the operation as internal. The x-internal flag is interpreted by third-party tools and it only impacts the visual documentation rendering.
fizz.XInternal()
```

**NOTES:**
* `fizz.InputModel` allows to override the operation input regardless of how the handler implementation really binds the request parameters. It is the developer responsibility to ensure that the binding matches the OpenAPI specification. The deepObject and cookie parameters are not bound if the model differs from the handler input.
* The first argument of the `fizz.Response` method which represents an HTTP status code is of type *string* because the spec accept the value `default`. See the [Responses Object](https://spec.openapis.org/oas/v3.2.1.html#responses-object) documentation for more informations.
* An option that cannot be applied (a duplicate media type, a parameter without a name, an invalid extension key, a failing `CustomizeOperation`...) is recorded in [`Errors`](#errors); the operation is still documented, without what failed.

To describe the request body yourself, for instance a body in XML instead of the JSON one that is inferred from the input:

<!-- example: ExampleRequestBody -->
```go
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
```

`CustomizeOperation` edits the operation once every other option has been applied, whatever the position of the option:

<!-- example: ExampleCustomizeOperation -->
```go
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
```

To help you declare additional headers, predefined variables for Go primitives types that you can use as the third argument of the `fizz.Header` method are available:
```go
var (
   Integer  int32
   Long     int64
   Float    float32
   Double   float64
   String   string
   Byte     []byte
   Binary   []byte
   Boolean  bool
   DateTime time.Time
)
```

### Groups

Exactly like you would do with *Gin*, you can create a group of routes using the method `Group`. Unlike *Gin* own method, Fizz's one takes two other optional arguments, `name` and `description`. These parameters will be used to create a tag in the **OpenAPI** specification that will be applied to all the routes added to the group.

```go
grp := f.Group("/subpath", "MyGroup", "Group description", middlewares...)
```
If the `name` parameter is empty, the tag won't be created and it won't be used.

Subgroups of subgroups can be created to an infinite depth, according yo your needs.

```go
foo := f.Group("/foo", "Foo", "Foo group")

// all routes registered on group bar will have
// a relative path starting with /foo/bar
bar := f.Group("/bar", "Bar", "Bar group")

// /foo/bar/{barID}
bar.GET("/:barID", nil, tonic.Handler(MyBarHandler, 200))
```

The `Use` method can be used with groups to register middlewares after their creation.
```go
grp.Use(middleware1, middleware2, ...)
```

If a group describes a tag that already exists with a different description, the first description is kept and a warning is recorded. To set a tag, with its `summary`, `kind` or `parent`, use `f.Generator().SetTag`.

## Tonic

The subpackage *tonic* handles path/query/header/body parameters binding in a single consolidated input object which allows you to remove all the boilerplate code that retrieves and tests the presence of various parameters. The *OpenAPI* generator make use of the input/output types informations of a tonic-wrapped handler reported by *tonic* to document the operation in the specification.

The handlers wrapped with *tonic* must follow the following signature.
```go
func(*gin.Context, [input object ptr]) ([output object], error)
```
Input and output objects are both optional, as such, the minimal accepted signature is:
```go
func(*gin.Context) error
```

To wrap a handler with *tonic*, use the `tonic.Handler` method. It takes a function that follow the above signature and a default status code and return a `gin.HandlerFunc` function that can be used when you register a route with Fizz of *Gin*.

Output objects can be of any type, and will be marshalled to the desired media type.
Note that the input object **MUST always be a pointer to a struct**, or the tonic wrapping will panic at runtime.

### Operation IDs

Unless you set one with `fizz.ID`, the ID of an operation comes from the name of its handler: the function name without the import path, the `-fm` suffix of a method value and the generic arguments. A closure takes the name of the function that encloses it. If that ID is already used, fizz tries `pkg.Name`, then the method and path (`get_users_id`). An ID set with `fizz.ID` is never changed, and one that is already used is an error. Use `fizz.WithOperationIDNamer` to choose the default yourself.

```go
func MyHandler() gin.HandlerFunc {
   return tonic.Handler(func(c *gin.Context) error {}, 200)
}

f.GET("/foo", []fizz.OperationOption{
   fizz.ID("MyOperationID"),
}, MyHandler())
```

### Location tags

*tonic* uses three struct tags to recognize the parameters it should bind to the input object of your tonic-wrapped handlers:
- `path`: bind from the request path
- `query`: bind from the request query string
- `header`: bind from the request headers

Fizz binds a fourth location, [`cookie`](#cookies), and a `style:"deepObject"` form of `query`, [see below](#deepobject-query-parameters).

The fields that doesn't use one of these tags will be considered as part of the request body.

The value of each struct tag represents the name of the field in each location, with options.
```go
type MyHandlerParams struct {
   ID  int64     `path:"id"`
   Foo string    `query:"foo"`
   Bar time.Time `header:"x-foo-bar"`
}
```

*tonic* will automatically convert the value extracted from the location described by the tag to the appropriate type before binding.

**NOTE**: A path parameter is always required and will appear required in the spec regardless of the `validate` tag content. A path parameter that is in the route but not in the input type (a handler with no input, for instance) is documented as a required `string`, with a warning. A gin catch-all `/*name` is documented as `/{name}`, and the description of the parameter says that the value may contain a slash.

### Tags

Two kinds of struct tags drive the specification. The **binder tags** are read by *tonic*, or by fizz on its behalf, and their names are fixed: they are `openapi.BinderTags`. The **fizz tags** only enrich the specification, and each can be renamed with `fizz.WithTagNames(openapi.TagNames{...})`: a zero field keeps the default name, and `"-"` disables the tag.

| tag | default name | renameable / fixed | effect |
| --- | --- | --- | --- |
| `Description` | `description` | `TagNames.Description` | Description of the field or parameter. |
| `Example` | `example` | `TagNames.Example` | Example value, converted to the type of the field. Custom types provide it with [`Exampler`](#providing-examples-for-custom-types). |
| `Format` | `format` | `TagNames.Format` | Override the format of the field. |
| `Deprecated` | `deprecated` | `TagNames.Deprecated` | Mark the field deprecated. Accepts what `strconv.ParseBool` does; an invalid value is `false`. |
| `Title` | `title` | `TagNames.Title` | Title of the schema. |
| `Pattern` | `pattern` | `TagNames.Pattern` | Regular expression of a string. An invalid expression is recorded and dropped. |
| `ReadOnly` | `readOnly` | `TagNames.ReadOnly` | Mark the field read-only. Together with `writeOnly`, both are dropped with a warning. |
| `WriteOnly` | `writeOnly` | `TagNames.WriteOnly` | Mark the field write-only. |
| `Nullable` | `nullable` | `TagNames.Nullable` | `true` makes the field accept `null`, `false` removes it, whatever the pointer rule gives. |
| `MultipleOf` | `multipleOf` | `TagNames.MultipleOf` | The value must be a multiple of this positive number. |
| `AdditionalProperties` | `additionalProperties` | `TagNames.AdditionalProperties` | Only `false` is supported: the object accepts no other property. On a `$ref` it emits `unevaluatedProperties: false`. |
| `Inline` | `inline` | `TagNames.Inline` | Emit the schema of a struct field in place of its `$ref`. |
| `Style` | `style` | `TagNames.Style` | Serialization style of a parameter, checked per location: query `form`, `spaceDelimited`, `pipeDelimited`, `deepObject`; header `simple`; path `simple`, `label`, `matrix`; cookie `form`, `cookie`. |
| `AllowReserved` | `allowReserved` | `TagNames.AllowReserved` | Let a query parameter hold reserved characters unescaped. Query parameters only. |
| `Cookie` | `cookie` | `TagNames.Cookie` | Bind the field from a cookie. See [Cookies](#cookies). |
| `path` | `path` | fixed (binder tag) | Bind from the path. Always required. |
| `query` | `query` | fixed (binder tag) | Bind from the query string. Options: `,required`, `,default=`. |
| `header` | `header` | fixed (binder tag) | Bind from a header. Same options as `query`. |
| `enum` | `enum` | fixed (binder tag) | Comma separated list of the accepted values. Wins over the `Enumer` of the type at this field. |
| `default` | `default` | fixed (binder tag) | Value bound when none is passed. On a parameter, `required` is dropped from the spec with a warning. On a body field *tonic* never applies it, so `required` is kept, `default` is dropped and a warning is recorded. A `time.Duration` default on a parameter is an integer of nanoseconds. |
| `validate` | `validate` | fixed (binder tag) | [validator.v9](https://godoc.org/gopkg.in/go-playground/validator.v9) rules. See [Schema validation](#schema-validation). |
| `explode` | `explode` | fixed (binder tag) | Whether a query array is exploded (`1`, `t`, `T`, `TRUE`, `true`, `True`, `0`, `f`, `F`, `FALSE`). |

Tags are use-site annotations: they never change the component schema of a type that is shared by several fields. On a `$ref` they are emitted as siblings of the `$ref`, or inside an `allOf` with `fizz.WithAnnotationStyle(openapi.AllOfWrap)`.

<!-- example: Example_tags -->
```go
f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
f.GET("/pets/best", []fizz.OperationOption{fizz.ID("bestPet")},
	tonic.Handler(func(*gin.Context) (*Tagged, error) { return nil, nil }, 200))

b, err := json.Marshal(f.Generator().API().Components.Schemas["Tagged"].Properties)
if err != nil {
	panic(err)
}
fmt.Println(string(b))
```

### JSON/XML

The JSON/XML encoders usually omit a field that has the tag `"-"`. This behaviour is reproduced by the *OpenAPI* generator ; a field with this tag won't appear in the properties of the schema. The fields are selected as `encoding/json` does: embedding, dominance, `,string` and tag names are followed.

In the following example, the field `Input` is used only for binding request body parameters and won't appear in the output encoding while `Output` will be marshaled but will not be used for parameters binding.
```go
type Model struct {
	Input  string `json:"-"`
	Output string `json:"output" binding:"-"`
}
```

### Request body

If you want to make a request body field mandatory, you can use the tag `validate:"required"`. The validator used by *tonic* will ensure that the field is present.
To be able to make a difference between a missing value and the zero value of a type, use a pointer.

To explicitly ignore a parameter from the request body, use the tag `binding:"-"`.

The request body is documented for every method except `GET` and `HEAD`, which have no generally defined semantics for it per [RFC 9110](https://www.rfc-editor.org/rfc/rfc9110#section-9.3.1). `DELETE` bodies are documented, because *tonic* binds them. The body is named after the operation ID (`ListPetsInput`, see [Naming](#naming)), and `requestBody.required` is set when one of its fields is required.

### Schema validation

The *OpenAPI* generator recognize some tags of the [go-playground/validator.v9](https://godoc.org/gopkg.in/go-playground/validator.v9) package (the one *tonic* uses) and translate those to the keywords of the schema that are taken from the [JSON Schema definition](https://json-schema.org/draft/2020-12/json-schema-validation).

The supported rules are: `len`, `min`, `max`, `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `oneof`, `email`, `url`, `uri`, `uuid`, `uuid3`, `uuid4`, `uuid5`, `ipv4`, `ipv6`, `hostname`, `alpha`, `alphanum`, `numeric`, `hexadecimal` and `unique`, with `dive` and `keys`/`endkeys` for the elements of slices and maps. An OR group (`a|b`) becomes an `anyOf` when every alternative maps to a keyword.

Based on the type of the field that carry the tag, the fields `maximum`, `minimum`, `exclusiveMaximum`, `exclusiveMinimum`, `minLength`, `maxLength`, `minItems`, `maxItems`, `minProperties` and `maxProperties` of its **JSON Schema** will be filled accordingly. A field is required only if its `validate` tag has a single `required` rule at the root and no `omitempty`.

## OpenAPI specification

To serve the generated OpenAPI specification in either `JSON` or `YAML` format, use the handler returned by the `fizz.OpenAPI` method. If your server is not *Gin*, `f.SpecHandler("json")` returns an `http.Handler`.

To enrich the specification, you can provide additional informations. Head to the [OpenAPI 3.2.1 spec](https://spec.openapis.org/oas/v3.2.1.html#info-object) for more informations about the API informations that you can specify, or take a look at the type `openapi.Info` in the file [`openapi/spec_root.go`](openapi/spec_root.go).

```go
f := fizz.New()
infos := &openapi.Info{
   Title:       "Fruits Market",
   Description: `This is a sample Fruits market server.`,
   Version:     "1.0.0",
}
f.GET("/openapi.json", nil, f.OpenAPI(infos, "json"))
```

The document is built once per change of the generator and shared by the requests, so serving it while you register routes is safe.

#### Errors

**NOTE**: The generator never panics. Registration problems are recorded instead, and `f.Errors()` returns them in registration order. A problem is a warning (`IsWarning() bool` is true, the document is still complete), or an error that concerns one operation. `f.Err()` joins the errors that are not warnings, or is `nil`: call it before starting your API.

Errors that concern an operation are wrapped in an `*openapi.OperationError{Method, Path, ID, Err}`, so use `errors.As` to reach the `*openapi.FieldError`, `*openapi.TypeError` or `*openapi.CollisionError` inside.

<!-- example: ExampleFizz_Err -->
```go
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
```

Most problems are limited to a field or a parameter: the operation is kept without it. An operation that cannot be built (a duplicate `fizz.ID`, an input that is not a struct, a response code used twice) is left out of the document, and the route is still served. Nothing a failed operation created remains in the document.

#### Servers information

If the OpenAPI specification refers to an API that is not hosted on the same domain, or using a path prefix not included in the spec, you will have to declare server information. This can be achieved using the `f.Generator().SetServers` method.

```go
f := fizz.New()
f.Generator().SetServers([]*openapi.Server{
   {
      Description: "Fruits Market - production",
      URL:         "https://example.org/api/1.0",
   },
})
```

#### Security schemes

If your API requires authentication, you have to declare the security schemes that can be used by the operations. This can be achieved using the `f.Generator().SetSecuritySchemes` method.

```go
f := fizz.New()
f.Generator().SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
   "apiToken": {
      Value: &openapi.SecurityScheme{
         Type: "apiKey",
         In:   "header",
         Name: "x-api-token",
      },
   },
})
```

Once defined, the security schemes will be available for all operations. You can override them on an per-operation basis using the `fizz.Security()` function.

```go
fizz.Security(&openapi.SecurityRequirement{
   "apiToken": []string{},
})
```

#### Components

The output types of your handlers are registered as components within the generated specification. See [Naming](#naming) for how they are named, and [Interfaces](#interfaces) to control their schema. A component is written once per type; a field that uses it only references it with a `$ref`.

A pointer to a struct is documented as `anyOf: [{$ref}, {type: "null"}]`, other pointers get `"null"` in their `type`. Use `fizz.WithPointersAreNullable(false)` to document pointers as plain types, or the `nullable` tag per field.

To add a component yourself, or other parts of the document that fizz does not infer, use the `Register*` and `Set*` methods of `f.Generator()`: `RegisterSchema`, `RegisterType`, `RegisterParameter`, `RegisterResponse`, `RegisterExample`, `RegisterRequestBody`, `RegisterHeader`, `SetTag`, `SetTagGroups`, `SetSelf`, `SetJSONSchemaDialect`, `SetExternalDocs` and `SetExtension`.

### Naming

By default, the name of a component is the package name and the type name in _CamelCase_, concatenated, without the import path (the `main` package has no prefix). As such, please ensure that you don't use the same type name in two eponym package in your application: when two types claim a name, the first keeps it, a `*openapi.CollisionError` is recorded, and the other one is inlined. Whatever the source, a name is sanitized to `^[A-Za-z0-9._-]+$`.

The instance of a generic type is named after its type arguments: `GenericPage_models.Item`, `GenericPair_string_int`. A slice argument becomes `List_T`, a map `Map_K_V`, and a pointer is its element.

The names of the components can be customized, from the highest precedence to the lowest.

1. **Global override**, before registering your handlers:
   ```go
   f := fizz.New()
   f.Generator().OverrideTypeName(reflect.TypeOf(T{}), "OverridedName")
   ```
2. **Interface**: implement the `openapi.Typer` interface on your types.
   ```go
   func (*T) TypeName() string { return "OverridedName" }
   ```
   **WARNING:** You **MUST** not rely on the method receiver to return the name, because the method will be called on a new instance created by the generator with the `reflect` package.
3. **Namer**: `fizz.WithSchemaNamer` takes a function of the type. An empty result selects the default name.

Operation IDs and request bodies have their own namers, `fizz.WithOperationIDNamer` and `fizz.WithRequestBodyNamer` (the default body name is the operation ID with its first letter upper-cased, followed by `Input`):

<!-- example: Example_naming -->
```go
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
```

### Interfaces

The spec generator creates OpenAPI schemas for your types based on their [reflection kind](https://golang.org/pkg/reflect/#Kind). To control the schema of a named type, implement one of these interfaces, on the value or on the pointer receiver. A type that implements any of them is always a component, a named scalar included, and the fields that use it reference it.

| interface | method | effect |
| --------- | ------ | ------ |
| `openapi.SchemaProvider` | `OpenAPISchema() *openapi.Schema` | Replaces the reflected schema. The type is not inspected. |
| `openapi.Enumer` | `Enum() []any` | Sets the `enum` of the component, converted to the kind of the type. |
| `openapi.SchemaCustomizer` | `CustomizeSchema(*openapi.Schema) error` | Edits the schema once it is built. On an error the schema keeps its previous state. |

They run in this order, once per type: reflection or `SchemaProvider`, then `Enumer`, then `SchemaCustomizer`, then the `SchemaHook`. A field tag still wins at the use site: a field `enum` tag overrides the `Enumer` of its type for that field, and the component is not touched.

The types of the example below implement `Enumer` and `SchemaProvider`:

```go
type Status string

func (Status) Enum() []any { return []any{"active", "archived"} }

type Money struct{ Cents int64 }

func (Money) OpenAPISchema() *openapi.Schema {
	return &openapi.Schema{Type: openapi.Types("string"), Pattern: `^\d+\.\d{2}$`}
}

type Account struct {
	Status  Status `json:"status"`
	Balance Money  `json:"balance"`
}
```

<!-- example: Example_interfaces -->
```go
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
```

Older, simpler hooks still work. Implement `openapi.DataType` (`Type() string` and `Format() string`) to give a type a `type` and a `format`, or call `OverrideDataType` for the highest precedence. Implement `openapi.Nullable` to choose whether a type is nullable.

```go
type NullString sql.NullString

func (NullString) Nullable() bool { return true }

f.Generator().OverrideDataType(reflect.TypeOf(&UUIDv4{}), "string", "uuid")
```

**WARNING:** You **MUST** not rely on the method receivers to return the type and format, because these methods will be called on a new instance created by the generator with the `reflect` package.

##### Native and imported types support

Fizz supports some native and imported types. A schema with a proper type and format will be generated automatically, removing the need for creating your own custom schema.

* [`time.Time`](https://golang.org/pkg/time/#Time): a `date-time` string.
* [`time.Duration`](https://golang.org/pkg/time/#Duration): an `int64`, in nanoseconds. A body field default such as `"1s"` is emitted as nanoseconds.
* [`url.URL`](https://golang.org/pkg/net/url/#URL): an object, as `encoding/json` encodes it. Use `OverrideDataType` (`f.Generator().OverrideDataType(reflect.TypeFor[url.URL](), "string", "uri")`) to document it as a string.
* [`net.IP`](https://golang.org/pkg/net/#IP), `netip.Addr` and `netip.Prefix`: a string with the format `ip`.

  Note that, according to the doc, the inherent version of the address is a semantic property, and thus cannot be determined by Fizz. Therefore, the format returned is simply `ip`. If you want to specify the version, you can use the tags `format:"ipv4"` or `format:"ipv6"`.
* `uuid.UUID`, from the standard library, from [`gofrs/uuid`](https://godoc.org/github.com/gofrs/uuid#UUID) and from `google/uuid`: a `uuid` string.
* `[]byte`: a string with `contentEncoding: base64`. `json.RawMessage` is the empty schema.
* `big.Int` is an integer, `big.Float` and `big.Rat` are strings.

A type that implements `encoding.TextMarshaler` is a `string`. One that only implements `json.Marshaler` keeps the schema of its kind, and a warning points to `SchemaProvider`.

#### Markdown

> Throughout the specification description fields are noted as supporting CommonMark markdown formatting. Where OpenAPI tooling renders rich text it MUST support, at a minimum, markdown syntax as described by CommonMark 0.27. Tooling MAY choose to ignore some CommonMark features to address security concerns.
[*source*](https://spec.openapis.org/oas/v3.2.1.html#rich-text-formatting)

To help you write markdown descriptions in Go, a simple builder is available in the sub-package `markdown`. This is quite handy to avoid conflicts with backticks that are both used in Go for litteral multi-lines strings and code blocks in markdown.

#### Providing Examples for Custom Types
To be able to provide examples for custom types, they must implement `json.Marshaler` and the following interface:
```go
type Exampler interface {
       ParseExample(v string) (any, error)
}
```

If the custom type implements the interface, Fizz will pass the value from the `example` tag to the `ParseExample` method and use the return value as the example in the OpenAPI specification. An example must be encodable as JSON: one that is not (`NaN`, a channel) is recorded in `Errors()` and dropped. A `yaml.Marshaler` is ignored, because the YAML document is derived from the JSON one.

## Hooks

Hooks adjust what fizz generates when a tag or an interface is not enough. Pass them with `fizz.WithHooks(openapi.Hooks{...})`; calls compose in order. Any member may be `nil`.

| hook | signature | when |
| ---- | --------- | ---- |
| `Schema` | `func(*openapi.SchemaContext, *openapi.Schema)` | Once per component written from a Go type. |
| `Field` | `func(*openapi.FieldContext, *openapi.Schema) (skip bool)` | Per body property, after its tags. `true` leaves it out. Not called for parameters. |
| `Parameter` | `func(*openapi.ParameterContext, *openapi.Parameter) (skip bool)` | Per parameter, after its tags. `true` leaves it out, except a path parameter, which is kept with a warning. |
| `Operation` | `func(*openapi.OperationContext, *openapi.Operation) error` | Per operation, last. It gets a copy; on an error the changes are discarded and the error is recorded. It cannot change the ID. |
| `Document` | `func(*openapi.OpenAPI) error` | Per document, on a deep copy, when it is served or returned by `f.Document()`. |

The order in which everything applies is the precedence of the generator:

1. **Component stage**, once per named type, written once into `components/schemas`: reflection (or `SchemaProvider`, which replaces it), then `Enumer` and `SchemaCustomizer`, then `SchemaHook`.
2. **Use-site stage**, per field, parameter or operation, and never mutating a component: struct tags, then `FieldHook` and `ParameterHook`, then the operation options (`fizz.CustomizeOperation` runs last of them), then `OperationHook`. A field tag wins over a type interface here.
3. **Document stage**: `DocumentHook`, on a deep copy, once per generation of the document.

`DocumentHook` applies only to the served output (`f.SpecHandler`, `f.OpenAPI`) and to `f.Document()`. It never runs for `f.Generator().API()`, which returns a deep copy of the unhooked document. If the hook fails, the document is unavailable: the handler answers `500` and the error is recorded once per generation.

The other hooks run while the generator is locked: they must not call methods of the `Generator`, and a panic is not recovered. They affect what is generated after they are set, so pass them to `fizz.New` before you register routes.

This example adds an `x-tagGroups` extension to the served document. The hook edits a copy, so `Generator().API()` is unchanged:

<!-- example: ExampleWithHooks -->
```go
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
```

## deepObject query parameters

A query parameter can use the OpenAPI `deepObject` style, `?filter[color]=red&filter[size]=m`. Tag a map or a struct field:

```go
type ListPetsInput struct {
	Filter map[string]string `query:"filter" style:"deepObject"`
}
```

The parameter is documented with `style: deepObject`, `explode: true` and an inline object schema, which is never nullable. The type of the field is a `map[string]S`, a struct, or a pointer to either, where `S` is a scalar (a string, a bool, an integer, a float, or a type that implements `encoding.TextUnmarshaler`) or a slice of scalars. A struct member is named after its `json` tag, matched exactly; one with `json:"-"`, or unexported, is skipped.

**Nesting policy**, enforced when binding:
- One level only. `filter[a][b]=x` is an error (`bind.ErrNestedKey`), and the request is answered `400`.
- The values are scalars or slices of scalars. A slice is filled by repeating the key, `filter[tag]=a&filter[tag]=b`; a scalar key repeated, an unclosed or empty bracket, or a value that does not convert is a `400`.
- A parameter whose keys are all absent stays `nil`.
- A deepObject parameter is required only with `validate:"required"` on the field. The `,required` option and a default value are generation errors, because *tonic* would bind them. `explode:"false"` is undefined for `deepObject`: it is a warning and `explode` stays `true`. A struct member with `validate:"required"` is listed in the `required` of the schema.

*tonic* knows nothing of `deepObject`, so fizz binds it through a **global bind hook**:
- `fizz.New` and `fizz.NewFromEngine` install it by calling `fizz.InstallBindHook()`. It wraps the current *tonic* bind hook, which is process-global: it is idempotent and safe for concurrent use, a hook that was set before `New` is kept, and it only acts on routes registered with fizz.
- `fizz.WithoutBindHook()` opts out. The parameters are still documented but not bound, unless you install the hook yourself. It does not remove a hook that an earlier `New` installed.
- A hook that you set after `New` replaces fizz's. Compose it with `tonic.SetBindHook(fizz.BindHook(next))`, or call `fizz.InstallBindHook()` again. A request that reaches a route whose hook was replaced logs a warning in gin debug mode.

<!-- example: Example_deepObject -->
```go
f := fizz.New()
f.GET("/pets", []fizz.OperationOption{fizz.ID("listPets")},
	tonic.Handler(func(c *gin.Context, in *deepObjectIn) error {
		fmt.Println(in.Filter)
		return nil
	}, 204))

req := httptest.NewRequest("GET", "/pets?filter[color]=red", nil)
f.ServeHTTP(httptest.NewRecorder(), req)
```

## Cookies

A field with a `cookie` tag is documented as an `in: cookie` parameter and bound from the request cookies. Only scalar fields (or pointers to scalars) are cookies; a struct, a slice or a map is a recorded error and the field is skipped. The tag is read by the same bind hook as deepObject, so it follows the same installation rules.

- **Present**: the value is converted to the type of the field, from the raw value of the cookie. It is **not** URL-decoded.
- **Absent**, with a `default` tag: the field takes the default.
- **Present but empty**: the field gets the empty value (`""` for a string). The default is **not** applied.
- **Absent**, without a default: the field is left alone.

Whether a cookie is required is left to `validate:"required"`, which *tonic* checks after the binding. Use the `default` tag, as below, to give a cookie a default that is bound: the `,default=` option of a `cookie` tag is documented in the spec but not applied at runtime.

<!-- example: Example_cookie -->
```go
f := fizz.New()
f.GET("/me", []fizz.OperationOption{fizz.ID("getMe")},
	tonic.Handler(func(c *gin.Context, in *cookieIn) error {
		fmt.Printf("session=%s lang=%s\n", in.Session, in.Lang)
		return nil
	}, 204))

req := httptest.NewRequest("GET", "/me", nil)
req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})
f.ServeHTTP(httptest.NewRecorder(), req)
```

## Docs UI

`f.Docs(path, specURL string, ui ...fizz.DocsUI)` registers a `GET` route that serves an interactive page for the document served at `specURL`. The route is not part of the specification.

- **Swagger UI is the default**: `f.Docs("/docs", "/openapi.json")`.
- `fizz.ScalarUI()` selects [Scalar](https://scalar.com), `fizz.SwaggerUI()` selects Swagger UI explicitly. Both accept `docsui` options (`WithTitle`, `WithUIConfig`, `WithCSPNonce`, `WithAssetURLs`).
- `fizz.EmbeddedSwaggerUI()` and `fizz.EmbeddedScalarUI()` serve the assets from your binary at `<path>/assets/`, instead of from jsDelivr, so the page makes no request to a third-party host. Only a program that calls one of them carries the assets.
- The choice is always honored: fizz never falls back to another UI. Giving more than one is a mistake: it is recorded in `Errors()` and the first is used.
- The CDN pages pin the assets (Swagger UI 5.34.1 and Scalar 1.76.4, see [docs/compat.md](docs/compat.md)) with Subresource Integrity, set `crossorigin` and a `no-referrer` policy, and force Swagger UI's `validatorUrl` to `null`, so the page never sends your document to a validator service.
- Without *Gin*, `docsui.SwaggerUI(specURL)` and `docsui.ScalarUI(specURL)` return an `http.Handler`.

<!-- example: ExampleFizz_Docs -->
```go
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
```

<!-- P8.4: verify rendering notes -->
**Swagger UI and OpenAPI 3.2.** These are known upstream gaps, taken from the Swagger UI issue tracker and a preliminary spike; browser verification of the documents that fizz generates is still pending, so treat them as reported behaviour, not as guarantees:
- `itemSchema`, the 3.2 schema of sequential media types, is not rendered ([swagger-ui#11065](https://github.com/swagger-api/swagger-ui/issues/11065)). Fizz does not generate `itemSchema`; it matters for documents you extend with a hook.
- Tag `summary`, `kind` and `parent`, the 3.2 tag fields, may not be displayed.
- A `$ref` may be shown as an anonymous `object` in an inline response schema, and `x-` extensions are hidden by default.

Use `fizz.ScalarUI()` for fuller 3.2 rendering. The details are in [docs/compat.md](docs/compat.md).

## YAML

`f.OpenAPI(info, "yaml")` and `f.SpecHandler("yaml")` serve the document as `application/yaml`. The YAML is derived from the JSON document, so both always have the same content. The model carries no `yaml` tag, and a `MarshalYAML` method on a type of yours is ignored, for examples too. Use `openapi.MarshalJSON(doc)` and `openapi.MarshalYAML(doc)` to encode a document yourself.

<!-- example: Example_yaml -->
```go
f := fizz.New()
f.GET("/openapi.yaml", nil, f.OpenAPI(&openapi.Info{Title: "Pets", Version: "1.0.0"}, "yaml"))

w := httptest.NewRecorder()
f.ServeHTTP(w, httptest.NewRequest("GET", "/openapi.yaml", nil))
first, _, _ := strings.Cut(w.Body.String(), "\n")
fmt.Println(w.Header().Get("Content-Type"))
fmt.Println(first)
```

## Known limitations

- Only operations that are wrapped with *tonic* are documented. A plain `gin.HandlerFunc` route is served and left out of the specification, as is a route with more than one tonic-wrapped handler (the latter is an error in `Errors()`).
- *Gin* and *tonic* are the only router and binder. The binder tags are fixed to *tonic*'s. The `openapi`, `bind` and `docsui` packages import neither, and can be used with another router.
- The request body is not documented for `GET` and `HEAD`.
- The `additionalProperties` tag only accepts `false`.
- Types that JSON cannot encode (channels, functions, complex numbers) are recorded as errors and the field is left out of the schema.
- Recursive types are supported: a cycle is cut with a `$ref`, and an embedded cycle follows the field selection of `encoding/json`. A recursive type cannot be inlined with the `inline` tag.
- A default value of a parameter must match what *tonic* can bind: a `time.Duration` is an integer of nanoseconds, not `"1s"`.
- The `deepObject` style has the [nesting policy](#deepobject-query-parameters) above, and cookies are scalar only, without a style or `explode`.
- Fizz does not infer callbacks, webhooks or links. Add them with the `Generator` setters, `fizz.CustomizeOperation` or a hook.
- Swagger UI renders OpenAPI 3.2 only partly, see [Docs UI](#docs-ui).
- Tools that read OpenAPI 3.0 only cannot use the generated document as is. See [MIGRATION.md](MIGRATION.md) for a `DocumentHook` that rewrites it.

## Examples

Every go block of this document that follows an `<!-- example: ... -->` marker is the body of a runnable `Example` function of the repository, and a test fails if they differ. The input types they use are declared next to them, in `example_test.go` and `example_readme_test.go`.

A simple runnable API is available in `examples/market`.
```shell
go build
./market
# Retrieve the specification marshaled in JSON.
curl -i http://localhost:4242/openapi.json
```

## Credits

Fizz is based on [gin-gonic/gin](https://github.com/gin-gonic/gin) and use [gadgeto/tonic](https://github.com/loopfz/gadgeto/tree/master/tonic). :heart:

<p align="right"><img src="https://forthebadge.com/images/badges/built-with-swag.svg"></p>

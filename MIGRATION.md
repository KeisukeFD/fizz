# Migrating from fizz v0.23 to v0.30

v0.30 is a rewrite of the generator. The way you register routes is the same: `fizz.New()`, `NewFromEngine(e)`, `GET`, `Group`, `fizz.ID`, `fizz.Response` and the other options keep their names and signatures. What changed is the **document** (OpenAPI 3.0.1 became 3.2.1, and many schemas changed), the **OpenAPI model** in package `openapi`, and how **errors** are reported.

The module path changed from `github.com/wI2L/fizz` to `github.com/KeisukeFD/fizz`: update your imports (and `go get github.com/KeisukeFD/fizz`). The release is `v0.30.x`, then `v1.0.0` once it is stable: expect small API corrections between the two.

Contents: [Requirements](#requirements) · [Go API](#go-api-changes-that-can-break-your-build) · [Generated document](#generated-document-changes-that-break-your-snapshots-and-clients) · [Recipes](#recipes) · [Checklist](#checklist)

## Requirements

- **Go 1.27 or later.** The module declares `go 1.27`, because the document is encoded with `encoding/json/v2`, which is generally available in Go 1.27. Go 1.26 and older are excluded. A toolchain older than 1.27 with `GOTOOLCHAIN=local` refuses to build the module; with the default `GOTOOLCHAIN=auto` Go 1.21 and later download 1.27 by themselves, unless your CI forbids it. If you cannot move to Go 1.27, stay on `v0.23.0`.
- gin is v1.12 (it was older). YAML no longer goes through `gopkg.in/yaml.v2`.
- The validator is still validator.v9 through `gadgeto` v0.9.0. Nothing changes there.

## Go API changes that can break your build

### The OpenAPI model: `SchemaOrRef`, `.Value`, `.Ref`, `TypeSet`, `json.Number`, `Explode`

The model follows OpenAPI 3.2 and JSON Schema 2020-12. If you only register routes you will not notice. If you read or build `openapi.*` values (in a test, a hook, a tool), these changed:

| v0.23 | v0.30 |
| ----- | ----- |
| `SchemaOrRef` (a struct that embeds `*Schema` and `*Reference`) | `Schema` holds everything. `SchemaOrRef` is kept as a deprecated alias of `Schema`. A reference is a `Schema` with `Ref` set (a plain `string`), and it may carry sibling keywords. |
| `ParameterOrRef`, `ResponseOrRef`, `HeaderOrRef`, `MediaTypeOrRef`, `ExampleOrRef`, `SecuritySchemeOrRef`: structs that embed the value and a `*Reference` | Aliases of the generic `OrRef[T]{Ref *Reference; Value *T}`. Read `p.Value.Name` instead of `p.Name`, and `p.Ref.Ref` instead of `p.Reference.Ref`. A value with neither is `openapi.ErrNilOrRef` when encoded. |
| `Type string` | `Type TypeSet`: `openapi.Types("string")`. A single type encodes as a string, several as an array. |
| `Nullable bool` | Gone: use a type union (`openapi.Types("string", "null")`) or `Schema.MakeNullable()`. |
| `Example interface{}` | `Examples []any`. |
| `Minimum`, `Maximum`, `MultipleOf int`, `ExclusiveMinimum bool` | `json.Number`: `openapi.Num(0)`. The exclusive bounds are numbers, as in 3.1. |
| `MaxLength`, `MinLength`, `MaxItems`... `int` | `*uint64`. |
| `AllOf`, `OneOf`, `AnyOf *SchemaOrRef` | `[]*Schema`. |
| `Parameter.Explode bool` | `Explode *bool`, so that an explicit `false` is emitted. |
| `yaml` struct tags and `MarshalYAML` methods | Removed. See [YAML](#yaml). |

Every extensible object has an `Extensions map[string]any` (the `x-` keys, encoded inline) and a `SetExtension(key, value) error` that checks the key. Decoding a document into the model uses the `encoding/json/v2` defaults: names are case-sensitive, a duplicate member is an error, and unknown members are kept and written back.

The example builds a parameter and a reference in the new shape:

<!-- example: Example_migrateModel -->
```go
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
```

### `SpecGenConfig` becomes `TagNames` and `BinderTags`

The binder fields of `openapi.SpecGenConfig` (`ValidatorTag`, `PathLocationTag`, `QueryLocationTag`, `HeaderLocationTag`, `EnumTag`, `DefaultTag`) are deprecated. The binder tags are `openapi.BinderTags`, fixed to the names *tonic* uses, and `openapi.NewGenerator` returns an error if a field holds another value. `fizz.New` always passes *tonic*'s names, so only code that builds a `Generator` by hand is affected.

The tags that fizz owns (`description`, `example`, `format`, `deprecated`, plus the new `title`, `pattern`, `readOnly`, `writeOnly`, `nullable`, `multipleOf`, `additionalProperties`, `inline`, `style`, `allowReserved` and `cookie`) are `openapi.TagNames`, set with `fizz.WithTagNames`. A zero field keeps the default name and `"-"` disables the tag. The README has the [full table](README.md#tags).

<!-- example: Example_migrateTagNames -->
```go
_, err := openapi.NewGenerator(&openapi.SpecGenConfig{ValidatorTag: "binding"})
fmt.Println(err)
fmt.Println(openapi.BinderTags.Validate, openapi.BinderTags.Query)

f := fizz.New(fizz.WithTagNames(openapi.TagNames{Description: "doc", Example: "-"}))
fmt.Println(f.Err())
```

### Errors instead of panics: `Errors()` and `Err()`

Registering a route no longer panics. The cases that used to panic are recorded and the route is still served:

| v0.23 | v0.30 |
| ----- | ----- |
| A second tonic handler on a route panics. | Recorded; the route is served and not documented. |
| A duplicate default operation ID panics. | The ID falls back (see [Operation IDs](#operation-ids-operationid)). |
| A duplicate `fizz.ID` panics. | Recorded; the route is served and not documented. |
| A path parameter missing from the input fails. | A required `string` is synthesized, with a warning. |
| `f.OpenAPI(info, "toml")` panics. | Recorded; the handler answers `500`. |

`f.Errors()` returns what was recorded, in registration order. Its entries are different from before:

- **Entries that concern an operation are wrapped in an `*openapi.OperationError{Method, Path, ID, Err}`.** If you compared an entry with `==` or a type assertion to `*openapi.FieldError` or `*openapi.TypeError`, use `errors.As`, which finds the inner error. New kinds are `*openapi.CollisionError` (two types want the same component name) and `*openapi.Warning`.
- **Warnings are entries too.** A `*openapi.Warning` has `IsWarning() bool` and the `Method`, `Path` and `ID` of its operation.
- `f.Err()` is new: it joins the entries that are not warnings with `errors.Join`, and is `nil` if there are none. Test it before you start serving.

<!-- example: Example_migrateErrors -->
```go
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
```

A problem that is limited to a field or a parameter does not fail the operation. Every error of a `fizz.RequestBody` or `fizz.Content`, a `fizz.Parameter`, a `fizz.CustomizeOperation` and an `OperationHook` is recorded, and the operation is kept, without what failed (a failing customizer or hook leaves the operation as it was before it ran). Only what makes an operation impossible (a duplicate `fizz.ID`, an input that is not a struct, a response code used twice) leaves it out. A `DocumentHook` is different: it works on the whole document, so an error makes the document unavailable (the handler answers `500`) and is recorded once per generation.

<!-- example: Example_migrateRecordedErrors -->
```go
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
```

### `Generator().API()` returns a copy

`f.Generator().API()` used to return the live document, and a change to it changed what you served. It now returns a **deep copy** (or `nil` if the copy fails, which is recorded). To change the document, use the setters (`SetInfo`, `SetServers`, `SetTag`, `SetExtension`...), the options, or a [hook](README.md#hooks). `f.Document()` returns the document as it is served, with the `DocumentHook` applied.

<!-- example: Example_migrateAPICopy -->
```go
f := fizz.New()
doc := f.Generator().API()
doc.Info.Title = "changed through the copy"
fmt.Printf("%q\n", f.Generator().API().Info.Title)

f.Generator().SetInfo(&openapi.Info{Title: "Pets", Version: "1.0.0"})
fmt.Printf("%q\n", f.Generator().API().Info.Title)
```

The same goes for `fizz.OperationFromContext`: the operation travels in the request context (not in a gin key), and each call returns a copy.

<!-- example: Example_migrateOperationFromContext -->
```go
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
```

### YAML: `MarshalYAML`

The `yaml` tags and the `MarshalYAML` methods of the model are removed. The YAML document is derived from the JSON one (`openapi.MarshalYAML(doc)` re-encodes `openapi.MarshalJSON(doc)`), so the two always match, and it is served as `application/yaml`. **A `MarshalYAML` method on a type of yours is ignored**, in a response example too: if the JSON and the YAML of a value differed, the YAML now follows the JSON. An example that cannot be encoded as JSON (`NaN`, a channel) is recorded in `Errors()` and dropped.

<!-- example: Example_migrateYAML -->
```go
f := fizz.New()
f.GET("/pets", []fizz.OperationOption{
	fizz.ID("listPets"),
	fizz.Response("409", "Conflict", exPet{}, nil, migrateYAMLValue{}),
}, tonic.Handler(listPets, 200))

w := httptest.NewRecorder()
f.SpecHandler("yaml").ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
body := w.Body.String()
fmt.Println(strings.Contains(body, "from json"), strings.Contains(body, "from yaml"))
```

### The bind hook: `InstallBindHook` and `WithoutBindHook`

*tonic* does not know the `deepObject` style or cookies, so fizz binds them through a *tonic* bind hook. `fizz.New` and `fizz.NewFromEngine` now call `fizz.InstallBindHook()`, which wraps the current process-global hook; this is new global state, and it acts only on routes registered with fizz.

For cookies, the hook honors the `,default=` and `,required` options of the `cookie` tag, as *tonic* does for `query` and `header`: an absent cookie takes the default (the option wins over the `default` tag), and an absent `,required` cookie without a default is a 400. An empty cookie does not take the default (see [Cookies](README.md#cookies)).

- If your program sets its own hook with `tonic.SetBindHook` **before** `fizz.New`, nothing to do: the hook is kept and composed.
- If it sets one **after**, the hook of fizz is replaced. Wrap yours with `tonic.SetBindHook(fizz.BindHook(yours))`, or call `fizz.InstallBindHook()` again. Routes that have `deepObject` or cookie parameters log a warning in gin debug mode when the hook did not run.
- To opt out, pass `fizz.WithoutBindHook()`. Those parameters are then documented but not bound, unless you compose `fizz.BindHook` yourself, as below. It does not remove a hook that an earlier `New` installed.

<!-- example: Example_migrateBindHook -->
```go
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
```

### Runtime binding changes (Phase 10)

The runtime follows the styles that the document states. *tonic* binds after the hook, so the hook rewrites the request (`c.Request.URL` and `c.Params`) and the handler sees the rewritten values. These apply only to routes registered with fizz and to the fields described.

- A query array tagged `style:"spaceDelimited"` or `style:"pipeDelimited"` (documented `explode: false`, unless you also tag `explode:"true"`) is split on the space or `|` after URL decoding: `?ids=1%202` binds `[1 2]`, not an error or `["1 2"]`. With `explode:"false"`, an element that contains a comma is a 400.
- A scalar path parameter tagged `style:"label"` binds `v` from `.v`, and one tagged `style:"matrix"` binds `v` from `;name=v` (or from `;name` when empty). A value without that prefix is a 400, where it used to be bound as is. Arrays are not covered: *tonic* reads a path parameter as one value.
- A `header` field with a `default` tag takes that value when the request does not have the header. *tonic* reads an absent header as one empty value and never applied the `default` tag, so the field used to stay zero (or, for an `int`, `bool` or enum, the request failed with a 400). A header that is present but empty is left as it is, and the `,default=` option of the `header` tag is still applied by *tonic* itself (to an absent or empty header). The caller's `*http.Request` is not modified: the hook works on a copy.

## Generated document: changes that break your snapshots and clients

### `openapi` and the version

`openapi: 3.0.1` becomes `openapi: 3.2.1`. Only 3.2.1 is emitted; there is no switch for another version. A snapshot that contains the whole document differs on its first line: see [the snapshot recipe](#snapshot-tests).

### `nullable` is gone

OpenAPI 3.1 and later removed `nullable: true`. A pointer is now nullable by default (`fizz.WithPointersAreNullable(false)` turns that off, and the `nullable` tag overrides it per field):

| Go | v0.23 | v0.30 |
| -- | ----- | ----- |
| `*string` | `{"type":"string","nullable":true}` | `{"type":["string","null"]}` |
| `*Owner` | `{"$ref":"#/components/schemas/Owner"}` | `{"anyOf":[{"$ref":"#/components/schemas/Owner"},{"type":"null"}]}` |
| `any` | `{"nullable":true,"description":"Value of any type, including null"}` | `{}` |

The `$ref` case changed output that you may not expect: pointers to structs used to be plain references and are now nullable.

### Type mapping: `time.Duration`, `url.URL`, integers, bytes

<!-- example: Example_migrateTypes -->
```go
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
```

| Go | v0.23 | v0.30 |
| -- | ----- | ----- |
| `time.Duration` | `{"type":"string","format":"duration"}` | `{"type":"integer","format":"int64"}`: nanoseconds, as `encoding/json` encodes it. A `Duration` default is nanoseconds too (a parameter cannot parse `"1s"`; a body field may use it and is emitted as nanoseconds). |
| `url.URL` | `{"type":"string","format":"url"}` | An object component, as `encoding/json` encodes the struct. Use `OverrideDataType` or a `SchemaProvider` type for a string. |
| `int`, `uint8`... | `int` and the small sized types were `int32`. | `int` is `int64`, the sized types keep their size, and unsigned types get `minimum: 0`. |
| `[]byte` | `{"type":"string","format":"byte"}` | `{"type":"string","contentEncoding":"base64"}` |
| `float32` default `0.1` | `0.10000000149011612` | `0.1` (the literal of the tag is kept) |
| `map[int]string` | ignored, with an error | An object with `propertyNames` that matches integers. |
| `json.RawMessage` | `{"type":"array","items":{"type":"integer","format":"int32"}}` | `{}` |

A type that implements `encoding.TextMarshaler` is a `string`. A type that only has `MarshalJSON` keeps the schema of its kind and records a warning that points to `SchemaProvider`.

### Annotations no longer change shared components

A `description`, `default`, `enum` or validation on a field used to be written **into the shared component** that the field references, so two fields with different descriptions overwrote each other. They are now use-site annotations, siblings of the `$ref` (the default), or inside an `allOf` with `fizz.WithAnnotationStyle(openapi.AllOfWrap)`. A tool that ignores `$ref` siblings in 3.0 mode loses them: use `AllOfWrap` for it.

### A type with `Enumer`, `SchemaProvider` or `SchemaCustomizer` is a `$ref`

A named type that implements one of these new interfaces is always a component, a named string included. A parameter of such a type used to be an inline `string` and is now a `$ref`. If a tool dislikes `$ref` parameters, inline the schema in a hook, or use the field's `enum` tag instead of `Enumer`.

<!-- example: Example_migrateEnumRef -->
```go
f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
f.GET("/accounts", []fizz.OperationOption{fizz.ID("listAccounts")},
	tonic.Handler(func(*gin.Context, *migrateEnumIn) error { return nil }, 200))

api := f.Generator().API()
b, err := json.Marshal(api.Paths["/accounts"].GET.Parameters[0].Value.Schema)
if err != nil {
	panic(err)
}
fmt.Println(string(b))
```

### Validation: `validate` tags are parsed, not split

`validate` is read with a grammar parser. `min=0` is no longer dropped (`minimum: 0`), bounds are exact `json.Number`s, `gt` and `lt` are exclusive, `len` sets both bounds, `oneof` becomes `enum`, `dive` and `keys` apply to elements, and an OR group becomes `anyOf` when every alternative maps. A field is `required` only if the root of its tag has a single `required` rule and no `omitempty`: `required_if`, `required_with` or `dive,required` do not make it required. A malformed tag is a recorded error, not a panic.

### Request bodies: `required`, `DELETE`, naming

- **`DELETE` bodies are documented.** *tonic* binds them, so `DELETE` operations now have a `requestBody`. `GET` and `HEAD` do not.
- `requestBody.required` is `true` when one of the body fields is required.
- The body component is named `<OperationID with a capital>Input`, claimed through the same registry as the other components (`fizz.WithRequestBodyNamer` changes it). If the name is taken by another type, a `*openapi.CollisionError` is recorded and the body stays inline.

<!-- example: Example_migrateDeleteBody -->
```go
f := fizz.New()
f.DELETE("/pets/:id", []fizz.OperationOption{fizz.ID("deletePet")},
	tonic.Handler(func(*gin.Context, *migrateDeleteIn) error { return nil }, 204))

body := f.Generator().API().Paths["/pets/{id}"].DELETE.RequestBody
fmt.Println(body != nil, body.Required, slices.Sorted(maps.Keys(body.Content)))
```

### Operation IDs: `operationId`

The default `operationId` came from *tonic*'s handler name, which is lossy: `Get-fm` for a method value, `func1` for a closure, and a panic when two handlers shared it. It is now the name of the function: no import path, no `-fm`, no generic arguments, and a closure takes the name of the function that encloses it. On a collision it tries `pkg.Name` (the package is the last segment of the import path), then the method and path (`get_users_id`). An explicit `fizz.ID` never falls back: a duplicate is an error. `fizz.WithOperationIDNamer` replaces the default name; its collision fallback is the method and path.

Snapshots and clients that use `operationId` change: generated client method names follow it.

<!-- example: Example_migrateOperationIDs -->
```go
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
```

### Component names

The default name is still the package name and the type name (`PetsPet`), and the `main` package has no prefix. Changes: every name is sanitized to `^[A-Za-z0-9._-]+$`; generic types name their arguments (`GenericPage_models.Item`, `List_T` for a slice, `Map_K_V` for a map); on a collision the first type keeps the name, a `*CollisionError` is recorded and the other type is inlined. `fizz.WithSchemaNamer` sets a namer; `OverrideTypeName` and `Typer` keep their precedence.

### Parameters and paths

- A gin catch-all `/*name` is `/{name}`, with a note in the description that it may contain a slash.
- A path parameter that the route declares but the input lacks is synthesized as a required `string`, with a warning, instead of an error.
- A `query` or `header` option that is malformed (`,omitempty`, for instance), an invalid `explode` value or an unsupported parameter type is a recorded error that skips the field. A dash or an empty name is not a parameter.
- `explode:"false"` is emitted (it used to be lost with the `bool`).
- A default on a parameter makes it optional: `required` is dropped and a warning is recorded. On a body field, `required` is kept and the `default` is dropped, with a warning.

### v0.30 conformance fixes (Phase 10)

Found by checking the output against the official OpenAPI 3.2 examples and the spec text.

- A `query`, `header` or `path` parameter of map or struct type is a generation error (the field is skipped), as a cookie already was. Only `style:"deepObject"` query parameters may be a map or a struct. Types that tonic binds from a string stay allowed: scalars, `time.Time`, `encoding.TextUnmarshaler` types and slices of them.
- The `default` tag of a slice or array parameter is split on `,` and each element is converted with the element type, as tonic does: `default:"1,2,3"` on `[]int` is `"default": [1, 2, 3]`. It used to be a generation error. With `explode:"false"` the whole value is one element, as in tonic.
- A header parameter named `Accept`, `Content-Type` or `Authorization` (any case) is dropped from the document with a warning, since OAS ignores them. Use security schemes for `Authorization` and the request or response content for the other two.
- Registering the same method and path twice is an operation error that names both operations; the first one is kept (it was overwritten silently). `RegisterRequestBody` with a nil `Content` is an error.
- Two routes whose path templates differ only in their variable names (`GET /x/:a` and `POST /x/:c`) are an operation error naming both: OAS does not allow two such path keys. Rename the variable to the same name.
- A `fizz.Parameter` option with an `in` other than `path`, `query`, `querystring`, `header` or `cookie`, or with both `schema` and `content`, is an error and is not added. An option that sets `Content` replaces the inferred `schema` (and one that sets `Schema` replaces a `content`).
- A cookie parameter is emitted with `style: cookie` (OAS 3.2: no percent-encoding), which matches how the bind hook reads the raw value. An explicit `style:"form"` tag still wins.
- The `description` tag of a parameter is only on the parameter, no longer repeated in `parameter.schema.description`. Body and field schemas keep their descriptions.
- The served document (`SpecHandler`, `Fizz.Document`) omits an empty `components` and an empty `paths`. A document with none of paths, components and webhooks keeps `"paths": {}`, which OAS 3.2 requires. `Generator().API()` is unchanged.
- A slice or array `header` or `path` parameter (a struct field, or a `fizz.Parameter` option whose schema is an array) is a generation error: `tonic binds header/path parameters from a single value; use a query parameter or a string field`. The field is skipped. `[]byte` and `net.IP` are strings, not slices. Real support for these needs a binding layer of fizz's own.
- An optional `header` parameter of a number, integer, boolean or enum type with no `default` records a generation warning: `absent optional header "X-Id" is bound as "" by tonic and returns 400; add a default or make it a string`. Add a `default` tag (the hook now applies it) or make the field a string.
- Nullable items (`[]*string` with an `enum`, or `validate:"dive,oneof=..."`) include `null` in their `enum`/`const`, like a nullable top-level schema.
- The spec model keeps an empty `parameters` on a path item and on a link when decoding and encoding.

### Responses

An empty examples map or a `nil` model no longer creates an empty media type. An empty description defaults to `Default response`, `<code> response` for a range such as `2XX`, the HTTP status text of a known code, and `Status <n>` otherwise. `"schema": null` no longer appears in a media type.

## Recipes

### Codegen tools

A tool that consumes the document sees `openapi: 3.2.1` and JSON Schema 2020-12 constructs that OpenAPI 3.0 does not have. Support differs per tool and changes quickly; at the time of writing:

- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) added initial OpenAPI 3.1 support in v2.8.0, built on [kin-openapi](https://github.com/getkin/kin-openapi), which reads 3.0 and 3.1. Neither documents 3.2 support.
- [ogen](https://github.com/ogen-go/ogen) supports 3.1 only partly (type arrays such as `[string, "null"]` were the long-standing gap).
- openapi-generator has partial 3.1 support; check its release notes for 3.2.

Check your tool's release notes: the first thing a tool that predates 3.2 does is usually to reject the `3.2.1` version string. You have three options, in this order of effort:

1. **Update the tool**, or report the missing 3.2 support upstream.
2. **Rewrite the document for it with a `DocumentHook`.** The hook runs on a copy, when the document is served and in `f.Document()`, so your Go code and `Generator().API()` keep the full model. For a tool that reads 3.1 but rejects the version string, `doc.OpenAPI = "3.1.0"` is enough as long as you use no 3.2-only feature (tag `summary`, `kind` and `parent`, `$self`, the `query` method, `itemSchema`, `in: querystring`, `additionalOperations`). For a tool that reads only 3.0.x the hook has to rewrite every construct that 3.0 lacks: the example below declares `3.0.3` and rewrites **only** the `type: [T, "null"]` unions into `nullable: true`. It still leaves: `anyOf` with a `null` branch (a nullable `$ref`), `$ref` siblings (use `AllOfWrap`), `examples` arrays, numeric `exclusiveMinimum` and `exclusiveMaximum`, `contentEncoding`, `const`, `propertyNames`, `unevaluatedProperties` and boolean schemas. Treat it as a starting point and run the result through your tool. (The `nullable` key is written into `Schema.Extensions` directly: `SetExtension` accepts only `x-` keys.)
3. **Stay on `v0.23.0`** for the services whose clients you cannot update. It keeps emitting 3.0.1. You lose everything in v0.30, and the Go 1.27 requirement does not apply to it.

<!-- example: ExampleWithHooks_downgrade -->
```go
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
```

### Snapshot tests

If you compare the served document with a file (a golden, a snapshot), every one of those tests fails once, on the first run with v0.30: the version, the nullable forms, the types, the operation IDs and the request bodies all differ. They do not become flaky: the output is deterministic (map keys are sorted, and the same program produces the same bytes), which `v0.30` tests with fresh generators and different `GOMAXPROCS`.

To regenerate them without losing what they protect:

1. Update fizz, run the tests, and **do not** regenerate yet. Read the diff of one snapshot against the table above: each line of the diff should fit one of the categories of [the generated document changes](#generated-document-changes-that-break-your-snapshots-and-clients).
2. Make the snapshot comparison structural, not textual, if it is not already (compare decoded JSON, and print the first differing JSON pointer), so that a reviewer sees `/paths/~1pets/get/operationId` and not a wall of text.
3. Regenerate with an `-update` flag, in one commit that contains only the snapshots, so that review is a review of the diff and not of the code. A helper:

   ```go
   var update = flag.Bool("update", false, "rewrite the snapshots")

   func assertSnapshot(t *testing.T, name string, doc *openapi.OpenAPI) {
   	t.Helper()
   	b, err := openapi.MarshalJSON(doc)
   	if err != nil {
   		t.Fatal(err)
   	}
   	var buf bytes.Buffer
   	if err := json.Indent(&buf, b, "", "  "); err != nil {
   		t.Fatal(err)
   	}
   	path := filepath.Join("testdata", name+".json")
   	if *update {
   		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
   			t.Fatal(err)
   		}
   	}
   	want, err := os.ReadFile(path)
   	if err != nil {
   		t.Fatal(err)
   	}
   	if !bytes.Equal(want, buf.Bytes()) {
   		t.Errorf("%s differs: run the tests with -update and review the diff", path)
   	}
   }
   ```
4. Snapshot `f.Document()` (what is served, with the hooks applied), not `f.Generator().API()`, unless your test is about the generator.
5. Validate the new snapshots against the OpenAPI 3.2 JSON Schema published at `spec.openapis.org/oas/3.2/schema`, with any JSON Schema 2020-12 validator. It catches a hand-written hook or extension that is not valid 3.2.
6. For the parts you care about, prefer an assertion on a JSON pointer over a whole-file snapshot: it survives the next change in the generator.

### Go 1.26 and older

The module declares `go 1.27` and uses `encoding/json/v2` and `new(expr)`. A toolchain older than Go 1.27 cannot build it (see [Requirements](#requirements)): upgrade the toolchain, or stay on `v0.23.0`. v0.30 is not tested with, and does not support, Go 1.26 and older.

## Checklist

- [ ] Go 1.27 in your `go.mod`, your CI and your Dockerfiles.
- [ ] Code that reads `openapi.*` values: `Schema` instead of `SchemaOrRef`, `.Value` and `.Ref` of `OrRef`, `TypeSet`, `json.Number`, `*bool` for `Explode`, `Examples`, no `Nullable`.
- [ ] `SpecGenConfig` binder fields removed or equal to *tonic*'s; tags renamed with `fizz.WithTagNames`.
- [ ] `f.Errors()` consumers use `errors.As` and know the `*openapi.OperationError` wrapper; startup checks use `f.Err()`.
- [ ] Nothing mutates `Generator().API()` or the operation from `OperationFromContext`.
- [ ] A `MarshalYAML` on your types is not relied on.
- [ ] If you set a *tonic* bind hook after `fizz.New`, wrap it with `fizz.BindHook`; or opt out with `fizz.WithoutBindHook()`.
- [ ] Snapshot tests regenerated and reviewed; client generators and codegen tools tried against the `3.2.1` document (or a `DocumentHook`).

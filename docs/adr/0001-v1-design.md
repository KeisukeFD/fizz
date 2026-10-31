# ADR 0001: fizz v0.30 / v1 design

- Status: accepted (P1.0)
- Date: 2026-10-30
- Sources: [PLAN-v2](../plan/PLAN-v2.md) §0, §1 and its addendum (D17–D21); [ACCEPTANCE](../plan/ACCEPTANCE.md) §2 and §5 with the user decisions.

This ADR freezes the decisions, the precedence model and the import rule, and it lists every new exported symbol with the task that owns it. A later task that needs to change any of this amends this ADR in the same PR.

## 1. Decisions

D1–D16 from PLAN-v2 §0 and D17–D21 from its addendum are frozen as written there. In short:

| # | Decision |
|---|---|
| D1 | Only OpenAPI **3.2.1** is emitted. `openapi.Version = "3.2.1"`. |
| D2 | `go 1.27` floor. CI runs 1.27.x and stable (blocking) and tip (non-blocking). |
| D3 | Router strategy C. `openapi/`, `bind/` and `docsui/` import neither gin nor gadgeto (§3). Spec and docs handlers are `http.Handler`. No gin or tonic types appear in hook contexts or options. The operation travels in the request context. |
| D4 | Same module path. Released as v0.30.x, then v1.0.0. |
| D5 | Embedded UI assets live in `docsui/embedded`, in the main module. |
| D6 | Pointers are nullable by default (`WithPointersAreNullable`). |
| D7 | Precedence has two stages (§2). A field tag beats a type interface at the use site. |
| D8 | Annotations are `$ref` siblings by default. `AllOfWrap` is an option. |
| D9 | Stay on gadgeto v0.9.0 (validator.v9). |
| D10 | A missing path param is synthesized as a required string, with a warning. |
| D11 | Plain gin handlers stay undocumented. |
| D12 | The document is serialized with `encoding/json/v2`: inline `Extensions` and `Deterministic(true)`. |
| D13 | DELETE bodies are documented. GET and HEAD bodies are not. |
| D14 | MarshalJSON+MarshalText → `string`. MarshalJSON only → kind schema + warning. Map keys follow encoding/json order. |
| D15 / D21 | Default + required: on a param, drop `required` and warn; on a body field, keep `required`, drop `default` and warn. |
| D16 | A Duration default on a param is integer ns only (`"1s"` is an error). On a body field, `"1s"` is accepted and emitted as ns. |
| D17 | The default docs UI is Swagger UI. Its 3.2 gaps are documented, with a pointer to `ScalarUI`. |
| D18 | An explicit UI choice is always honored. There is no silent fallback. |
| D19 | UI version pins are chosen at P8.1 and verified by the P0.6 spike. |
| D20 | `New`/`NewFromEngine` install the bind hook (process-global). `WithoutBindHook()` opts out. |

User decisions on the acceptance open items are also frozen: **O1** (docs UI API), **O2** (generic component names), **O4** (operation-ID collisions), **O16** (bind hook re-wrap by identity). See ACCEPTANCE "User decisions".

## 2. Precedence

1. **Component stage.** This runs once per named type and writes `components/schemas`:
   reflection, or `SchemaProvider` (which replaces it) → `Enumer` → `SchemaCustomizer` → `SchemaHook`.
   `Enumer` runs before `SchemaCustomizer` so that a customizer can edit `enum` (O9).
2. **Use-site stage.** This runs per field, param or operation and never mutates a component:
   struct tags → `FieldHook` / `ParameterHook` → operation options (`CustomizeOperation` runs last among them, O18) → `OperationHook`.
3. **Document stage.** `DocumentHook` runs on a deep clone at serve time and in `Fizz.Document()`, never in `Generator().API()`. It runs once per generation (O8).

## 3. Import rule

`./openapi/...`, `./bind/...` and `./docsui/...`, test files included, must not reach `github.com/gin-gonic/gin` or `github.com/loopfz/gadgeto`. The check is `internal/archtest.TestRouterAgnostic` (A01). It is Pending until P1.3, then blocking.

Corollaries:
- Gin→OpenAPI path translation and tonic config live in the root package.
- `bind` owns its tag struct (`bind.Tags`) and does not import `openapi` (O29).
- Root-package helpers that wrap `docsui` handlers use `gin.WrapH`.

## 4. Resolved open items (Opus)

O3, O5–O15, O17–O30 and O32–O36 are adopted as proposed in ACCEPTANCE §5. The deltas and clarifications:

- **Errors (O5, O6, O30).** `Errors() []error` returns a copy in registration order. The entries are:
  - `*Warning`, which has `IsWarning() bool`;
  - `*OperationError`, `*FieldError` and `*CollisionError`, which have `Unwrap()` where they wrap an error;
  - the sentinels: `openapi.ErrInvalidExtensionKey`, `ErrReservedExtension` and `ErrNilOrRef`, and `bind.ErrNestedKey` and `ErrTooManyKeys`.

  `Fizz.Err()` is `errors.Join` of the non-warnings.
- **Warning ownership.** D14 (P3.3) and D15 (P3.6) emit warnings before P4.8. So `openapi.Warning` is owned by **P3.3**, not P4.8. P4.8 still owns `Err()` and `OperationError`.
- **Clone (G11).** The method is `(*OpenAPI).Clone() (*OpenAPI, error)`, not a free function. A marshal failure such as a NaN extension is returned and never panics. G11 is written against this signature.
- **bind options (B34).** `WithMaxKeys` is a per-call `bind.QueryOption`: `bind.Query(r, dst, p, opts...)`. Plans stay cacheable per type, independent of call options.
- **Option ownership vs. behaviour.** `fizz.With*` options are all owned by **P6.1**, except `WithDefaultMediaType` (P4.4), `WithRequestBodyNamer` (P4.6), `WithOperationIDNamer` (P4.7) and `WithHooks` (P6.4). The behaviour behind an option may land earlier. Tagged tests that need an option and a behaviour from different tasks use a composite FixedBy, `"<behaviour>+<option>"`. For example, E03 is `P4.5+P6.1` and F34 is `P3.5+P6.1`. This replaces O36 for new tests.
- **Docs UI naming (O1 user decision).** The router-agnostic constructors are `docsui.SwaggerUI` and `docsui.ScalarUI`, and `embedded.SwaggerUI` and `embedded.ScalarUI`. PLAN-v2 P8.1's `docsui.Scalar` is superseded.
- **UUID types (P1.2).** Go 1.27 ships a standard-library `uuid` package (#62026). `openapi` recognises `uuid.UUID`, gofrs (`/v5` included) and google UUIDs by `PkgPath()+Name()`, and imports no third-party uuid module. The placeholder operation ID is `""`.
- **Path translation split (P1.3 / P4.3).** P1.3 moves gin→OpenAPI path translation into the root package (`path.go`) and translates only `:name` → `{name}`, which keeps output byte-identical. `*name` wildcards are translated in P4.3, together with their description note.
- **Phase 2 intermediate states (P2.2–P2.5).** Every Phase 2 task is its own commit and must stay green: build, vet, `-race`, `FIZZ_STALE=fail`, both builds. The plan's file lists cover only the model, but changing a model type breaks the generator's literals. So each task also makes the **minimal mechanical** generator edits needed to compile, pulling P2.6's renames forward:
  - Constructs that no longer exist are emitted in their nearest 3.2 form. Where the old generator set `nullable: true`, it now calls `MakeNullable()`, which emits `type: [T, "null"]` on inline schemas under the same conditions as before. P3.5 adds `$ref` nullability (`anyOf`) and the `PointersAreNullable` option. `example` → `examples:[v]`. A 3.0 boolean `exclusiveMinimum`/`exclusiveMaximum` becomes the numeric 3.1 form.
  - Goldens (`testdata/spec.*` and the legacy golden) are regenerated in the same commit, and Opus reviews the diff (G-GOLD). The legacy golden stays as a change log until P2.6 deletes it.
  - YAML may differ transiently until P2.5 (for example, `json.Number` quoted by yaml.v2). JSON is authoritative.
  - A kitchen-sink or acceptance case that starts passing early has its FixedBy relabelled to the task that made it pass (precedent: O34).
  - Until P2.3, the model is encoded with `encoding/json` v1 (`omitzero` is supported since Go 1.24).
- **`ExternalDocs` ownership.** `Schema.ExternalDocs` (P2.2) needs the type, so `openapi.ExternalDocs` lands in **P2.2**, not P2.1b.
- **JSON v2 bridge (P2.3).**
  - In Go 1.27 the v2 tag for the extension catch-all is `json:",embed"` (an "embedded fallback" map). D12's `inline` was the draft name.
  - Unknown members of any name are kept on decode and written back on encode.
  - Marshaling pins `json.DefaultOptionsV1()` plus `Deterministic(true)`, so v1 `json.Marshal(doc)` and `openapi.MarshalJSON(doc)` produce the same bytes.
  - Decoding uses v2 defaults: names are case-sensitive and duplicates are rejected. MIGRATION must mention this.
  - **P2.6 revisits the option set.** Once the generator no longer produces nil slices or maps, P2.6 switches to the v2 defaults if the G21 parity test still holds.
- **Binary size of `fizz.Embedded*UI`.** These make the root package import `docsui/embedded`. P8.3 must show (a test or a documented measurement) that a binary that does not call them carries no assets, because the linker drops the unreferenced `embed.FS`. If that does not hold, the selectors move to `docsui/embedded` and O1 goes back to the user.

## 5. Stub strategy (ACCEPTANCE §2, rule 2)

- New symbols are declared in `zz_stub_acceptance.go` files under `//go:build acceptance`, in `./`, `openapi/`, `bind/`, `docsui/` and `docsui/embedded/`. New packages also get an untagged `doc.go`, so A01 covers them now.
- Every stub declaration has a doc comment line `// FixedBy: Pn.m`. Stub bodies are no-ops or return zero values:
  - options are `func(*config) {}` or `func(*openapi.OperationInfo) {}`;
  - handlers are `http.NotFoundHandler()`;
  - error results are `errors.New("stub: Pn.m")`;
  - constants are `""`.
- When a task adds a real symbol, the untagged declaration collides with the stub, and `go vet -tags acceptance ./...` fails until the task deletes the stub. `TestStubsBelongToOpenTasks` (tagged) fails if any stub's FixedBy is in `internal/plan.Done`.
- Changed shapes of existing types are **not** stubbed: TypeSet on `Schema.Type`, `json.Number` bounds, `OrRef` aliases, `Explode *bool`, new fields on `Tag`/`ResponseHeader`, and `Extensions`. They stay task-owned red tests (rule 3).
- **Landed for real in P1.0:** `fizz.Option`, an unexported `config`, `New(opts ...Option)` and `NewFromEngine(e, opts ...Option)`. They are source-compatible. Options are applied and ignored until their tasks land.
- At P9.4 the tag and all stub files are removed (G-STUB).

## 6. New exported API

Each row is a stub until its task lands. Signatures are binding. A task that needs a different signature amends this table.

### 6.1 `fizz` (root)

| Symbol | FixedBy |
|---|---|
| `type Option func(*config)`, `New(opts ...Option)`, `NewFromEngine(e *gin.Engine, opts ...Option)` | **P1.0 (real)** |
| `WithTagNames(openapi.TagNames) Option` | P6.1 |
| `WithSchemaNamer(func(reflect.Type) string) Option` | P6.1 |
| `WithPointersAreNullable(bool) Option` | P6.1 |
| `WithAnnotationStyle(openapi.AnnotationStyle) Option` | P6.1 |
| `WithoutBindHook() Option` | P6.1 |
| `WithDefaultMediaType(mediaType string) Option` | P4.4 |
| `WithRequestBodyNamer(func(opID string, t reflect.Type) string) Option` | P4.6 |
| `WithOperationIDNamer(func(method, path, handlerName string) string) Option` | P4.7 |
| `WithHooks(openapi.Hooks) Option` (composes in order) | P6.4 |
| `(*Fizz).Err() error` | P4.8 |
| `(*Fizz).SpecHandler(format string) http.Handler` (`"json"`, `"yaml"`) | P5.1 |
| `(*Fizz).Document() (*openapi.OpenAPI, error)` (DocumentHook applied) | P6.4 |
| `type DocsUI struct{ /* unexported */ }` | P8.1 |
| `(*Fizz).Docs(path, specURL string, ui ...DocsUI)` | P8.1 |
| `SwaggerUI(opts ...docsui.Option) DocsUI`, `ScalarUI(opts ...docsui.Option) DocsUI` | P8.1 |
| `EmbeddedSwaggerUI(opts ...docsui.Option) DocsUI`, `EmbeddedScalarUI(opts ...docsui.Option) DocsUI` | P8.3 |
| `BindHook(next tonic.BindHook) tonic.BindHook`, `InstallBindHook()` | P7.2b |
| `type ContentOption`, `type EncodingOption` (see note*) | P6.6a |
| `RequestBody(desc string, required bool, contents ...ContentOption) OperationOption` | P6.6a |
| `Content(mediaType string, model any, encs ...EncodingOption) ContentOption` | P6.6a |
| `Encoding(property, contentType string) EncodingOption` | P6.6a |
| `Parameter(p *openapi.Parameter) OperationOption` | P6.6b |
| `Tags(tags ...string) OperationOption` | P6.6c |
| `ExternalDocs(url, desc string) OperationOption` | P6.6c |
| `Extension(key string, value any) OperationOption` | P6.6c |
| `CustomizeOperation(fn func(*openapi.Operation) error) OperationOption` | P6.6c |
| `type ResponseOption`, `ResponseWith(code, desc string, opts ...ResponseOption) OperationOption` | P6.6c |
| `WithModel(any)`, `WithHeaders(...*openapi.ResponseHeader)`, `WithExample(any)`, `WithExamples(map[string]any)`, `WithContent(mediaType string, model any)`, `WithResponseExtension(key string, value any)`, all `ResponseOption` | P6.6c |

\* `ContentOption`, `EncodingOption` and `ResponseOption` are opaque function types over unexported structs (`func(*contentSpec)`, `func(*encodingSpec)`, `func(*responseSpec)`). Their stubs declare the unexported structs empty.

### 6.2 `openapi`

| Symbol | FixedBy |
|---|---|
| `const Version = "3.2.1"` (stub: `""`) | P2.6 |
| `type TypeSet []string` | P2.2 |
| `type OrRef[T any] struct{ Ref *Reference; Value *T }` | P2.1a |
| `var ErrNilOrRef` | P2.1a |
| `type ExternalDocs struct{ Description, URL string }` | P2.2 |
| `var ErrInvalidExtensionKey`, `var ErrReservedExtension` | P2.3 |
| `SetExtension(key string, value any) error` on `OpenAPI`, `Info`, `Contact`, `License`, `Server`, `ServerVariable`, `Components`, `PathItem`, `Operation`, `Parameter`, `RequestBody`, `MediaType`, `Encoding`, `Response`, `Header`, `Example`, `Tag`, `Schema`, `SecurityScheme`, `OAuthFlows`, `OAuthFlow`, `ExternalDocs` | P2.3 |
| `MarshalJSON(doc *OpenAPI) ([]byte, error)` | P2.3 |
| `MarshalYAML(doc *OpenAPI) ([]byte, error)` | P2.5 |
| `ParseParamTag(tag string) (name string, required bool, defaultValue string, err error)` | P1.3 |
| `type BinderTagNames struct{ Path, Query, Header, Enum, Default, Validate, Explode string }`, `var BinderTags BinderTagNames` | P1.3 |
| `type Warning struct{ Method, Path, ID, Msg string }`, `Error()`, `IsWarning() bool` | P3.3 |
| `type AnnotationStyle int`, `const SiblingAnnotations, AllOfWrap` | P3.4 |
| `FieldError` gains `Field string`, `Err error` and `Unwrap()`. It already exists in openapi/errors.go, so this is a shape change: task-owned, not stubbed. | P4.2 |
| `type CollisionError struct{ Name string; Kept, Rejected reflect.Type }`, `Error()` | P4.5 |
| `type OperationError struct{ Method, Path, ID string; Err error }`, `Error()`, `Unwrap()` | P4.8 |
| `(*OpenAPI).Clone() (*OpenAPI, error)` | P5.1 |
| `type TagNames struct{ Description, Example, Format, Deprecated, Title, Pattern, ReadOnly, WriteOnly, Nullable, MultipleOf, AdditionalProperties, Inline, Style, AllowReserved, Cookie string }` (zero = default, `"-"` disables) | P6.1 |
| `SchemaProvider{ OpenAPISchema() *Schema }`, `SchemaCustomizer{ CustomizeSchema(*Schema) error }`, `Enumer{ Enum() []any }` | P6.3 |
| `SchemaContext{ Type reflect.Type; Name string }` | P6.4 |
| `FieldContext{ Parent reflect.Type; StructField reflect.StructField; Name string }` | P6.4 |
| `ParameterContext{ OperationID, Method, Path, In string; StructField reflect.StructField }` | P6.4 |
| `OperationContext{ OperationID, Method, Path string }` | P6.4 |
| `SchemaHook func(*SchemaContext, *Schema)`, `FieldHook func(*FieldContext, *Schema) (skip bool)`, `ParameterHook func(*ParameterContext, *Parameter) (skip bool)`, `OperationHook func(*OperationContext, *Operation) error`, `DocumentHook func(*OpenAPI) error` | P6.4 |
| `type Hooks struct{ Schema SchemaHook; Field FieldHook; Parameter ParameterHook; Operation OperationHook; Document DocumentHook }`, `(*Generator).SetHooks(Hooks)` | P6.4 |
| `(*Generator)`: `SetExtension(key string, value any) error`, `RegisterSchema(name string, s *Schema) error`, `RegisterType(t reflect.Type) error`, `RegisterParameter(name string, p *Parameter) error`, `RegisterResponse(name string, r *Response) error`, `RegisterExample(name string, e *Example) error`, `RegisterRequestBody(name string, rb *RequestBody) error`, `RegisterHeader(name string, h *Header) error`, `SetTag(t *Tag)`, `SetTagGroups(groups []*XTagGroup)`, `SetSelf(uri string)`, `SetJSONSchemaDialect(uri string)`, `SetExternalDocs(d *ExternalDocs)` | P6.5 |

Not stubbed (rule 3, task-owned): `Schema` field changes and helpers (`Types`, `Num`, `MakeNullable`, P2.2); `OrRef` aliases replacing the `*OrRef` structs (P2.1a); new fields on `OpenAPI`, `Info`, `License`, `Server`, `Tag` and `Components` (P2.1b); `Parameter`, `Header`, `MediaType`, `Encoding`, `Example` and `Response` fields, plus `Link` and `Callback` (P2.1c); `SecurityScheme` fields (P2.1d); the `Extensions` fields (P2.3); and `ResponseHeader` fields (P4.4).

### 6.3 `bind`

| Symbol | FixedBy |
|---|---|
| `type Tags struct{ Query, Cookie, Style, Default string }`, `type Plan struct{ /* unexported */ }` | P7.2a |
| `PlanFor(t reflect.Type, tags Tags) (*Plan, error)` (cached per type and tags) | P7.2a |
| `type QueryOption func(*queryConfig)`, `WithMaxKeys(n int) QueryOption` | P7.2a |
| `Query(r *http.Request, dst any, p *Plan, opts ...QueryOption) error` | P7.2a |
| `var ErrNestedKey`, `var ErrTooManyKeys` | P7.2a |
| `Cookies(r *http.Request, dst any, p *Plan) error` | P7.2c |

### 6.4 `docsui` and `docsui/embedded`

| Symbol | FixedBy |
|---|---|
| `type Option func(*config)` | P8.1 |
| `WithTitle(string)`, `WithUIConfig(map[string]any)`, `WithCSPNonce(string)`, `WithAssetURLs(js, css string, integrity ...string)`, all `Option` | P8.1 |
| `SwaggerUI(specURL string, opts ...Option) http.Handler`, `ScalarUI(specURL string, opts ...Option) http.Handler` | P8.1 |
| `const SwaggerUIJS, SwaggerUICSS, SwaggerUIJSIntegrity, SwaggerUICSSIntegrity, ScalarJS, ScalarJSIntegrity` | P8.1 |
| `embedded.SwaggerUI(specURL, prefix string, opts ...docsui.Option) http.Handler`, `embedded.ScalarUI(specURL, prefix string, opts ...docsui.Option) http.Handler` | P8.3 |

## 7. Consequences

- `go vet -tags acceptance ./...` is a P1.0 gate, and CI runs an `acceptance` job from now on.
- Tagged acceptance tests (batches C, D and the tagged parts of A, B, E–I) can be written against §6 immediately.
- The P1.0 deliverables are this ADR, the stubs, `doc.go` files, the real `Option`/variadic constructors, `TestStubsBelongToOpenTasks` and the CI job. `internal/apistub` from PLAN-v2 is not created (ACCEPTANCE §2 supersedes it).

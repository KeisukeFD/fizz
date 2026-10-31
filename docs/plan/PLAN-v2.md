# fizz Overhaul Plan v2: OAS 3.2.1, correctness, customization, deepObject/cookies, docs UI

Line refs are HEAD 87b41d4 (pre-split), tonic = `gadgeto@v0.9.0/tonic`. I re-checked them against the source for v2.

## 0. Decisions frozen for this plan

| # | Decision |
|---|---|
| D1 | Output only **OpenAPI 3.2.1**. Use one exported constant, `openapi.Version = "3.2.1"`, which replaces `version = "3.0.1"` at generator.go:18. A test checks it against the official 3.2 schema pattern `^3\.2\.\d+$`. There is no multi-version output. P0.6 is an early spike that checks rendering in the pinned Swagger UI and Scalar. |
| D2 | `go 1.27` floor. CI runs `1.27.x` and `stable` (blocking), plus `tip` (gotip, `continue-on-error: true`). The release notes state that Go ≤1.26 users are excluded. |
| D3 | Router strategy C. `openapi/`, `bind/` and `docsui/` import neither gin nor tonic, and an ADR rule plus a test enforce this. fizz owns its media-type config and param-tag parser. Gin→OpenAPI path translation lives in the router layer. gin + tonic stay the only router and binder in v1. Future-proofing rules: **(a)** deepObject and cookie decoding live in a standalone `bind` package on `*http.Request`, with plans cached per `reflect.Type`, and the tonic hook is a thin adapter; **(b)** the spec handler and the docs UI return `http.Handler` and gin wraps them with `gin.WrapH`; **(c)** no gin or tonic types appear in hook contexts or operation options, and the operation is injected through the request context, not `c.Set`. |
| D4 | Same module path. The overhaul ships as **v0.30.x**, then **v1.0.0** after stabilization. There is no v0.24.0. An optional v0.23.1 on `release-0.x` carries only the visited-set cycle fix. |
| D5 | Embedded UI assets live in subpackage `docsui/embedded` of the main module, not a nested module. |
| D6 | Pointers are nullable by default (option `PointersAreNullable`, default true). |
| D7 | Precedence has two stages (§1). Field tags win over type interfaces at the use site. |
| D8 | Annotations are `$ref` siblings by default. Option `AnnotationStyle: AllOfWrap` emits `allOf:[{$ref}]` + annotations instead. |
| D9 | Stay on gadgeto v0.9.0 (validator.v9). The old P1.3 "tonic upgrade assessment" is dropped. |
| D10 | Missing path params are synthesized as required `string` params, with a warning in `Errors()`. There is no Strict mode. |
| D11 | Plain (non-tonic) gin handlers stay undocumented (out of scope). |
| D12 | `encoding/json/v2` **is GA in Go 1.27**. `internal/buildcfg/exp.go:87` sets `JSONv2: true` in the baseline, and `api/go1.27.txt` lists 45 `encoding/json/v2` symbols. We use it to serialize the document: `Extensions map[string]any \`json:",embed"\`` (`embed` is the Go 1.27 name of the draft `inline` option; see ADR 0001 §4) replaces the planned `marshalWithExt` splice, with `json.Deterministic(true)` for sorted map keys. v1 `encoding/json` parity is tested in P2.3. |
| D13 | DELETE request bodies: tonic binds them (DefaultBindingHook skips only GET or empty bodies, tonic.go:95), but fizz forbids them (generator.go:274-276). **Decision: document DELETE bodies.** GET and HEAD stay excluded. |
| D14 | `json.Marshaler` policy. A type with both MarshalJSON and MarshalText becomes `string`. A type with only MarshalJSON keeps its kind-derived schema, plus a warning in `Errors()` that points to `SchemaProvider`. Map-key schemas follow encoding/json order: string kinds first, then TextMarshaler, then integer kinds. |
| D15 | Default + required. On **params**, tonic fills the default before validation (handler.go:208-216), so `required` is dropped from the spec and a warning is recorded. Path params are exempt: OAS requires `required: true` on them, so they keep it and no warning is recorded (P3.6). On **body** fields, tonic never applies defaults, so `required` is kept, `default` is dropped and a warning is recorded. |
| D16 | Duration defaults. For **params**, tonic cannot parse `"1s"`: tonic.go:396-411 sets the duration, then falls through to `strconv.ParseInt` at :410. Params therefore accept only nanosecond integers, and `"1s"` on a param is a generation error. Body fields accept `"1s"` and emit nanoseconds. |

## 1. Precedence (two stages, then document)
1. **Component stage**, per named type, written once into `components/schemas`: reflection (or `SchemaProvider`, which replaces it) → `SchemaCustomizer` / `Enumer` → `SchemaHook`.
2. **Use-site stage**, per field, param or operation, never mutating the component: struct tags → `FieldHook` / `ParameterHook` → operation options → `OperationHook`.
3. `DocumentHook` runs on a deep copy at serve or export time and is cached per generation.

## 2. Conventions (apply to every task)
- **Red first.** Each task names the test to write first. For bug cases this means un-skipping the P0 case marked `t.Skip("fixed by Pn.m")`, which is the task's red→green step.
- **Gates on every PR:** goldens, `internal/specvalidate` (3.2 meta-schema plus the stray-null lint), determinism, `go test -race`, and the import rule.
- **Goldens.**
  - Until P2.6, the only whole-doc golden is `testdata/legacy/kitchensink.json`, a capture of the current 3.0.1 output. It is a safety net for P1.1–P1.3 only, and P2.6 deletes it.
  - Per-case goldens are generated only after P2.6.
  - Any golden regeneration needs Opus review of the diff.
- **Agents.** Haiku explores (inventories, URL and version checks, golden-diff collection). Opus designs and reviews. Sonnet implements and drafts docs.
- **Sizes:** S ≈ 0.5 d, M ≈ 1.5 d, L ≈ 3 d of agent work plus human review.
- **Parallelism** is allowed only between tasks whose file sets (DAG in §13) are disjoint. Every task that touches `AddOperation` (generator.go) or `Handle` (router.go) is serialized on chain H.

---

## Phase 0: Test harness on master (not released)

**P0.1 Toolchain floor + modernize.** S · Sonnet (Haiku inventories `ioutil`/`interface{}` sites)
- Files: go.mod, go.sum, tools/go.mod (new), .github/workflows/{ci,release}.yml, ci/test.sh, all `*.go` (mechanical).
- Change:
  - Commit 1 is only `go mod edit -go=1.27 && go mod tidy && go fix ./...` (modernizers: `any`, `reflect.TypeFor`, `slices`, `strings.SplitSeq`, and so on). The go directive has to be in the same commit, because modernizers are gated on the file's Go version.
  - Commit 2: bump testify and other deps **except gin**. gin 1.12 renders YAML with goccy/go-yaml (gin@v1.12.0/render/yaml.go:10), which would change the YAML golden, so the gin bump is deferred to P2.5.
  - CI matrix: `1.27.x` and `stable` × ubuntu, `stable` × macOS and windows, plus a non-blocking `tip` job. actions v2→v5. release.yml Go 1.17→`stable`.
  - Dev tools (govulncheck, staticcheck) go in `tools/go.mod` through the `tool` directive and run with `go tool -modfile=tools/go.mod`, which keeps them out of consumers' module graphs.
  - Replace `io/ioutil`.
- Red first: none (toolchain change). Guard: the existing goldens testdata/spec.{json,yaml} stay byte-identical.
- Accept: `go test -race ./...` is green on 1.27 and stable, and `go vet` is clean.
- Deps: none.

**P0.2 Golden helper.** S · Sonnet
- Files: internal/golden/{golden.go,golden_test.go}, fizz_test.go:409/434, openapi/generator_test.go:216/237/553, openapi/validation_test.go:55.
- Change: `golden.Assert(t, path, got)` with an `-update` flag and canonical JSON indent. A diff reports the first differing JSON pointer. The ad-hoc golden reads move to the helper.
- Red first: `TestAssertReportsPointer`, which asserts that a mismatch message contains `/paths/~1x/get`.
- Accept: existing goldens unchanged.
- Deps: P0.1.

**P0.3 Meta-schema validator (test-only).** M · Haiku (exact URLs + sha256 of the 3.2 schema, schema-base, dialect and meta files) + Sonnet
- Files: internal/specvalidate/{validate.go,lint.go,schemas/3.2/*}, go.mod (santhosh-tekuri/jsonschema/v6).
- Change:
  - Offline loader. `Validate(doc []byte)` uses the 3.2 schema-base.
  - "No stray null" lint fails on `null` anywhere except inside `example`, `examples`, `default`, `const`, `enum`, `value` or `dataValue`.
  - No 3.0 schema is vendored: the legacy golden is byte-compared instead.
- Red first: `TestRejectsBadDoc` (missing `info`, `"schema": null`) and `TestVersionPattern` (`openapi.Version` must match the 3.2 schema's `^3\.2\.\d+`; skipped "fixed by P2.6").
- Accept: invalid docs fail with a readable pointer.
- Deps: P0.1. Can run in parallel with P0.2 (disjoint files).

**P0.4a Kitchen-sink harness + legacy golden.** M · Sonnet
- Files: internal/kitchensink/{case.go,assert.go,a/models,b/models (same package name),generic/}, internal/kitchensink/legacy_test.go, testdata/legacy/kitchensink.json.
- Change:
  - `Case{Name, Register func(*fizz.Fizz), Expect []Fragment, FixedBy string}`.
  - `Fragment{Pointer, JSON}` is a hand-written expected fragment of the **3.2 target output**, asserted by JSON pointer.
  - A case with `FixedBy` calls `t.Skip("fixed by "+FixedBy)` and prints the skip count.
  - Captures the whole current output as the legacy golden.
- Red first: `TestFragmentAssert` (a wrong fragment fails) and `TestLegacyGolden`.
- Accept: green, and skip counts are printed.
- Deps: P0.2, P0.3.

**P0.4b–f Bug cases by domain.** S–M each · Haiku inventory + Sonnet. Each task adds one file, so they can run in parallel with each other and with Phase 1.
- **b `cases_types.go`:**
  - integer kinds and Duration;
  - url.URL, big.Int/Float/Rat, net.IP, netip;
  - TextMarshaler only, MarshalJSON only, and both;
  - unsupported kinds (chan, func, complex, error);
  - `map[int]T` and map-key precedence;
  - pointer-receiver interfaces;
  - json.RawMessage, `[]byte`, `[N]T`;
  - cycles: struct, embedded, `type L []L`, `type M map[string]M`. Stack-overflow cases must stay skipped.
- **c `cases_fields.go`:**
  - json dominance, `-`, `-,`, `,string`, tagged embedded struct;
  - embedded cycle (generator.go:523, :1032);
  - the `json:"-"` leak (generator.go:623-647);
  - shared-component mutation: two fields with different descriptions on one component (generator.go:793-797);
  - nullable `*Struct`, `*[]T`, `*map`.
- **d `cases_params.go`:**
  - query tags `-`, `""`, `,required`, `,default=`, and `,omitempty` (panics via tonic.go:342-369 → fizz.go:220);
  - `*wildcard`, `:id` with no input, missing path param, conflicting locations, `explode:"false"`;
  - Duration default on a param (ns only);
  - DELETE body;
  - placeholders for cookie and deepObject.
- **e `cases_values.go`:**
  - default on a pointer;
  - Duration default and enum;
  - enum trim (generator.go:842);
  - invalid bool default (types.go:203-208 silently returns false), which should become an error;
  - default + required;
  - float32 default precision, which should emit the original literal as `json.Number`;
  - NaN example;
  - default-examples with a nil schema, and an empty examples map;
  - example→examples;
  - validate tags: `min=0`, decimals, float `gt`/`lt`, dive, OR, email break (generator.go:1188).
- **f `cases_ops.go`:**
  - body-name collision (generator.go:461), generic names, same-name packages, Typer collision;
  - generic and method-value op IDs;
  - default and unknown code descriptions;
  - response code clash (panic via fizz.go:220);
  - nil scopes, OAuth with no scopes;
  - Group tag description silently overwritten (generator.go:211-214 when a later Group passes `""`);
  - `OperationFromContext` returning the live mutable op (fizz.go:413-421);
  - Handle mutating the caller's slice (fizz.go:229-236);
  - YAML/JSON parity, `$ref` siblings, OrRef with both fields nil.
- Red first: these tasks *are* the red tests. Each case carries `FixedBy`.
- Accept: green with skips. Every M9 bug has a case.
- Deps: P0.4a.

**P0.5 Determinism + race.** S · Sonnet
- Files: internal/kitchensink/determinism_test.go, race_test.go (fizz root).
- Change:
  - Build the kitchen sink 20× with fresh generators under GOMAXPROCS 1 and 8; JSON and YAML must be byte-identical.
  - Race test: serve the spec while registering routes. Tonic handlers are created **before** any goroutine starts, because tonic's global `routes` map is written unlocked at handler.go:132.
  - Uses `t.Context()`. The race test is skipped "fixed by P5.1".
- Red first: the race test itself.
- Deps: P0.4a.

**P0.6 Renderer spike (decision gate for D1).** S · Haiku (verify on npm that `swagger-ui-dist@5.34.1` and `@scalar/api-reference@1.76.4` exist, and record jsDelivr SRI sha384) + Sonnet
- Files: docs/compat/spike/{spec-3.2.1.json,swagger.html,scalar.html}, docs/compat.md.
- Change: a hand-written 3.2.1 doc covering:
  - `openapi: 3.2.1`, `$ref` with a `description` sibling;
  - `type: [string, "null"]`, `anyOf` nullable `$ref`;
  - a deepObject param and a cookie param;
  - `contentEncoding`, `examples`.

  Load it in both pinned UIs in a headless browser and record what renders, warns or breaks.
- Red first: `TestSpikeDocValid` (specvalidate).
- Accept: docs/compat.md has a table per feature × UI and the verified pins. If a pinned UI rejects `3.2.x`, raise the issue (see §16, Q1) before P8.1.
- Deps: P0.3.

## Phase 1: Scaffolding (master; v0.23.0 has been branched to `release-0.x`)

**P1.0 ADR + API stub.** M · Opus (Sonnet types the stub)
- Files: docs/adr/0001-v1-design.md, internal/apistub/*.go (deleted after P6.6c), internal/archtest/imports_test.go.
- Change: freeze D1–D16, the §1 precedence, and the import rule. Write every new exported signature as a compiling stub:
  - `OrRef[T]` and `Reference{Ref, Summary, Description}`;
  - `Schema`, `TypeSet`;
  - `TagNames` (B1 list), `BinderTags`;
  - the hook context types (no gin or tonic types);
  - `SchemaProvider`, `SchemaCustomizer`, `Enumer`;
  - `fizz.Option` and its options;
  - `bind.Plan`, `bind.Query`, `bind.Cookies`;
  - `docsui` constructors returning `http.Handler`.
- Red first: `internal/archtest.TestRouterAgnostic`, which runs `go list -deps -test ./openapi/... ./bind/... ./docsui/...` and fails if `github.com/gin-gonic/gin` or `github.com/loopfz/gadgeto` appears. It is red today because openapi/generator.go imports tonic. It stays skipped "fixed by P1.3".
- Accept: `go build ./internal/apistub` and `go vet` pass, and Opus has signed off the ADR review.
- Deps: P0.4a, P0.6.

**P1.1 Mechanical splits (no logic change).** M · Sonnet
- Files: openapi/generator.go, openapi/spec.go, fizz.go, and the new files below.
- `openapi/generator.go` is split into:

  | New file | Contents (old line ranges) |
  |---|---|
  | generator.go | :1-345 (config, setters, AddTag, AddOperation, rewritePath, setOperationBymethod) **+ `error()` :1232-1234**, next to `Errors()` :127 |
  | responses.go | :347-437 |
  | params.go | :439-745 **+ `resolveParameter` :1117-1123** |
  | fields.go | :747-864, :1006-1089, :1236-1256 |
  | schema.go | :866-1004, :1091-1115 |
  | naming.go | :1125-1167 |
  | validate_tag.go | :1169-1230 |
  | examples.go | :1258-1310 |

- `openapi/spec.go` is split into:

  | New file | Contents (old line ranges) |
  |---|---|
  | spec_root.go | :7-94 OpenAPI…PathItem, :362-367 Tag |
  | spec_paths.go | :95-135, :190-361 |
  | spec_schema.go | :136-189 |
  | spec_security.go | :369-423 |
  | spec_ext.go | :425-445 |

- `fizz.go` is split into:

  | New file | Contents (old line ranges) |
  |---|---|
  | fizz.go | :1-107 (Fizz, New, NewFromEngine, accessors) |
  | router.go | :109-246, :410-450 (RouterGroup, verbs, Handle, OperationFromContext, helpers) |
  | options.go | :270-409 |
  | spechandler.go | :248-268 |

- Red first: none (pure move). Guard: the legacy golden and testdata/spec.* stay byte-identical, and the `go test -cover` delta is 0.
- Deps: P1.0.

**P1.2 Drop gofrs/uuid runtime dep.** S · Sonnet
- Files: openapi/generator.go (:238-240), openapi/types.go (:11, :27, :185-188), go.mod.
- Change:
  - The placeholder op ID becomes `""`. It is never observable: fizz always passes a non-nil `OperationInfo` (fizz.go:175), and real default IDs come in **P4.7**.
  - UUID detection matches by `PkgPath()+Name()` for the Go 1.27 standard-library `uuid.UUID` (#62026), `github.com/gofrs/uuid.UUID` (and `/v5`) and `github.com/google/uuid.UUID`.
- Red first: `TestNoUUIDDep` (`go list -deps ./openapi` has no gofrs), plus the google-uuid case from P0.4b.
- Deps: P1.1.

**P1.3 Router-agnostic `openapi/`.** M · Sonnet
- Files:
  - openapi/generator.go: config `DefaultMediaType`, `BinderTags`; remove `ginPathParamRe` :31 and `rewritePath` :320-322;
  - openapi/paramtag.go (new): `ParseParamTag`, same semantics and error as tonic's `parseTagKey` (tonic.go:342-369);
  - openapi/params.go: :668 `ParseTagKey`, :697 `ExplodeTag`;
  - openapi/responses.go, openapi/fields.go: `tonic.MediaType()` at :292, :302, :455, :604, :623, :1018;
  - openapi/generator_test.go: drop the tonic import;
  - path.go (new, fizz root): gin `:x` and `*x` → `{x}`;
  - router.go, fizz.go: NewFromEngine fills config from tonic constants and `tonic.MediaType()`.
- Change: `AddOperation` receives an OpenAPI-syntax path, and the router layer translates it. `BinderTags` (path, query, header, enum, default, validate, explode) are **fixed** to tonic.go:24-33 and are not user-configurable.
- Red first: un-skip `TestRouterAgnostic`. Add `TestBinderTagsMatchTonic` (fizz root) and `TestGinPathTranslation`.
- Accept: legacy golden byte-identical. Still panics where it did before (behavior is preserved until P4.2 and P4.8).
- Deps: P1.2.

## Phase 2: OAS 3.2.1 spec model (BREAKING). Strictly serial: P2.2 → P2.1a → P2.1b → P2.1c → P2.1d → P2.3 → P2.5 → P2.6

**P2.2 Schema model.** M · Sonnet
- Files: openapi/spec_schema.go, spec_schema_test.go.
- Change:
  - `Schema` absorbs `SchemaOrRef`: `Ref string \`json:"$ref,omitzero"\`` with siblings allowed. `type SchemaOrRef = Schema` remains as a deprecated alias.
  - `Type TypeSet` marshals as a string when it has one element.
  - `AllOf`, `OneOf`, `AnyOf` become `[]*Schema` (fixes the single pointers at spec.go:156-158). Also adds `Not`, `Const`, `Examples []any` (drops `example` and `nullable`).
  - Numeric bounds become `json.Number` with `omitzero`. Lengths and counts become `*uint64`.
  - Adds `ReadOnly`/`WriteOnly` (bool omitzero), `ContentMediaType`, `ContentEncoding`, `PropertyNames`, `PrefixItems`, `Discriminator` (with `defaultMapping`), `XML` (with `nodeType`), `ExternalDocs`.
  - Boolean schemas use `Bool *bool`.
  - Helpers: `Types(...)`, `Num(...)`, `(*Schema).MakeNullable()`.
- Red first: `TestSchemaMarshal` table: `{"$ref":"#/c","description":"d"}`; single type → string; `true` schema; `json.Number("0.1")` is exact; `UnmarshalJSON` round-trip.
- Deps: P1.3.

**P2.1a `OrRef[T]` + Reference.** S · Sonnet
- Files: openapi/spec_ref.go.
- Change:
  - `type OrRef[T any] struct{ Ref *Reference; Value *T }` with Marshal and UnmarshalJSON (the presence of `$ref` decides).
  - Both nil → `Validate()` error, checked at build (P4.8) and never emitted.
  - `Reference` gains `Summary` and `Description`.
  - Generic aliases cover the 6 non-schema types: `type ParameterOrRef = OrRef[Parameter]`, and the same for Response, Header, MediaType, Example and SecurityScheme. The 7th type, `SchemaOrRef`, is covered by P2.2.
  - Field access changes from `.Parameter`/`.Reference` to `.Value`/`.Ref` (MIGRATION).
- Red first: `TestOrRefMarshal` (ref, value, both-nil error).
- Deps: P2.2.

**P2.1b Root, info, tags, components.** S · Sonnet
- Files: openapi/spec_root.go.
- Change:
  - Root: `JSONSchemaDialect`, `Self` (`$self`), `Webhooks`; `Paths` omitzero.
  - `Info.Summary`, `License.Identifier`, `Server.Name`.
  - `Tag`: `Summary`, `Parent`, `Kind`, `ExternalDocs`. `ExternalDocs` is a new type.
  - Components gain `RequestBodies`, `Links`, `Callbacks`, `PathItems`, `MediaTypes`. `Webhooks`, `Links`, `Callbacks`, `PathItems` and `MediaTypes` are **model fields only** (no setters in v1).
- Red first: `TestRootMarshal` fragments.
- Deps: P2.1a.

**P2.1c Paths and operations.** M · Sonnet
- Files: openapi/spec_paths.go.
- Change:
  - Delete the `operationNilOmitted` hack (spec.go:205-252) and use `omitzero` instead.
  - `PathItem.Query` and `AdditionalOperations` (model only).
  - `Parameter`: `Explode *bool` (tri-state, kept as a pointer), `AllowReserved`, `Example`, `Examples`, `Content`. Locations `querystring` and `cookie`. `Deprecated`, `Required` and `AllowEmptyValue` become `bool` with omitzero.
  - `Header`: style, explode, example, examples, content.
  - `MediaType.Schema` gets omitzero (fixes the real cause of `"schema": null`, spec.go:322). Adds `ItemSchema`, `PrefixEncoding`, `ItemEncoding`.
  - `Encoding`: nested `Encoding`, `PrefixEncoding`, `ItemEncoding`.
  - `Example`: `DataValue`, `SerializedValue`.
  - `Response`: `Summary`; `Description` becomes optional.
  - `Link` and `Callback` types (model only).
- Red first: `TestOperationOmitZero`, `TestMediaTypeNoNullSchema`.
- Deps: P2.1b.

**P2.1d Security.** S · Sonnet
- Files: openapi/spec_security.go.
- Change: `SecurityScheme.Deprecated`, `OAuth2MetadataURL`, `OAuthFlows.DeviceAuthorization`.
- Red first: `TestSecuritySchemeMarshal`.
- Deps: P2.1c.

**P2.3 Extensions + fixed shapes (merges old P2.4).** M · Sonnet
- Files: all openapi/spec_*.go (adds the field), openapi/spec_ext.go, openapi/encode.go (new, JSON half).
- Change:
  - `Extensions map[string]any \`json:",inline"\`` on every object, marshaled with `encoding/json/v2` and `Deterministic(true)`.
  - `SetExtension` validates that keys start with `x-` and rejects `x-oai-` and `x-oas-`. On a duplicate with typed `XLogo`, `XTagGroups`, `XCodeSamples` or `XInternal`, the typed field wins and an error is recorded.
  - Fixed shapes:
    - `SecurityRequirement` nil → `[]` (spec.go:421);
    - `OAuthFlow.Scopes` always `{}`;
    - delete the wrong `OAuthFlow.MarshalYAML` (spec.go:410-418).
  - `openapi.MarshalJSON(doc)`.
- Red first:
  - `TestExtensionsInlineSorted`;
  - `TestV1EncodingJSONParity`: v1 `json.Marshal(doc)` must equal the v2 output. If v1 ignores `inline`, add a root `MarshalJSON` that delegates to v2 and document that nested objects should be marshaled via `openapi.MarshalJSON`;
  - `TestRejectReservedExtension`.
- Accept: the kitchen sink round-trips byte-identical.
- Deps: P2.1d.

**P2.5 YAML from JSON + gin bump.** M · Sonnet
- Files: openapi/encode.go (YAML half), all spec_*.go (remove `yaml:` tags and MarshalYAML at spec.go:120, :142, :223, :267, :289, :313, :336, :375), spechandler.go, go.mod (gin → 1.12.x, add go.yaml.in/yaml/v3, drop gopkg.in/yaml.v2).
- Change:
  - `openapi.MarshalYAML(doc)`: JSON → decode into `yaml.Node` → **keep tags, clear `Style` recursively** → encode with indent 2.
  - The spec handler serves these bytes, not `c.YAML`. That is why the gin bump happens here.
  - Examples are JSON-checked at registration: NaN or chan → `Errors()`, not a panic at serve time.
- Red first:
  - `TestYAMLParity`: decode(YAML) deep-equals decode(JSON);
  - `TestYAMLAmbiguousStrings`: `"123"`, `"true"`, `"null"`, `"~"`, `"1e3"`, `""`, `"0x1F"` stay strings;
  - the NaN case from P0.4e (error recorded, server returns 200).
- Deps: P2.3.

**P2.6 Minimal generator port (compile + validate only).** M · Sonnet + Opus review
- Files: openapi/{generator,params,fields,schema,responses,examples,validate_tag,validation,types}.go, openapi/sort.go (deleted), testdata/**.
- Change:
  - Mechanical type renames to the new model.
  - `Version` "3.2.1".
  - `TypeAny` → `{}` (generator.go:901-908).
  - `[N]T` minItems and maxItems as pointers (:949-952).
  - `[]byte` → `contentEncoding: base64`.
  - Stop setting `AllowEmptyValue` (:689-691).
  - Explode becomes `*bool` (:694-701).
  - validation.go emits `json.Number`.
  - sort.go is replaced by `slices.SortStableFunc` with `cmp.Compare`.
  - **Not ported or improved:** `buildSchemaRecursive`, `flattenStructSchema`, `buildParamsRecursive` (P3.1 and P3.2 delete them; renames only). Nullability is temporarily not emitted: those cases are skipped "fixed by P3.5".
  - Delete the legacy golden. Generate per-case goldens and the 3.2 whole kitchen-sink golden.
- Red first: un-skip `TestVersionPattern`, and run specvalidate over every unskipped case.
- Accept: the 3.2 meta-schema passes for spec.json, every unskipped case and the kitchen sink. Opus has signed off the golden diff.
- Deps: P2.5, P0.4b–f.

## Phase 3: Types and structs (BREAKING output). Order: (P3.1a → P3.1b) ∥ P3.2 → P3.3 → P3.5 → P3.4 → P3.6

**P3.1a `typeFields` + body/schema use.** L · Opus design + Sonnet
- Files: openapi/typefields.go (new), openapi/fields.go. The entry signature `flattenStructSchema(t, parent, schema)` is kept so that schema.go is untouched.
- Change:
  - A port of encoding/json's field algorithm: BFS by embedding depth with a **visited set per type** (fixes the embedded-cycle crash at generator.go:1032).
  - Dominance: tagged beats untagged at the same depth, and ambiguous fields are dropped.
  - `-` skips; `-,` names the field `-`; `,string` is recorded; a tagged embedded struct becomes a named property.
  - Tag options are parsed with `strings.SplitSeq`, and fields are exposed as `iter.Seq[field]`.
  - Replaces `fieldNameFromTag` (:1238-1256).
- Red first:
  - un-skip the P0.4c json cases;
  - **differential test (kept from v1):** marshal sample values with encoding/json and compare the key sets with the schema properties.
- Deps: P2.6.

**P3.1b Param walker.** M · Sonnet
- Files: openapi/params.go (replaces `buildParamsRecursive` :499-544).
- Change: uses `typeFields` with tonic location semantics (tonic recurses into every anonymous field, handler.go:176-191). Fixes the `json:"-"` → `""` leak (:623-647) and the cycle at :523.
- Red first: un-skip the P0.4c `json:"-"` param case and the embedded-cycle param case.
- Deps: P3.1a.

**P3.2 Unified type→schema recursion + cycle safety.** L · Opus design + Sonnet
- Files: openapi/schema.go only.
- Change:
  - `schemaFor(t, site)`: strip pointers while recording nullability → resolve (overrides → interfaces → known types) → kind switch.
  - `buildSchemaRecursive` is removed. Elements, map values and `,string` all take the same path, which fixes element datatypes and `OverrideDataType` on element structs.
  - Cycles: an in-progress stack. A named non-struct type that re-enters (`type L []L`, `type M map[string]M`) emits a `$ref` and is promoted to a component. Deterministic.
- Red first: un-skip the P0.4b cycle cases.
- Deps: P2.6. Can run in parallel with P3.1a/b (disjoint files).

**P3.3 Type mapping table + Marshaler policy.** M · Sonnet
- Files: openapi/types.go, openapi/schema.go.
- Change:
  - `knownTypes`:
    - int, int64 → integer/int64; int32 → int32; int8, int16 → int8, int16;
    - uint* → integer/uint* with `minimum: 0`;
    - `time.Duration` → integer/int64 (ns); `time.Time` → date-time;
    - json.RawMessage → `{}` (checked before `[]byte`);
    - uuid (stdlib `uuid`, gofrs, google) → string/uuid;
    - net.IP, netip.Addr, netip.Prefix → string;
    - url.URL → object component;
    - big.Int → integer; big.Float, big.Rat → string.
  - D14 Marshaler policy and map-key order. Unsupported kinds → error, property omitted.
  - Pointer-receiver checks look at both `t` and `reflect.PointerTo(t)` (old :878, types.go:130). Uses `reflect.TypeFor`.
- Red first: un-skip P0.4b.
- Deps: P3.1b, P3.2.

**P3.5 Nullability (absorbs the nullable work cut from P2.6).** S · Sonnet
- Files: openapi/schema.go, openapi/fields.go, openapi/generator.go (option).
- Change:
  - A `*Struct` / `$ref` becomes `anyOf:[{$ref},{type:"null"}]`, with annotations outside the `anyOf`. Other types get a type set `[T,"null"]`.
  - The `Nullable` interface is checked on both value and pointer receivers.
  - Generator option `PointersAreNullable` (default true).
- Red first: un-skip the nullable cases.
- Deps: P3.3.

**P3.4 Use-site annotations, never mutating components.** M · Sonnet
- Files: openapi/fields.go, openapi/schema.go (remove `resolveSchema` :1091-1115 from mutation paths).
- Change:
  - For a `$ref`, annotations become siblings, or `allOf` per D8 (`AnnotationStyle`).
  - An inline schema is copied, then annotated.
  - Array enum and validation go on a copied items wrapper (fixes :793-797).
- Red first: un-skip "two fields, different descriptions, one component". The component must be unchanged and independent of registration order.
- Deps: P3.5.

**P3.6 default/enum/example conversion.** M · Sonnet
- Files: openapi/types.go (`stringToType` :193-222), openapi/examples.go, openapi/fields.go.
- Change:
  - Dereference pointers.
  - D16 Duration rules.
  - float32 defaults emit the original literal as `json.Number`.
  - An invalid bool is an error (today :203-208 silently returns false).
  - D15 default+required.
  - Enum values are trimmed (:842). Values follow the `,string` wire form. `example` becomes `examples:[v]`.
  - Switch document encoding from the v1 option set to the v2 defaults: `omitzero` on struct-valued and bool fields, and no `time.Duration` left in `any`. G21 parity must stay green (ADR 0001 §4, "JSON v2 bridge").
- Red first: un-skip the P0.4e default, enum and example cases.
- Deps: P3.4.

## Phase 4: Params, validation, responses, naming, errors (BREAKING output). Starts after P3.6.

Lanes (file-disjoint):
- **Chain H** (serial; AddOperation, Handle or params.go): P4.2 → P4.2c → P4.6 → P4.9 → P4.8 → P4.7 → P4.3.
- **Lane V:** P4.1a → P4.1b → P4.1c.
- **Lane N:** P4.5.
- **Lane R:** P4.4.
- **P6.1** is scheduled at the head of Phase 4 because P4.2c, P4.4 and P7 need its options.

**P4.1a validate-tag grammar parser.** M · Opus grammar + Sonnet
- Files: openapi/internal/vtag/{parse.go,parse_test.go}.
- Change: go-playground v9 grammar: commas, `|`, `dive`, `keys`/`endkeys`, `omitempty`, and `0x2C` escapes.
- Red first: a parse table of about 20 tags, including `oneof=a b|eq=c`, `dive,keys,min=1,endkeys,max=3`, and the `0x2C` escape.
- Deps: P3.6.

**P4.1b Rule→keyword mapping.** M · Sonnet
- Files: openapi/validate_tag.go, openapi/validation.go.
- Change:
  - Container vs items/additionalProperties for pre- and post-`dive` rules; `keys` → propertyNames.
  - `min`, `gte`, `max`, `lte` are inclusive `json.Number` (fixes `min=0` and decimals). `gt` and `lt` are exclusive (fixes inverted float bounds).
  - String/slice/map length and count rules; `len` sets both bounds.
  - `eq` → `const`, `ne` → `not.const`, `oneof` → `enum`.
  - Formats: email (fixes the early `break` at :1188), url/uri, uuid*, ipv4, ipv6, hostname. **No `datetime`** (not in validator v9).
  - alpha, alphanum, numeric, hexadecimal → pattern; `unique` → uniqueItems.
  - OR groups → `anyOf` only if every alternative maps, otherwise dropped.
  - Duration bounds are parsed into ns. No length rules on `[]byte`.
- Red first: un-skip the P0.4e validate cases, plus about 30 table tests.
- Deps: P4.1a.

**P4.1c `isStructFieldRequired` on the parser.** S · Sonnet
- Files: openapi/fields.go (old :1073-1089).
- Red first: `validate:"omitempty,required"`, `required_if=…` (not required), `dive,required` (not required).
- Deps: P4.1b.

**P4.2 Query/path/header tag options.** M · Sonnet
- Files: openapi/params.go, openapi/paramtag.go.
- Change:
  - `-` or an empty name → not a param.
  - `,required` → required; `,default=` → default.
  - Unknown options (`omitempty`) → `FieldError` and the field is skipped, with no panic. The parser no longer errors in a way that makes fizz.go:220 panic.
  - `explode:"false"` is honored for query.
  - A param with an unsupported schema is omitted and an error recorded.
- Red first: un-skip P0.4d `-`, `""`, `,required`, `,default=` and `,omitempty`.
- Deps: P3.6.

**P4.2c Cookie params (generation).** S · Sonnet
- Files: openapi/params.go.
- Change: the `TagNames.Cookie` tag (default `cookie`) → `in: cookie`, scalar types only (anything else is an error). `explode:"false"` is honored. Runtime binding comes in P7.2c.
- Red first: un-skip the cookie placeholder.
- Deps: P4.2, **P6.1**.

**P4.3 Paths.** S · Sonnet
- Files: path.go (fizz root), openapi/params.go (old :476-487), openapi/generator.go.
- Change:
  - `/*name` → `{name}`, and the description notes that the value may contain `/`.
  - A path param declared in the path but missing from the input (including handlers with no input) is synthesized as a required string param, with a warning (D10). The error at :484-486 is removed.
- Red first: un-skip `*wildcard`, `:id` with no input, and the missing-param case.
- Deps: P4.7.

**P4.4 Responses.** M · Sonnet
- Files: openapi/responses.go, openapi/operation.go (`ResponseHeader` :23-27), options.go, config.go.
- Change:
  - An empty examples map or no schema → no media type.
  - `DefaultResponseExamples` merges into the default response.
  - Descriptions: `default` → "Default response"; `2XX`-style codes → "<class> response"; unknown codes → "Status <n>".
  - Response headers get required, deprecated, example and examples.
  - `fizz.WithDefaultMediaType(mt)` (config.go). This is needed because `tonic.MediaType()` returns a constant and ignores `SetRenderHook` (tonic.go:140-142 vs :183).
- Red first: un-skip the P0.4f code-description and empty-examples cases.
- Deps: P6.1.

**P4.5 Component naming + registry.** M · Sonnet
- Files: openapi/naming.go, openapi/registry.go (new), openapi/schema.go (component registration call site).
- Change:
  - `SchemaNamer func(reflect.Type) string`. The default is TitleASCII(pkg)+name.
  - `strings.Title` (generator.go:461, :1164, :1166) is replaced by an ASCII `upperFirst`. `x/text` **cannot** be dropped: markdown/builder.go:9 uses `unicode/norm`.
  - Generic type args are parsed from `t.Name()`, keeping the last path segment, joined with `_`. Names are sanitized to `^[A-Za-z0-9._-]+$`.
  - A name→`reflect.Type` registry. On a collision the first type is kept and an error recorded (`CollisionError`, the only policy in v1). This covers Typer, override and default collisions.
- Red first: un-skip same-name packages, generic names, and the Typer collision.
- Deps: P3.6.

**P4.6 Request body naming + DELETE body.** S · Sonnet
- Files: openapi/params.go (old :450-467), openapi/generator.go (`allowBody` :274-276).
- Change:
  - `RequestBodyNamer`, default `<OpID>Input`, goes through the registry and never overwrites.
  - If the body is exactly one named struct and there are no params, the request body references that struct directly.
  - `requestBody.required` is true if any body field is required.
  - D13: bodies are allowed on DELETE.
- Red first: un-skip the body-collision and DELETE-body cases.
- Deps: P4.2c, P4.5.

**P4.9 Handle must not mutate the caller's slice.** S · Sonnet
- Files: router.go (old fizz.go:229-236).
- Change: copy the handler slice before wrapping.
- Red first: un-skip the shared-slice case.
- Deps: P4.6 (chain H).

**P4.8 Errors, not panics; atomic AddOperation.** M · Sonnet
- Files: openapi/generator.go (:237-315), openapi/responses.go (code clash), router.go (fizz.go:194-196, :219-224), spechandler.go (fizz.go:265), fizz.go (`Err()`).
- Change:
  - A nil info becomes a zero-value info (fixes the nil dereference at :292).
  - Validate before mutating: the ID is reserved and the PathItem created only on success (:245-256).
  - These go to `Errors()`, and the gin route is still registered:
    - bad codes;
    - **response code clash** (today a panic via :220);
    - conflicting locations, duplicate IDs;
    - OrRef with both fields nil;
    - multiple tonic handlers.
  - `Errors()` returns `*openapi.Warning` and error values. `Fizz.Err() error` is `errors.Join` of the non-warnings. No `Strict()`.
- Red first: un-skip the code-clash and duplicate-ID cases, plus `TestFailedAddOperationLeavesNoTrace`.
- Deps: P4.9, P4.4.

**P4.7 Operation IDs.** S · Sonnet
- Files: router.go (fizz.go:203-205), openapi/generator.go.
- Change:
  - The default ID comes from `HandlerName` with `-fm` and generic `[...]` stripped. Closures use the enclosing function.
  - On a default-ID collision, prefix the package, then fall back to method+path.
  - An explicit `fizz.ID` collision is an error. `OperationIDNamer` option.
- Red first: un-skip the generic and method-value op-ID cases.
- Deps: P4.8.

## Phase 5: Concurrency and lifecycle

**P5.1 Locking, cache, deep copy, context injection.** M · Opus review + Sonnet
- Files: openapi/generator.go (mutex on all mutators), openapi/clone.go (new), spechandler.go, router.go (Handle injection, `OperationFromContext`).
- Change:
  - Only mutators lock (a `sync.Mutex`) and bump `gen atomic.Uint64`.
  - Cache: `atomic.Pointer[cacheEntry{gen, json, yaml func() ([]byte, error)}]`, where the funcs come from `sync.OnceValues`.
  - A reader loads the entry. If its `gen` is stale, it builds a new entry and CompareAndSwaps it in. Inside the once-func, cloning happens under the lock and marshaling outside it.
  - `Clone()` marshals with v2, then unmarshals with `WithUnmarshalers` mapping `*any` numbers to `json.Number` (the equivalent of UseNumber). `API()` returns a clone.
  - `f.SpecHandler(format) http.Handler` (rule b). `f.OpenAPI(info, ct) gin.HandlerFunc` keeps its signature and is `gin.WrapH(f.SpecHandler(ct))`.
  - Rule c: the Handle wrapper does `c.Request = c.Request.WithContext(withOperation(ctx, snap))` instead of `c.Set` (fizz.go:232).
  - `OperationFromContext(ctx)` unwraps `*gin.Context` → `Request.Context()`, because gin's `Value` reaches the request context only when `ContextWithFallback` is on. It returns a **deep clone of the per-generation frozen snapshot**, so mutating it never affects the spec.
- Red first:
  - un-skip the P0.5 race test (`-race`);
  - un-skip the `OperationFromContext` mutation case;
  - `TestCloneRoundTrip`, a property test: `Marshal(Clone(d)) == Marshal(d)` over the kitchen sink plus fuzzed extension values. Uses `testing/synctest` for cache invalidation.
- Deps: P4.3.

## Phase 6: Customization API

**P6.1 Fizz options + `TagNames` (B1).** M · Sonnet. Scheduled at the start of Phase 4.
- Files: config.go (new, fizz root), fizz.go (`New(opts ...Option)`, `NewFromEngine(e, opts...)`, both source-compatible), openapi/tags.go (new), openapi/generator.go (`SpecGenConfig` :83-93 and the consts at :20-22 become a deprecated shim).
- Change:
  - `openapi.TagNames{Description, Example, Format, Deprecated, Title, Pattern, ReadOnly, WriteOnly, Nullable, MultipleOf, AdditionalProperties, Inline, Style, AllowReserved, Cookie}`. These are the only configurable tags.
  - validate, path, query, header, enum, default and explode stay in `BinderTags`, fixed to tonic.go:24-33.
  - Options: `WithTagNames`, `WithSchemaNamer`, `WithPointersAreNullable`, `WithAnnotationStyle`, `WithoutBindHook` (consumed by P7.2b).
- Red first: `TestTagNamesOverride` (a custom `description` tag name is honored) and `TestBinderTagsNotConfigurable` (compile-time: the field does not exist).
- Deps: P3.6.

**P6.2 New struct tags.** M · Sonnet
- Files: openapi/fields.go, openapi/params.go.
- Change: `title`, `pattern`, `multipleOf`, `readOnly`, `writeOnly`, `nullable` (overrides the pointer default), `additionalProperties:"false"`, `inline`; for params, `style` and `allowReserved`. All are applied as use-site annotations (P3.4).
- Red first: one case per tag in cases_fields.go.
- Deps: P6.1, P3.4, P4.1c (fields.go), P4.3 (params.go). Can run in parallel with P5.1 (disjoint files).

**P6.3 Type interfaces.** M · Sonnet
- Files: openapi/interfaces.go (new), openapi/schema.go, openapi/types.go.
- Change:
  - `SchemaProvider{OpenAPISchema() *Schema}`, `SchemaCustomizer{CustomizeSchema(*Schema) error}`, `Enumer{Enum() []any}`, each with value and pointer receivers.
  - `Typer`, `DataType`, `Nullable` and `Exampler` are kept.
  - These run in the component stage (§1).
  - `OneOfer` is post-v1.
- Red first: `TestEnumerVsFieldTag` (the field tag wins at the use site, and the component keeps the Enumer values).
- Deps: P4.5 (schema.go).

**P6.5 Manual components + document setters.** M · Sonnet
- Files: openapi/components.go (new), openapi/generator.go (setters, `AddTag` :204-230), router.go (`Group` fizz.go:109-120).
- Change:
  - `RegisterSchema`, `RegisterType`, and `RegisterParameter`/`Response`/`Example`/`RequestBody`/`Header`, all through the registry.
  - `SetTagGroups`, `SetTag(*Tag)`, `SetSelf`, `SetJSONSchemaDialect`, `SetExternalDocs`, `SetExtension`.
  - **Group tag description bug:** an empty description no longer overwrites. A conflicting non-empty description becomes a warning.
- Red first: un-skip the P0.4f group-tag case.
- Deps: P4.5, P5.1. Can run in parallel with P6.3 (disjoint files).

**P6.4 Generator hooks.** L · Opus API review + Sonnet
- Files: openapi/hooks.go (new), openapi/{schema,fields,params,generator}.go, spechandler.go.
- Change:
  - `SchemaHook(*SchemaContext, *Schema)`, `FieldHook(*FieldContext, *Schema) (skip bool)`, `ParameterHook`, `OperationHook`, `DocumentHook(*OpenAPI) error`.
  - Contexts carry `reflect.Type`, `reflect.StructField`, component name, op ID, method and OpenAPI path, but **no gin or tonic types** (rule c).
  - Ordering follows §1. `DocumentHook` runs on the clone and is cached per generation.
- Red first: `TestPrecedence`, a table that asserts the winner at each §1 boundary.
- Deps: P6.2, P6.3, P6.5.

**P6.6a RequestBody / Content options.** M · Sonnet
- Files: options.go, openapi/operation.go, openapi/params.go.
- Change: `RequestBody(desc, required, ...Content(mediaType, model, ...EncodingOpt))`.
- Red first: `TestRequestBodyOption` (overrides the inferred required flag).
- Deps: P6.4.

**P6.6b Parameter merge.** M · Sonnet
- Files: options.go, openapi/operation.go, openapi/params.go.
- Change: `Parameter(*openapi.Parameter)` merges by (name, in), with non-zero fields winning. Otherwise it appends.
- Red first: `TestParameterMerge`.
- Deps: P6.6a.

**P6.6c Small options.** S · Sonnet
- Files: options.go, openapi/operation.go, openapi/generator.go.
- Change: `Tags(...)`, `ExternalDocs(url, desc)`, `Extension(k, v)`, `CustomizeOperation(func(*openapi.Operation) error)`, `ResponseWith(code, desc, ...ResponseOption)`. The positional `Response` is kept. The callback and link options are post-v1.
- Red first: one case per option.
- Deps: P6.6b.

## Phase 7: deepObject + cookies (generation + runtime binding)

**P7.1 deepObject generation.** M · Sonnet
- Files: openapi/deepobject.go (new), openapi/params.go.
- Change:
  - A query field with `style:"deepObject"` of type `map[string]S`, `struct`, or a pointer to either → `style: deepObject`, `explode: true`, and an object schema (additionalProperties for maps, properties from json names for structs).
  - Values must be scalars or slices of scalars. Anything else is a generation error.
  - **Generation errors:** the `,required` and `,default=` options on the query tag, and the `default` tag, because tonic would bind them at handler.go:208-216. Required-ness comes **only** from `validate:"required"`.
- Red first: un-skip the deepObject placeholders and the error cases.
- Deps: P6.6c (params.go), P6.2 (style tag).

**P7.2a `bind` package: deepObject decoding.** L · Opus design + Sonnet
- Files: bind/{plan.go,query.go,bind_test.go} (new; covered by the import rule).
- Change:
  - `bind.PlanFor(t reflect.Type, tags) (*Plan, error)` is cached in a `sync.Map` keyed by `reflect.Type`.
  - `bind.Query(r *http.Request, dst any, p *Plan) error` decodes keys with **`url.ParseQuery(r.URL.RawQuery)`**, so encoded `%5B…%5D` brackets work. The key grammar is `name[key]`:
    - map targets convert values to the value type, and repeated keys become slices;
    - struct targets match by json name, and unknown keys are ignored;
    - `name[a][b]` → error (400).
  - **Map left nil when no keys are present**, because validator.v9 `required` checks only nil.
  - No 256-key cap. `net/url` already caps a query at 10000 params (net/url/url.go:961, GODEBUG `urlmaxqueryparams`). There is an optional `bind.WithMaxKeys(n)`.
- Red first: `TestQueryDeepObject` table:
  - `filter[attr.sss]=v` → key `attr.sss`;
  - `filter[name]=x&filter[age]=3`;
  - `filter%5Bname%5D=x`;
  - a type error;
  - a nested key → error;
  - no keys → nil map.
- Deps: P6.1 (TagNames). Can run in parallel from P6.1 onward (disjoint files).

**P7.2b Tonic adapter + install.** M · Sonnet
- Files: bindhook.go (new, fizz root), router.go (Handle), config.go.
- Change:
  - `fizz.BindHook(next tonic.BindHook) tonic.BindHook` calls `next` first, then `bind.Query` with the plan taken **from the request context** (rule c).
  - `fizz.InstallBindHook()` is **process-global** (documented as such): it does `tonic.SetBindHook(BindHook(tonic.GetBindHook()))` once. `New` calls it unless `WithoutBindHook()` is passed.
  - The plan is built from **`hfunc.InputType()`**, not `InputModel`. If `InputModel` deref ≠ `InputType` deref, the plan is skipped and a warning recorded.
  - Errors are wrapped as `tonic.BindError`, so the user's error hook maps them to 400.
  - The Handle wrapper logs a one-time gin debug warning if a plan exists but the hook didn't run (the user replaced the hook).
  - Ordering is safe: tonic calls the hook at handler.go:75, before query binding at :80, and `filter` is absent from `URL.Query()`, so tonic skips the field (handler.go:205-219).
- Red first: an httptest suite: a 200 with the populated map, `validate:"required"` and `dive` still run (handler.go:97), composition with a user hook, and the InputModel-mismatch warning.
- Deps: P7.1, P7.2a, P5.1.

**P7.2c Cookie binding.** M · Sonnet
- Files: bind/cookie.go, bindhook.go.
- Change: `bind.Cookies(r, dst, p)` reads `r.Cookie(name)` for scalars, applies the `default` tag when the cookie is absent (tonic does not bind cookies), and leaves required-ness to `validate:"required"`.
- Red first: `TestCookieBind` (present, absent with default, absent + required → 400, bad type → 400).
- Deps: P7.2b, P4.2c.

## Phase 8: Docs UI (disjoint files: can run in parallel from P1.3)

**P8.1 Renderer core.** M · Sonnet
- Files: docsui/{docsui.go,swagger.go,scalar.go,page.tmpl}, docs.go (fizz root).
- Change:
  - `docsui.SwaggerUI(specURL, ...Option) http.Handler` and `docsui.Scalar(specURL, ...Option) http.Handler`, rendered with html/template.
  - Pinned jsDelivr URLs (versions verified in P0.6) with SRI sha384 constants, `crossorigin="anonymous"`, and a referrer policy.
  - Swagger UI config always sets `validatorUrl: null`.
  - Options: `WithAssetURLs(js, css, integrity...)` (an override without an integrity value drops SRI; documented), `WithTitle`, `WithUIConfig(map[string]any)`, `WithCSPNonce`.
  - `f.Docs(path string, h http.Handler)` registers an undocumented GET via `gin.WrapH` (rule b).
- Red first: golden HTML. `TestSRIAttributesMatchConstants`.
- Deps: P0.6, P1.3.

**P8.2 Asset pinning via `go generate`.** S · Sonnet
- Files: docsui/internal/pinassets/main.go, docsui/embedded/gen.go (`//go:generate go run ../internal/pinassets`), docsui/embedded/sri_test.go.
- Change: a maintainer runs `go generate` to download the pinned versions, refresh the embedded copies and rewrite the SRI constants. There is no scheduled CDN job. An offline test checks that the sha384 of each embedded file equals the constants.
- Red first: `TestEmbeddedSRI`.
- Deps: P8.1.

**P8.3 Embedded subpackage.** M · Sonnet
- Files: docsui/embedded/{embedded.go,assets/*}.
- Change: `//go:embed assets`. `embedded.SwaggerUI(specURL, prefix, ...)` and `embedded.Scalar(...)` serve `<prefix>/assets/*`. Scalar runs with `withDefaultFonts: false`, so there are no external font requests.
- Red first: `TestEmbeddedNoExternalURLs`, which parses the HTML and asserts there is no `http(s)://` src or href.
- Deps: P8.2.

**P8.4 Compatibility notes.** S · Haiku (browser check of the kitchen sink in both UIs) + Sonnet (draft)
- Files: docs/compat.md.
- Change: the P0.6 table, updated with real output. Includes the documented Swagger UI gaps, with **an explicit example of a `$ref` sibling `description` that Swagger UI ignores**, and the `AnnotationStyle: AllOfWrap` workaround.
- Red first: none (docs). Accept: Opus review.
- Deps: P8.3, P2.6.

## Phase 9: Docs, examples, CI, release

**P9.1 MIGRATION.md (v0.23 → v0.30).** M · Haiku (collects golden diffs) + Sonnet (drafts)
- Covers:
  - `SchemaOrRef` → `*Schema`, OrRef aliases and `.Value`/`.Ref`, `TypeSet`, `json.Number`, `Explode *bool`;
  - YAML tags removed, and **user `MarshalYAML` on examples is ignored** (YAML derives from JSON);
  - `SpecGenConfig` → `TagNames`/`BinderTags`;
  - panics → `Errors()`/`Err()`;
  - the auto-installed global bind hook;
  - type mapping, op ID, component name and body naming changes, nullable output, Duration in ns, url.URL as object, DELETE bodies;
  - **codegen tools:** oapi-codegen and kin-openapi are 3.0-only, and ogen and openapi-generator have partial 3.1/3.2 support. Recipe: a `DocumentHook` that rewrites for downstream tools, or stay on v0.23;
  - **users' snapshot tests will break.** Recipe for regenerating them;
  - Go ≤1.26 users are excluded.
- Deps: P7.2c.

**P9.2 README.** M · Sonnet
- Change:
  - 3.2.1 throughout: the link at README:393, and README:256, which wrongly says validator.v8.
  - New sections: tags table (TagNames vs BinderTags), interfaces, hooks + §1 precedence, naming, deepObject (nesting policy), cookies, docs UI, YAML.
  - Replace "Known limitations" (README:407-428).
- Deps: P7.2c, P8.1.

**P9.3 Market example.** S · Sonnet
- Files: examples/market/*, examples/market/market_test.go.
- Change: options, a Scalar docs route, a deepObject filter on ListFruits, a cookie param, RequestBody, CustomizeOperation, and a DocumentHook adding x-tagGroups.
- Red first: `TestMarketSpecValid`, plus an httptest call to the filter endpoint.
- Deps: P7.2c, P8.1.

**P9.4 CI.** S · Sonnet
- Files: .github/workflows/*.
- Change:
  - Matrix per D2.
  - Jobs: `-race`, golangci-lint (action), vet, staticcheck and govulncheck via `go tool -modfile=tools/go.mod`.
  - Meta-schema + determinism job: `-count=1 -shuffle=on`, kitchen sink 3×.
  - Archtest import rule; coverage floor 90%.
  - goreleaser release workflow with a changelog.
- Deps: P0.1 (it can land any time after P0.1, but before v0.30.0).

**P9.5 Release.** S · Sonnet
- Content: see §15.
- Deps: everything above the cut line.

---

## Phase 10: Official-spec conformance suite (added 2026-10-31; release gate)

Goal: prove that fizz output follows OAS 3.2.1 **and** matches real, official example documents, not only fizz's own goldens. Before this phase, the 3.2 meta-schema was the only external reference; no OAI document was ever used.

Decisions (user, 2026-10-31):
- **D22 Corpus.** Vendor official OAI material, pinned by commit and sha256. The 3.0/3.1 examples are **upgraded to 3.2.1** by a deterministic converter. Originals and upgraded copies are both committed, and every change is listed in a NOTICE (CC BY 4.0 requires this).
- **D23 Reproduction rule.** A fizz app rebuilds each example. Its generated spec is compared as JSON with the upgraded official doc. Any difference fails unless it appears in a per-example allowlist with a reason. An allowlist entry that is no longer used also fails. Opus reviews every allowlist entry.
- **D24 Timing.** Phase 10 runs before P9.5 and is above the cut line. The open review findings become red-until-done cases (FixedBy P10.7x).

Sources (verified 2026-10-31):

| Source | Pin | Licence | Content |
|---|---|---|---|
| OAI/learn.openapis.org `examples/` | main@82b82f372aa185a07ea11bc158619895ab21f741 | CC BY 4.0 | v3.0: api-with-examples, callback-example, link-example, petstore, petstore-expanded, uspto. v3.1: non-oauth-scopes, tictactoe, webhook-example. v3.2 (openapi 3.2.0): 3.2-query-example, 3.2-tags-example |
| OAI/OpenAPI-Specification `tests/schema/{pass,fail}` | dev@0f9aa49039215e2ed414cdefe35f718273b77976 | Apache-2.0 | 15 pass and 5 fail YAML fixtures, all 3.1.x. There are no 3.2 fixtures upstream |
| OAI/OpenAPI-Specification `versions/3.2.1.md` | main@4f86429f134f7e48b6351563549b4502fc69de52 | Apache-2.0 | 127 yaml blocks: 6 full 3.2.1 docs, the rest fragments |

Rules for every task:
- `go test` **never** touches the network. Fetching is done by a tool run by hand (the `docsui/internal/pinassets` pattern: verify by default, `-update` refetches and rewrites the manifest).
- Vendored files are data. They are only parsed, never executed or used as templates.
- Every corpus document must pass `specvalidate.Validate` after upgrading. A document that cannot is excluded in the manifest with a reason, never silently.

**P10.0 Vendor corpus + fetch tool.** S · Haiku (fetch inventory) + Sonnet
- Files: `internal/conformance/oai/{orig,MANIFEST.json,NOTICE,LICENSE-*}`, `internal/conformance/internal/fetchoai/`.
- MANIFEST lists source repo, commit, path, sha256 and licence for each file.
- Red first: `TestManifestMatchesFiles` (sha256 of every vendored file equals the manifest; no unlisted file).

**P10.1 Upgrader to 3.2.1 + spec-text extraction.** M · Sonnet
- Files: `internal/conformance/upgrade/`, `internal/conformance/oai/upgraded/`.
- 3.0→3.1 rules follow the official 3.1 migration guide: `nullable` → type array (plus null in `enum`), boolean `exclusiveMinimum/Maximum` → numbers, `format: binary/base64` file fields → `contentMediaType`/`contentEncoding`, `example` kept. 3.1→3.2: set `openapi: 3.2.1` and nothing else (3.2 is a superset).
- The converter records each change as a JSON-pointer list. NOTICE is generated from it.
- Spec text: extract yaml fences from `versions/3.2.1.md`. Full documents are kept as-is. Fragments are wrapped by kind (Schema → `components/schemas`, Parameter → `components/parameters`, Path Item → `paths`, …). Fragments that cannot be classified are listed in the manifest as excluded, with a reason.
- Red first: `TestUpgradeGolden` (converter output equals committed `upgraded/`), `TestCorpusValid` (every upgraded or extracted doc passes `Validate`).

**P10.2 Validator conformance + semantic lints.** M · Sonnet
- Files: `internal/specvalidate/{lint_semantic.go,conformance_test.go}`.
- The upgraded OAI `pass` fixtures must validate. Each `fail` fixture must be rejected, and the expected failing pointer is recorded per fixture, so the rejection is for the right reason.
- New `LintSemantics` for OAS rules JSON Schema cannot express:
  - path template variables have exactly one `in: path, required: true` param;
  - (name, in) is unique per operation;
  - operationId is unique;
  - a parameter has exactly one of `schema`/`content`;
  - `in` is valid;
  - header params named Accept, Content-Type or Authorization are reported (OAS: SHALL be ignored).
- The lint must accept the whole official corpus. That tests the lint as much as fizz. Wire it into `MetaProblems`.
- Red first: `TestOfficialPassFixtures`, `TestOfficialFailFixtures`, `TestLintSemanticsOnCorpus`.

**P10.2b Lint unresolved local `$ref`s.** S · Sonnet. `LintSemantics` rule (i): every `$ref` (and plain-pointer `$dynamicRef`) starting with `#/` must resolve as an RFC 6901 pointer in the same document; anchors, external and relative refs are skipped. The spec-text and schema-golden wrappers stub the components their fragment leaves out. The link repro app is red until P10.9k.

**P10.3 Model round-trip on the corpus.** M · Sonnet
- Files: `internal/conformance/roundtrip_test.go`.
- For every corpus doc: decode into `openapi.OpenAPI`, encode as JSON and YAML, then check the result is JSON-semantically equal to the input and still valid.
- Every loss is a red case with FixedBy (a new task ID if needed), never an allowlist.
- Red first: `TestCorpusRoundTrip`.

**P10.3a… Spec-model fixes found by P10.3** (each is red first in `internal/acceptance/corpus_roundtrip_test.go` and in the listed `TestCorpusRoundTrip` subtests; fixing one means dropping its FixedBy and entries in `roundTripFixedBy`)
- **P10.3a** `Parameter` (and `Header`) `required: false` is dropped (`omitzero` bool). Files: learn v3.0 petstore, petstore-expanded, v3.1 tictactoe, spectext/parameter-object-examples-5. Likely `openapi/spec_paths.go`.
- **P10.3b** `OpenAPI.webhooks: {}` is dropped, so a webhooks-only document no longer validates. Files: oas pass/minimal_hooks. Likely `openapi/spec_root.go`.
- **P10.3c** `Components.pathItems: {}` is dropped. Files: oas pass/comp_pathitems. Likely `openapi/spec_root.go`.
- **P10.3d** `Operation.parameters: []` is dropped. Files: oas pass/mega. Likely `openapi/spec_paths.go`.
- **P10.3e** `Schema.properties: {}` is dropped. Files: spectext/example-url-encoded-form-with-json-values-1. Likely `openapi/spec_schema.go`.
- **P10.3f** a `RequestBody` without `content` is encoded as `content: null`. Files: spectext/base-uri-within-content-1, resolving-relative-self-and-id-1. Likely `openapi/spec_paths.go`.

**P10.4 Gate every golden.** S · Sonnet
- Files: `internal/conformance/goldens_test.go`.
- Walks the repo for golden JSON/YAML that is a whole OAS document (`testdata/spec.*`, `openapi/testdata/schemas/*` wrapped as components, kitchensink, acceptance) and runs `MetaProblems`. New goldens are covered automatically.
- Red first: `TestAllGoldensValid`.

**P10.5a/b/c Reproduction apps.** L total, 3 parallel lanes · Sonnet (Opus reviews allowlists)
- Files: `internal/conformance/repro/<example>/{app.go,allow.json,repro_test.go}`. One dir per example, so the lanes are disjoint.
- Each app uses only public fizz API (types, tags, options, hooks) to rebuild the example. The test compares the served `/openapi.json` with the upgraded official doc under D23.
- An allowlist entry is `{pointer, kind: missing|extra|changed, reason}`. Allowed reasons:
  - (a) naming fizz deliberately controls, e.g. component names (ADR);
  - (b) an upstream example bug (link it);
  - (c) a fizz limitation with a task ID.
- (c) entries are listed in README "Known limitations".
- Lanes:
  - a: petstore, petstore-expanded, api-with-examples, uspto.
  - b: callback-example, link-example, webhook-example, non-oauth-scopes, tictactoe.
  - c: 3.2-query-example, 3.2-tags-example, plus the 6 full docs from the spec text where an app is feasible.
- Red first: each `TestRepro<Name>`.

**P10.6 Spec-driven contract tests.** M · Sonnet
- Files: `internal/conformance/contract/`.
- A small client reads a served spec. For each operation it serializes sample values (from example, default, enum, or else type) for every parameter, following the documented `style`/`explode`: form, simple, spaceDelimited, pipeDelimited, deepObject, cookie. It sends the request, and an echo handler asserts that the bound Go value equals the sent value.
- Runs against the kitchen-sink fixture, `examples/market` and the P10.5 apps.
- This catches doc/runtime drift (review findings 1, 4, 6).
- Red first: `TestContractFixture` (the dedicated contract fixture), `TestContractMarket`, `TestContractRepro`.

**P10.7 Fix the open review findings** (each one is red first in P10.2/P10.6 or as a kitchen-sink case, one commit per item):
- **P10.7a** `mergeParameter`: an option's Content replaces the Schema (and vice versa). `applyParameters` rejects an invalid `In` with an op error.
- **P10.7b** A non-deepObject query, header or path param of map or struct type is a generation error (same rule as cookies).
- **P10.7c** A slice `default` tag is split on `,` and converted per element, as tonic does.
- **P10.7d** Nullable items (`[]*T`) get null in `enum`/`const`.
- **P10.7e** Slice header params. The doc says `simple` (comma-split), but tonic does not split. **D25 (user, 2026-10-31):** the fizz bind hook splits simple-style slice header values on `,` and merges repeated header lines, so the runtime matches the doc. Runtime change: mention it in MIGRATION.
- **P10.7f** Reserved header param names (Accept, Content-Type, Authorization): op warning that points to security schemes / content.
- **P10.7g** Carried over from earlier: duplicate path+method is silently overwritten (→ op error); `RegisterRequestBody` with nil Content emits `content: null` (→ error).
- **P10.7h** Equivalent path templates. Routes that differ only in variable names (e.g. `GET /test/:a` + `POST /test/:c`, which gin accepts per method) produce two path keys that OAS forbids. P10.4 found it in the root fixture. Record an op error naming both routes. Red first: a kitchen-sink case.
- **P10.7i** Optional header parameter that is absent. tonic's `extractHeader` returns one empty value instead of none, so an optional integer, number or boolean header fails with a 400 (`ParseInt: parsing ""`), an optional enum header fails the enum check, and a `default` tag on a header is never applied (the field stays zero). The document says all of these are optional. Red first: `TestContractFixture/headerscalars/required_only`, `headerenum/required_only`, `defaults/header_X-Lang_default` (FixedBy P10.7i).
- **P10.7j** `spaceDelimited` / `pipeDelimited` query arrays are documented (`explode: false`), but tonic splits on `,` only, so `?space=a%20b` binds `["a b"]`. Fix in the bind hook (split on the delimiter) or reject the style at generation. Red first: `TestContractFixture/arrays/query_space`, `query_pipe`.
- **P10.7k** `label` and `matrix` path parameters are documented, but tonic binds `c.Param` as is, so `/label/.s1` binds `".s1"` and `/matrix/;pm=s1` binds `";pm=s1"`. Fix in the bind hook (strip the delimiter and name) or reject the styles. Red first: `TestContractFixture/label`, `matrix`.
- **P10.7l** A slice path parameter is documented as `simple` (comma-separated), but tonic does not split it (`/patharr/s1,s2` binds `["s1,s2"]`). Same remedy as P10.7e for headers: split in the bind hook, or make it a generation error. Red first: `TestContractFixture/patharr`.
- **P10.7m** A `form` cookie (the default style) is documented as percent-encoded, but the bind hook keeps the raw value (`a%20b` stays `a%20b`; B24 chose this). Either document `style: cookie` (no encoding) or decode. Red first: `TestContractFixture/special/cookie_csp`.

**Phase 10 decisions after the findings (user, 2026-10-31):**
- **D26 Runtime follows the doc.** The fizz bind hook is extended so that every style fizz documents is parsed as documented:
  - P10.7i: absent optional headers are skipped and header `default`s applied;
  - P10.7j: spaceDelimited/pipeDelimited are split;
  - P10.7k: label/matrix prefixes are stripped;
  - P10.7l: slice path params are split;
  - P10.7e: as D25.
  Runtime changes → MIGRATION.
- **D27 Cookies (P10.7m).** Cookie params are emitted with `style: cookie` (OAS 3.2, no percent-encoding), which matches the raw-value binder (B24). This is a spec output change only.
- **D28 Cut line.**
  - Above the line: P10.3a–g, P10.7a–m, P10.9c, P10.9d, P10.9n.
  - Below the line (missing features, kept allowlisted and listed in README "Known limitations"): P10.9a, b, h, i, k, l, m, o, p.
- **P10.3g** `PathItem.Parameters` and `Link.Parameters` still use `omitempty`, so empty containers are dropped (same class as P10.3b–e). Found during the P10.3 fixes.
- **P10.5d** Harness hygiene: a `limitation` allowlist entry whose task is in `plan.Done` fails (stale). The reason must then be reclassified or the entry removed.

**P10.8 Second validator (below the line).** S · Haiku (check whether any Go or CLI validator supports 3.2 at that time) + Sonnet
- Only if one exists: cross-check the corpus and all goldens in a test-only module (`tools/` or `internal/conformance/x/`), so the library go.mod stays unchanged.

**P10.9x Limitations found by reproduction** (referenced by the `limitation` entries of `internal/conformance/repro/*/allow.json`; each one is a task that removes its allowlist entries)
- **P10.9a** The default (tonic) response has no singular `example`, only `examples` (`DefaultResponseExamples`); `Response`/`ResponseWith` with the same code is a duplicate error. Affects uspto. Missing feature.
- **P10.9b** A required body field loses its `default` tag (warning), but a required property with a default is valid OAS. Affects uspto (`criteria`). Missing feature.
- **P10.9c** A document with no component still emits `"components": {}`. Affects api-with-examples. Cosmetic, valid OAS. Missing feature (omit the empty object).
- **P10.9d** The `description` tag on a parameter is duplicated into its schema (`schema.description`), besides `parameter.description`. Affects petstore, petstore-expanded, uspto. Bug (red case `params/description-tag-not-in-schema`).
- **P10.9n** A document with no route still emits `"paths": {}`; OAS 3.2 allows omitting it when there are components. Affects spec-text base-uri-from-the-retrieval-uri-1. Cosmetic, valid OAS. Missing feature (omit the empty object, like P10.9c).
- **P10.9o** An operation cannot reference an external request body: `Operation.RequestBody` is a `*RequestBody` with no `$ref` form, so `requestBody: {"$ref": "shared/foo#/components/requestBodies/Foo"}` cannot be written. Affects spec-text base-uri-within-content-1, resolving-relative-self-and-id-1 (not reproduced). Missing feature.
- **P10.9p** `Schema` has no `$id` keyword (nor `$anchor`), so a component schema cannot carry its identifier. Affects spec-text base-uri-within-content-2, resolving-relative-self-and-id-2 (not reproduced). Missing feature.
- ~~P10.9f~~ / ~~P10.9g~~ withdrawn (Opus review). A mandatory `operationId` and at least one response are deliberate fizz design (ADR op-ID rules; tonic handlers always have a status). Allowlisted as `naming`, consistent with P10.5c.
- **P10.9h** The description of an inline schema (anonymous struct body or response) cannot be set: the `description` tag is for fields and a `SchemaCustomizer` needs a named type, which becomes a component. Affects callback. Missing feature.
- **P10.9i** No API for webhooks (no option, router method or `Generator` setter), so the whole `webhooks` subtree is missing. Affects webhook-example, tictactoe. Missing feature.
- ~~P10.9j~~ merged into P10.9n (same symptom: empty `paths: {}`).
- **P10.9k** No `Generator.RegisterLink` (nor `RegisterCallback` / `RegisterPathItem`), so `components.links` cannot be registered; the `$ref`s to them (set with `CustomizeOperation`) dangle. Affects link-example. Missing feature.
- **P10.9l** No path-item `parameters` and no `$ref` to a component parameter on an operation: shared path parameters are repeated inline in every operation. Affects tictactoe. Missing feature.
- **P10.9m** `ResponseWith`'s `WithExample` / `WithExamples` go under the default media type only; `WithContent` has no example, so a non-default media type (`text/html`) cannot carry one. Affects tictactoe. Missing feature.

---

## 13. DAG (explicit, with file sets)

| Task | Depends on | Files touched (primary) | ∥ allowed with |
|---|---|---|---|
| P0.1 | — | go.mod/sum, tools/go.mod, workflows, ci/test.sh, all *.go | — |
| P0.2 | P0.1 | internal/golden, *_test.go golden reads | P0.3, P0.6 |
| P0.3 | P0.1 | internal/specvalidate, go.mod | P0.2 |
| P0.6 | P0.3 | docs/compat/** | P0.2, P0.4a |
| P0.4a | P0.2, P0.3 | internal/kitchensink/{case,assert,fixtures}, testdata/legacy | P0.6 |
| P0.4b–f | P0.4a | internal/kitchensink/cases_*.go (one each) | each other, P0.5, P1.x |
| P0.5 | P0.4a | determinism_test.go, race_test.go | P0.4b–f, P1.x |
| P1.0 | P0.4a, P0.6 | docs/adr, internal/apistub, internal/archtest | P0.4b–f |
| P1.1 | P1.0 | openapi/*.go, fizz.go → split files | P0.4b–f, P0.5 |
| P1.2 | P1.1 | generator.go, types.go, go.mod | P0.4b–f |
| P1.3 | P1.2 | generator/params/responses/fields.go, paramtag.go, path.go, router.go, fizz.go | P0.4b–f |
| P2.2 | P1.3 | spec_schema.go | Phase 8 |
| P2.1a→b→c→d | prev | spec_ref.go / spec_root.go / spec_paths.go / spec_security.go | Phase 8 |
| P2.3 | P2.1d | all spec_*.go, encode.go | Phase 8 |
| P2.5 | P2.3 | spec_*.go, encode.go, spechandler.go, go.mod | Phase 8 |
| P2.6 | P2.5, P0.4b–f | openapi generator files, sort.go, testdata | Phase 8 |
| P3.1a | P2.6 | typefields.go, fields.go | P3.2 |
| P3.1b | P3.1a | params.go | P3.2 |
| P3.2 | P2.6 | schema.go | P3.1a, P3.1b |
| P3.3 | P3.1b, P3.2 | types.go, schema.go | — |
| P3.5 | P3.3 | schema.go, fields.go, generator.go | — |
| P3.4 | P3.5 | fields.go, schema.go | — |
| P3.6 | P3.4 | types.go, examples.go, fields.go | — |
| P6.1 | P3.6 | config.go, fizz.go, openapi/tags.go, generator.go (config only) | P4.1a, P4.5 |
| P4.1a | P3.6 | openapi/internal/vtag | everything in Phases 4–6 |
| P4.1b | P4.1a | validate_tag.go, validation.go | chain H, P4.4, P4.5 |
| P4.1c | P4.1b | fields.go | chain H, P4.4, P4.5 |
| P4.5 | P3.6 | naming.go, registry.go, schema.go | P6.1, lane V, P4.2, P4.2c, P4.4 |
| P4.4 | P6.1 | responses.go, operation.go, options.go, config.go | lane V, P4.5, P4.2, P4.2c, P4.6 |
| P4.2 | P3.6 | params.go, paramtag.go | P6.1, lane V, P4.5 |
| P4.2c | P4.2, P6.1 | params.go | lane V, P4.5, P4.4 |
| P4.6 | P4.2c, P4.5 | params.go, generator.go | lane V, P4.4 |
| P4.9 | P4.6 | router.go | lane V, P4.4 |
| P4.8 | P4.9, P4.4 | generator.go, responses.go, router.go, spechandler.go, fizz.go | lane V |
| P4.7 | P4.8 | router.go, generator.go | lane V |
| P4.3 | P4.7 | path.go, params.go, generator.go | lane V |
| P5.1 | P4.3 | generator.go, clone.go, spechandler.go, router.go | P6.2, P7.2a |
| P6.2 | P6.1, P3.4, P4.1c, P4.3 | fields.go, params.go | P5.1, P7.2a |
| P6.3 | P4.5, P3.3 | interfaces.go, schema.go, types.go | P6.5, P5.1, P6.2 |
| P6.5 | P4.5, P5.1 | components.go, generator.go, router.go | P6.3, P6.2 |
| P6.4 | P6.2, P6.3, P6.5 | hooks.go, schema/fields/params/generator.go, spechandler.go | P7.2a, Phase 8 |
| P6.6a→b→c | P6.4, prev | options.go, operation.go, params.go (+generator.go in c) | P7.2a |
| P7.1 | P6.6c, P6.2 | deepobject.go, params.go | P7.2a |
| P7.2a | P6.1 | bind/{plan,query}.go | Phases 4–6, P7.1 |
| P7.2b | P7.1, P7.2a, P5.1 | bindhook.go, router.go, config.go | Phase 8 |
| P7.2c | P7.2b, P4.2c | bind/cookie.go, bindhook.go | Phase 8 |
| P8.1 | P0.6, P1.3 | docsui/*, docs.go | Phases 2–7 |
| P8.2 | P8.1 | docsui/internal/pinassets, docsui/embedded/{gen,sri_test}.go | Phases 2–7 |
| P8.3 | P8.2 | docsui/embedded/* | Phases 2–7 |
| P8.4 | P8.3, P2.6 | docs/compat.md | Phases 3–7 |
| P9.1, P9.2, P9.3 | P7.2c (+P8.1 for 9.2/9.3) | MIGRATION.md / README.md / examples/market | each other |
| P9.4 | P0.1 | workflows | anything |
| P10.0 | — | internal/conformance/oai, internal/conformance/internal/fetchoai | anything |
| P10.1 | P10.0 | internal/conformance/upgrade, oai/upgraded | anything outside conformance |
| P10.2 | P10.1 | internal/specvalidate/*, internal/kitchensink/meta.go | P10.3, P10.4, P10.5x |
| P10.2b | P10.2, P10.5b | internal/specvalidate/lint_semantic*.go, internal/conformance/upgrade/spectext.go, oai/upgraded/spectext | P10.3, P10.4, P10.5x |
| P10.3 | P10.1 | internal/conformance/roundtrip_test.go | P10.2, P10.4, P10.5x |
| P10.3a–f | P10.3 | openapi/spec_paths.go / spec_root.go / spec_schema.go | each other when they share a file, P10.2, P10.4, P10.5x |
| P10.4 | P10.2 | internal/conformance/goldens_test.go | P10.3, P10.5x |
| P10.5a/b/c | P10.1 | internal/conformance/repro/<example>/ (one dir each) | each other, P10.2–P10.4 |
| P10.6 | P10.5a | internal/conformance/contract | P10.7x |
| P10.7a–g | their red case (P10.2/P10.6) | params_option.go / params.go / fields.go / generator.go / bindhook.go | serialized on chain H when they touch generator.go/params.go |
| P10.8 | P10.4 | tools/ or internal/conformance/x | anything |
| P10.9a–d, h, i, k–p | P10.5a/b/c | responses.go / operation_options.go (a, m), params.go / fields.go (b, d, l), generator.go / spec_root.go (c, n), components.go / router.go / spec_root.go (i, k), schema.go (h), spec_paths.go (o), spec_schema.go (p) | serialized on chain H when they touch generator.go/params.go |
| P9.5 | cut line | tags, notes | — |

## 14. Estimates and critical path (rough; agent-days plus review)

| Phase | Work (sum) | Critical-path contribution |
|---|---|---|
| 0 | ~9 d | ~4 d (P0.1→P0.2→P0.4a; P0.4b–f off-path) |
| 1 | ~6 d | ~5 d |
| 2 | ~9 d | ~9 d (serial) |
| 3 | ~15 d | ~11 d (P3.1a+b ≈ 4.5 d vs P3.2 3 d) |
| 4 (+P6.1) | ~16 d | ~8 d (chain H) |
| 5 | ~2 d | ~2 d |
| 6 | ~13 d | ~8 d (P6.3∥P6.5 → P6.4 → P6.6a–c) |
| 7 | ~8 d | ~5 d (P7.2a off-path) |
| 8 | ~5 d | 0 (parallel) |
| 9 | ~6 d | ~3 d |
| 10 | ~14 d | ~7 d (P10.0→P10.1→P10.5a→P10.6→P10.7x) |
| **Total** | **~89 d of work** | **~55 d ≈ 11 weeks for one maintainer** |

**Critical path:** P0.1 → P0.2 → P0.4a → P1.0 → P1.1 → P1.2 → P1.3 → P2.2 → P2.1a–d → P2.3 → P2.5 → P2.6 → P3.1a → P3.1b → P3.3 → P3.5 → P3.4 → P3.6 → P6.1 → P4.2 → P4.2c → P4.6 → P4.9 → P4.8 → P4.7 → P4.3 → P5.1 → P6.5 → P6.4 → P6.6a → P6.6b → P6.6c → P7.1 → P7.2b → P7.2c → P9.1/P9.2 → v0.30.0.

**Cut line for v0.30.0** (the "rc" of the overhaul):
- **Above the line (blocking):** Phases 0–7, P8.1, P9.1, P9.2, P9.4, P10.0–P10.7 (incl. P10.2b, P10.3a–g, P10.5d, P10.7a–m), P10.9c, P10.9d, P10.9n.
- **Below the line (may ship in v0.30.x):** P8.2, P8.3, P8.4, P9.3, P10.8, P10.9a, b, h, i, k, l, m, o, p (D28).

## 15. Release plan
1. Branch `release-0.x` from tag v0.23.0. **Optional v0.23.1:** only a visited-set cycle fix in `buildSchemaRecursive`, `flattenStructSchema` and `buildParamsRecursive` (generator.go:523, :1032). The go directive and dependencies stay unchanged. Covered by the P0.4b/c cycle cases backported as plain tests.
2. Master becomes the overhaul line. No release before v0.30.0 (v0.24.0 is dropped).
3. **v0.30.0** ships when everything above the cut line is done. The release notes cover: Go 1.27 floor (Go ≤1.26 excluded), OAS 3.2.1 only, a link to MIGRATION, the global bind hook, and codegen-tool caveats.
4. **v0.30.x:** below-the-line tasks plus fixes. Non-breaking only.
5. **v1.0.0:** after at least 6 weeks on v0.30.x with no open breaking-change issues, below-the-line tasks landed, and the kitchen-sink golden unchanged across 2 consecutive patch releases. Tag only; no API change from the last v0.30.x.

## 16. Risks
1. **Golden churn in Phases 2–3 hides regressions.** Mitigations: the bug cases are hand-written target fragments (not generated), per-case goldens appear only after P2.6, and every regeneration gets Opus review.
2. **3.2 tool support.**
   - Swagger UI may ignore `$ref` siblings (P8.4 example); mitigation is the `AllOfWrap` option.
   - oapi-codegen and kin-openapi are 3.0-only; ogen and openapi-generator are partial. Mitigation: MIGRATION recipe via `DocumentHook`.
   - The P0.6 spike gates this risk.
3. **Global tonic bind hook.** A user who replaces it after `New` loses deepObject and cookie binding. Mitigations: the debug warning, exported `BindHook`/`InstallBindHook`, and docs that state it is global.
4. **tonic's `routes` map is written unlocked** (handler.go:132). Concurrent `tonic.Handler` creation is a data race outside fizz's control. Race tests create handlers before concurrency, and the docs say to build routes at init.
5. **`tonic.MediaType()` ignores `SetRenderHook`** (tonic.go:140-142). Generated media types can disagree with the rendered output. Mitigation: `WithDefaultMediaType`.
6. **encoding/json dominance port diverges.** Mitigation: the P3.1a differential test.
7. **Snapshot-test breakage for users.** Every spec byte changes; flagged prominently in MIGRATION.
8. **User `MarshalYAML` on example values is ignored** now that YAML derives from JSON.
9. **json/v2 semantics.**
   - `omitempty` differs from v1 (a v2 `false` is not "empty"); use `omitzero` for bool and number fields.
   - v1 callers marshaling nested objects may drop `inline` extensions; covered by the P2.3 parity test.
10. **uint64 and big numbers exceed float64.** `json.Number` keeps the spec exact, but clients may not.
11. **validator.v9 vs v10 rules.** Unknown rules are ignored by the mapping; `datetime` is not mapped.
12. **Embedded assets add a few MB to every consumer's module download** (D5). Accepted; it is documented.
13. **CDN pin or SRI drift.** Mitigations: the offline SRI test, `go generate`, and the embedded fallback.

## 17. Post-v1 roadmap (not tasks)
- A fizz-owned generic handler and binding layer (`fizz.Handler[In,Out]`, validator v10) with router adapters for gin, net/http 1.22+ patterns, chi and echo. It would reuse `bind` (rule a), `http.Handler` docs/spec (rule b) and gin-free hook contexts (rule c). A tonic compatibility layer would map `tonic.Handler` routes onto it.
- OAS features already in the model but without API: Callbacks, Links, Webhooks, `AdditionalOperations`, the QUERY method, and the `Register{Link,Callback,PathItem,MediaType}` setters.
- `OneOfer` + Discriminator inference.
- Multipart and form request bodies (`*multipart.FileHeader`, encoding map), the old P4.10.
- Collision policies `CollisionSuffix` and `CollisionFullPath`.
- A `Strict()` mode.
- A scheduled CDN hash check and asset downloader in CI.

## 18. Open questions (new only)
1. **If P0.6 shows that the pinned Swagger UI does not render `openapi: 3.2.x`:** do we make Scalar the default and document Swagger UI as best-effort, or bump the Swagger UI pin to whatever release supports 3.2 (re-verified)? D1 rules out downgrading the output version.
2. **Bind-hook installation:** auto-install in `New()` with `WithoutBindHook()` opt-out (as planned), or require an explicit `fizz.InstallBindHook()` call, since it mutates process-global tonic state?
3. **D15 on params** silently drops `required` when a default exists. Is a warning enough, or should it be a generation error?

---

4. ~~O-10a~~ resolved as D25 (bind hook splits slice headers).

## Addendum — user decisions after v2 (2026-10-30)

- **D17 Default docs UI = Swagger UI** (user decision: most widely used/familiar). Scalar is the explicit alternative via `fizz.ScalarUI()`. Verified 2026-10-30: Swagger UI 5.34.1 detects 3.2.1 (`^3\.2\.(?:[1-9]\d*|0)$`), has basic 3.2 support (5.32+) and additionalOperations (5.34.0), but open gap #11065 (itemSchema not rendered) and unverified $ref siblings / tag kind+parent — these gaps MUST be documented in README/godoc, with a pointer to ScalarUI for fuller 3.2 rendering. P0.6 spike must confirm Swagger UI renders the kitchen-sink 3.2.1 doc acceptably (esp. $ref siblings, anyOf-nullable); if it doesn't, raise it to the user — do not silently switch default.
- **D18 Explicit choice is always honored — no silent fallback.** `f.Docs(path, spec)` with no renderer = documented default (Swagger UI). `fizz.SwaggerUI()` always serves Swagger UI; `fizz.ScalarUI()` always Scalar. Known 3.2 gaps of the chosen UI are documented (README + godoc), never auto-swapped. GoDoc of `Docs` states the default explicitly.
- **D19 Version pins** are chosen at P8.1 implementation time from then-current releases, verified by P0.6 spike (render a 3.2.1 kitchen-sink doc). Candidates verified 2026-10-30 (HTTP 200 on jsDelivr):
  - `swagger-ui-dist@5.34.1/swagger-ui-bundle.js`, `swagger-ui.css`, `swagger-ui-standalone-preset.js`
  - `@scalar/api-reference@1.76.4/dist/browser/standalone.js` (~4.9 MB)
  - `redoc@3.0.1/bundles/redoc.standalone.js` (possible later renderer)
- **D20 Bind hook auto-installed** by `fizz.New`/`NewFromEngine` (process-global, documented), opt-out `WithoutBindHook()`; warning logged if a plan exists but the hook was replaced. (Recommended default; user did not object — revisit if they choose manual.)
- **D21 default + required on a param:** drop `required`, record a warning in `Errors()` (confirms D15).

# fizz v0.30 Acceptance-Test Plan

Companion to `docs/plan/PLAN-v2.md` (tasks P0.1–P9.5, decisions D1–D21). The goal is to prove that every fix and feature in the plan works, and to write the tests **before** the code wherever possible.

State at writing (HEAD `e470f59`):
- `TestKitchenSink`: 101 cases. 12 pass, 89 skipped with `FixedBy`, 0 skipped but passing.
- Skips per task: P2.3=2 P2.5=2 P2.6=1 P3.1a=5 P3.1b=4 P3.2=4 P3.3=20 P3.4=4 P3.5=4 P3.6=6 P4.1b=8 P4.2=5 P4.2c=1 P4.3=3 P4.4=4 P4.5=3 P4.6=3 P4.7=2 P4.8=3 P4.9=1 P5.1=1 P6.5=2 P7.1=1.
- `TestAllCaseDocsValid` (`internal/kitchensink/schema_gate_test.go`) runs `Validate`, `LintNoStrayNull`, `LintNoNullable` and `LintComponentNames`. It is skipped "fixed by P2.6".
- Raw `t.Skip("fixed by …")` tests: `TestVersionPattern` (P2.6), `TestAllCaseDocsValid` (P2.6), `TestConcurrentServeAndRegister` (P5.1).

Conventions in this document:
- **IDs.** `K*` = new kitchen-sink cases. `A*`–`I*` = new acceptance tests, by batch (§3). `T:` = a task-owned red test that the task writes itself (see §2, rule 3).
- **Kinds.** `frag` = kitchen-sink fragment. `meta` = meta-schema/lint gate. `gold` = golden. `http` = httptest behaviour. `unit` = unit test. `imp` = import graph. `html` = browser-free HTML check. `doc` = `Example*` test or doc lint.
- **Build.** `now` = default build, compiles today. `tag` = `//go:build acceptance` (§2).

---

## 1. Traceability matrix

### 1.1 Plan tasks

Phase 0 is done. Its rows list the existing tests that keep guarding it.

| Task | Acceptance tests | Kind | Pass criteria |
|---|---|---|---|
| P0.1 | A04 `TestModuleInvariants`, A05 `TestToolsModuleIsolated`, CI matrix (§4 G-CI) | unit, CI | go.mod has `go 1.27`; `go list -m all` has no staticcheck/govulncheck; `go test -race ./...` green on 1.27.x and stable |
| P0.2 | `TestAssertReportsPointer` (existing) | unit | A mismatch message contains `/paths/~1x/get` |
| P0.3 | `TestRejectsBadDoc`, `TestAcceptsMinimalDoc`, `TestLintAllowsNullInValueSubtrees`, `TestVersionPattern` (existing) | unit | As written. `TestVersionPattern` is un-skipped at P2.6 |
| P0.4a | `TestFragmentAssert`, `TestLegacyGolden` (existing); Batch 0 harness tests 0.1–0.6 | unit, gold | A wrong fragment fails; the legacy golden is byte-equal until P2.6 |
| P0.4b–f | the 101 existing cases + K-fix1..3 (§3 K) | frag | Each case passes once its `FixedBy` task merges |
| P0.5 | `TestDeterminism`, `TestConcurrentServeAndRegister` (existing) | unit, race | 20× byte-identical at GOMAXPROCS 1 and 8; `-race` clean from P5.1 |
| P0.6 | `TestSpikeDocValid` (existing), C06 `TestSRIConstantsMatchCompatDoc` | meta, doc | Spike doc valid; docsui SRI constants equal the compat.md table |
| P1.0 | A01 `TestRouterAgnostic` (skip→P1.3), A02 self-check; G-STUB gate: `go vet -tags acceptance ./...` compiles | imp, build | The tagged acceptance suite compiles against the stubs. ADR sign-off: **untestable (process)** |
| P1.1 | `TestLegacyGolden`, `testdata/spec.{json,yaml}` goldens, coverage-delta script | gold | Byte-identical; `go test -cover` total unchanged (script, not a Go test) |
| P1.2 | A03 `TestNoUUIDDep`; `types/elem/uuid-google`, `types/elem/uuid-gofrs` (P3.3) | imp, frag | `go list -deps ./openapi` has no `github.com/gofrs/uuid`. Both uuid libs map to `format: uuid` |
| P1.3 | A01 `TestRouterAgnostic`, A08 `TestBinderTagsMatchTonic`; T: `TestGinPathTranslation`; K40 | imp, unit, frag | No gin or gadgeto in `go list -deps -test ./openapi/...`; BinderTags == tonic.go:24-33; legacy golden byte-identical |
| P2.2 | T: `TestSchemaMarshal`; `meta` gate from P2.6 | unit | Table in PLAN P2.2 |
| P2.1a | T: `TestOrRefMarshal`; `ops/orref-both-nil` (P4.8) | unit | ref / value / both-nil error |
| P2.1b | T: `TestRootMarshal`; H10 `TestSetSelfAndDialect` (P6.5) | unit | `$self`, `jsonSchemaDialect`, `Info.summary`, `License.identifier`, `Server.name`, Tag `summary/parent/kind` marshal |
| P2.1c | T: `TestOperationOmitZero`, `TestMediaTypeNoNullSchema`; stray-null lint over all docs | unit, meta | No `"schema": null`; empty ops omitted; `values/default-response-examples-no-output` passes since P2.1c (`MediaType.Schema` omitzero) |
| P2.1d | T: `TestSecuritySchemeMarshal` | unit | `deprecated`, `oauth2MetadataUrl`, `deviceAuthorization` |
| P2.3 | `ops/security-nil-scopes`, `ops/oauth-flow-without-scopes`; H01 `TestExtensionsRoundTripEverywhere`, H02 `TestExtensionKeysSorted`, H03 `TestSetExtensionValidation`, H04 `TestTypedExtensionWins`, G21 `TestJSONv1Parity`; H12 `TestOAuthFlowYAML` (relabelled from P2.5: passes with the P2.3 shapes); T: `TestExtensionsInlineSorted`, `TestV1EncodingJSONParity`, `TestRejectReservedExtension` | frag, unit | `[]` for nil scopes; `scopes:{}`; every 3.2 extensible object keeps `x-*` through a round trip; keys sorted; `x-oai-`/`x-oas-` rejected; v1 == v2 bytes |
| P2.5 | `ops/yaml-json-parity-hidden-field`, `values/nan-example-recorded`; G14–G17, G19; A11 `TestYAMLDepSwap` | http, unit, imp | decode(YAML) deep-equals decode(JSON); ambiguous strings stay strings; user `MarshalYAML` ignored; NaN → `Errors()` + 200; no `gopkg.in/yaml.v2` in deps |
| P2.6 | `sample/fixed-by`, `TestVersionPattern`, `TestAllCaseDocsValid` (un-skip); K01–K03; G22 `TestVersionConstant` | frag, meta | Every non-`FixedBy` case validates; `openapi: "3.2.1"`; `TypeAny` → `{}`; no `allowEmptyValue`; `explode` absent unless set |
| P3.1a | 5 cases (`fields/dominance-ambiguous-dropped`, `json-dash-comma-named-dash`, `json-string-option-on-int`, `embedded-cycle-response`, `embedded-non-struct`); K06 `TestDifferentialJSONKeys` | frag, unit | The key set of `json.Marshal(sample)` == schema `properties` keys for every kitchen-sink sample type |
| P3.1b | `fields/tagged-embedded-struct-request`, `embedded-cycle-request`, `json-dash-no-empty-property-request`, `sample/crashes` | frag | No crash, no `""` property |
| P3.2 | `types/elem/datatype`, `elem/override-datatype`, `cycle/slice-of-self`, `cycle/map-of-self` | frag | Crash cases un-`Crashes`ed and passing; `$ref` promotion is deterministic (G-DET) |
| P3.3 | 20 cases (19 `types/*` + `params/query-duration-default-ns-integer`) | frag | As written |
| P3.5 | `types/ptr-receiver/nullable`, `fields/nullable-ptr-{struct,slice,map}`; F34 `TestPointersAreNullableOption` (needs P6.1) | frag | `anyOf:[{$ref},{type:null}]`; `[T,"null"]`; option false → no null |
| P3.4 | `fields/shared-component-*` (3), `ops/ref-with-description-sibling` (after K-fix2); F33 `TestComponentNeverMutatedByUseSite`; F35 `TestAnnotationStyleAllOfWrap` (needs P6.1) | frag | Component bytes independent of use sites and registration order |
| P3.6 | 6 cases (`params/default-with-required-drops-required`, `values/default-pointer-field`, `enum-trimmed`, `invalid-bool-default`, `float32-default-literal`, `example-tag-to-examples`); K04 | frag | D15/D16/D21 rules; `"1s"` on a param → error, no panic |
| P4.1a | T: vtag parse table (≥20 rows, §3 K note) | unit | AST equality per row |
| P4.1b | 8 `validate/*` cases; K07–K16; T: ~30-row mapping table | frag, unit | Keyword mapping per PLAN P4.1b |
| P4.1c | K17–K19 | frag | `omitempty,required` → required; `required_if`, `dive,required` → not required |
| P4.2 | 5 `params/query-*` cases; G01 rows r1 | frag, unit | `-`/`""` skipped; `,required`; `,default=`; `omitempty` → `Errors()`, no panic; `explode:false` |
| P4.2c | `params/cookie-placeholder`; K31; G01 row r10; F12 | frag | `in: cookie`; non-scalar → error |
| P4.3 | `params/gin-wildcard-path`, `path-param-without-input`, `path-param-missing-from-input`; K39, K40; G02 (D10 warning) | frag | `{name}`; synthesized required string; a warning, not an error |
| P4.4 | `ops/response-descriptions`, `ops/response-empty-examples`, `values/empty-examples-map`; K20; D15 `TestWithDefaultMediaType`; T: response-header fields test | frag, unit | Descriptions per rule; no empty media types; custom media type used everywhere |
| P4.5 | `ops/generic-component-names`, `same-name-packages`, `typer-name-collision`; E01–E07 | frag, unit | Name format (§5 O2); first wins plus `CollisionError`; all names match the regex |
| P4.6 | `ops/body-name-collision-op-ids`, `body-name-collision-go-type` (K-fix1), `params/delete-body-documented` (K-fix3); K21–K24; E08 | frag | `<OpID>Input`; never overwrites; direct `$ref` for a single struct; `required` inferred; DELETE body documented; GET/HEAD not |
| P4.9 | `ops/handle-shared-handlers-slice` | frag | Both routes documented |
| P4.8 | `params/conflicting-locations`, `ops/response-code-clash-default-status`, `ops/orref-both-nil`; K29, K30; G01–G05; T: `TestFailedAddOperationLeavesNoTrace` | frag, unit, http | No panics; gin route still registered; `Err()` = join of non-warnings; failed op leaves no ID/PathItem |
| P4.7 | `ops/default-id-generic-handler`, `default-id-method-value`; K25–K28; E11–E13 | frag, unit | Default IDs per rule; collision fallback (§5 O4); explicit collision → error |
| P5.1 | `ops/operation-from-context-snapshot`, `TestConcurrentServeAndRegister` (un-skip); G06–G13, F29; T: `TestCloneRoundTrip` | race, http, unit | `-race` clean; snapshot is a deep clone; context-injected (no `c.Set`); cache invalidated by mutation; `http.Handler` on `net/http` |
| P6.1 | F11, F12, F14, A09, F34, F35, D15, E03, E13, B29 (option exists) | unit, frag | `TagNames` honored; BinderTags not configurable; renamed binder tag rejected |
| P6.2 | F01–F10 | frag | One case per tag (title, pattern, multipleOf, readOnly/writeOnly, nullable, additionalProperties, inline, style, allowReserved) + use-site only |
| P6.3 | F15–F22 | frag, unit | Provider replaces; Customizer/Enumer at component stage; field tag wins at use site (D7); value and pointer receivers |
| P6.5 | `ops/group-tag-empty-description-overwrite`, `group-tag-conflicting-description`; H05–H11 | frag, unit | Manual components via the registry; setters emit; Group description rules |
| P6.4 | F23–F33, A07, G20 | unit, imp | §1 precedence table green; contexts carry no gin/tonic; DocumentHook on a clone, cached per generation |
| P6.6a | D01–D04 | frag | `RequestBody` overrides `required`; Content replaces (§5 O17); encoding emitted |
| P6.6b | D05–D08 | frag | Merge by (name,in), non-zero fields win, last writer wins, otherwise append |
| P6.6c | D09–D14, D16 (`TestXCodeSampleTypedWins`) | frag | One case per option; `CustomizeOperation` deferred (§5 O18) |
| P7.1 | `params/deep-object-placeholder`; K32–K38; G01 rows r11–r12 | frag | `style: deepObject`, `explode: true`, object schema; generation errors per PLAN P7.1 |
| P7.2a | B31–B34 | unit, race | `PlanFor` cached per type; `Query` table; encoded brackets; nested → error; nil map |
| P7.2b | B01–B16, B25, B26, B28–B30 | http, race | 200 bound / 400 cases; validator runs after the hook; composes with a user hook; mismatch warning; one-time replaced-hook warning |
| P7.2c | B19–B24, B35 | http, unit | present / default / required 400 / bad type 400 |
| P8.1 | C01–C14, C16 | html, http, doc | Default Swagger UI; explicit choice honored; SRI + crossorigin + referrer policy; `validatorUrl: null`; escaping; undocumented route; plain `net/http` mount |
| P8.2 | C20 `TestEmbeddedSRI`, C21 | unit | sha384(file) == constants; `go:generate` directive present. Running `go generate` needs the network: **untestable in CI** |
| P8.3 | C17–C19, A06 | html, http | No external URLs; assets served with matching hash; `withDefaultFonts:false`; same module |
| P8.4 | I15 `TestDocsCompatTable` | doc | Pins == constants; AllOfWrap workaround documented. Real-browser rendering is **untestable in CI** (manual Haiku browser run) |
| P9.1 | I11 `TestMigrationCoversTopics`, I01–I09 (shared Examples) | doc | Every listed topic present; every "after" snippet is a compiling `Example*` |
| P9.2 | I10 `TestREADMESnippetsMatchExamples`, C16 | doc | Snippets == Example bodies; no `validator.v8`/`3.0.1` links; tags table complete; Swagger UI gaps + `ScalarUI` pointer present |
| P9.3 | I13 `TestMarketSpecValid`, I14 `TestMarketFilterEndpoint` | meta, http | Valid 3.2.1 + lints; deepObject filter works; cookie, Scalar route, `x-tagGroups` present |
| P9.4 | G-CI jobs (§4); I12 | CI | The CI config is self-verifying when it runs; an optional `actionlint` step. **Partly untestable as a Go test** |
| P9.5 | I12 `TestReleaseNotes` (FixedBy P9.5); G-STUB release check | doc | Notes mention Go 1.27 floor, Go ≤1.26 excluded, OAS 3.2.1, MIGRATION link, global bind hook, codegen caveats; no stub files, no `acceptance` tag. Tagging and goreleaser: **untestable (process)** |

### 1.2 Decisions

| D | Tests | Notes |
|---|---|---|
| D1 3.2.1 only | `TestVersionPattern`, G22, `sample/fixed-by`, meta gate | — |
| D2 go 1.27 floor | A04, G-CI matrix | `tip` non-blocking: CI config only (**untestable as a test**) |
| D3 router-agnostic | A01, A02, A07; (b) C14, G06; (c) G12, B16 | — |
| D4 same module path | A04 | — |
| D5 embedded subpackage | A06 | — |
| D6 pointers nullable | P3.5 cases, F05, F34 | — |
| D7 two-stage precedence | F19, F32, F33 | — |
| D8 `$ref` siblings / AllOfWrap | `ops/ref-with-description-sibling`, F35 | — |
| D9 gadgeto v0.9.0 | A04 | — |
| D10 missing path param | `params/path-param-*` (P4.3), G02 | Warning only: `Err()==nil` |
| D11 plain gin undocumented | K41 | Guard (passes now) |
| D12 json/v2 | G21, T: `TestV1EncodingJSONParity` | — |
| D13 DELETE body | `params/delete-body-documented`, B17, B18, K23, K24 | — |
| D14 Marshaler policy | `types/marshaler/*`, `types/map-key/*` | — |
| D15/D21 default+required | `params/default-with-required-drops-required`, `values/default-required-body`, G02 | Warning, not an error |
| D16 Duration defaults | `params/query-duration-default-ns-integer`, `values/duration-default-enum-body`, K04 | — |
| D17 default Swagger UI | C01, C16, I10 | Rendering quality: manual (P8.4) |
| D18 explicit choice honored | C02, C03, C04 | — |
| D19 pins | C05, C06, C20 | — |
| D20 auto-install bind hook | B25, B28, B29, B30 | — |
| §1 precedence | F32, F33, F28 | — |
| §2 gates | §4 | — |
| Risk 3, 5, 8, 9 | B28; D15; G16; G21 | Risks 4 and 10: **untestable/out of scope** (tonic global map; client float64) |

### 1.3 Untestable or partly testable requirements
1. ADR sign-off (P1.0), Opus golden-diff reviews (§2, P2.6), README/MIGRATION prose quality. These are process steps. Tests cover only presence and compilation.
2. Real-browser rendering (P0.6, P8.4). CI covers only the doc and constant consistency (C06, I15).
3. `go generate` asset refresh (P8.2) needs the network. Proxy: the offline C20.
4. CI and release mechanics (D2 `tip`, P9.4 coverage floor, P9.5 tag and goreleaser): they are verified by running the pipeline, not by a test.
5. "Marshal outside the lock" (P5.1): there is no direct observable. Proxy: G09, where a blocked DocumentHook must not block `AddOperation`.
6. §15 v1.0.0 criteria (time-based) and Risk 10.
7. **Gin duplicate-route panic.** It is gin's, not fizz's. "No panics" cannot cover it unless fizz pre-checks (§5 O32).

---

## 2. Compile strategy

**Recommendation: (c) now + (a′) after the P1.0 stub + (b) for changed shapes.** Option (a) as written does not work: P1.0 puts the stub in `internal/apistub`, a separate package, so a test that calls `fizz.WithTagNames` cannot compile against it. The fix is three rules.

1. **No new API needed → write now in the default build (c).** These tests use struct tags, existing options, HTTP through tonic handlers, the import graph, and served JSON/YAML. They cover:
   - deepObject and cookie runtime binding (B01–B28 minus tagged);
   - DELETE/GET body (B17, B18);
   - YAML parity (G14–G17);
   - the import rule (A01–A06);
   - new struct tags (F01–F10);
   - the no-panic table (G01);
   - context injection (G12, G13);
   - `API()` clone (G10);
   - the extensions round trip (H01, H02);
   - all K cases.

   Each wraps its assertions in `at.Pending(t, "Pn.m", fn)` (Batch 0). These run in CI from today, and every task flips them red→green.
2. **New symbols → `//go:build acceptance` (a′), stubs in the real packages.**
   - P1.0's stub sub-step is amended: stubs live in `zz_stub_acceptance.go` files in `./`, `openapi/`, `bind/`, `docsui/` and `docsui/embedded/`, all guarded by `//go:build acceptance`. They do not go in `internal/apistub`.
   - New packages also get an untagged `doc.go`, so A01 covers them from day one.
   - Stub bodies return no-ops or zero values: options return `func(*config){}`, handlers return `http.NotFoundHandler()`, and methods return `errors.New("stub: Pn.m")`. Tests therefore reach their assertions and skip as red, not crash. `at.Pending` also recovers panics.
   - Every stub declaration carries `// FixedBy: Pn.m`. When a task adds the real symbol (untagged), `-tags acceptance` fails with a redeclaration error. This forces the task to delete the stub and flip its tests.
   - The one source-compatible signature change that cannot be stubbed under a tag: **P1.0 lands `type Option` and variadic `New(opts ...Option)` / `NewFromEngine(e, opts ...Option)` for real**. Master is unreleased (§15.2), so this is safe.
   - P1.0 accept: `go vet -tags acceptance ./...` compiles.
   - At P9.4 the tag is removed from all files (G-STUB).
   - P1.0's dependencies (P0.4a, P0.6) are done, so the stub sub-step can start immediately.
3. **Changed shapes of existing types → task-owned (b).** This covers anything that reads or writes fields whose type changes: `Schema.Type` (TypeSet), numeric bounds (`json.Number`), `OrRef.Value/.Ref`, `Explode *bool`, `ResponseHeader.Required/Deprecated/Example`, the inline `Extensions` field. Such tests cannot be stubbed, so they stay the task's own red tests (`T:` in §1).
   - Acceptance tests avoid these fields entirely. They assert on serialized JSON. They build model values with `at.FromJSON[openapi.Parameter](`{…}`)` (unmarshal), which compiles against both shapes. They mutate only string fields present in both shapes (`Description`, `Summary`, `Title`, `Pattern`, `Format`, `Name`, `In`).
   - They touch new behaviour only through **functions and methods**, which can be stubbed. For example, they call `(*openapi.Operation).SetExtension` (§5 O24), never `.Extensions[...]`.

Why not (b) alone: the user asked for tests before the fixes, and (b) leaves P6–P8 features unspecified until mid-project. Why not (c) alone: hooks, interfaces, options, docsui and bind need API to test at all.

Global-state rules for writers:
- tonic's bind/error/render hooks, `gin.SetMode` and `gin.DefaultWriter` are process-global. Tests that touch them are never `t.Parallel()` and restore state in `t.Cleanup`.
- Bind-hook install order is per process, so:
  - `internal/acceptance` covers auto-install;
  - `internal/acceptance/hookorder` sets a user hook in `TestMain` **before** the first `fizz.New`;
  - `internal/acceptance/nohook` never auto-installs.

---

## 3. New test inventory

Identifier rule in `package acceptance`: every declared type, func or var starts with the batch letter, e.g. `bDeepIn`, `cPage`. This prevents clashes between batch files, as the O/P/T prefixes do in the kitchen sink. Component names then read `AcceptanceBDeepIn`.

### Batch 0: harness (serial, lands first). Files: `internal/kitchensink/{case.go,assert.go}`, `internal/plan/done.go`, `internal/acceptance/at/at.go`, `internal/acceptance/at/at_test.go`

Harness changes:
- `Case.New func() *fizz.Fizz` (optional constructor, for options).
- `Case.Warnings []string`: matches only entries that implement `interface{ IsWarning() bool }`. This compiles today; see §5 O6.
- Export `kitchensink.AssertFragments(tb, doc, []Fragment)` and `kitchensink.BuildSpec(f)`.
- `Run` applies the meta gate to every passing case once `plan.Done["P2.6"]` is true.
- `internal/plan.Done map[string]bool` is the **done ledger**. Each task PR adds its own ID.
- `at.Pending(t, fixedBy, func(tb testing.TB))`. It uses a recorder TB in which `Fatalf` aborts through a recovered sentinel panic, and it also recovers user panics.
  - Failing → `t.Skip("fixed by X: <first failure>")`.
  - Passing with `FixedBy` → skip "(passes now: drop FixedBy)", or `t.Error` when `FIZZ_STALE=fail`.
  - `FixedBy ∈ plan.Done` and still failing → `t.Error` ("task done, test red").
- Helpers:
  - `at.Do(h, method, target, ...ReqOpt)`, with `WithBody`, `WithCookie`, `WithHeader`;
  - `at.Spec(t, f) []byte`;
  - `at.Frag` / `at.Absent`;
  - `at.HTML(body) []Elem`, built on `golang.org/x/net/html` (already in the module graph at v0.60.0; add it as a direct require);
  - `at.YAMLToAny`;
  - `at.FromJSON[T]`.
- Convert the three raw `t.Skip("fixed by")` tests to `at.Pending`.

| ID | Test | Asserts | FixedBy |
|---|---|---|---|
| 0.1 | `TestPendingSkipsFailing` | A failing fn with FixedBy → skipped, message has "fixed by X" | — |
| 0.2 | `TestPendingStale` | A passing fn with FixedBy → skip text "(passes now"; with `FIZZ_STALE=fail` → error | — |
| 0.3 | `TestPendingDoneLedger` | A failing fn with FixedBy in Done → error | — |
| 0.4 | `TestPendingRecoversPanic` | Panicking fn → recorded failure; process alive | — |
| 0.5 | `TestCaseCustomNew` | `Case.New` is used instead of `fizz.New()` | — |
| 0.6 | `TestCaseWarnings` | Warnings matched only against `IsWarning()` entries | — |

### Batch A: architecture. Files: `internal/archtest/imports_test.go`, `internal/archtest/repo_test.go`, `internal/acceptance/arch_test.go` (tag)

| ID | Test | Asserts (input → expected) | FixedBy | Build |
|---|---|---|---|---|
| A01 | `TestRouterAgnostic` | `go list -deps -test` over `./openapi/... ./bind/... ./docsui/...` (patterns with no packages are skipped) → no import path with prefix `github.com/gin-gonic/gin` or `github.com/loopfz/gadgeto`. The failure prints the importer chain | P1.3 | now |
| A02 | `TestRouterAgnosticSelfCheck` | The same checker on `.` (root) **does** report gin, so the rule is not vacuous | — | now |
| A03 | `TestNoUUIDDep` | `go list -deps ./openapi` (non-test) has no `github.com/gofrs/uuid` (subtest `gofrs`) or `github.com/google/uuid` (subtest `google`, reached via tonic until P1.3) | P1.2 / P1.3 | now |
| A04 | `TestModuleInvariants` | go.mod: `module github.com/wI2L/fizz`, `go 1.27`, `github.com/loopfz/gadgeto v0.9.0` | — | now |
| A05 | `TestToolsModuleIsolated` | `go list -m all` has no `honnef.co/go/tools` or `golang.org/x/vuln` | — | now |
| A06 | `TestDocsUIEmbeddedSameModule` | `docsui/embedded` exists; no `go.mod` under `docsui/`; `go list -f {{.Module.Path}} ./docsui/embedded` == main module | P8.3 | now |
| A07 | `TestNoGinTypesInPublicSignatures` | Reflection walk (fields, func params/results, recursively) over `fizz.OperationOption`, `openapi.OperationInfo`, `openapi.Hooks` and every `*Context` hook type → no `PkgPath` under gin or gadgeto | P6.4 | tag |
| A08 | `TestBinderTagsMatchTonic` | `openapi.BinderTags` values == `tonic.QueryTag, PathTag, HeaderTag, EnumTag, DefaultTag, ValidationTag, ExplodeTag` | P1.3 | tag |
| A09 | `TestBinderTagsNotConfigurable` | `reflect.TypeFor[openapi.TagNames]()` has exactly the 15 fields of P6.1 and no `Path, Query, Header, Enum, Default, Validate, Explode` | P6.1 | tag |
| A10 | `TestYAMLDepSwap` | `go list -deps ./...` (non-test) has no `gopkg.in/yaml.v2`; go.mod gin ≥ v1.12.0 | P2.5 | now |

### Batch B: runtime binding. Files: `internal/acceptance/bind_http_test.go` (now), `bind_api_test.go` (tag), `internal/acceptance/hookorder/hookorder_test.go` (now), `internal/acceptance/nohook/nohook_test.go` (tag)

Fixtures. All handlers are `bEcho[T]`, which returns `in` as the 200 JSON body. Default tonic error hook → 400 `{"error": …}`.

```go
type bMapIn    struct{ Filter map[string]string   `query:"filter" style:"deepObject"` }
type bReqIn    struct{ Filter map[string]string   `query:"filter" style:"deepObject" validate:"required"` }
type bIntIn    struct{ Filter map[string]int      `query:"filter" style:"deepObject"` }
type bSliceIn  struct{ Filter map[string][]string `query:"filter" style:"deepObject"` }
type bS        struct{ Name string `json:"name"`; Age int `json:"age"`; Tags []string `json:"tags"` }
type bStructIn struct{ Filter *bS                 `query:"filter" style:"deepObject"` }
type bDiveIn   struct{ Filter map[string]string   `query:"filter" style:"deepObject" validate:"dive,keys,min=2,endkeys,max=3"` }
type bMixIn    struct{ Page int `query:"page"`; Filter map[string]string `query:"filter" style:"deepObject"` }
type bBodyIn   struct{ Name string `json:"name"`; Filter map[string]string `query:"filter" style:"deepObject"` }
type bCookieIn struct{ Session string `cookie:"session"`; Lang string `cookie:"lang" default:"en"`; N int `cookie:"n"` }
type bCookieReq struct{ Token string `cookie:"token" validate:"required"` }
```

| ID | Test | Asserts | FixedBy | Build |
|---|---|---|---|---|
| B01 | `TestDeepObjectMapBinds` | `GET ?filter[name]=x&filter[age]=3` → 200 `{"Filter":{"name":"x","age":"3"}}` | P7.2b | now |
| B02 | `TestDeepObjectDottedKeys` | `?filter[a.b]=x&filter[attr.sss]=v` → `{"a.b":"x","attr.sss":"v"}` | P7.2b | now |
| B03 | `TestDeepObjectEncodedBrackets` | `?filter%5Bname%5D=x` → `{"name":"x"}` | P7.2b | now |
| B04 | `TestDeepObjectNestedKey400` | `?filter[a][b]=x` → 400; the body `error` contains `filter[a][b]` (negative) | P7.2b | now |
| B05 | `TestDeepObjectNoKeysNilMap` | No query → 200 `"Filter":null` (nil, not `{}`). Guard: passes today and must keep passing | — | now |
| B06 | `TestDeepObjectValidateRequired` | `bReqIn`, no keys → 400; `?filter[k]=v` → 200 | P7.2b | now |
| B07 | `TestDeepObjectValueConversion` | `bIntIn` `filter[a]=3` → `{"a":3}`; `filter[a]=x` → 400 mentioning `filter[a]` | P7.2b | now |
| B08 | `TestDeepObjectRepeatedKeys` | `bSliceIn` `filter[t]=a&filter[t]=b` → `{"t":["a","b"]}`; `bMapIn` with a repeated key → 400 (§5 O14) | P7.2b | now |
| B09 | `TestDeepObjectStructTarget` | `filter[name]=x&filter[age]=3&filter[tags]=a&filter[zzz]=1` → `{"name":"x","age":3,"tags":["a"]}` (unknown ignored); `filter[age]=x` → 400; no keys → `null`; `filter[NAME]=x` → ignored (exact json name, §5 O14) | P7.2b | now |
| B10 | `TestDeepObjectDiveRunsAfterBind` | `bDiveIn` `filter[a]=x` (key length 1) → 400; `filter[ab]=x` → 200. Proves handler.go:97 validation runs after the hook | P7.2b | now |
| B11 | `TestDeepObjectMalformedKeys` | Table, one Pending per row: `filter[]=x` → 400; `filter[a=x` → 400; `filter=x` (bare) → 400; `filterx[a]=1` → 200 ignored | P7.2b | now |
| B12 | `TestDeepObjectWithRegularParams` | `bMixIn` `?page=2&filter[a]=b` → `{"Page":2,"Filter":{"a":"b"}}` | P7.2b | now |
| B13 | `TestDeepObjectWithJSONBody` | `POST bBodyIn` body `{"name":"n"}` + `?filter[a]=b` → both bound (the hook calls `next` first) | P7.2b | now |
| B14 | `TestDeepObjectErrorIsBindError` | Custom error hook (restored) records `_, ok := err.(tonic.BindError)` → true for `filter[a][b]` | P7.2b | now |
| B15 | `TestDeepObjectConcurrentRequests` | 32 goroutines, distinct `filter[k]=i` → each response echoes its own i; `-race` clean | P7.2b | now |
| B16 | `TestInputModelMismatchSkipsPlan` | Route with handler input `bMixIn` + `fizz.InputModel(&bMapIn{})` → `Errors()` has a warning containing `InputModel` and the op ID; `?filter[a]=b` → 200 with `Filter:null` (plan skipped) | P7.2b | now |
| B17 | `TestDeleteBodyBoundByTonic` | `DELETE /i/1` body `{"name":"x"}` → handler sees `"x"` (D13 rationale guard) | — | now |
| B18 | `TestGetBodyNotBoundByTonic` | `GET` with body `{"name":"x"}` → `""` (D13 guard) | — | now |
| B19 | `TestCookieBinds` | `Cookie: session=abc; n=5` → `{"Session":"abc","N":5,...}` | P7.2c | now |
| B20 | `TestCookieDefaultWhenAbsent` | No `lang` → `"en"`; `lang=fr` → `"fr"` | P7.2c | now |
| B21 | `TestCookieRequired400` | `bCookieReq` with no cookie → 400; `token=t` → 200 | P7.2c | now |
| B22 | `TestCookieBadType400` | `n=abc` → 400, error mentions `n` | P7.2c | now |
| B23 | `TestCookieEmptyVsAbsent` | `lang=` (present, empty) → `""`, default **not** applied (§5 O15) | P7.2c | now |
| B24 | `TestCookieRawValue` | `session=a%20b` → `"a%20b"` (no decoding, net/http semantics) | P7.2c | now |
| B25 | `TestUserHookComposed` (hookorder) | `TestMain` sets a counting hook that wraps `tonic.DefaultBindingHook`, then calls `fizz.New()` 3×. One request with `filter[a]=b` → count == 1 **and** filter bound | P7.2b | now |
| B26 | `TestUserHookErrorShortCircuits` (hookorder) | The user hook returns `errors.New("nope")` → 400 containing `nope`. Guard | — | now |
| B28 | `TestReplacedHookWarnsOnce` (hookorder, after B25) | `tonic.SetBindHook(tonic.DefaultBindingHook)` after New; gin debug mode, `gin.DefaultWriter` = buffer; 2 requests → warning substring (`bind hook`, op ID) appears exactly once; filter `null` | P7.2b | now |
| B29 | `TestWithoutBindHook` (nohook) | `fizz.New(fizz.WithoutBindHook())` → `reflect.ValueOf(tonic.GetBindHook()).Pointer()` == that of `tonic.DefaultBindingHook`; filter not bound | P7.2b | tag |
| B30 | `TestInstallBindHookExplicit` (nohook, after B29) | `fizz.InstallBindHook()` → the next request binds the filter; a second call does not double-wrap (counting hook == 1) | P7.2b | tag |
| B31 | `TestPlanForCachedAndConcurrent` | `bind.PlanFor(T, tags)` twice → same pointer; 50 goroutines → same pointer; `-race` | P7.2a | tag |
| B32 | `TestPlanForRejectsUnsupported` | deepObject value `map[string]struct{X int}` → error; `map[string]map[string]string` → error | P7.2a | tag |
| B33 | `TestQueryDecodeTable` | `bind.Query(httptest.NewRequest(...), &dst, plan)` on plain `*http.Request` (no gin): rows from B01–B11 | P7.2a | tag |
| B34 | `TestQueryMaxKeys` | `bind.WithMaxKeys(2)`, 3 keys → error `errors.Is(err, bind.ErrTooManyKeys)` | P7.2a | tag |
| B35 | `TestCookiesDecodeTable` | `bind.Cookies(r, &dst, plan)` rows from B19–B24 | P7.2c | tag |

(There is no B27; its assertion is merged into B25.)

### Batch C: docs UI. Files: `internal/acceptance/docsui_test.go`, `internal/acceptance/embedded_test.go` (tag)

All HTML checks parse with `at.HTML`. Constants: `docsui.SwaggerUIJS/CSS`, `docsui.SwaggerUIJSIntegrity/...`, `docsui.ScalarJS(+Integrity)`. API per §5 O1.

| ID | Test | Asserts | FixedBy |
|---|---|---|---|
| C01 | `TestDocsDefaultIsSwaggerUI` | `f.Docs("/docs", "/openapi.json")` → GET 200, `Content-Type: text/html; charset=utf-8`; exactly one `<script src>` == Swagger JS const; `<link rel=stylesheet href>` == CSS const; body contains no `scalar` (case-insensitive) | P8.1 |
| C02 | `TestDocsExplicitSwaggerNeverScalar` | `fizz.SwaggerUI()`, also with `WithTitle`, `WithUIConfig`, `WithAssetURLs` → no `scalar` substring anywhere (negative) | P8.1 |
| C03 | `TestDocsExplicitScalarNeverSwagger` | `fizz.ScalarUI()` → script == Scalar const; no `swagger-ui` substring | P8.1 |
| C04 | `TestDocsRendererConflict` | `f.Docs(p, s, fizz.SwaggerUI(), fizz.ScalarUI())` → `Errors()` has an error naming both; Swagger UI (first) served; no panic | P8.1 |
| C05 | `TestSRIAttributesMatchConstants` | Every external script/stylesheet has `integrity` == its const (`^sha384-[A-Za-z0-9+/]{64}$`), `crossorigin="anonymous"`, `referrerpolicy="no-referrer"` | P8.1 |
| C06 | `TestSRIConstantsMatchCompatDoc` | The docs/compat.md pin table (package@version, file, sha384) == docsui constants and URLs | P8.1 |
| C07 | `TestSwaggerValidatorURLNull` | The config JSON (`<script type="application/json" id="fizz-docs-config">`, §5 O27) has `"validatorUrl": null`; `WithUIConfig({"validatorUrl":"https://validator.swagger.io"})` → still `null` | P8.1 |
| C08 | `TestDocsUIConfigMerged` | `WithUIConfig({"deepLinking":true,"x":"</script><b>"})` → `deepLinking:true` present; the number of `</script>` tags is unchanged (escaped) | P8.1 |
| C09 | `TestDocsTitleEscaped` | `WithTitle("A & <b>")` → `<title>A &amp; &lt;b&gt;</title>` | P8.1 |
| C10 | `TestDocsSpecURLEscaped` | specURL `/o.json?x="y'</script>` appears only JSON-escaped inside the config | P8.1 |
| C11 | `TestWithAssetURLsSRI` | `WithAssetURLs(js, css)` → no `integrity` attribute; with integrity values → exactly those | P8.1 |
| C12 | `TestCSPNonce` | `WithCSPNonce("n0nce")` → every `<script>` has `nonce="n0nce"`; no `on*=` attributes | P8.1 |
| C13 | `TestDocsRouteUndocumented` | After `f.Docs`, spec has no `/docs` path; `Errors()` empty | P8.1 |
| C14 | `TestDocsHandlersOnNetHTTP` | `http.NewServeMux().Handle("/d", docsui.SwaggerUI("/s.json"))` → 200 html (rule b, no gin) | P8.1 |
| C16 | `TestDocsGodoc` | `go/doc` on the root package: the `Docs` doc contains `Swagger UI` and `default`; the `SwaggerUI` doc contains `ScalarUI` and `itemSchema` (gap #11065) | P8.1 |
| C17 | `TestEmbeddedScalarNoFonts` | `embedded.Scalar` config has `"withDefaultFonts":false` | P8.3 |
| C18 | `TestEmbeddedNoExternalURLs` | Both embedded pages: no `src`/`href` starting with `http://`, `https://` or `//` | P8.3 |
| C19 | `TestEmbeddedAssetsServed` | `GET /docs/assets/swagger-ui-bundle.js` → 200, `text/javascript; charset=utf-8`, sha384(body) == const; `/docs/assets/nope.js` → 404; `/docs/assets/../embedded.go` → not 200 | P8.3 |
| C20 | `TestEmbeddedSRI` | Offline: sha384 of each file in `embedded/assets` == constants | P8.2 |
| C21 | `TestPinAssetsDirective` | `docsui/embedded/gen.go` contains `//go:generate go run ../internal/pinassets` | P8.2 |

(There is no C15.)

### Batch D: operation options. File: `internal/acceptance/options_test.go` (tag)

| ID | Test | Asserts | FixedBy |
|---|---|---|---|
| D01 | `TestRequestBodyOverridesRequired` | Body without required fields + `RequestBody("Create", true)` → `/requestBody` `{"description":"Create","required":true,…}`; body with a required field + `RequestBody("", false)` → `required` absent | P6.6a |
| D02 | `TestRequestBodyContentReplaces` | `RequestBody("", true, Content("application/xml", dPet{}))` → content keys == `["application/xml"]`, schema `$ref` to `AcceptanceDPet` | P6.6a |
| D03 | `TestRequestBodyEncoding` | `Content("multipart/form-data", dUpload{}, Encoding("file", "image/png"))` → `/requestBody/content/multipart~1form-data/encoding/file/contentType` == `"image/png"` | P6.6a |
| D04 | `TestRequestBodyOnGETIsError` | `RequestBody` on GET → error naming the op; no `requestBody` emitted | P6.6a |
| D05 | `TestParameterMergeByNameIn` | Input `Q string query:"q"` + `Parameter(at.FromJSON[openapi.Parameter](`{"name":"q","in":"query","description":"m","deprecated":true}`))` → param `{"name":"q","in":"query","description":"m","deprecated":true,"schema":{"type":"string"}}`. The schema survives because zero fields do not erase | P6.6b |
| D06 | `TestParameterAppendsWhenNew` | `{"name":"X-Trace","in":"header","schema":{"type":"string"}}` → 2 params | P6.6b |
| D07 | `TestParameterMergeLastWriterWins` | Two merges on (q,query) with descriptions `a`, then `b` → `b` | P6.6b |
| D08 | `TestParameterMergeKeyIncludesIn` | (q, header) → appended, not merged into (q, query) | P6.6b |
| D09 | `TestTagsOption` | Group tag `pets` + `Tags("admin","pets")` → `["pets","admin"]` (append, dedupe, order kept) | P6.6c |
| D10 | `TestExternalDocsOption` | `ExternalDocs("https://x","More")` → `/externalDocs` == `{"url":"https://x","description":"More"}` | P6.6c |
| D11 | `TestExtensionOption` | `Extension("x-rate",10)` → op `"x-rate":10`; `Extension("rate",1)` and `Extension("x-oai-a",1)` → errors (`errors.Is(err, openapi.ErrInvalidExtensionKey)` / `ErrReservedExtension`), not emitted | P6.6c |
| D12 | `TestCustomizeOperationDeferred` | `CustomizeOperation(fn)` listed **before** `Summary("s")`: fn sees `Summary=="s"`; its `Summary="c"` wins; fn error → error with op ID, op still documented | P6.6c |
| D13 | `TestResponseWith` | `ResponseWith("404","Not found", WithModel(dErr{}), WithExamples(map{"a":…}))` → `/responses/404` description, `$ref` `AcceptanceDErr`, `examples/a/value` | P6.6c |
| D14 | `TestResponsePositionalKept` | `fizz.Response("404","nf",nil,nil,nil)` still emits `{"description":"nf"}` (regression) | — |
| D15 | `TestWithDefaultMediaType` | `New(WithDefaultMediaType("application/vnd.api+json"))` → response and requestBody content keys == that type | P4.4 |
| D16 | `TestXCodeSampleTypedWins` | `XCodeSample(...)` + `Extension("x-codeSamples", other)` → the typed value is emitted; error recorded | P6.6c |

### Batch E: naming and IDs. Files: `internal/acceptance/naming_test.go`, fixtures `internal/acceptance/fixtures/naming/{x,y}/models.go` (two packages, both named `models`)

| ID | Test | Asserts | FixedBy | Build |
|---|---|---|---|---|
| E01 | `TestDefaultComponentNameFormat` | `eFoo` → `AcceptanceEFoo`; `fixtures.Box[string]` → `FixturesBox_string`; `Box[models.Item]` → `FixturesBox_models.Item`; `Box[[]string]` → `FixturesBox_List_string`; `Box[map[string]int]` → `FixturesBox_Map_string_int`; `Box[*models.Item]` → `FixturesBox_models.Item` (§5 O2). All match `^[A-Za-z0-9._-]+$` | P4.5 | now |
| E02 | `TestComponentNamesSanitized` | Every component name of a doc with generics, Typer and unicode type names matches the regex (`LintComponentNames`) | P4.5 | now |
| E03 | `TestSchemaNamerOption` | `WithSchemaNamer(func(t) string {return "X_"+t.Name()})` → `$ref` `#/components/schemas/X_EFoo` | P4.5 | tag |
| E04 | `TestSchemaNamerInvalidSanitized` | Namer returns `a b/c` → `a_b_c` + warning | P4.5 | tag |
| E05 | `TestCollisionFirstWins` | `x.models.Item` then `y.models.Item` → `ModelsItem` == the x schema; `errors.As(err, &*openapi.CollisionError)` with `Name=="ModelsItem"`, `Kept==x`, `Rejected==y`; the y use site is **inlined** (has `sku`), not a dangling `$ref` (§5 O3) | P4.5 | tag |
| E06 | `TestCollisionIsHardError` | The collision makes `f.Err()` non-nil | P4.8 | tag |
| E07 | `TestOverrideTypeNameCollision` | After `eA` is registered as `AcceptanceEA`, `Generator().OverrideTypeName(TypeFor[eB](), "AcceptanceEA")` → non-nil error; component unchanged | P4.5 | now |
| E08 | `TestRequestBodyNamer` | `WithRequestBodyNamer(func(opID string, t reflect.Type) string {return opID+"Body"})` → `#/components/schemas/createBody` | P4.6 | tag |
| E11 | `TestDefaultOperationIDs` | Table: func `eListPets` → `eListPets`; method value `(*eSvc).List` → `List`; generic `eGen[int]` → `eGen`; closure inside `eRegister` → `eRegister`; `x.Get` and `y.Get` → `Get` and `y.Get` (qualifier = last import-path segment: a function's declared package name is not available at run time); third clash `GET /a/{id}` → `get_a_id` (§5 O4) | P4.7 | now |
| E12 | `TestExplicitIDCollision` | `fizz.ID("dup")` on two routes → error containing `dup`; the second op is absent from the spec; the second gin route still answers 200 | P4.7 | now |
| E13 | `TestOperationIDNamer` | `WithOperationIDNamer(func(m, p, h string) string)` → used for default IDs; not used when `fizz.ID` is set | P4.7 | tag |

(There are no E09/E10; they moved to K21/K22.)

### Batch F: tags, interfaces, hooks, precedence. Files: `internal/acceptance/tags_test.go` (now), `interfaces_test.go`, `hooks_test.go`, `precedence_test.go` (tag)

| ID | Test | Asserts | FixedBy | Build |
|---|---|---|---|---|
| F01 | `TestTagTitle` | `title:"Pet name"` → property `title` | P6.2 | now |
| F02 | `TestTagPattern` | `pattern:"^[a-z]+$"` on a string → `pattern` | P6.2 | now |
| F03 | `TestTagMultipleOf` | `multipleOf:"0.01"` → `"multipleOf":0.01` (raw bytes `0.01`); `"0"` and `"x"` → errors, keyword omitted | P6.2 | now |
| F04 | `TestTagReadWriteOnly` | `readOnly:"true"` → `readOnly:true`; `writeOnly:"true"`; `readOnly:"maybe"` → error; both true → warning | P6.2 | now |
| F05 | `TestTagNullable` | `*string` + `nullable:"false"` → `{"type":"string"}`; `string` + `nullable:"true"` → `["string","null"]`; `*fStruct` + `nullable:"false"` → plain `$ref` | P6.2 | now |
| F06 | `TestTagAdditionalPropertiesFalse` | Anonymous struct field → `additionalProperties:false`; named-struct (`$ref`) field → sibling `unevaluatedProperties:false` (§5 O11); `int` field → error | P6.2 | now |
| F07 | `TestTagInline` | `fAddr` field with `inline:"true"` → inline `properties`, no `$ref`; `AcceptanceFAddr` still exists when also used elsewhere | P6.2 | now |
| F08 | `TestParamTagStyle` | `query:"ids" style:"pipeDelimited"` on `[]string` → `style:"pipeDelimited"`; `style:"matrix"` on query → error, no style | P6.2 | now |
| F09 | `TestParamTagAllowReserved` | `query:"u" allowReserved:"true"` → `allowReserved:true`; on a header → error | P6.2 | now |
| F10 | `TestTagsUseSiteOnly` | Two fields of `fAddr`, one with `title:"T"` → component `AcceptanceFAddr` has no `title` | P6.2 | now |
| F11 | `TestTagNamesOverride` | `WithTagNames(TagNames{Description:"doc"})` → `doc:"x"` gives description x; `description:"y"` ignored; `example` (zero field) still works (§5 O23) | P6.1 | tag |
| F12 | `TestTagNamesCookie` | `TagNames{Cookie:"ck"}` + `ck:"sid"` → `in: cookie` | P4.2c | tag |
| F14 | `TestRenamedBinderTagRejected` | `openapi.NewGenerator(&SpecGenConfig{QueryLocationTag:"q"})` → error naming `QueryLocationTag`; all-tonic values → ok (§5 O22) | P6.1 | now |
| F15 | `TestSchemaProviderReplaces` | `fMoney` (struct) `OpenAPISchema()` from `{"type":"string","pattern":"^\\d+\\.\\d{2}$"}` → component == exactly that (no `properties`); the pointer-receiver variant `fMoneyP` is detected too | P6.3 | tag |
| F16 | `TestSchemaProviderNilFallsBack` | Returns nil → reflected schema, no error | P6.3 | tag |
| F17 | `TestSchemaCustomizer` | Sets `Description="c"` → component description; returning an error → error with the type name, component without the customizer's changes | P6.3 | tag |
| F18 | `TestEnumerComponent` | `type fColor string` with `Enum() []any{"red","green"}` → component `AcceptanceFColor` == `{"type":"string","enum":["red","green"]}`; field → `$ref` (§5 O10) | P6.3 | tag |
| F19 | `TestEnumerVsFieldTag` | Field `enum:"red"` on `fColor` → use site `{"$ref":…,"enum":["red"]}`; component enum unchanged | P6.3 | tag |
| F20 | `TestInterfacesPointerReceivers` | All three interfaces declared on `*T` are honored | P6.3 | tag |
| F21 | `TestLegacyInterfacesKept` | `Typer`, `DataType`, `Nullable`, `Exampler` still honored (regression) | — | now |
| F22 | `TestEnumerTypeMismatch` | `Enum()` returns `1` for a string type → error, enum omitted | P6.3 | tag |
| F23 | `TestSchemaHookContext` | `ctx.Type == TypeFor[fPet]`, `ctx.Name == "AcceptanceFPet"`; called once per component even with 3 use sites | P6.4 | tag |
| F24 | `TestFieldHookSkip` | Skip on `Secret` → property absent and not in `required` | P6.4 | tag |
| F25 | `TestFieldHookContext` | `ctx.StructField.Name`, `ctx.Parent`; setting a description on a `$ref` field leaves the component untouched | P6.4 | tag |
| F26 | `TestParameterHookContext` | `ctx.OperationID=="getPet"`, `ctx.Method=="GET"`, `ctx.Path=="/pets/{id}"` (not `:id`), `ctx.In=="path"` | P6.4 | tag |
| F27 | `TestOperationHook` | Sets the summary (wins over `fizz.Summary`); an error → recorded with op ID, op kept | P6.4 | tag |
| F28 | `TestDocumentHookOnClone` | Adds `x-tagGroups` → served JSON has it; `Generator().API()` does not (§5 O8); appending a tag in the hook twice across generations → it appears once | P6.4 | tag |
| F29 | `TestDocumentHookCachedPerGeneration` | 3× JSON + 1× YAML GET → hook count 1; register a route → next GET → 2 | P6.4 | tag |
| F30 | `TestDocumentHookError` | Hook error → spec handler 500, body does not contain `"openapi"`; `Errors()` contains it once per generation | P6.4 | tag |
| F32 | `TestPrecedence` | Table (one row per §1 boundary, winner asserted): (1) reflection < Provider; (2) Provider < Enumer; (3) Enumer < Customizer (§5 O9); (4) Customizer < SchemaHook; (5) component vs use-site tag: tag wins at use site, component keeps its value (D7); (6) tag < FieldHook; (7) ParameterHook < `Parameter` option; (8) options < OperationHook; (9) OperationHook < DocumentHook (served only); (10) `CustomizeOperation` < OperationHook | P6.4 | tag |
| F33 | `TestComponentNeverMutatedByUseSite` | Component bytes are identical with and without all use-site annotations, and in both registration orders | P3.4 | now |
| F34 | `TestPointersAreNullableOption` | `WithPointersAreNullable(false)`: `*string` → `{"type":"string"}`, `*fAddr` → plain `$ref` | P3.5 | tag |
| F35 | `TestAnnotationStyleAllOfWrap` | `WithAnnotationStyle(openapi.AllOfWrap)` + description → `{"allOf":[{"$ref":…}],"description":"d"}`; nullable + description → `{"anyOf":[{"$ref"},{"type":"null"}],"description":"d"}` in both styles (§5 O26) | P3.4 | tag |

(There are no F13/F31. In F32, Enumer runs before Customizer so that a Customizer can edit `enum`.)

### Batch G: lifecycle, errors, spec handler, YAML. Files: `internal/acceptance/lifecycle_test.go` (now + tag split: `lifecycle_tag_test.go`), `spechandler_test.go` (tag), `yaml_test.go` (now)

| ID | Test | Asserts | FixedBy | Build |
|---|---|---|---|---|
| G01 | `TestNoPanicsOnBadInput` | One fizz, one route per row, each row its own Pending. Every row: no panic; the gin route answers ≠404; an `Errors()` entry contains the row's path. Rows: r1 `query:"q,omitempty"` (P4.2); r2 response code clash (P4.8); r3 conflicting locations (P4.8); r4 duplicate explicit ID (P4.7); r5 two tonic handlers (P4.8); r6 `chan int` body field (P3.3); r7 NaN example (P2.5); r8 `default:"maybe"` bool (P3.6); r9 param Duration `default:"1s"` (P3.6); r10 `cookie` on a struct (P4.2c); r11 deepObject `map[string]struct{}` (P7.1); r12 deepObject `query:"f,required"` (P7.1); r13 security scheme `{}` (P4.8) | per row | now |
| G02 | `TestWarningsDoNotFailErr` | Missing path param (D10), MarshalJSON-only type (D14), param default+required (D21), conflicting Group description (P6.5) → `f.Err()==nil`; each `Errors()` entry satisfies `errors.As(e, new(*openapi.Warning))` | P6.5 | tag |
| G03 | `TestErrJoinsErrors` | Code clash + omitempty → `Err()` non-nil; `errors.As` finds both `*openapi.OperationError`; `Err().Error()` contains both route paths, newline-separated; no warning text | P4.8 | tag |
| G04 | `TestErrorsReturnsCopy` | Mutating the returned slice does not change the next `Errors()` | P4.8 | now |
| G05 | `TestFailedOpLeavesNoTrace` | Op `fizz.ID("x")` with a code clash → absent from the spec; a later op `fizz.ID("x")` → succeeds, no duplicate error | P4.8 | now |
| G06 | `TestSpecHandlerNetHTTP` | `mux.Handle("/o.json", f.SpecHandler("json"))` → 200, `application/json`, valid; `"yaml"` → `application/yaml`; `"xml"` → 500 + an error recorded at construction (§5 O7) | P5.1 | tag |
| G07 | `TestOpenAPIWrapsSpecHandler` | `f.OpenAPI(info,"json")` body == `SpecHandler("json")` body after `SetInfo(info)` | P5.1 | tag |
| G08 | `TestSpecCacheInvalidation` | GET, GET → identical bytes; register a route → GET contains the new path | P5.1 | now |
| G09 | `TestSlowDocumentHookDoesNotBlockRegistration` | `testing/synctest`: DocumentHook blocks on a channel during a GET; a concurrent `f.GET(...)` returns; `synctest.Wait`; then release | P6.4 | tag |
| G10 | `TestAPIReturnsDeepClone` | `a := Generator().API(); a.Paths["/x"] = nil`, set `a.Info.Title` → the next `API()` and the served doc are unchanged | P5.1 | now |
| G11 | `FuzzCloneRoundTrip` | Seed extension values `18446744073709551615`, `0.1`, `1e400` (as `json.Number`), nested maps → `MarshalJSON(Clone(d)) == MarshalJSON(d)` | P5.1 | tag |
| G12 | `TestOperationInRequestContext` | The handler calls `fizz.OperationFromContext(c.Request.Context())` → op ID `getCtx`; `c.Keys` holds no `*openapi.Operation` (rule c) | P5.1 | now |
| G13 | `TestOperationFromContextIsClone` | Two calls → distinct pointers; mutation → spec unchanged; works with `*gin.Context` when `ContextWithFallback=false` | P5.1 | now |
| G14 | `TestYAMLParityServed` | For the legacy fizz: yaml.v3-decode(served YAML) deep-equals json-decode(served JSON), numbers normalized | P2.5 | now |
| G15 | `TestYAMLAmbiguousStrings` | Example values `"123","true","null","~","1e3","","0x1F","yes","on","2026-10-30"` → decoded from YAML as the same strings | P2.5 | now |
| G16 | `TestYAMLIgnoresUserMarshalYAML` | The example type's `MarshalYAML` returns `"yaml-only"` → served YAML lacks `yaml-only` and contains the JSON value | P2.5 | now |
| G17 | `TestYAMLBlockStyle` | No flow mappings other than empty `{}`/`[]`; 2-space indent | P2.5 | now |
| G18 | `TestMarshalYAMLMatchesHandler` | `openapi.MarshalYAML(doc)` == served bytes | P2.5 | tag |
| G19 | `TestYAMLContentType` | Served YAML `Content-Type: application/yaml` (§5 O7) | P2.5 | now |
| G20 | `TestDeterminismFeatureDoc` | The "everything" fizz (B–H fixtures together: hooks, extensions, deepObject, cookie, generics) 20× at GOMAXPROCS 1 and 8 → byte-identical JSON and YAML | P6.4 | tag |
| G21 | `TestJSONv1Parity` | `encoding/json` v1 `json.Marshal(doc)` == `openapi.MarshalJSON(doc)` for the kitchen sink + extensions | P2.3 | tag |
| G22 | `TestVersionConstant` | `openapi.Version == "3.2.1"` == served `/openapi` | P2.6 | tag |

### Batch H: extensions and components. Files: `internal/acceptance/extensions_test.go`, `components_test.go`, `internal/acceptance/testdata/extensions-everywhere.json`

| ID | Test | Asserts | FixedBy | Build |
|---|---|---|---|---|
| H01 | `TestExtensionsRoundTripEverywhere` | The hand-written valid 3.2.1 doc with `x-k` on every extensible object (OpenAPI, Info, Contact, License, Server, ServerVariable, Components, PathItem, Operation, ExternalDocs, Parameter, RequestBody, MediaType, Encoding, Response, Header, Example, Link, Callback, Tag, Schema, Discriminator, XML, SecurityScheme, OAuthFlows, OAuthFlow) → `json.Unmarshal` into `openapi.OpenAPI` → `json.Marshal` → semantically equal; `specvalidate.Validate` ok | P2.3 | now |
| H02 | `TestExtensionKeysSorted` | Round trip of `{"x-b":1,"x-a":{"z":1,"a":2}}` on Info → bytes have `x-a` before `x-b` and `a` before `z` | P2.3 | now |
| H03 | `TestSetExtensionValidation` | `(*openapi.Operation).SetExtension`: `"foo"` → `ErrInvalidExtensionKey`; `"x-"` → error; `"x-oai-a"`, `"x-oas-a"`, `"X-OAI-a"` → `ErrReservedExtension`; `"x-ok"` → nil | P2.3 | tag |
| H04 | `TestTypedExtensionWins` | Typed `XLogo`/`XTagGroups`/`XCodeSamples`/`XInternal` + `SetExtension` with the same key → the typed value emitted, error recorded | P2.3 | tag |
| H05 | `TestGeneratorSetExtension` | `Generator().SetExtension("x-api-id","a")` → root `x-api-id`; set twice → last writer wins | P6.5 | tag |
| H06 | `TestRegisterSchema` | `RegisterSchema("Money", s)` → `/components/schemas/Money`; the same name again → `CollisionError`, first kept | P6.5 | tag |
| H07 | `TestRegisterType` | `RegisterType(TypeFor[hPet]())` with no route → component `AcceptanceHPet`; a later route uses the same `$ref` | P6.5 | tag |
| H08 | `TestRegisterOtherComponents` | `RegisterParameter/Response/Example/RequestBody/Header(name, v)` → `/components/<kind>/<name>`; duplicate → error | P6.5 | tag |
| H09 | `TestSetTagAndGroups` | `SetTag(&Tag{Name:"pets",Summary:"Pets",Kind:"nav",Parent:"root"})`, `SetTagGroups(...)` → tag fields and `x-tagGroups` | P6.5 | tag |
| H10 | `TestSetSelfAndDialect` | `SetSelf(u)` → `$self`; `SetJSONSchemaDialect(u)` → `jsonSchemaDialect`; `SetExternalDocs` → root `externalDocs` | P6.5 | tag |
| H11 | `TestSetTagAfterGroupOverrides` | `Group(.., "grp","A")` then `SetTag(&Tag{Name:"grp",Description:"B"})` → `B`, no warning (explicit setter = last writer wins) | P6.5 | tag |
| H12 | `TestOAuthFlowYAML` | Served YAML of `ops/oauth-flow-without-scopes` has `scopes: {}` and `tokenUrl` (wrong MarshalYAML removed) | P2.3 | now |

### Batch I: docs, examples, market. Files: `example_test.go` (root, `package fizz_test`, tag until P8.1), `internal/acceptance/docs_test.go` (now), `examples/market/market_test.go` (now)

| ID | Test | Asserts | FixedBy |
|---|---|---|---|
| I01 | `ExampleNew_options` | `New(WithTagNames(...), WithPointersAreNullable(false))`, prints one fragment; `// Output:` | P6.1 |
| I02 | `ExampleFizz_Docs` | Default Swagger UI; prints the status and whether the body contains `swagger-ui` | P8.1 |
| I03 | `ExampleScalarUI` | Prints the Scalar script URL | P8.1 |
| I04 | `ExampleWithHooks` | DocumentHook adds `x-tagGroups`; prints it | P6.4 |
| I05 | `ExampleSchemaProvider` | Prints the component | P6.3 |
| I06 | `Example_deepObject` | `?filter[color]=red` → prints `map[color:red]` | P7.2b |
| I07 | `Example_cookie` | Prints the bound cookie | P7.2c |
| I08 | `ExampleFizz_Err` | Prints the joined error | P4.8 |
| I09 | `ExampleRequestBody` / `ExampleCustomizeOperation` | Prints fragments | P6.6c |
| I10 | `TestREADMESnippetsMatchExamples` | Every ```go block preceded by `<!-- example: ExampleX -->` == the gofmt'd body of ExampleX; ≥8 such blocks. README has no `validator.v8` and no `versions/3.0.1.md`; it contains `3.2.1`, a tags table row for each of the 15 TagNames fields and the 7 BinderTags, `11065` and `ScalarUI`, and sections Interfaces, Hooks, Naming, deepObject, Cookies, Docs UI, YAML | P9.2 |
| I11 | `TestMigrationCoversTopics` | MIGRATION.md contains: `SchemaOrRef`, `.Value`, `.Ref`, `TypeSet`, `json.Number`, `Explode`, `MarshalYAML`, `SpecGenConfig`, `TagNames`, `BinderTags`, `Errors()`, `Err()`, `InstallBindHook`, `WithoutBindHook`, `operationId`, `DELETE`, `Duration`, `url.URL`, `nullable`, `oapi-codegen`, `kin-openapi`, `ogen`, `DocumentHook`, `snapshot`, `1.26`; its "after" snippets match Examples (same marker rule as I10) | P9.1 |
| I12 | `TestReleaseNotes` | `CHANGELOG.md` v0.30.0 section contains `Go 1.27`, `1.26`, `3.2.1`, `MIGRATION`, `bind hook`, `oapi-codegen` | P9.5 |
| I13 | `TestMarketSpecValid` | `NewRouter()` → GET `/openapi.json` passes `Validate` + 3 lints; root `x-tagGroups`; `ListFruits` param `filter` with `style: deepObject`; one `in: cookie` param; GET `/docs` serves Scalar (`@scalar/api-reference`) | P9.3 |
| I14 | `TestMarketFilterEndpoint` | `GET /market?filter[origin]=Spain` → 200, only Spanish fruits; `?filter[a][b]=x` → 400 | P9.3 |
| I15 | `TestDocsCompatTable` | docs/compat.md: pins == docsui constants (shares the C06 parser), mentions `AllOfWrap` and `11065` | P8.4 |

### Batch K: kitchen-sink gaps and fixes. Files: `internal/kitchensink/cases_gaps.go` (new), `cases_ops.go`, `cases_params.go` (fixes only), `differential_test.go`

Fixes. These existing cases contradict the default naming (`fullNames: true`, generator.go:75), so they can never pass as written:
- **K-fix1** `ops/body-name-collision-go-type`. Go type `OXInput` is named `KitchensinkOXInput`, but body `oX` is named `OXInput`, so there is no collision. Use `fizz.ID("kitchensinkOX")` and expect `KitchensinkOXInput` (the Go type is kept, error recorded).
- **K-fix2** `ops/ref-with-description-sibling`: `OHolder`/`OUser` → `KitchensinkOHolder`/`KitchensinkOUser`.
- **K-fix3** `params/delete-body-documented`: `PDeleteIn` has a path param, so P4.6 names its body `OpInput`. Expect `{"$ref":"#/components/schemas/OpInput"}` and `OpInput/properties/name`.

| ID | Case | Expect | FixedBy |
|---|---|---|---|
| K01 | `types/interface-any` | `any` field → `{}` | P2.6 |
| K02 | `params/no-allow-empty-value` | `parameters/0/allowEmptyValue` Absent | P2.6 |
| K03 | `params/explode-absent-by-default` | `parameters/0/explode` Absent when there is no tag | P2.6 |
| K04 | `params/query-duration-default-string-error` | `default:"1s"` on a Duration query → Errors `d`, no panic (D16) | P3.6 |
| K06 | `TestDifferentialJSONKeys` (test) | For each `T*`/`F*` sample: keys of `json.Marshal(sample)` == `properties` keys | P3.1a |
| K07 | `validate/ne-not-const` | `ne=a` → `{"not":{"const":"a"}}` | P4.1b |
| K08 | `validate/keys-property-names` | `dive,keys,min=2,endkeys` on map → `propertyNames:{minLength:2}` | P4.1b |
| K09 | `validate/unique-items` | `unique` → `uniqueItems:true` | P4.1b |
| K10 | `validate/alpha-patterns` | alpha, alphanum, numeric, hexadecimal → the four patterns | P4.1b |
| K11 | `validate/formats` | url→`uri`, uri, uuid, uuid4, ipv4, ipv6, hostname, email → `format` | P4.1b |
| K12 | `validate/datetime-not-mapped` | `datetime=2006` → no format, no error | P4.1b |
| K13 | `validate/string-len-both-bounds` | `len=3` on a string → min/maxLength 3 | P4.1b |
| K14 | `validate/duration-bounds-ns` | `min=1s` on Duration → `minimum:1000000000` | P4.1b |
| K15 | `validate/or-group-dropped-if-unmapped` | `email\|customrule` → no `anyOf` | P4.1b |
| K16 | `validate/gte-lte-inclusive-decimal` | `gte=0.5,lte=1.5` → `minimum:0.5,maximum:1.5` | P4.1b |
| K17 | `required/omitempty-required` | in `required` | P4.1c |
| K18 | `required/required-if-not-required` | not in `required` | P4.1c |
| K19 | `required/dive-required-not-required` | not in `required` | P4.1c |
| K20 | `ops/default-response-examples-merge` | `DefaultResponseExamples({"a":…})` → `/responses/default/content/application~1json/examples/a` | P4.4 |
| K21 | `ops/body-single-struct-direct-ref` | Input `*KPet` (json fields only) → requestBody `$ref` `KitchensinkKPet`; no `…Input` component | P4.6 |
| K22 | `ops/body-required-if-any-field-required` | Field `validate:"required"` → `requestBody/required:true`; none → Absent | P4.6 |
| K23 | `params/get-body-not-documented` | GET input with a json field → `requestBody` Absent (guard) | — |
| K24 | `params/head-body-not-documented` | Same for HEAD (guard) | — |
| K25 | `ops/default-id-closure` | Closure in `kRegister` → `"kRegister"` | P4.7 |
| K26 | `ops/default-id-collision-package-prefix` | `a/models.Get` vs `b/models.Get` → `Get`, `models.Get` (§5 O4) | P4.7 |
| K27 | `ops/default-id-collision-method-path` | Third clash → `get_c_id` | P4.7 |
| K28 | `ops/explicit-id-collision-error` | Errors `dup`; second op Absent | P4.7 |
| K29 | `ops/multiple-tonic-handlers-error` | Errors contain path; no panic | P4.8 |
| K30 | `ops/failed-op-id-reusable` | Same as G05, fragment form | P4.8 |
| K31 | `params/cookie-non-scalar-error` | `cookie` on a struct → Errors `cookie`; no param | P4.2c |
| K32 | `params/deep-object-struct` | Struct target → `schema.properties` from json names | P7.1 |
| K33 | `params/deep-object-ptr-map` | `*map[string]string` → same as map | P7.1 |
| K34 | `params/deep-object-nested-value-error` | `map[string]struct{}` → error | P7.1 |
| K35 | `params/deep-object-required-option-error` | `query:"f,required"` → error | P7.1 |
| K36 | `params/deep-object-default-tag-error` | `default:"x"` → error | P7.1 |
| K37 | `params/deep-object-validate-required` | `validate:"required"` → `required:true` | P7.1 |
| K38 | `params/deep-object-slice-values` | `map[string][]string` → `additionalProperties:{type:array,items:{type:string}}` | P7.1 |
| K39 | `params/wildcard-description-notes-slash` | `/f/*path` → description contains `/` | P4.3 |
| K40 | `params/gin-path-translation-multi` | `/u/:id/f/*path` → path key `/u/{id}/f/{path}` | P4.3 |
| K41 | `ops/plain-gin-handler-undocumented` | `f.GET("/plain", nil, func(*gin.Context){})` → path Absent; no errors (D11 guard) | — |

P4.1a's task-owned parse table must contain at least these rows: `required`; `omitempty,min=1`; `min=1,max=3`; `oneof=a b|eq=c`; `dive,min=1`; `dive,keys,min=1,endkeys,max=3`; `gt=0x2C`; `len=3`; `required_if=F 1`; `email|url`; `unique`; `dive,dive,required`; `keys` without `dive` (error); `endkeys` alone (error); `omitempty`; `` (empty); `min=`; `oneof='a b' c` (single quotes stay literal in v9); `max=10,dive,max=3`; `hexadecimal`.

### Batch summary

| Batch | Files (disjoint) | Tests | now / tag | Can start |
|---|---|---|---|---|
| 0 | kitchensink/{case,assert}.go, internal/plan, internal/acceptance/at | 6 | now | immediately (first) |
| S (P1.0 stub step) | `zz_stub_acceptance.go` + `doc.go` in ./, openapi/, bind/, docsui/, docsui/embedded/; variadic `New` | build | tag | immediately |
| A | internal/archtest/*, acceptance/arch_test.go | 10 | 7 / 3 | after 0 |
| B | acceptance/bind_*_test.go, acceptance/hookorder, acceptance/nohook | 34 | 27 / 7 | after 0 (tag part after S) |
| C | acceptance/docsui_test.go, embedded_test.go | 20 | 0 / 20 | after S |
| D | acceptance/options_test.go | 16 | 1 / 15 | after S |
| E | acceptance/naming_test.go, acceptance/fixtures/naming | 11 | 5 / 6 | after 0 |
| F | acceptance/{tags,interfaces,hooks,precedence}_test.go | 33 | 13 / 20 | after 0 |
| G | acceptance/{lifecycle,lifecycle_tag,spechandler,yaml}_test.go | 22 | 12 / 10 | after 0 |
| H | acceptance/{extensions,components}_test.go, testdata | 12 | 3 / 9 | after 0 |
| I | example_test.go (root), acceptance/docs_test.go, examples/market/market_test.go | 15 | 6 / 9 | after S |
| K | kitchensink/cases_gaps.go, cases_ops.go, cases_params.go, differential_test.go | 3 fixes + 40 | now | after 0 |
| **Total** | | **222 (219 new tests/cases + 3 case fixes)**; 123 compile now, 99 need the tag | | |

---

## 4. Gates (every task PR)

| Gate | Rule | Mechanism |
|---|---|---|
| G-META | Every document built by a passing kitchen-sink case or acceptance test passes `specvalidate.Validate`, `LintNoStrayNull`, `LintNoNullable` and `LintComponentNames` | `TestAllCaseDocsValid` (un-skip at P2.6); `kitchensink.Run` and `at.Spec` apply the four checks automatically once `plan.Done["P2.6"]`; `TestSpikeDocValid`, I13 |
| G-NULL | Stray-null lint: `null` only under example/examples/default/const/enum/value/dataValue | Part of G-META; also applied to the YAML-decoded doc in G14 |
| G-DET | Byte-identical output | `TestDeterminism` (switch to the full kitchen-sink fizz at P2.6), G20; CI `-count=1 -shuffle=on`, kitchen sink and acceptance `-count=3` |
| G-RACE | `go test -race ./...` clean | All packages; specific: `TestConcurrentServeAndRegister`, B15, B31, G09 |
| G-STALE | 0 skipped-but-passing | `FIZZ_STALE=fail` in CI → `kitchensink.Run` and `at.Pending` fail stale skips. Raw `t.Skip("fixed by")` is forbidden (Batch 0 converts the 3 existing ones) |
| G-COMPOSITE | Composite FixedBy IDs: `"A+B"` (e.g. `"P3.3+P4.8"`) stays red until both tasks are in `Done` | `plan.IsDone` splits on `+` and requires every part |
| G-DONE | When task X merges, 0 skips with FixedBy X | The PR adds X to `internal/plan/done.go`; `at.Pending` and `kitchensink.Run` fail any red test whose FixedBy is Done |
| G-STUB | Stubs only for unfinished tasks | Tagged `TestStubsBelongToOpenTasks` parses `// FixedBy:` on every stub decl → none in Done. Redeclaration breaks `go vet -tags acceptance` if a stub is left. At P9.4: no `zz_stub_acceptance.go`, no `//go:build acceptance` (grep step) |
| G-IMP | Router-agnostic rule | A01 (blocking from P1.3) |
| G-GOLD | Goldens unchanged unless reviewed | `golden.Assert`; regeneration diff reviewed by Opus (process) |
| G-LINT | vet, staticcheck, govulncheck, golangci-lint | `go tool -modfile=tools/go.mod …` |
| G-COV | Coverage ≥ 90% | From P9.4 |

Burn-down targets (skipped tests with FixedBy in the named range, after the phase merges):

| After | Existing kitchen-sink skips | Rule for all tests (K, acceptance) |
|---|---|---|
| Now | 89 | — |
| Phase 1 | 89 | 0 FixedBy P1.* (A01, A03, A08) |
| Phase 2 | 105 measured at P2.6 (142 cases incl. Batch K; the 84 estimate predates them) | 0 FixedBy P2.* (incl. K01–K03, G14–G19, G21, G22, H01–H04, H12, A10) |
| Phase 3 | 37 | 0 FixedBy P3.* (incl. K04, K06, F33–F35*) |
| Phase 4 (+P6.1) | 4 (P5.1=1, P6.5=2, P7.1=1) | 0 FixedBy P4.*, P6.1 |
| Phase 5 | 3 | 0 FixedBy P5.1 |
| Phase 6 | 1 | 0 FixedBy P6.* |
| Phase 7 | 0 | 0 FixedBy P7.* |
| Phase 8 / 9 | 0 | 0 FixedBy anywhere; no stubs; no `acceptance` tag |

*F34 and F35 need P6.1's option. Their FixedBy stays P3.5/P3.4 for behaviour, but they compile only under the tag until P6.1. G-DONE treats a tagged test whose option is still a no-op stub as red. Resolution: set their FixedBy to **P6.1** so that P3 is not blocked. This is listed under §5 O36.

CI wiring (`.github/workflows/ci.yml`, P9.4; the first three jobs can land now):
1. `test`:
   - matrix: {1.27.x, stable} × ubuntu, plus stable × {macos, windows};
   - `FIZZ_STALE=fail go test -race -count=1 -shuffle=on ./...`.
2. `acceptance` (from P1.0): `go vet -tags acceptance ./...` && `FIZZ_STALE=fail go test -tags acceptance -race -count=1 ./...`.
3. `kitchensink`:
   - `go test -count=3 -shuffle=on ./internal/kitchensink/... ./internal/acceptance/...`;
   - `go test -json` piped into `go run ./internal/plan/burndown`, which writes a per-FixedBy skip table to `$GITHUB_STEP_SUMMARY` and fails on any Done task with skips.
4. `archtest`: `go test ./internal/archtest/...` (blocking from P1.3; until then the test is Pending).
5. `lint`: vet, staticcheck, govulncheck, golangci-lint.
6. `tip`: gotip, `continue-on-error: true`.
7. `release-gate` (on tags): no stubs, no tag, I12 green, coverage ≥ 90%.

---

## 5. Open items blocking exact assertions

Each item has a proposal so the tests can be exact now. **USER** = needs the user's decision (public API or output a user sees). Otherwise Opus decides in the P1.0 ADR.

| # | Open item | Proposal (used by the tests above) | Who |
|---|---|---|---|
| O1 | Docs UI API. D18 says `f.Docs(path, spec)`, `fizz.SwaggerUI()`, `fizz.ScalarUI()`; P8.1 says `f.Docs(path string, h http.Handler)` and `docsui.SwaggerUI(specURL, …)` | `func (f *Fizz) Docs(path, specURL string, ui ...DocsUI)` with default `SwaggerUI()`. `fizz.SwaggerUI(...docsui.Option) DocsUI`, `fizz.ScalarUI(...)`, `fizz.EmbeddedSwaggerUI(...)`, `fizz.EmbeddedScalarUI(...)`; embedded variants also register `path+"/assets/*"`. Router-agnostic level: `docsui.SwaggerUI(specURL, ...Option) http.Handler`, `docsui.Scalar(...)`, `embedded.SwaggerUI(specURL, prefix, ...)`. More than one `ui` → error, first honored | **USER** |
| O2 | Generic and composite component names | `UpperFirstASCII(pkg)+Name` + `_`-joined args. A named arg renders as `pkg.Name` (last path segment); `[]T` → `List_T`; `map[K]V` → `Map_K_V`; `*T` → `T`; other disallowed runs → `_`. Matches the existing case `GenericPage_models.Item` | **USER** |
| O3 | What the losing type's use site references after a collision | Inline the loser's schema at its use sites + `CollisionError{Name, Kept, Rejected}`. A dangling `$ref` to the wrong schema is never emitted | Opus |
| O4 | Default op-ID collision formats | 1st `Name`; 2nd `pkg.Name` (last path segment); 3rd `lower(method)_` + path segments joined by `_` with braces stripped (`get_a_id`). Explicit `fizz.ID` collision → error, second op not documented | **USER** |
| O5 | Exact error messages | Tests assert the **type + fields + identifiers**, never full text. Types: `*openapi.OperationError{Method, Path, ID, Err}`, `*openapi.FieldError{Type, Field, Err}`, `*openapi.CollisionError{Name, Kept, Rejected}`. Sentinels: `ErrInvalidExtensionKey`, `ErrReservedExtension`, `ErrNilOrRef`, `bind.ErrNestedKey`, `bind.ErrTooManyKeys`. Messages contain path, op ID, field and component names | Opus |
| O6 | Warning shape | `type Warning struct{ Method, Path, ID, Msg string }` with `Error()` and `IsWarning() bool`; `Errors() []error` keeps its signature and returns a copy | Opus |
| O7 | Spec handler | `SpecHandler(format string) http.Handler` uses the generator's Info. `OpenAPI(info, ct)` = `SetInfo(info)` + `gin.WrapH`. Content types `application/json` and `application/yaml`. An unknown format → error recorded and the handler answers 500 (no panic) | Opus |
| O8 | Where DocumentHook applies | In served output (`SpecHandler`/`OpenAPI`) and in `f.Document() (*openapi.OpenAPI, error)`; **not** in `Generator().API()`. Hook error → 500 and one `Errors()` entry per generation. One run per generation, shared by JSON and YAML | Opus |
| O9 | Hook registration and signatures | `fizz.WithHooks(openapi.Hooks{Schema, Field, Parameter, Operation, Document})` and `(*Generator).SetHooks`. Multiple `WithHooks` compose in order. Signatures: `ParameterHook(*ParameterContext, *Parameter) (skip bool)`, `OperationHook(*OperationContext, *Operation) error`. Component-stage order: reflection/Provider → Enumer → Customizer → SchemaHook | Opus |
| O10 | Do named scalar types with Enumer/Provider/Customizer become components? | Yes. Plain named scalars stay inline as today | Opus (flag to user in MIGRATION) |
| O11 | `additionalProperties:"false"` on a `$ref` field. A sibling `additionalProperties:false` next to `$ref` rejects **every** property, because siblings do not see `$ref`'s properties | On a `$ref`, emit `unevaluatedProperties:false`. On an inline object, emit `additionalProperties:false`. On non-objects, error | Opus |
| O12 | `inline` tag semantics (undefined in the plan) | `inline:"true"` on a named-struct field emits the reflected schema inline; the component is unaffected | Opus |
| O13 | P4.2c says `explode:"false"` is honored, but cookies are scalar-only | Drop the explode mention; any non-scalar → error | Opus |
| O14 | deepObject grammar edges | Repeated key into a scalar value → 400. `name[]`, an unclosed `[`, bare `name=` → 400. Struct keys match json names exactly (case-sensitive, unlike encoding/json). Keys not starting with `name[` are ignored | Opus |
| O15 | Cookie absent vs empty; decoding | `default` applies only when absent. Values are raw (no URL-decoding) | Opus |
| O16 | Install-once semantics. With `sync.Once`, a user hook set **after** the first `New` replaces fizz's hook, and a later `New` does not re-wrap | `InstallBindHook` is idempotent **by identity**: if the current hook is not fizz's wrapper, wrap it again. `New` then always composes, and the replaced-hook warning (B28) covers only replacement after the last `New`. Affects B25/B28/B30 | **USER** (touches D20 global behaviour) |
| O17 | `RequestBody` + `Content` vs the inferred body | Any `Content` given → replaces all inferred content. Desc/required only → override just those. On GET/HEAD → error | Opus |
| O18 | `CustomizeOperation` ordering | Deferred until after all other options of that op, in declaration order among `CustomizeOperation`s | Opus |
| O19 | `Tags(...)` | Appends to the Group tag, deduplicated, order kept. Operation tags are not auto-declared at the root | Opus |
| O20 | `ResponseOption` set | `WithModel(any)`, `WithHeaders(...)`, `WithExample(any)`, `WithExamples(map)`, `WithContent(mt, model)`, `WithResponseExtension(k, v)` | Opus |
| O21 | Namer option names | `WithRequestBodyNamer(func(opID string, t reflect.Type) string)`, `WithOperationIDNamer(func(method, path, handlerName string) string)` | Opus |
| O22 | Deprecated `SpecGenConfig` with a renamed binder tag | `NewGenerator` returns an error naming the field when a non-empty value ≠ tonic's; empty → tonic's | Opus |
| O23 | `TagNames` zero values | A zero field = default name; `"-"` disables that tag | Opus |
| O24 | Extension key rules and API | `SetExtension(k string, v any) error` method on every extensible model type (stub-able). Key must start with lowercase `x-` and have a non-empty suffix. `x-oai-`/`x-oas-` rejected case-insensitively | Opus |
| O25 | P8.4 premise is wrong: the spike (docs/compat.md) shows Swagger UI 5.34.1 **renders** `$ref` sibling descriptions | Reword P8.4 to document the observed gaps: `$ref` target names lost in inline schemas, tag `summary`/`kind`/`parent`, `itemSchema` (#11065), extensions hidden. Keep the AllOfWrap mention as an option, not a workaround | Opus |
| O26 | AllOfWrap combined with nullable | Nullable already wraps in `anyOf`, so annotations go beside the `anyOf` in both styles | Opus |
| O27 | Docs page structure (needed for exact HTML assertions) | Config in `<script type="application/json" id="fizz-docs-config">`; `referrerpolicy="no-referrer"` on asset tags; `Content-Type: text/html; charset=utf-8` | Opus |
| O28 | Bind error body | Errors pass through tonic (handler.go:76 already wraps in `BindError`); default hook → 400 `{"error":…}` containing the raw key | Opus |
| O29 | `bind.PlanFor` `tags` parameter type | `bind.Tags{Query, Cookie, Style, Default string}`, owned by `bind` (no `openapi` import) | Opus |
| O30 | Default `Errors()` ordering | Registration order (deterministic, needed by G20) | Opus |
| O31 | The 3 existing kitchen-sink cases with impossible names (K-fix1..3) | Fix in Batch K before P3.4 and P4.6 | none |
| O32 | Duplicate gin route (same method and path) panics inside gin | Out of scope for "no panics". Document it; optionally fizz pre-checks and records an error. Tests exclude it | Opus |
| O33 | Plan text: race test lives in `internal/kitchensink/race_test.go`, not the root | Update PLAN P0.5 | none |
| O34 | `types/byte-slice` has FixedBy P3.3, but P2.6 implements `[]byte → contentEncoding` | G-STALE will force P2.6 to drop it. Relabel to P2.6 now | none |
| O35 | Hook error/panic policy | Errors are recorded. Panics from user hooks propagate (a user bug); fizz itself never panics | Opus |
| O36 | Tagged tests whose behaviour task precedes their option task (F34, F35) | FixedBy = the later of the two (P6.1) | none |

Questions for the user: **O1** (docs UI API shape), **O2** (generic component name format), **O4** (op-ID collision formats), **O16** (bind-hook re-wrap semantics vs `sync.Once`).

---

## User decisions on blocking questions (2026-10-30)

- **O1 Docs UI API:** `f.Docs(path, specURL string, ui ...DocsUI)`; default = Swagger UI; selectors `fizz.SwaggerUI()`, `fizz.ScalarUI()`, `fizz.EmbeddedSwaggerUI()`, `fizz.EmbeddedScalarUI()`; explicit choice always honored. Router-agnostic `docsui.SwaggerUI(specURL)` / `docsui.ScalarUI(specURL)` return `http.Handler`. Resolves the D18 vs P8.1 contradiction in favor of this shape.
- **O2 Generic component names:** `<PkgPrefix><Type>_<arg>...` with each arg as `pkg.Type` (e.g. `GenericPage_models.Item`, `GenericPair_string_int`), `[]T` → `List_T`, `map[K]V` → `Map_K_V`, `*T` → `T`. Must match `^[A-Za-z0-9._-]+$`.
- **O4 Default operation-ID collisions:** handler `Name` → on collision `pkg.Name` → on collision `method_path` (e.g. `get_users_id`); strip `-fm` and generic suffixes; explicit `fizz.ID` collision = error.
- **O16 Bind hook:** re-wrap by identity on every `fizz.New()` (no `sync.Once`): if fizz's step is not in the current chain, wrap the current hook. Hooks installed after `New` must compose via `fizz.BindHook(next)`; a warning is logged when a deepObject/cookie plan exists but fizz's step didn't run.

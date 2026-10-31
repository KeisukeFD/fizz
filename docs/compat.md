# Docs UI compatibility notes (OpenAPI 3.2.1)

How the two docs UIs that fizz serves, Swagger UI (the default) and Scalar,
handle the OpenAPI 3.2.1 documents fizz generates. Decisions: D17-D19.

## Verification pending

**No claim in this file about how a UI renders a document was checked against
a document that fizz generates.** The only browser run is the P0.6 spike
(below). It used a hand-made document and data-url auto-init, not the pages
that `docsui` serves. Every rendering claim carries one of these statuses:

| Status | Meaning |
|---|---|
| `verified (P0.6 spike, ...)` | Observed in headless Chrome on 2026-10-30 with the hand-made `docs/compat/spike/spec-3.2.1.json`. Text was read from the DOM, not judged visually. |
| `upstream issue #N` | Taken from the upstream issue tracker. Not observed here. |
| `unverified` | No evidence: neither observed nor taken from an issue. |

Before tagging v0.30.0, the maintainer runs the following in a browser (see
[Run the check locally](#run-the-check-locally)) and updates the statuses:

- [ ] `/docs/swagger` loads with no console error and lists the kitchen-sink
      operations (`listPets`, `getPet`, `createPet`, `deletePet`, `getItem`).
- [ ] `/docs/scalar` loads and lists the same operations. Confirm that
      `Scalar.createApiReference("#app", cfg)` initializes with
      `@scalar/api-reference@1.76.4`, and that `{"url": "/openapi.json"}` is
      accepted as its configuration. The page (`docsui/page.tmpl`) uses this
      call; the P0.6 spike used data-url auto-init, so the call itself has
      never been run.
- [ ] Both pages load their script and style sheet with no SRI or CORS error in
      the console. (Chrome accepted the three hashes in the spike.)
- [ ] The Swagger UI page makes no request to `validator.swagger.io`
      (`validatorUrl` is `null`).
- [ ] The `$ref` sibling `description` below is shown in Swagger UI; repeat
      with `fizz.WithAnnotationStyle(openapi.AllOfWrap)` and compare.
- [ ] The nullable `$ref` property (`anyOf`) is shown in both UIs.
- [ ] A document with a deepObject query parameter and a cookie parameter
      (README sections "deepObject query parameters" and "Cookies") renders in
      both UIs.
- [ ] A document with a tag that has `summary`, `kind` and `parent` (set with
      `Generator().SetTag`) shows what the tables claim.
- [ ] A request body with several media types (`fizz.RequestBody` with
      `fizz.Content`) lists each of them in both UIs.
- [ ] Embedded variants (`fizz.EmbeddedSwaggerUI`, `fizz.EmbeddedScalarUI`):
      each page works offline, with no request to `cdn.jsdelivr.net` (network
      tab) and, for Scalar, no font request.
- [ ] Replace every `unverified` below with the result, and reconcile the
      README "Docs UI" section (the `P8.4` marker).

## Pinned versions and SRI

All verified HTTP 200 on jsDelivr on 2026-10-30. Hashes computed with
`curl -sL URL | openssl dgst -sha384 -binary | openssl base64 -A`. The browser
accepted all three hashes in the spike (a wrong hash is blocked by Chrome).
These are the values of the constants in `docsui/swagger.go` and
`docsui/scalar.go`.

| Package | File | sha384 |
|---|---|---|
| `swagger-ui-dist@5.34.1` | `swagger-ui-bundle.js` | `sha384-XFYN47VzPiDY6qPtxYKg0nqUeW/upo20xf1j2SUE7gT/i35FJB4cuS/71Iegzk8Z` |
| `swagger-ui-dist@5.34.1` | `swagger-ui.css` | `sha384-4c3vsE5f+trhFyHrFrLM1fFGUn531NIaC7dcqEWIJdoRQJ5CkJbSuSCwXLpaS1KF` |
| `@scalar/api-reference@1.76.4` | `dist/browser/standalone.js` | `sha384-hwB9WroDTpZaI38gr+u9CXA6ksz5gUYgD2GxXazKl/agT1L2GWUiRFWGJYNiFu4Q` |

URL base: `https://cdn.jsdelivr.net/npm/<package>@<version>/<file>`.

What the pages do (`docsui/docsui.go`, `docsui/page.tmpl`):

- Each external `<script>` and `<link>` carries `integrity`,
  `crossorigin="anonymous"` and `referrerpolicy="no-referrer"`.
- The configuration is JSON in
  `<script type="application/json" id="fizz-docs-config">`, parsed by the
  page script.
- Swagger UI: `SwaggerUIBundle(cfg)` with `url` set to the spec URL,
  `dom_id: "#ui"` and `validatorUrl: null`. The `null` always wins over
  `WithUIConfig`, because the public validator does not understand OpenAPI 3.2
  and would be sent the spec URL.
- Scalar: `Scalar.createApiReference("#app", cfg)` with `url` set to the spec
  URL.
- `WithAssetURLs` without integrity values drops SRI.

### Updating a pin: `go generate`

`docsui/embedded/gen.go` carries `//go:generate go run ../internal/pinassets`
(P8.2).

- `go generate ./docsui/embedded` downloads each pinned URL, refreshes the
  files in `docsui/embedded/assets`, and fails if a downloaded sha384 differs
  from the `...Integrity` constant. It needs network access.
- To move a pin: edit the URL constant in `docsui/swagger.go` or
  `docsui/scalar.go`, run `go run ../internal/pinassets -update` from
  `docsui/embedded` (rewrites the `Integrity` constants), then update the pin
  table above. A test checks that the table equals the constants.
- `TestEmbeddedSRI` is offline: it checks the sha384 of each embedded file
  against the constants.

### Embedded variants

`fizz.EmbeddedSwaggerUI` and `fizz.EmbeddedScalarUI` (P8.3, package
`docsui/embedded`) serve the same pinned files from the binary, at
`<path>/assets/`: `swagger-ui-bundle.js`, `swagger-ui.css` and `standalone.js`
(4.9 MB for Scalar). The page carries the same integrity values. Scalar is run
with `withDefaultFonts: false` so that it requests no font from a CDN. Their
rendering is the same code as the CDN variants; whether the embedded pages work
offline in a browser is `unverified`.

## Method of the P0.6 spike

Chrome for Testing (headless, Playwright cache), driven with `playwright-core`.
Pages served from a temp copy with `python3 -m http.server`. Operation
expanded, response and parameter schemas expanded, text read back from the DOM,
console messages captured. Run on 2026-10-30. Screenshots were taken, but the
verdicts come from DOM text, not from the layout or styling. The spike document
is `docs/compat/spike/spec-3.2.1.json`; `TestSpikeDocValid` checks that it is a
valid 3.2.1 document.

## Feature x UI

Each cell is `status`, then the note. Statuses are defined in
[Verification pending](#verification-pending). "Fizz emits it" says whether
the generator produces the construct, so that a row about something fizz never
writes is not mistaken for a gap that affects you.

| Feature | Fizz emits it | Swagger UI 5.34.1 | Scalar 1.76.4 |
|---|---|---|---|
| `openapi: 3.2.1` detected | always | verified (P0.6 spike, no error, badge "OAS 3.2"); the plan's D17 notes support from 5.32 (release notes, not re-checked) | verified (P0.6 spike, "OpenAPI 3.2.1") |
| `$ref` with a sibling `description` | by default, on a use-site annotation | verified (P0.6 spike, sibling description shown on `owner`); not checked on a fizz document: unverified | verified (P0.6 spike, sibling description shown); not checked on a fizz document: unverified |
| `type: ["string","null"]` | for nullable scalars | verified (P0.6 spike, `string \| null`) | verified (P0.6 spike, `string \| null`) |
| nullable `$ref` as `anyOf` | for a pointer to a struct | verified (P0.6 spike, shown as `(object \| null)`; the `$ref` target name is lost and shown as `object`) | verified (P0.6 spike, `Owner`, `nullable`, with the target name) |
| `itemSchema` | no (model field only; reachable from a hook) | upstream issue #11065 (not rendered); not exercised in the spike | unverified |
| tag `summary` | if you set it with `Generator().SetTag` | verified (P0.6 spike, not shown; name and description are) | unverified (the sidebar showed "Pets", not `pets`, which suggests it is used; not confirmed) |
| tag `kind`, `parent` | if you set them with `Generator().SetTag` | unverified (D17 listed them as unverified; the spike did not exercise them) | unverified |
| `x-` extensions | if you set them (`fizz.Extension` and others) | verified (P0.6 spike, an operation extension is not shown; Swagger UI has a `showExtensions` option, not tried) | verified (P0.6 spike, an operation extension is not shown) |
| deepObject query parameter | with `style:"deepObject"` | verified (P0.6 spike, rendered as `object` with additional properties; no deepObject hint; serialization by Try it out not exercised) | verified (P0.6 spike, same; no hint; serialization not exercised) |
| cookie parameter | with the `cookie` tag | verified (P0.6 spike, `session`, `(cookie)`) | verified (P0.6 spike, under "Cookies") |
| `additionalOperations` | no (model field `PathItem.AdditionalOperations`; reachable from a hook) | unverified (D17 notes support in 5.34.0 from the release notes; not re-checked, not exercised in the spike) | unverified |
| `query` method (3.2) | no (model field `PathItem.Query`) | unverified | unverified |
| schema `examples` | with example options and `example` tags | verified (P0.6 spike, #0 "Rex", #1 "Felix") | verified (P0.6 spike, "Examples" label) |
| `contentEncoding: base64` | for `[]byte` fields | verified (P0.6 spike, `media type: image/png`, `encoding: base64`) | verified (P0.6 spike, `string - base64`) |
| request body, several media types | with `fizz.RequestBody` and `fizz.Content` | unverified (the spike document has one body) | unverified |
| `default` response | yes | verified (P0.6 spike) | verified (P0.6 spike) |
| named schemas in Models | yes, as components | verified (P0.6 spike; an inline response schema shows `array<object>`, not `Pet`) | verified (P0.6 spike, `array of Pet`) |
| page init by fizz | n/a | `SwaggerUIBundle(cfg)`: unverified (the spike page was its own) | `Scalar.createApiReference("#app", cfg)`: unverified (the spike used data-url auto-init) |

Console in the spike: Swagger UI produced no error except a 404 for
`/favicon.ico` (an artifact of the test server); Scalar produced none.

The kitchen-sink document of `internal/kitchensink` (`NewWholeFizz`) has pets
operations with query, header and path parameters, generics, security and
servers. It has no deepObject, cookie, `additionalOperations`, `itemSchema` or
`x-` member, so the rows above that need them rest on the spike document only.

## Swagger UI gaps

1. **`itemSchema` is not rendered** (upstream issue
   [swagger-ui#11065](https://github.com/swagger-api/swagger-ui/issues/11065)).
   Fizz does not generate `itemSchema`; it matters for documents you extend
   with a hook.
2. **Tag `summary`** was not shown in the spike. `kind` and `parent`:
   `unverified`.
3. **`$ref` target names are lost in inline schemas.** An inline response
   schema is shown as `array<object>`, not `array<Pet>` (verified, P0.6 spike).
   The Models section keeps the names.
4. **`x-` extensions are hidden** by default (verified, P0.6 spike).
5. **No deepObject hint.** The parameter is shown as an object (verified, P0.6
   spike). Whether Try it out serializes it as `filter[color]=red` is
   unverified.

Scalar shares gaps 4 and 5 (verified, P0.6 spike). Its own gaps beyond the spike
are `unverified`.

### `$ref` siblings

By default fizz puts a use-site annotation next to the `$ref`, which OpenAPI
3.1 and later allow. A tool that ignores siblings of `$ref` would show the
`owner` property without its description. This is the fragment fizz emits (the
`properties` of a `Pet` schema whose fields are `Owner Owner` and
`Prev *Owner`, both with a `description` tag; output of the generator, not
written by hand):

```json
{
  "id": {
    "format": "int64",
    "type": "integer"
  },
  "owner": {
    "$ref": "#/components/schemas/Owner",
    "description": "Current owner of the pet"
  },
  "previousOwner": {
    "anyOf": [
      {
        "$ref": "#/components/schemas/Owner"
      },
      {
        "type": "null"
      }
    ],
    "description": "Previous owner, if any"
  }
}
```

**Status of "Swagger UI ignores the sibling `description`": not observed.** In
the P0.6 spike, Swagger UI 5.34.1 and Scalar 1.76.4 both showed the sibling
`description` of a `$ref` (verified, P0.6 spike). The plan expected the
opposite (P8.4, "a `$ref` sibling `description` that Swagger UI ignores"), and
the spike does not support it, so this file does not claim it. The fragment
above is still the construct to look at in the browser run: it is a fizz
document, and the spike's was not.

`fizz.WithAnnotationStyle(openapi.AllOfWrap)` is the option if a tool you use
does ignore the siblings. It wraps the reference instead:

```json
{
  "id": {
    "format": "int64",
    "type": "integer"
  },
  "owner": {
    "allOf": [
      {
        "$ref": "#/components/schemas/Owner"
      }
    ],
    "description": "Current owner of the pet"
  },
  "previousOwner": {
    "anyOf": [
      {
        "$ref": "#/components/schemas/Owner"
      },
      {
        "type": "null"
      }
    ],
    "description": "Previous owner, if any"
  }
}
```

The nullable property is unchanged: an `anyOf` wrapper already exists, so the
annotations stay beside it in both styles. Whether `allOf` renders better or
worse in either UI is `unverified`; it changes how a model is displayed (an
`allOf` of one schema).

## Run the check locally

Not committed as code. The kitchen-sink fixture is in an `internal` package,
so the program must live inside the fizz module. From the repository root:

```sh
mkdir internal/tmpcheck
cat > internal/tmpcheck/main.go <<'EOF'
package main

import (
	"log"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/kitchensink"
	"github.com/wI2L/fizz/openapi"
)

func main() {
	f := kitchensink.NewWholeFizz()
	f.GET("/openapi.json", nil, f.OpenAPI(&openapi.Info{Title: "Kitchen sink", Version: "1.0.0"}, "json"))
	f.Docs("/docs/swagger", "/openapi.json", fizz.SwaggerUI())
	f.Docs("/docs/scalar", "/openapi.json", fizz.ScalarUI())
	log.Fatal(f.Engine().Run("127.0.0.1:8080"))
}
EOF
go run ./internal/tmpcheck
```

Open <http://127.0.0.1:8080/docs/swagger> and <http://127.0.0.1:8080/docs/scalar>,
with the browser console open. Delete `internal/tmpcheck` afterwards. The
CDN variants need network access; to try the embedded ones, replace
`fizz.SwaggerUI()` and `fizz.ScalarUI()` with `fizz.EmbeddedSwaggerUI()` and
`fizz.EmbeddedScalarUI()`. `NewWholeFizz` takes no options, so to try
`AllOfWrap`, build your own router with
`fizz.New(fizz.WithAnnotationStyle(openapi.AllOfWrap))` instead. The program
above was run and its three routes answered 200 with the expected content
types; nothing was opened in a browser.

## Conclusion for D17 (from the P0.6 spike)

No blocker was found in the spike. Swagger UI 5.34.1 accepted `3.2.1` with no
error and rendered `$ref` siblings, `type: [..., "null"]`, `anyOf` nullable
`$ref`, deepObject, cookie parameters, `contentEncoding`, schema `examples` and
the `default` response acceptably. The cosmetic losses are `$ref` target names
in inline schemas, the tag `summary`, and extensions. Recommendation, unchanged:
keep Swagger UI as the default, document the gaps above in README and godoc,
and point to `fizz.ScalarUI()` for fuller 3.2 rendering. This conclusion rests
on the spike document; the checklist at the top is what confirms it for fizz's
own output.

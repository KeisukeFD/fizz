# OAS 3.2.1 renderer compatibility (P0.6 spike)

Spike for decisions D17-D19. Source files live in `docs/compat/spike/`:
`spec-3.2.1.json` (hand-made kitchen-sink document), `swagger.html`, `scalar.html`.
`docs/compat/spike_test.go` (`TestSpikeDocValid`) checks that the document passes
`specvalidate.Validate` and `LintNoStrayNull`.

## Pinned versions and SRI

All verified HTTP 200 on jsDelivr on 2026-10-30. Hashes computed with
`curl -sL URL | openssl dgst -sha384 -binary | openssl base64 -A`. The browser
accepted all three hashes (a wrong hash is blocked by Chrome).

| Package | File | sha384 |
|---|---|---|
| `swagger-ui-dist@5.34.1` | `swagger-ui-bundle.js` | `sha384-XFYN47VzPiDY6qPtxYKg0nqUeW/upo20xf1j2SUE7gT/i35FJB4cuS/71Iegzk8Z` |
| `swagger-ui-dist@5.34.1` | `swagger-ui.css` | `sha384-4c3vsE5f+trhFyHrFrLM1fFGUn531NIaC7dcqEWIJdoRQJ5CkJbSuSCwXLpaS1KF` |
| `@scalar/api-reference@1.76.4` | `dist/browser/standalone.js` | `sha384-hwB9WroDTpZaI38gr+u9CXA6ksz5gUYgD2GxXazKl/agT1L2GWUiRFWGJYNiFu4Q` |

URL base: `https://cdn.jsdelivr.net/npm/<package>@<version>/<file>`.
Swagger UI is configured with `validatorUrl: null`.

## Method

Chrome for Testing (headless, Playwright cache), driven with `playwright-core`.
Pages served from a temp copy with `python3 -m http.server`. Operation expanded,
response and parameter schemas expanded, text read back from the DOM, console
messages captured. Run on 2026-10-30. Visual screenshots were taken but the
verdicts below come from DOM text, not from visual inspection of layout or styling.

## Feature x UI

Legend: rendered = shown correctly; warned = console warning/error related to it;
broken = wrong or lost; not shown = silently not displayed.

| Feature | Swagger UI 5.34.1 | Scalar 1.76.4 |
|---|---|---|
| `openapi: 3.2.1` accepted | rendered (badge "OAS 3.2", no error) | rendered ("OpenAPI 3.2.1") |
| `$ref` + sibling `description` | rendered (sibling description shown on `owner`) | rendered (sibling description shown) |
| `type: ["string","null"]` | rendered (`string \| null`) | rendered (`string \| null`) |
| nullable `$ref` via `anyOf` | rendered (`(object \| null)`, Any of #0 object, #1 null; `$ref` target name lost, shown as `object`) | rendered (`Owner`, `nullable`, with target name) |
| deepObject query param (`filter`) | rendered (`object`, Additional properties: string; no deepObject serialization hint) | rendered (`object`, additionalProperty: string; no deepObject hint) |
| cookie param | rendered (`session`, `string`, `(cookie)`) | rendered (under "Cookies") |
| `contentEncoding: base64` | rendered (`media type: image/png`, `encoding: base64`) | rendered (`string - base64`) |
| schema `examples: [...]` | rendered (#0 "Rex", #1 "Felix"); example response uses "Rex" | rendered ("Examples" label; example uses "Rex") |
| `default` response | rendered | rendered |
| tag `summary` | not shown (tag name `pets` and description shown) | likely shown (sidebar reads "Pets", not `pets`); not separately confirmed |
| `x-` extension on operation | not shown (Swagger UI hides extensions unless `showExtensions` is set) | not shown |
| named schemas in Models | rendered (Pet, Owner, Error); inline response schema shows `array<object>`, not `Pet` | rendered (Pet, Owner, Error; `array of Pet`) |

Console: Swagger UI produced no console errors except a 404 for `/favicon.ico`
(artifact of the test server). Scalar produced none.

## Known gaps

- Swagger UI issue #11065: `itemSchema` (3.2 streaming/sequential media types) is
  not rendered. Not part of this spike document; taken from the issue and D17,
  not re-verified here.
- Swagger UI does not display tag `summary`, `kind` or `parent` (3.2 tag fields);
  `kind`/`parent` were not exercised here, `summary` was.
- Swagger UI shows resolved `$ref` targets as anonymous `object` in inline
  response schemas (names are lost; the Schemas section keeps them).
- Neither UI displays vendor `x-` extensions by default.
- Neither UI marks the deepObject style visually; the serialization is not
  verified (Try it out was not exercised).
- Not exercised: `additionalOperations`, `query` method, `itemSchema`,
  `$self`, tag `kind`/`parent`, Try it out requests.

## Conclusion for D17

No blocker found. Swagger UI 5.34.1 accepts `3.2.1` with no errors and renders
`$ref` siblings, `type: [..., "null"]`, `anyOf` nullable `$ref`, deepObject,
cookie params, `contentEncoding`, schema `examples` and the `default` response
acceptably. The only cosmetic losses are `$ref` target names in inline schemas,
the tag `summary`, and extensions. Recommendation: keep Swagger UI as the
default, document the gaps above in README/godoc, and point to `fizz.ScalarUI()`
for fuller 3.2 rendering. The pins in the table above are viable for D19.

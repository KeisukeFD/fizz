# Vendored OAS 3.2 JSON Schemas

Latest revisions listed on https://spec.openapis.org/oas/ (section "Schema
Iterations", v3.2) as of 2026-10-30. They target OAS **3.2.1**: the index
states that the latest dated revision is the most correct schema for all
patch releases of a minor version. They are loaded offline via `embed.FS`;
do not edit. The JSON Schema 2020-12 meta-schemas they reference are built
into santhosh-tekuri/jsonschema.

The kinds are revised independently, so their dates differ. Each file's
`$id` was checked to match its URL. The loader's kind-to-date mapping is
`schemaDates` in `validate.go`.

| File | Source URL | sha256 |
|------|------------|--------|
| schema-2026-08-30.json | https://spec.openapis.org/oas/3.2/schema/2026-08-30 | ec53db6e114b768ff2d273b232bda210f51b70c5e57d57ac8daf7e8496a6e073 |
| schema-base-2026-08-30.json | https://spec.openapis.org/oas/3.2/schema-base/2026-08-30 | 9df52b98a92075a917125364d23a5f70825d6c292c7aeb15c31b822e95f031e9 |
| dialect-2026-02-26.json | https://spec.openapis.org/oas/3.2/dialect/2026-02-26 | 9375b7042aa0b01848403f0a9ebf12a3425e01a9e24337ce12624f10fea73833 |
| meta-2026-02-26.json | https://spec.openapis.org/oas/3.2/meta/2026-02-26 | f444d02ef7d36f30293983fad8118c823a3c70a49a8e5aa6f1afe21215e35417 |

`schema-base` is the validation entry point: it validates Schema Objects
against the OAS dialect. The 3.0 schema is deliberately not vendored.

The `openapi` pattern is `^3\.2\.[0-9]+(-.+)?$`, which accepts `3.2.1`.

## Changes versus the 2025-09-17 revision (JSON diff)

- schema: new `explode-for-form` and `token` `$defs` replace
  `styles-for-form` (explode defaults reworked for Encoding, query and
  cookie parameters; `allowReserved` default only under `style: form`);
  header parameter names and `headers` keys must be RFC 9110 tokens; path
  parameter names must match `^[^{}]+$`; Encoding `contentType` no longer
  uses `format: media-range`; `content` `$comment` expanded; `openapi`
  pattern `\d` became `[0-9]`.
- schema-base: only `$id` and refs bumped to the new dates.
- dialect: `$id` and `$ref` to meta bumped.
- meta: the Discriminator Object now requires `propertyName`; `$id` bumped.

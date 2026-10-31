# Vendored OAS 3.2 JSON Schemas

Fetched from https://spec.openapis.org/oas/3.2/ (dated release 2025-09-17,
the `$id` of each file). They are loaded offline via `embed.FS`; do not edit.
The JSON Schema 2020-12 meta-schemas they reference are built into
santhosh-tekuri/jsonschema.

| File | Source URL | sha256 |
|------|------------|--------|
| schema-2025-09-17.json | https://spec.openapis.org/oas/3.2/schema/2025-09-17 | 0c9d74bf25f9b9388b2d81e421ef60fdefa9feffa94898dadfc501b342b3bfcc |
| schema-base-2025-09-17.json | https://spec.openapis.org/oas/3.2/schema-base/2025-09-17 | 43bb51dbd831b2215cf27e734969ff50fe7043a1e60879afe9631c5a6c262214 |
| dialect-2025-09-17.json | https://spec.openapis.org/oas/3.2/dialect/2025-09-17 | 4e2c989f3d1e6489d41bc1ca4ade11743278c612b82fba9144c6116c79f1c273 |
| meta-2025-09-17.json | https://spec.openapis.org/oas/3.2/meta/2025-09-17 | a1959c0aa1f9a7ce58f2b75699be5bc5187c8a25901fe04a227f1d9419ba4e9c |

`schema-base` is the validation entry point: it validates Schema Objects
against the OAS dialect. The 3.0 schema is deliberately not vendored.

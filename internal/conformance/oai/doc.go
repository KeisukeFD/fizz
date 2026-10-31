// Package oai holds the vendored official OpenAPI Initiative material used by
// the conformance suite (see docs/plan/PLAN-v2.md, Phase 10).
//
// The pristine downloads live under orig/, pinned by commit and sha256 in
// MANIFEST.json. The package has no code: it exists to anchor go generate and
// the offline manifest test. Vendored files are data; they are only parsed,
// never executed.
package oai

//go:generate go run ../internal/fetchoai

//go:generate go run ../internal/upgradeoai

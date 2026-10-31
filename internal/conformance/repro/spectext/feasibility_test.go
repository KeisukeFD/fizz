// Package spectext records which of the full documents quoted by the
// OpenAPI 3.2 specification text (oai/upgraded/spectext, base URI and
// $self examples) fizz can reproduce with its public API (PLAN-v2 P10.5c,
// decision D23).
package spectext

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/conformance/repro"
)

// TestSpecTextFullDocsFeasibility lists the five full spec-text documents
// about $self, base URIs and external references. A document that fizz
// cannot rebuild is "not reproducible", with the reason and the
// limitation task (PLAN-v2 P10.9x). The one document that can be rebuilt
// has its app in ../baseuriretrieval.
func TestSpecTextFullDocsFeasibility(t *testing.T) {
	const (
		// noExternalRef: Operation.RequestBody is a *RequestBody, so an
		// operation cannot hold {"$ref": "shared/foo#/..."}.
		noExternalRef = "not reproducible: an operation cannot reference an external request body (Operation.RequestBody has no $ref form; a hook would have to patch the operation wholesale), limitation P10.9o"
		// noSchemaID: Schema has no $id.
		noSchemaID = "not reproducible: Schema has no $id keyword, so the Foo and Bar component schemas cannot carry their identifiers, limitation P10.9p"
	)
	tests := []struct {
		doc string
		// verdict is "reproducible: <app>" or "not reproducible: <reason>, limitation <task>".
		verdict string
		// marker is the construct of the document that fizz lacks, checked
		// against the official text so the verdict cannot drift.
		marker string
	}{
		{
			doc:     "base-uri-from-the-retrieval-uri-1",
			verdict: "reproducible: ../baseuriretrieval (RegisterRequestBody with a relative $ref schema), one limitation P10.9n (empty paths)",
			marker:  `"$ref": "schemas/foo"`,
		},
		{doc: "base-uri-within-content-1", verdict: noExternalRef, marker: `"$ref": "shared/foo#/components/requestBodies/Foo"`},
		{doc: "base-uri-within-content-2", verdict: noSchemaID, marker: `"$id": "https://example.com/api/schemas/foo"`},
		{doc: "resolving-relative-self-and-id-1", verdict: noExternalRef, marker: `"$ref": "shared/foo#/components/requestBodies/Foo"`},
		{doc: "resolving-relative-self-and-id-2", verdict: noSchemaID, marker: `"$id": "/api/schemas/foo"`},
	}

	taskRE := regexp.MustCompile(`limitation (P10\.9[a-z])`)
	covered := make([]string, 0, len(tests))
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			covered = append(covered, tc.doc)
			official := string(repro.OfficialAt(t, "spectext/"+tc.doc+".json"))
			if !strings.Contains(official, tc.marker) {
				t.Fatalf("official document no longer contains %s", tc.marker)
			}
			m := taskRE.FindStringSubmatch(tc.verdict)
			if m == nil {
				t.Fatalf("verdict %q names no limitation task", tc.verdict)
			}
			t.Log(tc.verdict)
		})
	}

	// Every full document of the family is in the table.
	var want []string
	for _, pat := range []string{"base-uri-*.json", "resolving-relative-self-and-id-*.json"} {
		files, err := filepath.Glob(filepath.Join("..", "..", "oai", "upgraded", "spectext", pat))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			want = append(want, strings.TrimSuffix(filepath.Base(f), ".json"))
		}
	}
	slices.Sort(want)
	slices.Sort(covered)
	if !slices.Equal(want, covered) {
		t.Errorf("table covers %v, the spec-text directory has %v", covered, want)
	}
}

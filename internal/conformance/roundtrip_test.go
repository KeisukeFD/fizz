package conformance

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/wI2L/fizz/internal/golden"
	"github.com/wI2L/fizz/internal/plan"
	"github.com/wI2L/fizz/internal/specvalidate"
	"github.com/wI2L/fizz/openapi"
)

// roundTripFixedBy maps a corpus document (the name of its subtest) to the
// plan tasks that fix its round-trip losses, joined with "+". A document
// that is absent must round-trip exactly. This is never an allowlist: each
// task is a spec-model bug listed in docs/plan/PLAN-v2.md (P10.3a...), and
// once the task is in plan.Done its documents must pass.
var roundTripFixedBy = map[string]string{
	// schema "properties": {} dropped.
	"spectext/example-url-encoded-form-with-json-values-1": "P10.3e",
	// request body without "content" gains "content": null.
	"spectext/base-uri-within-content-1":        "P10.3f",
	"spectext/resolving-relative-self-and-id-1": "P10.3f",
}

// diffJSON reports the semantic differences between want and got (at most
// maxDiffs of them), or "" when they are equal. Key order and number
// formatting (1.0 vs 1) are not differences.
func diffJSON(want, got []byte) string {
	const maxDiffs = 12
	diffs, err := golden.Diffs(want, got)
	if err != nil {
		return err.Error()
	}
	var b strings.Builder
	for i, d := range diffs {
		if i == maxDiffs {
			fmt.Fprintf(&b, "\n  ... and %d more", len(diffs)-maxDiffs)
			break
		}
		fmt.Fprintf(&b, "\n  at %q: input %s, output %s", d.Pointer, clip(d.Want), clip(d.Got))
	}
	return b.String()
}

func clip(s string) string {
	const n = 160
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// roundTripProblems decodes raw into the spec model, re-encodes it as JSON
// and as YAML, and returns one message per failed check.
func roundTripProblems(raw []byte) []string {
	var doc openapi.OpenAPI
	if err := json.Unmarshal(raw, &doc); err != nil {
		return []string{"unmarshal: " + err.Error()}
	}
	var problems []string

	out, err := json.Marshal(&doc)
	if err != nil {
		return []string{"marshal JSON: " + err.Error()}
	}
	if d := diffJSON(raw, out); d != "" {
		problems = append(problems, "JSON round trip differs:"+d)
	}
	if err := specvalidate.Validate(out); err != nil {
		problems = append(problems, "re-encoded JSON does not validate: "+err.Error())
	}

	y, err := openapi.MarshalYAML(&doc)
	if err != nil {
		return append(problems, "marshal YAML: "+err.Error())
	}
	var generic any
	if err := yaml.Unmarshal(y, &generic); err != nil {
		return append(problems, "decode YAML: "+err.Error())
	}
	yj, err := json.Marshal(generic)
	if err != nil {
		return append(problems, "YAML is not JSON-compatible: "+err.Error())
	}
	if d := diffJSON(raw, yj); d != "" {
		problems = append(problems, "YAML round trip differs:"+d)
	}
	return problems
}

// TestCorpusRoundTrip decodes every corpus document into openapi.OpenAPI and
// requires the JSON and YAML encodings to be semantically equal to the input,
// and the re-encoded JSON to validate. Any loss is a spec-model bug.
func TestCorpusRoundTrip(t *testing.T) {
	c := buildCorpus(t)
	var names []string
	for name := range c.Files {
		switch {
		case strings.HasPrefix(name, "upgraded/learn/"),
			strings.HasPrefix(name, "upgraded/oas/tests/schema/pass/"),
			strings.HasPrefix(name, "upgraded/spectext/"):
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("no corpus documents")
	}
	used := map[string]bool{}
	for _, name := range names {
		sub := strings.TrimPrefix(strings.TrimSuffix(name, ".json"), "upgraded/")
		fixedBy := roundTripFixedBy[sub]
		if fixedBy != "" {
			used[sub] = true
		}
		t.Run(sub, func(t *testing.T) {
			plan.Pending(t, fixedBy, func(tb testing.TB) {
				if p := roundTripProblems(c.Files[name]); len(p) > 0 {
					tb.Fatal(strings.Join(p, "\n"))
				}
			})
		})
	}
	for sub := range roundTripFixedBy {
		if !used[sub] {
			t.Errorf("roundTripFixedBy entry %s matches no corpus document", sub)
		}
	}
}

// TestDiffJSON proves the comparison catches losses and ignores
// formatting-only differences.
func TestDiffJSON(t *testing.T) {
	const in = `{"a":{"b":[1,{"c":"x"}]},"n":1.0}`
	tests := []struct {
		name, got string
		differs   bool
		want      string
	}{
		{"identical reordered", `{"n":1,"a":{"b":[1,{"c":"x"}]}}`, false, ""},
		{"dropped field", `{"a":{"b":[1,{}]},"n":1}`, true, `"/a/b/1/c"`},
		{"changed value", `{"a":{"b":[1,{"c":"y"}]},"n":1}`, true, `"/a/b/1/c"`},
		{"type changed", `{"a":{"b":[1,{"c":"x"}]},"n":"1"}`, true, `"/n"`},
		{"added field", `{"a":{"b":[1,{"c":"x"}]},"n":1,"z":null}`, true, `"/z"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := diffJSON([]byte(in), []byte(tt.got))
			if (d != "") != tt.differs {
				t.Fatalf("diffJSON = %q, differs want %v", d, tt.differs)
			}
			if tt.want != "" && !strings.Contains(d, tt.want) {
				t.Errorf("diff %q does not name pointer %s", d, tt.want)
			}
		})
	}
}

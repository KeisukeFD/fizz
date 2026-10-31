package upgrade_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/conformance/upgrade"
	"github.com/wI2L/fizz/internal/specvalidate"
)

func TestExtractBlocks(t *testing.T) {
	const md = "# Title\n\n## Same\n\n```yaml\na: 1\n```\n\n```json\n{}\n```\n\n```text\n# not a heading\n```\n\n## Same\n\n```yaml\nb: 2\n```\n\n### With `Code` & Punct!\n\n```yaml\n# comment\nc: 3\n```\n"
	got := upgrade.ExtractBlocks([]byte(md))
	if len(got) != 3 {
		t.Fatalf("blocks = %+v", got)
	}
	wants := []struct {
		name, text string
		line       int
	}{
		{"same-1", "a: 1\n", 5},
		{"same-2", "b: 2\n", 19},
		{"with-code--punct-1", "# comment\nc: 3\n", 25},
	}
	for i, w := range wants {
		if got[i].Name != w.name || got[i].Text != w.text || got[i].Line != w.line {
			t.Errorf("block %d = %+v, want %+v", i, got[i], w)
		}
	}
	if got[2].Heading != "With `Code` & Punct!" {
		t.Errorf("heading = %q", got[2].Heading)
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name, heading, yaml string
		kind                string // empty means unclassified
	}{
		{"document", "x", "openapi: 3.2.1\ninfo: {title: t, version: '1'}\npaths: {}\n", "document"},
		{"root fields", "x", "components:\n  schemas: {}\n", "document-fields"},
		{"tags", "x", "tags:\n- name: a\n", "document-fields"},
		{"paths", "Paths Object Example", "/pets:\n  get:\n    responses:\n      '200': {description: ok}\n", "paths"},
		{"info", "Info Object Example", "title: t\nversion: '1'\n", "info"},
		{"contact", "Contact Object Example", "name: n\nurl: https://e.com\n", "info/contact"},
		{"license", "License Object Example", "name: n\nidentifier: MIT\n", "info/license"},
		{"external docs", "External Documentation Object Example", "description: d\nurl: https://e.com\n", "externalDocs"},
		{"server", "Server Object Example", "url: https://e.com\ndescription: d\n", "servers"},
		{"tag", "Tag Object Example", "name: a\nsummary: s\n", "tags"},
		{"security scheme", "Security Scheme Object Example", "type: http\nscheme: basic\n", "components/securitySchemes"},
		{"schema", "Schema Object Example", "type: object\nproperties:\n  a: {type: string}\n", "components/schemas"},
		{"reference", "Reference Object Example", "$ref: '#/components/schemas/Pet'\n", "components/schemas"},
		{"parameter", "Parameter Object Examples", "name: a\nin: query\nschema: {type: string}\n", "components/parameters"},
		{"path item", "Path Item Object Example", "get:\n  responses:\n    '200': {description: ok}\n", "paths/path-item"},
		{"path item parameters", "x", "parameters:\n- name: a\n  in: query\n  schema: {type: string}\n", "paths/path-item (parameters)"},
		{"operation", "Operation Object Example", "operationId: x\nresponses:\n  '200': {description: ok}\n", "paths/operation"},
		{"responses", "Responses Object Example", "'200':\n  description: ok\ndefault:\n  description: err\n", "paths/operation/responses"},
		{"media type map", "x", "application/json:\n  schema: {type: string}\n", "components/requestBodies (content)"},
		{"request body", "Request Body Examples", "description: d\nrequired: true\ncontent: {}\n", "components/requestBodies"},
		{"response", "Response Object Examples", "description: d\ncontent: {}\n", "components/responses"},
		{"links", "Examples", "links:\n  a:\n    operationId: x\n", "components/responses (links)"},
		{"security requirement", "Security Requirement", "api_key: []\n", "security"},
		{"headers", "Header Object Example", "X-A:\n  schema: {type: string}\n", "components/headers (named map)"},
		{"callbacks", "Callback Object Examples", "cb:\n  '{$request.query.u}':\n    post:\n      responses:\n        '200': {description: ok}\n", "components/callbacks (named map)"},
		{"named schemas", "Examples", "Pet:\n  type: object\n", "components/schemas (named map)"},
		{"example", "Example Object Examples", "summary: s\nvalue: 1\n", "components/examples"},
		{"link", "Link Object Examples", "operationId: x\n", "components/links"},
		{"header", "Header Object Example", "description: d\nschema: {type: string}\n", "components/headers"},
		{"media type", "Media Type Examples", "schema: {type: string}\nexamples: {}\n", "components/mediaTypes"},
		{"data", "Examples", "formulas:\n  a: x\nwords: [a]\n", ""},
		{"scalar", "Examples", "just text\n", ""},
		{"unknown fragment", "Something", "foo:\n  bar: 1\nbaz: [1]\n", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := upgrade.DecodeYAML([]byte(tc.yaml))
			if err != nil {
				t.Fatal(err)
			}
			w, reason := upgrade.Wrap(tc.heading, "wrap test", v)
			if tc.kind == "" {
				if reason == "" {
					t.Fatalf("classified as %q, want unclassified", w.Kind)
				}
				return
			}
			if reason != "" || w.Kind != tc.kind {
				t.Fatalf("kind = %q reason = %q, want %q", w.Kind, reason, tc.kind)
			}
			b, err := upgrade.EncodeJSON(w.Doc)
			if err != nil {
				t.Fatal(err)
			}
			if w.Kind == "document" {
				return
			}
			if err := specvalidate.Validate(b); err != nil {
				t.Errorf("wrapper does not validate: %v\n%s", err, b)
			}
		})
	}
}

// TestWrapPathTemplate checks that the synthesized path carries one {name}
// segment per in:path parameter, in declaration order.
func TestWrapPathTemplate(t *testing.T) {
	for _, tc := range []struct {
		name, src, want string
	}{
		{"path-item", "parameters:\n  - {name: id, in: path, required: true, schema: {type: string}}\n  - {name: q, in: query, schema: {type: string}}\nget:\n  responses: {'200': {description: ok}}\n", "/example/{id}"},
		{"operation", "operationId: x\nparameters:\n  - {name: a, in: path, required: true, schema: {type: string}}\n  - {name: b, in: path, required: true, schema: {type: string}}\nresponses: {'200': {description: ok}}\n", "/example/{a}/{b}"},
		{"op-in-item", "get:\n  parameters:\n    - {name: petId, in: path, required: true, schema: {type: string}}\n  responses: {'200': {description: ok}}\n", "/example/{petId}"},
		{"none", "get:\n  responses: {'200': {description: ok}}\n", "/example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := upgrade.DecodeYAML([]byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			w, reason := upgrade.Wrap("Path Item Object Example", "t", v)
			if reason != "" {
				t.Fatal(reason)
			}
			paths := w.Doc["paths"].(map[string]any)
			if _, ok := paths[tc.want]; !ok || len(paths) != 1 {
				t.Errorf("paths = %v, want only %q", paths, tc.want)
			}
		})
	}
}

// TestWrapKeepsFragmentContent checks the wrapper is the only synthesized part.
func TestWrapKeepsFragmentContent(t *testing.T) {
	const src = "type: object\nproperties:\n  a:\n    type: string\n    nullable: true\n"
	v, err := upgrade.DecodeYAML([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	w, reason := upgrade.Wrap("Schema Object Example", "t", v)
	if reason != "" {
		t.Fatal(reason)
	}
	b, err := json.Marshal(w.Doc["components"])
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), `{"schemas":{"Example":{"properties":{"a":{"nullable":true,"type":"string"}},"type":"object"}}}`; got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
	if strings.Contains(w.Kind, "nullable") {
		t.Error("unexpected kind")
	}
}

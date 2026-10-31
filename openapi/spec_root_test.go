package openapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRootMarshal(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want []string // fragments that must be present
		not  []string // fragments that must be absent
	}{
		{
			"root new fields",
			&OpenAPI{
				OpenAPI:           "3.2.0",
				Info:              &Info{Title: "t", Version: "1"},
				JSONSchemaDialect: "https://spec.openapis.org/oas/3.2/dialect/2026-02-26",
				Self:              "https://example.com/api",
				Webhooks:          map[string]*PathItem{"ev": {Summary: "s"}},
			},
			[]string{
				`"jsonSchemaDialect":"https://spec.openapis.org/oas/3.2/dialect/2026-02-26"`,
				`"$self":"https://example.com/api"`,
				`"webhooks":{"ev":{"summary":"s"}}`,
			},
			nil,
		},
		{
			"root zero",
			&OpenAPI{OpenAPI: "3.2.0", Info: &Info{Title: "t", Version: "1"}, Paths: Paths{}},
			[]string{`"paths":{}`},
			[]string{"jsonSchemaDialect", "$self", "webhooks"},
		},
		{
			"root nil paths omitted",
			&OpenAPI{OpenAPI: "3.2.0"},
			nil,
			[]string{"paths"},
		},
		{
			"info license server",
			&OpenAPI{
				Info:    &Info{Title: "t", Version: "1", Summary: "sum", License: &License{Name: "MIT", Identifier: "MIT"}},
				Servers: []*Server{{URL: "u", Name: "prod"}},
			},
			[]string{`"summary":"sum"`, `"identifier":"MIT"`, `"name":"prod"`},
			nil,
		},
		{
			"info license server zero",
			&OpenAPI{Info: &Info{Title: "t", Version: "1", License: &License{Name: "MIT"}}, Servers: []*Server{{URL: "u"}}},
			nil,
			[]string{"summary", "identifier", `"name":"prod"`, `"servers":[{"url":"u","name"`},
		},
		{
			"tag",
			&Tag{Name: "n", Summary: "s", Parent: "p", Kind: "nav", ExternalDocs: &ExternalDocs{URL: "http://x"}},
			[]string{`"summary":"s"`, `"parent":"p"`, `"kind":"nav"`, `"externalDocs":{"url":"http://x"}`},
			nil,
		},
		{
			"tag zero",
			&Tag{Name: "n"},
			[]string{`{"name":"n"}`},
			[]string{"summary", "parent", "kind", "externalDocs"},
		},
		{
			"components",
			&Components{
				RequestBodies: map[string]*RequestBodyOrRef{"rb": {Value: &RequestBody{Content: map[string]*MediaType{}}}},
				PathItems:     map[string]*PathItem{"pi": {Summary: "s"}},
				MediaTypes:    map[string]*MediaTypeOrRef{"mt": {Ref: &Reference{Ref: "#/x"}}},
			},
			[]string{`"requestBodies":{"rb":{"content":{}}}`, `"pathItems":{"pi":{"summary":"s"}}`, `"mediaTypes":{"mt":{"$ref":"#/x"}}`},
			nil,
		},
		{
			"components zero",
			&Components{},
			[]string{`{}`},
			[]string{"requestBodies", "pathItems", "mediaTypes"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			s := string(b)
			for _, f := range tt.want {
				if !strings.Contains(s, f) {
					t.Errorf("missing %s in %s", f, s)
				}
			}
			for _, f := range tt.not {
				if strings.Contains(s, f) {
					t.Errorf("unexpected %s in %s", f, s)
				}
			}
		})
	}
}

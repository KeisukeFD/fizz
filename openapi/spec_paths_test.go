package openapi

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestOperationOmitZero(t *testing.T) {
	tests := []struct {
		name string
		in   *Operation
		want string
	}{
		{"empty", &Operation{}, `{}`},
		{"nil security omitted", &Operation{ID: "x"}, `{"operationId":"x"}`},
		{"empty security kept", &Operation{Security: []*SecurityRequirement{}}, `{"security":[]}`},
		{"security kept", &Operation{Security: []*SecurityRequirement{{"k": {}}}}, `{"security":[{"k":[]}]}`},
		{"bool zero absent", &Operation{Deprecated: false, XInternal: false}, `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.want {
				t.Errorf("got %s, want %s", b, tt.want)
			}
		})
	}
}

func TestMediaTypeNoNullSchema(t *testing.T) {
	b, err := json.Marshal(&MediaType{})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{}` {
		t.Errorf("got %s, want {}", b)
	}
	if strings.Contains(string(b), "schema") {
		t.Errorf("unexpected schema key in %s", b)
	}
}

func TestPathsModelMarshal(t *testing.T) {
	f := false
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"path item query", &PathItem{Query: &Operation{ID: "q"}}, `{"query":{"operationId":"q"}}`},
		{"additional operations", &PathItem{AdditionalOperations: map[string]*Operation{"COPY": {ID: "c"}}}, `{"additionalOperations":{"COPY":{"operationId":"c"}}}`},
		{"explode false kept", &Parameter{Name: "n", In: InQueryString, Explode: &f}, `{"name":"n","in":"querystring","explode":false}`},
		{"explode nil", &Parameter{Name: "n", In: InPath}, `{"name":"n","in":"path"}`},
		{"parameter new", &Parameter{Name: "n", In: InCookie, AllowReserved: true, Example: 1, Content: map[string]*MediaType{"a/b": {}}}, `{"name":"n","in":"cookie","allowReserved":true,"example":1,"content":{"a/b":{}}}`},
		{"header new", &Header{Style: "simple", Explode: &f}, `{"style":"simple","explode":false}`},
		{"media type new", &MediaType{ItemSchema: &Schema{}, PrefixEncoding: []*Encoding{{}}, ItemEncoding: &Encoding{}}, `{"itemSchema":{},"prefixEncoding":[{}],"itemEncoding":{}}`},
		{"encoding nested", &Encoding{Encoding: map[string]*Encoding{"a": {}}, PrefixEncoding: []*Encoding{{}}, ItemEncoding: &Encoding{}}, `{"encoding":{"a":{}},"prefixEncoding":[{}],"itemEncoding":{}}`},
		{"encoding explode", &Encoding{Explode: &f}, `{"explode":false}`},
		{"example", &Example{DataValue: 1, SerializedValue: "1"}, `{"dataValue":1,"serializedValue":"1"}`},
		{"response", &Response{Summary: "s"}, `{"summary":"s"}`},
		{"response links", &Response{Links: map[string]*LinkOrRef{"l": {Value: &Link{OperationID: "o"}}}}, `{"links":{"l":{"operationId":"o"}}}`},
		{"operation callbacks", &Operation{Callbacks: map[string]*CallbackOrRef{"cb": {Value: &Callback{"{$url}": {Summary: "s"}}}}}, `{"callbacks":{"cb":{"{$url}":{"summary":"s"}}}}`},
		{"link", &Link{OperationRef: "r", OperationID: "o", Parameters: map[string]any{"p": "$x"}, RequestBody: "b", Description: "d", Server: &Server{URL: "u"}}, `{"operationRef":"r","operationId":"o","parameters":{"p":"$x"},"requestBody":"b","description":"d","server":{"url":"u"}}`},
		{"callback", &Components{Callbacks: map[string]*CallbackOrRef{"c": {Value: &Callback{"{$url}": {Summary: "s"}}}}, Links: map[string]*LinkOrRef{"l": {Ref: &Reference{Ref: "#/x"}}}}, `{"links":{"l":{"$ref":"#/x"}},"callbacks":{"c":{"{$url}":{"summary":"s"}}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.want {
				t.Errorf("got  %s\nwant %s", b, tt.want)
			}
		})
	}
}

func TestOperationSecurityYAMLAndJSON(t *testing.T) {
	tests := []struct {
		name     string
		sec      []*SecurityRequirement
		wantYAML string
		wantJSON string
	}{
		{"nil", nil, "operationId: o\n", `{"operationId":"o"}`},
		{"empty", []*SecurityRequirement{}, "operationId: o\nsecurity: []\n", `{"operationId":"o","security":[]}`},
		{"set", []*SecurityRequirement{{"k": {}}}, "operationId: o\nsecurity:\n- k: []\n", `{"operationId":"o","security":[{"k":[]}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &Operation{ID: "o", Security: tt.sec}
			y, err := yaml.Marshal(op)
			if err != nil {
				t.Fatal(err)
			}
			if string(y) != tt.wantYAML {
				t.Errorf("yaml = %q, want %q", y, tt.wantYAML)
			}
			j, err := json.Marshal(op)
			if err != nil {
				t.Fatal(err)
			}
			if string(j) != tt.wantJSON {
				t.Errorf("json = %s, want %s", j, tt.wantJSON)
			}
		})
	}
}

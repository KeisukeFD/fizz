package openapi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/wI2L/fizz/openapi"
)

func readDoc(t *testing.T, path string) *openapi.OpenAPI {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc openapi.OpenAPI
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return &doc
}

// handBuilt is a document with extensions on most objects, escaped
// characters, bool schemas, a callback and the fixed shapes.
func handBuilt(t *testing.T) *openapi.OpenAPI {
	t.Helper()
	set := func(x interface{ SetExtension(string, any) error }, k string, v any) {
		t.Helper()
		if err := x.SetExtension(k, v); err != nil {
			t.Fatal(err)
		}
	}
	info := &openapi.Info{Title: "<API> & \u2028 co", Version: "1", XLogo: &openapi.XLogo{URL: "u"}}
	set(info, "x-z", 1.5)
	set(info, "x-a", map[string]any{"z": []any{"<b>", nil, true}, "a": json.Number("12")})
	set(info, "x-m", map[string]any{"k2": 2, "k1": 1})
	schema := &openapi.Schema{Type: openapi.Types("object"), Properties: map[string]*openapi.Schema{
		"yes": {Bool: new(true)},
		"no":  {Bool: new(false)},
		"n":   {Type: openapi.Types("integer", "null"), Minimum: openapi.Num(1.5)},
		"e":   {},
	}, Default: "", Examples: []any{map[string]any{"b": 2, "a": 1}}}
	set(schema, "x-s", "v")
	set(schema.Properties["n"], "x-n", false)
	cb := &openapi.Callback{Expressions: map[string]*openapi.PathItem{"{$request.body#/url}": {Summary: "cb"}}}
	set(cb, "x-cb", 1)
	op := &openapi.Operation{
		ID:       "op",
		Security: []*openapi.SecurityRequirement{{"bearer": nil, "oauth": {"read"}}},
		Callbacks: map[string]*openapi.CallbackOrRef{
			"c": {Value: cb},
			"r": {Ref: &openapi.Reference{Ref: "#/components/callbacks/c"}},
		},
		Responses: openapi.Responses{"200": {Value: &openapi.Response{Description: "ok", Content: map[string]*openapi.MediaTypeOrRef{
			"application/json": {Value: &openapi.MediaType{Schema: schema}},
		}}}},
	}
	set(op, "x-op", []any{3, 2, 1})
	item := &openapi.PathItem{GET: op}
	set(item, "x-item", "i")
	scheme := &openapi.SecurityScheme{Type: "oauth2", Flows: &openapi.OAuthFlows{
		ClientCredentials: &openapi.OAuthFlow{TokenURL: "https://example.com/token"},
	}}
	set(scheme.Flows.ClientCredentials, "x-flow", 1)
	doc := &openapi.OpenAPI{
		OpenAPI: "3.2.0",
		Info:    info,
		Paths:   openapi.Paths{"/b": item, "/a": {}},
		Components: &openapi.Components{
			SecuritySchemes: map[string]*openapi.SecuritySchemeOrRef{"oauth": {Value: scheme}},
		},
	}
	set(doc, "x-root", map[string]any{"b": 1, "a": 2})
	return doc
}

func TestExtensionsInlineSorted(t *testing.T) {
	info := &openapi.Info{Title: "t", Version: "1"}
	for k, v := range map[string]any{
		"x-b": 1,
		"x-a": map[string]any{"z": 1, "a": 2, "m": map[string]any{"y": 1, "b": 2}},
	} {
		if err := info.SetExtension(k, v); err != nil {
			t.Fatal(err)
		}
	}
	doc := &openapi.OpenAPI{OpenAPI: "3.2.0", Info: info, Paths: openapi.Paths{"/": {
		GET: &openapi.Operation{Extensions: map[string]any{"x-op": true}},
	}}}
	got, err := openapi.MarshalJSON(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"openapi":"3.2.0","info":{"title":"t","version":"1",` +
		`"x-a":{"a":2,"m":{"b":2,"y":1},"z":1},"x-b":1},` +
		`"paths":{"/":{"get":{"x-op":true}}}}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestV1EncodingJSONParity(t *testing.T) {
	docs := map[string]*openapi.OpenAPI{
		"hand built":            handBuilt(t),
		"generator output":      readDoc(t, "../testdata/spec.json"),
		"kitchen sink":          readDoc(t, "../internal/kitchensink/testdata/kitchensink.json"),
		"extensions everywhere": readDoc(t, "../internal/acceptance/testdata/extensions-everywhere.json"),
	}
	for name, doc := range docs {
		t.Run(name, func(t *testing.T) {
			want, err := openapi.MarshalJSON(doc)
			if err != nil {
				t.Fatal(err)
			}
			for what, v := range map[string]any{"pointer": doc, "value": *doc} {
				got, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf("v1 Marshal(%s) differs from MarshalJSON\nv1: %s\nv2: %s", what, got, want)
				}
			}
			if !strings.Contains(name, "hand") {
				return
			}
			for _, frag := range []string{
				`"x-a":{"a":12,"z":["\u003cb\u003e",null,true]}`, // sorted, HTML escaped like v1
				`\u2028`,
				`"bearer":[]`,
				`"scopes":{}`,
				`"yes":true`,
				`"x-cb":1`,
			} {
				if !bytes.Contains(want, []byte(frag)) {
					t.Errorf("missing %s in\n%s", frag, want)
				}
			}
		})
	}
}

func TestMarshalJSONRoundTripStable(t *testing.T) {
	for name, doc := range map[string]*openapi.OpenAPI{
		"hand built":            handBuilt(t),
		"generator output":      readDoc(t, "../testdata/spec.json"),
		"extensions everywhere": readDoc(t, "../internal/acceptance/testdata/extensions-everywhere.json"),
	} {
		t.Run(name, func(t *testing.T) {
			first, err := openapi.MarshalJSON(doc)
			if err != nil {
				t.Fatal(err)
			}
			var again openapi.OpenAPI
			if err := json.Unmarshal(first, &again); err != nil {
				t.Fatal(err)
			}
			second, err := openapi.MarshalJSON(&again)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first, second) {
				t.Errorf("round trip is not byte-identical\nfirst:  %s\nsecond: %s", first, second)
			}
		})
	}
}

func TestCallbackJSON(t *testing.T) {
	in := `{"{$url}":{"summary":"s"},"x-k":{"b":1,"a":2},"a-expr":{}}`
	var cb openapi.Callback
	if err := json.Unmarshal([]byte(in), &cb); err != nil {
		t.Fatal(err)
	}
	if len(cb.Expressions) != 2 || cb.Extensions["x-k"] == nil {
		t.Fatalf("got %+v", cb)
	}
	b, err := json.Marshal(cb)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a-expr":{},"{$url}":{"summary":"s"},"x-k":{"a":2,"b":1}}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}

func TestRejectReservedExtension(t *testing.T) {
	tests := []struct {
		key  string
		want error
	}{
		{"foo", openapi.ErrInvalidExtensionKey},
		{"", openapi.ErrInvalidExtensionKey},
		{"x-", openapi.ErrInvalidExtensionKey},
		{"X-a", openapi.ErrInvalidExtensionKey},
		{"x-oai-a", openapi.ErrReservedExtension},
		{"x-oas-a", openapi.ErrReservedExtension},
		{"X-OAI-a", openapi.ErrReservedExtension},
		{"x-OAS-a", openapi.ErrReservedExtension},
		{"x-ok", nil},
		{"x-oai", nil},
		{"x-oas", nil},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			var op openapi.Operation
			err := op.SetExtension(tt.key, 1)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if (tt.want == nil) != (len(op.Extensions) == 1) {
				t.Errorf("Extensions = %v", op.Extensions)
			}
		})
	}
}

func TestSetExtensionTypedKeyWins(t *testing.T) {
	tests := []struct {
		name string
		set  func(key string) (error, map[string]any)
		keys []string
	}{
		{"info", func(k string) (error, map[string]any) {
			x := &openapi.Info{}
			return x.SetExtension(k, 1), x.Extensions
		}, []string{"x-logo"}},
		{"openapi", func(k string) (error, map[string]any) {
			x := &openapi.OpenAPI{}
			return x.SetExtension(k, 1), x.Extensions
		}, []string{"x-tagGroups"}},
		{"operation", func(k string) (error, map[string]any) {
			x := &openapi.Operation{}
			return x.SetExtension(k, 1), x.Extensions
		}, []string{"x-codeSamples", "x-internal"}},
	}
	for _, tt := range tests {
		for _, k := range tt.keys {
			t.Run(tt.name+"/"+k, func(t *testing.T) {
				err, ext := tt.set(k)
				if !errors.Is(err, openapi.ErrInvalidExtensionKey) {
					t.Errorf("err = %v, want ErrInvalidExtensionKey", err)
				}
				if len(ext) != 0 {
					t.Errorf("typed key stored: %v", ext)
				}
			})
		}
	}
	// x-logo is an ordinary extension elsewhere.
	var s openapi.Server
	if err := s.SetExtension("x-logo", 1); err != nil {
		t.Errorf("Server x-logo: %v", err)
	}
}

func TestSetExtensionAllTypes(t *testing.T) {
	for _, x := range []interface{ SetExtension(string, any) error }{
		&openapi.OpenAPI{}, &openapi.Info{}, &openapi.Contact{}, &openapi.License{},
		&openapi.Server{}, &openapi.ServerVariable{}, &openapi.Components{},
		&openapi.PathItem{}, &openapi.Operation{}, &openapi.ExternalDocs{},
		&openapi.Parameter{}, &openapi.RequestBody{}, &openapi.MediaType{},
		&openapi.Encoding{}, &openapi.Response{}, &openapi.Header{},
		&openapi.Example{}, &openapi.Link{}, &openapi.Tag{}, &openapi.Schema{},
		&openapi.Discriminator{}, &openapi.XML{}, &openapi.SecurityScheme{},
		&openapi.OAuthFlows{}, &openapi.OAuthFlow{}, &openapi.Callback{},
	} {
		if err := x.SetExtension("x-k", "v"); err != nil {
			t.Errorf("%T: %v", x, err)
			continue
		}
		b, err := json.Marshal(x)
		if err != nil {
			t.Errorf("%T: %v", x, err)
			continue
		}
		if !bytes.Contains(b, []byte(`"x-k":"v"`)) {
			t.Errorf("%T lost its extension: %s", x, b)
		}
	}
}

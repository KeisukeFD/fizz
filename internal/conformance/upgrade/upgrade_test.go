package upgrade_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/conformance/upgrade"
	"github.com/wI2L/fizz/internal/specvalidate"
)

// upgradeJSON decodes a JSON document, upgrades it and returns the result and
// changes, both normalised through JSON for comparison.
func upgradeJSON(t *testing.T, src string) (any, []upgrade.Change) {
	t.Helper()
	var doc any
	if err := json.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatal(err)
	}
	out, changes, err := upgrade.Upgrade(doc)
	if err != nil {
		t.Fatal(err)
	}
	return norm(t, out), changes
}

func norm(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func want(t *testing.T, src string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(src), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func doc30(schema string) string {
	return `{"openapi":"3.0.3","info":{"title":"t","version":"1"},"paths":{},"components":{"schemas":{"S":` + schema + `}}}`
}

func doc32(schema string) string {
	return `{"openapi":"3.2.1","info":{"title":"t","version":"1"},"paths":{},"components":{"schemas":{"S":` + schema + `}}}`
}

func rules(cs []upgrade.Change) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Pointer+" "+c.Rule)
	}
	return out
}

func TestSchemaRules(t *testing.T) {
	tests := []struct {
		name      string
		in, out   string
		wantRules []string
	}{
		{
			name: "nullable without enum",
			in:   `{"type":"string","nullable":true}`,
			out:  `{"type":["string","null"]}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/type nullable-to-type-null",
				"/components/schemas/S/nullable nullable-removed",
			},
		},
		{
			name: "nullable with enum",
			in:   `{"type":"string","enum":["a","b"],"nullable":true}`,
			out:  `{"type":["string","null"],"enum":["a","b",null]}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/type nullable-to-type-null",
				"/components/schemas/S/enum nullable-enum-add-null",
				"/components/schemas/S/nullable nullable-removed",
			},
		},
		{
			name: "nullable with enum already holding null",
			in:   `{"type":"integer","enum":[1,null],"nullable":true}`,
			out:  `{"type":["integer","null"],"enum":[1,null]}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/type nullable-to-type-null",
				"/components/schemas/S/nullable nullable-removed",
			},
		},
		{
			name: "nullable false is dropped",
			in:   `{"type":"string","nullable":false}`,
			out:  `{"type":"string"}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/nullable nullable-false-removed",
			},
		},
		{
			name: "nullable without type is dropped, enum untouched",
			in:   `{"enum":["a"],"nullable":true}`,
			out:  `{"enum":["a"]}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/nullable nullable-without-type-removed",
			},
		},
		{
			name: "exclusive minimum",
			in:   `{"type":"integer","minimum":0,"exclusiveMinimum":true}`,
			out:  `{"type":"integer","exclusiveMinimum":0}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/exclusiveMinimum exclusive-bound-to-number",
				"/components/schemas/S/minimum exclusive-bound-inclusive-removed",
			},
		},
		{
			name: "exclusive maximum",
			in:   `{"type":"number","maximum":1.5,"exclusiveMaximum":true}`,
			out:  `{"type":"number","exclusiveMaximum":1.5}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/exclusiveMaximum exclusive-bound-to-number",
				"/components/schemas/S/maximum exclusive-bound-inclusive-removed",
			},
		},
		{
			name: "exclusive false keeps the inclusive bound",
			in:   `{"type":"integer","minimum":3,"exclusiveMinimum":false}`,
			out:  `{"type":"integer","minimum":3}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/exclusiveMinimum exclusive-bound-false-removed",
			},
		},
		{
			name: "exclusive true without a bound is left alone",
			in:   `{"type":"integer","exclusiveMinimum":true}`,
			out:  `{"type":"integer","exclusiveMinimum":true}`,
			wantRules: []string{
				"/openapi openapi-version",
			},
		},
		{
			name: "binary and base64 outside a form body are kept",
			in:   `{"type":"object","properties":{"f":{"type":"string","format":"binary"}}}`,
			out:  `{"type":"object","properties":{"f":{"type":"string","format":"binary"}}}`,
			wantRules: []string{
				"/openapi openapi-version",
			},
		},
		{
			name: "nested in properties items allOf additionalProperties",
			in: `{"type":"object","properties":{
				"a":{"type":"array","items":{"type":"string","nullable":true}},
				"b":{"allOf":[{"type":"integer","nullable":true}]},
				"c":{"type":"object","additionalProperties":{"type":"number","minimum":0,"exclusiveMinimum":true}},
				"d":{"oneOf":[{"type":"string","nullable":true}],"not":{"type":"boolean","nullable":true}}}}`,
			out: `{"type":"object","properties":{
				"a":{"type":"array","items":{"type":["string","null"]}},
				"b":{"allOf":[{"type":["integer","null"]}]},
				"c":{"type":"object","additionalProperties":{"type":"number","exclusiveMinimum":0}},
				"d":{"oneOf":[{"type":["string","null"]}],"not":{"type":["boolean","null"]}}}}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/properties/a/items/type nullable-to-type-null",
				"/components/schemas/S/properties/a/items/nullable nullable-removed",
				"/components/schemas/S/properties/b/allOf/0/type nullable-to-type-null",
				"/components/schemas/S/properties/b/allOf/0/nullable nullable-removed",
				"/components/schemas/S/properties/c/additionalProperties/exclusiveMinimum exclusive-bound-to-number",
				"/components/schemas/S/properties/c/additionalProperties/minimum exclusive-bound-inclusive-removed",
				"/components/schemas/S/properties/d/not/type nullable-to-type-null",
				"/components/schemas/S/properties/d/not/nullable nullable-removed",
				"/components/schemas/S/properties/d/oneOf/0/type nullable-to-type-null",
				"/components/schemas/S/properties/d/oneOf/0/nullable nullable-removed",
			},
		},
		{
			name: "example is kept",
			in:   `{"type":"string","nullable":true,"example":"x"}`,
			out:  `{"type":["string","null"],"example":"x"}`,
			wantRules: []string{
				"/openapi openapi-version",
				"/components/schemas/S/type nullable-to-type-null",
				"/components/schemas/S/nullable nullable-removed",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, changes := upgradeJSON(t, doc30(tc.in))
			if w := want(t, doc32(tc.out)); !reflect.DeepEqual(got, w) {
				gb, _ := json.MarshalIndent(got, "", " ")
				wb, _ := json.MarshalIndent(w, "", " ")
				t.Fatalf("document mismatch\n got: %s\nwant: %s", gb, wb)
			}
			if r := rules(changes); !reflect.DeepEqual(r, tc.wantRules) {
				t.Errorf("changes\n got: %q\nwant: %q", r, tc.wantRules)
			}
		})
	}
}

func TestChangeRecord(t *testing.T) {
	_, changes := upgradeJSON(t, doc30(`{"type":"string","enum":["a"],"nullable":true}`))
	b, err := json.Marshal(changes)
	if err != nil {
		t.Fatal(err)
	}
	const wantJSON = `[` +
		`{"pointer":"/openapi","rule":"openapi-version","before":"3.0.3","after":"3.2.1"},` +
		`{"pointer":"/components/schemas/S/type","rule":"nullable-to-type-null","before":"string","after":["string","null"]},` +
		`{"pointer":"/components/schemas/S/enum","rule":"nullable-enum-add-null","before":["a"],"after":["a",null]},` +
		`{"pointer":"/components/schemas/S/nullable","rule":"nullable-removed","before":true}]`
	if string(b) != wantJSON {
		t.Errorf("got  %s\nwant %s", b, wantJSON)
	}
}

func TestFileFieldsInFormBodies(t *testing.T) {
	const in = `{"openapi":"3.0.0","info":{"title":"t","version":"1"},"paths":{"/up":{"post":{
	  "requestBody":{"content":{
	    "multipart/form-data":{"schema":{"type":"object","properties":{
	      "file":{"type":"string","format":"binary"},
	      "files":{"type":"array","items":{"type":"string","format":"binary"}},
	      "blob":{"type":"string","format":"base64"},
	      "other":{"type":"integer","format":"binary"}}}},
	    "application/octet-stream":{"schema":{"type":"string","format":"binary"}}}},
	  "responses":{"200":{"description":"ok","content":{"application/pdf":{"schema":{"type":"string","format":"binary"}}}}}}}}}`
	const out = `{"openapi":"3.2.1","info":{"title":"t","version":"1"},"paths":{"/up":{"post":{
	  "requestBody":{"content":{
	    "multipart/form-data":{"schema":{"type":"object","properties":{
	      "file":{"type":"string","contentMediaType":"application/octet-stream"},
	      "files":{"type":"array","items":{"type":"string","contentMediaType":"application/octet-stream"}},
	      "blob":{"type":"string","contentEncoding":"base64"},
	      "other":{"type":"integer","format":"binary"}}}},
	    "application/octet-stream":{"schema":{"type":"string","format":"binary"}}}},
	  "responses":{"200":{"description":"ok","content":{"application/pdf":{"schema":{"type":"string","format":"binary"}}}}}}}}}`
	got, changes := upgradeJSON(t, in)
	if !reflect.DeepEqual(got, want(t, out)) {
		gb, _ := json.MarshalIndent(got, "", " ")
		t.Fatalf("got %s", gb)
	}
	var n int
	for _, c := range changes {
		if strings.HasPrefix(c.Rule, "binary-") || strings.HasPrefix(c.Rule, "base64-") {
			n++
		}
	}
	if n != 6 {
		t.Errorf("file-field changes = %d, want 6 (%v)", n, rules(changes))
	}
}

func TestSchemaWalkLocations(t *testing.T) {
	// A nullable schema in every place a Schema Object can live.
	const in = `{"openapi":"3.0.3","info":{"title":"t","version":"1"},
	"paths":{"/p":{
	  "parameters":[{"name":"a","in":"query","schema":{"type":"string","nullable":true}}],
	  "get":{
	    "parameters":[{"name":"b","in":"header","content":{"application/json":{"schema":{"type":"string","nullable":true}}}}],
	    "requestBody":{"content":{"application/json":{"schema":{"type":"string","nullable":true}}}},
	    "responses":{"200":{"description":"ok",
	      "headers":{"X":{"schema":{"type":"string","nullable":true}}},
	      "content":{"application/json":{"schema":{"type":"string","nullable":true}}}}},
	    "callbacks":{"cb":{"{$request.body#/u}":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"string","nullable":true}}}},"responses":{}}}}}}}},
	"components":{
	  "schemas":{"S":{"type":"string","nullable":true}},
	  "parameters":{"P":{"name":"p","in":"query","schema":{"type":"string","nullable":true}}},
	  "headers":{"H":{"schema":{"type":"string","nullable":true}}},
	  "responses":{"R":{"description":"d","content":{"application/json":{"schema":{"type":"string","nullable":true}}}}},
	  "requestBodies":{"B":{"content":{"application/json":{"schema":{"type":"string","nullable":true}}}}}}}`
	got, changes := upgradeJSON(t, in)
	if s := string(mustJSON(t, got)); strings.Contains(s, `"nullable"`) {
		t.Errorf("nullable survived somewhere: %s", s)
	}
	var n int
	for _, c := range changes {
		if c.Rule == "nullable-to-type-null" {
			n++
		}
	}
	if n != 11 {
		t.Errorf("nullable-to-type-null = %d, want 11 (%v)", n, rules(changes))
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNonSchemaObjectsUntouched(t *testing.T) {
	const in = `{"openapi":"3.0.3","info":{"title":"t","version":"1","x-nullable":true,"nullable":true},
	"x-ext":{"nullable":true,"type":"string"},
	"paths":{"/p":{"get":{"nullable":true,"x-op":{"nullable":true},
	  "parameters":[{"name":"n","in":"query","nullable":true,"schema":{"type":"string"},
	    "example":{"nullable":true,"type":"string"},"examples":{"e":{"value":{"nullable":true,"type":"string"}}}}],
	  "responses":{"200":{"description":"ok","nullable":true,"x-r":{"nullable":true},
	    "content":{"application/json":{"nullable":true,"example":{"nullable":true},
	       "schema":{"type":"object","default":{"nullable":true,"type":"string"},"example":{"nullable":true,"type":"string"},
	         "properties":{"nullable":{"type":"string"},"minimum":{"type":"integer"}},"enum":[{"nullable":true}]}}}}}}}},
	"components":{"examples":{"E":{"value":{"nullable":true,"type":"string"}}},
	  "securitySchemes":{"s":{"type":"http","scheme":"basic","nullable":true}}}}`
	var orig any
	if err := json.Unmarshal([]byte(in), &orig); err != nil {
		t.Fatal(err)
	}
	got, changes := upgradeJSON(t, in)
	if len(changes) != 1 || changes[0].Rule != "openapi-version" {
		t.Errorf("changes = %v, want only the version bump", rules(changes))
	}
	m := orig.(map[string]any)
	m["openapi"] = "3.2.1"
	if !reflect.DeepEqual(got, orig) {
		gb, _ := json.MarshalIndent(got, "", " ")
		ob, _ := json.MarshalIndent(orig, "", " ")
		t.Errorf("document changed\n got: %s\nwant: %s", gb, ob)
	}
}

func TestNoSchemaRulesOutside30(t *testing.T) {
	for _, ver := range []string{"3.1.0", "3.1.1", "3.2.0", "3.2.1"} {
		t.Run(ver, func(t *testing.T) {
			in := strings.Replace(doc30(`{"type":"string","nullable":true,"minimum":1,"exclusiveMinimum":true}`), "3.0.3", ver, 1)
			got, changes := upgradeJSON(t, in)
			wantDoc := want(t, doc32(`{"type":"string","nullable":true,"minimum":1,"exclusiveMinimum":true}`))
			if !reflect.DeepEqual(got, wantDoc) {
				t.Errorf("schema rules ran on a %s document: %s", ver, mustJSON(t, got))
			}
			wantChanges := 1
			if ver == "3.2.1" {
				wantChanges = 0
			}
			if len(changes) != wantChanges {
				t.Errorf("changes = %v, want %d", rules(changes), wantChanges)
			}
		})
	}
}

func TestVersionBump(t *testing.T) {
	got, changes := upgradeJSON(t, `{"openapi":"3.2.0","info":{"title":"t","version":"1"},"paths":{}}`)
	if !reflect.DeepEqual(got, want(t, `{"openapi":"3.2.1","info":{"title":"t","version":"1"},"paths":{}}`)) {
		t.Errorf("got %s", mustJSON(t, got))
	}
	if len(changes) != 1 || changes[0].Pointer != "/openapi" || changes[0].Before != "3.2.0" || changes[0].After != "3.2.1" {
		t.Errorf("changes = %+v", changes)
	}
}

func TestMalformedPassesThrough(t *testing.T) {
	got, changes := upgradeJSON(t, `{"info":{},"components":{"schemas":{"a":null,"b":0,"c":[]},"parameters":[1]},"paths":[]}`)
	if len(changes) != 0 {
		t.Errorf("changes = %v", changes)
	}
	if string(mustJSON(t, got)) != `{"components":{"parameters":[1],"schemas":{"a":null,"b":0,"c":[]}},"info":{},"paths":[]}` {
		t.Errorf("got %s", mustJSON(t, got))
	}
	if _, _, err := upgrade.Upgrade([]any{}); err == nil {
		t.Error("a non-object document must be an error")
	}
}

func TestPointerEscaping(t *testing.T) {
	const in = `{"openapi":"3.0.3","info":{"title":"t","version":"1"},"paths":{"/pets/{id}":{"get":{"responses":{"200":{"description":"ok","content":{"a~b/c":{"schema":{"type":"string","nullable":true}}}}}}}}}`
	_, changes := upgradeJSON(t, in)
	if got := changes[1].Pointer; got != "/paths/~1pets~1{id}/get/responses/200/content/a~0b~1c/schema/type" {
		t.Errorf("pointer = %s", got)
	}
}

func TestDeterministic(t *testing.T) {
	const src = `{"openapi":"3.0.3","info":{"title":"t","version":"1"},"paths":{},"components":{"schemas":{
	  "B":{"type":"string","nullable":true},"A":{"type":"string","nullable":true},"C":{"properties":{"z":{"type":"string","nullable":true},"y":{"type":"string","nullable":true}}}}}}`
	_, first := upgradeJSON(t, src)
	for range 20 {
		_, again := upgradeJSON(t, src)
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("change list differs between runs:\n%v\n%v", rules(first), rules(again))
		}
	}
}

func TestDecodeYAMLAndEncodeJSON(t *testing.T) {
	const src = "a: 1\nb: 1.0\nc: 2024-01-02\nd: [x, true, ~]\n200: ok\ne: &x {k: v}\nf: *x\n"
	v, err := upgrade.DecodeYAML([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	b, err := upgrade.EncodeJSON(v)
	if err != nil {
		t.Fatal(err)
	}
	const wantOut = `{
  "200": "ok",
  "a": 1,
  "b": 1,
  "c": "2024-01-02",
  "d": [
    "x",
    true,
    null
  ],
  "e": {
    "k": "v"
  },
  "f": {
    "k": "v"
  }
}
`
	if string(b) != wantOut {
		t.Errorf("got:\n%s\nwant:\n%s", b, wantOut)
	}
	if _, err := upgrade.DecodeYAML([]byte("")); err == nil {
		t.Error("empty input must fail")
	}
	if _, err := upgrade.DecodeYAML([]byte("a: [unclosed")); err == nil {
		t.Error("bad YAML must fail")
	}
}

func upgradeYAML(t *testing.T, src string) (map[string]any, []upgrade.Change) {
	t.Helper()
	v, err := upgrade.DecodeYAML([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	out, changes, err := upgrade.Upgrade(v)
	if err != nil {
		t.Fatal(err)
	}
	return norm(t, out).(map[string]any), changes
}

func TestInfoVersionString(t *testing.T) {
	tests := []struct {
		version string
		want    any
		changes int
	}{
		{"1.0", "1.0", 1},
		{"2", "2", 1},
		{"1.10", "1.10", 1},
		{"true", "true", 1},
		{`"1.0"`, "1.0", 0},
		{"1.0.0", "1.0.0", 0},
	}
	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			got, changes := upgradeYAML(t, "openapi: 3.2.1\ninfo:\n  title: t\n  version: "+tc.version+"\npaths: {}\n")
			if v := got["info"].(map[string]any)["version"]; v != tc.want {
				t.Errorf("version = %#v, want %#v", v, tc.want)
			}
			if len(changes) != tc.changes {
				t.Fatalf("changes = %v", rules(changes))
			}
			if tc.changes == 1 {
				c := changes[0]
				if c.Pointer != "/info/version" || c.Rule != "info-version-string" || c.After != tc.want {
					t.Errorf("change = %+v", c)
				}
			}
		})
	}
	t.Run("applies to every version", func(t *testing.T) {
		got, changes := upgradeYAML(t, "openapi: 3.0.3\ninfo: {title: t, version: 1.0}\npaths: {}\n")
		if got["info"].(map[string]any)["version"] != "1.0" || len(changes) != 2 {
			t.Errorf("got %v %v", got, rules(changes))
		}
	})
	t.Run("json number without source text", func(t *testing.T) {
		got, _ := upgradeJSON(t, `{"openapi":"3.2.1","info":{"title":"t","version":2},"paths":{}}`)
		if got.(map[string]any)["info"].(map[string]any)["version"] != "2" {
			t.Errorf("got %v", got)
		}
	})
	t.Run("not info.version is untouched", func(t *testing.T) {
		got, changes := upgradeYAML(t, "openapi: 3.2.1\ninfo: {title: t, version: x}\npaths: {}\nx-ext: {version: 1.0}\n")
		if len(changes) != 0 || got["x-ext"].(map[string]any)["version"] != 1.0 {
			t.Errorf("got %v %v", got, rules(changes))
		}
	})
}

func TestDialectURI(t *testing.T) {
	const old = "https://spec.openapis.org/oas/3.1/dialect/base"
	src := `{"openapi":"3.1.0","info":{"title":"t","version":"1"},"jsonSchemaDialect":"` + old + `","paths":{},
	"components":{"schemas":{
	  "A":{"$schema":"` + old + `","type":"string"},
	  "B":{"type":"object","properties":{"p":{"$schema":"https://spec.openapis.org/oas/3.1/dialect/WORK-IN-PROGRESS"}}},
	  "C":{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"},
	  "D":{"type":"string","example":{"$schema":"` + old + `"}}}}}`
	got, changes := upgradeJSON(t, src)
	m := got.(map[string]any)
	if m["jsonSchemaDialect"] != specvalidate.DialectURL {
		t.Errorf("jsonSchemaDialect = %v", m["jsonSchemaDialect"])
	}
	schemas := m["components"].(map[string]any)["schemas"].(map[string]any)
	if schemas["A"].(map[string]any)["$schema"] != specvalidate.DialectURL {
		t.Errorf("A = %v", schemas["A"])
	}
	if schemas["B"].(map[string]any)["properties"].(map[string]any)["p"].(map[string]any)["$schema"] != specvalidate.DialectURL {
		t.Errorf("B = %v", schemas["B"])
	}
	if schemas["C"].(map[string]any)["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Errorf("a 2020-12 $schema was changed: %v", schemas["C"])
	}
	if schemas["D"].(map[string]any)["example"].(map[string]any)["$schema"] != old {
		t.Errorf("$schema inside an example was changed: %v", schemas["D"])
	}
	want := []string{
		"/openapi openapi-version",
		"/jsonSchemaDialect dialect-uri",
		"/components/schemas/A/$schema dialect-uri",
		"/components/schemas/B/properties/p/$schema dialect-uri",
	}
	if r := rules(changes); !reflect.DeepEqual(r, want) {
		t.Errorf("changes\n got: %q\nwant: %q", r, want)
	}
	// The validator accepts only the 3.2 dialect for $schema, so a document
	// validates once its 3.1 URIs are rewritten and the 2020-12 $schema
	// (left alone by design) is absent.
	delete(schemas, "C")
	if err := specvalidate.Validate(mustJSON(t, got)); err != nil {
		t.Errorf("upgraded document does not validate: %v", err)
	}
}

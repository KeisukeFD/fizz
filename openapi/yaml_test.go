package openapi_test

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/wI2L/fizz/openapi"
)

// normalize converts every number of a decoded document to float64, so
// that the int/float choice of the YAML decoder does not matter.
func normalize(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			x[k] = normalize(e)
		}
	case []any:
		for i, e := range x {
			x[i] = normalize(e)
		}
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case uint64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	}
	return v
}

func yamlParityDocs(t *testing.T) map[string]*openapi.OpenAPI {
	return map[string]*openapi.OpenAPI{
		"hand built":            handBuilt(t),
		"generator output":      readDoc(t, "../testdata/spec.json"),
		"legacy kitchen sink":   readDoc(t, "../testdata/legacy/kitchensink.json"),
		"extensions everywhere": readDoc(t, "../internal/acceptance/testdata/extensions-everywhere.json"),
	}
}

func TestYAMLParity(t *testing.T) {
	for name, doc := range yamlParityDocs(t) {
		t.Run(name, func(t *testing.T) {
			j, err := openapi.MarshalJSON(doc)
			if err != nil {
				t.Fatal(err)
			}
			y, err := openapi.MarshalYAML(doc)
			if err != nil {
				t.Fatal(err)
			}
			var fromJSON, fromYAML any
			if err := json.Unmarshal(j, &fromJSON); err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal(y, &fromYAML); err != nil {
				t.Fatalf("%v\n%s", err, y)
			}
			if !reflect.DeepEqual(normalize(fromJSON), normalize(fromYAML)) {
				t.Errorf("YAML and JSON documents differ\njson: %s\nyaml:\n%s", j, y)
			}
		})
	}
}

func TestYAMLAmbiguousStrings(t *testing.T) {
	want := []any{
		"123", "true", "null", "~", "1e3", "", "0x1F", "yes", "on", "2026-10-30",
		"1.5", "-", "<<", "a: b", " lead", "trail ", "multi\nline", "multi\nline\n",
		"# not a comment", "<b> &  ", "\t", "0o17", ".inf", "NaN", "Off", "Y",
	}
	doc := &openapi.OpenAPI{
		OpenAPI: "3.2.0",
		Info:    &openapi.Info{Title: "123", Version: "1.0"},
		Components: &openapi.Components{Schemas: map[string]*openapi.Schema{
			"S": {Examples: want, Enum: want},
		}},
	}
	y, err := openapi.MarshalYAML(doc)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := yaml.Unmarshal(y, &got); err != nil {
		t.Fatalf("%v\n%s", err, y)
	}
	s := got["components"].(map[string]any)["schemas"].(map[string]any)["S"].(map[string]any)
	for _, field := range []string{"examples", "enum"} {
		if !reflect.DeepEqual(s[field], want) {
			t.Errorf("%s decoded from YAML = %#v, want %#v\n%s", field, s[field], want, y)
		}
	}
	if title := got["info"].(map[string]any)["title"]; title != "123" {
		t.Errorf("title = %#v, want the string \"123\"", title)
	}
	// Keys are strings too.
	for _, k := range want {
		d := &openapi.Schema{Default: map[string]any{k.(string): k}}
		y, err := openapi.MarshalYAML(&openapi.OpenAPI{OpenAPI: "3.2.0", Info: &openapi.Info{Title: "t", Version: "1"},
			Components: &openapi.Components{Schemas: map[string]*openapi.Schema{"S": d}}})
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := yaml.Unmarshal(y, &m); err != nil {
			t.Fatalf("key %q: %v\n%s", k, err, y)
		}
		def := m["components"].(map[string]any)["schemas"].(map[string]any)["S"].(map[string]any)["default"]
		if !reflect.DeepEqual(def, map[string]any{k.(string): k}) {
			t.Errorf("key %q: decoded %#v", k, def)
		}
	}
}

func TestYAMLNumbersKeepType(t *testing.T) {
	doc := &openapi.OpenAPI{OpenAPI: "3.2.0", Info: &openapi.Info{Title: "t", Version: "1"},
		Components: &openapi.Components{Schemas: map[string]*openapi.Schema{
			"S": {Examples: []any{1, 1.5, json.Number("1e3"), int64(math.MaxInt64), true, nil}},
		}}}
	y, err := openapi.MarshalYAML(doc)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(y, &m); err != nil {
		t.Fatal(err)
	}
	ex := m["components"].(map[string]any)["schemas"].(map[string]any)["S"].(map[string]any)["examples"].([]any)
	want := []any{1, 1.5, 1000.0, math.MaxInt64, true, nil}
	if !reflect.DeepEqual(ex, want) {
		t.Errorf("examples = %#v, want %#v", ex, want)
	}
}

var flowStart = regexp.MustCompile(`(?:^|[:-]\s)\s*([\[{].*)$`)

func TestMarshalYAMLBlockStyle(t *testing.T) {
	for name, doc := range yamlParityDocs(t) {
		t.Run(name, func(t *testing.T) {
			y, err := openapi.MarshalYAML(doc)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasSuffix(y, []byte("\n")) {
				t.Error("no trailing newline")
			}
			prev, sawTwo := 0, false
			for i, line := range strings.Split(string(y), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				if m := flowStart.FindStringSubmatch(line); m != nil && m[1] != "{}" && m[1] != "[]" {
					t.Errorf("line %d: flow collection: %q", i+1, line)
				}
				indent := len(line) - len(strings.TrimLeft(line, " "))
				if indent%2 != 0 || indent > prev+2 {
					t.Errorf("line %d: indent %d after %d, want steps of 2: %q", i+1, indent, prev, line)
				}
				sawTwo = sawTwo || indent == 2
				// The content of a sequence item starts after its "- ".
				prev = indent
				for rest := line[indent:]; strings.HasPrefix(rest, "- "); rest = rest[2:] {
					prev += 2
				}
			}
			if !sawTwo {
				t.Error("no line is indented by exactly 2 spaces")
			}
		})
	}
}

func TestOperationSecurityYAML(t *testing.T) {
	tests := []struct {
		name string
		sec  []*openapi.SecurityRequirement
		want string
	}{
		{"nil", nil, "      operationId: o\n"},
		{"empty", []*openapi.SecurityRequirement{}, "      operationId: o\n      security: []\n"},
		{"set", []*openapi.SecurityRequirement{{"k": {}}}, "      operationId: o\n      security:\n        - k: []\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &openapi.OpenAPI{OpenAPI: "3.2.0", Info: &openapi.Info{Title: "t", Version: "1"},
				Paths: openapi.Paths{"/x": {GET: &openapi.Operation{ID: "o", Security: tt.sec}}}}
			y, err := openapi.MarshalYAML(doc)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(y), "    get:\n"+tt.want) {
				t.Errorf("yaml = %s", y)
			}
		})
	}
}

// TestMarshalYAMLRefsAndTypes covers the $ref/value choice of the
// OrRef types and the single-or-list shape of type sets through YAML.
func TestMarshalYAMLRefsAndTypes(t *testing.T) {
	doc := &openapi.OpenAPI{OpenAPI: "3.2.0", Info: &openapi.Info{Title: "t", Version: "1"},
		Components: &openapi.Components{
			Parameters: map[string]*openapi.ParameterOrRef{
				"ref": {Ref: &openapi.Reference{Ref: "#/components/parameters/val"}},
				"val": {Value: &openapi.Parameter{Name: "n", In: openapi.InQuery,
					Schema: &openapi.Schema{Type: openapi.Types("string")}}},
				"set": {Value: &openapi.Parameter{Name: "m", In: openapi.InQuery,
					Schema: &openapi.Schema{Type: openapi.Types("string", "null")}}},
			},
		}}
	y, err := openapi.MarshalYAML(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, frag := range []string{
		"    ref:\n      $ref: '#/components/parameters/val'\n",
		"        type: string\n",
		"          - string\n          - \"null\"\n",
	} {
		if !strings.Contains(string(y), frag) {
			t.Errorf("missing %q in\n%s", frag, y)
		}
	}
}

func TestMarshalYAMLNilOrRef(t *testing.T) {
	doc := &openapi.OpenAPI{OpenAPI: "3.2.0", Info: &openapi.Info{Title: "t", Version: "1"},
		Components: &openapi.Components{Parameters: map[string]*openapi.ParameterOrRef{"x": {}}}}
	if _, err := openapi.MarshalYAML(doc); err == nil {
		t.Error("expected an error for an OrRef with neither $ref nor value")
	}
}

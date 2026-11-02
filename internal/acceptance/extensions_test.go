package acceptance

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/internal/specvalidate"
	"github.com/KeisukeFD/fizz/openapi"
)

// H01: x-k survives a round trip on every extensible object.
func TestExtensionsRoundTripEverywhere(t *testing.T) {
	raw, err := os.ReadFile("testdata/extensions-everywhere.json")
	if err != nil {
		t.Fatal(err)
	}
	// The input itself must be a valid 3.2 document.
	if err = specvalidate.Validate(raw); err != nil {
		t.Fatalf("testdata is not a valid OAS 3.2 document: %v", err)
	}
	var doc openapi.OpenAPI
	if err = json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(&doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var want, got any
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("round trip differs\nwant: %s\ngot:  %s", bHCompact(raw), out)
	}
	if err := specvalidate.Validate(out); err != nil {
		t.Fatalf("round-tripped document invalid: %v", err)
	}

}

func bHCompact(b []byte) []byte {
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		return b
	}
	return buf.Bytes()
}

// H02: extension keys (and nested map keys) are emitted sorted.
func TestExtensionKeysSorted(t *testing.T) {
	info := at.FromJSON[openapi.Info](`{"title":"t","version":"1","x-b":1,"x-a":{"z":1,"a":2}}`)
	out, err := json.Marshal(&info)
	if err != nil {
		t.Fatal(err)
	}
	ia, ib := bytes.Index(out, []byte(`"x-a"`)), bytes.Index(out, []byte(`"x-b"`))
	if ia < 0 || ib < 0 {
		t.Fatalf("extensions lost: %s", out)
	}
	if ia > ib {
		t.Errorf("x-a must precede x-b: %s", out)
	}
	a, z := bytes.Index(out, []byte(`"a":2`)), bytes.Index(out, []byte(`"z":1`))
	if a < 0 || z < 0 || a > z {
		t.Errorf(`"a" must precede "z": %s`, out)
	}

}

// H12: the served YAML of an OAuth flow without scopes has scopes: {}
// and tokenUrl (the wrong OAuthFlow.MarshalYAML is gone).
func TestOAuthFlowYAML(t *testing.T) {
	f := fizz.New()
	f.Generator().SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
		"oauth": {Value: &openapi.SecurityScheme{
			Type: "oauth2",
			Flows: &openapi.OAuthFlows{ClientCredentials: &openapi.OAuthFlow{
				TokenURL: "https://example.com/token",
			}},
		}},
	})
	f.GET("/openapi.yaml", nil, f.OpenAPI(&openapi.Info{Title: "t", Version: "1"}, "yaml"))

	w := at.Do(f, http.MethodGet, "/openapi.yaml")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	v := at.YAMLToAny(t, w.Body.Bytes())
	flow := bHPath(v, "components", "securitySchemes", "oauth", "flows", "clientCredentials")
	m, ok := flow.(map[string]any)
	if !ok {
		t.Fatalf("clientCredentials missing or not a map: %v\n%s", flow, w.Body)
	}
	if m["tokenUrl"] != "https://example.com/token" {
		t.Errorf("tokenUrl = %v", m["tokenUrl"])
	}
	sc, ok := m["scopes"].(map[string]any)
	if !ok || len(sc) != 0 {
		t.Errorf("scopes = %#v, want {}", m["scopes"])
	}

}

func bHPath(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

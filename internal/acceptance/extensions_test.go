package acceptance

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/internal/specvalidate"
	"github.com/wI2L/fizz/openapi"
)

// H01: x-k survives a round trip on every extensible object.
func TestExtensionsRoundTripEverywhere(t *testing.T) {
	raw, err := os.ReadFile("testdata/extensions-everywhere.json")
	if err != nil {
		t.Fatal(err)
	}
	// The input itself must be a valid 3.2 document (not pending).
	if err := specvalidate.Validate(raw); err != nil {
		t.Fatalf("testdata is not a valid OAS 3.2 document: %v", err)
	}
	at.Pending(t, "P2.3", func(tb testing.TB) {
		var doc openapi.OpenAPI
		if err := json.Unmarshal(raw, &doc); err != nil {
			tb.Fatalf("unmarshal: %v", err)
		}
		out, err := json.Marshal(&doc)
		if err != nil {
			tb.Fatalf("marshal: %v", err)
		}
		var want, got any
		if err := json.Unmarshal(raw, &want); err != nil {
			tb.Fatal(err)
		}
		if err := json.Unmarshal(out, &got); err != nil {
			tb.Fatal(err)
		}
		if !reflect.DeepEqual(want, got) {
			tb.Fatalf("round trip differs\nwant: %s\ngot:  %s", bHCompact(raw), out)
		}
		if err := specvalidate.Validate(out); err != nil {
			tb.Fatalf("round-tripped document invalid: %v", err)
		}
	})
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
	at.Pending(t, "P2.3", func(tb testing.TB) {
		info := at.FromJSON[openapi.Info](`{"title":"t","version":"1","x-b":1,"x-a":{"z":1,"a":2}}`)
		out, err := json.Marshal(&info)
		if err != nil {
			tb.Fatal(err)
		}
		ia, ib := bytes.Index(out, []byte(`"x-a"`)), bytes.Index(out, []byte(`"x-b"`))
		if ia < 0 || ib < 0 {
			tb.Fatalf("extensions lost: %s", out)
		}
		if ia > ib {
			tb.Errorf("x-a must precede x-b: %s", out)
		}
		a, z := bytes.Index(out, []byte(`"a":2`)), bytes.Index(out, []byte(`"z":1`))
		if a < 0 || z < 0 || a > z {
			tb.Errorf(`"a" must precede "z": %s`, out)
		}
	})
}

// H12: the served YAML of an OAuth flow without scopes has scopes: {}
// and tokenUrl (the wrong OAuthFlow.MarshalYAML is gone).
func TestOAuthFlowYAML(t *testing.T) {
	at.Pending(t, "P2.5", func(tb testing.TB) {
		f := fizz.New()
		f.Generator().SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
			"oauth": {SecurityScheme: &openapi.SecurityScheme{
				Type: "oauth2",
				Flows: &openapi.OAuthFlows{ClientCredentials: &openapi.OAuthFlow{
					TokenURL: "https://example.com/token",
				}},
			}},
		})
		f.GET("/openapi.yaml", nil, f.OpenAPI(&openapi.Info{Title: "t", Version: "1"}, "yaml"))

		w := at.Do(f, http.MethodGet, "/openapi.yaml")
		if w.Code != http.StatusOK {
			tb.Fatalf("status %d: %s", w.Code, w.Body)
		}
		v := at.YAMLToAny(tb, w.Body.Bytes())
		flow := bHPath(v, "components", "securitySchemes", "oauth", "flows", "clientCredentials")
		m, ok := flow.(map[string]any)
		if !ok {
			tb.Fatalf("clientCredentials missing or not a map: %v\n%s", flow, w.Body)
		}
		if m["tokenUrl"] != "https://example.com/token" {
			tb.Errorf("tokenUrl = %v", m["tokenUrl"])
		}
		sc, ok := m["scopes"].(map[string]any)
		if !ok || len(sc) != 0 {
			tb.Errorf("scopes = %#v, want {}", m["scopes"])
		}
	})
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

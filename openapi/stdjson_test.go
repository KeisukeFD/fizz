package openapi_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/KeisukeFD/fizz/openapi"
)

// TestStdlibJSONRoundTrip exercises the encoding/json entry points
// (MarshalJSON/UnmarshalJSON) of every spec object: a zero value must
// marshal to a JSON object and decode back to an equal value.
func TestStdlibJSONRoundTrip(t *testing.T) {
	zero := []any{
		openapi.OpenAPI{}, openapi.Info{}, openapi.Contact{}, openapi.License{},
		openapi.Server{}, openapi.ServerVariable{}, openapi.Components{},
		openapi.PathItem{}, openapi.Operation{}, openapi.ExternalDocs{},
		openapi.Parameter{}, openapi.RequestBody{}, openapi.MediaType{},
		openapi.Encoding{}, openapi.Response{}, openapi.Header{},
		openapi.Example{}, openapi.Link{}, openapi.Tag{},
		openapi.Discriminator{}, openapi.XML{}, openapi.SecurityScheme{},
		openapi.OAuthFlows{}, openapi.OAuthFlow{},
	}
	for _, v := range zero {
		typ := reflect.TypeOf(v)
		t.Run(typ.Name(), func(t *testing.T) {
			b, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if len(b) == 0 || b[0] != '{' {
				t.Fatalf("want a JSON object, got %s", b)
			}
			ptr := reflect.New(typ)
			if err := json.Unmarshal(b, ptr.Interface()); err != nil {
				t.Fatalf("unmarshal %s: %v", b, err)
			}
			b2, err := json.Marshal(ptr.Elem().Interface())
			if err != nil {
				t.Fatalf("remarshal: %v", err)
			}
			if string(b) != string(b2) {
				t.Errorf("round trip mismatch: %s != %s", b, b2)
			}
			if err := json.Unmarshal([]byte(`[]`), ptr.Interface()); err == nil {
				t.Error("expected an error decoding a JSON array")
			}
		})
	}
}

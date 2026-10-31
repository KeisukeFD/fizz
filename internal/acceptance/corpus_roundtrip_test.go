package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/internal/golden"
	"github.com/wI2L/fizz/openapi"
)

// Minimal reproductions of the spec-model losses that the official corpus
// round trip (internal/conformance TestCorpusRoundTrip) found. Each is
// the smallest input that loses information, and is red until the task
// named in FixedBy lands.

// exactRoundTrip decodes in into a T, re-encodes it and requires the same
// JSON value back.
func exactRoundTrip[T any](tb testing.TB, in string) {
	tb.Helper()
	v := at.FromJSON[T](in)
	out, err := json.Marshal(&v)
	if err != nil {
		tb.Fatal(err)
	}
	diffs, err := golden.Diffs([]byte(in), out)
	if err != nil {
		tb.Fatal(err)
	}
	for _, d := range diffs {
		tb.Errorf("%s: input %s, output %s", d.Pointer, d.Want, d.Got)
	}
}

// Parameter "required": false is an explicit value and is dropped.
func TestCorpusParameterRequiredFalse(t *testing.T) {
	exactRoundTrip[openapi.Parameter](t, `{"name":"q","in":"query","required":false,"schema":{"type":"string"}}`)
}

// "webhooks": {} is dropped; a document whose only top-level field is an
// empty webhooks then no longer validates.
func TestCorpusEmptyWebhooks(t *testing.T) {
	at.Pending(t, "P10.3b", func(tb testing.TB) {
		exactRoundTrip[openapi.OpenAPI](tb, `{"openapi":"3.2.1","info":{"title":"t","version":"1"},"webhooks":{}}`)
	})
}

// Components "pathItems": {} is dropped.
func TestCorpusEmptyComponentsPathItems(t *testing.T) {
	at.Pending(t, "P10.3c", func(tb testing.TB) {
		exactRoundTrip[openapi.Components](tb, `{"pathItems":{}}`)
	})
}

// Operation "parameters": [] is dropped.
func TestCorpusEmptyOperationParameters(t *testing.T) {
	at.Pending(t, "P10.3d", func(tb testing.TB) {
		exactRoundTrip[openapi.Operation](tb, `{"parameters":[],"responses":{"200":{"description":"ok"}}}`)
	})
}

// Schema "properties": {} is dropped.
func TestCorpusEmptySchemaProperties(t *testing.T) {
	at.Pending(t, "P10.3e", func(tb testing.TB) {
		exactRoundTrip[openapi.Schema](tb, `{"type":"object","properties":{}}`)
	})
}

// A request body without "content" gains "content": null.
func TestCorpusRequestBodyWithoutContent(t *testing.T) {
	at.Pending(t, "P10.3f", func(tb testing.TB) {
		exactRoundTrip[openapi.RequestBody](tb, `{"description":"d"}`)
	})
}

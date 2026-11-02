package specvalidate_test

import (
	"fmt"
	"testing"

	"github.com/KeisukeFD/fizz/internal/specvalidate"
)

// doc wraps the given paths and components JSON in a 3.2.1 document.
func doc(paths, components string) []byte {
	if paths == "" {
		paths = `{}`
	}
	if components == "" {
		components = `{}`
	}
	return fmt.Appendf(nil, `{"openapi":"3.2.1","info":{"title":"t","version":"1"},
	"paths":%s,"components":%s}`, paths, components)
}

// op wraps one operation body at GET /x.
func op(body string) string {
	return `{"/x":{"get":` + body + `}}`
}

const okResp = `"responses":{"200":{"description":"ok"}}`

// TestValidatorRejectsKnownBadShapes proves that Validate plus
// LintNoStrayNull catch the invalid shapes the fizz generator produces
// today, so the kitchen-sink gate is meaningful.
func TestValidatorRejectsKnownBadShapes(t *testing.T) {
	schemaIn := func(s string) string {
		return op(`{"responses":{"200":{"description":"ok","content":{"application/json":{"schema":` + s + `}}}}}`)
	}
	tests := []struct {
		name  string
		doc   []byte
		valid bool
	}{
		{"valid baseline", doc(op(`{`+okResp+`}`), ""), true},
		{"schema null", doc(schemaIn(`null`), ""), false},
		{"nullable true in 3.2", doc(schemaIn(`{"type":"string","nullable":true}`), ""), false},
		{"exclusiveMinimum boolean", doc(schemaIn(`{"type":"integer","minimum":1,"exclusiveMinimum":true}`), ""), false},
		{"allOf object", doc(schemaIn(`{"allOf":{"type":"string"}}`), ""), false},
		{"parameter without schema or content",
			doc(op(`{"parameters":[{"name":"q","in":"query"}],`+okResp+`}`), ""), false},
		{"response without description",
			doc(op(`{"responses":{"200":{}}}`), ""), true},
		{"security requirement value null",
			doc(op(`{"security":[{"k":null}],`+okResp+`}`), ""), false},
		{"oauth flow without scopes",
			doc("", `{"securitySchemes":{"o":{"type":"oauth2","flows":{"implicit":{"authorizationUrl":"https://e.com/a"}}}}}`), false},
		{"type bogus", doc(schemaIn(`{"type":"bogus"}`), ""), false},
		{"component name with bracket",
			doc("", `{"schemas":{"Foo[Bar]":{"type":"string"}}}`), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := specvalidate.Validate(tt.doc)
			for _, lint := range []func([]byte) error{
				specvalidate.LintNoStrayNull,
				specvalidate.LintNoNullable,
				specvalidate.LintComponentNames,
			} {
				if err == nil {
					err = lint(tt.doc)
				}
			}
			switch {
			case tt.valid && err != nil:
				t.Errorf("must be accepted, got: %v", err)
			case !tt.valid && err == nil:
				t.Error("must be rejected, was accepted")
			case err != nil:
				t.Log(err)
			}
		})
	}
}
